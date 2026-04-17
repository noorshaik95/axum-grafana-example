use prometheus::{
    HistogramOpts, HistogramVec, IntCounter, IntCounterVec, IntGauge, Opts, Registry,
};
use std::sync::Arc;
use tokio::sync::RwLock;

use crate::auth::AuthService;
use crate::config::GatewayConfig;
use crate::discovery::RouteDiscoveryService;
use crate::grpc::client::GrpcClientPool;
use crate::middleware::ClientIpExtractor;
use crate::router::RequestRouter;
use common_rust::rate_limit::IpRateLimiter;

#[derive(Clone)]
pub struct AppState {
    pub config: GatewayConfig,
    pub grpc_pool: Arc<GrpcClientPool>,
    pub auth_service: Arc<AuthService>,
    pub router_lock: Arc<RwLock<RequestRouter>>,
    pub rate_limiter: Option<Arc<IpRateLimiter>>,
    pub client_ip_extractor: Arc<ClientIpExtractor>,
    pub registry: Registry,
    pub metrics: GatewayMetrics,
    // For dynamic route updates (used by admin endpoint and periodic refresh)
    pub discovery_service: Option<Arc<RouteDiscoveryService>>,
}

#[derive(Clone)]
pub struct GatewayMetrics {
    pub request_counter: IntCounterVec,
    pub request_duration: HistogramVec,
    pub grpc_call_counter: IntCounterVec,
    pub auth_failure_counter: IntCounter,
    pub rate_limit_counter: IntCounter,
    #[allow(dead_code)]
    pub circuit_breaker_state: IntCounterVec,
    #[allow(dead_code)]
    pub active_connections: IntCounterVec,
    // Rate limiter metrics
    pub rate_limiter_tracked_clients: IntGauge,
    pub rate_limiter_evictions_total: IntCounter,
}

impl GatewayMetrics {
    pub fn new(registry: &Registry) -> Self {
        let metrics = Self::create_metrics();
        Self::register_metrics(registry, &metrics);
        metrics
    }

    /// Create all metric instances.
    fn create_metrics() -> Self {
        let request_counter = IntCounterVec::new(
            Opts::new(
                "gateway_requests_total",
                "Total number of requests processed by the gateway",
            )
            .namespace("api_gateway"),
            &["route", "method", "status"],
        )
        .expect("Failed to create request_counter metric");

        let request_duration = HistogramVec::new(
            HistogramOpts::new(
                "gateway_request_duration_seconds",
                "Request duration in seconds",
            )
            .namespace("api_gateway")
            .buckets(vec![
                0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1.0, 2.5, 5.0, 10.0,
            ]),
            &["route", "method"],
        )
        .expect("Failed to create request_duration metric");

        let grpc_call_counter = IntCounterVec::new(
            Opts::new(
                "gateway_grpc_calls_total",
                "Total number of gRPC calls to backend services",
            )
            .namespace("api_gateway"),
            &["service", "method", "status"],
        )
        .expect("Failed to create grpc_call_counter metric");

        let auth_failure_counter = IntCounter::new(
            "gateway_auth_failures_total",
            "Total number of authentication failures",
        )
        .expect("Failed to create auth_failure_counter metric");

        let rate_limit_counter = IntCounter::new(
            "gateway_rate_limit_exceeded_total",
            "Total number of requests rejected due to rate limiting",
        )
        .expect("Failed to create rate_limit_counter metric");

        let circuit_breaker_state = IntCounterVec::new(
            Opts::new(
                "gateway_circuit_breaker_state_changes_total",
                "Total number of circuit breaker state changes",
            )
            .namespace("api_gateway"),
            &["service", "from_state", "to_state"],
        )
        .expect("Failed to create circuit_breaker_state metric");

        let active_connections = IntCounterVec::new(
            Opts::new(
                "gateway_active_connections_total",
                "Number of active connections to backend services",
            )
            .namespace("api_gateway"),
            &["service"],
        )
        .expect("Failed to create active_connections metric");

        let rate_limiter_tracked_clients = IntGauge::new(
            "gateway_rate_limiter_tracked_clients",
            "Current number of clients being tracked by the rate limiter",
        )
        .expect("Failed to create rate_limiter_tracked_clients metric");

        let rate_limiter_evictions_total = IntCounter::new(
            "gateway_rate_limiter_evictions_total",
            "Total number of client entries evicted from the rate limiter cache",
        )
        .expect("Failed to create rate_limiter_evictions_total metric");

        GatewayMetrics {
            request_counter,
            request_duration,
            grpc_call_counter,
            auth_failure_counter,
            rate_limit_counter,
            circuit_breaker_state,
            active_connections,
            rate_limiter_tracked_clients,
            rate_limiter_evictions_total,
        }
    }

    /// Register all metrics with the registry.
    fn register_metrics(registry: &Registry, metrics: &GatewayMetrics) {
        registry
            .register(Box::new(metrics.request_counter.clone()))
            .expect("Failed to register request_counter metric");
        registry
            .register(Box::new(metrics.request_duration.clone()))
            .expect("Failed to register request_duration metric");
        registry
            .register(Box::new(metrics.grpc_call_counter.clone()))
            .expect("Failed to register grpc_call_counter metric");
        registry
            .register(Box::new(metrics.auth_failure_counter.clone()))
            .expect("Failed to register auth_failure_counter metric");
        registry
            .register(Box::new(metrics.rate_limit_counter.clone()))
            .expect("Failed to register rate_limit_counter metric");
        registry
            .register(Box::new(metrics.circuit_breaker_state.clone()))
            .expect("Failed to register circuit_breaker_state metric");
        registry
            .register(Box::new(metrics.active_connections.clone()))
            .expect("Failed to register active_connections metric");
        registry
            .register(Box::new(metrics.rate_limiter_tracked_clients.clone()))
            .expect("Failed to register rate_limiter_tracked_clients metric");
        registry
            .register(Box::new(metrics.rate_limiter_evictions_total.clone()))
            .expect("Failed to register rate_limiter_evictions_total metric");
    }
}

impl AppState {
    /// Create AppState with optional discovery service support
    pub fn new(
        config: GatewayConfig,
        grpc_pool: GrpcClientPool,
        auth_service: AuthService,
        router_lock: Arc<RwLock<RequestRouter>>,
        rate_limiter: Option<IpRateLimiter>,
        client_ip_extractor: ClientIpExtractor,
        discovery_service: Option<RouteDiscoveryService>,
    ) -> Self {
        let registry = Registry::new();
        let metrics = GatewayMetrics::new(&registry);

        AppState {
            config,
            grpc_pool: Arc::new(grpc_pool),
            auth_service: Arc::new(auth_service),
            router_lock,
            rate_limiter: rate_limiter.map(Arc::new),
            client_ip_extractor: Arc::new(client_ip_extractor),
            registry,
            metrics,
            discovery_service: discovery_service.map(Arc::new),
        }
    }
}
