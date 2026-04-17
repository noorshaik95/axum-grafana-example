-- Rollback Migration 008: Remove seeded data

DELETE FROM user_roles
WHERE user_id IN (SELECT id FROM users WHERE email = 'admin@slate.edu')
  AND role_id IN (SELECT id FROM roles WHERE name = 'admin');

DELETE FROM users WHERE email = 'admin@slate.edu';
