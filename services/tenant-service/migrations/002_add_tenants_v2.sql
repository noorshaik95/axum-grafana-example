-- Migration 002: Add tenants_v2 table for Docker-provisioned multi-tenant model

CREATE TABLE IF NOT EXISTS tenants_v2 (
    id VARCHAR(36) PRIMARY KEY,
    slug VARCHAR(100) UNIQUE NOT NULL,
    name VARCHAR(255) NOT NULL,
    admin_email VARCHAR(500),
    status VARCHAR(50) NOT NULL DEFAULT 'provisioning',
    plan JSONB NOT NULL DEFAULT '{}',
    subdomain VARCHAR(200),
    container_ids TEXT[] DEFAULT '{}',
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_tenants_v2_slug ON tenants_v2(slug);
CREATE INDEX IF NOT EXISTS idx_tenants_v2_status ON tenants_v2(status);
CREATE INDEX IF NOT EXISTS idx_tenants_v2_created_at ON tenants_v2(created_at DESC);

-- Record migration
INSERT INTO schema_migrations (version) VALUES ('002_add_tenants_v2') ON CONFLICT DO NOTHING;
