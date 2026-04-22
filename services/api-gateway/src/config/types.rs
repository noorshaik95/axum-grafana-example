use serde::{Deserialize, Serialize};
use std::collections::HashMap;

#[derive(Debug, Clone, Deserialize, Serialize)]
pub struct GatewayConfig {
    pub server: ServerConfig,
    pub services: HashMap<String, ServiceConfig>,
    pub auth: AuthConfig,
    pub rate_limit: Option<RateLimitConfig>,
    pub observability: ObservabilityConfig,
    pub discovery: DiscoveryConfig,
    #[serde(default)]
    pub route_overrides: Vec<RouteOverride>,
    #[serde(default)]
    pub cors: Option<CorsConfig>,
    #[serde(default)]
    pub body_limit: Option<BodyLimitConfig>,
    #[serde(default)]
    pub trusted_proxies: Vec<String>,
}

#[derive(Debug, Clone, Deserialize, Serialize)]
pub struct ServerConfig {
    pub host: String,
    pub port: u16,
    #[serde(default = "default_request_timeout_ms")]
    pub request_timeout_ms: u64,
}

fn default_request_timeout_ms() -> u64 {
    30000 // 30 seconds
}

#[derive(Debug, Clone, Default, Deserialize, Serialize)]
pub struct RouteConfig {
    pub path: String,
    pub method: String,
    pub service: String,
    pub grpc_method: String,
    /// If set, this route bypasses the gRPC transcoding pipeline and is
    /// forwarded as a plain HTTP request to the given base URL (§1a).
    #[serde(default)]
    pub http_proxy_url: Option<String>,
}

#[derive(Debug, Clone, Deserialize, Serialize)]
pub struct ServiceConfig {
    pub name: String,
    pub endpoint: String,
    pub timeout_ms: u64,
    pub connection_pool_size: usize,
    #[serde(default)]
    pub auto_discover: bool,
    #[serde(default)]
    pub tls_enabled: bool,
    #[serde(default)]
    pub tls_domain: Option<String>,
    #[serde(default)]
    pub tls_ca_cert_path: Option<String>,
    #[serde(default)]
    pub circuit_breaker: CircuitBreakerConfig,
}

#[derive(Debug, Clone, Deserialize, Serialize)]
pub struct AuthConfig {
    pub service_endpoint: String,
    pub timeout_ms: u64,
    #[serde(default)]
    pub public_routes: Vec<PublicRoute>,
}

#[derive(Debug, Clone, Deserialize, Serialize)]
pub struct PublicRoute {
    pub path: String,
    pub method: String,
}

// Re-export common-rust types for configuration
pub use common_rust::circuit_breaker::CircuitBreakerConfig;
pub use common_rust::rate_limit::RateLimitConfig;

#[derive(Debug, Clone, Deserialize, Serialize)]
pub struct ObservabilityConfig {
    pub tempo_endpoint: String,
    pub service_name: String,
    #[serde(default = "default_otlp_timeout_secs")]
    pub otlp_timeout_secs: u64,
    #[serde(default = "default_max_events_per_span")]
    pub max_events_per_span: u32,
    #[serde(default = "default_max_attributes_per_span")]
    pub max_attributes_per_span: u32,
}

fn default_otlp_timeout_secs() -> u64 {
    3 // 3 seconds
}

fn default_max_events_per_span() -> u32 {
    64
}

fn default_max_attributes_per_span() -> u32 {
    16
}

#[derive(Debug, Clone, Deserialize, Serialize)]
pub struct DiscoveryConfig {
    #[serde(default = "default_refresh_interval")]
    pub refresh_interval_seconds: u64,
    #[serde(default)]
    pub enabled: bool,
}

fn default_refresh_interval() -> u64 {
    300 // 5 minutes
}

#[derive(Debug, Clone, Default, Deserialize, Serialize)]
pub struct RouteOverride {
    #[serde(default)]
    pub grpc_method: String,
    pub http_path: Option<String>,
    pub http_method: Option<String>,
    #[serde(default)]
    pub service: Option<String>,
    /// Base URL to reverse-proxy this route to (§1a). When set, the route
    /// is treated as HTTP passthrough and `grpc_method` may be empty.
    #[serde(default)]
    pub http_proxy_target: Option<String>,
}

#[derive(Debug, Clone, Deserialize, Serialize)]
pub struct CorsConfig {
    #[serde(default)]
    pub enabled: bool,
    #[serde(default)]
    pub allowed_origins: Vec<String>,
    #[serde(default)]
    pub allowed_methods: Vec<String>,
    #[serde(default)]
    pub allowed_headers: Vec<String>,
    #[serde(default)]
    pub expose_headers: Vec<String>,
    #[serde(default = "default_max_age")]
    pub max_age_seconds: u64,
    #[serde(default)]
    pub allow_credentials: bool,
}

fn default_max_age() -> u64 {
    3600 // 1 hour
}

impl Default for CorsConfig {
    fn default() -> Self {
        Self {
            enabled: false,
            allowed_origins: vec!["*".to_string()],
            allowed_methods: vec![
                "GET".to_string(),
                "POST".to_string(),
                "PUT".to_string(),
                "DELETE".to_string(),
                "OPTIONS".to_string(),
            ],
            allowed_headers: vec![
                "content-type".to_string(),
                "authorization".to_string(),
                "x-trace-id".to_string(),
            ],
            expose_headers: vec![],
            max_age_seconds: 3600,
            allow_credentials: false,
        }
    }
}

#[derive(Debug, Clone, Deserialize, Serialize)]
pub struct BodyLimitConfig {
    /// Default body size limit in bytes (applies to most endpoints)
    #[serde(default = "default_body_limit")]
    pub default_limit: usize,
    /// Upload body size limit in bytes (applies to upload endpoints)
    #[serde(default = "default_upload_limit")]
    pub upload_limit: usize,
    /// Paths that should use the upload limit
    #[serde(default = "default_upload_paths")]
    pub upload_paths: Vec<String>,
}

fn default_body_limit() -> usize {
    1024 * 1024 // 1MB
}

fn default_upload_limit() -> usize {
    10 * 1024 * 1024 // 10MB
}

fn default_upload_paths() -> Vec<String> {
    vec!["/upload".to_string(), "/api/upload".to_string()]
}

impl Default for BodyLimitConfig {
    fn default() -> Self {
        Self {
            default_limit: default_body_limit(),
            upload_limit: default_upload_limit(),
            upload_paths: default_upload_paths(),
        }
    }
}
