-- W14.x: add `username` field to users for @mention resolution.
-- Backfill existing rows with the lowercase email local-part so all users
-- resolve out of the box. Uniqueness is enforced case-insensitively via the
-- LOWER() expression index (matches ResolveUsername lookup normalization).

ALTER TABLE users ADD COLUMN IF NOT EXISTS username VARCHAR(64);

UPDATE users
SET username = LOWER(SPLIT_PART(email, '@', 1))
WHERE username IS NULL OR username = '';

ALTER TABLE users ALTER COLUMN username SET NOT NULL;

CREATE UNIQUE INDEX IF NOT EXISTS idx_users_username_lower
    ON users (LOWER(username));
