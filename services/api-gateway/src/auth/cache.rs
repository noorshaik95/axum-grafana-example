//! Redis-backed token validation cache (W17.1).
//!
//! Wraps the `validate_token` gRPC call with a short-lived Redis lookup. Each
//! successful validation is serialised and stored under a key derived from
//! `sha256(token)`, scoped by tenant (`tenant:{slug}:session:{hash}`) or
//! platform (`platform:auth:{hash}`), with a 60-second TTL.
//!
//! Design notes:
//! - Cache is **opportunistic**: any Redis error (connection failure, serde
//!   error, key miss) degrades to a gRPC round-trip, never a 5xx.
//! - Logout-style flows call `invalidate()` to remove the cached entry so a
//!   revoked token can't continue making requests for up to the TTL window.
//! - Cache hits / misses are counted via Prometheus so we can verify the
//!   ">90% warm-session hit rate" acceptance criterion.

use redis::aio::ConnectionManager;
use redis::AsyncCommands;
use serde::{Deserialize, Serialize};
use sha2::{Digest, Sha256};
use std::sync::Arc;
use std::time::Duration;
use tracing::{debug, warn};

use super::types::TokenClaims;

/// Default cache TTL for a validated token entry.
pub const TOKEN_CACHE_TTL_SECS: u64 = 60;

/// Marker for callers that don't operate in a tenant context (admin.slate.local).
pub const PLATFORM_SCOPE: &str = "__platform__";

/// Serialisable form of a validated token's auth context.
///
/// `exp` is a unix-seconds timestamp. When the cached entry's own expiry is
/// past, the cache treats it as a miss — so a compromised TTL config can't
/// leak auth beyond the token's own lifetime.
#[derive(Debug, Clone, Serialize, Deserialize, PartialEq, Eq)]
pub struct CachedAuth {
    pub user_id: String,
    pub tenant_id: Option<String>,
    pub roles: Vec<String>,
    pub exp: i64,
}

impl CachedAuth {
    pub fn from_claims(claims: &TokenClaims, tenant_id: Option<String>) -> Self {
        Self {
            user_id: claims.user_id.clone(),
            tenant_id,
            roles: claims.roles.clone(),
            exp: claims.exp,
        }
    }

    pub fn to_claims(&self) -> TokenClaims {
        TokenClaims {
            user_id: self.user_id.clone(),
            roles: self.roles.clone(),
            exp: self.exp,
        }
    }
}

/// Prometheus counters for cache hit/miss rate observation.
#[derive(Clone)]
pub struct CacheMetrics {
    pub hits: prometheus::IntCounter,
    pub misses: prometheus::IntCounter,
    pub errors: prometheus::IntCounter,
}

impl CacheMetrics {
    pub fn register(registry: &prometheus::Registry) -> Self {
        let hits = prometheus::IntCounter::new(
            "gateway_token_cache_hits_total",
            "Total successful Redis token-cache lookups",
        )
        .expect("Failed to create token_cache_hits metric");
        let misses = prometheus::IntCounter::new(
            "gateway_token_cache_misses_total",
            "Total Redis token-cache misses (fell back to gRPC)",
        )
        .expect("Failed to create token_cache_misses metric");
        let errors = prometheus::IntCounter::new(
            "gateway_token_cache_errors_total",
            "Total Redis errors treated as cache miss",
        )
        .expect("Failed to create token_cache_errors metric");

        registry
            .register(Box::new(hits.clone()))
            .expect("register token_cache_hits");
        registry
            .register(Box::new(misses.clone()))
            .expect("register token_cache_misses");
        registry
            .register(Box::new(errors.clone()))
            .expect("register token_cache_errors");

        Self {
            hits,
            misses,
            errors,
        }
    }
}

/// Redis-backed token cache.
///
/// Thread-safe and cheap to clone (`ConnectionManager` pools internally).
#[derive(Clone)]
pub struct TokenCache {
    conn: ConnectionManager,
    metrics: CacheMetrics,
    ttl_secs: u64,
}

impl TokenCache {
    /// Build a token cache from a Redis URL. Errors out only if the URL can't
    /// be parsed; a dead Redis server defers errors until first use.
    pub async fn connect(
        redis_url: &str,
        metrics: CacheMetrics,
    ) -> Result<Self, redis::RedisError> {
        let client = redis::Client::open(redis_url)?;
        let conn = ConnectionManager::new(client).await?;
        Ok(Self {
            conn,
            metrics,
            ttl_secs: TOKEN_CACHE_TTL_SECS,
        })
    }

    /// Build a cache with a custom TTL (used in tests).
    #[cfg(test)]
    pub fn with_ttl(mut self, ttl_secs: u64) -> Self {
        self.ttl_secs = ttl_secs;
        self
    }

    /// Compute the canonical Redis key for a token under a tenant scope.
    ///
    /// `PLATFORM_SCOPE` (or an empty slug) routes to `platform:auth:{hash}`
    /// so admin.slate.local sessions are isolated from tenant sessions.
    pub fn key_for(scope: &str, token: &str) -> String {
        let hash = Self::hash_token(token);
        if scope.is_empty() || scope == PLATFORM_SCOPE {
            format!("platform:auth:{}", hash)
        } else {
            format!("tenant:{}:session:{}", scope, hash)
        }
    }

    fn hash_token(token: &str) -> String {
        let mut hasher = Sha256::new();
        hasher.update(token.as_bytes());
        hex::encode(hasher.finalize())
    }

    /// Look up a cached auth record. Returns `None` on miss OR any Redis
    /// error (with appropriate counter bumped). Never propagates errors up.
    pub async fn get(&self, scope: &str, token: &str) -> Option<CachedAuth> {
        let key = Self::key_for(scope, token);
        let mut conn = self.conn.clone();
        match conn.get::<_, Option<String>>(&key).await {
            Ok(Some(raw)) => match serde_json::from_str::<CachedAuth>(&raw) {
                Ok(entry) => {
                    // Defensive: evict if the cached entry claims to already
                    // be expired per the token's own `exp`.
                    if entry.exp > 0 && entry.exp < chrono::Utc::now().timestamp() {
                        debug!(key = %key, "Cached token entry past its own exp, evicting");
                        let _: Result<(), _> = conn.del(&key).await;
                        self.metrics.misses.inc();
                        return None;
                    }
                    self.metrics.hits.inc();
                    debug!(key = %key, user_id = %entry.user_id, "Token cache hit");
                    Some(entry)
                }
                Err(e) => {
                    warn!(error = %e, key = %key, "Failed to deserialise cached token entry");
                    self.metrics.errors.inc();
                    None
                }
            },
            Ok(None) => {
                self.metrics.misses.inc();
                None
            }
            Err(e) => {
                debug!(error = %e, key = %key, "Token cache lookup error (treating as miss)");
                self.metrics.errors.inc();
                None
            }
        }
    }

    /// Store a validated auth record under the given scope. Errors are
    /// logged but never surfaced — a failing cache must not break auth.
    pub async fn put(&self, scope: &str, token: &str, entry: &CachedAuth) {
        let key = Self::key_for(scope, token);
        let payload = match serde_json::to_string(entry) {
            Ok(s) => s,
            Err(e) => {
                warn!(error = %e, "Failed to serialise auth entry for cache");
                self.metrics.errors.inc();
                return;
            }
        };

        let mut conn = self.conn.clone();
        let res: Result<(), _> = conn.set_ex(&key, payload, self.ttl_secs).await;
        if let Err(e) = res {
            debug!(error = %e, key = %key, "Token cache write error (ignored)");
            self.metrics.errors.inc();
        }
    }

    /// Delete a cached entry — call this on explicit logout to cut the
    /// revocation lag from TTL down to zero.
    pub async fn invalidate(&self, scope: &str, token: &str) {
        let key = Self::key_for(scope, token);
        let mut conn = self.conn.clone();
        let res: Result<(), _> = conn.del(&key).await;
        if let Err(e) = res {
            debug!(error = %e, key = %key, "Token cache invalidate error (ignored)");
            self.metrics.errors.inc();
        }
    }

    /// Effective TTL used for `put` (exposed for diagnostics and tests).
    pub fn ttl(&self) -> Duration {
        Duration::from_secs(self.ttl_secs)
    }
}

/// Optional handle carried in `AppState`. Callers use `as_ref()` to decide
/// whether caching is enabled at runtime.
pub type SharedTokenCache = Option<Arc<TokenCache>>;

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn platform_scope_routes_to_platform_namespace() {
        let k = TokenCache::key_for(PLATFORM_SCOPE, "abc123");
        assert!(k.starts_with("platform:auth:"));
        assert_eq!(k.len(), "platform:auth:".len() + 64);
    }

    #[test]
    fn empty_scope_also_routes_to_platform() {
        let k = TokenCache::key_for("", "abc123");
        assert!(k.starts_with("platform:auth:"));
    }

    #[test]
    fn tenant_scope_routes_to_tenant_namespace() {
        let k = TokenCache::key_for("eastfield", "abc123");
        assert!(k.starts_with("tenant:eastfield:session:"));
        assert_eq!(k.len(), "tenant:eastfield:session:".len() + 64);
    }

    #[test]
    fn hashed_token_is_deterministic() {
        let a = TokenCache::key_for("eastfield", "token-xyz");
        let b = TokenCache::key_for("eastfield", "token-xyz");
        assert_eq!(a, b);
    }

    #[test]
    fn hashed_token_distinguishes_tokens() {
        let a = TokenCache::key_for("eastfield", "token-1");
        let b = TokenCache::key_for("eastfield", "token-2");
        assert_ne!(a, b);
    }

    #[test]
    fn hashed_token_distinguishes_scopes() {
        let a = TokenCache::key_for("eastfield", "same");
        let b = TokenCache::key_for("northridge", "same");
        assert_ne!(a, b);
    }

    #[test]
    fn cached_auth_round_trips_through_json() {
        let e = CachedAuth {
            user_id: "u-1".into(),
            tenant_id: Some("eastfield".into()),
            roles: vec!["student".into()],
            exp: 9_999_999_999,
        };
        let s = serde_json::to_string(&e).unwrap();
        let d: CachedAuth = serde_json::from_str(&s).unwrap();
        assert_eq!(e, d);
    }

    #[test]
    fn cached_auth_claims_conversion() {
        let claims = TokenClaims {
            user_id: "u-2".into(),
            roles: vec!["admin".into()],
            exp: 123,
        };
        let cached = CachedAuth::from_claims(&claims, Some("eastfield".into()));
        assert_eq!(cached.user_id, "u-2");
        assert_eq!(cached.tenant_id.as_deref(), Some("eastfield"));
        let back = cached.to_claims();
        assert_eq!(back.user_id, "u-2");
        assert_eq!(back.roles, vec!["admin".to_string()]);
    }
}
