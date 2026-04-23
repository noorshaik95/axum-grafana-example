-- Video resume positions: persisted fallback for Redis position cache.
-- Keyed by (user_id, content_id); latest position wins on upsert.
CREATE TABLE IF NOT EXISTS video_positions (
    user_id UUID NOT NULL,
    content_id UUID NOT NULL,
    position_seconds INTEGER NOT NULL DEFAULT 0,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (user_id, content_id)
);

CREATE INDEX IF NOT EXISTS idx_video_positions_content ON video_positions(content_id);
