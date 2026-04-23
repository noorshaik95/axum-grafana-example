use axum::{
    body::Body,
    extract::{Request, State},
    http::{HeaderMap, StatusCode},
    middleware::Next,
    response::{IntoResponse, Response},
    Json,
};
use serde::Serialize;
use std::sync::Arc;
use tracing::{debug, error, warn};

use crate::auth::{AuthError, AuthResult, AuthService};
use crate::grpc::client::GrpcClientPool;
use crate::router::RequestRouter;
use tokio::sync::RwLock;

/// Extension type to pass auth context to downstream handlers
#[derive(Debug, Clone)]
pub struct AuthContext {
    pub user_id: Option<String>,
    /// Optional tenant scope. Platform-scope admin tokens leave this None;
    /// tenant-scope tokens set it so downstream services can read the
    /// authenticated caller's tenant without re-parsing the JWT.
    pub tenant_id: Option<String>,
    pub roles: Vec<String>,
    pub authenticated: bool,
}

impl AuthContext {
    /// Create an unauthenticated context
    pub fn unauthenticated() -> Self {
        Self {
            user_id: None,
            tenant_id: None,
            roles: vec![],
            authenticated: false,
        }
    }

    /// Create an authenticated context from auth result
    pub fn from_auth_result(result: &AuthResult) -> Self {
        if let Some(claims) = &result.claims {
            Self {
                user_id: Some(claims.user_id.clone()),
                // TokenClaims doesn't carry tenant_id yet; left None until
                // the tenant-aware claim parsing lands. flag_middleware and
                // conversion.rs treat None as "platform scope".
                tenant_id: None,
                roles: claims.roles.clone(),
                authenticated: true,
            }
        } else {
            Self::unauthenticated()
        }
    }
}

/// State required for authorization middleware
#[derive(Clone)]
pub struct AuthMiddlewareState {
    pub auth_service: Arc<AuthService>,
    pub grpc_pool: Arc<GrpcClientPool>,
    pub router_lock: Arc<RwLock<RequestRouter>>,
    pub public_routes: Vec<(String, String)>, // (path, method) tuples
    pub auth_failure_counter: prometheus::IntCounter,
}

/// Check if path should skip authentication.
fn should_skip_auth(path: &str, method: &str, public_routes: &[(String, String)]) -> bool {
    // System endpoints
    if path == "/health" || path == "/metrics" {
        return true;
    }

    // Public routes
    public_routes.iter().any(|(p, m)| p == path && m == method)
}

/// Route the request and get routing decision.
async fn get_routing_decision(
    router_lock: &Arc<RwLock<RequestRouter>>,
    path: &str,
    method: &str,
) -> Result<crate::router::RoutingDecision, AuthMiddlewareResponse> {
    let router_guard = router_lock.read().await;
    let routing_decision = router_guard.route(path, method).map_err(|e| {
        warn!(
            path = %path,
            method = %method,
            error = %e,
            "Route not found"
        );
        AuthMiddlewareResponse::NotFound
    })?;

    debug!(
        service = %routing_decision.service,
        grpc_method = %routing_decision.grpc_method,
        "Request routed to backend service"
    );

    Ok(routing_decision)
}

/// Get auth policy from backend service.
async fn get_auth_policy(
    auth_service: &AuthService,
    grpc_pool: &GrpcClientPool,
    routing_decision: &crate::router::RoutingDecision,
) -> Result<crate::auth::types::AuthPolicy, AuthMiddlewareResponse> {
    let service_channel = grpc_pool
        .get_channel(&routing_decision.service)
        .map_err(|e| {
            error!(
                service = %routing_decision.service,
                error = %e,
                "Failed to get service channel"
            );
            AuthMiddlewareResponse::ServiceUnavailable
        })?;

    let auth_policy = auth_service
        .get_auth_policy(
            &routing_decision.service,
            &routing_decision.grpc_method,
            service_channel,
        )
        .await
        .map_err(|e| {
            error!(
                service = %routing_decision.service,
                method = %routing_decision.grpc_method,
                error = %e,
                "Failed to get auth policy"
            );
            AuthMiddlewareResponse::ServiceUnavailable
        })?;

    debug!(
        service = %routing_decision.service,
        method = %routing_decision.grpc_method,
        require_auth = auth_policy.require_auth,
        required_roles = ?auth_policy.required_roles,
        "Auth policy retrieved"
    );

    Ok(auth_policy)
}

/// Perform authorization check.
async fn perform_authorization(
    auth_service: &AuthService,
    headers: &HeaderMap,
    auth_policy: &crate::auth::types::AuthPolicy,
    routing_decision: &crate::router::RoutingDecision,
    auth_failure_counter: &prometheus::IntCounter,
) -> Result<AuthContext, AuthMiddlewareResponse> {
    let token = AuthService::extract_token(headers);

    let auth_result = auth_service
        .check_authorization(token.as_deref(), auth_policy)
        .await
        .map_err(|e| {
            // Increment auth failure counter
            auth_failure_counter.inc();
            
            warn!(
                service = %routing_decision.service,
                method = %routing_decision.grpc_method,
                error = %e,
                "Authorization check failed"
            );
            map_auth_error_to_response(e)
        })?;

    let auth_ctx = AuthContext::from_auth_result(&auth_result);

    debug!(
        authenticated = auth_ctx.authenticated,
        user_id = ?auth_ctx.user_id,
        roles = ?auth_ctx.roles,
        "Authorization successful"
    );

    Ok(auth_ctx)
}

/// Authorization middleware that enforces dynamic auth policies
///
/// This middleware:
/// 1. Routes the request to determine target service and method
/// 2. Queries the backend service for its auth policy
/// 3. If auth required, extracts and validates the token
/// 4. Checks user roles against required roles
/// 5. Passes auth context to downstream handlers
pub async fn auth_middleware(
    State(state): State<AuthMiddlewareState>,
    headers: HeaderMap,
    mut request: Request<Body>,
    next: Next,
) -> Result<Response, AuthMiddlewareResponse> {
    let path = request.uri().path().to_string();
    let method = request.method().as_str().to_string();

    debug!(
        path = %path,
        method = %method,
        "Processing authorization middleware"
    );

    // Check if we should skip auth
    if should_skip_auth(&path, &method, &state.public_routes) {
        debug!(path = %path, "Skipping authentication");
        request
            .extensions_mut()
            .insert(AuthContext::unauthenticated());
        return Ok(next.run(request).await);
    }

    // Route the request
    let routing_decision = get_routing_decision(&state.router_lock, &path, &method).await?;

    // Store routing decision for reuse in gateway handler
    request
        .extensions_mut()
        .insert(Arc::new(routing_decision.clone()));

    // Get auth policy from backend
    let auth_policy =
        get_auth_policy(&state.auth_service, &state.grpc_pool, &routing_decision).await?;

    // Perform authorization
    let auth_ctx = perform_authorization(
        &state.auth_service,
        &headers,
        &auth_policy,
        &routing_decision,
        &state.auth_failure_counter,
    )
    .await?;

    // Insert auth context and continue
    request.extensions_mut().insert(auth_ctx);
    Ok(next.run(request).await)
}

/// Map AuthError to appropriate HTTP response
pub fn map_auth_error_to_response(error: AuthError) -> AuthMiddlewareResponse {
    match error {
        AuthError::MissingToken => {
            debug!("Authorization failed: missing token");
            AuthMiddlewareResponse::Unauthorized("Missing authentication token".to_string())
        }
        AuthError::InvalidToken(msg) => {
            debug!(error = %msg, "Authorization failed: invalid token");
            AuthMiddlewareResponse::Forbidden(format!("Invalid token: {}", msg))
        }
        AuthError::ExpiredToken => {
            debug!("Authorization failed: expired token");
            AuthMiddlewareResponse::Unauthorized("Token has expired".to_string())
        }
        AuthError::InsufficientPermissions(msg) => {
            debug!(error = %msg, "Authorization failed: insufficient permissions");
            AuthMiddlewareResponse::Forbidden(msg)
        }
        AuthError::ServiceError(msg) => {
            error!(error = %msg, "Auth service error");
            AuthMiddlewareResponse::ServiceUnavailable
        }
        AuthError::ConnectionError(msg) => {
            error!(error = %msg, "Auth service connection error");
            AuthMiddlewareResponse::ServiceUnavailable
        }
        AuthError::CacheError(msg) => {
            error!(error = %msg, "Policy cache error");
            AuthMiddlewareResponse::InternalError
        }
    }
}

/// Response types for authorization middleware errors
#[derive(Debug)]
pub enum AuthMiddlewareResponse {
    Unauthorized(String),
    Forbidden(String),
    NotFound,
    ServiceUnavailable,
    InternalError,
}

/// Error response body structure
#[derive(Debug, Serialize)]
struct ErrorResponse {
    error: ErrorDetail,
}

#[derive(Debug, Serialize)]
struct ErrorDetail {
    message: String,
    status: u16,
}

impl IntoResponse for AuthMiddlewareResponse {
    fn into_response(self) -> Response {
        let (status, message) = match self {
            AuthMiddlewareResponse::Unauthorized(msg) => (StatusCode::UNAUTHORIZED, msg),
            AuthMiddlewareResponse::Forbidden(msg) => (StatusCode::FORBIDDEN, msg),
            AuthMiddlewareResponse::NotFound => {
                (StatusCode::NOT_FOUND, "Route not found".to_string())
            }
            AuthMiddlewareResponse::ServiceUnavailable => (
                StatusCode::SERVICE_UNAVAILABLE,
                "Service temporarily unavailable".to_string(),
            ),
            AuthMiddlewareResponse::InternalError => (
                StatusCode::INTERNAL_SERVER_ERROR,
                "Internal server error".to_string(),
            ),
        };

        let body = ErrorResponse {
            error: ErrorDetail {
                message,
                status: status.as_u16(),
            },
        };

        (status, Json(body)).into_response()
    }
}
