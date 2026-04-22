//! Request routing and processing pipeline.
//!
//! Handles the main request processing pipeline including routing decisions,
//! rate limiting, and coordination of backend service calls.
//!
//! Also hosts the per-tenant routing table (W17.4): given an
//! `X-Tenant-Slug` header and a service key (one of `user-auth`, `course`,
//! `assignment`, `content`, `video`, `discussion`, `scheduling`, `ai`),
//! return the target endpoint URL for that tenant's service container.

use axum::{
    body::Body,
    extract::{ConnectInfo, Request, State},
    http::HeaderMap,
    response::Response,
};
use std::collections::HashMap;
use std::net::SocketAddr;
use std::sync::Arc;
use std::sync::RwLock;
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

    // §1a HTTP reverse-proxy short-circuit. When a route is marked as
    // HTTP passthrough (user-auth SSO / impersonation / MFA-reset), bypass
    // the gRPC transcoding pipeline entirely.
    if let Some(proxy_url) = routing_decision.http_proxy_url.clone() {
        let result = super::proxy::forward_http_request(
            &state.http_client,
            &proxy_url,
            request,
            &routing_decision.path_params,
            &path,
        )
        .await;
        match result {
            Ok(resp) => {
                record_success_metrics(&state, &routing_decision, &path, &method, start_time);
                log_request_completion(
                    &path,
                    &method,
                    &routing_decision,
                    start_time,
                    &trace_id,
                );
                return Ok(resp);
            }
            Err(e) => {
                error!(
                    path = %path,
                    method = %method,
                    service = %routing_decision.service,
                    error = %e,
                    trace_id = %trace_id,
                    "HTTP-proxy forward failed"
                );
                state
                    .metrics
                    .request_counter
                    .with_label_values(&[&path, &method, &"502".to_string()])
                    .inc();
                return Err(e);
            }
        }
    }

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

// ---------------------------------------------------------------------------
// W17.4 — Per-tenant service routing table
// ---------------------------------------------------------------------------

/// Canonical service keys understood by the tenant routing table. New
/// Wave-3 services (`discussion`, `scheduling`, `ai`) have been added
/// alongside the original five.
#[allow(dead_code)]
pub const TENANT_SERVICE_KEYS: &[&str] = &[
    "user-auth",
    "course",
    "assignment",
    "content",
    "video",
    "discussion",
    "scheduling",
    "ai",
];

/// Per-tenant service endpoint map, loaded from tenant YAML files (one per
/// tenant, keyed by slug) and hot-reloadable via the admin refresh route.
///
/// Expected tenant yaml (e.g. `config/tenants/eastfield.yaml`):
/// ```yaml
/// services:
///   user-auth:  "http://user-auth-eastfield:50051"
///   course:     "http://course-eastfield:50052"
///   assignment: "http://assignment-eastfield:50053"
///   content:    "http://content-eastfield:50054"
///   video:      "http://video-eastfield:50055"
///   discussion: "http://discussion-eastfield:50056"
///   scheduling: "http://scheduling-eastfield:50063"
///   ai:         "http://ai-eastfield:50064"
/// ```
///
/// Backward compatibility: tenants that omit new keys (`discussion`,
/// `scheduling`, `ai`) are still accepted — requests to those services
/// return 503 rather than panicking, letting us roll out the new services
/// tenant-by-tenant.
#[derive(Debug, Default)]
#[allow(dead_code)]
pub struct TenantRoutingTable {
    // Inner RwLock so the table can be hot-reloaded without swapping the
    // Arc<TenantRoutingTable> in AppState.
    inner: RwLock<HashMap<String, TenantServiceMap>>,
}

/// Per-tenant `{service-key -> endpoint URL}` map. Endpoint URLs are
/// stored as `String` (not `Arc<str>`) so the table can be rebuilt
/// cheaply on tenant provisioning.
#[derive(Debug, Clone, Default)]
#[allow(dead_code)]
pub struct TenantServiceMap {
    pub endpoints: HashMap<String, String>,
}

#[allow(dead_code)]
impl TenantServiceMap {
    pub fn new(endpoints: HashMap<String, String>) -> Self {
        Self { endpoints }
    }

    /// Lookup a service's endpoint for this tenant. Returns `None` if the
    /// tenant config omits that key — callers map this to a 503 response.
    pub fn endpoint(&self, service_key: &str) -> Option<&str> {
        self.endpoints.get(service_key).map(|s| s.as_str())
    }

    /// Return the set of service keys this tenant has configured.
    pub fn configured_keys(&self) -> Vec<&str> {
        self.endpoints.keys().map(|k| k.as_str()).collect()
    }
}

#[allow(dead_code)]
impl TenantRoutingTable {
    /// Build an empty routing table — the gateway boots cleanly even when
    /// no tenants are provisioned yet.
    pub fn empty() -> Self {
        Self::default()
    }

    /// Build a table seeded from a map of `{slug -> TenantServiceMap}`.
    pub fn new(tenants: HashMap<String, TenantServiceMap>) -> Self {
        Self {
            inner: RwLock::new(tenants),
        }
    }

    /// Install or replace a single tenant's service map. Called by the
    /// tenant-service (W16) over the admin refresh endpoint when a new
    /// tenant is provisioned.
    pub fn upsert(&self, slug: impl Into<String>, map: TenantServiceMap) {
        if let Ok(mut guard) = self.inner.write() {
            guard.insert(slug.into(), map);
        }
    }

    /// Drop a tenant from the table (called on deprovision).
    pub fn remove(&self, slug: &str) -> Option<TenantServiceMap> {
        self.inner.write().ok().and_then(|mut g| g.remove(slug))
    }

    /// Clone out a tenant's service map, if present.
    pub fn tenant(&self, slug: &str) -> Option<TenantServiceMap> {
        self.inner.read().ok().and_then(|g| g.get(slug).cloned())
    }

    /// Resolve a (tenant_slug, service_key) pair to an endpoint URL.
    ///
    /// Returns `None` if the tenant is unknown OR the tenant's yaml does
    /// not declare that service — upstream code converts both into the
    /// same 503 response so callers don't leak which case occurred.
    pub fn resolve(&self, tenant_slug: &str, service_key: &str) -> Option<String> {
        self.inner
            .read()
            .ok()?
            .get(tenant_slug)?
            .endpoint(service_key)
            .map(|s| s.to_string())
    }

    /// Number of tenants currently in the table (for metrics / admin UI).
    pub fn tenant_count(&self) -> usize {
        self.inner.read().map(|g| g.len()).unwrap_or(0)
    }

    /// Snapshot of all tenant slugs (sorted for stable output).
    pub fn tenant_slugs(&self) -> Vec<String> {
        let mut slugs: Vec<String> = self
            .inner
            .read()
            .map(|g| g.keys().cloned().collect())
            .unwrap_or_default();
        slugs.sort();
        slugs
    }
}

/// Map a canonical service key to the logical gateway service name used
/// by `GrpcClientPool::get_channel`. When the per-tenant routing table
/// has an entry, that override takes precedence; otherwise the gateway
/// falls back to the shared (non-tenant) service name.
#[allow(dead_code)]
pub fn gateway_service_name_for_key(service_key: &str) -> &'static str {
    match service_key {
        "user-auth" => "user-auth-service",
        "course" => "course-service",
        "assignment" => "assignment-grading-service",
        "content" => "content-management-service",
        "video" => "video-conferencing-service",
        "discussion" => "discussion-service",
        "scheduling" => "scheduling-service",
        "ai" => "ai-service",
        other => {
            // Unknown keys are passed through so the router's 404 path can
            // handle them uniformly; no panic.
            debug!(service_key = %other, "Unknown tenant service key");
            ""
        }
    }
}

#[cfg(test)]
mod tenant_routing_tests {
    use super::*;

    fn sample_tenant(slug: &str, include_new: bool) -> TenantServiceMap {
        let mut m = HashMap::new();
        m.insert("user-auth".into(), format!("http://user-auth-{slug}:50051"));
        m.insert("course".into(), format!("http://course-{slug}:50052"));
        m.insert(
            "assignment".into(),
            format!("http://assignment-{slug}:50053"),
        );
        m.insert("content".into(), format!("http://content-{slug}:50054"));
        m.insert("video".into(), format!("http://video-{slug}:50055"));
        if include_new {
            m.insert(
                "discussion".into(),
                format!("http://discussion-{slug}:50056"),
            );
            m.insert(
                "scheduling".into(),
                format!("http://scheduling-{slug}:50063"),
            );
            m.insert("ai".into(), format!("http://ai-{slug}:50064"));
        }
        TenantServiceMap::new(m)
    }

    #[test]
    fn empty_table_is_safe() {
        let t = TenantRoutingTable::empty();
        assert!(t.resolve("eastfield", "course").is_none());
        assert_eq!(t.tenant_count(), 0);
    }

    #[test]
    fn upsert_then_resolve() {
        let t = TenantRoutingTable::empty();
        t.upsert("eastfield", sample_tenant("eastfield", true));
        assert_eq!(
            t.resolve("eastfield", "discussion"),
            Some("http://discussion-eastfield:50056".to_string())
        );
        assert_eq!(
            t.resolve("eastfield", "scheduling"),
            Some("http://scheduling-eastfield:50063".to_string())
        );
        assert_eq!(
            t.resolve("eastfield", "ai"),
            Some("http://ai-eastfield:50064".to_string())
        );
    }

    #[test]
    fn legacy_tenant_without_new_keys_returns_none_no_panic() {
        let t = TenantRoutingTable::empty();
        t.upsert("northridge", sample_tenant("northridge", false));
        // Old services still resolve.
        assert!(t.resolve("northridge", "course").is_some());
        // New services return None rather than panicking — upstream maps
        // that to a 503 so the tenant can be rolled forward independently.
        assert!(t.resolve("northridge", "discussion").is_none());
        assert!(t.resolve("northridge", "scheduling").is_none());
        assert!(t.resolve("northridge", "ai").is_none());
    }

    #[test]
    fn unknown_tenant_returns_none() {
        let t = TenantRoutingTable::empty();
        t.upsert("eastfield", sample_tenant("eastfield", true));
        assert!(t.resolve("unknown-school", "course").is_none());
    }

    #[test]
    fn remove_drops_tenant() {
        let t = TenantRoutingTable::empty();
        t.upsert("eastfield", sample_tenant("eastfield", true));
        assert!(t.remove("eastfield").is_some());
        assert!(t.resolve("eastfield", "course").is_none());
    }

    #[test]
    fn tenant_slugs_is_sorted() {
        let t = TenantRoutingTable::empty();
        t.upsert("zephyr", sample_tenant("zephyr", true));
        t.upsert("alpha", sample_tenant("alpha", true));
        t.upsert("meridian", sample_tenant("meridian", true));
        assert_eq!(
            t.tenant_slugs(),
            vec!["alpha".to_string(), "meridian".into(), "zephyr".into()]
        );
    }

    #[test]
    fn service_name_mapping_covers_all_eight_keys() {
        for key in TENANT_SERVICE_KEYS {
            let name = gateway_service_name_for_key(key);
            assert!(!name.is_empty(), "missing mapping for {key}");
        }
    }

    #[test]
    fn unknown_service_key_returns_empty_sentinel() {
        assert_eq!(gateway_service_name_for_key("nonsense"), "");
    }

    #[test]
    fn configured_keys_reflect_yaml_shape() {
        let m = sample_tenant("eastfield", true);
        let mut keys = m.configured_keys();
        keys.sort();
        assert_eq!(
            keys,
            vec![
                "ai",
                "assignment",
                "content",
                "course",
                "discussion",
                "scheduling",
                "user-auth",
                "video"
            ]
        );
    }
}
