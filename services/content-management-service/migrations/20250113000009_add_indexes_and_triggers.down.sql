-- Rollback Migration 009

DROP TRIGGER IF EXISTS trg_modules_updated_at ON modules;
DROP TRIGGER IF EXISTS trg_lessons_updated_at ON lessons;
DROP TRIGGER IF EXISTS trg_resources_updated_at ON resources;
DROP TRIGGER IF EXISTS trg_progress_tracking_updated_at ON progress_tracking;
DROP FUNCTION IF EXISTS update_updated_at_column();

DROP INDEX IF EXISTS idx_modules_created_by;
DROP INDEX IF EXISTS idx_resources_lesson_content_type;
DROP INDEX IF EXISTS idx_progress_student_resource;
DROP INDEX IF EXISTS idx_download_student_resource;
DROP INDEX IF EXISTS idx_upload_sessions_active;
