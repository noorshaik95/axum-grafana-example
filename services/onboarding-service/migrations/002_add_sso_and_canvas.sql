-- W13 — onboarding-service extensions
-- W13.1 SSO config step + W13.2 Canvas LMS migration.
--
-- The authoritative state of each workflow lives inside Restate (durable
-- virtual-object state). These tables are the query-side mirror used by
-- the admin UI and by observability.

-- Onboarding → SSO config snapshot for admin UI listing.
ALTER TABLE tenants
    ADD COLUMN IF NOT EXISTS sso_provider VARCHAR(32),
    ADD COLUMN IF NOT EXISTS sso_config   JSONB;

COMMENT ON COLUMN tenants.sso_provider IS
    'W13.1 — "saml" | "oidc" | "google" | "microsoft". Set after configure_sso step resolves.';
COMMENT ON COLUMN tenants.sso_config IS
    'W13.1 — provider-specific config blob, captured from the admin SSO form.';

-- Canvas LMS migrations — one row per migration; Restate state is the
-- source of truth, this table is the readable mirror.
CREATE TABLE IF NOT EXISTS canvas_migrations (
    id                    VARCHAR(36) PRIMARY KEY,
    tenant_id             VARCHAR(36) REFERENCES tenants(id) ON DELETE CASCADE,
    canvas_base_url       VARCHAR(500) NOT NULL,
    status                VARCHAR(32)  NOT NULL DEFAULT 'not_started',
    last_imported_page    INTEGER      NOT NULL DEFAULT 0,
    total_courses_imported INTEGER     NOT NULL DEFAULT 0,
    total_users_matched   INTEGER      NOT NULL DEFAULT 0,
    total_users_created   INTEGER      NOT NULL DEFAULT 0,
    awakeable_id          VARCHAR(128),
    error                 TEXT,
    created_at            TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at            TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_canvas_migrations_tenant ON canvas_migrations(tenant_id);
CREATE INDEX IF NOT EXISTS idx_canvas_migrations_status ON canvas_migrations(status);

CREATE TRIGGER update_canvas_migrations_updated_at BEFORE UPDATE ON canvas_migrations
FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

INSERT INTO schema_migrations (version) VALUES ('002_add_sso_and_canvas')
ON CONFLICT (version) DO NOTHING;
