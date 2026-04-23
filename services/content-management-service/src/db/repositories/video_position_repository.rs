use crate::models::VideoPosition;
use anyhow::{Context, Result};
use sqlx::PgPool;
use uuid::Uuid;

/// Persists student video resume positions; acts as a durable fallback for the
/// Redis `video_pos` cache.
pub struct VideoPositionRepository {
    pool: PgPool,
}

impl VideoPositionRepository {
    pub fn new(pool: PgPool) -> Self {
        Self { pool }
    }

    pub async fn upsert(
        &self,
        user_id: Uuid,
        content_id: Uuid,
        position_seconds: i32,
    ) -> Result<VideoPosition> {
        let row = sqlx::query_as::<_, VideoPosition>(
            r#"
            INSERT INTO video_positions (user_id, content_id, position_seconds, updated_at)
            VALUES ($1, $2, $3, NOW())
            ON CONFLICT (user_id, content_id) DO UPDATE
                SET position_seconds = EXCLUDED.position_seconds,
                    updated_at = NOW()
            RETURNING user_id, content_id, position_seconds, updated_at
            "#,
        )
        .bind(user_id)
        .bind(content_id)
        .bind(position_seconds)
        .fetch_one(&self.pool)
        .await
        .context("Failed to upsert video position")?;

        Ok(row)
    }

    pub async fn find(
        &self,
        user_id: Uuid,
        content_id: Uuid,
    ) -> Result<Option<VideoPosition>> {
        let row = sqlx::query_as::<_, VideoPosition>(
            r#"
            SELECT user_id, content_id, position_seconds, updated_at
            FROM video_positions
            WHERE user_id = $1 AND content_id = $2
            "#,
        )
        .bind(user_id)
        .bind(content_id)
        .fetch_optional(&self.pool)
        .await
        .context("Failed to fetch video position")?;

        Ok(row)
    }
}
