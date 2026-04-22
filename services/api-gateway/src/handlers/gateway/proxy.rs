//! HTTP reverse-proxy handler (§1a).
//!
//! Forwards the incoming Axum request to a plain HTTP upstream without any
//! gRPC transcoding. Used for a small handful of user-auth-service routes
//! (SSO, impersonation, MFA-reset) that do not have a gRPC counterpart.
//!
//! Header policy:
//! * Forwarded as-is if present: `Authorization`, `Content-Type`, `Accept`,
//!   `X-Request-ID`, `traceparent`, `tracestate`, `X-Tenant-Slug`.
//! * Stripped: hop-by-hop headers (`Host`, `Connection`, `Transfer-Encoding`,
//!   `Content-Length`, `Keep-Alive`, `Upgrade`, `Proxy-Authenticate`,
//!   `Proxy-Authorization`, `TE`, `Trailers`).
//! * `X-Request-ID` is echoed back on the response even if the upstream
//!   omits it, per the trace-propagation contract.

use axum::{
    body::Body,
    extract::Request,
    http::{HeaderMap, HeaderName, HeaderValue, Method, StatusCode},
    response::Response,
};
use std::collections::HashMap;
use tracing::{debug, error, warn};

use crate::handlers::types::GatewayError;

/// Hop-by-hop headers that must NOT be forwarded to the upstream.
/// See RFC 7230 §6.1.
const HOP_BY_HOP: &[&str] = &[
    "host",
    "connection",
    "keep-alive",
    "proxy-authenticate",
    "proxy-authorization",
    "te",
    "trailers",
    "transfer-encoding",
    "upgrade",
    "content-length",
];

/// Build the upstream URL by joining `base_url` with the original request path
/// (with path params already resolved by the router).
pub(crate) fn build_upstream_url(base_url: &str, path: &str, query: Option<&str>) -> String {
    let trimmed_base = base_url.trim_end_matches('/');
    let path_with_leading_slash = if path.starts_with('/') {
        path.to_string()
    } else {
        format!("/{}", path)
    };
    match query {
        Some(q) if !q.is_empty() => format!("{}{}?{}", trimmed_base, path_with_leading_slash, q),
        _ => format!("{}{}", trimmed_base, path_with_leading_slash),
    }
}

/// Copy forwarded request headers from an Axum `HeaderMap` into a
/// `reqwest::header::HeaderMap`, dropping hop-by-hop headers.
pub(crate) fn copy_forwardable_headers(
    src: &HeaderMap,
) -> reqwest::header::HeaderMap {
    let mut dst = reqwest::header::HeaderMap::with_capacity(src.len());
    for (name, value) in src.iter() {
        let name_str = name.as_str().to_ascii_lowercase();
        if HOP_BY_HOP.contains(&name_str.as_str()) {
            continue;
        }
        if let (Ok(header_name), Ok(header_value)) = (
            reqwest::header::HeaderName::from_bytes(name.as_str().as_bytes()),
            reqwest::header::HeaderValue::from_bytes(value.as_bytes()),
        ) {
            dst.append(header_name, header_value);
        }
    }
    dst
}

/// Convert an Axum HTTP method to a reqwest method. Both re-export `http`
/// types so this is a cheap conversion.
fn to_reqwest_method(method: &Method) -> reqwest::Method {
    reqwest::Method::from_bytes(method.as_str().as_bytes())
        .unwrap_or(reqwest::Method::GET)
}

/// Forward an incoming Axum request to a plain HTTP upstream.
///
/// Preserves the method, path, query string, body, and all non-hop-by-hop
/// headers. Always echoes `X-Request-ID` on the response.
pub async fn forward_http_request(
    client: &reqwest::Client,
    proxy_base_url: &str,
    request: Request<Body>,
    _path_params: &HashMap<String, String>,
    original_path: &str,
) -> Result<Response, GatewayError> {
    let method = request.method().clone();
    let query = request.uri().query().map(|q| q.to_string());

    // Snapshot the inbound X-Request-ID so we can echo it on the response
    // even if the upstream forgets to.
    let request_id = request
        .headers()
        .get("x-request-id")
        .and_then(|v| v.to_str().ok())
        .map(|s| s.to_string());

    let forwarded_headers = copy_forwardable_headers(request.headers());

    // Read the body. reqwest::Body::wrap_stream would preserve streaming,
    // but the §1a routes are all tiny JSON payloads — buffer for simplicity.
    let (_parts, body) = request.into_parts();
    let body_bytes = match axum::body::to_bytes(body, usize::MAX).await {
        Ok(b) => b,
        Err(e) => {
            warn!(error = %e, "Failed to read request body for HTTP proxy");
            return Err(GatewayError::InternalError(format!(
                "failed to read request body: {}",
                e
            )));
        }
    };

    let upstream_url = build_upstream_url(proxy_base_url, original_path, query.as_deref());

    debug!(
        upstream = %upstream_url,
        method = %method,
        body_bytes = body_bytes.len(),
        "HTTP-proxy: forwarding request to upstream"
    );

    let reqwest_method = to_reqwest_method(&method);
    let upstream_response = client
        .request(reqwest_method, &upstream_url)
        .headers(forwarded_headers)
        .body(body_bytes.to_vec())
        .send()
        .await
        .map_err(|e| {
            error!(
                upstream = %upstream_url,
                error = %e,
                "HTTP-proxy: upstream request failed"
            );
            if e.is_timeout() {
                GatewayError::Timeout
            } else {
                GatewayError::ServiceUnavailable(format!("upstream request failed: {}", e))
            }
        })?;

    build_axum_response(upstream_response, request_id.as_deref()).await
}

/// Convert a `reqwest::Response` into an `axum::response::Response`,
/// preserving status, headers (minus hop-by-hop), and body.
async fn build_axum_response(
    upstream: reqwest::Response,
    inbound_request_id: Option<&str>,
) -> Result<Response, GatewayError> {
    let status = StatusCode::from_u16(upstream.status().as_u16()).unwrap_or(StatusCode::BAD_GATEWAY);

    // Copy response headers minus hop-by-hop.
    let mut response_headers = HeaderMap::new();
    for (name, value) in upstream.headers().iter() {
        let name_str = name.as_str().to_ascii_lowercase();
        if HOP_BY_HOP.contains(&name_str.as_str()) {
            continue;
        }
        if let (Ok(header_name), Ok(header_value)) = (
            HeaderName::from_bytes(name.as_str().as_bytes()),
            HeaderValue::from_bytes(value.as_bytes()),
        ) {
            response_headers.append(header_name, header_value);
        }
    }

    // Echo X-Request-ID if the upstream didn't set one.
    if !response_headers.contains_key("x-request-id") {
        if let Some(rid) = inbound_request_id {
            if let Ok(v) = HeaderValue::from_str(rid) {
                response_headers.insert(HeaderName::from_static("x-request-id"), v);
            }
        }
    }

    let body_bytes = upstream.bytes().await.map_err(|e| {
        warn!(error = %e, "HTTP-proxy: failed to read upstream body");
        GatewayError::ServiceUnavailable(format!("failed to read upstream body: {}", e))
    })?;

    let mut response = Response::builder()
        .status(status)
        .body(Body::from(body_bytes))
        .map_err(|e| GatewayError::InternalError(format!("failed to build response: {}", e)))?;

    *response.headers_mut() = response_headers;
    Ok(response)
}

#[cfg(test)]
mod tests {
    use super::*;
    use axum::http::HeaderValue;

    #[test]
    fn build_upstream_url_joins_base_and_path() {
        let url = build_upstream_url(
            "http://user-auth-service:8081",
            "/api/auth/sso/initiate",
            None,
        );
        assert_eq!(url, "http://user-auth-service:8081/api/auth/sso/initiate");
    }

    #[test]
    fn build_upstream_url_strips_trailing_slash_on_base() {
        let url = build_upstream_url(
            "http://user-auth-service:8081/",
            "/api/auth/impersonate",
            None,
        );
        assert_eq!(url, "http://user-auth-service:8081/api/auth/impersonate");
    }

    #[test]
    fn build_upstream_url_preserves_query_string() {
        let url = build_upstream_url(
            "http://user-auth-service:8081",
            "/api/auth/sso/callback",
            Some("code=abc123&state=xyz"),
        );
        assert_eq!(
            url,
            "http://user-auth-service:8081/api/auth/sso/callback?code=abc123&state=xyz"
        );
    }

    #[test]
    fn build_upstream_url_adds_leading_slash_if_missing() {
        let url = build_upstream_url("http://svc", "api/path", None);
        assert_eq!(url, "http://svc/api/path");
    }

    #[test]
    fn copy_forwardable_headers_preserves_auth_and_trace() {
        let mut src = HeaderMap::new();
        src.insert("authorization", HeaderValue::from_static("Bearer abc"));
        src.insert("x-request-id", HeaderValue::from_static("req-123"));
        src.insert(
            "traceparent",
            HeaderValue::from_static("00-0af7651916cd43dd8448eb211c80319c-b7ad6b7169203331-01"),
        );
        src.insert("tracestate", HeaderValue::from_static("vendor=abc"));
        src.insert("x-tenant-slug", HeaderValue::from_static("eastfield"));
        src.insert("content-type", HeaderValue::from_static("application/json"));
        src.insert("accept", HeaderValue::from_static("application/json"));

        let dst = copy_forwardable_headers(&src);
        assert!(dst.contains_key("authorization"));
        assert!(dst.contains_key("x-request-id"));
        assert!(dst.contains_key("traceparent"));
        assert!(dst.contains_key("tracestate"));
        assert!(dst.contains_key("x-tenant-slug"));
        assert!(dst.contains_key("content-type"));
        assert!(dst.contains_key("accept"));
    }

    #[test]
    fn copy_forwardable_headers_strips_hop_by_hop() {
        let mut src = HeaderMap::new();
        src.insert("host", HeaderValue::from_static("evil.example.com"));
        src.insert("connection", HeaderValue::from_static("keep-alive"));
        src.insert("transfer-encoding", HeaderValue::from_static("chunked"));
        src.insert("content-length", HeaderValue::from_static("100"));
        src.insert("upgrade", HeaderValue::from_static("websocket"));
        src.insert("proxy-authorization", HeaderValue::from_static("Basic x"));
        src.insert("te", HeaderValue::from_static("trailers"));

        let dst = copy_forwardable_headers(&src);
        assert!(!dst.contains_key("host"));
        assert!(!dst.contains_key("connection"));
        assert!(!dst.contains_key("transfer-encoding"));
        assert!(!dst.contains_key("content-length"));
        assert!(!dst.contains_key("upgrade"));
        assert!(!dst.contains_key("proxy-authorization"));
        assert!(!dst.contains_key("te"));
    }

    #[test]
    fn copy_forwardable_headers_is_case_insensitive_for_hop_by_hop() {
        let mut src = HeaderMap::new();
        // Axum/http lowercases header names by default, but double-check.
        src.insert("host", HeaderValue::from_static("x"));
        src.insert("authorization", HeaderValue::from_static("Bearer y"));

        let dst = copy_forwardable_headers(&src);
        assert!(!dst.contains_key("host"));
        assert!(dst.contains_key("authorization"));
    }
}
