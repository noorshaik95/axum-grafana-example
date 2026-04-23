//! gRPC type definitions.
//!
//! These types are part of the public API and may be used by external code.

#![allow(dead_code)]

use std::collections::HashMap;
use tonic::Status;

/// Error types for gRPC client operations
#[derive(Debug, thiserror::Error)]
pub enum GrpcError {
    #[error("Connection error: {0}")]
    ConnectionError(String),

    #[error("Service not found: {0}")]
    ServiceNotFound(String),

    #[error("Call failed: {0}")]
    CallFailed(String),

    /// gRPC call returned a structured Status. Preserves the canonical gRPC
    /// code so the HTTP mapper (and circuit breaker) can distinguish
    /// business errors (NotFound, InvalidArgument, …) from infra failures
    /// (Unavailable, Internal, DeadlineExceeded, …).
    #[error("gRPC status {0:?}: {1}")]
    Status(tonic::Code, String),

    #[error("Timeout error: {0}")]
    Timeout(String),

    #[error("Invalid configuration: {0}")]
    InvalidConfig(String),

    #[error("Conversion error: {0}")]
    ConversionError(String),
}

impl GrpcError {
    /// Return the gRPC code carried by a `Status` variant; `None` for other
    /// variants (connection/codec/config — all treated as infra failures).
    pub fn grpc_code(&self) -> Option<tonic::Code> {
        match self {
            GrpcError::Status(code, _) => Some(*code),
            _ => None,
        }
    }

    /// Return just the message text without the `"gRPC status X: "` prefix.
    /// Used when the caller will re-wrap this into a `GatewayError::GrpcStatus`
    /// that already formats the code, so the prefix would double up.
    pub fn grpc_message(&self) -> String {
        match self {
            GrpcError::Status(_, msg) => msg.clone(),
            _ => self.to_string(),
        }
    }
}

/// Classify a gRPC code as a "client/business" error (maps to HTTP 4xx) vs
/// an "infra" error (maps to HTTP 5xx).
///
/// Only infra errors should trip the circuit breaker; counting NotFound or
/// InvalidArgument bursts toward the breaker threshold causes legitimate
/// traffic (e.g. login after a profile 404) to cascade into 503s.
pub fn is_client_side_grpc_code(code: tonic::Code) -> bool {
    use tonic::Code;
    matches!(
        code,
        Code::NotFound
            | Code::InvalidArgument
            | Code::AlreadyExists
            | Code::PermissionDenied
            | Code::Unauthenticated
            | Code::FailedPrecondition
            | Code::OutOfRange
            | Code::Unimplemented
            | Code::ResourceExhausted
            | Code::Cancelled
    )
}

impl From<tonic::transport::Error> for GrpcError {
    fn from(err: tonic::transport::Error) -> Self {
        GrpcError::ConnectionError(err.to_string())
    }
}

impl From<Status> for GrpcError {
    fn from(status: Status) -> Self {
        GrpcError::Status(status.code(), status.message().to_string())
    }
}

/// Request structure for generic gRPC calls
#[derive(Debug, Clone)]
pub struct GrpcRequest {
    pub service: String,
    pub method: String,
    pub payload: Vec<u8>,
    pub metadata: HashMap<String, String>,
}

/// Response structure for generic gRPC calls
#[derive(Debug, Clone)]
pub struct GrpcResponse {
    pub status: tonic::Code,
    pub payload: Vec<u8>,
    pub metadata: HashMap<String, String>,
}
