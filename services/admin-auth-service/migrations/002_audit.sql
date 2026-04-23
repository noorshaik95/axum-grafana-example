-- platform_audit: append-only ledger of every admin action.
-- Every AdminAuthService RPC that mutates state or authenticates an admin
-- inserts one row here and emits the same payload on Kafka audit.admin_action.

CREATE TABLE IF NOT EXISTS platform_audit (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    actor_id    VARCHAR(36) NOT NULL,
    action      VARCHAR(100) NOT NULL,
    target_id   VARCHAR(36),
    target_type VARCHAR(50),
    metadata    JSONB,
    request_id  VARCHAR(64),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_platform_audit_actor ON platform_audit (actor_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_platform_audit_action ON platform_audit (action, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_platform_audit_target ON platform_audit (target_id, created_at DESC);
