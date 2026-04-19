use async_trait::async_trait;
use dashmap::DashMap;
use std::sync::Arc;

/// Per-session lecture state tracked alongside attendance.
#[derive(Debug, Clone)]
pub struct LectureState {
    pub session_id: String,
    pub tenant_slug: String,
    pub enrolled_count: i32,
    pub started_at_unix: i64,
    pub active: bool,
}

/// Redis sorted-set style attendance store. Keys are formatted as
/// `tenant:{slug}:live:{session_id}`; each member is a user_id scored by
/// the unix-seconds `join_timestamp`. TTL (Redis expiry) is the
/// caller's responsibility — the trait just enforces ordering semantics.
#[async_trait]
pub trait AttendanceStore: Send + Sync {
    async fn init_session(&self, state: LectureState);
    async fn end_session(&self, tenant_slug: &str, session_id: &str);
    async fn record_join(
        &self,
        tenant_slug: &str,
        session_id: &str,
        user_id: &str,
        join_timestamp: i64,
    );
    async fn attendee_count(&self, tenant_slug: &str, session_id: &str) -> i32;
    async fn get_state(&self, tenant_slug: &str, session_id: &str) -> Option<LectureState>;
}

/// In-memory implementation used in tests (and acceptable for single-replica
/// dev). Maintains sorted-set semantics (duplicates don't inflate the count —
/// ZADD of an existing member just updates its score).
#[derive(Default, Clone)]
pub struct InMemoryAttendance {
    sessions: Arc<DashMap<String, LectureState>>,
    attendees: Arc<DashMap<String, Arc<DashMap<String, i64>>>>,
}

impl InMemoryAttendance {
    pub fn new() -> Self {
        Self::default()
    }

    fn key(tenant_slug: &str, session_id: &str) -> String {
        format!("tenant:{}:live:{}", tenant_slug, session_id)
    }
}

#[async_trait]
impl AttendanceStore for InMemoryAttendance {
    async fn init_session(&self, state: LectureState) {
        let k = Self::key(&state.tenant_slug, &state.session_id);
        self.attendees
            .entry(k.clone())
            .or_insert_with(|| Arc::new(DashMap::new()));
        self.sessions.insert(k, state);
    }

    async fn end_session(&self, tenant_slug: &str, session_id: &str) {
        let k = Self::key(tenant_slug, session_id);
        if let Some(mut entry) = self.sessions.get_mut(&k) {
            entry.active = false;
        }
        self.attendees.remove(&k);
    }

    async fn record_join(
        &self,
        tenant_slug: &str,
        session_id: &str,
        user_id: &str,
        join_timestamp: i64,
    ) {
        let k = Self::key(tenant_slug, session_id);
        let bucket = self
            .attendees
            .entry(k)
            .or_insert_with(|| Arc::new(DashMap::new()))
            .value()
            .clone();
        bucket.insert(user_id.to_string(), join_timestamp);
    }

    async fn attendee_count(&self, tenant_slug: &str, session_id: &str) -> i32 {
        let k = Self::key(tenant_slug, session_id);
        self.attendees
            .get(&k)
            .map(|b| b.value().len() as i32)
            .unwrap_or(0)
    }

    async fn get_state(&self, tenant_slug: &str, session_id: &str) -> Option<LectureState> {
        let k = Self::key(tenant_slug, session_id);
        self.sessions.get(&k).map(|e| e.value().clone())
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    fn state(slug: &str, sid: &str, enrolled: i32) -> LectureState {
        LectureState {
            session_id: sid.to_string(),
            tenant_slug: slug.to_string(),
            enrolled_count: enrolled,
            started_at_unix: 1_000,
            active: true,
        }
    }

    #[tokio::test]
    async fn zadd_zcard_math_basic() {
        // W11.1 acceptance: ZADD then ZCARD reflects unique members.
        let store = InMemoryAttendance::new();
        store.init_session(state("eastfield", "s1", 30)).await;

        store.record_join("eastfield", "s1", "u1", 100).await;
        store.record_join("eastfield", "s1", "u2", 101).await;
        store.record_join("eastfield", "s1", "u3", 102).await;

        assert_eq!(store.attendee_count("eastfield", "s1").await, 3);
    }

    #[tokio::test]
    async fn zadd_duplicate_member_does_not_inflate() {
        // Sorted-set semantics: rejoining (same user_id) must not double-count.
        let store = InMemoryAttendance::new();
        store.init_session(state("eastfield", "s2", 10)).await;

        store.record_join("eastfield", "s2", "u1", 100).await;
        store.record_join("eastfield", "s2", "u1", 200).await;
        store.record_join("eastfield", "s2", "u1", 300).await;

        assert_eq!(store.attendee_count("eastfield", "s2").await, 1);
    }

    #[tokio::test]
    async fn sessions_are_tenant_isolated() {
        // Same session_id across tenants must not collide.
        let store = InMemoryAttendance::new();
        store.init_session(state("eastfield", "s", 10)).await;
        store.init_session(state("riverdale", "s", 10)).await;

        store.record_join("eastfield", "s", "u1", 100).await;
        store.record_join("eastfield", "s", "u2", 101).await;
        store.record_join("riverdale", "s", "u9", 110).await;

        assert_eq!(store.attendee_count("eastfield", "s").await, 2);
        assert_eq!(store.attendee_count("riverdale", "s").await, 1);
    }

    #[tokio::test]
    async fn end_session_clears_attendance_and_deactivates() {
        let store = InMemoryAttendance::new();
        store.init_session(state("eastfield", "s3", 10)).await;
        store.record_join("eastfield", "s3", "u1", 100).await;

        store.end_session("eastfield", "s3").await;

        assert_eq!(store.attendee_count("eastfield", "s3").await, 0);
        let st = store.get_state("eastfield", "s3").await.unwrap();
        assert!(!st.active);
    }
}
