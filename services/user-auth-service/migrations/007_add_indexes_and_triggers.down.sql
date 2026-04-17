-- Rollback Migration 007

DROP TRIGGER IF EXISTS trg_users_updated_at ON users;
DROP TRIGGER IF EXISTS trg_roles_updated_at ON roles;
DROP TRIGGER IF EXISTS trg_oauth_providers_updated_at ON oauth_providers;
DROP TRIGGER IF EXISTS trg_user_mfa_updated_at ON user_mfa;
DROP TRIGGER IF EXISTS trg_user_groups_updated_at ON user_groups;
DROP TRIGGER IF EXISTS trg_parent_child_accounts_updated_at ON parent_child_accounts;
DROP FUNCTION IF EXISTS update_updated_at_column();

DROP INDEX IF EXISTS idx_saml_sessions_config_id;
DROP INDEX IF EXISTS idx_users_last_name_first_name;
DROP INDEX IF EXISTS idx_users_org_active;
DROP INDEX IF EXISTS idx_group_members_role;
