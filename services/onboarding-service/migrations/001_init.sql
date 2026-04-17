-- Onboarding Service Database Schema
-- Multi-tenant bulk user onboarding with compliance features

-- Enable UUID extension
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Tenants table for multi-tenancy
CREATE TABLE IF NOT EXISTS tenants (
    id VARCHAR(36) PRIMARY KEY DEFAULT uuid_generate_v4()::TEXT,
    name VARCHAR(255) NOT NULL,
    domain VARCHAR(255) UNIQUE NOT NULL,
    is_active BOOLEAN DEFAULT TRUE,
    settings JSONB DEFAULT '{}',
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP
);

CREATE INDEX idx_tenants_domain ON tenants(domain) WHERE deleted_at IS NULL;
CREATE INDEX idx_tenants_is_active ON tenants(is_active) WHERE deleted_at IS NULL;

-- Integration configurations table
CREATE TABLE IF NOT EXISTS integration_configs (
    id VARCHAR(36) PRIMARY KEY DEFAULT uuid_generate_v4()::TEXT,
    tenant_id VARCHAR(36) NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    integration_type VARCHAR(50) NOT NULL, -- ldap, saml, google, microsoft, csv, api
    name VARCHAR(255) NOT NULL,
    config JSONB NOT NULL, -- Encrypted configuration details
    is_active BOOLEAN DEFAULT TRUE,
    last_sync_at TIMESTAMP,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_integration_configs_tenant ON integration_configs(tenant_id);
CREATE INDEX idx_integration_configs_type ON integration_configs(integration_type);

-- Onboarding jobs table for bulk operations
CREATE TABLE IF NOT EXISTS onboarding_jobs (
    id VARCHAR(36) PRIMARY KEY DEFAULT uuid_generate_v4()::TEXT,
    tenant_id VARCHAR(36) NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    source_type VARCHAR(50) NOT NULL, -- csv, ldap, saml, google, microsoft, api
    source_reference VARCHAR(500), -- File path, API endpoint, etc.
    status VARCHAR(50) NOT NULL DEFAULT 'pending', -- pending, processing, completed, failed, cancelled
    total_users INTEGER DEFAULT 0,
    processed_users INTEGER DEFAULT 0,
    successful_users INTEGER DEFAULT 0,
    failed_users INTEGER DEFAULT 0,
    error_summary JSONB,
    created_by VARCHAR(36), -- User ID who created the job
    started_at TIMESTAMP,
    completed_at TIMESTAMP,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_onboarding_jobs_tenant ON onboarding_jobs(tenant_id);
CREATE INDEX idx_onboarding_jobs_status ON onboarding_jobs(status);
CREATE INDEX idx_onboarding_jobs_created_at ON onboarding_jobs(created_at DESC);

-- Onboarding tasks table for individual user processing
CREATE TABLE IF NOT EXISTS onboarding_tasks (
    id VARCHAR(36) PRIMARY KEY DEFAULT uuid_generate_v4()::TEXT,
    job_id VARCHAR(36) NOT NULL REFERENCES onboarding_jobs(id) ON DELETE CASCADE,
    tenant_id VARCHAR(36) NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    email VARCHAR(255) NOT NULL,
    first_name VARCHAR(100),
    last_name VARCHAR(100),
    role VARCHAR(50) NOT NULL, -- student, instructor, staff, admin
    student_id VARCHAR(50),
    department VARCHAR(100),
    course_codes TEXT[], -- Array of course codes
    graduation_year INTEGER,
    phone VARCHAR(20),
    preferred_language VARCHAR(10) DEFAULT 'en',
    custom_fields JSONB DEFAULT '{}',
    status VARCHAR(50) NOT NULL DEFAULT 'pending', -- pending, processing, completed, failed, skipped
    user_id VARCHAR(36), -- Created user ID after successful onboarding
    retry_count INTEGER DEFAULT 0,
    error_message TEXT,
    error_details JSONB,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    processed_at TIMESTAMP
);

CREATE INDEX idx_onboarding_tasks_job ON onboarding_tasks(job_id);
CREATE INDEX idx_onboarding_tasks_tenant ON onboarding_tasks(tenant_id);
CREATE INDEX idx_onboarding_tasks_status ON onboarding_tasks(status);
CREATE INDEX idx_onboarding_tasks_email ON onboarding_tasks(tenant_id, email);
CREATE INDEX idx_onboarding_tasks_user ON onboarding_tasks(user_id) WHERE user_id IS NOT NULL;

-- Audit logs table for FERPA/GDPR compliance (immutable)
CREATE TABLE IF NOT EXISTS onboarding_audit_logs (
    id VARCHAR(36) PRIMARY KEY DEFAULT uuid_generate_v4()::TEXT,
    tenant_id VARCHAR(36) NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    job_id VARCHAR(36) REFERENCES onboarding_jobs(id) ON DELETE CASCADE,
    task_id VARCHAR(36) REFERENCES onboarding_tasks(id) ON DELETE CASCADE,
    event_type VARCHAR(100) NOT NULL, -- job_created, job_started, task_processed, user_created, etc.
    event_data JSONB NOT NULL,
    performed_by VARCHAR(36), -- User ID who performed the action
    ip_address VARCHAR(45),
    user_agent TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Immutable audit logs - no updates or deletes allowed
CREATE INDEX idx_audit_logs_tenant ON onboarding_audit_logs(tenant_id);
CREATE INDEX idx_audit_logs_job ON onboarding_audit_logs(job_id);
CREATE INDEX idx_audit_logs_task ON onboarding_audit_logs(task_id);
CREATE INDEX idx_audit_logs_event_type ON onboarding_audit_logs(event_type);
CREATE INDEX idx_audit_logs_created_at ON onboarding_audit_logs(created_at DESC);

-- Prevent updates and deletes on audit logs
CREATE OR REPLACE FUNCTION prevent_audit_log_modification()
RETURNS TRIGGER AS $$
BEGIN
    RAISE EXCEPTION 'Audit logs are immutable and cannot be modified or deleted';
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER prevent_audit_log_update
BEFORE UPDATE ON onboarding_audit_logs
FOR EACH ROW EXECUTE FUNCTION prevent_audit_log_modification();

CREATE TRIGGER prevent_audit_log_delete
BEFORE DELETE ON onboarding_audit_logs
FOR EACH ROW EXECUTE FUNCTION prevent_audit_log_modification();

-- Progress tracking table for real-time updates
CREATE TABLE IF NOT EXISTS job_progress (
    job_id VARCHAR(36) PRIMARY KEY REFERENCES onboarding_jobs(id) ON DELETE CASCADE,
    current_stage VARCHAR(100) NOT NULL, -- validation, user_creation, role_assignment, etc.
    progress_percentage DECIMAL(5,2) DEFAULT 0.00,
    estimated_completion_time TIMESTAMP,
    current_task_id VARCHAR(36),
    metrics JSONB DEFAULT '{}',
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_job_progress_updated_at ON job_progress(updated_at DESC);

-- Schema migrations tracking
CREATE TABLE IF NOT EXISTS schema_migrations (
    version VARCHAR(50) PRIMARY KEY,
    applied_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Insert initial migration version
INSERT INTO schema_migrations (version) VALUES ('001_init') ON CONFLICT (version) DO NOTHING;

-- Function to update updated_at timestamp
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Triggers to auto-update updated_at
CREATE TRIGGER update_tenants_updated_at BEFORE UPDATE ON tenants
FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_integration_configs_updated_at BEFORE UPDATE ON integration_configs
FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_onboarding_jobs_updated_at BEFORE UPDATE ON onboarding_jobs
FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_onboarding_tasks_updated_at BEFORE UPDATE ON onboarding_tasks
FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- Insert default tenant for development
INSERT INTO tenants (id, name, domain, settings) VALUES
('00000000-0000-0000-0000-000000000001', 'Default University', 'default.edu',
 '{"max_users": 100000, "retention_years": 7, "features": ["bulk_import", "ldap", "saml", "sso"]}'::JSONB)
ON CONFLICT (id) DO NOTHING;

-- Comments for documentation
COMMENT ON TABLE tenants IS 'Multi-tenant organizations using the onboarding service';
COMMENT ON TABLE integration_configs IS 'Configuration for various integration types (LDAP, SAML, etc.)';
COMMENT ON TABLE onboarding_jobs IS 'Bulk onboarding operations tracking';
COMMENT ON TABLE onboarding_tasks IS 'Individual user onboarding tasks within a job';
COMMENT ON TABLE onboarding_audit_logs IS 'Immutable audit trail for FERPA/GDPR compliance (7-year retention)';
COMMENT ON TABLE job_progress IS 'Real-time progress tracking for WebSocket updates';
