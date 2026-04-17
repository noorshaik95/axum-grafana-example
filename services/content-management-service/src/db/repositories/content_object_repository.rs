use crate::models::ContentObject;
use anyhow::{Context, Result};
use sqlx::PgPool;
use uuid::Uuid;

/// Repository for managing ContentObject entities
pub struct ContentObjectRepository {
    pool: PgPool,
}

impl ContentObjectRepository {
    pub fn new(pool: PgPool) -> Self {
        Self { pool }
    }

    /// Insert a new content object record
    pub async fn create(
        &self,
        tenant_id: Uuid,
        uploader_id: Uuid,
        course_id: Option<Uuid>,
        module_id: Option<String>,
        filename: String,
        content_type: String,
        file_size_bytes: i64,
        minio_bucket: String,
        minio_object_key: String,
        content_category: Option<String>,
        visibility_rules: serde_json::Value,
    ) -> Result<ContentObject> {
        let obj = sqlx::query_as::<_, ContentObject>(
            r#"
            INSERT INTO content_objects (
                tenant_id, uploader_id, course_id, module_id,
                filename, content_type, file_size_bytes,
                minio_bucket, minio_object_key,
                content_category, visibility_rules
            )
            VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
            RETURNING id, tenant_id, uploader_id, course_id, module_id,
                      filename, content_type, file_size_bytes,
                      minio_bucket, minio_object_key,
                      content_category, visibility_rules, is_deleted,
                      created_at, updated_at
            "#,
        )
        .bind(tenant_id)
        .bind(uploader_id)
        .bind(course_id)
        .bind(module_id)
        .bind(filename)
        .bind(content_type)
        .bind(file_size_bytes)
        .bind(minio_bucket)
        .bind(minio_object_key)
        .bind(content_category)
        .bind(visibility_rules)
        .fetch_one(&self.pool)
        .await
        .context("Failed to create content object")?;

        Ok(obj)
    }

    /// Find a content object by ID (not soft-deleted)
    pub async fn find_by_id(
        &self,
        id: Uuid,
        tenant_id: Uuid,
    ) -> Result<Option<ContentObject>> {
        let obj = sqlx::query_as::<_, ContentObject>(
            r#"
            SELECT id, tenant_id, uploader_id, course_id, module_id,
                   filename, content_type, file_size_bytes,
                   minio_bucket, minio_object_key,
                   content_category, visibility_rules, is_deleted,
                   created_at, updated_at
            FROM content_objects
            WHERE id = $1 AND tenant_id = $2 AND is_deleted = false
            "#,
        )
        .bind(id)
        .bind(tenant_id)
        .fetch_optional(&self.pool)
        .await
        .context("Failed to find content object by ID")?;

        Ok(obj)
    }

    /// List all content objects for a course (not soft-deleted)
    pub async fn list_by_course(
        &self,
        tenant_id: Uuid,
        course_id: Uuid,
    ) -> Result<Vec<ContentObject>> {
        let objects = sqlx::query_as::<_, ContentObject>(
            r#"
            SELECT id, tenant_id, uploader_id, course_id, module_id,
                   filename, content_type, file_size_bytes,
                   minio_bucket, minio_object_key,
                   content_category, visibility_rules, is_deleted,
                   created_at, updated_at
            FROM content_objects
            WHERE tenant_id = $1 AND course_id = $2 AND is_deleted = false
            ORDER BY created_at DESC
            "#,
        )
        .bind(tenant_id)
        .bind(course_id)
        .fetch_all(&self.pool)
        .await
        .context("Failed to list content objects by course")?;

        Ok(objects)
    }

    /// Soft-delete a content object
    pub async fn soft_delete(&self, id: Uuid, tenant_id: Uuid) -> Result<bool> {
        let result = sqlx::query(
            r#"
            UPDATE content_objects
            SET is_deleted = true, updated_at = NOW()
            WHERE id = $1 AND tenant_id = $2 AND is_deleted = false
            "#,
        )
        .bind(id)
        .bind(tenant_id)
        .execute(&self.pool)
        .await
        .context("Failed to soft-delete content object")?;

        Ok(result.rows_affected() > 0)
    }

    /// Update visibility rules for a content object
    pub async fn update_visibility(
        &self,
        id: Uuid,
        tenant_id: Uuid,
        visibility_rules: serde_json::Value,
    ) -> Result<Option<ContentObject>> {
        let obj = sqlx::query_as::<_, ContentObject>(
            r#"
            UPDATE content_objects
            SET visibility_rules = $3, updated_at = NOW()
            WHERE id = $1 AND tenant_id = $2 AND is_deleted = false
            RETURNING id, tenant_id, uploader_id, course_id, module_id,
                      filename, content_type, file_size_bytes,
                      minio_bucket, minio_object_key,
                      content_category, visibility_rules, is_deleted,
                      created_at, updated_at
            "#,
        )
        .bind(id)
        .bind(tenant_id)
        .bind(visibility_rules)
        .fetch_optional(&self.pool)
        .await
        .context("Failed to update visibility rules")?;

        Ok(obj)
    }

    /// Associate content with a course (set course_id)
    pub async fn associate_with_course(
        &self,
        id: Uuid,
        tenant_id: Uuid,
        course_id: Uuid,
        content_category: Option<String>,
        module_id: Option<String>,
    ) -> Result<Option<ContentObject>> {
        let obj = sqlx::query_as::<_, ContentObject>(
            r#"
            UPDATE content_objects
            SET course_id = $3,
                content_category = COALESCE($4, content_category),
                module_id = COALESCE($5, module_id),
                updated_at = NOW()
            WHERE id = $1 AND tenant_id = $2 AND is_deleted = false
            RETURNING id, tenant_id, uploader_id, course_id, module_id,
                      filename, content_type, file_size_bytes,
                      minio_bucket, minio_object_key,
                      content_category, visibility_rules, is_deleted,
                      created_at, updated_at
            "#,
        )
        .bind(id)
        .bind(tenant_id)
        .bind(course_id)
        .bind(content_category)
        .bind(module_id)
        .fetch_optional(&self.pool)
        .await
        .context("Failed to associate content with course")?;

        Ok(obj)
    }

    /// Soft-delete all content for a course (used when course is deleted)
    pub async fn soft_delete_by_course(
        &self,
        tenant_id: Uuid,
        course_id: Uuid,
    ) -> Result<Vec<ContentObject>> {
        let objects = sqlx::query_as::<_, ContentObject>(
            r#"
            UPDATE content_objects
            SET is_deleted = true, updated_at = NOW()
            WHERE tenant_id = $1 AND course_id = $2 AND is_deleted = false
            RETURNING id, tenant_id, uploader_id, course_id, module_id,
                      filename, content_type, file_size_bytes,
                      minio_bucket, minio_object_key,
                      content_category, visibility_rules, is_deleted,
                      created_at, updated_at
            "#,
        )
        .bind(tenant_id)
        .bind(course_id)
        .fetch_all(&self.pool)
        .await
        .context("Failed to soft-delete content by course")?;

        Ok(objects)
    }

    /// Mark all tenant content as restricted (hidden visibility)
    pub async fn restrict_tenant_content(&self, tenant_id: Uuid) -> Result<u64> {
        let result = sqlx::query(
            r#"
            UPDATE content_objects
            SET visibility_rules = '{"type":"hidden"}', updated_at = NOW()
            WHERE tenant_id = $1 AND is_deleted = false
            "#,
        )
        .bind(tenant_id)
        .execute(&self.pool)
        .await
        .context("Failed to restrict tenant content")?;

        Ok(result.rows_affected())
    }
}
