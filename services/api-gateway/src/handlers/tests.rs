use super::gateway::conversion::{convert_grpc_to_http, convert_http_to_grpc};
use super::types::map_grpc_error_to_status;
use super::*;
use axum::http::StatusCode;

#[test]
fn test_map_grpc_error_to_status() {
    assert_eq!(
        map_grpc_error_to_status(&GatewayError::NotFound),
        StatusCode::NOT_FOUND
    );

    assert_eq!(
        map_grpc_error_to_status(&GatewayError::RateLimitExceeded),
        StatusCode::TOO_MANY_REQUESTS
    );

    assert_eq!(
        map_grpc_error_to_status(&GatewayError::ServiceUnavailable("test".to_string())),
        StatusCode::SERVICE_UNAVAILABLE
    );

    assert_eq!(
        map_grpc_error_to_status(&GatewayError::Timeout),
        StatusCode::GATEWAY_TIMEOUT
    );
}

#[tokio::test]
async fn test_convert_http_to_grpc_empty_body() {
    use axum::body::Body;
    use axum::http::{HeaderMap, Request};

    let request = Request::builder().uri("/test").body(Body::empty()).unwrap();

    let headers = HeaderMap::new();
    let path_params = std::collections::HashMap::new();

    let result =
        convert_http_to_grpc(request, "test.Service/Method", &path_params, &headers, None).await;

    assert!(result.is_ok());
    let payload = result.unwrap();
    let json: serde_json::Value = serde_json::from_slice(&payload).unwrap();
    assert!(json.is_object());
}

#[tokio::test]
async fn test_convert_http_to_grpc_with_path_params() {
    use axum::body::Body;
    use axum::http::{HeaderMap, Request};

    let body = serde_json::json!({"name": "test"});
    let body_bytes = serde_json::to_vec(&body).unwrap();

    let request = Request::builder()
        .uri("/test")
        .body(Body::from(body_bytes))
        .unwrap();

    let headers = HeaderMap::new();
    let mut path_params = std::collections::HashMap::new();
    path_params.insert("id".to_string(), "123".to_string());

    let result =
        convert_http_to_grpc(request, "test.Service/Method", &path_params, &headers, None).await;

    assert!(result.is_ok());
    let payload = result.unwrap();
    let json: serde_json::Value = serde_json::from_slice(&payload).unwrap();

    assert_eq!(json["name"], "test");
    assert_eq!(json["id"], "123");
}

#[tokio::test]
async fn test_convert_http_to_grpc_with_auth_context() {
    use crate::auth::middleware::AuthContext;
    use axum::body::Body;
    use axum::http::{HeaderMap, Request};

    let request = Request::builder().uri("/test").body(Body::empty()).unwrap();

    let headers = HeaderMap::new();
    let path_params = std::collections::HashMap::new();

    let auth_context = AuthContext {
        user_id: Some("user123".to_string()),
        tenant_id: None,
        roles: vec!["admin".to_string()],
        authenticated: true,
    };

    let result = convert_http_to_grpc(
        request,
        "test.Service/Method",
        &path_params,
        &headers,
        Some(&auth_context),
    )
    .await;

    assert!(result.is_ok());
    let payload = result.unwrap();
    let json: serde_json::Value = serde_json::from_slice(&payload).unwrap();

    assert_eq!(json["_auth_user_id"], "user123");
    assert_eq!(json["_auth_roles"], serde_json::json!(["admin"]));
}

#[tokio::test]
async fn test_convert_grpc_to_http() {
    let grpc_response = serde_json::json!({
        "status": "success",
        "data": {"id": 1, "name": "test"}
    });

    let response_bytes = serde_json::to_vec(&grpc_response).unwrap();

    let result = convert_grpc_to_http(response_bytes).await;
    assert!(result.is_ok());
}

#[tokio::test]
async fn test_convert_grpc_to_http_invalid_json() {
    let invalid_bytes = vec![0xFF, 0xFE, 0xFD];

    let result = convert_grpc_to_http(invalid_bytes).await;
    assert!(result.is_err());

    match result.unwrap_err() {
        GatewayError::ConversionError(msg) => {
            assert!(msg.contains("Failed to parse gRPC response"));
        }
        _ => panic!("Expected ConversionError"),
    }
}
