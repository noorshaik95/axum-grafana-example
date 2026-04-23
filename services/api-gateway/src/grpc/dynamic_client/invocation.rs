//! gRPC method invocation with raw bytes.
//!
//! Handles low-level gRPC calls using Tonic's unary RPC mechanism with
//! custom byte codecs.

use bytes::{Buf, BufMut};
use tonic::transport::Channel;
use tonic::{codec::Codec, Request, Status};
use tracing::error;

use crate::grpc::types::GrpcError;

/// Invoke a unary gRPC call with raw bytes.
///
/// # Arguments
///
/// * `channel` - The gRPC channel to use
/// * `method` - The full method path (e.g., "/user.UserService/GetUser")
/// * `request` - The request containing protobuf bytes
/// * `service_name` - The service name for error logging
///
/// # Returns
///
/// The response as raw protobuf bytes
pub async fn invoke_unary_bytes(
    channel: &mut Channel,
    method: String,
    request: Request<Vec<u8>>,
    service_name: &str,
) -> Result<Vec<u8>, GrpcError> {
    use tonic::client::Grpc;

    let mut grpc = Grpc::new(channel.clone());
    let codec = BytesCodec;

    // CRITICAL: Must call ready() before unary() to ensure tower buffer is ready
    // This prevents "buffer full; poll_ready must be called first" panic
    grpc.ready().await.map_err(|e| {
        error!(
            service = %service_name,
            error = %e,
            "Failed to ready gRPC client"
        );
        GrpcError::CallFailed(format!("Failed to ready gRPC client: {}", e))
    })?;

    let response = grpc
        .unary(request, method.parse().unwrap(), codec)
        .await
        .map_err(|status| {
            error!(
                service = %service_name,
                status_code = ?status.code(),
                message = %status.message(),
                "gRPC call failed"
            );
            // Preserve the tonic Code so the HTTP mapper can distinguish
            // business errors (NotFound → 404) from infra errors
            // (Unavailable → 503). Losing the code here is what caused
            // R2: every NotFound surfaced as 502 BAD_GATEWAY and tripped
            // the circuit breaker on normal profile-404 traffic.
            GrpcError::Status(status.code(), status.message().to_string())
        })?;

    Ok(response.into_inner())
}

/// Map gRPC status codes to HTTP status codes.
///
/// Provides a standard mapping from gRPC status codes to their HTTP equivalents.
#[allow(dead_code)]
pub fn grpc_status_to_http(status: &Status) -> u16 {
    use tonic::Code;

    match status.code() {
        Code::Ok => 200,
        Code::Cancelled => 499, // Client closed request
        Code::Unknown => 500,
        Code::InvalidArgument => 400,
        Code::DeadlineExceeded => 504,
        Code::NotFound => 404,
        Code::AlreadyExists => 409,
        Code::PermissionDenied => 403,
        Code::ResourceExhausted => 429,
        Code::FailedPrecondition => 400,
        Code::Aborted => 409,
        Code::OutOfRange => 400,
        Code::Unimplemented => 501,
        Code::Internal => 500,
        Code::Unavailable => 503,
        Code::DataLoss => 500,
        Code::Unauthenticated => 401,
    }
}

/// Simple codec for raw bytes.
#[derive(Debug, Clone, Default)]
struct BytesCodec;

impl Codec for BytesCodec {
    type Encode = Vec<u8>;
    type Decode = Vec<u8>;

    type Encoder = BytesEncoder;
    type Decoder = BytesDecoder;

    fn encoder(&mut self) -> Self::Encoder {
        BytesEncoder
    }

    fn decoder(&mut self) -> Self::Decoder {
        BytesDecoder
    }
}

/// Encoder for raw bytes.
#[derive(Debug, Clone, Default)]
struct BytesEncoder;

impl tonic::codec::Encoder for BytesEncoder {
    type Item = Vec<u8>;
    type Error = Status;

    fn encode(
        &mut self,
        item: Self::Item,
        buf: &mut tonic::codec::EncodeBuf<'_>,
    ) -> Result<(), Self::Error> {
        buf.put_slice(&item);
        Ok(())
    }
}

/// Decoder for raw bytes.
#[derive(Debug, Clone, Default)]
struct BytesDecoder;

impl tonic::codec::Decoder for BytesDecoder {
    type Item = Vec<u8>;
    type Error = Status;

    fn decode(
        &mut self,
        buf: &mut tonic::codec::DecodeBuf<'_>,
    ) -> Result<Option<Self::Item>, Self::Error> {
        let chunk = buf.chunk();
        // An empty chunk means the server sent a zero-byte proto3 body (all
        // fields at default values). Return Some(vec![]) so tonic doesn't
        // produce "Missing response message" — prost DynamicMessage::decode
        // handles empty bytes as a valid all-defaults message.
        let bytes = chunk.to_vec();
        buf.advance(chunk.len());
        Ok(Some(bytes))
    }
}
