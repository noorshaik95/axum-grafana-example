//! Routing decision types and errors.
//!
//! Defines the result types for routing operations.

use std::collections::HashMap;
use std::sync::Arc;
use thiserror::Error;

/// Errors that can occur during routing.
#[derive(Debug, Error)]
pub enum RouterError {
    #[error("Route not found for path: {path}, method: {method}")]
    RouteNotFound { path: String, method: String },
}

/// Result of a routing decision.
///
/// Performance: Uses Arc<str> for service and grpc_method to avoid cloning strings
/// in the hot path. Cloning Arc is cheap (atomic reference count increment) compared
/// to cloning the actual string data.
#[derive(Debug, Clone)]
pub struct RoutingDecision {
    pub service: Arc<str>,
    pub grpc_method: Arc<str>,
    pub path_params: HashMap<String, String>,
    /// When `Some`, the request is forwarded as a plain HTTP reverse-proxy
    /// call to this base URL (§1a). The gRPC transcoding pipeline is
    /// bypassed entirely for these routes.
    pub http_proxy_url: Option<Arc<str>>,
    /// Optional upstream path template (§1b). When `Some`, the proxy
    /// handler substitutes `:param` placeholders with `path_params` before
    /// calling the upstream. When `None`, the original request path is
    /// forwarded verbatim (§1a behavior).
    pub http_proxy_path_template: Option<Arc<str>>,
}

impl RoutingDecision {
    /// Create a new routing decision.
    pub fn new(service: impl AsRef<str>, grpc_method: impl AsRef<str>) -> Self {
        Self {
            service: Arc::from(service.as_ref()),
            grpc_method: Arc::from(grpc_method.as_ref()),
            path_params: HashMap::new(),
            http_proxy_url: None,
            http_proxy_path_template: None,
        }
    }

    /// Create a new routing decision with path parameters.
    pub fn with_params(
        service: impl AsRef<str>,
        grpc_method: impl AsRef<str>,
        path_params: HashMap<String, String>,
    ) -> Self {
        Self {
            service: Arc::from(service.as_ref()),
            grpc_method: Arc::from(grpc_method.as_ref()),
            path_params,
            http_proxy_url: None,
            http_proxy_path_template: None,
        }
    }

    /// Attach an HTTP reverse-proxy target URL to this decision. When set,
    /// the gateway bypasses the gRPC pipeline and forwards the raw request
    /// to the configured URL.
    pub fn with_http_proxy(mut self, proxy_url: impl AsRef<str>) -> Self {
        self.http_proxy_url = Some(Arc::from(proxy_url.as_ref()));
        self
    }

    /// Attach an upstream path template (§1b). Must only be combined with
    /// `with_http_proxy`. Placeholders like `:id` in the template are
    /// substituted from `path_params` at forward time.
    pub fn with_path_template(mut self, template: impl AsRef<str>) -> Self {
        self.http_proxy_path_template = Some(Arc::from(template.as_ref()));
        self
    }
}
