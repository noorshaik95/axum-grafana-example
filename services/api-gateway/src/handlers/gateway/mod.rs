//! Gateway request handler.
//!
//! Main entry point for all gateway requests, coordinating routing, rate limiting,
//! authentication, and backend service calls.

use axum::{
    body::Body,
    extract::{ConnectInfo, Request, State},
    http::HeaderMap,
    response::Response,
};
use std::net::SocketAddr;
use std::sync::Arc;

use crate::shared::state::AppState;

mod backend;
mod circuit_breaker;
pub(crate) mod conversion;
mod metrics;
pub(crate) mod proxy;
mod rate_limiting;
mod response;
pub mod routing;
mod timeout;


/// Main gateway handler with timeout wrapper.
///
/// This is the primary entry point for all gateway requests. It wraps the actual
/// handler with a configurable timeout to prevent hanging requests.
///
/// # Arguments
///
/// * `state` - Application state containing configuration and services
/// * `addr` - Client socket address
/// * `headers` - HTTP request headers
/// * `request` - The HTTP request
///
/// # Returns
///
/// An HTTP response
#[tracing::instrument(name = "gateway_handler", skip(state, headers, request))]
pub async fn gateway_handler(
    State(state): State<Arc<AppState>>,
    ConnectInfo(addr): ConnectInfo<SocketAddr>,
    headers: HeaderMap,
    request: Request<Body>,
) -> Response {
    timeout::handle_with_timeout(state, addr, headers, request).await
}
