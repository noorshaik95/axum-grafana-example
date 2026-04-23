//! Backend service calling utilities.
//!
//! Handles calling backend services via gRPC with proper trace propagation.

use std::collections::HashMap;
use tracing::{debug, error};

use crate::handlers::types::GatewayError;

/// Call backend service via gRPC using dynamic client.
///
/// Extracts trace context from the current OpenTelemetry span and injects it
/// into gRPC metadata for proper trace propagation.
pub async fn call_backend_service(
    channel: tonic::transport::Channel,
    grpc_method: &str,
    request_payload: Vec<u8>,
) -> Result<Vec<u8>, GatewayError> {
    debug!(
        grpc_method = %grpc_method,
        payload_size = request_payload.len(),
        "Calling backend service with dynamic client"
    );

    // Extract service and method names
    let (service_name, method_name) = parse_grpc_method(grpc_method)?;

    // Extract trace context from current span
    let trace_headers = extract_trace_context();

    debug!(
        trace_headers = ?trace_headers,
        "Extracted trace context from current span"
    );

    // Route to appropriate handler
    if service_name == "user.UserService" && is_auth_method(method_name) {
        call_typed_user_service(channel, method_name, request_payload).await
    } else {
        call_dynamic_client(
            channel,
            service_name,
            method_name,
            request_payload,
            trace_headers,
        )
        .await
    }
}

/// Parse gRPC method into service and method names.
fn parse_grpc_method(grpc_method: &str) -> Result<(&str, &str), GatewayError> {
    let service_name = grpc_method
        .split('/')
        .next()
        .ok_or_else(|| GatewayError::ConversionError("Invalid gRPC method format".to_string()))?;

    let method_name = grpc_method
        .split('/')
        .nth(1)
        .ok_or_else(|| GatewayError::ConversionError("Invalid gRPC method format".to_string()))?;

    Ok((service_name, method_name))
}

/// Extract trace context from current OpenTelemetry span.
fn extract_trace_context() -> HashMap<String, String> {
    use tracing_opentelemetry::OpenTelemetrySpanExt;

    let mut trace_headers = HashMap::new();
    let current_span = tracing::Span::current();
    let context = current_span.context();

    opentelemetry::global::get_text_map_propagator(|propagator| {
        propagator.inject_context(&context, &mut trace_headers);
    });

    trace_headers
}

/// Call typed UserService handler for auth methods.
async fn call_typed_user_service(
    channel: tonic::transport::Channel,
    method_name: &str,
    request_payload: Vec<u8>,
) -> Result<Vec<u8>, GatewayError> {
    use crate::handlers::user_service::call_user_service;

    debug!(
        service = "user.UserService",
        method = %method_name,
        "Using typed UserService handler for auth method"
    );

    call_user_service(channel, method_name, request_payload).await
}

/// Call dynamic client for all other services.
async fn call_dynamic_client(
    channel: tonic::transport::Channel,
    service_name: &str,
    method_name: &str,
    request_payload: Vec<u8>,
    trace_headers: HashMap<String, String>,
) -> Result<Vec<u8>, GatewayError> {
    use crate::grpc::DynamicGrpcClient;

    debug!(
        service = %service_name,
        method = %method_name,
        "Using dynamic client"
    );

    let mut client = DynamicGrpcClient::new(channel, service_name.to_string());
    client
        .call(method_name, request_payload, trace_headers)
        .await
        .map_err(|e| {
            error!(
                grpc_method = %format!("{}/{}", service_name, method_name),
                error = %e,
                "Dynamic gRPC call failed"
            );
            // Preserve the tonic Code when available so the HTTP mapper
            // emits 404/400/401/403/… for business errors. Only fall back
            // to opaque BAD_GATEWAY for infra-level failures (reflection,
            // codec, connection) that don't carry a Status.
            match e.grpc_code() {
                Some(code) => GatewayError::GrpcStatus {
                    code,
                    message: e.to_string(),
                },
                None => GatewayError::GrpcCallFailed(e.to_string()),
            }
        })
}

/// Check if a method is an auth-related method.
fn is_auth_method(method_name: &str) -> bool {
    matches!(
        method_name,
        "Login"
            | "Register"
            | "RefreshToken"
            | "ValidateToken"
            | "Logout"
            | "OAuthCallback"
            | "LinkOAuthProvider"
            | "UnlinkOAuthProvider"
            | "GetOAuthProviders"
            | "GetOAuthAuthorizationURL"
            | "HandleOAuthCallback"
            | "GetSAMLAuthRequest"
            | "HandleSAMLAssertion"
            | "GetSAMLMetadata"
            | "SetupMFA"
            | "VerifyMFA"
            | "DisableMFA"
            | "GetMFAStatus"
            | "ValidateMFACode"
    )
}
