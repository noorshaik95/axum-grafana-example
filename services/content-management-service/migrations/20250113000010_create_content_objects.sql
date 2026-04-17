-- Content objects table for tenant-isolated file storage
CREATE TABLE IF NOT EXISTS content_objects (
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
    content_category VARCHAR(50),
    visibility_rules JSONB DEFAULT '{"type":"always"}',
    is_deleted BOOLEAN DEFAULT false,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_content_tenant_course ON content_objects(tenant_id, course_id);
CREATE INDEX idx_content_uploader ON content_objects(tenant_id, uploader_id);
CREATE INDEX idx_content_not_deleted ON content_objects(tenant_id, is_deleted) WHERE is_deleted = false;
