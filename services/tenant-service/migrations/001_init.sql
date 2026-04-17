-- Tenant Service Database Schema
-- Epic 9: US-9.1 Tenant Provisioning

-- Subscription tiers table
CREATE TABLE IF NOT EXISTS subscription_tiers (
    id VARCHAR(36) PRIMARY KEY,
    name VARCHAR(50) UNIQUE NOT NULL,
    display_name VARCHAR(100) NOT NULL,
    tier_level INTEGER NOT NULL, -- 0=free, 1=basic, 2=professional, 3=enterprise
    storage_quota_bytes BIGINT NOT NULL DEFAULT 0,
    max_users INTEGER NOT NULL DEFAULT 0,
    max_courses INTEGER NOT NULL DEFAULT 0,
    api_rate_limit INTEGER NOT NULL DEFAULT 100,
    dedicated_database BOOLEAN NOT NULL DEFAULT FALSE, -- AC2: Professional+ gets dedicated DB
    custom_domain BOOLEAN NOT NULL DEFAULT FALSE,
    priority_support BOOLEAN NOT NULL DEFAULT FALSE,
    price_cents_monthly INTEGER NOT NULL DEFAULT 0,
    features JSONB,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

-- Tenants table (AC1, AC2, AC5, AC6)
CREATE TABLE IF NOT EXISTS tenants (
    id VARCHAR(36) PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    domain VARCHAR(255) UNIQUE NOT NULL, -- AC5: Custom subdomain
    tier_id VARCHAR(36) NOT NULL REFERENCES subscription_tiers(id),
    is_active BOOLEAN NOT NULL DEFAULT TRUE,

    -- AC2: Dedicated database for Professional+ tier
    database_name VARCHAR(255),
    database_host VARCHAR(255),
    database_port INTEGER,
    database_username VARCHAR(255),
    database_password_encrypted TEXT,
    database_connection_string TEXT,

    -- AC6: Storage quota set by tier
    storage_quota_bytes BIGINT NOT NULL DEFAULT 0,
    storage_used_bytes BIGINT NOT NULL DEFAULT 0,
    file_count INTEGER NOT NULL DEFAULT 0,

    -- Usage tracking
    user_count INTEGER NOT NULL DEFAULT 0,
    course_count INTEGER NOT NULL DEFAULT 0,

    -- Metadata
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    last_active_at TIMESTAMP,

    -- Indexes
    CONSTRAINT valid_storage_usage CHECK (storage_used_bytes <= storage_quota_bytes)
);

CREATE INDEX idx_tenants_tier ON tenants(tier_id);
CREATE INDEX idx_tenants_is_active ON tenants(is_active);
CREATE INDEX idx_tenants_domain ON tenants(domain);
CREATE INDEX idx_tenants_created_at ON tenants(created_at DESC);

-- Tenant admins table (AC3: Create default admin account)
CREATE TABLE IF NOT EXISTS tenant_admins (
    id VARCHAR(36) PRIMARY KEY,
    tenant_id VARCHAR(36) NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    user_id VARCHAR(36) NOT NULL, -- Reference to user in user-auth-service
    email VARCHAR(255) NOT NULL,
    first_name VARCHAR(100),
    last_name VARCHAR(100),
    is_primary BOOLEAN NOT NULL DEFAULT FALSE, -- First admin is primary
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),

    UNIQUE(tenant_id, user_id)
);

CREATE INDEX idx_tenant_admins_tenant ON tenant_admins(tenant_id);
CREATE INDEX idx_tenant_admins_user ON tenant_admins(user_id);
CREATE INDEX idx_tenant_admins_email ON tenant_admins(email);

-- Provisioning tracking table (AC7: Provisioning completes within 2 minutes)
CREATE TABLE IF NOT EXISTS tenant_provisioning (
    id VARCHAR(36) PRIMARY KEY,
    tenant_id VARCHAR(36) NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    status VARCHAR(50) NOT NULL, -- pending, provisioning_database, creating_admin, setting_quota, sending_email, completed, failed
    current_step VARCHAR(255),
    progress_percentage INTEGER NOT NULL DEFAULT 0,
    error_message TEXT,
    started_at TIMESTAMP NOT NULL DEFAULT NOW(),
    completed_at TIMESTAMP,
    duration_seconds INTEGER, -- Must be < 120 seconds (AC7)

    CONSTRAINT valid_progress CHECK (progress_percentage >= 0 AND progress_percentage <= 100),
    CONSTRAINT valid_duration CHECK (duration_seconds IS NULL OR duration_seconds >= 0)
);

CREATE INDEX idx_tenant_provisioning_tenant ON tenant_provisioning(tenant_id);
CREATE INDEX idx_tenant_provisioning_status ON tenant_provisioning(status);
CREATE INDEX idx_tenant_provisioning_started ON tenant_provisioning(started_at DESC);

-- Storage files table (for quota tracking)
CREATE TABLE IF NOT EXISTS tenant_storage_files (
    id VARCHAR(36) PRIMARY KEY,
    tenant_id VARCHAR(36) NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    user_id VARCHAR(36) NOT NULL,
    filename VARCHAR(500) NOT NULL,
    file_path TEXT NOT NULL,
    file_size_bytes BIGINT NOT NULL,
    mime_type VARCHAR(100),
    checksum VARCHAR(64),
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMP
);

CREATE INDEX idx_storage_files_tenant ON tenant_storage_files(tenant_id);
CREATE INDEX idx_storage_files_user ON tenant_storage_files(user_id);
CREATE INDEX idx_storage_files_created ON tenant_storage_files(created_at DESC);
CREATE INDEX idx_storage_files_deleted ON tenant_storage_files(deleted_at) WHERE deleted_at IS NULL;

-- Setup link tokens table (AC4: Welcome email sent with setup link)
CREATE TABLE IF NOT EXISTS tenant_setup_tokens (
    id VARCHAR(36) PRIMARY KEY,
    tenant_id VARCHAR(36) NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    admin_id VARCHAR(36) NOT NULL REFERENCES tenant_admins(id) ON DELETE CASCADE,
    token VARCHAR(255) UNIQUE NOT NULL,
    expires_at TIMESTAMP NOT NULL,
    used_at TIMESTAMP,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),

    CONSTRAINT valid_expiry CHECK (expires_at > created_at)
);

CREATE INDEX idx_setup_tokens_token ON tenant_setup_tokens(token);
CREATE INDEX idx_setup_tokens_tenant ON tenant_setup_tokens(tenant_id);
CREATE INDEX idx_setup_tokens_expires ON tenant_setup_tokens(expires_at);

-- Schema version tracking
CREATE TABLE IF NOT EXISTS schema_migrations (
    version VARCHAR(50) PRIMARY KEY,
    applied_at TIMESTAMP NOT NULL DEFAULT NOW()
);

-- Insert default subscription tiers
INSERT INTO subscription_tiers (id, name, display_name, tier_level, storage_quota_bytes, max_users, max_courses, api_rate_limit, dedicated_database, custom_domain, priority_support, price_cents_monthly, features)
VALUES
    ('tier-free-001', 'free', 'Free', 0, 1073741824, 10, 5, 100, FALSE, FALSE, FALSE, 0, '{"api_access": true, "basic_analytics": true}'::jsonb),
    ('tier-basic-001', 'basic', 'Basic', 1, 10737418240, 50, 25, 500, FALSE, FALSE, FALSE, 1999, '{"api_access": true, "basic_analytics": true, "email_support": true}'::jsonb),
    ('tier-pro-001', 'professional', 'Professional', 2, 107374182400, 500, 100, 2000, TRUE, TRUE, FALSE, 9999, '{"api_access": true, "advanced_analytics": true, "email_support": true, "dedicated_database": true, "custom_domain": true, "sso_enabled": false}'::jsonb),
    ('tier-ent-001', 'enterprise', 'Enterprise', 3, 1099511627776, -1, -1, 10000, TRUE, TRUE, TRUE, 49999, '{"api_access": true, "advanced_analytics": true, "priority_support": true, "dedicated_database": true, "custom_domain": true, "sso_enabled": true, "webhook_integrations": true, "audit_logs": true}'::jsonb)
ON CONFLICT (id) DO NOTHING;

-- Record migration
INSERT INTO schema_migrations (version) VALUES ('001_init') ON CONFLICT DO NOTHING;
