-- Migration 009: Add updated_at triggers and additional indexes for CMS DB

-- ============================================================
-- 1. Create updated_at trigger function
-- ============================================================
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Apply updated_at triggers to tables with updated_at column
CREATE TRIGGER trg_modules_updated_at
    BEFORE UPDATE ON modules
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER trg_lessons_updated_at
    BEFORE UPDATE ON lessons
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER trg_resources_updated_at
    BEFORE UPDATE ON resources
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER trg_progress_tracking_updated_at
    BEFORE UPDATE ON progress_tracking
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- ============================================================
-- 2. Additional indexes for common query patterns
-- ============================================================

-- Instructor view: modules created by a specific user
CREATE INDEX IF NOT EXISTS idx_modules_created_by ON modules(created_by);

-- Content search: resources by content type within a lesson
CREATE INDEX IF NOT EXISTS idx_resources_lesson_content_type ON resources(lesson_id, content_type);

-- Student progress: find all progress for a student across resources
CREATE INDEX IF NOT EXISTS idx_progress_student_resource ON progress_tracking(student_id, resource_id);

-- Analytics: download tracking by student and resource
CREATE INDEX IF NOT EXISTS idx_download_student_resource ON download_tracking(student_id, resource_id);

-- Partial index for active upload sessions
CREATE INDEX IF NOT EXISTS idx_upload_sessions_active ON upload_sessions(user_id, status) WHERE status = 'in_progress';
