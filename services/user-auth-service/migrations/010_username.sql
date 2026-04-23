-- W14.x: add `username` field to users for @mention resolution.
-- Backfill existing rows with the lowercase email local-part so all users
-- resolve out of the box. Uniqueness is enforced case-insensitively via the
-- LOWER() expression index (matches ResolveUsername lookup normalization).
--
-- Collision-safe backfill: when two emails share a local-part (e.g.
-- admin@a.com + admin@b.com), suffix duplicates with a row_number so the
-- unique index can be created. Lexicographic order of email wins the bare
-- form; later rows get -2, -3, etc. Admin can rename via API after migration.

ALTER TABLE users ADD COLUMN IF NOT EXISTS username VARCHAR(64);

WITH derived AS (
    SELECT
        id,
        LOWER(SPLIT_PART(email, '@', 1)) AS base,
        ROW_NUMBER() OVER (
            PARTITION BY LOWER(SPLIT_PART(email, '@', 1))
            ORDER BY email, id
        ) AS rn
    FROM users
    WHERE username IS NULL OR username = ''
)
UPDATE users u
SET username = CASE
    WHEN d.rn = 1 THEN d.base
    ELSE d.base || '-' || d.rn::text
END
FROM derived d
WHERE u.id = d.id;

ALTER TABLE users ALTER COLUMN username SET NOT NULL;

CREATE UNIQUE INDEX IF NOT EXISTS idx_users_username_lower
    ON users (LOWER(username));
