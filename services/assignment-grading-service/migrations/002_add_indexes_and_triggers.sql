-- Migration 002: Add updated_at triggers and additional indexes for assignment_grading DB

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

-- Apply updated_at triggers
CREATE TRIGGER trg_assignments_updated_at
    BEFORE UPDATE ON assignments
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER trg_submissions_updated_at
    BEFORE UPDATE ON submissions
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER trg_grades_updated_at
    BEFORE UPDATE ON grades
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- ============================================================
-- 2. Additional composite indexes for common query patterns
-- ============================================================

-- Instructor view: list assignments for a course ordered by due date
CREATE INDEX IF NOT EXISTS idx_assignments_course_due ON assignments(course_id, due_date);

-- Student view: find all submissions by a student across courses
CREATE INDEX IF NOT EXISTS idx_submissions_student_status ON submissions(student_id, status);

-- Grading view: find ungraded submissions for an assignment
CREATE INDEX IF NOT EXISTS idx_submissions_assignment_status ON submissions(assignment_id, status);

-- Grade report: grades by student across all assignments, filtered by status
CREATE INDEX IF NOT EXISTS idx_grades_student_status ON grades(student_id, status);

-- Partial index for pending/draft grades (grading queue)
CREATE INDEX IF NOT EXISTS idx_grades_draft ON grades(assignment_id, graded_by) WHERE status = 'draft';
