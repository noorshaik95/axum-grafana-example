-- W2.5: roles.permissions column + billing role + seed matrix.
-- Additive only; existing rows keep their defaults unless explicitly updated.

ALTER TABLE platform_roles
    ADD COLUMN IF NOT EXISTS permissions TEXT[] NOT NULL DEFAULT ARRAY[]::TEXT[];

INSERT INTO platform_roles (name, description, permissions) VALUES
    ('billing', 'Billing operations access', ARRAY['billing:read','billing:write','admin:read'])
ON CONFLICT (name) DO NOTHING;

-- Seed permissions onto the roles already created by migration 001.
-- Only updates rows that still carry the default empty array, so re-running
-- this migration after a permission tweak won't clobber curated values.
UPDATE platform_roles SET permissions = ARRAY['*']
    WHERE name = 'superadmin' AND permissions = '{}';

UPDATE platform_roles SET permissions = ARRAY['admin:read','impersonate','audit:read']
    WHERE name = 'support'    AND permissions = '{}';

UPDATE platform_roles SET permissions = ARRAY['admin:read','audit:read']
    WHERE name = 'readonly'   AND permissions = '{}';
