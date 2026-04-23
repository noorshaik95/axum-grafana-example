use axum::http::{HeaderName, HeaderValue, Method};
use std::time::Duration;
use tower_http::cors::{Any, CorsLayer};
use tracing::{info, warn};

/// CORS configuration with security-focused defaults
#[derive(Debug, Clone)]
pub struct CorsConfig {
    pub allowed_origins: Vec<String>,
    pub allowed_headers: Vec<String>,
    pub dev_mode: bool,
}

impl CorsConfig {
    /// Create a new CORS configuration
    pub fn new(allowed_origins: Vec<String>, dev_mode: bool) -> Self {
        Self::with_headers(allowed_origins, Vec::new(), dev_mode)
    }

    pub fn with_headers(
        allowed_origins: Vec<String>,
        allowed_headers: Vec<String>,
        dev_mode: bool,
    ) -> Self {
        Self {
            allowed_origins,
            allowed_headers,
            dev_mode,
        }
    }

    /// Build a CorsLayer from the configuration
    ///
    /// This method creates a properly configured CORS layer with:
    /// - Allowed origins (or all origins in dev mode)
    /// - Allowed methods (GET, POST, PUT, DELETE, PATCH, OPTIONS)
    /// - Allowed headers (content-type, authorization, x-request-id, x-trace-id)
    /// - Exposed headers (x-trace-id for error correlation)
    /// - Max age of 3600 seconds for preflight caching
    pub fn build_layer(&self) -> CorsLayer {
        let mut cors = CorsLayer::new();

        // Configure allowed origins
        if self.dev_mode {
            warn!("CORS: Allowing all origins (development mode) - DO NOT USE IN PRODUCTION");
            cors = cors.allow_origin(Any);
        } else if self.allowed_origins.is_empty() {
            warn!("CORS: No allowed origins configured, rejecting all cross-origin requests");
            // Don't set any origins - will reject all CORS requests
        } else if self.allowed_origins.contains(&"*".to_string()) {
            warn!("CORS: Wildcard origin (*) configured - consider restricting to specific origins in production");
            cors = cors.allow_origin(Any);
        } else {
            // Parse and validate origins
            let origins: Vec<HeaderValue> = self
                .allowed_origins
                .iter()
                .filter_map(|origin| match origin.parse::<HeaderValue>() {
                    Ok(header_value) => {
                        info!(origin = %origin, "CORS: Allowing origin");
                        Some(header_value)
                    }
                    Err(e) => {
                        warn!(
                            origin = %origin,
                            error = %e,
                            "CORS: Failed to parse origin, skipping"
                        );
                        None
                    }
                })
                .collect();

            if origins.is_empty() {
                warn!("CORS: No valid origins after parsing, rejecting all cross-origin requests");
            } else {
                cors = cors.allow_origin(origins);
            }
        }

        // Configure allowed methods
        // Include OPTIONS for preflight requests
        cors = cors.allow_methods([
            Method::GET,
            Method::POST,
            Method::PUT,
            Method::DELETE,
            Method::PATCH,
            Method::OPTIONS,
        ]);

        // Configure allowed headers. Start from a minimum safe set (content-type,
        // authorization, trace-propagation) and merge anything extra the yaml
        // wants — X-Tenant-ID is critical so UI requests actually pass preflight.
        let mut header_names: Vec<String> = vec![
            "content-type".into(),
            "authorization".into(),
            "x-request-id".into(),
            "x-trace-id".into(),
            "traceparent".into(),
            "tracestate".into(),
        ];
        for h in &self.allowed_headers {
            let lower = h.to_ascii_lowercase();
            if !header_names.iter().any(|n| n == &lower) {
                header_names.push(lower);
            }
        }
        let allowed_headers: Vec<HeaderName> = header_names
            .into_iter()
            .filter_map(|h| h.parse().ok())
            .collect();

        if !allowed_headers.is_empty() {
            cors = cors.allow_headers(allowed_headers);
        }

        // Configure exposed headers — clients read x-request-id / traceparent
        // to correlate with Grafana Tempo traces.
        let exposed_headers: Vec<HeaderName> = vec!["x-trace-id", "x-request-id", "traceparent"]
            .into_iter()
            .filter_map(|h| h.parse().ok())
            .collect();

        if !exposed_headers.is_empty() {
            cors = cors.expose_headers(exposed_headers);
        }

        // Set max age for preflight cache (3600 seconds = 1 hour)
        // This reduces the number of preflight requests
        cors = cors.max_age(Duration::from_secs(3600));

        // Note: allow_credentials is intentionally not set by default
        // If needed, it should be explicitly configured based on requirements
        // Setting credentials requires specific origins (not wildcard)

        info!("CORS layer configured successfully");
        cors
    }

    /// Create CORS config from environment variables
    ///
    /// Reads:
    /// - CORS_ALLOWED_ORIGINS: Comma-separated list of allowed origins
    /// - DEV_MODE or ENVIRONMENT: Set to "development" or "dev" for dev mode
    #[allow(dead_code)]
    pub fn from_env() -> Self {
        let allowed_origins_str = std::env::var("CORS_ALLOWED_ORIGINS").unwrap_or_default();

        let allowed_origins: Vec<String> = if allowed_origins_str.is_empty() {
            Vec::new()
        } else {
            allowed_origins_str
                .split(',')
                .map(|s| s.trim().to_string())
                .filter(|s| !s.is_empty())
                .collect()
        };

        let dev_mode = std::env::var("DEV_MODE")
            .or_else(|_| std::env::var("ENVIRONMENT"))
            .map(|v| {
                let lower = v.to_lowercase();
                lower == "development" || lower == "dev" || lower == "true"
            })
            .unwrap_or(false);

        if dev_mode {
            warn!("CORS: Development mode detected from environment");
        }

        if allowed_origins.is_empty() && !dev_mode {
            warn!("CORS: No allowed origins configured and not in dev mode");
        }

        Self::new(allowed_origins, dev_mode)
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_cors_config_creation() {
        let config = CorsConfig::new(vec!["https://example.com".to_string()], false);

        assert_eq!(config.allowed_origins.len(), 1);
        assert_eq!(config.allowed_origins[0], "https://example.com");
        assert!(!config.dev_mode);
    }

    #[test]
    fn test_cors_config_dev_mode() {
        let config = CorsConfig::new(Vec::new(), true);

        assert!(config.dev_mode);
        assert!(config.allowed_origins.is_empty());
    }

    #[test]
    fn test_cors_config_multiple_origins() {
        let config = CorsConfig::new(
            vec![
                "https://example.com".to_string(),
                "https://app.example.com".to_string(),
                "https://admin.example.com".to_string(),
            ],
            false,
        );

        assert_eq!(config.allowed_origins.len(), 3);
        assert!(!config.dev_mode);
    }

    #[test]
    fn test_cors_config_wildcard() {
        let config = CorsConfig::new(vec!["*".to_string()], false);

        assert_eq!(config.allowed_origins.len(), 1);
        assert_eq!(config.allowed_origins[0], "*");
    }

    #[test]
    fn test_build_layer_creates_cors_layer() {
        let config = CorsConfig::new(vec!["https://example.com".to_string()], false);

        // Should not panic
        let _layer = config.build_layer();
    }

    #[test]
    fn test_build_layer_dev_mode() {
        let config = CorsConfig::new(Vec::new(), true);

        // Should not panic and allow all origins
        let _layer = config.build_layer();
    }

    #[test]
    fn test_build_layer_empty_origins_production() {
        let config = CorsConfig::new(Vec::new(), false);

        // Should not panic but will reject all CORS requests
        let _layer = config.build_layer();
    }
}
