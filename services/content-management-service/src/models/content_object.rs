use chrono::{DateTime, Utc};
use serde::{Deserialize, Serialize};
use sqlx::FromRow;
use uuid::Uuid;

/// ContentObject represents a tenant-isolated file stored in MinIO
#[derive(Debug, Clone, Serialize, Deserialize, FromRow)]
pub struct ContentObject {
    pub id: Uuid,
    pub tenant_id: Uuid,
    pub uploader_id: Uuid,
    pub course_id: Option<Uuid>,
    pub module_id: Option<String>,
    pub filename: String,
    pub content_type: String,
    pub file_size_bytes: i64,
    pub minio_bucket: String,
    pub minio_object_key: String,
    pub content_category: Option<String>,
    pub visibility_rules: serde_json::Value,
    pub is_deleted: bool,
    pub created_at: DateTime<Utc>,
    pub updated_at: DateTime<Utc>,
}
