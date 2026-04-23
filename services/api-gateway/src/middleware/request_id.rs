//! X-Request-ID + W3C traceparent middleware.
//!
//! Two responsibilities — split deliberately:
//!
//! 1. `request_id_middleware` — pure header canonicalization. Mint/reuse
//!    `X-Request-ID`, mint a fresh `traceparent` if missing, echo both on
//!    the response, stash a `RequestContext` for downstream code.
//!
//! 2. `make_gateway_span(&Request)` — used by `tower_http::TraceLayer::
//!    make_span_with` to build the per-request root tracing span. This is
//!    where #63's OTEL remote-parent adoption happens, because
//!    `make_span_with` runs *at span-creation time* — before
//!    `tracing-opentelemetry` attaches an OTEL context. Calling
//!    `span.set_parent()` inside this closure means the span's OTEL
//!    context is stamped with the caller's trace_id, and every child
//!    span (including the `#[instrument]`-marked `gateway_handler`)
//!    inherits it. Setting the parent *after* the span is established —
//!    the approach initially tried in `request_id_middleware` — doesn't
//!    work: children are already bound to the old context.

use axum::{
    body::Body,
    extract::Request,
    http::{header::HeaderName, HeaderValue, Response},
    middleware::Next,
};
use opentelemetry::trace::{SpanContext, SpanId, TraceContextExt, TraceFlags, TraceId, TraceState};
use std::sync::OnceLock;
use tracing::Span;
use tracing_opentelemetry::OpenTelemetrySpanExt;
use uuid::Uuid;

/// Header name for the request-scoped correlation ID.
pub const REQUEST_ID_HEADER: &str = "x-request-id";
/// W3C Trace Context header.
pub const TRACEPARENT_HEADER: &str = "traceparent";

static REQUEST_ID_HEADER_NAME: OnceLock<HeaderName> = OnceLock::new();
static TRACEPARENT_HEADER_NAME: OnceLock<HeaderName> = OnceLock::new();

fn request_id_header_name() -> &'static HeaderName {
    REQUEST_ID_HEADER_NAME.get_or_init(|| HeaderName::from_static(REQUEST_ID_HEADER))
}

fn traceparent_header_name() -> &'static HeaderName {
    TRACEPARENT_HEADER_NAME.get_or_init(|| HeaderName::from_static(TRACEPARENT_HEADER))
}

/// Extension attached to the Axum request so downstream code (gRPC clients,
/// handlers, tracing layers) can read the active request_id and traceparent
/// without re-parsing headers.
#[allow(dead_code)]
#[derive(Clone, Debug)]
pub struct RequestContext {
    pub request_id: String,
    pub traceparent: String,
}

impl RequestContext {
    /// Build metadata key/value pairs to forward onto a gRPC request.
    #[allow(dead_code)]
    pub fn grpc_metadata(&self) -> [(HeaderName, HeaderValue); 2] {
        [
            (
                request_id_header_name().clone(),
                HeaderValue::from_str(&self.request_id)
                    .unwrap_or_else(|_| HeaderValue::from_static("unknown")),
            ),
            (
                traceparent_header_name().clone(),
                HeaderValue::from_str(&self.traceparent)
                    .unwrap_or_else(|_| HeaderValue::from_static("00-0-0-00")),
            ),
        ]
    }
}

/// Middleware: stamp every request with `X-Request-ID` + `traceparent`
/// and echo them on the response. All OTEL/tempo wiring now lives in
/// `make_gateway_span` so that parent-context adoption runs at span-
/// creation time (see module docs).
pub async fn request_id_middleware(mut request: Request, next: Next) -> Response<Body> {
    let request_id = request
        .headers()
        .get(REQUEST_ID_HEADER)
        .and_then(|v| v.to_str().ok())
        .filter(|s| !s.is_empty())
        .map(|s| s.to_string())
        .unwrap_or_else(|| Uuid::new_v4().to_string());

    let traceparent = request
        .headers()
        .get(TRACEPARENT_HEADER)
        .and_then(|v| v.to_str().ok())
        .filter(|s| is_valid_traceparent(s))
        .map(|s| s.to_string())
        .unwrap_or_else(new_traceparent);

    // Write back onto the request so inner services + gRPC clients see the
    // canonical values (guaranteed present, no duplicates).
    if let Ok(hv) = HeaderValue::from_str(&request_id) {
        request.headers_mut().insert(request_id_header_name(), hv);
    }
    if let Ok(hv) = HeaderValue::from_str(&traceparent) {
        request.headers_mut().insert(traceparent_header_name(), hv);
    }

    let ctx = RequestContext {
        request_id: request_id.clone(),
        traceparent: traceparent.clone(),
    };
    request.extensions_mut().insert(ctx);

    let mut response = next.run(request).await;

    // Always echo the canonical values on the response.
    if let Ok(hv) = HeaderValue::from_str(&request_id) {
        response
            .headers_mut()
            .insert(request_id_header_name(), hv);
    }
    if let Ok(hv) = HeaderValue::from_str(&traceparent) {
        response
            .headers_mut()
            .insert(traceparent_header_name(), hv);
    }

    response
}

/// Build the per-request root tracing span. Intended to be passed to
/// `tower_http::TraceLayer::new_for_http().make_span_with(make_gateway_span)`.
///
/// #63: this closure runs **before** `tracing-opentelemetry` has attached
/// an OTEL context to the span, so `span.set_parent(remote_ctx)` called
/// here actually changes the span's OTEL trace_id. The span is explicitly
/// INFO-level so it passes the default `RUST_LOG=info` filter — the
/// library default of DEBUG would drop it and leave the OTEL pipeline
/// with no span to attach children to.
///
/// Attributes set:
/// - `request_id` (from X-Request-ID, or minted UUIDv4 if missing)
/// - `tenant.slug` (when X-Tenant-Slug is injected by Traefik)
/// - `http.method`, `http.route` for dashboard filters
///
/// The span's target is `gateway_request` so Tempo's `rootTraceName`
/// search surfaces it cleanly.
pub fn make_gateway_span(request: &axum::http::Request<Body>) -> Span {
    let request_id = request
        .headers()
        .get(REQUEST_ID_HEADER)
        .and_then(|v| v.to_str().ok())
        .filter(|s| !s.is_empty())
        .map(|s| s.to_string())
        .unwrap_or_default();

    let tenant_slug = request
        .headers()
        .get("x-tenant-slug")
        .and_then(|v| v.to_str().ok())
        .unwrap_or("")
        .to_string();

    let method = request.method().as_str().to_string();
    let path = request.uri().path().to_string();

    // Create the span at INFO so the env_filter `info` default keeps it.
    // Fields must be pre-declared here; `tracing-opentelemetry` promotes
    // them to OTLP attributes, which is what Tempo indexes on.
    let span = tracing::info_span!(
        "gateway_request",
        request_id = %request_id,
        "tenant.slug" = %tenant_slug,
        "http.method" = %method,
        "http.route" = %path,
    );

    // Adopt the caller's W3C traceparent (when present and valid) as the
    // OTEL remote parent. Doing this *inside* make_span_with is the
    // load-bearing ordering decision: tracing-opentelemetry's
    // on_new_span hook fires when this span is first entered, and by
    // then the parent override is already in place.
    if let Some(tp) = request
        .headers()
        .get(TRACEPARENT_HEADER)
        .and_then(|v| v.to_str().ok())
        .filter(|s| is_valid_traceparent(s))
    {
        if let Some(remote_ctx) = parse_traceparent_to_context(tp) {
            span.set_parent(remote_ctx);
        }
    }

    span
}

/// Parse a validated W3C traceparent into an `opentelemetry::Context` that
/// carries the caller's SpanContext as `is_remote = true`. Returns `None`
/// on any byte-shape failure — callers should have validated with
/// `is_valid_traceparent` first, but this helper re-checks defensively so
/// it's safe to call on raw input.
fn parse_traceparent_to_context(s: &str) -> Option<opentelemetry::Context> {
    let parts: Vec<&str> = s.split('-').collect();
    if parts.len() != 4 {
        return None;
    }
    let trace_id_bytes: [u8; 16] = hex::decode(parts[1]).ok()?.try_into().ok()?;
    let span_id_bytes: [u8; 8] = hex::decode(parts[2]).ok()?.try_into().ok()?;
    let flags_byte: u8 = u8::from_str_radix(parts[3], 16).ok()?;

    let sc = SpanContext::new(
        TraceId::from_bytes(trace_id_bytes),
        SpanId::from_bytes(span_id_bytes),
        TraceFlags::new(flags_byte),
        /* is_remote */ true,
        TraceState::default(),
    );
    if !sc.is_valid() {
        return None;
    }
    Some(opentelemetry::Context::current().with_remote_span_context(sc))
}

/// W3C traceparent validator: version-trace_id-span_id-flags (00-<32hex>-<16hex>-<2hex>).
fn is_valid_traceparent(s: &str) -> bool {
    let parts: Vec<&str> = s.split('-').collect();
    if parts.len() != 4 {
        return false;
    }
    if parts[0] != "00" {
        return false;
    }
    if parts[1].len() != 32 || !parts[1].chars().all(|c| c.is_ascii_hexdigit()) {
        return false;
    }
    if parts[2].len() != 16 || !parts[2].chars().all(|c| c.is_ascii_hexdigit()) {
        return false;
    }
    if parts[3].len() != 2 || !parts[3].chars().all(|c| c.is_ascii_hexdigit()) {
        return false;
    }
    // All-zero trace IDs are invalid.
    if parts[1].bytes().all(|b| b == b'0') {
        return false;
    }
    true
}

/// Build a new W3C traceparent: version 00, random 128-bit trace id, random
/// 64-bit parent span id, sampled flag on.
pub fn new_traceparent() -> String {
    let trace_id = hex_lower(&Uuid::new_v4().into_bytes());
    let span_id = hex_lower(&Uuid::new_v4().into_bytes()[..8]);
    format!("00-{}-{}-01", trace_id, span_id)
}

fn hex_lower(bytes: &[u8]) -> String {
    const HEX: &[u8; 16] = b"0123456789abcdef";
    let mut out = String::with_capacity(bytes.len() * 2);
    for &b in bytes {
        out.push(HEX[(b >> 4) as usize] as char);
        out.push(HEX[(b & 0x0f) as usize] as char);
    }
    out
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn validates_well_formed_traceparent() {
        let tp = "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01";
        assert!(is_valid_traceparent(tp));
    }

    #[test]
    fn rejects_wrong_version() {
        let tp = "ff-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01";
        assert!(!is_valid_traceparent(tp));
    }

    #[test]
    fn rejects_short_trace_id() {
        assert!(!is_valid_traceparent("00-abc-00f067aa0ba902b7-01"));
    }

    #[test]
    fn rejects_zero_trace_id() {
        let tp = "00-00000000000000000000000000000000-00f067aa0ba902b7-01";
        assert!(!is_valid_traceparent(tp));
    }

    #[test]
    fn rejects_non_hex_chars() {
        let tp = "00-xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx-00f067aa0ba902b7-01";
        assert!(!is_valid_traceparent(tp));
    }

    #[test]
    fn new_traceparent_is_valid_and_sampled() {
        for _ in 0..16 {
            let tp = new_traceparent();
            assert!(is_valid_traceparent(&tp), "generated bad traceparent: {tp}");
            assert!(tp.ends_with("-01"));
        }
    }

    // #63: parse helper adopts inbound traceparent as remote SpanContext.
    #[test]
    fn parse_traceparent_extracts_remote_span_context() {
        let tp = "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01";
        let ctx = parse_traceparent_to_context(tp).expect("should parse");
        let sc = ctx.span().span_context().clone();
        assert!(sc.is_valid());
        assert!(sc.is_remote());
        assert_eq!(sc.trace_id().to_string(), "4bf92f3577b34da6a3ce929d0e0e4736");
        assert_eq!(sc.span_id().to_string(), "00f067aa0ba902b7");
        assert_eq!(sc.trace_flags().to_u8(), 0x01);
    }

    #[test]
    fn parse_traceparent_rejects_malformed_input() {
        // Unvalidated inputs should cleanly return None without panicking.
        assert!(parse_traceparent_to_context("").is_none());
        assert!(parse_traceparent_to_context("not-a-traceparent").is_none());
        assert!(parse_traceparent_to_context("00-short-00f067aa0ba902b7-01").is_none());
        // All-zero trace id fails validity check.
        assert!(parse_traceparent_to_context(
            "00-00000000000000000000000000000000-00f067aa0ba902b7-01"
        )
        .is_none());
    }

    use axum::{body::Body, http::StatusCode, routing::get, Router};
    use tower::ServiceExt;

    async fn ok_handler() -> StatusCode {
        StatusCode::OK
    }

    fn app() -> Router {
        Router::new()
            .route("/x", get(ok_handler))
            .layer(axum::middleware::from_fn(request_id_middleware))
    }

    #[tokio::test]
    async fn middleware_echoes_incoming_request_id() {
        let req = axum::http::Request::builder()
            .uri("/x")
            .header(REQUEST_ID_HEADER, "caller-supplied-id")
            .body(Body::empty())
            .unwrap();
        let res = app().oneshot(req).await.unwrap();
        assert_eq!(res.status(), StatusCode::OK);
        assert_eq!(
            res.headers().get(REQUEST_ID_HEADER).unwrap(),
            "caller-supplied-id"
        );
        assert!(res.headers().get(TRACEPARENT_HEADER).is_some());
    }

    #[tokio::test]
    async fn middleware_generates_request_id_when_absent() {
        let req = axum::http::Request::builder()
            .uri("/x")
            .body(Body::empty())
            .unwrap();
        let res = app().oneshot(req).await.unwrap();
        assert_eq!(res.status(), StatusCode::OK);
        let rid = res.headers().get(REQUEST_ID_HEADER).unwrap();
        // Generated UUID v4 has length 36.
        assert_eq!(rid.to_str().unwrap().len(), 36);
    }

    #[tokio::test]
    async fn middleware_generates_traceparent_when_absent() {
        let req = axum::http::Request::builder()
            .uri("/x")
            .body(Body::empty())
            .unwrap();
        let res = app().oneshot(req).await.unwrap();
        let tp = res.headers().get(TRACEPARENT_HEADER).unwrap();
        assert!(is_valid_traceparent(tp.to_str().unwrap()));
    }
}
