-- Migration 007: Add missing indexes and updated_at triggers for userauth DB

-- ============================================================
-- 1. Add updated_at trigger function (reusable across tables)
-- ============================================================
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Apply updated_at triggers to all tables with updated_at column
CREATE TRIGGER trg_users_updated_at
    BEFORE UPDATE ON users
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER trg_roles_updated_at
    BEFORE UPDATE ON roles
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER trg_oauth_providers_updated_at
    BEFORE UPDATE ON oauth_providers
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER trg_user_mfa_updated_at
    BEFORE UPDATE ON user_mfa
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER trg_user_groups_updated_at
    BEFORE UPDATE ON user_groups
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER trg_parent_child_accounts_updated_at
    BEFORE UPDATE ON parent_child_accounts
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- ============================================================
-- 2. Add missing indexes for common query patterns
-- ============================================================

-- Composite index for SAML session lookups by config
CREATE INDEX IF NOT EXISTS idx_saml_sessions_config_id ON saml_sessions(saml_config_id);

-- Composite index for user search by name (admin user listings)
CREATE INDEX IF NOT EXISTS idx_users_last_name_first_name ON users(last_name, first_name);

-- Index for active user filtering combined with org
CREATE INDEX IF NOT EXISTS idx_users_org_active ON users(organization_id, is_active) WHERE is_active = true;

-- Index for group member role lookups
CREATE INDEX IF NOT EXISTS idx_group_members_role ON group_members(role);
