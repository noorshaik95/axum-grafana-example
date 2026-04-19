//! W11.3 — Zoom OAuth integration (stub).
//!
//! First-pass scaffolding per docs/plan.md W11.3. Full token exchange,
//! meeting creation, and webhook verification are left as TODO until the
//! Zoom SDK contract is ratified (blocker → SendMessage team-lead).

use chrono::{DateTime, Utc};
use serde::{Deserialize, Serialize};

/// Encrypted Zoom tokens persisted per instructor. Encryption at rest is
/// delegated to the caller (DB-layer KMS wrap); this struct only carries
/// the already-encrypted payload + refresh metadata.
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ZoomTokens {
    pub instructor_id: String,
    pub access_token_ciphertext: String,
    pub refresh_token_ciphertext: String,
    pub expires_at: DateTime<Utc>,
    pub scope: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ZoomMeeting {
    pub meeting_id: String,
    pub join_url: String,
    pub start_url: String,
    pub password: Option<String>,
}

#[derive(Debug, thiserror::Error)]
pub enum ZoomError {
    #[error("zoom integration is not yet wired — ratify SDK contract with team-lead")]
    NotImplemented,
}

/// Build the OAuth authorization URL the caller should redirect the user to.
/// Stubbed: returns a canonical Zoom OAuth URL shape so the HTTP handler can
/// be wired end-to-end before real credentials are provisioned.
pub fn build_auth_url(client_id: &str, redirect_uri: &str, state: &str) -> String {
    format!(
        "https://zoom.us/oauth/authorize?response_type=code&client_id={}&redirect_uri={}&state={}",
        client_id, redirect_uri, state
    )
}

/// Exchange the authorization code for tokens. TODO: wire reqwest call to
/// `POST https://zoom.us/oauth/token` + KMS-wrap tokens before persist.
pub async fn exchange_code(_code: &str, _instructor_id: &str) -> Result<ZoomTokens, ZoomError> {
    Err(ZoomError::NotImplemented)
}

/// Create a Zoom meeting via the Zoom REST API. TODO: POST /v2/users/me/meetings.
pub async fn create_meeting(
    _tokens: &ZoomTokens,
    _topic: &str,
    _start_time: DateTime<Utc>,
    _duration_minutes: i32,
) -> Result<ZoomMeeting, ZoomError> {
    Err(ZoomError::NotImplemented)
}

/// Verify a Zoom webhook signature. TODO: implement Zoom's HMAC-SHA256
/// signature scheme (`authorization: v0=...` header + body hash).
pub fn verify_webhook_signature(_body: &[u8], _signature_header: &str, _secret: &str) -> bool {
    false
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn auth_url_has_expected_shape() {
        let url = build_auth_url("cid", "https://app.example/cb", "xyz");
        assert!(url.starts_with("https://zoom.us/oauth/authorize"));
        assert!(url.contains("client_id=cid"));
        assert!(url.contains("state=xyz"));
    }

    #[tokio::test]
    async fn stubs_return_not_implemented() {
        let err = exchange_code("c", "i").await.unwrap_err();
        assert!(matches!(err, ZoomError::NotImplemented));
    }
}
