# Migration Guide: Adopting common-rust

This guide provides step-by-step instructions for migrating your Rust microservice to use the `common-rust` shared library.

## Overview

The `common-rust` library consolidates common functionality across Rust microservices:

- Circuit breaker for fault tolerance
- Rate limiting with sliding window algorithm
- Health checks for monitoring
- Retry logic with exponential backoff
- Error response handling
- Observability utilities (tracing, logging)

## Prerequisites

- Rust 1.70 or later
- Existing Rust microservice in the monorepo

## Migration Steps

### Step 1: Add Dependency

Add `common-rust` to your service's `Cargo.toml`:

```toml
[dependencies]
common-rust = { path = "../../libs/common-rust", features = ["full"] }
```

**Feature Flags:**

- `default`: Includes `observability` feature
- `observability`: OpenTelemetry tracing and logging
- `grpc`: gRPC interceptors and utilities
- `http`: HTTP utilities and Axum integration
- `full`: All features enabled

Choose the features you need. For most services, `["full"]` is recommended.

### Step 2: Replace Circuit Breaker

**Before (api-gateway):**

```rust
use crate::circuit_breaker::{CircuitBreaker, CircuitBreakerConfig};

let config = CircuitBreakerConfig {
    failure_threshold: 5,
    success_threshold: 2,
    timeout_seconds: 30,
};

let breaker = CircuitBreaker::new(config);
```

**After:**

```rust
use common_rust::circuit_breaker::{CircuitBreaker, CircuitBreakerConfig};

let config = CircuitBreakerConfig {
    failure_threshold: 5,
    success_threshold: 2,
    timeout_seconds: 30,
};

let breaker = CircuitBreaker::with_name("my-service".to_string(), config);
```

**Breaking Changes:**

- `content-management-service` used `timeout: Duration` - now use `timeout_seconds: u64`
- Migration: `Duration::from_secs(60)` → `timeout_seconds: 60`

**Files to Remove:**

- `src/circuit_breaker/` (entire directory)

### Step 3: Replace Rate Limiter

**Before (api-gateway):**

```rust
use crate::rate_limit::{RateLimiter, RateLimitConfig};
use std::net::IpAddr;

let config = RateLimitConfig {
    enabled: true,
    requests_per_minute: 60,
    window_seconds: 60,
};

let limiter = RateLimiter::new(config, 10_000);
```

**After:**

```rust
use common_rust::rate_limit::{IpRateLimiter, RateLimitConfig};

let config = RateLimitConfig {
    enabled: true,
    requests_per_minute: 60,
    window_seconds: 60,
};

let limiter = IpRateLimiter::new(config, 10_000);
```

**Files to Remove:**

- `src/rate_limit/` (entire directory)

### Step 4: Replace Observability Utilities

**Before (api-gateway):**

```rust
use crate::observability::tracing_utils::{extract_trace_id_from_span, TraceContext};
```

**After:**

```rust
use common_rust::observability::{extract_trace_id_from_span, TraceContext};
```

**Before (content-management-service):**

```rust
use crate::observability::interceptor::{extract_trace_context_from_grpc, log_grpc_metadata};
```

**After:**

```rust
use common_rust::observability::{extract_trace_context_from_grpc, log_grpc_metadata};
```

**Files to Remove:**

- `src/observability/tracing_utils.rs`
- `src/observability/interceptor.rs`
- `src/observability/json_formatter.rs` (moved to common-rust)

**Files to Keep:**

- Service-specific observability code (metrics, custom logging)

### Step 5: Add Health Checks (New Feature)

```rust
use common_rust::health::{HealthChecker, HealthStatus, CircuitBreakerHealthCheck};

// Create health checker
let mut health_checker = HealthChecker::new("my-service");

// Register circuit breaker health check
let cb_health = CircuitBreakerHealthCheck::new(
    "database".to_string(),
    database_circuit_breaker.clone()
);
health_checker.register(Box::new(cb_health)).await;

// Liveness endpoint
let liveness = health_checker.liveness().await;

// Readiness endpoint
let readiness = health_checker.readiness().await;
```

### Step 6: Add Retry Logic (New Feature)

```rust
use common_rust::retry::{retry_operation, OperationType};

// Use preset configuration
let result = retry_operation(
    OperationType::Database,
    || async { database_query().await }
).await;

// Or use custom configuration
use common_rust::retry::{retry_with_backoff, RetryConfig};

let config = RetryConfig {
    max_attempts: 3,
    initial_backoff_ms: 100,
    max_backoff_ms: 2000,
    backoff_multiplier: 2.0,
};

let result = retry_with_backoff(config, || async {
    external_api_call().await
}).await;
```

### Step 7: Use Error Response (New Feature)

```rust
use common_rust::error::{ErrorResponse, extract_trace_id_from_http};

// In HTTP handler (with http feature)
async fn handler(
    headers: axum::http::HeaderMap,
) -> Result<axum::Json<Data>, axum::response::Response> {
    let trace_id = extract_trace_id_from_http(&headers);

    // On error
    let error = ErrorResponse::new(
        "VALIDATION_ERROR",
        "Invalid input",
        trace_id
    );

    return Err(error.to_http_response(400));
}

// In gRPC handler (with grpc feature)
async fn grpc_handler(
    request: tonic::Request<MyRequest>,
) -> Result<tonic::Response<MyResponse>, tonic::Status> {
    use common_rust::error::extract_trace_id_from_grpc;

    let trace_id = extract_trace_id_from_grpc(&request);

    // On error
    let error = ErrorResponse::new(
        "NOT_FOUND",
        "Resource not found",
        trace_id
    );

    return Err(error.to_grpc_status(tonic::Code::NotFound));
}
```

### Step 8: Update Docker Configuration

Update your service's `Dockerfile` to include the `common-rust` library:

**Before:**

```dockerfile
COPY services/my-service ./my-service
WORKDIR /app/my-service
RUN cargo build --release
```

**After:**

```dockerfile
# Copy common-rust library
COPY libs/common-rust ./common-rust

# Copy service
COPY services/my-service ./my-service

# Build service (path dependency will find common-rust)
WORKDIR /app/my-service
RUN cargo build --release
```

### Step 9: Run Tests

```bash
# Test your service
cd services/my-service
cargo test

# Test with all features
cargo test --all-features

# Run specific test
cargo test test_circuit_breaker
```

### Step 10: Verify Functionality

1. **Build succeeds:**

   ```bash
   cargo build --release
   ```

2. **Docker build succeeds:**

   ```bash
   docker build -t my-service .
   ```

3. **Service starts correctly:**

   ```bash
   docker-compose up my-service
   ```

4. **Circuit breaker works:**
   - Test with failing backend
   - Verify circuit opens after threshold
   - Verify circuit recovers

5. **Rate limiting works:**
   - Send multiple requests
   - Verify rate limit enforcement

6. **Tracing works:**
   - Check logs for trace IDs
   - Verify trace propagation

## Verification Checklist

- [ ] Added `common-rust` dependency to `Cargo.toml`
- [ ] Updated circuit breaker imports
- [ ] Updated rate limiter imports (if applicable)
- [ ] Updated observability imports
- [ ] Removed old module directories
- [ ] Updated Dockerfile to copy `common-rust`
- [ ] All tests pass (`cargo test`)
- [ ] Docker build succeeds
- [ ] Service starts correctly
- [ ] Circuit breaker functionality verified
- [ ] Rate limiting functionality verified (if applicable)
- [ ] Tracing functionality verified
- [ ] No compilation warnings

## Common Issues

### Issue: "cannot find crate `common_rust`"

**Solution:** Ensure the path in `Cargo.toml` is correct:

```toml
common-rust = { path = "../../libs/common-rust", features = ["full"] }
```

### Issue: Docker build fails with "no such file or directory"

**Solution:** Ensure `COPY libs/common-rust ./common-rust` is before your service copy in Dockerfile.

### Issue: "timeout_seconds" field not found

**Solution:** You're using old config format. Change:

```rust
// Old
timeout: Duration::from_secs(60)

// New
timeout_seconds: 60
```

### Issue: Tests fail after migration

**Solution:**

1. Check that all imports are updated
2. Verify configuration structs match new format
3. Run `cargo clean` and rebuild

## Getting Help

If you encounter issues:

1. Check this migration guide
2. Review the [common-rust README](../../libs/common-rust/README.md)
3. Look at existing migrations (api-gateway, content-management-service)
4. Ask in #rust-services channel

## Example Migrations

See these PRs for complete migration examples:

- API Gateway: [Link to PR]
- Content Management Service: [Link to PR]
