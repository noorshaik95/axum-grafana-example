CREATE TABLE IF NOT EXISTS threads (
    id UUID PRIMARY KEY,
    tenant_id UUID NOT NULL,
    course_id UUID NOT NULL,
    title VARCHAR(500) NOT NULL,
    created_by UUID NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_activity_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    reply_count INT NOT NULL DEFAULT 0
);

CREATE INDEX IF NOT EXISTS idx_threads_course ON threads(tenant_id, course_id, last_activity_at DESC);
CREATE INDEX IF NOT EXISTS idx_threads_creator ON threads(tenant_id, created_by);

CREATE TABLE IF NOT EXISTS posts (
    id UUID PRIMARY KEY,
    tenant_id UUID NOT NULL,
    thread_id UUID NOT NULL REFERENCES threads(id) ON DELETE CASCADE,
    author_id UUID NOT NULL,
    author_role VARCHAR(32) NOT NULL DEFAULT 'student',
    content TEXT NOT NULL,
    parent_post_id UUID,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    edited_at TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_posts_thread ON posts(thread_id, created_at);
CREATE INDEX IF NOT EXISTS idx_posts_author ON posts(tenant_id, author_id);
CREATE INDEX IF NOT EXISTS idx_posts_instructor_activity
    ON posts(thread_id, created_at DESC)
    WHERE author_role = 'instructor';

CREATE TABLE IF NOT EXISTS mentions (
    id UUID PRIMARY KEY,
    tenant_id UUID NOT NULL,
    post_id UUID NOT NULL REFERENCES posts(id) ON DELETE CASCADE,
    mentioned_user_id UUID NOT NULL,
    seen_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_mentions_user ON mentions(tenant_id, mentioned_user_id, seen_at);
CREATE UNIQUE INDEX IF NOT EXISTS idx_mentions_post_user ON mentions(post_id, mentioned_user_id);

CREATE TABLE IF NOT EXISTS inbox_items (
    id UUID PRIMARY KEY,
    tenant_id UUID NOT NULL,
    user_id UUID NOT NULL,
    type VARCHAR(50) NOT NULL,
    reference_id UUID NOT NULL,
    seen_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_inbox_user ON inbox_items(tenant_id, user_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_inbox_unseen ON inbox_items(tenant_id, user_id, seen_at, created_at DESC);

CREATE TABLE IF NOT EXISTS instructor_courses (
    tenant_id UUID NOT NULL,
    instructor_id UUID NOT NULL,
    course_id UUID NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (tenant_id, instructor_id, course_id)
);
