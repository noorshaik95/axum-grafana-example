-- Rollback Migration 002

DROP TRIGGER IF EXISTS trg_assignments_updated_at ON assignments;
DROP TRIGGER IF EXISTS trg_submissions_updated_at ON submissions;
DROP TRIGGER IF EXISTS trg_grades_updated_at ON grades;
DROP FUNCTION IF EXISTS update_updated_at_column();

DROP INDEX IF EXISTS idx_assignments_course_due;
DROP INDEX IF EXISTS idx_submissions_student_status;
DROP INDEX IF EXISTS idx_submissions_assignment_status;
DROP INDEX IF EXISTS idx_grades_student_status;
DROP INDEX IF EXISTS idx_grades_draft;
