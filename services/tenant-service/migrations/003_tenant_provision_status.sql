-- Migration 003: Add status_detail column to tenants_v2 for CONTRACTS.md tenant.provision-status.
-- Callers (seed-dev.sh, admin-fe) read `status` + `status_detail` from GET /api/tenants/:id
-- to distinguish transient "provisioning" from terminal "failed" with a readable reason.

ALTER TABLE tenants_v2
    ADD COLUMN IF NOT EXISTS status_detail TEXT NOT NULL DEFAULT '';

INSERT INTO schema_migrations (version) VALUES ('003_tenant_provision_status') ON CONFLICT DO NOTHING;
