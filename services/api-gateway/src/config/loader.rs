use config::{Config, ConfigError, Environment, File};
use std::path::Path;

use super::constants::*;
use super::types::GatewayConfig;

impl GatewayConfig {
    /// Load configuration from a YAML file with environment variable overrides
    pub fn load_config<P: AsRef<Path>>(path: P) -> Result<Self, ConfigError> {
        let config = Config::builder()
            // Load from YAML file
            .add_source(File::from(path.as_ref()))
            // Add environment variable overrides with GATEWAY_ prefix
            // Example: GATEWAY_SERVER__PORT=9090 will override server.port
            // Note: Double underscore (__) is used for nested fields
            .add_source(
                Environment::with_prefix("GATEWAY")
                    .separator("__")
                    .try_parsing(true),
            )
            .build()?;

        let gateway_config: GatewayConfig = config.try_deserialize()?;

        // Validate the configuration
        gateway_config.validate()?;

        Ok(gateway_config)
    }

    /// Validate the configuration for correctness and completeness
    pub fn validate(&self) -> Result<(), ConfigError> {
        // Validate server configuration
        if self.server.host.is_empty() {
            return Err(ConfigError::Message(ERR_EMPTY_HOST.to_string()));
        }

        if self.server.port == 0 {
            return Err(ConfigError::Message(ERR_ZERO_PORT.to_string()));
        }

        // Validate discovery configuration
        if self.discovery.enabled {
            // Validate refresh interval is within acceptable range (60-3600 seconds)
            if self.discovery.refresh_interval_seconds < 60
                || self.discovery.refresh_interval_seconds > 3600
            {
                return Err(ConfigError::Message(
                    "Discovery refresh interval must be between 60 and 3600 seconds".to_string(),
                ));
            }
        }

        // Validate route overrides. HTTP-proxy passthrough routes (§1a) do
        // not require a grpc_method; everything else does.
        for override_config in &self.route_overrides {
            if override_config.http_proxy_target.is_some() {
                if override_config.http_path.is_none() || override_config.http_method.is_none() {
                    return Err(ConfigError::Message(
                        "HTTP-proxy route override must specify http_path and http_method"
                            .to_string(),
                    ));
                }
                continue;
            }
            if override_config.grpc_method.is_empty() {
                return Err(ConfigError::Message(
                    "Route override must specify a grpc_method".to_string(),
                ));
            }
        }

        // Validate services
        if self.services.is_empty() {
            return Err(ConfigError::Message(ERR_NO_SERVICES.to_string()));
        }

        for (name, service) in &self.services {
            if service.name.is_empty() {
                return Err(ConfigError::Message(format!(
                    "{}: {}",
                    ERR_EMPTY_SERVICE_NAME, name
                )));
            }

            if service.endpoint.is_empty() {
                return Err(ConfigError::Message(format!(
                    "{}: {}",
                    ERR_EMPTY_SERVICE_ENDPOINT, name
                )));
            }

            if service.timeout_ms == 0 {
                return Err(ConfigError::Message(format!(
                    "{}: {}",
                    ERR_ZERO_SERVICE_TIMEOUT, name
                )));
            }

            if service.connection_pool_size == 0 {
                return Err(ConfigError::Message(format!(
                    "{}: {}",
                    ERR_ZERO_SERVICE_POOL_SIZE, name
                )));
            }
        }

        // Validate auth configuration
        if self.auth.service_endpoint.is_empty() {
            return Err(ConfigError::Message(ERR_EMPTY_AUTH_ENDPOINT.to_string()));
        }

        if self.auth.timeout_ms == 0 {
            return Err(ConfigError::Message(ERR_ZERO_AUTH_TIMEOUT.to_string()));
        }

        // Validate rate limit configuration if present
        if let Some(rate_limit) = &self.rate_limit {
            if rate_limit.enabled {
                if rate_limit.requests_per_minute == 0 {
                    return Err(ConfigError::Message(ERR_ZERO_RATE_LIMIT_RPM.to_string()));
                }

                if rate_limit.window_seconds == 0 {
                    return Err(ConfigError::Message(ERR_ZERO_RATE_LIMIT_WINDOW.to_string()));
                }
            }
        }

        // Validate observability configuration
        if self.observability.tempo_endpoint.is_empty() {
            return Err(ConfigError::Message(ERR_EMPTY_TEMPO_ENDPOINT.to_string()));
        }

        if self.observability.service_name.is_empty() {
            return Err(ConfigError::Message(
                ERR_EMPTY_OBSERVABILITY_SERVICE_NAME.to_string(),
            ));
        }

        Ok(())
    }
}
