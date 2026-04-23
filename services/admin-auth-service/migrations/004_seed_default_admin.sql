-- Seed a default platform admin so fresh local stacks can log in.
-- Password: Admin@123456 (bcrypt cost 10). Matches scripts/db-setup.sh docs.
-- Idempotent: only inserts if no row with this email exists.

INSERT INTO platform_admins (email, password_hash, full_name, roles, disabled)
SELECT
    'admin@slate.edu',
    '$2a$10$y4blt.jzMVvro6UeogKBHOaj.dF6/vn7vpwMOHoKuQ.NlxF0dGpdy',
    'Platform Admin',
    ARRAY['superadmin'],
    false
WHERE NOT EXISTS (
    SELECT 1 FROM platform_admins WHERE email = 'admin@slate.edu'
);
