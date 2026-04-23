-- Migration 012: Seed default instructor + student users (R4)
-- Companion to 008's admin seed. Provides deterministic creds for the
-- provider + student portals so QA + live demos aren't blocked on
-- registration while the flow is hardened.
--
-- Credentials (development only — rotate before prod):
--   instructor@slate.edu / Instructor@123456
--   student@slate.edu    / Student@123456
--
-- Usernames match migration 010's backfill convention (lowercase local-part).

-- Instructor seed
INSERT INTO users (id, email, password_hash, first_name, last_name, is_active, auth_method, username)
VALUES (
    gen_random_uuid(),
    'instructor@slate.edu',
    '$2a$10$nsM3G3Yk.wOQizPDw5NhG.fBB3Bo2QGplNlB4Y4wvSPFARBebLVIC',
    'Default',
    'Instructor',
    true,
    'normal',
    'instructor'
)
ON CONFLICT (email) DO NOTHING;

INSERT INTO user_roles (user_id, role_id, assigned_at)
SELECT u.id, r.id, NOW()
FROM users u, roles r
WHERE u.email = 'instructor@slate.edu' AND r.name = 'instructor'
ON CONFLICT (user_id, role_id) DO NOTHING;

-- Student seed
INSERT INTO users (id, email, password_hash, first_name, last_name, is_active, auth_method, username)
VALUES (
    gen_random_uuid(),
    'student@slate.edu',
    '$2a$10$Fp560T2pjBeyjVdvZ8wKFe/ukq/aa4ROnAqv7srFY7pYU3i4xnhNa',
    'Default',
    'Student',
    true,
    'normal',
    'student'
)
ON CONFLICT (email) DO NOTHING;

INSERT INTO user_roles (user_id, role_id, assigned_at)
SELECT u.id, r.id, NOW()
FROM users u, roles r
WHERE u.email = 'student@slate.edu' AND r.name = 'student'
ON CONFLICT (user_id, role_id) DO NOTHING;
