//! Application routing configuration.
//!
//! Defines all HTTP routes and their handlers.

use axum::{
    routing::{any, get, post},
    Router,
};
use std::sync::Arc;
use tracing::info;

use crate::auth::middleware::{auth_middleware, AuthMiddlewareState};
use crate::auth::TenantResolver;
use crate::docs;
use crate::handlers::{gateway::gateway_handler, refresh_routes_handler};
use crate::health::{health_handler, liveness_handler, readiness_handler, HealthChecker};
use crate::middleware::{body_limit_middleware, BodyLimitConfig};
use crate::shared::state::AppState;

/// Create the application router with all routes.
///
/// Registers all HTTP endpoints including health checks, metrics, and gateway routes.
pub async fn create_router(
    app_state: Arc<AppState>,
) -> Result<Router, Box<dyn std::error::Error + Send + Sync>> {
    info!("Creating application router");

    // Initialize health checker
    let health_checker = Arc::new(HealthChecker::new(app_state.grpc_pool.clone()));

    // Create auth middleware state
    let auth_middleware_state = create_auth_middleware_state(&app_state);

    // Initialize body size limit configuration
    let body_limit_config = create_body_limit_config(&app_state);

    // Create individual routers
    let health_router = create_health_router(health_checker);
    let metrics_router = create_metrics_router(app_state.clone());
    let admin_router = create_admin_router(app_state.clone(), auth_middleware_state.clone());
    let gateway_router =
        create_gateway_router(app_state.clone(), auth_middleware_state, body_limit_config);

    // Merge all routers
    let router = Router::new()
        .merge(docs::create_docs_router())
        .merge(health_router)
        .merge(metrics_router)
        .merge(admin_router)
        .merge(gateway_router);

    info!("Application router created successfully");
    Ok(router)
}

/// Create auth middleware state from app state.
fn create_auth_middleware_state(app_state: &Arc<AppState>) -> AuthMiddlewareState {
    let public_routes: Vec<(String, String)> = app_state
        .config
        .auth
        .public_routes
        .iter()
        .map(|r| (r.path.clone(), r.method.clone()))
        .collect();

    // #64: TenantResolver maps `{slug}.slate.local` Host headers to tenant
    // UUIDs by querying tenant-service's REST list endpoint. Override the
    // target with `TENANT_SERVICE_REST_URL` (defaults to the in-network
    // docker hostname).
    let tenant_service_url = std::env::var("TENANT_SERVICE_REST_URL")
        .unwrap_or_else(|_| "http://tenant-service:8083".to_string());
    let tenant_resolver = Arc::new(TenantResolver::new(tenant_service_url));

    AuthMiddlewareState {
        auth_service: app_state.auth_service.clone(),
        grpc_pool: app_state.grpc_pool.clone(),
        router_lock: app_state.router_lock.clone(),
        public_routes,
        auth_failure_counter: app_state.metrics.auth_failure_counter.clone(),
        tenant_resolver,
    }
}

/// Create health check router.
fn create_health_router(health_checker: Arc<HealthChecker>) -> Router {
    Router::new()
        .route("/health", get(health_handler))
        .route("/health/live", get(liveness_handler))
        .route("/health/ready", get(readiness_handler))
        .with_state(health_checker)
}

/// Create metrics router.
fn create_metrics_router(app_state: Arc<AppState>) -> Router {
    Router::new()
        .route("/metrics", get(metrics_handler))
        .with_state(app_state)
}

/// Create admin router with authentication.
fn create_admin_router(
    app_state: Arc<AppState>,
    auth_middleware_state: AuthMiddlewareState,
) -> Router {
    Router::new()
        .route(
            "/admin/refresh-routes",
            post(refresh_routes_handler).layer(axum::middleware::from_fn_with_state(
                auth_middleware_state,
                auth_middleware,
            )),
        )
        .with_state(app_state)
}

/// Create gateway router with middleware.
fn create_gateway_router(
    app_state: Arc<AppState>,
    auth_middleware_state: AuthMiddlewareState,
    body_limit_config: BodyLimitConfig,
) -> Router {
    Router::new()
        .route(
            "/*path",
            any(gateway_handler)
                .layer(axum::middleware::from_fn_with_state(
                    auth_middleware_state,
                    auth_middleware,
                ))
                .layer(axum::middleware::from_fn(move |req, next| {
                    body_limit_middleware(body_limit_config.clone(), req, next)
                })),
        )
        .with_state(app_state)
}

/// Create body limit configuration.
fn create_body_limit_config(_app_state: &Arc<AppState>) -> BodyLimitConfig {
    info!("Configuring request body size limits");

    let default_body_limit = std::env::var("MAX_REQUEST_BODY_SIZE")
        .ok()
        .and_then(|s| s.parse::<usize>().ok())
        .unwrap_or(1024 * 1024); // Default: 1MB

    let upload_body_limit = std::env::var("MAX_UPLOAD_BODY_SIZE")
        .ok()
        .and_then(|s| s.parse::<usize>().ok())
        .unwrap_or(10 * 1024 * 1024); // Default: 10MB

    let upload_paths = std::env::var("UPLOAD_PATHS")
        .ok()
        .map(|s| s.split(',').map(|p| p.trim().to_string()).collect())
        .unwrap_or_else(|| vec!["/upload".to_string(), "/api/upload".to_string()]);

    let body_limit_config =
        BodyLimitConfig::new(default_body_limit, upload_body_limit, upload_paths.clone());

    info!(
        default_limit_bytes = default_body_limit,
        upload_limit_bytes = upload_body_limit,
        upload_paths = ?upload_paths,
        "Body size limits configured"
    );

    body_limit_config
}

/// Metrics endpoint handler.
#[tracing::instrument(skip(state))]
async fn metrics_handler(
    axum::extract::State(state): axum::extract::State<Arc<AppState>>,
) -> axum::response::Response {
    use axum::http::{HeaderMap, HeaderValue, StatusCode};
    use axum::response::IntoResponse;
    use prometheus::{Encoder, TextEncoder};
    use tracing::error;

    let metric_families = state.registry.gather();
    let mut buffer = Vec::new();
    let encoder = TextEncoder::new();

    let (status, body) = match encoder.encode(&metric_families, &mut buffer) {
        Ok(_) => match String::from_utf8(buffer) {
            Ok(s) => (StatusCode::OK, s),
            Err(e) => {
                error!(error = %e, "Metrics not UTF-8");
                (
                    StatusCode::INTERNAL_SERVER_ERROR,
                    format!("Metrics not UTF-8: {}", e),
                )
            }
        },
        Err(e) => {
            error!(error = %e, "Failed to encode metrics");
            (
                StatusCode::INTERNAL_SERVER_ERROR,
                format!("Failed to encode metrics: {}", e),
            )
        }
    };

    let mut headers = HeaderMap::new();
    headers.insert(
        "Content-Type",
        HeaderValue::from_static("text/plain; version=0.0.4"),
    );

    (status, headers, body).into_response()
}
