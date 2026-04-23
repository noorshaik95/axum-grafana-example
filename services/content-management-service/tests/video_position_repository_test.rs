//! W10.1 — DB-backed tests for VideoPositionRepository (upsert + find).
//!
//! Gated on TEST_DATABASE_URL so it only runs when a Postgres fixture is
//! available. Run with:
//!   cargo test --test video_position_repository_test -- --ignored

use content_management_service::db::repositories::VideoPositionRepository;
use sqlx::PgPool;
use uuid::Uuid;

fn test_db_url() -> Option<String> {
    std::env::var("TEST_DATABASE_URL").ok()
}

#[tokio::test]
#[ignore]
async fn upsert_inserts_then_updates() {
    let url = match test_db_url() {
        Some(u) => u,
        None => {
            eprintln!("Skipping: TEST_DATABASE_URL not set");
            return;
        }
    };
    let pool = PgPool::connect(&url).await.expect("connect");
    sqlx::migrate!("./migrations")
        .run(&pool)
        .await
        .expect("migrate");

    let repo = VideoPositionRepository::new(pool);
    let user = Uuid::new_v4();
    let content = Uuid::new_v4();

    let first = repo.upsert(user, content, 42).await.expect("upsert 42");
    assert_eq!(first.position_seconds, 42);

    let second = repo.upsert(user, content, 123).await.expect("upsert 123");
    assert_eq!(second.position_seconds, 123, "second upsert should overwrite");

    let found = repo.find(user, content).await.expect("find").expect("row");
    assert_eq!(found.position_seconds, 123);
    assert!(found.updated_at >= first.updated_at);
}

#[tokio::test]
#[ignore]
async fn find_returns_none_when_absent() {
    let url = match test_db_url() {
        Some(u) => u,
        None => {
            eprintln!("Skipping: TEST_DATABASE_URL not set");
            return;
        }
    };
    let pool = PgPool::connect(&url).await.expect("connect");
    sqlx::migrate!("./migrations")
        .run(&pool)
        .await
        .expect("migrate");

    let repo = VideoPositionRepository::new(pool);
    let user = Uuid::new_v4();
    let content = Uuid::new_v4();
    let result = repo.find(user, content).await.expect("find");
    assert!(result.is_none());
}
