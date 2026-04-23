//! Gateway handler type definitions.
//!
//! These types are part of the public API for gateway handlers.

#![allow(dead_code)]

use axum::{
    http::StatusCode,
    response::{IntoResponse, Response},
};
use serde::Serialize;

use crate::router::RouterError;
use common_rust::rate_limit::RateLimitError;

/// Gateway error types
#[derive(Debug, thiserror::Error)]
pub enum GatewayError {
    #[error("Route not found: {0}")]
    RouteNotFound(#[from] RouterError),

    #[error("Service unavailable: {0}")]
    ServiceUnavailable(String),

    #[error("Rate limit exceeded")]
    RateLimitExceeded,

    #[error("Conversion error: {0}")]
    ConversionError(String),

    /// Opaque backend failure (connection, codec, reflection, etc.) — always
    /// maps to HTTP 502. Prefer `GrpcStatus` when the tonic `Code` is known.
    #[error("gRPC call failed: {0}")]
    GrpcCallFailed(String),

    /// Structured gRPC failure preserving the canonical `tonic::Code` so the
    /// HTTP mapper can emit the correct 4xx/5xx (NotFound→404, etc.) instead
    /// of collapsing everything to 502.
    #[error("gRPC status {code:?}: {message}")]
    GrpcStatus {
        code: tonic::Code,
        message: String,
    },

    #[error("Request timeout")]
    Timeout,

    #[error("Not found")]
    NotFound,

    #[error("Internal error: {0}")]
    InternalError(String),
}

impl From<RateLimitError> for GatewayError {
    fn from(_: RateLimitError) -> Self {
        GatewayError::RateLimitExceeded
    }
}

/// Error response structure
#[derive(Debug, Serialize)]
pub struct ErrorResponse {
    pub error: ErrorDetail,
}

#[derive(Debug, Serialize)]
pub struct ErrorDetail {
    pub code: String,
    pub message: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub trace_id: Option<String>,
}

impl GatewayError {
    /// Convert to response with trace ID
    pub fn into_response_with_trace_id(self, trace_id: String) -> Response {
        use super::constants::*;

        let status = map_grpc_error_to_status(&self);

        let error_code = match &self {
            GatewayError::RouteNotFound(_) => ERR_CODE_ROUTE_NOT_FOUND,
            GatewayError::ServiceUnavailable(_) => ERR_CODE_SERVICE_UNAVAILABLE,
            GatewayError::RateLimitExceeded => ERR_CODE_RATE_LIMIT_EXCEEDED,
            GatewayError::ConversionError(_) => ERR_CODE_CONVERSION_ERROR,
            GatewayError::GrpcCallFailed(_) => ERR_CODE_BACKEND_ERROR,
            GatewayError::GrpcStatus { code, .. } => grpc_status_error_code(*code),
            GatewayError::Timeout => ERR_CODE_TIMEOUT,
            GatewayError::NotFound => ERR_CODE_NOT_FOUND,
            GatewayError::InternalError(_) => ERR_CODE_INTERNAL_ERROR,
        };

        let error_response =
            super::error::ErrorResponse::new(error_code, self.to_string(), trace_id);

        error_response.into_response_with_status(status)
    }
}

impl IntoResponse for GatewayError {
    fn into_response(self) -> Response {
        // Fallback implementation without trace ID (for backward compatibility)
        // In practice, this should rarely be used as the gateway handler extracts trace ID
        let trace_id = uuid::Uuid::new_v4().to_string();
        self.into_response_with_trace_id(trace_id)
    }
}

/// Map gRPC errors to HTTP status codes
pub fn map_grpc_error_to_status(error: &GatewayError) -> StatusCode {
    match error {
        GatewayError::RouteNotFound(_) => StatusCode::NOT_FOUND,
        GatewayError::ServiceUnavailable(_) => StatusCode::SERVICE_UNAVAILABLE,
        GatewayError::RateLimitExceeded => StatusCode::TOO_MANY_REQUESTS,
        GatewayError::ConversionError(_) => StatusCode::BAD_REQUEST,
        GatewayError::GrpcCallFailed(_) => StatusCode::BAD_GATEWAY,
        GatewayError::GrpcStatus { code, .. } => grpc_code_to_http_status(*code),
        GatewayError::Timeout => StatusCode::GATEWAY_TIMEOUT,
        GatewayError::NotFound => StatusCode::NOT_FOUND,
        GatewayError::InternalError(_) => StatusCode::INTERNAL_SERVER_ERROR,
    }
}

/// Canonical gRPC Code → HTTP status mapping.
///
/// - Business errors land as 4xx (NotFound → 404, InvalidArgument → 400, …)
///   so clients can react without the circuit breaker firing.
/// - Infra errors land as 5xx (Unavailable → 503, Internal → 502, …)
///   and are the only ones that should count toward the breaker.
pub fn grpc_code_to_http_status(code: tonic::Code) -> StatusCode {
    use tonic::Code;
    match code {
        Code::Ok => StatusCode::OK,
        Code::Cancelled => StatusCode::REQUEST_TIMEOUT,
        Code::InvalidArgument => StatusCode::BAD_REQUEST,
        Code::DeadlineExceeded => StatusCode::GATEWAY_TIMEOUT,
        Code::NotFound => StatusCode::NOT_FOUND,
        Code::AlreadyExists => StatusCode::CONFLICT,
        Code::PermissionDenied => StatusCode::FORBIDDEN,
        Code::ResourceExhausted => StatusCode::TOO_MANY_REQUESTS,
        Code::FailedPrecondition => StatusCode::PRECONDITION_FAILED,
        Code::Aborted => StatusCode::CONFLICT,
        Code::OutOfRange => StatusCode::BAD_REQUEST,
        Code::Unimplemented => StatusCode::NOT_IMPLEMENTED,
        Code::Unauthenticated => StatusCode::UNAUTHORIZED,
        // Infra failures — trip the breaker.
        Code::Unavailable => StatusCode::SERVICE_UNAVAILABLE,
        Code::Internal | Code::Unknown | Code::DataLoss => StatusCode::BAD_GATEWAY,
    }
}

/// Error-code constant for a gRPC-status-carrying error. Mirrors the
/// HTTP-status mapping so clients receive a stable string label.
fn grpc_status_error_code(code: tonic::Code) -> &'static str {
    use super::constants::*;
    use tonic::Code;
    match code {
        Code::NotFound => ERR_CODE_NOT_FOUND,
        Code::InvalidArgument | Code::OutOfRange | Code::FailedPrecondition => {
            ERR_CODE_CONVERSION_ERROR
        }
        Code::Unauthenticated => "UNAUTHORIZED",
        Code::PermissionDenied => "FORBIDDEN",
        Code::AlreadyExists | Code::Aborted => "CONFLICT",
        Code::ResourceExhausted => ERR_CODE_RATE_LIMIT_EXCEEDED,
        Code::Unimplemented => "NOT_IMPLEMENTED",
        Code::Unavailable => ERR_CODE_SERVICE_UNAVAILABLE,
        Code::DeadlineExceeded | Code::Cancelled => ERR_CODE_TIMEOUT,
        _ => ERR_CODE_BACKEND_ERROR,
    }
}

#[cfg(test)]
mod status_mapping_tests {
    use super::*;
    use tonic::Code;

    #[test]
    fn not_found_maps_to_404_not_502() {
        // R2 primary bug: NotFound was surfacing as 502 because the
        // gRPC status was lost in the CallFailed wrapper.
        let err = GatewayError::GrpcStatus {
            code: Code::NotFound,
            message: "user not found".into(),
        };
        assert_eq!(map_grpc_error_to_status(&err), StatusCode::NOT_FOUND);
    }

    #[test]
    fn unauthenticated_maps_to_401() {
        let err = GatewayError::GrpcStatus {
            code: Code::Unauthenticated,
            message: "bad token".into(),
        };
        assert_eq!(map_grpc_error_to_status(&err), StatusCode::UNAUTHORIZED);
    }

    #[test]
    fn permission_denied_maps_to_403() {
        let err = GatewayError::GrpcStatus {
            code: Code::PermissionDenied,
            message: "forbidden".into(),
        };
        assert_eq!(map_grpc_error_to_status(&err), StatusCode::FORBIDDEN);
    }

    #[test]
    fn invalid_argument_maps_to_400() {
        let err = GatewayError::GrpcStatus {
            code: Code::InvalidArgument,
            message: "bad payload".into(),
        };
        assert_eq!(map_grpc_error_to_status(&err), StatusCode::BAD_REQUEST);
    }

    #[test]
    fn failed_precondition_maps_to_412() {
        let err = GatewayError::GrpcStatus {
            code: Code::FailedPrecondition,
            message: "stale".into(),
        };
        assert_eq!(
            map_grpc_error_to_status(&err),
            StatusCode::PRECONDITION_FAILED
        );
    }

    #[test]
    fn unavailable_still_maps_to_503() {
        // Infra errors continue to 5xx so the breaker trips on genuine
        // upstream degradation.
        let err = GatewayError::GrpcStatus {
            code: Code::Unavailable,
            message: "down".into(),
        };
        assert_eq!(
            map_grpc_error_to_status(&err),
            StatusCode::SERVICE_UNAVAILABLE
        );
    }

    #[test]
    fn internal_maps_to_502() {
        let err = GatewayError::GrpcStatus {
            code: Code::Internal,
            message: "crash".into(),
        };
        assert_eq!(map_grpc_error_to_status(&err), StatusCode::BAD_GATEWAY);
    }

    #[test]
    fn deadline_exceeded_maps_to_504() {
        let err = GatewayError::GrpcStatus {
            code: Code::DeadlineExceeded,
            message: "slow".into(),
        };
        assert_eq!(map_grpc_error_to_status(&err), StatusCode::GATEWAY_TIMEOUT);
    }

    #[test]
    fn opaque_grpc_call_failed_still_maps_to_502() {
        let err = GatewayError::GrpcCallFailed("dns error".into());
        assert_eq!(map_grpc_error_to_status(&err), StatusCode::BAD_GATEWAY);
    }
}
