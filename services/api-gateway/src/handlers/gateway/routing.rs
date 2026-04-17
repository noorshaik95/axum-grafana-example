//! Request routing and processing pipeline.
//!
//! Handles the main request processing pipeline including routing decisions,
//! rate limiting, and coordination of backend service calls.

use axum::{
    body::Body,
    extract::{ConnectInfo, Request, State},
    http::HeaderMap,
    response::Response,
};
use std::net::SocketAddr;
use std::sync::Arc;
use std::time::Instant;
use tracing::{debug, error, info};

use crate::auth::middleware::AuthContext;
use crate::handlers::constants::*;
use crate::handlers::types::GatewayError;
use crate::shared::state::AppState;
use common_rust::observability::extract_trace_id_from_span;

use super::circuit_breaker::call_with_circuit_breaker;
use super::conversion::{convert_grpc_to_http, convert_http_to_grpc};
use super::metrics::record_success_metrics;
use super::rate_limiting::apply_rate_limit;

/// Process incoming gateway request.
///
/// This is the main request processing pipeline that:
/// 1. Routes the request to determine target service
/// 2. Applies rate limiting
/// 3. Converts HTTP request to gRPC
/// 4. Calls backend service via gRPC client pool
/// 5. Converts gRPC response to HTTP
/// 6. Handles errors and emits metrics
#[tracing::instrument(name = "gateway_handler_inner", skip(state, addr, headers, request))]
pub async fn process_request(
    State(state): State<Arc<AppState>>,
    ConnectInfo(addr): ConnectInfo<SocketAddr>,
    headers: HeaderMap,
    request: Request<Body>,
) -> Result<Response, GatewayError> {
    let start_time = Instant::now();
    let path = request.uri().path().to_string();
    let method = request.method().as_str().to_string();
    let trace_id = extract_trace_id_from_span();

    log_request_start(&path, &method, addr.ip(), &trace_id);

    // Skip gateway processing for system endpoints
    if is_system_endpoint(&path) {
        debug!(path = %path, "Skipping gateway processing for system endpoint");
        return Err(GatewayError::NotFound);
    }

    // Extract real client IP and apply rate limiting
    let client_ip = state.client_ip_extractor.extract_client_ip(&request);
    apply_rate_limit(&state, client_ip, &path, &method, addr.ip(), start_time).await?;

    // Get routing decision
    let routing_decision = get_routing_decision(&request, &path, &method, &state, start_time)?;
    log_routing_decision(&routing_decision, &trace_id);

    // Get auth context and service channel
    let auth_context = request.extensions().get::<AuthContext>().cloned();
    let service_channel = get_service_channel(&state, &routing_decision, &path, &method)?;

    // Convert HTTP to gRPC
    let grpc_request = convert_request_to_grpc(
        &state,
        request,
        &routing_decision,
        &headers,
        auth_context.as_ref(),
        &path,
        &method,
    )
    .await?;

    log_backend_call(&routing_decision, grpc_request.len(), &trace_id);

    // Call backend service with circuit breaker
    let grpc_response = call_with_circuit_breaker(
        &state,
        &routing_decision,
        service_channel,
        grpc_request,
        &path,
        &method,
        start_time,
    )
    .await?;

    log_backend_response(&routing_decision, grpc_response.len(), &trace_id);

    // Convert gRPC to HTTP
    let http_response =
        convert_response_to_http(&state, grpc_response, &routing_decision, &path, &method).await?;

    // Record success metrics and log completion
    record_success_metrics(&state, &routing_decision, &path, &method, start_time);
    log_request_completion(&path, &method, &routing_decision, start_time, &trace_id);

    Ok(http_response)
}

/// Log request start information.
fn log_request_start(path: &str, method: &str, client_ip: std::net::IpAddr, trace_id: &str) {
    info!(
        path = %path,
        method = %method,
        client_ip = %client_ip,
        trace_id = %trace_id,
        "Processing gateway request"
    );
}

/// Log routing decision information.
fn log_routing_decision(routing_decision: &crate::router::RoutingDecision, trace_id: &str) {
    debug!(
        service = %routing_decision.service,
        grpc_method = %routing_decision.grpc_method,
        path_params = ?routing_decision.path_params,
        trace_id = %trace_id,
        "Request routed to backend service"
    );
}

/// Log backend service call information.
fn log_backend_call(
    routing_decision: &crate::router::RoutingDecision,
    payload_size: usize,
    trace_id: &str,
) {
    debug!(
        service = %routing_decision.service,
        grpc_method = %routing_decision.grpc_method,
        payload_size = payload_size,
        trace_id = %trace_id,
        "Calling backend service via gRPC"
    );
}

/// Log backend service response information.
fn log_backend_response(
    routing_decision: &crate::router::RoutingDecision,
    response_size: usize,
    trace_id: &str,
) {
    info!(
        service = %routing_decision.service,
        trace_id = %trace_id,
        response_size = response_size,
        "Received response from backend service"
    );
}

/// Log request completion information.
fn log_request_completion(
    path: &str,
    method: &str,
    routing_decision: &crate::router::RoutingDecision,
    start_time: Instant,
    trace_id: &str,
) {
    info!(
        path = %path,
        method = %method,
        service = %routing_decision.service,
        duration_ms = start_time.elapsed().as_millis(),
        trace_id = %trace_id,
        "Request completed successfully"
    );
}

/// Convert HTTP request to gRPC format with error handling.
async fn convert_request_to_grpc(
    state: &Arc<AppState>,
    request: Request<Body>,
    routing_decision: &crate::router::RoutingDecision,
    headers: &HeaderMap,
    auth_context: Option<&AuthContext>,
    path: &str,
    method: &str,
) -> Result<Vec<u8>, GatewayError> {
    convert_http_to_grpc(
        request,
        &routing_decision.grpc_method,
        &routing_decision.path_params,
        headers,
        auth_context,
    )
    .await
    .map_err(|e| {
        error!(
            service = %routing_decision.service,
            error = %e,
            "Failed to convert HTTP request to gRPC"
        );
        state
            .metrics
            .request_counter
            .with_label_values(&[path, method, "400"])
            .inc();
        e
    })
}

/// Convert gRPC response to HTTP format with error handling.
async fn convert_response_to_http(
    state: &Arc<AppState>,
    grpc_response: Vec<u8>,
    routing_decision: &crate::router::RoutingDecision,
    path: &str,
    method: &str,
) -> Result<Response, GatewayError> {
    convert_grpc_to_http(grpc_response).await.map_err(|e| {
        error!(
            service = %routing_decision.service,
            error = %e,
            "Failed to convert gRPC response to HTTP"
        );
        state
            .metrics
            .request_counter
            .with_label_values(&[path, method, "500"])
            .inc();
        e
    })
}

/// Check if path is a system endpoint.
fn is_system_endpoint(path: &str) -> bool {
    path == SYSTEM_PATH_HEALTH || path == SYSTEM_PATH_METRICS
}

/// Get routing decision from request extensions.
fn get_routing_decision(
    _request: &Request<Body>,
    path: &str,
    method: &str,
    state: &Arc<AppState>,
    start_time: Instant,
) -> Result<crate::router::RoutingDecision, GatewayError> {
    // Route the request - use try_read to avoid blocking
    let router_guard = state.router_lock.try_read()
        .map_err(|_| GatewayError::InternalError("Failed to acquire router lock".to_string()))?;
    router_guard.route(path, method).map_err(|e| {
        let duration_ms = start_time.elapsed().as_millis();
        error!(
            path = %path,
            method = %method,
            duration_ms = %duration_ms,
            error = %e,
            "Failed to route request"
        );
        GatewayError::from(e)
    })
}

/// Get service channel from the gRPC pool.
fn get_service_channel(
    state: &Arc<AppState>,
    routing_decision: &crate::router::RoutingDecision,
    path: &str,
    method: &str,
) -> Result<tonic::transport::Channel, GatewayError> {
    state
        .grpc_pool
        .get_channel(&routing_decision.service)
        .map_err(|e| {
            error!(
                service = %routing_decision.service,
                path = %path,
                method = %method,
                error = %e,
                "Service channel not found in pool"
            );
            GatewayError::ServiceUnavailable(format!(
                "Service {} not available: {}",
                &*routing_decision.service, e
            ))
        })
}
