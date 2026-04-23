//! Outbound + Kafka propagation helpers for Slate services.
//!
//! The existing `interceptor.rs` covers inbound gRPC extraction; this module
//! adds the counterpart injection helpers (outbound gRPC, Kafka headers) plus
//! small utilities for tagging spans with the standard Slate correlation
//! attributes (`request_id`, `tenant.slug`).
//!
//! All public items here are feature-gated on `grpc` + `observability` so the
//! crate can be compiled in lean profiles (e.g. tests or workers that don't
//! link OTel).

/// Slate-standard header/metadata keys.
pub const TRACEPARENT_HEADER: &str = "traceparent";
pub const TRACESTATE_HEADER: &str = "tracestate";
pub const REQUEST_ID_HEADER: &str = "x-request-id";
pub const TENANT_SLUG_HEADER: &str = "x-tenant-slug";

#[cfg(all(feature = "grpc", feature = "observability"))]
pub use self::grpc_impl::*;

#[cfg(all(feature = "grpc", feature = "observability"))]
mod grpc_impl {
    use super::*;
    use opentelemetry::trace::TraceContextExt;
    use tonic::metadata::{AsciiMetadataValue, MetadataMap};
    use tonic::Request;
    use tracing::Span;
    use tracing_opentelemetry::OpenTelemetrySpanExt;

    /// Write the current span's W3C traceparent (plus request_id / tenant_slug
    /// if supplied) onto an outbound tonic `Request`.
    ///
    /// Call this right before handing the request to a gRPC client stub. The
    /// request is mutated in-place and returned for chaining.
    pub fn inject_traceparent<T>(
        request: &mut Request<T>,
        request_id: Option<&str>,
        tenant_slug: Option<&str>,
    ) {
        let md = request.metadata_mut();
        write_traceparent_into(md);

        if let Some(rid) = request_id.filter(|s| !s.is_empty()) {
            if let Ok(v) = AsciiMetadataValue::try_from(rid) {
                md.insert(REQUEST_ID_HEADER, v);
            }
        }
        if let Some(slug) = tenant_slug.filter(|s| !s.is_empty()) {
            if let Ok(v) = AsciiMetadataValue::try_from(slug) {
                md.insert(TENANT_SLUG_HEADER, v);
            }
        }
    }

    fn write_traceparent_into(md: &mut MetadataMap) {
        let span = Span::current();
        let cx = span.context();
        let sc = cx.span().span_context().clone();
        if !sc.is_valid() {
            return;
        }

        let traceparent = format!(
            "00-{}-{}-{:02x}",
            sc.trace_id(),
            sc.span_id(),
            sc.trace_flags().to_u8()
        );
        if let Ok(v) = AsciiMetadataValue::try_from(traceparent.as_str()) {
            md.insert(TRACEPARENT_HEADER, v);
        }

        let ts = sc.trace_state();
        let ts_header = ts.header();
        if !ts_header.is_empty() {
            if let Ok(v) = AsciiMetadataValue::try_from(ts_header.as_str()) {
                md.insert(TRACESTATE_HEADER, v);
            }
        }
    }

    /// Tag the given span with `request_id` and `tenant.slug` attributes. Call
    /// once at the top of every gRPC handler so Grafana Tempo traces carry the
    /// Slate correlation fields consistently.
    pub fn tag_span_with_correlation(
        span: &Span,
        request_id: Option<&str>,
        tenant_slug: Option<&str>,
    ) {
        if let Some(v) = request_id.filter(|s| !s.is_empty()) {
            span.record("request_id", tracing::field::display(v));
        }
        if let Some(v) = tenant_slug.filter(|s| !s.is_empty()) {
            span.record("tenant.slug", tracing::field::display(v));
        }
    }

    /// Convenience: pull the `x-request-id` / `x-tenant-slug` values off an
    /// incoming tonic `Request`. Returns `(request_id, tenant_slug)` where
    /// each entry is `None` when absent.
    pub fn read_correlation_from_incoming<T>(
        request: &Request<T>,
    ) -> (Option<String>, Option<String>) {
        let md = request.metadata();
        let req_id = md
            .get(REQUEST_ID_HEADER)
            .and_then(|v| v.to_str().ok())
            .map(|s| s.to_string());
        let tenant = md
            .get(TENANT_SLUG_HEADER)
            .and_then(|v| v.to_str().ok())
            .map(|s| s.to_string());
        (req_id, tenant)
    }
}

// ----- Kafka headers (no tonic dependency needed) -----------------------------

/// Minimal Kafka header shape. Compatible with rdkafka / kafka-rust which both
/// expose headers as `(key: String, value: Option<Vec<u8>>)`.
#[derive(Debug, Clone)]
pub struct KafkaHeader {
    pub key: String,
    pub value: Vec<u8>,
}

impl KafkaHeader {
    pub fn new<K: Into<String>, V: Into<Vec<u8>>>(k: K, v: V) -> Self {
        Self {
            key: k.into(),
            value: v.into(),
        }
    }
}

/// Build Kafka headers carrying traceparent + correlation for a produced
/// message. Pass the active span's trace context via `traceparent` (use
/// `current_traceparent()` below) to avoid this module depending on the full
/// OTel stack in feature-lean builds.
pub fn kafka_headers_for(
    traceparent: Option<&str>,
    tracestate: Option<&str>,
    request_id: Option<&str>,
    tenant_slug: Option<&str>,
) -> Vec<KafkaHeader> {
    let mut out = Vec::new();
    if let Some(tp) = traceparent.filter(|s| !s.is_empty()) {
        out.push(KafkaHeader::new(TRACEPARENT_HEADER, tp.as_bytes().to_vec()));
    }
    if let Some(ts) = tracestate.filter(|s| !s.is_empty()) {
        out.push(KafkaHeader::new(TRACESTATE_HEADER, ts.as_bytes().to_vec()));
    }
    if let Some(rid) = request_id.filter(|s| !s.is_empty()) {
        out.push(KafkaHeader::new(REQUEST_ID_HEADER, rid.as_bytes().to_vec()));
    }
    if let Some(slug) = tenant_slug.filter(|s| !s.is_empty()) {
        out.push(KafkaHeader::new(
            TENANT_SLUG_HEADER,
            slug.as_bytes().to_vec(),
        ));
    }
    out
}

/// Extracted carrier from consumed Kafka headers.
#[derive(Debug, Clone, Default)]
pub struct KafkaTraceCarrier {
    pub traceparent: Option<String>,
    pub tracestate: Option<String>,
    pub request_id: Option<String>,
    pub tenant_slug: Option<String>,
}

/// Parse Slate correlation state back off a consumed Kafka message's headers.
pub fn context_from_kafka_headers(headers: &[KafkaHeader]) -> KafkaTraceCarrier {
    let mut c = KafkaTraceCarrier::default();
    for h in headers {
        let val = std::str::from_utf8(&h.value).ok().map(|s| s.to_string());
        match h.key.as_str() {
            TRACEPARENT_HEADER => c.traceparent = val,
            TRACESTATE_HEADER => c.tracestate = val,
            REQUEST_ID_HEADER => c.request_id = val,
            TENANT_SLUG_HEADER => c.tenant_slug = val,
            _ => {}
        }
    }
    c
}

/// Return the current span's W3C traceparent string, if it has a valid span
/// context. Uses the global OTel tracer, so callers must have initialized
/// tracing on startup. Returns `None` when observability isn't compiled in.
#[cfg(feature = "observability")]
pub fn current_traceparent() -> Option<String> {
    use opentelemetry::trace::TraceContextExt;
    use tracing::Span;
    use tracing_opentelemetry::OpenTelemetrySpanExt;

    let span = Span::current();
    let cx = span.context();
    let sc = cx.span().span_context().clone();
    if !sc.is_valid() {
        return None;
    }
    Some(format!(
        "00-{}-{}-{:02x}",
        sc.trace_id(),
        sc.span_id(),
        sc.trace_flags().to_u8()
    ))
}

#[cfg(not(feature = "observability"))]
pub fn current_traceparent() -> Option<String> {
    None
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn kafka_headers_include_only_present_fields() {
        let h = kafka_headers_for(
            Some("00-aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa-bbbbbbbbbbbbbbbb-01"),
            None,
            Some("req-1"),
            None,
        );

        let keys: Vec<&str> = h.iter().map(|x| x.key.as_str()).collect();
        assert!(keys.contains(&TRACEPARENT_HEADER));
        assert!(keys.contains(&REQUEST_ID_HEADER));
        assert!(!keys.contains(&TRACESTATE_HEADER));
        assert!(!keys.contains(&TENANT_SLUG_HEADER));
    }

    #[test]
    fn round_trip_kafka_carrier() {
        let headers = kafka_headers_for(
            Some("00-aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa-bbbbbbbbbbbbbbbb-01"),
            Some("vendor=foo"),
            Some("req-42"),
            Some("eastfield"),
        );
        let carrier = context_from_kafka_headers(&headers);
        assert_eq!(carrier.request_id.as_deref(), Some("req-42"));
        assert_eq!(carrier.tenant_slug.as_deref(), Some("eastfield"));
        assert_eq!(carrier.tracestate.as_deref(), Some("vendor=foo"));
        assert!(carrier.traceparent.as_deref().unwrap().starts_with("00-"));
    }

    #[test]
    fn empty_strings_are_ignored() {
        let h = kafka_headers_for(Some(""), Some(""), Some(""), Some(""));
        assert!(h.is_empty());
    }
}
