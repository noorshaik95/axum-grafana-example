-- Migration 008: Seed default roles and admin user for Slate LMS

-- Upsert LMS-specific roles with correct permissions
INSERT INTO roles (id, name, description, permissions) VALUES
    (gen_random_uuid(), 'student', 'Student role', ARRAY['courses:read', 'assignments:submit', 'grades:read', 'profile:read', 'profile:update']),
    (gen_random_uuid(), 'instructor', 'Instructor role', ARRAY['courses:write', 'assignments:create', 'assignments:grade', 'grades:write', 'students:read', 'profile:read', 'profile:update']),
    (gen_random_uuid(), 'admin', 'Administrator role', ARRAY['*'])
ON CONFLICT (name) DO UPDATE SET
    description = EXCLUDED.description,
    permissions = EXCLUDED.permissions,
    updated_at = NOW();

-- Insert default admin user (password: Admin@123456)
INSERT INTO users (id, email, password_hash, first_name, last_name, is_active, auth_method)
VALUES (
    gen_random_uuid(),
    'admin@slate.edu',
    '$2b$10$VrRGtMhC/9hUf9IvGHFrgOzgRgKB4qzAEWnGU4nGmH0F85OOS3.km',
    'System',
    'Administrator',
    true,
    'normal'
)
ON CONFLICT (email) DO NOTHING;

-- Assign admin role to the default admin user
INSERT INTO user_roles (user_id, role_id, assigned_at)
SELECT u.id, r.id, NOW()
FROM users u, roles r
WHERE u.email = 'admin@slate.edu' AND r.name = 'admin'
ON CONFLICT (user_id, role_id) DO NOTHING;
