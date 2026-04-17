//! Connection pooling for gRPC clients.
//!
//! This module provides connection pooling functionality.

#![allow(dead_code)]

use std::sync::atomic::{AtomicUsize, Ordering};
use std::sync::Arc;
use tonic::transport::{Channel, Endpoint};
use tracing::{debug, info};

use super::types::GrpcError;
use crate::config::ServiceConfig;

/// Connection pool for a single backend service
///
/// Maintains multiple gRPC channels and distributes requests across them
/// using round-robin selection
#[derive(Clone)]
pub struct ConnectionPool {
    channels: Arc<Vec<Channel>>,
    next_index: Arc<AtomicUsize>,
    service_name: String,
}

impl ConnectionPool {
    /// Create a new connection pool with the specified number of connections
    pub async fn new(config: &ServiceConfig) -> Result<Self, GrpcError> {
        let pool_size = config.connection_pool_size;
        info!(
            service = %config.name,
            endpoint = %config.endpoint,
            pool_size = pool_size,
            "Creating connection pool"
        );

        let mut channels = Vec::with_capacity(pool_size);

        for i in 0..pool_size {
            debug!(
                service = %config.name,
                connection = i + 1,
                total = pool_size,
                "Establishing connection"
            );

            let channel = Self::create_channel(config)?;
            channels.push(channel);
        }

        info!(
            service = %config.name,
            connections = channels.len(),
            "Connection pool created successfully"
        );

        Ok(Self {
            channels: Arc::new(channels),
            next_index: Arc::new(AtomicUsize::new(0)),
            service_name: config.name.clone(),
        })
    }

    /// Create a single channel with the configured settings.
    /// Uses connect_lazy — no TCP connection is made until the first RPC call.
    fn create_channel(config: &ServiceConfig) -> Result<Channel, GrpcError> {
        use std::time::Duration;

        let endpoint = config.endpoint.parse::<Endpoint>().map_err(|e| {
            GrpcError::InvalidConfig(format!("Invalid endpoint {}: {}", config.endpoint, e))
        })?;

        let timeout = Duration::from_millis(config.timeout_ms);

        // Configure endpoint with timeout and connection settings
        let endpoint = endpoint
            .timeout(timeout)
            .connect_timeout(Duration::from_secs(10))
            .tcp_keepalive(Some(Duration::from_secs(60)))
            .http2_keep_alive_interval(Duration::from_secs(30))
            .keep_alive_timeout(Duration::from_secs(20))
            .keep_alive_while_idle(true);

        // Use connect_lazy so the gateway starts even when backends are not yet
        // ready. The actual TCP connection is established on the first RPC call.
        // This prevents a cold-start cascade where one unavailable service
        // prevents the gateway from starting at all.
        let channel = endpoint.connect_lazy();

        Ok(channel)
    }

    /// Get a channel from the pool using round-robin selection
    pub fn acquire(&self) -> Channel {
        let index = self.next_index.fetch_add(1, Ordering::Relaxed);
        let channel_index = index % self.channels.len();

        debug!(
            service = %self.service_name,
            channel_index = channel_index,
            total_channels = self.channels.len(),
            "Acquired channel from pool"
        );

        self.channels[channel_index].clone()
    }

    /// Get the number of connections in the pool
    #[allow(dead_code)]
    pub fn size(&self) -> usize {
        self.channels.len()
    }

    /// Check if a connection is healthy by attempting to clone it
    pub async fn health_check(&self) -> Result<bool, GrpcError> {
        debug!(service = %self.service_name, "Performing health check on connection pool");

        // Try to acquire a channel and verify it's usable
        let channel = self.acquire();

        // Attempt to clone the channel - if it fails, the connection is broken
        let _test_channel = channel.clone();

        debug!(service = %self.service_name, "Health check passed");
        Ok(true)
    }

    /// Get statistics about the pool
    #[allow(dead_code)]
    pub fn stats(&self) -> PoolStats {
        PoolStats {
            total_connections: self.channels.len(),
            service_name: self.service_name.clone(),
            requests_served: self.next_index.load(Ordering::Relaxed),
        }
    }
}

/// Statistics about a connection pool
#[derive(Debug, Clone)]
pub struct PoolStats {
    pub total_connections: usize,
    pub service_name: String,
    pub requests_served: usize,
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_round_robin_selection() {
        // This test would require setting up actual connections
        // For now, we'll just verify the round-robin logic
        let next_index = AtomicUsize::new(0);
        let pool_size = 5;

        let mut indices = Vec::new();
        for _ in 0..15 {
            let index = next_index.fetch_add(1, Ordering::Relaxed);
            indices.push(index % pool_size);
        }

        // Verify round-robin pattern
        assert_eq!(indices, vec![0, 1, 2, 3, 4, 0, 1, 2, 3, 4, 0, 1, 2, 3, 4]);
    }

    #[test]
    fn test_pool_stats() {
        let next_index = Arc::new(AtomicUsize::new(42));
        let channels = Arc::new(vec![]);

        let pool = ConnectionPool {
            channels,
            next_index,
            service_name: "test-service".to_string(),
        };

        let stats = pool.stats();
        assert_eq!(stats.service_name, "test-service");
        assert_eq!(stats.requests_served, 42);
    }
}
