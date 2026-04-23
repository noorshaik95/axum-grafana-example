//! W17 acceptance tests: route registration + tenant routing + flag middleware.
//!
//! These tests intentionally poke the public surface (no private internals) so
//! they continue to verify the documented contract even if the internals are
//! refactored.

use api_gateway::auth::flag_middleware::{
    FeatureFlags, FlagMiddlewareState, FlagProvider, FlagProviderError, UnavailableProvider,
};
use api_gateway::auth::middleware::AuthContext;
use api_gateway::config::GatewayConfig;
use api_gateway::handlers::gateway::routing::{
    gateway_service_name_for_key, TenantRoutingTable, TenantServiceMap, TENANT_SERVICE_KEYS,
};
use std::collections::HashMap;
use std::sync::Arc;

// ---------------------------------------------------------------------------
// Route registration — verify every route from docs/plan.md summary is present
// ---------------------------------------------------------------------------

fn load_gateway_config() -> GatewayConfig {
    let path = std::env::current_dir()
        .unwrap()
        .ancestors()
        .find_map(|p| {
            let candidate = p.join("config/gateway-config.yaml");
            if candidate.exists() {
                Some(candidate)
            } else {
                None
            }
        })
        .expect("locate config/gateway-config.yaml from any ancestor");
    GatewayConfig::load_config(path.to_str().unwrap()).expect("load gateway-config.yaml")
}

#[test]
fn w17_config_registers_wave_3_platform_services() {
    let cfg = load_gateway_config();
    // The four new platform services must be declared so the gRPC pool boots.
    for name in [
        "admin-auth-service",
        "incident-service",
        "feature-flag-service",
        "onboarding-service",
    ] {
        assert!(
            cfg.services.contains_key(name),
            "missing platform service: {name}"
        );
    }
}

#[test]
fn w17_config_registers_per_tenant_services() {
    let cfg = load_gateway_config();
    for name in ["discussion-service", "scheduling-service", "ai-service"] {
        assert!(
            cfg.services.contains_key(name),
            "missing per-tenant service: {name}"
        );
    }
}

#[test]
fn w17_route_overrides_cover_summary_table() {
    let cfg = load_gateway_config();
    let paths: Vec<&str> = cfg
        .route_overrides
        .iter()
        .filter_map(|o| o.http_path.as_deref())
        .collect();

    // Spot-check the distinctive routes from docs/plan.md §Cross-cutting Concerns.
    for expected in [
        "/api/admin/auth/login",
        "/api/admin/auth/logout",
        "/api/admin/users",
        "/api/admin/impersonate/:id",
        "/api/incidents",
        "/api/incidents/:id",
        "/api/incidents/:id/events",
        "/api/flags",
        "/api/flags/:key",
        "/api/status",
        // §1b: legacy /api/onboarding and /api/onboarding/canvas-import
        // were removed when onboarding migrated to Restate Virtual Objects.
        // Restate handlers are keyed by :id; see /api/onboarding/:id/start
        // and /api/onboarding/:id/canvas/* for the current shape.
        "/api/onboarding/:id/start",
        "/api/onboarding/:id/canvas/import-courses",
        "/api/broadcast",
        "/api/discussions/threads",
        "/api/discussions/inbox",
        "/api/scheduling/slots",
        "/api/scheduling/bookings",
        "/api/scheduling/instructor-day",
        "/api/ai/welcome",
        "/api/ai/cmd-palette",
        "/api/ai/study-plan",
        "/api/ai/grade-projection",
    ] {
        assert!(
            paths.contains(&expected),
            "missing route registration for: {expected}"
        );
    }
}

#[test]
fn w17_public_routes_include_status_and_admin_login() {
    let cfg = load_gateway_config();
    let public_paths: Vec<(&str, &str)> = cfg
        .auth
        .public_routes
        .iter()
        .map(|r| (r.path.as_str(), r.method.as_str()))
        .collect();
    assert!(public_paths.contains(&("/api/status", "GET")));
    assert!(public_paths.contains(&("/api/admin/auth/login", "POST")));
}

// ---------------------------------------------------------------------------
// W17.4 tenant routing table
// ---------------------------------------------------------------------------

fn sample_tenant(slug: &str, keys: &[&str]) -> TenantServiceMap {
    let mut m = HashMap::new();
    for k in keys {
        // Mirror the ARCHITECTURE.md §4 yaml shape.
        let port = match *k {
            "user-auth" => 50051,
            "course" => 50052,
            "assignment" => 50053,
            "content" => 50054,
            "video" => 50055,
            "discussion" => 50056,
            "scheduling" => 50063,
            "ai" => 50064,
            _ => 50099,
        };
        m.insert((*k).to_string(), format!("http://{k}-{slug}:{port}"));
    }
    TenantServiceMap::new(m)
}

#[test]
fn tenant_routing_resolves_all_eight_keys_for_fully_provisioned_tenant() {
    let table = TenantRoutingTable::empty();
    table.upsert("eastfield", sample_tenant("eastfield", TENANT_SERVICE_KEYS));
    for key in TENANT_SERVICE_KEYS {
        let ep = table
            .resolve("eastfield", key)
            .unwrap_or_else(|| panic!("missing endpoint for {key}"));
        assert!(ep.contains(&format!("{key}-eastfield")));
    }
}

#[test]
fn tenant_routing_handles_legacy_tenant_gracefully() {
    let table = TenantRoutingTable::empty();
    // Legacy tenant: only the first 5 original keys.
    table.upsert(
        "northridge",
        sample_tenant(
            "northridge",
            &["user-auth", "course", "assignment", "content", "video"],
        ),
    );
    // New keys return None — caller maps that to 503 rather than panic.
    assert!(table.resolve("northridge", "discussion").is_none());
    assert!(table.resolve("northridge", "scheduling").is_none());
    assert!(table.resolve("northridge", "ai").is_none());
}

#[test]
fn tenant_routing_upsert_is_idempotent() {
    let table = TenantRoutingTable::empty();
    table.upsert("eastfield", sample_tenant("eastfield", TENANT_SERVICE_KEYS));
    table.upsert("eastfield", sample_tenant("eastfield", TENANT_SERVICE_KEYS));
    assert_eq!(table.tenant_count(), 1);
}

#[test]
fn service_key_mapping_matches_gateway_service_names() {
    // Sanity: mapping layer returns the config-file names the gRPC pool
    // registers. Changing these two sides independently breaks routing.
    assert_eq!(gateway_service_name_for_key("discussion"), "discussion-service");
    assert_eq!(gateway_service_name_for_key("scheduling"), "scheduling-service");
    assert_eq!(gateway_service_name_for_key("ai"), "ai-service");
    assert_eq!(gateway_service_name_for_key("user-auth"), "user-auth-service");
    assert_eq!(gateway_service_name_for_key("content"), "content-management-service");
}

// ---------------------------------------------------------------------------
// Flag middleware — fallback + provider path
// ---------------------------------------------------------------------------

struct AlwaysFailProvider;

#[async_trait::async_trait]
impl FlagProvider for AlwaysFailProvider {
    async fn evaluate(
        &self,
        _tenant_id: &str,
        _user_id: &str,
        _roles: &[String],
    ) -> Result<FeatureFlags, FlagProviderError> {
        Err(FlagProviderError::Unavailable("test".into()))
    }
}

struct FixedProvider(FeatureFlags);

#[async_trait::async_trait]
impl FlagProvider for FixedProvider {
    async fn evaluate(
        &self,
        _tenant_id: &str,
        _user_id: &str,
        _roles: &[String],
    ) -> Result<FeatureFlags, FlagProviderError> {
        Ok(self.0.clone())
    }
}

fn authed(tenant: Option<&str>) -> AuthContext {
    AuthContext {
        user_id: Some("u-1".into()),
        tenant_id: tenant.map(|s| s.to_string()),
        roles: vec!["student".into()],
        authenticated: true,
    }
}

#[tokio::test]
async fn flag_middleware_returns_empty_when_provider_down() {
    let state = FlagMiddlewareState::new(Arc::new(AlwaysFailProvider), None);
    let ctx = authed(Some("eastfield"));
    let flags = api_gateway::auth::flag_middleware::resolve_flags(&state, &ctx).await;
    assert!(flags.flags.is_empty());
    // No panic, no error — "graceful degradation" acceptance criterion.
}

#[tokio::test]
async fn flag_middleware_returns_empty_for_unauthenticated() {
    let state = FlagMiddlewareState::new(Arc::new(UnavailableProvider::default()), None);
    let ctx = AuthContext::unauthenticated();
    let flags = api_gateway::auth::flag_middleware::resolve_flags(&state, &ctx).await;
    assert!(flags.flags.is_empty());
}

#[tokio::test]
async fn flag_middleware_returns_provider_flags_when_healthy() {
    let mut map = HashMap::new();
    map.insert("ff.new-grading".to_string(), true);
    map.insert("ff.legacy-banner".to_string(), false);
    let state = FlagMiddlewareState::new(
        Arc::new(FixedProvider(FeatureFlags { flags: map })),
        None,
    );
    let ctx = authed(Some("eastfield"));
    let flags = api_gateway::auth::flag_middleware::resolve_flags(&state, &ctx).await;
    assert!(flags.is_enabled("ff.new-grading"));
    assert!(!flags.is_enabled("ff.legacy-banner"));
    assert!(!flags.is_enabled("ff.unknown"));
}

#[tokio::test]
async fn unavailable_provider_surface_is_stable() {
    // Document the default provider: always errors so the middleware falls
    // back to an empty map. This is the expected Wave-2 boot state.
    let p = UnavailableProvider::default();
    let err = p
        .evaluate("eastfield", "u-1", &["student".into()])
        .await
        .expect_err("should error by design");
    assert!(matches!(err, FlagProviderError::Unavailable(_)));
}

// ---------------------------------------------------------------------------
// Token cache key shape — verify namespace separation without Redis
// ---------------------------------------------------------------------------

#[test]
fn token_cache_key_separates_platform_and_tenant_scopes() {
    use api_gateway::auth::cache::{TokenCache, PLATFORM_SCOPE};
    let platform = TokenCache::key_for(PLATFORM_SCOPE, "same-token");
    let tenant = TokenCache::key_for("eastfield", "same-token");
    assert!(platform.starts_with("platform:auth:"));
    assert!(tenant.starts_with("tenant:eastfield:session:"));
    assert_ne!(platform, tenant);
}

#[test]
fn token_cache_key_hashes_token() {
    use api_gateway::auth::cache::TokenCache;
    // Raw tokens never appear in keys; only their sha256.
    let key = TokenCache::key_for("eastfield", "super-secret-jwt");
    assert!(!key.contains("super-secret-jwt"));
    assert_eq!(
        key.len(),
        "tenant:eastfield:session:".len() + 64,
        "sha256 hex is 64 chars"
    );
}
