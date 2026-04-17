-- Metrics Service Database Schema
-- Analytics: event_log, student_progress, grade_stats

-- Enable TimescaleDB if available, otherwise use regular PostgreSQL
-- Note: TimescaleDB hypertables are optional — use regular tables if extension not available
DO $$
BEGIN
    CREATE EXTENSION IF NOT EXISTS timescaledb;
EXCEPTION WHEN OTHERS THEN
    RAISE NOTICE 'TimescaleDB extension not available, using regular tables';
END $$;

-- Event log for all platform events (time-series)
CREATE TABLE IF NOT EXISTS event_log (
    id UUID DEFAULT gen_random_uuid(),
    time TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    tenant_id UUID NOT NULL,
    event_type VARCHAR(100) NOT NULL,
    user_id UUID,
    course_id UUID,
    assignment_id UUID,
    metadata JSONB DEFAULT '{}'
);

CREATE INDEX IF NOT EXISTS idx_event_log_tenant_type ON event_log(tenant_id, event_type, time DESC);
CREATE INDEX IF NOT EXISTS idx_event_log_user ON event_log(tenant_id, user_id, time DESC);
CREATE INDEX IF NOT EXISTS idx_event_log_time ON event_log(time DESC);

-- Try to create hypertable if TimescaleDB is available
DO $$
BEGIN
    PERFORM create_hypertable('event_log', 'time', if_not_exists => TRUE);
EXCEPTION WHEN OTHERS THEN
    RAISE NOTICE 'Skipping hypertable creation for event_log';
END $$;

-- Student progress tracking (one row per student per course)
CREATE TABLE IF NOT EXISTS student_progress (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    student_id UUID NOT NULL,
    course_id UUID NOT NULL,
    completion_pct DECIMAL(5,2) DEFAULT 0,
    time_on_task_minutes INT DEFAULT 0,
    lessons_completed INT DEFAULT 0,
    last_activity_at TIMESTAMPTZ,
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(tenant_id, student_id, course_id)
);

CREATE INDEX IF NOT EXISTS idx_student_progress_tenant ON student_progress(tenant_id);
CREATE INDEX IF NOT EXISTS idx_student_progress_student ON student_progress(tenant_id, student_id);

-- Pre-computed grade statistics per course/assignment
CREATE TABLE IF NOT EXISTS grade_stats (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    course_id UUID NOT NULL,
    assignment_id UUID,
    mean_score DECIMAL(5,2),
    median_score DECIMAL(5,2),
    p25 DECIMAL(5,2),
    p75 DECIMAL(5,2),
    std_dev DECIMAL(5,2),
    student_count INT,
    computed_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_grade_stats_course ON grade_stats(tenant_id, course_id, computed_at DESC);
CREATE INDEX IF NOT EXISTS idx_grade_stats_assignment ON grade_stats(tenant_id, course_id, assignment_id, computed_at DESC);

-- Schema version tracking
CREATE TABLE IF NOT EXISTS schema_migrations (
    version VARCHAR(50) PRIMARY KEY,
    applied_at TIMESTAMP NOT NULL DEFAULT NOW()
);

INSERT INTO schema_migrations (version) VALUES ('001_create_metrics') ON CONFLICT DO NOTHING;
