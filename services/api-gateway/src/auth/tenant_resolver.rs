//! Slug → tenant_id resolver for Host-subdomain-scoped requests (#64).
//!
//! Traefik routes `{slug}.slate.local` to the api-gateway (see
//! `config/traefik/dynamic/*.yml`). The gateway derives `tenant_id` from
//! the subdomain here so downstream services get a populated
//! `AuthContext.tenant_id` without the FE/JWT carrying it.
//!
//! Lookup is lazy + in-process-cached: on first hit for a slug the resolver
//! calls tenant-service's `/tenants?search=<slug>` REST endpoint, filters
//! for an exact slug match, and caches the resulting UUID. Tenant slugs are
//! immutable post-creation, so no TTL is applied. A negative-result entry
//! is also cached (for at most `NEG_CACHE_TTL`) so a misspelled or
//! unprovisioned subdomain doesn't pound the tenant-service on every
//! request.

use std::collections::HashMap;
use std::sync::Arc;
use std::time::{Duration, Instant};
use tokio::sync::RwLock;
use tracing::{debug, warn};

/// Entries for missing tenants expire after this interval. Positive entries
/// are held for the lifetime of the process (slugs are immutable).
const NEG_CACHE_TTL: Duration = Duration::from_secs(60);

/// Reserved subdomains that are not tenant scope: the platform API host
/// and common dev aliases. A request with one of these as the leftmost
/// label is treated as platform-scope (tenant_id = None).
const RESERVED_SUBDOMAINS: &[&str] = &["api", "admin", "app", "teach", "traefik", "localhost"];

/// Slug → tenant UUID resolver with in-process caching.
#[derive(Clone)]
pub struct TenantResolver {
    http: reqwest::Client,
    base_url: String,
    positive: Arc<RwLock<HashMap<String, String>>>,
    negative: Arc<RwLock<HashMap<String, Instant>>>,
}

impl TenantResolver {
    /// Build a resolver pointing at tenant-service's REST surface.
    /// `base_url` is typically `http://tenant-service:8083`.
    pub fn new(base_url: impl Into<String>) -> Self {
        Self {
            http: reqwest::Client::builder()
                .timeout(Duration::from_secs(2))
                .build()
                .expect("reqwest client build"),
            base_url: base_url.into().trim_end_matches('/').to_string(),
            positive: Arc::new(RwLock::new(HashMap::new())),
            negative: Arc::new(RwLock::new(HashMap::new())),
        }
    }

    /// Extract the tenant-scope slug from an HTTP `Host` header value.
    ///
    /// Returns `None` for the platform host, reserved subdomains, IP
    /// literals, or any non-`.slate.local` host. Case is lowered. A
    /// port suffix (`:8080`) is stripped.
    pub fn slug_from_host(host: &str) -> Option<String> {
        let host = host.trim().to_ascii_lowercase();
        // Strip port (host header shape: "subdomain.slate.local:8080").
        let host = host.split(':').next().unwrap_or(&host);
        // Must end with ".slate.local" to be considered a tenant host.
        let prefix = host.strip_suffix(".slate.local")?;
        if prefix.is_empty() {
            return None;
        }
        // Reject further-nested subdomains — only the leftmost label matters.
        // For `foo.bar.slate.local`, the slug is the full "foo.bar"; we
        // reject that because tenant slugs are a single DNS label.
        if prefix.contains('.') {
            return None;
        }
        if RESERVED_SUBDOMAINS.contains(&prefix) {
            return None;
        }
        Some(prefix.to_string())
    }

    /// Resolve a slug to a tenant UUID, hitting tenant-service on cache miss.
    /// Returns `None` for unknown slugs; a negative entry is cached briefly.
    pub async fn resolve(&self, slug: &str) -> Option<String> {
        // Fast path: positive cache.
        if let Some(id) = self.positive.read().await.get(slug).cloned() {
            return Some(id);
        }
        // Negative cache — suppress repeated lookups for bogus slugs.
        if let Some(at) = self.negative.read().await.get(slug).copied() {
            if at.elapsed() < NEG_CACHE_TTL {
                return None;
            }
        }

        match self.fetch_from_service(slug).await {
            Ok(Some(id)) => {
                self.positive
                    .write()
                    .await
                    .insert(slug.to_string(), id.clone());
                debug!(slug = %slug, tenant_id = %id, "resolved tenant via subdomain");
                Some(id)
            }
            Ok(None) => {
                self.negative
                    .write()
                    .await
                    .insert(slug.to_string(), Instant::now());
                None
            }
            Err(e) => {
                warn!(slug = %slug, error = %e, "tenant resolver lookup failed");
                // Don't cache on transport errors — let the next request retry.
                None
            }
        }
    }

    async fn fetch_from_service(&self, slug: &str) -> reqwest::Result<Option<String>> {
        #[derive(serde::Deserialize)]
        struct ListResp {
            tenants: Option<Vec<TenantRow>>,
        }
        #[derive(serde::Deserialize)]
        struct TenantRow {
            id: String,
            slug: String,
        }

        let url = format!("{}/tenants?search={}&pageSize=50", self.base_url, slug);
        let resp = self.http.get(&url).send().await?.error_for_status()?;
        let body: ListResp = resp.json().await?;
        Ok(body
            .tenants
            .unwrap_or_default()
            .into_iter()
            .find(|t| t.slug == slug)
            .map(|t| t.id))
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn slug_from_host_strips_slate_local() {
        assert_eq!(
            TenantResolver::slug_from_host("eastfield.slate.local"),
            Some("eastfield".to_string())
        );
    }

    #[test]
    fn slug_from_host_is_case_insensitive() {
        assert_eq!(
            TenantResolver::slug_from_host("Eastfield.SLATE.local"),
            Some("eastfield".to_string())
        );
    }

    #[test]
    fn slug_from_host_strips_port() {
        assert_eq!(
            TenantResolver::slug_from_host("eastfield.slate.local:8080"),
            Some("eastfield".to_string())
        );
    }

    #[test]
    fn slug_from_host_rejects_platform_host() {
        assert_eq!(TenantResolver::slug_from_host("api.slate.local"), None);
    }

    #[test]
    fn slug_from_host_rejects_fe_subdomains() {
        for h in ["admin.slate.local", "app.slate.local", "teach.slate.local"] {
            assert_eq!(TenantResolver::slug_from_host(h), None, "host: {}", h);
        }
    }

    #[test]
    fn slug_from_host_rejects_non_slate_host() {
        assert_eq!(TenantResolver::slug_from_host("example.com"), None);
        assert_eq!(TenantResolver::slug_from_host("localhost"), None);
        assert_eq!(TenantResolver::slug_from_host("slate.local"), None);
    }

    #[test]
    fn slug_from_host_rejects_nested_subdomain() {
        // Tenant slugs are a single DNS label; multi-label prefixes are
        // rejected to avoid ambiguous routing.
        assert_eq!(
            TenantResolver::slug_from_host("foo.bar.slate.local"),
            None
        );
    }

    #[test]
    fn slug_from_host_handles_empty_and_whitespace() {
        assert_eq!(TenantResolver::slug_from_host(""), None);
        assert_eq!(TenantResolver::slug_from_host("   "), None);
    }
}
