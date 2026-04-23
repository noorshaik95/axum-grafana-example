//! X-Request-ID + W3C traceparent middleware.
//!
//! Behavior:
//! - If the incoming request carries `X-Request-ID`, reuse it; otherwise generate
//!   a UUIDv4. Always echo the value back on the response.
//! - Parse the incoming `traceparent` (W3C Trace Context v00). If missing,
//!   derive a parent span from the current OpenTelemetry span (or generate a
//!   new root) so downstream services always receive a traceparent.
//! - Record `request_id` as a span attribute and tag the request's extensions
//!   so downstream handlers (including gRPC outbound) can propagate it.

use axum::{
    body::Body,
    extract::Request,
    http::{header::HeaderName, HeaderValue, Response},
    middleware::Next,
};
use std::sync::OnceLock;
use tracing::Span;
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

/// Middleware: stamp every request with X-Request-ID + traceparent.
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

    let span = Span::current();
    span.record("request_id", tracing::field::display(&request_id));
    span.record("traceparent", tracing::field::display(&traceparent));

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
