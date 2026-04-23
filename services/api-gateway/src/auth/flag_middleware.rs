//! Feature-flag middleware (W17.3).
//!
//! Runs **after** the auth middleware: given an `AuthContext`, fetch the
//! effective feature flags for the authenticated principal and stash them
//! on the request as a `FeatureFlags` extension so downstream handlers and
//! frontends can branch without an extra round-trip.
//!
//! Fetch strategy (first-win):
//!   1. Redis (`platform:flags:{tenant_id}:{sha1(user_id+roles)}`, TTL 5m)
//!   2. feature-flag-service gRPC EvaluateFlags (falls back here on miss)
//!   3. Empty map (if the service is unreachable — never 5xx).
//!
//! The middleware is designed to be mounted **alongside** the auth layer so
//! unauthenticated / public routes (`/api/status`, `/health`) skip the
//! lookup entirely: if no `AuthContext::authenticated` is present, we
//! insert an empty `FeatureFlags` and pass through.

use axum::{
    body::Body,
    extract::{Request, State},
    middleware::Next,
    response::Response,
};
use redis::aio::ConnectionManager;
use redis::AsyncCommands;
use serde::{Deserialize, Serialize};
use sha1::{Digest, Sha1};
use std::collections::HashMap;
use std::sync::Arc;
use tracing::{debug, warn};

use super::middleware::AuthContext;

/// Per-request bundle of `{flag_key -> enabled}` attached to the request via
/// `.extensions()`. Clone is cheap (Arc-backed) so handlers can copy it.
#[derive(Debug, Clone, Default, Serialize, Deserialize)]
pub struct FeatureFlags {
    pub flags: HashMap<String, bool>,
}

impl FeatureFlags {
    pub fn empty() -> Self {
        Self {
            flags: HashMap::new(),
        }
    }

    /// `true` when the named flag is present and set to `true`.
    pub fn is_enabled(&self, key: &str) -> bool {
        self.flags.get(key).copied().unwrap_or(false)
    }
}

/// Trait used to fetch flags from the feature-flag-service. Abstracted so
/// tests can inject a fake without spinning up a gRPC stub.
#[async_trait::async_trait]
pub trait FlagProvider: Send + Sync {
    async fn evaluate(
        &self,
        tenant_id: &str,
        user_id: &str,
        roles: &[String],
    ) -> Result<FeatureFlags, FlagProviderError>;
}

#[derive(Debug, thiserror::Error)]
pub enum FlagProviderError {
    #[error("feature-flag-service unavailable: {0}")]
    Unavailable(String),
    #[error("feature-flag-service error: {0}")]
    Remote(String),
}

/// Placeholder provider: until the feature-flag-service is scaffolded this
/// always errors with `Unavailable`, and the middleware falls back to an
/// empty flag map. Swap in the real gRPC stub later without touching the
/// middleware code.
#[derive(Default)]
pub struct UnavailableProvider;

#[async_trait::async_trait]
impl FlagProvider for UnavailableProvider {
    async fn evaluate(
        &self,
        _tenant_id: &str,
        _user_id: &str,
        _roles: &[String],
    ) -> Result<FeatureFlags, FlagProviderError> {
        Err(FlagProviderError::Unavailable(
            "feature-flag-service not scaffolded".to_string(),
        ))
    }
}

/// Default 5-minute cache window for evaluated flags.
pub const FLAG_CACHE_TTL_SECS: u64 = 300;

/// Middleware state: provider + optional Redis connection.
#[derive(Clone)]
pub struct FlagMiddlewareState {
    pub provider: Arc<dyn FlagProvider>,
    pub redis: Option<ConnectionManager>,
    pub ttl_secs: u64,
}

impl FlagMiddlewareState {
    pub fn new(provider: Arc<dyn FlagProvider>, redis: Option<ConnectionManager>) -> Self {
        Self {
            provider,
            redis,
            ttl_secs: FLAG_CACHE_TTL_SECS,
        }
    }
}

fn cache_key(tenant_id: &str, user_id: &str, roles: &[String]) -> String {
    // Normalize roles to match feature-flag-service's cache-key writer:
    // sort alphabetically and lowercase, so role ordering or casing
    // variations hash to the same key (prevents cold-start cache miss on
    // every first-touch request).
    let mut normalized: Vec<String> = roles.iter().map(|r| r.to_lowercase()).collect();
    normalized.sort();

    let mut hasher = Sha1::new();
    hasher.update(user_id.as_bytes());
    hasher.update(b"|");
    for r in &normalized {
        hasher.update(r.as_bytes());
        hasher.update(b",");
    }
    let digest = hex::encode(hasher.finalize());
    // Prefix platform: so the flag-service namespace is segregated from the
    // tenant session cache; tenant scoping is embedded in the key itself.
    format!("platform:flags:{}:{}", tenant_id, digest)
}

async fn cached_flags(
    redis: Option<&ConnectionManager>,
    key: &str,
) -> Option<FeatureFlags> {
    let mut conn = redis?.clone();
    match conn.get::<_, Option<String>>(key).await {
        Ok(Some(raw)) => match serde_json::from_str::<FeatureFlags>(&raw) {
            Ok(f) => Some(f),
            Err(e) => {
                warn!(error = %e, key = %key, "Failed to deserialise cached flags");
                None
            }
        },
        _ => None,
    }
}

async fn store_flags(
    redis: Option<&ConnectionManager>,
    key: &str,
    flags: &FeatureFlags,
    ttl_secs: u64,
) {
    let Some(redis) = redis else { return };
    let payload = match serde_json::to_string(flags) {
        Ok(s) => s,
        Err(e) => {
            warn!(error = %e, "Failed to serialise flags for cache");
            return;
        }
    };
    let mut conn = redis.clone();
    let res: Result<(), _> = conn.set_ex(key, payload, ttl_secs).await;
    if let Err(e) = res {
        debug!(error = %e, key = %key, "Flag cache write error (ignored)");
    }
}

/// Resolve the effective flags for an auth context. Public so portals that
/// need to prefetch (e.g. `TraceWidget` impersonation checks) can reuse
/// the exact same logic.
pub async fn resolve_flags(state: &FlagMiddlewareState, ctx: &AuthContext) -> FeatureFlags {
    let Some(user_id) = ctx.user_id.as_deref() else {
        return FeatureFlags::empty();
    };
    // No tenant_id plumbed through today — use "__platform__" until the
    // tenant-aware AuthContext lands (tracked in ARCHITECTURE.md §4).
    let tenant_id = ctx
        .tenant_id
        .as_deref()
        .unwrap_or(super::cache::PLATFORM_SCOPE);

    let key = cache_key(tenant_id, user_id, &ctx.roles);

    if let Some(hit) = cached_flags(state.redis.as_ref(), &key).await {
        debug!(key = %key, "Flag cache hit");
        return hit;
    }

    match state.provider.evaluate(tenant_id, user_id, &ctx.roles).await {
        Ok(flags) => {
            store_flags(state.redis.as_ref(), &key, &flags, state.ttl_secs).await;
            flags
        }
        Err(e) => {
            debug!(error = %e, "Flag evaluation unavailable — using empty map");
            FeatureFlags::empty()
        }
    }
}

/// Axum middleware: attach `FeatureFlags` to every request's extensions.
pub async fn flag_middleware(
    State(state): State<FlagMiddlewareState>,
    mut request: Request<Body>,
    next: Next,
) -> Response {
    let flags = match request.extensions().get::<AuthContext>() {
        Some(ctx) if ctx.authenticated => resolve_flags(&state, ctx).await,
        _ => FeatureFlags::empty(),
    };
    request.extensions_mut().insert(flags);
    next.run(request).await
}

#[cfg(test)]
mod tests {
    use super::*;
    use std::sync::atomic::{AtomicUsize, Ordering};

    #[derive(Default)]
    struct FakeProvider {
        calls: AtomicUsize,
        fail: bool,
    }

    #[async_trait::async_trait]
    impl FlagProvider for FakeProvider {
        async fn evaluate(
            &self,
            _tenant_id: &str,
            _user_id: &str,
            _roles: &[String],
        ) -> Result<FeatureFlags, FlagProviderError> {
            self.calls.fetch_add(1, Ordering::SeqCst);
            if self.fail {
                return Err(FlagProviderError::Unavailable("down".into()));
            }
            let mut flags = HashMap::new();
            flags.insert("ff.new-ui".into(), true);
            flags.insert("ff.legacy-grading".into(), false);
            Ok(FeatureFlags { flags })
        }
    }

    #[test]
    fn feature_flags_is_enabled_defaults_false() {
        let f = FeatureFlags::empty();
        assert!(!f.is_enabled("ff.missing"));
    }

    #[test]
    fn feature_flags_is_enabled_checks_value() {
        let mut map = HashMap::new();
        map.insert("ff.on".into(), true);
        map.insert("ff.off".into(), false);
        let f = FeatureFlags { flags: map };
        assert!(f.is_enabled("ff.on"));
        assert!(!f.is_enabled("ff.off"));
        assert!(!f.is_enabled("ff.missing"));
    }

    #[test]
    fn cache_key_stable_across_role_order() {
        // Roles are sorted + lowercased before hashing so shuffled orderings
        // produce the same key. Matches feature-flag-service's writer; a
        // divergent key here causes a cold-start miss on every first-touch
        // request until the gateway warms its own keys.
        let k1 = cache_key("eastfield", "u-1", &["a".into(), "b".into()]);
        let k2 = cache_key("eastfield", "u-1", &["b".into(), "a".into()]);
        assert_eq!(k1, k2);
    }

    #[test]
    fn cache_key_case_insensitive() {
        // Mixed-case and lowercase roles must hash to the same key.
        let k1 = cache_key("eastfield", "u-1", &["Student".into(), "Instructor".into()]);
        let k2 = cache_key("eastfield", "u-1", &["instructor".into(), "student".into()]);
        assert_eq!(k1, k2);
    }

    #[test]
    fn cache_key_distinguishes_tenant() {
        let k1 = cache_key("eastfield", "u-1", &["student".into()]);
        let k2 = cache_key("northridge", "u-1", &["student".into()]);
        assert_ne!(k1, k2);
    }

    #[tokio::test]
    async fn resolve_flags_returns_empty_when_unauthenticated() {
        let provider = Arc::new(FakeProvider::default());
        let state = FlagMiddlewareState::new(provider.clone(), None);
        let ctx = AuthContext::unauthenticated();
        let flags = resolve_flags(&state, &ctx).await;
        assert!(flags.flags.is_empty());
        assert_eq!(provider.calls.load(Ordering::SeqCst), 0);
    }

    #[tokio::test]
    async fn resolve_flags_delegates_to_provider() {
        let provider = Arc::new(FakeProvider::default());
        let state = FlagMiddlewareState::new(provider.clone(), None);
        let ctx = AuthContext {
            user_id: Some("u-1".into()),
            tenant_id: Some("eastfield".into()),
            roles: vec!["student".into()],
            authenticated: true,
        };
        let flags = resolve_flags(&state, &ctx).await;
        assert!(flags.is_enabled("ff.new-ui"));
        assert!(!flags.is_enabled("ff.legacy-grading"));
        assert_eq!(provider.calls.load(Ordering::SeqCst), 1);
    }

    #[tokio::test]
    async fn resolve_flags_falls_back_to_empty_on_provider_error() {
        let provider = Arc::new(FakeProvider {
            calls: AtomicUsize::new(0),
            fail: true,
        });
        let state = FlagMiddlewareState::new(provider.clone(), None);
        let ctx = AuthContext {
            user_id: Some("u-1".into()),
            tenant_id: Some("eastfield".into()),
            roles: vec!["student".into()],
            authenticated: true,
        };
        let flags = resolve_flags(&state, &ctx).await;
        assert!(flags.flags.is_empty());
        // Provider was attempted, but the request still completed cleanly.
        assert_eq!(provider.calls.load(Ordering::SeqCst), 1);
    }

    #[tokio::test]
    async fn unavailable_provider_always_errors() {
        let p = UnavailableProvider::default();
        let res = p.evaluate("eastfield", "u-1", &["student".into()]).await;
        assert!(matches!(res, Err(FlagProviderError::Unavailable(_))));
    }
}
