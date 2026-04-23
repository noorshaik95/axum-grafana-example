//! Canvas LMS API client for the W13.2 migration workflow.
//!
//! Handles:
//! - OAuth code → access-token exchange (`connect`)
//! - Paginated course fetching (`list_courses_page`)
//! - Paginated user/roster fetching (`list_users_page`)
//! - Rate-limit backoff driven by Canvas `X-Rate-Limit-Remaining` header.
//!
//! All network calls are wrapped behind the `CanvasApi` trait so the Restate
//! workflow can be unit-tested with a fake client (see `tests/` at bottom).

use anyhow::Result;
use async_trait::async_trait;
use reqwest::header::HeaderMap;
use serde::{Deserialize, Serialize};
use std::time::Duration;

/// Canvas considers anything below this remaining-budget "low" and we back off
/// before burning through the last slice. Value per Canvas public docs.
pub const CANVAS_LOW_RATE_LIMIT: f64 = 100.0;

/// Pause applied when `X-Rate-Limit-Remaining < CANVAS_LOW_RATE_LIMIT`.
pub const CANVAS_BACKOFF: Duration = Duration::from_secs(2);

#[derive(Debug, Serialize, Deserialize, Clone)]
pub struct CanvasCourse {
    pub id: u64,
    pub name: String,
    #[serde(default)]
    pub course_code: Option<String>,
}

#[derive(Debug, Serialize, Deserialize, Clone)]
pub struct CanvasUser {
    pub id: u64,
    pub email: Option<String>,
    #[serde(default)]
    pub name: Option<String>,
}

#[derive(Debug, Clone)]
pub struct CanvasPage<T> {
    pub items: Vec<T>,
    pub has_more: bool,
    /// Canvas returns `X-Rate-Limit-Remaining` as a float. Exposed so the
    /// caller can decide whether to back off between pages.
    pub rate_limit_remaining: Option<f64>,
}

#[async_trait]
pub trait CanvasApi: Send + Sync {
    async fn exchange_oauth_code(&self, base_url: &str, code: &str) -> Result<String>;
    async fn list_courses_page(
        &self,
        base_url: &str,
        token: &str,
        page: u32,
    ) -> Result<CanvasPage<CanvasCourse>>;
    async fn list_users_page(
        &self,
        base_url: &str,
        token: &str,
        page: u32,
    ) -> Result<CanvasPage<CanvasUser>>;
}

pub struct HttpCanvasClient {
    client: reqwest::Client,
}

impl HttpCanvasClient {
    pub fn new() -> Self {
        Self {
            client: reqwest::Client::builder()
                .timeout(Duration::from_secs(30))
                .build()
                .expect("reqwest client"),
        }
    }
}

impl Default for HttpCanvasClient {
    fn default() -> Self {
        Self::new()
    }
}

#[async_trait]
impl CanvasApi for HttpCanvasClient {
    async fn exchange_oauth_code(&self, base_url: &str, code: &str) -> Result<String> {
        let url = format!("{}/login/oauth2/token", base_url.trim_end_matches('/'));
        #[derive(Deserialize)]
        struct TokenResp {
            access_token: String,
        }
        let resp = self
            .client
            .post(url)
            .form(&[("grant_type", "authorization_code"), ("code", code)])
            .send()
            .await?
            .error_for_status()?
            .json::<TokenResp>()
            .await?;
        Ok(resp.access_token)
    }

    async fn list_courses_page(
        &self,
        base_url: &str,
        token: &str,
        page: u32,
    ) -> Result<CanvasPage<CanvasCourse>> {
        let url = format!(
            "{}/api/v1/accounts/self/courses?per_page=50&page={}",
            base_url.trim_end_matches('/'),
            page
        );
        let resp = self
            .client
            .get(url)
            .bearer_auth(token)
            .send()
            .await?
            .error_for_status()?;
        parse_page(resp).await
    }

    async fn list_users_page(
        &self,
        base_url: &str,
        token: &str,
        page: u32,
    ) -> Result<CanvasPage<CanvasUser>> {
        let url = format!(
            "{}/api/v1/accounts/self/users?per_page=100&page={}",
            base_url.trim_end_matches('/'),
            page
        );
        let resp = self
            .client
            .get(url)
            .bearer_auth(token)
            .send()
            .await?
            .error_for_status()?;
        parse_page(resp).await
    }
}

async fn parse_page<T: for<'de> Deserialize<'de>>(
    resp: reqwest::Response,
) -> Result<CanvasPage<T>> {
    let has_more = has_next_link(resp.headers());
    let rate_limit_remaining = resp
        .headers()
        .get("x-rate-limit-remaining")
        .and_then(|v| v.to_str().ok())
        .and_then(|s| s.parse::<f64>().ok());
    let items: Vec<T> = resp.json().await?;
    Ok(CanvasPage {
        items,
        has_more,
        rate_limit_remaining,
    })
}

/// Canvas paginates via RFC 5988 `Link` headers — `rel="next"` means more
/// pages remain.
pub fn has_next_link(headers: &HeaderMap) -> bool {
    headers
        .get("link")
        .and_then(|v| v.to_str().ok())
        .map(|s| s.split(',').any(|p| p.trim().ends_with(r#"rel="next""#)))
        .unwrap_or(false)
}

/// Return the backoff duration suggested by the Canvas rate-limit budget.
/// Zero means "no backoff needed"; any positive value should be awaited
/// before the next request.
pub fn backoff_for_remaining(remaining: Option<f64>) -> Duration {
    match remaining {
        Some(r) if r < CANVAS_LOW_RATE_LIMIT => CANVAS_BACKOFF,
        _ => Duration::ZERO,
    }
}

#[cfg(test)]
pub mod tests {
    use super::*;
    use async_trait::async_trait;
    use std::sync::atomic::{AtomicU32, Ordering};
    use std::sync::Arc;

    #[test]
    fn backoff_zero_when_plenty_remaining() {
        assert_eq!(backoff_for_remaining(Some(5000.0)), Duration::ZERO);
    }

    #[test]
    fn backoff_applied_when_low_remaining() {
        let d = backoff_for_remaining(Some(42.0));
        assert_eq!(d, CANVAS_BACKOFF);
    }

    #[test]
    fn backoff_zero_when_header_missing() {
        assert_eq!(backoff_for_remaining(None), Duration::ZERO);
    }

    #[test]
    fn backoff_zero_at_exact_boundary() {
        assert_eq!(
            backoff_for_remaining(Some(CANVAS_LOW_RATE_LIMIT)),
            Duration::ZERO,
            "remaining == low threshold should not backoff (use < not <=)"
        );
    }

    #[test]
    fn has_next_link_parses_multiple_rels() {
        let mut h = HeaderMap::new();
        h.insert(
            "link",
            r#"<https://canvas.example/api?page=2>; rel="next", <https://canvas.example/api?page=1>; rel="first""#
                .parse()
                .unwrap(),
        );
        assert!(has_next_link(&h));
    }

    #[test]
    fn has_next_link_false_when_last_page() {
        let mut h = HeaderMap::new();
        h.insert(
            "link",
            r#"<https://canvas.example/api?page=1>; rel="first", <https://canvas.example/api?page=5>; rel="last""#
                .parse()
                .unwrap(),
        );
        assert!(!has_next_link(&h));
    }

    // --- Fake Canvas used by workflow tests ---------------------------------

    pub struct FakeCanvas {
        pub courses: Vec<CanvasCourse>,
        pub users: Vec<CanvasUser>,
        pub per_page: usize,
        /// Rate limit remaining to report for each course page. If the ith
        /// page sees `Some(Some(low))`, the workflow should back off.
        pub course_page_rate_limits: Vec<Option<f64>>,
        pub calls: Arc<AtomicU32>,
    }

    #[async_trait]
    impl CanvasApi for FakeCanvas {
        async fn exchange_oauth_code(&self, _: &str, code: &str) -> Result<String> {
            self.calls.fetch_add(1, Ordering::SeqCst);
            if code.is_empty() {
                return Err(anyhow::anyhow!("empty code"));
            }
            Ok(format!("token-for-{code}"))
        }

        async fn list_courses_page(
            &self,
            _: &str,
            _: &str,
            page: u32,
        ) -> Result<CanvasPage<CanvasCourse>> {
            self.calls.fetch_add(1, Ordering::SeqCst);
            let start = (page as usize).saturating_sub(1) * self.per_page;
            let end = usize::min(start + self.per_page, self.courses.len());
            let items = if start < end {
                self.courses[start..end].to_vec()
            } else {
                vec![]
            };
            let has_more = end < self.courses.len();
            let rate_limit_remaining = self
                .course_page_rate_limits
                .get(page as usize - 1)
                .cloned()
                .unwrap_or(None);
            Ok(CanvasPage {
                items,
                has_more,
                rate_limit_remaining,
            })
        }

        async fn list_users_page(
            &self,
            _: &str,
            _: &str,
            page: u32,
        ) -> Result<CanvasPage<CanvasUser>> {
            self.calls.fetch_add(1, Ordering::SeqCst);
            let start = (page as usize).saturating_sub(1) * self.per_page;
            let end = usize::min(start + self.per_page, self.users.len());
            let items = if start < end {
                self.users[start..end].to_vec()
            } else {
                vec![]
            };
            let has_more = end < self.users.len();
            Ok(CanvasPage {
                items,
                has_more,
                rate_limit_remaining: Some(5000.0),
            })
        }
    }
}
