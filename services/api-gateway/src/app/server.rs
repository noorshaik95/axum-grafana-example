//! HTTP server configuration and setup.
//!
//! Handles creation and configuration of the Axum HTTP server with all middleware.

use axum::Router;
use std::net::SocketAddr;
use std::sync::Arc;
use tokio::net::TcpListener;
use tower_http::trace::TraceLayer;
use tracing::{debug, info, warn};

use crate::shared::state::AppState;
use common_rust::rate_limit::IpRateLimiter;

use super::routes::create_router;

/// Start the HTTP server with all middleware and handle graceful shutdown.
///
/// This function creates the server, starts background tasks, and handles
/// the complete server lifecycle including graceful shutdown.
pub async fn start_server(
    app_state: Arc<AppState>,
) -> Result<(), Box<dyn std::error::Error + Send + Sync>> {
    info!("Creating HTTP server");

    // Start background tasks
    let cleanup_handles = start_background_tasks(&app_state);

    // Create and configure server
    let (server_handle, shutdown_tx) = create_and_start_server(app_state.clone()).await?;

    info!("API Gateway is ready to accept requests");

    // Wait for shutdown signal
    wait_for_shutdown_signal().await;

    // Perform graceful shutdown
    perform_graceful_shutdown(app_state, cleanup_handles, server_handle, shutdown_tx).await?;

    Ok(())
}

/// Create and start the HTTP server.
async fn create_and_start_server(
    app_state: Arc<AppState>,
) -> Result<
    (
        tokio::task::JoinHandle<()>,
        tokio::sync::oneshot::Sender<()>,
    ),
    Box<dyn std::error::Error + Send + Sync>,
> {
    // Create the application router
    let app_router = create_router(app_state.clone()).await?;

    // Apply middleware layers
    let app = apply_middleware(app_router, &app_state)?;

    // Create TCP listener
    let listener = create_tcp_listener(&app_state).await?;
    let addr = listener.local_addr()?;

    log_server_info(&app_state, addr);

    // Setup graceful shutdown
    let (shutdown_tx, shutdown_rx) = tokio::sync::oneshot::channel::<()>();

    let server = axum::serve(
        listener,
        app.into_make_service_with_connect_info::<SocketAddr>(),
    );

    let server_handle = tokio::spawn(async move {
        server
            .with_graceful_shutdown(async {
                shutdown_rx.await.ok();
            })
            .await
            .expect("Server failed to start or encountered an error during execution");
    });

    Ok((server_handle, shutdown_tx))
}

/// Log server information.
fn log_server_info(app_state: &Arc<AppState>, addr: SocketAddr) {
    info!(
        host = %app_state.config.server.host,
        port = app_state.config.server.port,
        actual_addr = %addr,
        "HTTP server configured"
    );

    info!(
        "API Documentation available at: http://{}:{}/docs",
        app_state.config.server.host, app_state.config.server.port
    );
}

/// Wait for shutdown signal.
async fn wait_for_shutdown_signal() {
    tokio::select! {
        _ = tokio::signal::ctrl_c() => {
            info!("Received shutdown signal (Ctrl+C)");
        }
    }
}

/// Perform graceful shutdown of all components.
async fn perform_graceful_shutdown(
    app_state: Arc<AppState>,
    cleanup_handles: CleanupHandles,
    server_handle: tokio::task::JoinHandle<()>,
    shutdown_tx: tokio::sync::oneshot::Sender<()>,
) -> Result<(), Box<dyn std::error::Error + Send + Sync>> {
    let shutdown_start = std::time::Instant::now();
    info!("Initiating graceful shutdown");

    // Send shutdown signal to server
    let _ = shutdown_tx.send(());

    // Abort background tasks
    abort_background_tasks(cleanup_handles);

    // Wait for server to finish
    let _ = server_handle.await;

    // Close all gRPC connections
    close_grpc_connections(&app_state).await;

    let total_shutdown_duration = shutdown_start.elapsed();
    info!(
        total_shutdown_duration_ms = total_shutdown_duration.as_millis(),
        "Graceful shutdown complete"
    );

    Ok(())
}

/// Abort background tasks.
fn abort_background_tasks(cleanup_handles: CleanupHandles) {
    if let Some(handle) = cleanup_handles.refresh_task {
        info!("Stopping periodic route refresh task");
        handle.abort();
    }

    if let Some(handle) = cleanup_handles.rate_limiter_cleanup {
        info!("Stopping rate limiter cleanup task");
        handle.abort();
    }
}

/// Close gRPC connections.
async fn close_grpc_connections(app_state: &Arc<AppState>) {
    info!("Closing gRPC connection pools");
    let closed_count = app_state.grpc_pool.close().await;
    info!(closed_connections = closed_count, "gRPC connections closed");
}

/// Cleanup task handles for graceful shutdown.
struct CleanupHandles {
    refresh_task: Option<tokio::task::JoinHandle<()>>,
    rate_limiter_cleanup: Option<tokio::task::JoinHandle<()>>,
}

/// Apply middleware layers to the router.
fn apply_middleware(
    router: Router,
    app_state: &Arc<AppState>,
) -> Result<Router, Box<dyn std::error::Error + Send + Sync>> {
    info!("Applying middleware layers");

    // X-Request-ID + W3C traceparent: must run first so every downstream layer
    // (including the TraceLayer span and gRPC clients) sees the canonical values.
    //
    // #63: `TraceLayer::make_span_with(make_gateway_span)` produces the
    // per-request root span. `make_gateway_span` runs at span-creation time
    // (before tracing-opentelemetry attaches an OTEL context), which is the
    // correct point to adopt an inbound W3C traceparent as the OTEL remote
    // parent. Doing this in a regular middleware runs too late — children
    // have already been bound to the fresh (wrong) OTEL context.
    let mut app = router
        .layer(axum::middleware::from_fn(
            crate::middleware::request_id_middleware,
        ))
        .layer(
            TraceLayer::new_for_http()
                .make_span_with(crate::middleware::request_id::make_gateway_span),
        );

    // Add CORS middleware if enabled
    if let Some(cors_config) = &app_state.config.cors {
        if cors_config.enabled {
            app = apply_cors_middleware(app, app_state)?;
        } else {
            info!("CORS middleware disabled in configuration");
        }
    } else {
        info!("CORS configuration not found, CORS middleware not applied");
    }

    Ok(app)
}

/// Apply CORS middleware.
fn apply_cors_middleware(
    router: Router,
    app_state: &Arc<AppState>,
) -> Result<Router, Box<dyn std::error::Error + Send + Sync>> {
    use crate::middleware::CorsConfig;

    info!("Configuring CORS middleware");

    let cors_config = app_state.config.cors.as_ref().unwrap();

    // Determine dev mode and allowed origins
    let dev_mode = determine_dev_mode();
    let allowed_origins = get_allowed_origins(cors_config);

    // Create CORS configuration — pass through allowed_headers from yaml so
    // app-specific headers like X-Tenant-ID don't get blocked at preflight.
    let allowed_headers = cors_config.allowed_headers.clone();
    let cors_middleware_config =
        CorsConfig::with_headers(allowed_origins.clone(), allowed_headers, dev_mode);

    // Log configuration
    log_cors_configuration(dev_mode, &allowed_origins);

    // Build and apply CORS layer
    let cors_layer = cors_middleware_config.build_layer();
    let app = router.layer(cors_layer);

    info!("CORS middleware configured successfully");
    Ok(app)
}

/// Determine if running in development mode.
fn determine_dev_mode() -> bool {
    std::env::var("DEV_MODE")
        .or_else(|_| std::env::var("ENVIRONMENT"))
        .map(|v| {
            let lower = v.to_lowercase();
            lower == "development" || lower == "dev" || lower == "true"
        })
        .unwrap_or(false)
}

/// Get allowed origins from environment or config.
fn get_allowed_origins(cors_config: &crate::config::CorsConfig) -> Vec<String> {
    std::env::var("CORS_ALLOWED_ORIGINS")
        .ok()
        .map(|s| {
            s.split(',')
                .map(|origin| origin.trim().to_string())
                .filter(|origin| !origin.is_empty())
                .collect()
        })
        .unwrap_or_else(|| cors_config.allowed_origins.clone())
}

/// Log CORS configuration details.
fn log_cors_configuration(dev_mode: bool, allowed_origins: &[String]) {
    if dev_mode {
        warn!("CORS: Running in development mode - allowing all origins");
    } else if allowed_origins.is_empty() {
        warn!("CORS: No allowed origins configured - will reject all cross-origin requests");
    } else if allowed_origins.contains(&"*".to_string()) {
        warn!("CORS: Wildcard origin configured - consider restricting in production");
    } else {
        info!(
            allowed_origins = ?allowed_origins,
            "CORS: Configured with specific allowed origins"
        );
    }
}

/// Create TCP listener for the server.
async fn create_tcp_listener(
    app_state: &Arc<AppState>,
) -> Result<TcpListener, Box<dyn std::error::Error + Send + Sync>> {
    let bind_addr = SocketAddr::new(
        app_state.config.server.host.parse()?,
        app_state.config.server.port,
    );

    info!(addr = %bind_addr, "Binding TCP listener");

    let listener = TcpListener::bind(bind_addr)
        .await
        .map_err(|e| format!("Failed to bind to {}: {}", bind_addr, e))?;

    Ok(listener)
}

/// Start background tasks for rate limiter cleanup and route refresh.
fn start_background_tasks(app_state: &Arc<AppState>) -> CleanupHandles {
    let rate_limiter_cleanup = start_rate_limiter_cleanup(app_state);
    let refresh_task = start_route_refresh_task(app_state);

    CleanupHandles {
        refresh_task,
        rate_limiter_cleanup,
    }
}

/// Start rate limiter cleanup background task.
fn start_rate_limiter_cleanup(app_state: &Arc<AppState>) -> Option<tokio::task::JoinHandle<()>> {
    if let Some(ref limiter) = app_state.rate_limiter {
        info!("Starting rate limiter cleanup background task (interval: 1 minute)");
        let limiter_clone = (**limiter).clone();
        let metrics_clone = Arc::new(app_state.metrics.clone());

        Some(tokio::spawn(async move {
            run_rate_limiter_cleanup_loop(limiter_clone, metrics_clone).await;
        }))
    } else {
        None
    }
}

/// Run the rate limiter cleanup loop.
async fn run_rate_limiter_cleanup_loop(
    limiter: IpRateLimiter,
    metrics: Arc<crate::shared::state::GatewayMetrics>,
) {
    let mut interval = tokio::time::interval(std::time::Duration::from_secs(60));

    loop {
        interval.tick().await;
        perform_rate_limiter_cleanup(&limiter, &metrics).await;
    }
}

/// Perform a single rate limiter cleanup cycle.
async fn perform_rate_limiter_cleanup(
    limiter: &IpRateLimiter,
    metrics: &Arc<crate::shared::state::GatewayMetrics>,
) {
    // Get tracked clients count before cleanup
    let tracked_before = limiter.tracked_clients_count().await;

    // Perform cleanup and get eviction count
    let evicted_count = limiter.cleanup_expired().await;

    // Get tracked clients count after cleanup
    let tracked_after = limiter.tracked_clients_count().await;

    // Update Prometheus metrics
    update_rate_limiter_metrics(metrics, tracked_after, evicted_count);

    // Log cleanup results
    log_cleanup_results(tracked_before, tracked_after, evicted_count);
}

/// Update rate limiter Prometheus metrics.
fn update_rate_limiter_metrics(
    metrics: &Arc<crate::shared::state::GatewayMetrics>,
    tracked_after: usize,
    evicted_count: usize,
) {
    metrics
        .rate_limiter_tracked_clients
        .set(tracked_after as i64);
    metrics
        .rate_limiter_evictions_total
        .inc_by(evicted_count as u64);
}

/// Log rate limiter cleanup results.
fn log_cleanup_results(tracked_before: usize, tracked_after: usize, evicted_count: usize) {
    info!(
        tracked_clients = tracked_after,
        evicted_entries = evicted_count,
        "Rate limiter cleanup completed"
    );

    debug!(
        tracked_before = tracked_before,
        tracked_after = tracked_after,
        evicted = evicted_count,
        "Rate limiter cleanup details"
    );
}

/// Start periodic route refresh background task.
fn start_route_refresh_task(app_state: &Arc<AppState>) -> Option<tokio::task::JoinHandle<()>> {
    if let Some(discovery_svc) = app_state.discovery_service.clone() {
        info!("Starting periodic refresh background task");
        let services = app_state.config.services.clone();
        let router_lck = app_state.router_lock.clone();
        let route_overrides = app_state.config.route_overrides.clone();

        Some(discovery_svc.start_refresh_task(router_lck, services, route_overrides))
    } else {
        None
    }
}
