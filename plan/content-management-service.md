# Content Management Service Plan

## Owner Agent: `content-expert`

## Stack: Rust, tonic (gRPC), axum, MinIO (S3-compatible), PostgreSQL (sqlx), Kafka

---

## Objective

Complete the content-management-service with full file upload/management APIs, tenant-isolated MinIO buckets, signed URL generation, visibility rules, and Kafka integration.

---

## Target APIs

### REST (port 8084)

```
POST   /content/upload                    — upload file (multipart, tenant-isolated)
GET    /content/:id                       — file metadata + signed download URL
DELETE /content/:id                       — delete file + MinIO object
GET    /courses/:courseId/materials       — list all materials for a course
POST   /courses/:courseId/materials       — associate uploaded content with course
PUT    /content/:id/visibility            — set visibility rules
GET    /content/:id/download              — redirect to signed MinIO URL
POST   /content/presigned-url             — generate presigned upload URL
```

### gRPC (port 50054)

```protobuf
service ContentManagementService {
    rpc UploadContent(UploadRequest) returns (UploadResponse);
    rpc GetContent(GetContentRequest) returns (ContentMetadata);
    rpc DeleteContent(DeleteContentRequest) returns (DeleteResponse);
    rpc ListCourseMaterials(ListMaterialsRequest) returns (ListMaterialsResponse);
    rpc GenerateDownloadUrl(DownloadUrlRequest) returns (DownloadUrlResponse);
    rpc UpdateVisibility(UpdateVisibilityRequest) returns (UpdateVisibilityResponse);
}
```

---

## Data Models (PostgreSQL)

### content_objects

```sql
CREATE TABLE content_objects (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    uploader_id UUID NOT NULL,
    course_id UUID,
    module_id VARCHAR(100),
    filename VARCHAR(1000) NOT NULL,
    content_type VARCHAR(200) NOT NULL,
    file_size_bytes BIGINT NOT NULL,
    minio_bucket VARCHAR(200) NOT NULL,
    minio_object_key VARCHAR(1000) NOT NULL,
    content_category VARCHAR(50),     -- 'syllabus' | 'lecture' | 'assignment' | 'recording'
    visibility_rules JSONB,           -- { type: 'always'|'after_date'|'specific_groups', date?, groupIds? }
    is_deleted BOOLEAN DEFAULT false,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);
```

---

## MinIO Bucket Strategy

- **Bucket per tenant**: `slate-content-{tenantId}`
- Created when tenant is provisioned (Kafka: `tenant.provisioned`)
- Object key: `{courseId}/{category}/{objectId}/{filename}`
- Signed URL expiry: 1 hour for downloads, 15 minutes for uploads

```rust
// src/storage/minio.rs
pub struct MinioClient {
    client: s3::Client,
}

impl MinioClient {
    pub async fn generate_presigned_upload_url(&self, bucket: &str, key: &str) -> Result<String> {
        // Returns presigned PUT URL (15min expiry)
    }

    pub async fn generate_presigned_download_url(&self, bucket: &str, key: &str) -> Result<String> {
        // Returns presigned GET URL (1hr expiry)
    }

    pub async fn delete_object(&self, bucket: &str, key: &str) -> Result<()> {
        // Hard delete from MinIO
    }

    pub async fn create_bucket(&self, tenant_id: &str) -> Result<()> {
        // Create slate-content-{tenantId} bucket
    }
}
```

## Visibility Rules

```rust
#[derive(Serialize, Deserialize)]
#[serde(tag = "type")]
pub enum VisibilityRule {
    Always,
    AfterDate { date: DateTime<Utc> },
    SpecificGroups { group_ids: Vec<String> },
    Hidden,
}

pub fn is_visible_to(rule: &VisibilityRule, user_groups: &[String]) -> bool {
    match rule {
        VisibilityRule::Always => true,
        VisibilityRule::AfterDate { date } => Utc::now() >= *date,
        VisibilityRule::SpecificGroups { group_ids } => {
            user_groups.iter().any(|g| group_ids.contains(g))
        }
        VisibilityRule::Hidden => false,
    }
}
```

---

## Kafka Events

### Produced

- `content.uploaded` `{ contentId, tenantId, courseId, uploaderId, filename, contentType }`
- `content.deleted` `{ contentId, tenantId, courseId }`

### Consumed

- `course.deleted` → delete all content for that course (MinIO + DB)
- `tenant.disabled` → mark all tenant content as restricted (no new downloads)
- `tenant.provisioned` → create MinIO bucket for tenant

---

## File Upload Flow

1. Client requests presigned URL: `POST /content/presigned-url`
2. Client uploads directly to MinIO via presigned URL
3. Client calls `POST /content/upload` with metadata (filename, courseId, category)
4. Service records metadata in DB
5. Returns content object with signed download URL

Or: direct multipart upload through service (for smaller files)

---

## Tests

- Unit: visibility rule evaluation, URL signing
- Integration: upload → metadata store → download URL → download
- Kafka: verify events on upload/delete/course-deleted
- Tenant isolation: verify cross-tenant file access denied
- Cleanup: course.deleted → all files purged from MinIO

---

## Files to Create/Modify

- [MODIFY] `services/content-management-service/src/` — complete all handlers
- [NEW] `services/content-management-service/src/storage/minio.rs` — MinIO client
- [NEW] `services/content-management-service/src/kafka/` — consumer + producer
- [MODIFY] `services/content-management-service/src/grpc/service.rs` — all RPCs
- [MODIFY] `services/content-management-service/migrations/` — content_objects table
- [MODIFY] `proto/content_management.proto`
