use prometheus::{
    HistogramOpts, HistogramVec, IntCounter, IntCounterVec, IntGauge, Opts, Registry,
};
use redis::aio::ConnectionManager;
use std::sync::Arc;
use tokio::sync::RwLock;

use crate::auth::flag_middleware::FlagProvider;
use crate::auth::AuthService;
use crate::config::GatewayConfig;
use crate::discovery::RouteDiscoveryService;
use crate::grpc::client::GrpcClientPool;
use crate::handlers::gateway::routing::TenantRoutingTable;
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
    /// Shared Redis connection (W17.1 token cache, W17.3 flag cache).
    /// `None` when Redis is unconfigured or unreachable at startup — the
    /// gateway still functions, just without caching.
    pub redis_conn: Option<ConnectionManager>,
    /// Optional feature-flag provider. `None` falls back to
    /// `UnavailableProvider` (empty flag maps) in the middleware wiring.
    pub flag_provider: Option<Arc<dyn FlagProvider>>,
    /// Per-tenant service routing table (W17.4). Layered lookup: a request
    /// with `X-Tenant-Slug: eastfield` targeting the `discussion` service
    /// resolves to `http://discussion-eastfield:50056` via this table.
    /// Read by handlers/gateway/routing once tenant-service (W16) ships
    /// the provisioning path that populates it.
    #[allow(dead_code)]
    pub tenant_routing: Arc<TenantRoutingTable>,
    /// Shared reqwest client used by the §1a HTTP reverse-proxy handler.
    /// Built once with connection pooling + 10s request timeout.
    pub http_client: reqwest::Client,
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

        // Build a reqwest client with a 10s request timeout. If the builder
        // ever fails (it currently can't — reqwest::Client::new() is infallible
        // in the versions we use), fall back to the default client so we
        // never panic on startup.
        let http_client = reqwest::Client::builder()
            .timeout(std::time::Duration::from_secs(10))
            .build()
            .unwrap_or_else(|_| reqwest::Client::new());

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
            redis_conn: None,
            flag_provider: None,
            tenant_routing: Arc::new(TenantRoutingTable::empty()),
            http_client,
        }
    }

    /// Attach a shared Redis connection for use by token cache + flag cache.
    pub fn with_redis(mut self, redis_conn: ConnectionManager) -> Self {
        self.redis_conn = Some(redis_conn);
        self
    }

    /// Install a feature-flag provider. When unset the router falls back
    /// to `UnavailableProvider` and returns empty flag maps.
    #[allow(dead_code)]
    pub fn with_flag_provider(mut self, provider: Arc<dyn FlagProvider>) -> Self {
        self.flag_provider = Some(provider);
        self
    }

    /// Replace the per-tenant service routing table.
    #[allow(dead_code)]
    pub fn with_tenant_routing(mut self, table: Arc<TenantRoutingTable>) -> Self {
        self.tenant_routing = table;
        self
    }
}
