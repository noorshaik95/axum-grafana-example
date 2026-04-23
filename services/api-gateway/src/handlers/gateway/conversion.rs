//! HTTP to gRPC conversion utilities.
//!
//! Handles bidirectional conversion between HTTP and gRPC formats.

use axum::{
    body::Body,
    extract::Request,
    http::{HeaderMap, StatusCode},
    response::{IntoResponse, Response},
    Json,
};
use tracing::debug;

use crate::auth::middleware::AuthContext;
use crate::handlers::constants::*;
use crate::handlers::types::GatewayError;
use crate::security::PathValidator;

/// Convert HTTP request to gRPC format.
///
/// This function:
/// - Validates and sanitizes path parameters
/// - Extracts the request body
/// - Includes path parameters in the payload
/// - Includes auth context in metadata
///
/// # Arguments
///
/// * `request` - The HTTP request
/// * `grpc_method` - The target gRPC method
/// * `path_params` - Path parameters extracted from the route
/// * `_headers` - HTTP headers (currently unused)
/// * `auth_context` - Authentication context if available
///
/// # Returns
///
/// JSON-encoded bytes ready for gRPC transmission
pub async fn convert_http_to_grpc(
    request: Request<Body>,
    grpc_method: &str,
    path_params: &std::collections::HashMap<String, String>,
    _headers: &HeaderMap,
    auth_context: Option<&AuthContext>,
) -> Result<Vec<u8>, GatewayError> {
    debug!(
        grpc_method = %grpc_method,
        path_params = ?path_params,
        "Converting HTTP request to gRPC"
    );

    // Validate and sanitize path parameters
    let sanitized_params = sanitize_path_params(path_params)?;

    // Capture the URI query before consuming the request body.
    let query = request.uri().query().map(|s| s.to_string());

    // Extract request body
    let body_bytes = extract_body(request).await?;

    // Parse JSON body
    let mut payload = parse_json_body(&body_bytes)?;

    // T3 Gap 1: merge URI query params into the gRPC payload. GETs arriving
    // with `?foo=bar&baz=qux` previously reached the service as `{}` —
    // backends like scheduling (`?instructor_id=…&from=…&to=…`) then 400'd
    // with "instructor_id required". Query wins over body, but path params
    // still win over query (path is the most specific source).
    merge_query_params(&mut payload, query.as_deref());

    // Merge path parameters (highest precedence after everything else).
    merge_path_params(&mut payload, &sanitized_params);

    // Add auth context (private `_auth_*` + canonical `user_id`/`tenant_id`/
    // `roles` — see T3 Gap 2 in add_auth_context).
    add_auth_context(&mut payload, auth_context);

    // Serialize to bytes
    let payload_bytes = serde_json::to_vec(&payload).map_err(|e| {
        GatewayError::ConversionError(format!("{}: {}", ERR_MSG_SERIALIZE_PAYLOAD, e))
    })?;

    debug!(
        grpc_method = %grpc_method,
        payload_size = payload_bytes.len(),
        "HTTP request converted to gRPC format"
    );

    Ok(payload_bytes)
}

/// Decode a URI query string and merge each `key=value` pair into the JSON
/// payload. Existing body keys always win (body is more expressive: allows
/// nested objects, typed numbers, booleans). Repeated keys in the query
/// collect into a JSON array so `?role=student&role=ta` arrives as
/// `["student", "ta"]`. Bare keys (`?flag`) become `"true"`.
fn merge_query_params(payload: &mut serde_json::Value, query: Option<&str>) {
    let Some(q) = query else { return };
    if q.is_empty() {
        return;
    }
    let Some(obj) = payload.as_object_mut() else {
        return;
    };

    use std::collections::HashMap;
    let mut grouped: HashMap<String, Vec<String>> = HashMap::new();

    for pair in q.split('&') {
        if pair.is_empty() {
            continue;
        }
        let (raw_k, raw_v) = match pair.split_once('=') {
            Some((k, v)) => (k, v),
            None => (pair, "true"),
        };
        let k = match urlencoding::decode(raw_k) {
            Ok(s) => s.into_owned(),
            Err(_) => continue,
        };
        if k.is_empty() {
            continue;
        }
        let v = match urlencoding::decode(raw_v) {
            Ok(s) => s.into_owned(),
            Err(_) => raw_v.to_string(),
        };
        grouped.entry(k).or_default().push(v);
    }

    for (key, mut values) in grouped {
        if obj.contains_key(&key) {
            // Body keys take precedence over query; don't clobber.
            continue;
        }
        let json_val = if values.len() == 1 {
            serde_json::Value::String(values.remove(0))
        } else {
            serde_json::Value::Array(
                values
                    .into_iter()
                    .map(serde_json::Value::String)
                    .collect(),
            )
        };
        obj.insert(key, json_val);
    }
}

/// Convert gRPC response to HTTP format.
///
/// Parses the gRPC response as JSON and creates an HTTP response.
pub async fn convert_grpc_to_http(grpc_response: Vec<u8>) -> Result<Response, GatewayError> {
    debug!(
        response_size = grpc_response.len(),
        "Converting gRPC response to HTTP"
    );

    // Parse the response as JSON
    let json_value: serde_json::Value = serde_json::from_slice(&grpc_response).map_err(|e| {
        GatewayError::ConversionError(format!("{}: {}", ERR_MSG_PARSE_GRPC_RESPONSE, e))
    })?;

    // Create HTTP response with JSON body
    let response = (StatusCode::OK, Json(json_value)).into_response();

    debug!("gRPC response converted to HTTP successfully");

    Ok(response)
}

/// Sanitize path parameters to prevent security issues.
fn sanitize_path_params(
    path_params: &std::collections::HashMap<String, String>,
) -> Result<std::collections::HashMap<String, String>, GatewayError> {
    if path_params.is_empty() {
        return Ok(path_params.clone());
    }

    PathValidator::sanitize_path_params(path_params).map_err(|e| {
        tracing::error!(error = %e, "Path parameter validation failed");
        GatewayError::ConversionError(format!("Invalid path parameter: {}", e))
    })
}

/// Extract request body with size limit.
async fn extract_body(request: Request<Body>) -> Result<bytes::Bytes, GatewayError> {
    axum::body::to_bytes(request.into_body(), MAX_REQUEST_BODY_SIZE)
        .await
        .map_err(|e| GatewayError::ConversionError(format!("{}: {}", ERR_MSG_READ_BODY, e)))
}

/// Parse JSON body from bytes.
fn parse_json_body(body_bytes: &[u8]) -> Result<serde_json::Value, GatewayError> {
    if body_bytes.is_empty() {
        Ok(serde_json::json!({}))
    } else {
        serde_json::from_slice(body_bytes)
            .map_err(|e| GatewayError::ConversionError(format!("{}: {}", ERR_MSG_INVALID_JSON, e)))
    }
}

/// Merge path parameters into payload.
fn merge_path_params(
    payload: &mut serde_json::Value,
    sanitized_params: &std::collections::HashMap<String, String>,
) {
    if !sanitized_params.is_empty() {
        if let Some(obj) = payload.as_object_mut() {
            for (key, value) in sanitized_params {
                obj.insert(key.clone(), serde_json::Value::String(value.clone()));
            }
        }
    }
}

/// Add authentication context to payload.
///
/// T3 Gap 2: alongside the private `_auth_*` keys (used by gateway-internal
/// policy checks), also inject canonical `user_id` / `tenant_id` / `roles`
/// keys so downstream services can read the authenticated caller's identity
/// directly from the gRPC request. Body-supplied values win (caller-supplied
/// overrides trusted context would be a security bug, but path params /
/// explicit body fields are authoritative for lookups where the caller is
/// asking about *another* user, e.g. admin endpoints).
fn add_auth_context(payload: &mut serde_json::Value, auth_context: Option<&AuthContext>) {
    let Some(ctx) = auth_context else { return };
    if !ctx.authenticated {
        return;
    }
    let Some(obj) = payload.as_object_mut() else {
        return;
    };

    if let Some(user_id) = &ctx.user_id {
        // Private gateway-facing key (legacy, still consumed by internal
        // code that already reads `_auth_user_id`).
        obj.insert(
            METADATA_AUTH_USER_ID.to_string(),
            serde_json::Value::String(user_id.clone()),
        );
        // Canonical proto-field-shaped key. Don't overwrite if the body
        // already carried one (e.g. admin querying another user's profile).
        obj.entry("user_id")
            .or_insert_with(|| serde_json::Value::String(user_id.clone()));
    }

    if let Some(tenant_id) = &ctx.tenant_id {
        obj.entry("tenant_id")
            .or_insert_with(|| serde_json::Value::String(tenant_id.clone()));
    }

    if !ctx.roles.is_empty() {
        obj.insert(
            METADATA_AUTH_ROLES.to_string(),
            serde_json::json!(ctx.roles),
        );
        obj.entry("roles")
            .or_insert_with(|| serde_json::json!(ctx.roles));
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    fn authed(user_id: &str, tenant_id: Option<&str>, roles: &[&str]) -> AuthContext {
        AuthContext {
            user_id: Some(user_id.to_string()),
            tenant_id: tenant_id.map(|s| s.to_string()),
            roles: roles.iter().map(|s| s.to_string()).collect(),
            authenticated: true,
        }
    }

    // --- Gap 1: query-param merge --------------------------------------

    #[test]
    fn query_params_merge_single_pair() {
        let mut payload = serde_json::json!({});
        merge_query_params(&mut payload, Some("instructor_id=u1"));
        assert_eq!(payload["instructor_id"], "u1");
    }

    #[test]
    fn query_params_merge_multiple_pairs() {
        let mut payload = serde_json::json!({});
        merge_query_params(
            &mut payload,
            Some("instructor_id=u1&from=2026-01-01&to=2026-12-31"),
        );
        assert_eq!(payload["instructor_id"], "u1");
        assert_eq!(payload["from"], "2026-01-01");
        assert_eq!(payload["to"], "2026-12-31");
    }

    #[test]
    fn query_params_decode_percent_encoding() {
        let mut payload = serde_json::json!({});
        merge_query_params(&mut payload, Some("q=hello%20world&tag=a%26b"));
        assert_eq!(payload["q"], "hello world");
        assert_eq!(payload["tag"], "a&b");
    }

    #[test]
    fn query_params_repeat_key_becomes_array() {
        let mut payload = serde_json::json!({});
        merge_query_params(&mut payload, Some("role=student&role=ta"));
        assert_eq!(payload["role"], serde_json::json!(["student", "ta"]));
    }

    #[test]
    fn query_params_do_not_clobber_body() {
        let mut payload = serde_json::json!({"instructor_id": "body-wins"});
        merge_query_params(&mut payload, Some("instructor_id=query-loses"));
        assert_eq!(payload["instructor_id"], "body-wins");
    }

    #[test]
    fn query_params_empty_or_absent_is_noop() {
        let mut payload = serde_json::json!({"a": 1});
        merge_query_params(&mut payload, None);
        merge_query_params(&mut payload, Some(""));
        assert_eq!(payload, serde_json::json!({"a": 1}));
    }

    #[test]
    fn query_params_bare_key_defaults_to_true_string() {
        let mut payload = serde_json::json!({});
        merge_query_params(&mut payload, Some("debug"));
        assert_eq!(payload["debug"], "true");
    }

    // --- Gap 2: canonical auth-key injection ---------------------------

    #[test]
    fn auth_context_injects_canonical_user_id_and_legacy_underscore_key() {
        let mut payload = serde_json::json!({});
        let ctx = authed("u-1", Some("eastfield"), &["student"]);
        add_auth_context(&mut payload, Some(&ctx));
        assert_eq!(payload["user_id"], "u-1");
        assert_eq!(payload["tenant_id"], "eastfield");
        assert_eq!(payload["roles"], serde_json::json!(["student"]));
        // Private keys still populated for gateway-internal consumers.
        assert_eq!(payload[METADATA_AUTH_USER_ID], "u-1");
        assert_eq!(payload[METADATA_AUTH_ROLES], serde_json::json!(["student"]));
    }

    #[test]
    fn auth_context_does_not_overwrite_body_user_id() {
        // Admin querying *another* user's profile — caller passes
        // `user_id` in the body; the gateway must not clobber it.
        let mut payload = serde_json::json!({"user_id": "target-user"});
        let ctx = authed("admin-1", Some("platform"), &["admin"]);
        add_auth_context(&mut payload, Some(&ctx));
        assert_eq!(payload["user_id"], "target-user");
        // Private key still records the authenticated caller.
        assert_eq!(payload[METADATA_AUTH_USER_ID], "admin-1");
    }

    #[test]
    fn auth_context_skips_when_unauthenticated() {
        let mut payload = serde_json::json!({});
        let ctx = AuthContext::unauthenticated();
        add_auth_context(&mut payload, Some(&ctx));
        assert!(payload.as_object().unwrap().is_empty());
    }

    #[test]
    fn auth_context_omits_tenant_id_when_absent() {
        let mut payload = serde_json::json!({});
        let ctx = authed("u-1", None, &["admin"]);
        add_auth_context(&mut payload, Some(&ctx));
        assert!(!payload.as_object().unwrap().contains_key("tenant_id"));
        assert_eq!(payload["user_id"], "u-1");
    }
}
