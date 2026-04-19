-- W14.3: audit_events table for authentication audit trail.
-- Appended to on: login, logout, failed_login, password_change, mfa_change,
-- impersonation_start, impersonation_end, sso_login.

CREATE TABLE IF NOT EXISTS audit_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    actor_id VARCHAR(36),
    actor_type VARCHAR(20),
    action VARCHAR(100) NOT NULL,
    target_id VARCHAR(36),
    target_type VARCHAR(50),
    metadata JSONB,
    ip_address INET,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_audit_events_actor ON audit_events(actor_id);
CREATE INDEX IF NOT EXISTS idx_audit_events_created ON audit_events(created_at);
CREATE INDEX IF NOT EXISTS idx_audit_events_action ON audit_events(action);
