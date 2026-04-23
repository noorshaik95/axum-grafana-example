-- platform_admins: root account table for Slate platform operators.
-- schema is hardcoded; admin-auth-service is single-tenant by design.

CREATE EXTENSION IF NOT EXISTS "pgcrypto";

CREATE TABLE IF NOT EXISTS platform_roles (
    name        VARCHAR(64) PRIMARY KEY,
    description TEXT,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

INSERT INTO platform_roles (name, description) VALUES
    ('superadmin', 'Full platform access including impersonation'),
    ('support',    'Support access including impersonation, read-most'),
    ('readonly',   'Read-only platform access')
ON CONFLICT (name) DO NOTHING;

CREATE TABLE IF NOT EXISTS platform_admins (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email           VARCHAR(255) UNIQUE NOT NULL,
    password_hash   VARCHAR(255) NOT NULL,
    full_name       VARCHAR(255) NOT NULL DEFAULT '',
    roles           TEXT[] NOT NULL DEFAULT ARRAY['readonly']::TEXT[],
    disabled        BOOLEAN NOT NULL DEFAULT FALSE,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_login_at   TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_platform_admins_email ON platform_admins (email);
