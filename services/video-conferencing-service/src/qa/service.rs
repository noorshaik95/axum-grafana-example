use super::store::{QaStore, Question};
use chrono::Utc;
use std::sync::Arc;
use uuid::Uuid;

#[derive(Clone)]
pub struct QaService {
    store: Arc<dyn QaStore>,
}

impl QaService {
    pub fn new(store: Arc<dyn QaStore>) -> Self {
        Self { store }
    }

    pub async fn submit(
        &self,
        tenant_slug: &str,
        session_id: &str,
        user_id: &str,
        text: &str,
    ) -> Question {
        let q = Question {
            id: Uuid::new_v4().to_string(),
            user_id: user_id.to_string(),
            text: text.to_string(),
            upvotes: 0,
            submitted_at_unix: Utc::now().timestamp(),
        };
        self.store.push(tenant_slug, session_id, q.clone()).await;
        q
    }

    /// Returns the updated question, or None if the question doesn't exist.
    pub async fn upvote(
        &self,
        tenant_slug: &str,
        session_id: &str,
        question_id: &str,
    ) -> Option<Question> {
        self.store
            .incr_upvote(tenant_slug, session_id, question_id)
            .await?;
        self.store.get(tenant_slug, session_id, question_id).await
    }

    /// List questions sorted by upvotes desc. `limit` of 0 returns all.
    pub async fn list(
        &self,
        tenant_slug: &str,
        session_id: &str,
        limit: usize,
    ) -> Vec<Question> {
        let mut items = self.store.list(tenant_slug, session_id).await;
        if limit > 0 && items.len() > limit {
            items.truncate(limit);
        }
        items
    }
}

#[cfg(test)]
mod tests {
    use super::super::store::InMemoryQaStore;
    use super::*;

    fn svc() -> QaService {
        QaService::new(Arc::new(InMemoryQaStore::new()))
    }

    #[tokio::test]
    async fn submit_pushes_with_zero_upvotes() {
        let svc = svc();
        let q = svc.submit("eastfield", "s1", "u1", "What's the deadline?").await;
        assert_eq!(q.upvotes, 0);
        assert_eq!(q.text, "What's the deadline?");
    }

    #[tokio::test]
    async fn upvote_increments_counter() {
        let svc = svc();
        let q = svc.submit("eastfield", "s1", "u1", "q").await;
        let after1 = svc.upvote("eastfield", "s1", &q.id).await.unwrap();
        let after2 = svc.upvote("eastfield", "s1", &q.id).await.unwrap();
        assert_eq!(after1.upvotes, 1);
        assert_eq!(after2.upvotes, 2);
    }

    #[tokio::test]
    async fn list_sorted_by_upvotes_desc() {
        // W11.2 acceptance: sorted by upvotes desc.
        let svc = svc();
        let q1 = svc.submit("eastfield", "s1", "u1", "low").await;
        let q2 = svc.submit("eastfield", "s1", "u2", "high").await;
        let q3 = svc.submit("eastfield", "s1", "u3", "mid").await;

        svc.upvote("eastfield", "s1", &q2.id).await;
        svc.upvote("eastfield", "s1", &q2.id).await;
        svc.upvote("eastfield", "s1", &q2.id).await;
        svc.upvote("eastfield", "s1", &q3.id).await;

        let list = svc.list("eastfield", "s1", 0).await;
        assert_eq!(list.len(), 3);
        assert_eq!(list[0].id, q2.id); // 3 upvotes
        assert_eq!(list[1].id, q3.id); // 1 upvote
        assert_eq!(list[2].id, q1.id); // 0 upvotes
    }

    #[tokio::test]
    async fn list_respects_limit() {
        let svc = svc();
        for i in 0..5 {
            svc.submit("eastfield", "s1", "u", &format!("q{}", i)).await;
        }
        let list = svc.list("eastfield", "s1", 3).await;
        assert_eq!(list.len(), 3);
    }

    #[tokio::test]
    async fn upvote_missing_question_returns_none() {
        let svc = svc();
        let res = svc.upvote("eastfield", "s1", "nope").await;
        assert!(res.is_none());
    }

    #[tokio::test]
    async fn tenant_isolation_on_questions() {
        let svc = svc();
        let _a = svc.submit("eastfield", "s1", "u1", "east").await;
        let _b = svc.submit("riverdale", "s1", "u1", "river").await;
        assert_eq!(svc.list("eastfield", "s1", 0).await.len(), 1);
        assert_eq!(svc.list("riverdale", "s1", 0).await.len(), 1);
    }
}
