use crate::db::repositories::ContentObjectRepository;
use crate::kafka::KafkaProducer;
use crate::kafka::producer::{ContentDeletedEvent, ContentUploadedEvent};
use crate::storage::minio::MinioClient;
use crate::visibility::VisibilityRule;
use axum::{
    extract::{Multipart, Path, State},
    http::{HeaderMap, StatusCode},
    response::{IntoResponse, Redirect, Response},
    routing::{delete, get, post, put},
    Json, Router,
};
use serde::{Deserialize, Serialize};
use std::sync::Arc;
use tracing::{error, info};
use uuid::Uuid;

/// Shared application state for content REST handlers
#[derive(Clone)]
pub struct ContentAppState {
    pub minio_client: Arc<MinioClient>,
    pub content_repo: Arc<ContentObjectRepository>,
    pub kafka_producer: Arc<KafkaProducer>,
    pub bucket_prefix: String,
}

/// Build the content router
pub fn content_routes(state: ContentAppState) -> Router {
    Router::new()
        .route("/content/upload", post(upload_content))
        .route("/content/presigned-url", post(presigned_upload_url))
        .route("/content/:id", get(get_content))
        .route("/content/:id", delete(delete_content))
        .route("/content/:id/visibility", put(update_visibility))
        .route("/content/:id/download", get(download_content))
        .route("/courses/:course_id/materials", get(list_course_materials))
        .route(
            "/courses/:course_id/materials",
            post(associate_course_material),
        )
        .with_state(state)
}

// ---- Request / Response types ----

#[derive(Deserialize)]
struct PresignedUrlRequest {
    tenant_id: Uuid,
    filename: String,
    content_type: String,
    course_id: Option<Uuid>,
    category: Option<String>,
}

#[derive(Serialize)]
struct PresignedUrlResponse {
    upload_url: String,
    object_key: String,
    bucket: String,
}

#[derive(Serialize)]
struct ContentMetadataResponse {
    id: Uuid,
    tenant_id: Uuid,
    uploader_id: Uuid,
    course_id: Option<Uuid>,
    module_id: Option<String>,
    filename: String,
    content_type: String,
    file_size_bytes: i64,
    content_category: Option<String>,
    visibility_rules: serde_json::Value,
    download_url: Option<String>,
    created_at: String,
    updated_at: String,
}

#[derive(Deserialize)]
struct UpdateVisibilityRequest {
    visibility_rules: VisibilityRule,
}

#[derive(Deserialize)]
struct AssociateMaterialRequest {
    content_id: Uuid,
    category: Option<String>,
    module_id: Option<String>,
}

// ---- Error response ----

struct AppError(StatusCode, String);

impl IntoResponse for AppError {
    fn into_response(self) -> Response {
        let body = serde_json::json!({
            "error": self.1,
        });
        (self.0, Json(body)).into_response()
    }
}

// ---- Helper: extract tenant ID from headers ----

fn extract_tenant_id(headers: &HeaderMap) -> Result<Uuid, AppError> {
    let tenant_str = headers
        .get("X-Tenant-ID")
        .and_then(|v| v.to_str().ok())
        .ok_or_else(|| {
            AppError(
                StatusCode::BAD_REQUEST,
                "Missing X-Tenant-ID header".to_string(),
            )
        })?;
    Uuid::parse_str(tenant_str).map_err(|_| {
        AppError(
            StatusCode::BAD_REQUEST,
            "Invalid X-Tenant-ID header".to_string(),
        )
    })
}

fn extract_uploader_id(headers: &HeaderMap) -> Result<Uuid, AppError> {
    let uploader_str = headers
        .get("X-User-ID")
        .and_then(|v| v.to_str().ok())
        .ok_or_else(|| {
            AppError(
                StatusCode::BAD_REQUEST,
                "Missing X-User-ID header".to_string(),
            )
        })?;
    Uuid::parse_str(uploader_str).map_err(|_| {
        AppError(
            StatusCode::BAD_REQUEST,
            "Invalid X-User-ID header".to_string(),
        )
    })
}

// ---- Handlers ----

/// POST /content/upload — multipart upload through service
async fn upload_content(
    State(state): State<ContentAppState>,
    headers: HeaderMap,
    mut multipart: Multipart,
) -> Result<(StatusCode, Json<ContentMetadataResponse>), AppError> {
    let tenant_id = extract_tenant_id(&headers)?;
    let uploader_id = extract_uploader_id(&headers)?;

    let mut course_id: Option<Uuid> = None;
    let mut category: Option<String> = None;
    let mut module_id: Option<String> = None;
    let mut file_data: Option<bytes::Bytes> = None;
    let mut file_name: Option<String> = None;
    let mut file_content_type: Option<String> = None;

    while let Some(field) = multipart.next_field().await.map_err(|e| {
        AppError(
            StatusCode::BAD_REQUEST,
            format!("Failed to read multipart field: {}", e),
        )
    })? {
        let name = field.name().unwrap_or("").to_string();
        match name.as_str() {
            "courseId" | "course_id" => {
                let val = field.text().await.map_err(|e| {
                    AppError(StatusCode::BAD_REQUEST, format!("Invalid courseId: {}", e))
                })?;
                if !val.is_empty() {
                    course_id = Some(Uuid::parse_str(&val).map_err(|_| {
                        AppError(StatusCode::BAD_REQUEST, "Invalid courseId UUID".to_string())
                    })?);
                }
            }
            "category" => {
                let val = field.text().await.map_err(|e| {
                    AppError(StatusCode::BAD_REQUEST, format!("Invalid category: {}", e))
                })?;
                if !val.is_empty() {
                    category = Some(val);
                }
            }
            "moduleId" | "module_id" => {
                let val = field.text().await.map_err(|e| {
                    AppError(StatusCode::BAD_REQUEST, format!("Invalid moduleId: {}", e))
                })?;
                if !val.is_empty() {
                    module_id = Some(val);
                }
            }
            "file" => {
                file_name = field.file_name().map(|s| s.to_string());
                file_content_type = field.content_type().map(|s| s.to_string());
                file_data = Some(field.bytes().await.map_err(|e| {
                    AppError(
                        StatusCode::BAD_REQUEST,
                        format!("Failed to read file bytes: {}", e),
                    )
                })?);
            }
            _ => {
                // Skip unknown fields
            }
        }
    }

    let file_data = file_data.ok_or_else(|| {
        AppError(StatusCode::BAD_REQUEST, "Missing file field".to_string())
    })?;
    let file_name = file_name.unwrap_or_else(|| "unnamed".to_string());
    let file_content_type = file_content_type.unwrap_or_else(|| "application/octet-stream".to_string());
    let file_size = file_data.len() as i64;

    // Generate object key: {courseId}/{category}/{uuid}/{filename}
    let object_id = Uuid::new_v4();
    let course_part = course_id
        .map(|c| c.to_string())
        .unwrap_or_else(|| "uncategorized".to_string());
    let category_part = category.as_deref().unwrap_or("general");
    let object_key = format!("{}/{}/{}/{}", course_part, category_part, object_id, file_name);

    // Tenant bucket
    let bucket = format!("{}-{}", state.bucket_prefix, tenant_id);

    // Ensure bucket exists
    state
        .minio_client
        .create_bucket(&bucket)
        .await
        .map_err(|e| {
            error!("Failed to create bucket: {}", e);
            AppError(
                StatusCode::INTERNAL_SERVER_ERROR,
                "Failed to create storage bucket".to_string(),
            )
        })?;

    // Upload to MinIO
    state
        .minio_client
        .put_object(&bucket, &object_key, file_data, &file_content_type)
        .await
        .map_err(|e| {
            error!("Failed to upload to MinIO: {}", e);
            AppError(
                StatusCode::INTERNAL_SERVER_ERROR,
                "Failed to upload file".to_string(),
            )
        })?;

    // Insert record into database
    let content_obj = state
        .content_repo
        .create(
            tenant_id,
            uploader_id,
            course_id,
            module_id.clone(),
            file_name.clone(),
            file_content_type.clone(),
            file_size,
            bucket.clone(),
            object_key.clone(),
            category.clone(),
            serde_json::json!({"type": "always"}),
        )
        .await
        .map_err(|e| {
            error!("Failed to insert content record: {}", e);
            AppError(
                StatusCode::INTERNAL_SERVER_ERROR,
                "Failed to record content metadata".to_string(),
            )
        })?;

    // Generate download URL
    let download_url = state
        .minio_client
        .generate_presigned_download_url(&bucket, &object_key, 3600)
        .await
        .ok();

    // Emit Kafka event
    state
        .kafka_producer
        .emit_content_uploaded(ContentUploadedEvent {
            content_id: content_obj.id,
            tenant_id,
            course_id,
            uploader_id,
            filename: file_name,
            content_type: file_content_type.clone(),
        })
        .await;

    info!(
        content_id = %content_obj.id,
        tenant_id = %tenant_id,
        "Content uploaded successfully"
    );

    Ok((
        StatusCode::CREATED,
        Json(ContentMetadataResponse {
            id: content_obj.id,
            tenant_id: content_obj.tenant_id,
            uploader_id: content_obj.uploader_id,
            course_id: content_obj.course_id,
            module_id: content_obj.module_id,
            filename: content_obj.filename,
            content_type: content_obj.content_type,
            file_size_bytes: content_obj.file_size_bytes,
            content_category: content_obj.content_category,
            visibility_rules: content_obj.visibility_rules,
            download_url,
            created_at: content_obj.created_at.to_rfc3339(),
            updated_at: content_obj.updated_at.to_rfc3339(),
        }),
    ))
}

/// POST /content/presigned-url — generate presigned PUT URL for direct browser upload
async fn presigned_upload_url(
    State(state): State<ContentAppState>,
    Json(req): Json<PresignedUrlRequest>,
) -> Result<Json<PresignedUrlResponse>, AppError> {
    let bucket = format!("{}-{}", state.bucket_prefix, req.tenant_id);

    // Ensure bucket exists
    state
        .minio_client
        .create_bucket(&bucket)
        .await
        .map_err(|e| {
            error!("Failed to create bucket: {}", e);
            AppError(
                StatusCode::INTERNAL_SERVER_ERROR,
                "Failed to create storage bucket".to_string(),
            )
        })?;

    let object_id = Uuid::new_v4();
    let course_part = req
        .course_id
        .map(|c| c.to_string())
        .unwrap_or_else(|| "uncategorized".to_string());
    let category_part = req.category.as_deref().unwrap_or("general");
    let object_key = format!(
        "{}/{}/{}/{}",
        course_part, category_part, object_id, req.filename
    );

    let upload_url = state
        .minio_client
        .generate_presigned_upload_url(&bucket, &object_key, 900) // 15 minutes
        .await
        .map_err(|e| {
            error!("Failed to generate presigned upload URL: {}", e);
            AppError(
                StatusCode::INTERNAL_SERVER_ERROR,
                "Failed to generate upload URL".to_string(),
            )
        })?;

    Ok(Json(PresignedUrlResponse {
        upload_url,
        object_key,
        bucket,
    }))
}

/// GET /content/:id — get metadata + presigned download URL
async fn get_content(
    State(state): State<ContentAppState>,
    headers: HeaderMap,
    Path(id): Path<Uuid>,
) -> Result<Json<ContentMetadataResponse>, AppError> {
    let tenant_id = extract_tenant_id(&headers)?;

    let obj = state
        .content_repo
        .find_by_id(id, tenant_id)
        .await
        .map_err(|e| {
            error!("Failed to fetch content: {}", e);
            AppError(
                StatusCode::INTERNAL_SERVER_ERROR,
                "Failed to fetch content".to_string(),
            )
        })?
        .ok_or_else(|| AppError(StatusCode::NOT_FOUND, "Content not found".to_string()))?;

    let download_url = state
        .minio_client
        .generate_presigned_download_url(&obj.minio_bucket, &obj.minio_object_key, 3600)
        .await
        .ok();

    Ok(Json(ContentMetadataResponse {
        id: obj.id,
        tenant_id: obj.tenant_id,
        uploader_id: obj.uploader_id,
        course_id: obj.course_id,
        module_id: obj.module_id,
        filename: obj.filename,
        content_type: obj.content_type,
        file_size_bytes: obj.file_size_bytes,
        content_category: obj.content_category,
        visibility_rules: obj.visibility_rules,
        download_url,
        created_at: obj.created_at.to_rfc3339(),
        updated_at: obj.updated_at.to_rfc3339(),
    }))
}

/// DELETE /content/:id — soft delete + MinIO object delete
async fn delete_content(
    State(state): State<ContentAppState>,
    headers: HeaderMap,
    Path(id): Path<Uuid>,
) -> Result<StatusCode, AppError> {
    let tenant_id = extract_tenant_id(&headers)?;

    // Get the object first to find MinIO key
    let obj = state
        .content_repo
        .find_by_id(id, tenant_id)
        .await
        .map_err(|e| {
            error!("Failed to fetch content for deletion: {}", e);
            AppError(
                StatusCode::INTERNAL_SERVER_ERROR,
                "Failed to fetch content".to_string(),
            )
        })?
        .ok_or_else(|| AppError(StatusCode::NOT_FOUND, "Content not found".to_string()))?;

    // Soft delete in DB
    state
        .content_repo
        .soft_delete(id, tenant_id)
        .await
        .map_err(|e| {
            error!("Failed to soft-delete content: {}", e);
            AppError(
                StatusCode::INTERNAL_SERVER_ERROR,
                "Failed to delete content".to_string(),
            )
        })?;

    // Hard delete from MinIO
    if let Err(e) = state
        .minio_client
        .delete_object(&obj.minio_bucket, &obj.minio_object_key)
        .await
    {
        error!(
            content_id = %id,
            "Failed to delete MinIO object (DB already soft-deleted): {}",
            e
        );
    }

    // Emit Kafka event
    state
        .kafka_producer
        .emit_content_deleted(ContentDeletedEvent {
            content_id: id,
            tenant_id,
            course_id: obj.course_id,
        })
        .await;

    info!(content_id = %id, "Content deleted");

    Ok(StatusCode::NO_CONTENT)
}

/// PUT /content/:id/visibility — update visibility rules
async fn update_visibility(
    State(state): State<ContentAppState>,
    headers: HeaderMap,
    Path(id): Path<Uuid>,
    Json(req): Json<UpdateVisibilityRequest>,
) -> Result<Json<ContentMetadataResponse>, AppError> {
    let tenant_id = extract_tenant_id(&headers)?;

    let rules_json = serde_json::to_value(&req.visibility_rules).map_err(|e| {
        AppError(
            StatusCode::BAD_REQUEST,
            format!("Invalid visibility rules: {}", e),
        )
    })?;

    let obj = state
        .content_repo
        .update_visibility(id, tenant_id, rules_json)
        .await
        .map_err(|e| {
            error!("Failed to update visibility: {}", e);
            AppError(
                StatusCode::INTERNAL_SERVER_ERROR,
                "Failed to update visibility".to_string(),
            )
        })?
        .ok_or_else(|| AppError(StatusCode::NOT_FOUND, "Content not found".to_string()))?;

    Ok(Json(ContentMetadataResponse {
        id: obj.id,
        tenant_id: obj.tenant_id,
        uploader_id: obj.uploader_id,
        course_id: obj.course_id,
        module_id: obj.module_id,
        filename: obj.filename,
        content_type: obj.content_type,
        file_size_bytes: obj.file_size_bytes,
        content_category: obj.content_category,
        visibility_rules: obj.visibility_rules,
        download_url: None,
        created_at: obj.created_at.to_rfc3339(),
        updated_at: obj.updated_at.to_rfc3339(),
    }))
}

/// GET /content/:id/download — redirect to signed MinIO URL
async fn download_content(
    State(state): State<ContentAppState>,
    headers: HeaderMap,
    Path(id): Path<Uuid>,
) -> Result<Redirect, AppError> {
    let tenant_id = extract_tenant_id(&headers)?;

    let obj = state
        .content_repo
        .find_by_id(id, tenant_id)
        .await
        .map_err(|e| {
            error!("Failed to fetch content for download: {}", e);
            AppError(
                StatusCode::INTERNAL_SERVER_ERROR,
                "Failed to fetch content".to_string(),
            )
        })?
        .ok_or_else(|| AppError(StatusCode::NOT_FOUND, "Content not found".to_string()))?;

    // Check visibility
    let rule: VisibilityRule =
        serde_json::from_value(obj.visibility_rules.clone()).unwrap_or(VisibilityRule::Always);

    // Extract user groups from header (comma-separated)
    let user_groups: Vec<String> = headers
        .get("X-User-Groups")
        .and_then(|v| v.to_str().ok())
        .map(|s| s.split(',').map(|g| g.trim().to_string()).collect())
        .unwrap_or_default();

    if !rule.is_visible_to(&user_groups) {
        return Err(AppError(
            StatusCode::FORBIDDEN,
            "Content is not currently accessible".to_string(),
        ));
    }

    let download_url = state
        .minio_client
        .generate_presigned_download_url(&obj.minio_bucket, &obj.minio_object_key, 3600)
        .await
        .map_err(|e| {
            error!("Failed to generate download URL: {}", e);
            AppError(
                StatusCode::INTERNAL_SERVER_ERROR,
                "Failed to generate download URL".to_string(),
            )
        })?;

    Ok(Redirect::temporary(&download_url))
}

/// GET /courses/:courseId/materials — list all materials for a course
async fn list_course_materials(
    State(state): State<ContentAppState>,
    headers: HeaderMap,
    Path(course_id): Path<Uuid>,
) -> Result<Json<Vec<ContentMetadataResponse>>, AppError> {
    let tenant_id = extract_tenant_id(&headers)?;

    let objects = state
        .content_repo
        .list_by_course(tenant_id, course_id)
        .await
        .map_err(|e| {
            error!("Failed to list course materials: {}", e);
            AppError(
                StatusCode::INTERNAL_SERVER_ERROR,
                "Failed to list materials".to_string(),
            )
        })?;

    // Extract user groups for visibility filtering
    let user_groups: Vec<String> = headers
        .get("X-User-Groups")
        .and_then(|v| v.to_str().ok())
        .map(|s| s.split(',').map(|g| g.trim().to_string()).collect())
        .unwrap_or_default();

    let results: Vec<ContentMetadataResponse> = objects
        .into_iter()
        .filter(|obj| {
            let rule: VisibilityRule = serde_json::from_value(obj.visibility_rules.clone())
                .unwrap_or(VisibilityRule::Always);
            rule.is_visible_to(&user_groups)
        })
        .map(|obj| ContentMetadataResponse {
            id: obj.id,
            tenant_id: obj.tenant_id,
            uploader_id: obj.uploader_id,
            course_id: obj.course_id,
            module_id: obj.module_id,
            filename: obj.filename,
            content_type: obj.content_type,
            file_size_bytes: obj.file_size_bytes,
            content_category: obj.content_category,
            visibility_rules: obj.visibility_rules,
            download_url: None,
            created_at: obj.created_at.to_rfc3339(),
            updated_at: obj.updated_at.to_rfc3339(),
        })
        .collect();

    Ok(Json(results))
}

/// POST /courses/:courseId/materials — associate uploaded content with course
async fn associate_course_material(
    State(state): State<ContentAppState>,
    headers: HeaderMap,
    Path(course_id): Path<Uuid>,
    Json(req): Json<AssociateMaterialRequest>,
) -> Result<Json<ContentMetadataResponse>, AppError> {
    let tenant_id = extract_tenant_id(&headers)?;

    let obj = state
        .content_repo
        .associate_with_course(
            req.content_id,
            tenant_id,
            course_id,
            req.category,
            req.module_id,
        )
        .await
        .map_err(|e| {
            error!("Failed to associate content with course: {}", e);
            AppError(
                StatusCode::INTERNAL_SERVER_ERROR,
                "Failed to associate content".to_string(),
            )
        })?
        .ok_or_else(|| AppError(StatusCode::NOT_FOUND, "Content not found".to_string()))?;

    Ok(Json(ContentMetadataResponse {
        id: obj.id,
        tenant_id: obj.tenant_id,
        uploader_id: obj.uploader_id,
        course_id: obj.course_id,
        module_id: obj.module_id,
        filename: obj.filename,
        content_type: obj.content_type,
        file_size_bytes: obj.file_size_bytes,
        content_category: obj.content_category,
        visibility_rules: obj.visibility_rules,
        download_url: None,
        created_at: obj.created_at.to_rfc3339(),
        updated_at: obj.updated_at.to_rfc3339(),
    }))
}
