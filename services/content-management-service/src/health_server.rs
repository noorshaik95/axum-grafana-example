use crate::health::HealthChecker;
use axum::{
    extract::State,
    http::StatusCode,
    response::{IntoResponse, Json, Response},
    routing::get,
    Router,
};
use std::sync::Arc;

/// Health server state
#[derive(Clone)]
pub struct HealthState {
    pub health_checker: Arc<HealthChecker>,
}

/// Start the health check HTTP server (can be combined with metrics server)
pub fn health_routes(health_checker: Arc<HealthChecker>) -> Router {
    let state = HealthState { health_checker };

    Router::new()
        .route("/health", get(readiness_handler))
        .route("/health/live", get(liveness_handler))
        .route("/health/ready", get(readiness_handler))
        .with_state(state)
}

/// Liveness probe handler - checks if the service is running
async fn liveness_handler(State(state): State<HealthState>) -> Response {
    let health = state.health_checker.liveness().await;

    match health.status {
        crate::health::HealthStatus::Healthy => (StatusCode::OK, Json(health)).into_response(),
        _ => (StatusCode::SERVICE_UNAVAILABLE, Json(health)).into_response(),
    }
}

/// Readiness probe handler - checks if the service is ready to handle requests
async fn readiness_handler(State(state): State<HealthState>) -> Response {
    let health = state.health_checker.readiness().await;

    match health.status {
        crate::health::HealthStatus::Healthy => (StatusCode::OK, Json(health)).into_response(),
        crate::health::HealthStatus::Degraded => {
            // Service is degraded but still operational
            (StatusCode::OK, Json(health)).into_response()
        }
        crate::health::HealthStatus::Unhealthy => {
            (StatusCode::SERVICE_UNAVAILABLE, Json(health)).into_response()
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::db::DatabasePool;

    #[tokio::test]
    async fn test_liveness_handler() {
        // Create a mock database pool for testing
        let db_url = std::env::var("DATABASE_URL")
            .unwrap_or_else(|_| "postgresql://test:test@localhost:5432/test".to_string());

        // Skip test if database is not available
        if DatabasePool::new(&db_url).await.is_err() {
            return;
        }

        let db_pool = DatabasePool::new(&db_url).await.unwrap();
        let health_checker = Arc::new(HealthChecker::new(Arc::new(db_pool)));
        let state = HealthState { health_checker };

        let response = liveness_handler(State(state)).await;
        let status = response.status();

        assert_eq!(status, StatusCode::OK);
    }
}
