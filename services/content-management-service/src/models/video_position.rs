use chrono::{DateTime, Utc};
use serde::{Deserialize, Serialize};
use sqlx::FromRow;
use uuid::Uuid;

#[derive(Debug, Clone, Serialize, Deserialize, FromRow)]
pub struct VideoPosition {
    pub user_id: Uuid,
    pub content_id: Uuid,
    pub position_seconds: i32,
    pub updated_at: DateTime<Utc>,
}
