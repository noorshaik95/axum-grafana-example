//! TestSSOConnection client.
//!
//! **STUB — TODO**: auth-tenant-expert is building the `TestSSOConnection`
//! gRPC endpoint on user-auth-service under W14.1. Once the proto + handler
//! land, swap `TestSsoClient::test_connection` to use the real tonic client.
//!
//! The stub here intentionally returns `Ok(())` for any config containing a
//! non-empty `provider` and a truthy `config` object, and `Err` otherwise —
//! enough to exercise the happy-path and the failure-path from the Restate
//! workflow in tests without a live user-auth-service.

use crate::state::SsoConfig;
use anyhow::{anyhow, Result};
use async_trait::async_trait;

#[async_trait]
pub trait TestSsoClient: Send + Sync {
    /// Validate that the given SSO config can reach the IdP. Real impl will
    /// call `user-auth-service.AuthService/TestSSOConnection` over gRPC.
    async fn test_connection(&self, cfg: &SsoConfig) -> Result<()>;
}

/// Production stub. Replace with `GrpcTestSsoClient` once W14 lands.
pub struct StubTestSsoClient;

#[async_trait]
impl TestSsoClient for StubTestSsoClient {
    async fn test_connection(&self, cfg: &SsoConfig) -> Result<()> {
        if cfg.provider.trim().is_empty() {
            return Err(anyhow!("sso provider is required"));
        }
        if cfg.config.is_null() {
            return Err(anyhow!("sso config body is required"));
        }
        match serde_json::to_string(&cfg.config) {
            Ok(s) if s == "{}" => Err(anyhow!("sso config body is empty")),
            Ok(_) => Ok(()),
            Err(e) => Err(anyhow!("sso config not serializable: {e}")),
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use serde_json::json;

    #[tokio::test]
    async fn stub_accepts_well_formed_config() {
        let c = StubTestSsoClient;
        let cfg = SsoConfig {
            provider: "saml".into(),
            config: json!({"metadata_url": "https://idp.example/meta"}),
        };
        assert!(c.test_connection(&cfg).await.is_ok());
    }

    #[tokio::test]
    async fn stub_rejects_empty_provider() {
        let c = StubTestSsoClient;
        let cfg = SsoConfig {
            provider: "".into(),
            config: json!({"x": 1}),
        };
        assert!(c.test_connection(&cfg).await.is_err());
    }

    #[tokio::test]
    async fn stub_rejects_null_config() {
        let c = StubTestSsoClient;
        let cfg = SsoConfig {
            provider: "saml".into(),
            config: serde_json::Value::Null,
        };
        assert!(c.test_connection(&cfg).await.is_err());
    }

    #[tokio::test]
    async fn stub_rejects_empty_config_object() {
        let c = StubTestSsoClient;
        let cfg = SsoConfig {
            provider: "oidc".into(),
            config: json!({}),
        };
        assert!(c.test_connection(&cfg).await.is_err());
    }
}
