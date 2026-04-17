use axum::http::HeaderMap;
use std::collections::HashMap;
use std::sync::{Arc, RwLock};
use std::time::{Duration, Instant};
use tonic::transport::Channel;
use tracing::{debug, error, info, warn};

use crate::config::AuthConfig;

use super::auth::auth_service_client::AuthServiceClient;
use super::auth::{ValidateTokenRequest, ValidateTokenResponse};
use super::constants::{
    DEFAULT_CACHE_TTL, ERR_CACHE_CLEAR, ERR_CACHE_READ, ERR_CACHE_WRITE,
    ERR_INSUFFICIENT_PERMISSIONS_PREFIX, ERR_INVALID_ENDPOINT, ERR_INVALID_TOKEN, ERR_NO_CLAIMS,
    FALLBACK_CACHE_TTL,
};
use super::gateway::service_auth_policy_client::ServiceAuthPolicyClient;
use super::gateway::{AuthPolicyRequest, AuthPolicyResponse};
use super::types::{AuthError, AuthPolicy, AuthResult, PolicyCacheKey, TokenClaims};

/// Authorization service client with policy caching
pub struct AuthService {
    auth_client_channel: Channel,
    policy_cache: Arc<RwLock<HashMap<PolicyCacheKey, AuthPolicy>>>,
}

impl AuthService {
    /// Create a new authorization service client
    pub async fn new(config: AuthConfig) -> Result<Self, AuthError> {
        info!(
            endpoint = %config.service_endpoint,
            timeout_ms = config.timeout_ms,
            "Initializing authorization service client"
        );

        // Create channel to auth service
        let endpoint = config
            .service_endpoint
            .parse::<tonic::transport::Endpoint>()
            .map_err(|e| AuthError::ConnectionError(format!("{}: {}", ERR_INVALID_ENDPOINT, e)))?;

        let timeout = Duration::from_millis(config.timeout_ms);

        let endpoint = endpoint
            .timeout(timeout)
            .connect_timeout(Duration::from_secs(10))
            .tcp_keepalive(Some(Duration::from_secs(60)));

        let channel = endpoint.connect_lazy();

        info!("Authorization service client initialized successfully");

        Ok(Self {
            auth_client_channel: channel,
            policy_cache: Arc::new(RwLock::new(HashMap::new())),
        })
    }

    /// Extract authentication token from request headers
    pub fn extract_token(headers: &HeaderMap) -> Option<String> {
        headers
            .get("authorization")
            .and_then(|value| value.to_str().ok())
            .map(|auth_header| {
                // Support both "Bearer <token>" and raw token formats
                if let Some(token) = auth_header.strip_prefix("Bearer ") {
                    token.to_string()
                } else if let Some(token) = auth_header.strip_prefix("bearer ") {
                    token.to_string()
                } else {
                    auth_header.to_string()
                }
            })
    }

    /// Validate an authentication token via the auth service
    #[tracing::instrument(
        name = "validate_token",
        skip(self, token),
        fields(
            user.id = tracing::field::Empty,
            validation.result = tracing::field::Empty
        )
    )]
    pub async fn validate_token(&self, token: &str) -> Result<AuthResult, AuthError> {
        debug!("Validating authentication token");

        let mut client = AuthServiceClient::new(self.auth_client_channel.clone());

        let request = tonic::Request::new(ValidateTokenRequest {
            token: token.to_string(),
        });

        match client.validate_token(request).await {
            Ok(response) => {
                let response: ValidateTokenResponse = response.into_inner();

                if response.valid {
                    debug!(user_id = %response.user_id, "Token validation successful");

                    // Record user ID and validation result in span
                    tracing::Span::current().record("user.id", &response.user_id);
                    tracing::Span::current().record("validation.result", true);

                    Ok(AuthResult {
                        valid: true,
                        claims: Some(TokenClaims {
                            user_id: response.user_id,
                            roles: response.roles,
                            exp: 0, // Not provided by current proto, could be added
                        }),
                        error: None,
                    })
                } else {
                    warn!("Token validation failed: {}", response.error);

                    // Record validation failure in span
                    tracing::Span::current().record("validation.result", false);

                    Ok(AuthResult {
                        valid: false,
                        claims: None,
                        error: Some(response.error),
                    })
                }
            }
            Err(status) => {
                error!(error = %status, "Auth service call failed");

                // Record validation failure in span
                tracing::Span::current().record("validation.result", false);

                Err(AuthError::ServiceError(status.to_string()))
            }
        }
    }

    /// Get authorization policy for a service method from backend service
    pub async fn get_auth_policy(
        &self,
        service: &str,
        grpc_method: &str,
        service_channel: Channel,
    ) -> Result<AuthPolicy, AuthError> {
        let cache_key = PolicyCacheKey {
            service: service.to_string(),
            method: grpc_method.to_string(),
        };

        // Check cache first
        {
            let cache = self
                .policy_cache
                .read()
                .map_err(|e| AuthError::CacheError(format!("{}: {}", ERR_CACHE_READ, e)))?;

            if let Some(policy) = cache.get(&cache_key) {
                if policy.is_valid() {
                    debug!(
                        service = %service,
                        method = %grpc_method,
                        "Using cached authorization policy"
                    );
                    return Ok(policy.clone());
                }
            }
        }

        // Cache miss or expired - query backend service
        debug!(
            service = %service,
            method = %grpc_method,
            "Querying backend service for authorization policy"
        );

        let mut client = ServiceAuthPolicyClient::new(service_channel);

        let request = tonic::Request::new(AuthPolicyRequest {
            grpc_method: grpc_method.to_string(),
        });

        match client.get_auth_policy(request).await {
            Ok(response) => {
                let response: AuthPolicyResponse = response.into_inner();

                let cache_ttl = if response.cache_ttl_seconds > 0 {
                    Duration::from_secs(response.cache_ttl_seconds as u64)
                } else {
                    DEFAULT_CACHE_TTL
                };

                let policy = AuthPolicy {
                    service: service.to_string(),
                    method: grpc_method.to_string(),
                    require_auth: response.require_auth,
                    required_roles: response.required_roles,
                    cached_at: Instant::now(),
                    cache_ttl,
                };

                // Update cache
                {
                    let mut cache = self.policy_cache.write().map_err(|e| {
                        AuthError::CacheError(format!("{}: {}", ERR_CACHE_WRITE, e))
                    })?;
                    cache.insert(cache_key, policy.clone());
                }

                debug!(
                    service = %service,
                    method = %grpc_method,
                    require_auth = policy.require_auth,
                    required_roles = ?policy.required_roles,
                    "Authorization policy cached"
                );

                Ok(policy)
            }
            Err(status) => {
                warn!(
                    service = %service,
                    method = %grpc_method,
                    error = %status,
                    "Failed to get auth policy from backend service, defaulting to require auth"
                );

                // Fail-secure: default to requiring auth if policy query fails
                Ok(AuthPolicy {
                    service: service.to_string(),
                    method: grpc_method.to_string(),
                    require_auth: true,
                    required_roles: vec![],
                    cached_at: Instant::now(),
                    cache_ttl: FALLBACK_CACHE_TTL,
                })
            }
        }
    }

    /// Check authorization based on policy and token
    pub async fn check_authorization(
        &self,
        token: Option<&str>,
        policy: &AuthPolicy,
    ) -> Result<AuthResult, AuthError> {
        // If auth not required, allow request
        if !policy.require_auth {
            debug!(
                service = %policy.service,
                method = %policy.method,
                "Authorization not required for this endpoint"
            );
            return Ok(AuthResult {
                valid: true,
                claims: None,
                error: None,
            });
        }

        // Auth required - check for token
        let token = token.ok_or(AuthError::MissingToken)?;

        // Validate token
        let auth_result = self.validate_token(token).await?;

        if !auth_result.valid {
            return Err(AuthError::InvalidToken(
                auth_result
                    .error
                    .unwrap_or_else(|| ERR_INVALID_TOKEN.to_string()),
            ));
        }

        // Check required roles if specified
        if !policy.required_roles.is_empty() {
            if let Some(claims) = &auth_result.claims {
                let has_required_role = policy
                    .required_roles
                    .iter()
                    .any(|required_role| claims.roles.contains(required_role));

                if !has_required_role {
                    return Err(AuthError::InsufficientPermissions(format!(
                        "{}: {:?}",
                        ERR_INSUFFICIENT_PERMISSIONS_PREFIX, policy.required_roles
                    )));
                }
            } else {
                return Err(AuthError::InvalidToken(ERR_NO_CLAIMS.to_string()));
            }
        }

        debug!(
            service = %policy.service,
            method = %policy.method,
            user_id = ?auth_result.claims.as_ref().map(|c| &c.user_id),
            "Authorization check passed"
        );

        Ok(auth_result)
    }

    /// Clear the policy cache (useful for testing or manual refresh)
    #[allow(dead_code)]
    pub fn clear_policy_cache(&self) -> Result<(), AuthError> {
        let mut cache = self
            .policy_cache
            .write()
            .map_err(|e| AuthError::CacheError(format!("{}: {}", ERR_CACHE_CLEAR, e)))?;
        cache.clear();
        info!("Policy cache cleared");
        Ok(())
    }

    /// Get cache statistics
    #[allow(dead_code)]
    pub fn cache_stats(&self) -> Result<(usize, usize), AuthError> {
        let cache = self
            .policy_cache
            .read()
            .map_err(|e| AuthError::CacheError(format!("{}: {}", ERR_CACHE_READ, e)))?;

        let total = cache.len();
        let valid = cache.values().filter(|p| p.is_valid()).count();

        Ok((total, valid))
    }
}
