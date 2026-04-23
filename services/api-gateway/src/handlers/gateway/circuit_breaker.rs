//! Circuit breaker integration for backend service calls.
//!
//! Handles circuit breaker protection for gRPC calls to backend services.
//!
//! Breaker-counting policy (R2):
//! - Infra failures (Unavailable, Internal/Unknown/DataLoss, connection
//!   errors, DeadlineExceeded, timeouts) count toward the failure
//!   threshold — these represent a genuinely degraded upstream.
//! - Client/business gRPC codes (NotFound, InvalidArgument, Unauthenticated,
//!   PermissionDenied, AlreadyExists, FailedPrecondition, OutOfRange,
//!   Unimplemented, ResourceExhausted, Cancelled) are **not** counted;
//!   they are legitimate 4xx outcomes. Counting them previously caused
//!   cascading 503s after small bursts of profile-404s.

use std::sync::Arc;
use std::time::Instant;
use tracing::{error, warn};

use crate::handlers::types::GatewayError;
use crate::shared::state::AppState;
use common_rust::observability::extract_trace_id_from_span;

use super::backend::call_backend_service;

/// `true` if a `GatewayError` represents a business/client outcome that
/// should NOT count toward the circuit-breaker failure threshold.
fn is_client_error(err: &GatewayError) -> bool {
    match err {
        GatewayError::GrpcStatus { code, .. } => {
            crate::grpc::types::is_client_side_grpc_code(*code)
        }
        // Route/rate/conversion errors are shaped as 4xx by the HTTP mapper
        // and originate in the gateway itself — they never indicate an
        // unhealthy upstream.
        GatewayError::RouteNotFound(_)
        | GatewayError::RateLimitExceeded
        | GatewayError::ConversionError(_)
        | GatewayError::NotFound => true,
        _ => false,
    }
}

/// Call backend service with circuit breaker protection.
///
/// If a circuit breaker is configured for the service, wraps the call with
/// circuit breaker logic. Otherwise, calls directly.
pub async fn call_with_circuit_breaker(
    state: &Arc<AppState>,
    routing_decision: &crate::router::RoutingDecision,
    service_channel: tonic::transport::Channel,
    grpc_request: Vec<u8>,
    path: &str,
    method: &str,
    start_time: Instant,
) -> Result<Vec<u8>, GatewayError> {
    if let Some(circuit_breaker) = state
        .grpc_pool
        .get_circuit_breaker(&routing_decision.service)
    {
        handle_with_breaker(
            state,
            routing_decision,
            service_channel,
            grpc_request,
            circuit_breaker,
            path,
            method,
            start_time,
        )
        .await
    } else {
        handle_without_breaker(
            state,
            routing_decision,
            service_channel,
            grpc_request,
            path,
            method,
        )
        .await
    }
}

/// Handle call with circuit breaker enabled.
///
/// Wraps `call_backend_service` so the breaker only sees infra-level errors
/// as failures. Business/client gRPC codes (NotFound, InvalidArgument,
/// Unauthenticated, …) are returned as `Ok(Err(business_error))` so the
/// breaker counts them as successes, then we unwrap the inner `Err` back
/// to the caller unchanged.
#[allow(clippy::too_many_arguments)]
async fn handle_with_breaker(
    state: &Arc<AppState>,
    routing_decision: &crate::router::RoutingDecision,
    service_channel: tonic::transport::Channel,
    grpc_request: Vec<u8>,
    circuit_breaker: common_rust::circuit_breaker::CircuitBreaker,
    path: &str,
    method: &str,
    start_time: Instant,
) -> Result<Vec<u8>, GatewayError> {
    // Outer `Ok` = breaker-counted success (either a real success or a
    // client-side business error). Outer `Err(GatewayError)` = infra
    // failure that should count against the breaker threshold.
    let op = async {
        match call_backend_service(
            service_channel,
            &routing_decision.grpc_method,
            grpc_request,
        )
        .await
        {
            Ok(bytes) => Ok::<Result<Vec<u8>, GatewayError>, GatewayError>(Ok(bytes)),
            Err(err) if is_client_error(&err) => {
                // Business error — don't trip the breaker. Still bubbles
                // out to the HTTP mapper as the correct 4xx.
                Ok(Err(err))
            }
            Err(err) => Err(err),
        }
    };

    match circuit_breaker.call(op).await {
        Ok(Ok(resp)) => Ok(resp),
        Ok(Err(client_err)) => Err(client_err),
        Err(common_rust::circuit_breaker::CircuitBreakerError::Open) => {
            handle_circuit_open(state, routing_decision, path, method)
        }
        Err(common_rust::circuit_breaker::CircuitBreakerError::OperationFailed(e)) => {
            handle_operation_failed(state, routing_decision, path, method, start_time, e)
        }
    }
}

/// Handle circuit breaker open state.
fn handle_circuit_open(
    state: &Arc<AppState>,
    routing_decision: &crate::router::RoutingDecision,
    path: &str,
    method: &str,
) -> Result<Vec<u8>, GatewayError> {
    warn!(
        service = %routing_decision.service,
        "Circuit breaker is OPEN - rejecting request"
    );

    state
        .metrics
        .request_counter
        .with_label_values(&[path, method, "503"])
        .inc();

    state
        .metrics
        .grpc_call_counter
        .with_label_values(&[
            routing_decision.service.as_ref(),
            routing_decision.grpc_method.as_ref(),
            "circuit_open",
        ])
        .inc();

    Err(GatewayError::ServiceUnavailable(format!(
        "Service {} is currently unavailable (circuit breaker open)",
        &*routing_decision.service
    )))
}

/// Handle operation failed error from circuit breaker.
fn handle_operation_failed(
    state: &Arc<AppState>,
    routing_decision: &crate::router::RoutingDecision,
    path: &str,
    method: &str,
    start_time: Instant,
    error_msg: String,
) -> Result<Vec<u8>, GatewayError> {
    let duration_ms = start_time.elapsed().as_millis();
    let trace_id = extract_trace_id_from_span();

    error!(
        service = %routing_decision.service,
        grpc_method = %routing_decision.grpc_method,
        path = %path,
        method = %method,
        trace_id = %trace_id,
        duration_ms = %duration_ms,
        error_type = "grpc_error",
        error = %error_msg,
        "Backend service call failed"
    );

    state
        .metrics
        .grpc_call_counter
        .with_label_values(&[
            routing_decision.service.as_ref(),
            routing_decision.grpc_method.as_ref(),
            "error",
        ])
        .inc();

    Err(GatewayError::GrpcCallFailed(error_msg))
}

/// Handle direct call without circuit breaker.
async fn handle_without_breaker(
    state: &Arc<AppState>,
    routing_decision: &crate::router::RoutingDecision,
    service_channel: tonic::transport::Channel,
    grpc_request: Vec<u8>,
    path: &str,
    method: &str,
) -> Result<Vec<u8>, GatewayError> {
    use crate::handlers::types::map_grpc_error_to_status;

    match call_backend_service(service_channel, &routing_decision.grpc_method, grpc_request).await {
        Ok(resp) => Ok(resp),
        Err(e) => {
            let trace_id = extract_trace_id_from_span();
            error!(
                service = %routing_decision.service,
                trace_id = %trace_id,
                grpc_method = %routing_decision.grpc_method,
                error = %e,
                "Backend service call failed"
            );

            let status_code = map_grpc_error_to_status(&e);
            let status_str = status_code.as_u16().to_string();

            state
                .metrics
                .request_counter
                .with_label_values(&[path, method, &status_str])
                .inc();

            state
                .metrics
                .grpc_call_counter
                .with_label_values(&[
                    routing_decision.service.as_ref(),
                    routing_decision.grpc_method.as_ref(),
                    "error",
                ])
                .inc();

            Err(e)
        }
    }
}

#[cfg(test)]
mod breaker_filter_tests {
    use super::*;
    use tonic::Code;

    #[test]
    fn not_found_is_client_side_not_counted() {
        let err = GatewayError::GrpcStatus {
            code: Code::NotFound,
            message: "user missing".into(),
        };
        assert!(is_client_error(&err));
    }

    #[test]
    fn invalid_argument_is_client_side() {
        let err = GatewayError::GrpcStatus {
            code: Code::InvalidArgument,
            message: "bad".into(),
        };
        assert!(is_client_error(&err));
    }

    #[test]
    fn unauthenticated_permission_denied_are_client_side() {
        for code in [Code::Unauthenticated, Code::PermissionDenied] {
            let err = GatewayError::GrpcStatus {
                code,
                message: "auth".into(),
            };
            assert!(is_client_error(&err), "{code:?} should not trip breaker");
        }
    }

    #[test]
    fn unavailable_counts_against_breaker() {
        let err = GatewayError::GrpcStatus {
            code: Code::Unavailable,
            message: "upstream down".into(),
        };
        assert!(!is_client_error(&err));
    }

    #[test]
    fn internal_and_unknown_count_against_breaker() {
        for code in [Code::Internal, Code::Unknown, Code::DataLoss] {
            let err = GatewayError::GrpcStatus {
                code,
                message: "infra".into(),
            };
            assert!(!is_client_error(&err), "{code:?} should trip breaker");
        }
    }

    #[test]
    fn deadline_exceeded_counts_against_breaker() {
        // Treated as infra — a slow/unresponsive upstream deserves circuit
        // breaking, unlike Cancelled which is client-initiated.
        let err = GatewayError::GrpcStatus {
            code: Code::DeadlineExceeded,
            message: "slow".into(),
        };
        assert!(!is_client_error(&err));
    }

    #[test]
    fn opaque_grpc_call_failed_counts_against_breaker() {
        // No tonic code → connection/codec/reflection error → infra.
        let err = GatewayError::GrpcCallFailed("dns error".into());
        assert!(!is_client_error(&err));
    }

    #[test]
    fn rate_limit_and_conversion_do_not_trip_breaker() {
        assert!(is_client_error(&GatewayError::RateLimitExceeded));
        assert!(is_client_error(&GatewayError::ConversionError(
            "bad json".into()
        )));
    }
}
