-- Remove the instructor + student seed rows (R4 rollback).
DELETE FROM user_roles
WHERE user_id IN (
    SELECT id FROM users WHERE email IN ('instructor@slate.edu', 'student@slate.edu')
);

DELETE FROM users WHERE email IN ('instructor@slate.edu', 'student@slate.edu');
