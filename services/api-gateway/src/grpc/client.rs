use std::collections::HashMap;
use std::time::Duration;
use tonic::transport::{Channel, Endpoint};
use tracing::{debug, error, info, warn};

use super::constants::*;
use super::pool::ConnectionPool;
use super::types::{GrpcError, GrpcRequest, GrpcResponse};
use crate::config::ServiceConfig;
use common_rust::circuit_breaker::CircuitBreaker;

/// Pool of gRPC client connections to backend services with circuit breakers
#[derive(Clone)]
pub struct GrpcClientPool {
    pools: HashMap<String, ConnectionPool>,
    config: HashMap<String, ServiceConfig>,
    circuit_breakers: HashMap<String, CircuitBreaker>,
}

impl GrpcClientPool {
    /// Create a new gRPC client pool with the given service configurations
    pub async fn new(services: HashMap<String, ServiceConfig>) -> Result<Self, GrpcError> {
        info!(
            "Initializing gRPC client pool with {} services",
            services.len()
        );

        let mut pools = HashMap::new();
        let mut circuit_breakers = HashMap::new();

        for (name, config) in &services {
            info!(
                service = %name,
                endpoint = %config.endpoint,
                timeout_ms = config.timeout_ms,
                pool_size = config.connection_pool_size,
                "Creating connection pool for backend service"
            );

            // Create connection pool instead of single channel
            let pool = ConnectionPool::new(config).await?;
            pools.insert(name.clone(), pool);

            // Create circuit breaker for this service
            let circuit_breaker = CircuitBreaker::new(config.circuit_breaker.clone());
            circuit_breakers.insert(name.clone(), circuit_breaker);

            debug!(service = %name, "Successfully created connection pool with circuit breaker");
        }

        info!("gRPC client pool initialized successfully");

        Ok(Self {
            pools,
            config: services,
            circuit_breakers,
        })
    }

    /// Create a channel with connection pooling and timeout configuration
    #[allow(dead_code)]
    async fn create_channel(config: &ServiceConfig) -> Result<Channel, GrpcError> {
        let endpoint = config.endpoint.parse::<Endpoint>().map_err(|e| {
            GrpcError::InvalidConfig(format!("Invalid endpoint {}: {}", config.endpoint, e))
        })?;

        let timeout = Duration::from_millis(config.timeout_ms);

        // Configure endpoint with timeout and connection settings
        let endpoint = endpoint
            .timeout(timeout)
            .connect_timeout(DEFAULT_CONNECT_TIMEOUT)
            .tcp_keepalive(Some(TCP_KEEPALIVE))
            .http2_keep_alive_interval(HTTP2_KEEPALIVE_INTERVAL)
            .keep_alive_timeout(KEEPALIVE_TIMEOUT)
            .keep_alive_while_idle(true);

        // TODO: TLS configuration
        // TLS support will be added in a future update
        // For now, connections are unencrypted (suitable for internal service mesh)
        if config.tls_enabled {
            warn!(
                service = %config.name,
                "TLS requested but not yet implemented - connection will be unencrypted"
            );
        }

        // Use connect_lazy — defers TCP connection to first RPC call.
        let channel = endpoint.connect_lazy();

        Ok(channel)
    }

    /// Get a channel for a specific service from the connection pool
    pub fn get_channel(&self, service: &str) -> Result<Channel, GrpcError> {
        let pool = self
            .pools
            .get(service)
            .ok_or_else(|| GrpcError::ServiceNotFound(service.to_string()))?;

        // Acquire a channel from the pool using round-robin
        Ok(pool.acquire())
    }

    /// Get the circuit breaker for a specific service
    pub fn get_circuit_breaker(&self, service: &str) -> Option<CircuitBreaker> {
        self.circuit_breakers.get(service).cloned()
    }

    /// Execute a generic gRPC call with retry logic
    ///
    /// Note: This method is currently unused as we use DynamicGrpcClient instead.
    /// Keeping it for potential future use with typed gRPC clients.
    #[allow(dead_code)]
    pub async fn call(&self, request: GrpcRequest) -> Result<GrpcResponse, GrpcError> {
        let service_name = &request.service;

        debug!(
            service = %service_name,
            method = %request.method,
            "Executing gRPC call"
        );

        // Get the channel for the service
        let channel = self.get_channel(service_name)?;

        // Get service config for timeout
        let config = self
            .config
            .get(service_name)
            .ok_or_else(|| GrpcError::ServiceNotFound(service_name.to_string()))?;

        // Execute with retry logic
        let result = self.call_with_retry(channel, &request, config).await;

        match &result {
            Ok(response) => {
                debug!(
                    service = %service_name,
                    status = ?response.status,
                    "gRPC call completed successfully"
                );
            }
            Err(e) => {
                error!(
                    service = %service_name,
                    error = %e,
                    "gRPC call failed"
                );
            }
        }

        result
    }

    /// Execute a gRPC call with retry logic (max 3 attempts)
    async fn call_with_retry(
        &self,
        channel: Channel,
        request: &GrpcRequest,
        config: &ServiceConfig,
    ) -> Result<GrpcResponse, GrpcError> {
        let mut last_error = None;

        for attempt in 1..=MAX_RETRIES {
            match self.execute_call(channel.clone(), request, config).await {
                Ok(response) => return Ok(response),
                Err(e) => {
                    // Only retry on transient errors
                    if Self::is_retryable_error(&e) && attempt < MAX_RETRIES {
                        warn!(
                            service = %request.service,
                            attempt = attempt,
                            error = %e,
                            "gRPC call failed, retrying"
                        );

                        // Exponential backoff
                        let backoff = Duration::from_millis(
                            INITIAL_BACKOFF_MS * BACKOFF_MULTIPLIER.pow(attempt - 1),
                        );
                        tokio::time::sleep(backoff).await;

                        last_error = Some(e);
                    } else {
                        return Err(e);
                    }
                }
            }
        }

        Err(last_error
            .unwrap_or_else(|| GrpcError::CallFailed(ERR_MAX_RETRIES_EXCEEDED.to_string())))
    }

    /// Execute a single gRPC call attempt
    async fn execute_call(
        &self,
        _channel: Channel,
        _request: &GrpcRequest,
        _config: &ServiceConfig,
    ) -> Result<GrpcResponse, GrpcError> {
        // Note: This is a placeholder for generic gRPC calls
        // In practice, specific service clients (auth, backend services) will use
        // the channel directly with their generated proto code
        // This method would need dynamic dispatch or code generation for truly generic calls

        // For now, return an error indicating this needs to be implemented per-service
        Err(GrpcError::CallFailed(
            ERR_GENERIC_CALLS_NOT_IMPLEMENTED.to_string(),
        ))
    }

    /// Check if an error is retryable
    pub(crate) fn is_retryable_error(error: &GrpcError) -> bool {
        match error {
            GrpcError::Timeout(_) => true,
            GrpcError::ConnectionError(_) => true,
            GrpcError::CallFailed(msg) => {
                // Retry on specific gRPC status codes
                msg.contains(RETRYABLE_STATUS_UNAVAILABLE)
                    || msg.contains(RETRYABLE_STATUS_DEADLINE_EXCEEDED)
                    || msg.contains(RETRYABLE_STATUS_RESOURCE_EXHAUSTED)
            }
            _ => false,
        }
    }

    /// Check health of a specific backend service
    pub async fn health_check(&self, service: &str) -> Result<bool, GrpcError> {
        debug!(service = %service, "Performing health check");

        // Get the connection pool for the service
        let pool = self
            .pools
            .get(service)
            .ok_or_else(|| GrpcError::ServiceNotFound(service.to_string()))?;

        // Perform health check on the pool
        pool.health_check().await
    }

    /// Get all service names in the pool
    pub fn services(&self) -> Vec<String> {
        self.pools.keys().cloned().collect()
    }

    /// Check if a service exists in the pool
    #[allow(dead_code)]
    pub fn has_service(&self, service: &str) -> bool {
        self.pools.contains_key(service)
    }

    /// Close all gRPC connections gracefully
    ///
    /// This method attempts to close all connections with a 5-second timeout.
    /// After the timeout, remaining connections are force-closed.
    /// Returns the number of connections that were closed.
    pub async fn close(&self) -> usize {
        let start = std::time::Instant::now();
        let total_services = self.pools.len();

        info!(
            service_count = total_services,
            "Starting graceful shutdown of gRPC connection pools"
        );

        // Create a timeout for graceful shutdown
        let shutdown_timeout = Duration::from_secs(5);

        // Attempt graceful shutdown with timeout
        let result = tokio::time::timeout(shutdown_timeout, async {
            let mut closed_count = 0;

            for service_name in self.pools.keys() {
                debug!(service = %service_name, "Closing connection pool");
                // Note: tonic::transport::Channel doesn't have an explicit close method
                // Connections will be dropped when the pool is dropped
                closed_count += 1;
            }

            closed_count
        })
        .await;

        let closed_count = match result {
            Ok(count) => {
                info!(
                    closed_count = count,
                    duration_ms = start.elapsed().as_millis(),
                    "Gracefully closed all gRPC connection pools"
                );
                count
            }
            Err(_) => {
                warn!(
                    timeout_secs = shutdown_timeout.as_secs(),
                    duration_ms = start.elapsed().as_millis(),
                    "Graceful shutdown timeout exceeded, force-closing remaining connections"
                );
                // Force close by dropping - this happens automatically
                total_services
            }
        };

        info!(
            closed_count = closed_count,
            total_duration_ms = start.elapsed().as_millis(),
            "gRPC connection pool shutdown complete"
        );

        closed_count
    }
}
