use serde::{Deserialize, Serialize};
use std::time::Instant;

/// Circuit breaker configuration
#[derive(Debug, Clone, Deserialize, Serialize)]
pub struct CircuitBreakerConfig {
    /// Number of consecutive failures before opening circuit
    #[serde(default = "default_failure_threshold")]
    pub failure_threshold: u32,

    /// Number of consecutive successes in half-open before closing
    #[serde(default = "default_success_threshold")]
    pub success_threshold: u32,

    /// Seconds to wait before transitioning from open to half-open
    #[serde(default = "default_timeout_seconds")]
    pub timeout_seconds: u64,
}

fn default_failure_threshold() -> u32 {
    5 // Open circuit after 5 consecutive failures
}

fn default_success_threshold() -> u32 {
    2 // Close circuit after 2 consecutive successes in half-open state
}

fn default_timeout_seconds() -> u64 {
    // R2: reduced from 30s → 10s; overridable per-service in gateway yaml
    // circuit_breaker.timeout_seconds or via GATEWAY_SERVICES__<NAME>__CIRCUIT_BREAKER__TIMEOUT_SECONDS
    std::env::var("CIRCUIT_BREAKER_TIMEOUT_SECONDS")
        .ok()
        .and_then(|v| v.parse().ok())
        .unwrap_or(10)
}

impl Default for CircuitBreakerConfig {
    fn default() -> Self {
        Self {
            failure_threshold: default_failure_threshold(),
            success_threshold: default_success_threshold(),
            timeout_seconds: default_timeout_seconds(),
        }
    }
}

/// Circuit breaker state
#[derive(Debug, Clone, PartialEq)]
pub enum CircuitState {
    /// Circuit is closed, requests pass through normally
    Closed,
    /// Circuit is open, requests are rejected immediately
    Open { opened_at: Instant },
    /// Circuit is testing if service has recovered
    HalfOpen,
}

/// Circuit breaker error
#[derive(Debug, thiserror::Error)]
pub enum CircuitBreakerError {
    #[error("Circuit breaker is open - service unavailable")]
    Open,

    #[error("Operation failed: {0}")]
    OperationFailed(String),
}

/// Circuit breaker statistics for monitoring
#[derive(Debug, Clone)]
pub struct CircuitBreakerStats {
    pub name: Option<String>,
    pub state: CircuitState,
    pub consecutive_failures: u32,
    pub consecutive_successes: u32,
    pub total_failures: u64,
    pub total_successes: u64,
}

/// Internal state tracking for circuit breaker
#[derive(Debug)]
pub(crate) struct CircuitStats {
    pub consecutive_failures: u32,
    pub consecutive_successes: u32,
    pub total_failures: u64,
    pub total_successes: u64,
}

impl Default for CircuitStats {
    fn default() -> Self {
        Self {
            consecutive_failures: 0,
            consecutive_successes: 0,
            total_failures: 0,
            total_successes: 0,
        }
    }
}
