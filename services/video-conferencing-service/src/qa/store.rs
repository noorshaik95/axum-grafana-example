use async_trait::async_trait;
use dashmap::DashMap;
use std::sync::Arc;

#[derive(Debug, Clone)]
pub struct Question {
    pub id: String,
    pub user_id: String,
    pub text: String,
    pub upvotes: i32,
    pub submitted_at_unix: i64,
}

#[async_trait]
pub trait QaStore: Send + Sync {
    async fn push(&self, tenant_slug: &str, session_id: &str, q: Question);
    /// Atomically INCR upvote counter. Returns the new upvote count, or None
    /// if the question doesn't exist.
    async fn incr_upvote(
        &self,
        tenant_slug: &str,
        session_id: &str,
        question_id: &str,
    ) -> Option<i32>;
    async fn get(
        &self,
        tenant_slug: &str,
        session_id: &str,
        question_id: &str,
    ) -> Option<Question>;
    async fn list(&self, tenant_slug: &str, session_id: &str) -> Vec<Question>;
}

#[derive(Default, Clone)]
pub struct InMemoryQaStore {
    inner: Arc<DashMap<String, Arc<DashMap<String, Question>>>>,
}

impl InMemoryQaStore {
    pub fn new() -> Self {
        Self::default()
    }

    fn key(tenant_slug: &str, session_id: &str) -> String {
        format!("tenant:{}:live:{}:questions", tenant_slug, session_id)
    }
}

#[async_trait]
impl QaStore for InMemoryQaStore {
    async fn push(&self, tenant_slug: &str, session_id: &str, q: Question) {
        let k = Self::key(tenant_slug, session_id);
        let bucket = self
            .inner
            .entry(k)
            .or_insert_with(|| Arc::new(DashMap::new()))
            .value()
            .clone();
        bucket.insert(q.id.clone(), q);
    }

    async fn incr_upvote(
        &self,
        tenant_slug: &str,
        session_id: &str,
        question_id: &str,
    ) -> Option<i32> {
        let k = Self::key(tenant_slug, session_id);
        let bucket = self.inner.get(&k)?.value().clone();
        let mut entry = bucket.get_mut(question_id)?;
        entry.upvotes += 1;
        Some(entry.upvotes)
    }

    async fn get(
        &self,
        tenant_slug: &str,
        session_id: &str,
        question_id: &str,
    ) -> Option<Question> {
        let k = Self::key(tenant_slug, session_id);
        let bucket = self.inner.get(&k)?.value().clone();
        bucket.get(question_id).map(|q| q.value().clone())
    }

    async fn list(&self, tenant_slug: &str, session_id: &str) -> Vec<Question> {
        let k = Self::key(tenant_slug, session_id);
        let Some(bucket) = self.inner.get(&k).map(|b| b.value().clone()) else {
            return Vec::new();
        };
        let mut items: Vec<Question> = bucket.iter().map(|e| e.value().clone()).collect();
        items.sort_by(|a, b| {
            b.upvotes
                .cmp(&a.upvotes)
                .then(a.submitted_at_unix.cmp(&b.submitted_at_unix))
        });
        items
    }
}
