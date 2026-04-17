-- Migration 003: Extend schema for full grading engine
-- Adds tenant_id, rubric, assignment_type, grading_rules table, and updates existing tables

-- ============================================================
-- 1. Extend assignments table
-- ============================================================
ALTER TABLE assignments ADD COLUMN IF NOT EXISTS tenant_id UUID;
ALTER TABLE assignments ADD COLUMN IF NOT EXISTS instructor_id UUID;
ALTER TABLE assignments ADD COLUMN IF NOT EXISTS rubric JSONB;
ALTER TABLE assignments ADD COLUMN IF NOT EXISTS assignment_type VARCHAR(50);
ALTER TABLE assignments ADD COLUMN IF NOT EXISTS max_file_size_mb INT DEFAULT 50;
ALTER TABLE assignments ADD COLUMN IF NOT EXISTS allowed_file_types TEXT[];
ALTER TABLE assignments ADD COLUMN IF NOT EXISTS is_deleted BOOLEAN DEFAULT false;

-- Add check constraint for assignment_type
ALTER TABLE assignments ADD CONSTRAINT chk_assignment_type
    CHECK (assignment_type IS NULL OR assignment_type IN ('exam', 'homework', 'quiz', 'project'));

-- Index for tenant + course lookups
CREATE INDEX IF NOT EXISTS idx_assignments_tenant_course ON assignments(tenant_id, course_id);
-- Filter out soft-deleted assignments
CREATE INDEX IF NOT EXISTS idx_assignments_not_deleted ON assignments(course_id) WHERE is_deleted = false;

-- ============================================================
-- 2. Extend submissions table
-- ============================================================
ALTER TABLE submissions ADD COLUMN IF NOT EXISTS tenant_id UUID;
ALTER TABLE submissions ADD COLUMN IF NOT EXISTS file_urls TEXT[];

-- Update status check constraint to allow 'late' status
ALTER TABLE submissions DROP CONSTRAINT IF EXISTS submissions_status_check;
ALTER TABLE submissions ADD CONSTRAINT submissions_status_check
    CHECK (status IN ('submitted', 'graded', 'returned', 'late'));

-- ============================================================
-- 3. Extend grades table
-- ============================================================
ALTER TABLE grades ADD COLUMN IF NOT EXISTS tenant_id UUID;
ALTER TABLE grades ADD COLUMN IF NOT EXISTS course_id UUID;
ALTER TABLE grades ADD COLUMN IF NOT EXISTS max_score DECIMAL(5,2);
ALTER TABLE grades ADD COLUMN IF NOT EXISTS percentage DECIMAL(5,2);
ALTER TABLE grades ADD COLUMN IF NOT EXISTS letter_grade VARCHAR(5);
ALTER TABLE grades ADD COLUMN IF NOT EXISTS rubric_scores JSONB;
ALTER TABLE grades ADD COLUMN IF NOT EXISTS override_justification TEXT;
ALTER TABLE grades ADD COLUMN IF NOT EXISTS percentile DECIMAL(5,2);

CREATE INDEX IF NOT EXISTS idx_grades_tenant_course ON grades(tenant_id, course_id);
CREATE INDEX IF NOT EXISTS idx_grades_tenant_student ON grades(tenant_id, student_id);

-- ============================================================
-- 4. Create grading_rules table
-- ============================================================
CREATE TABLE IF NOT EXISTS grading_rules (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    course_id UUID,
    assignment_type VARCHAR(50),
    weight DECIMAL(5,2),
    late_penalty_per_day DECIMAL(5,2) DEFAULT 0,
    max_late_penalty DECIMAL(5,2) DEFAULT 0,
    grade_scale JSONB DEFAULT '[{"grade":"A","min":90},{"grade":"B","min":80},{"grade":"C","min":70},{"grade":"D","min":60},{"grade":"F","min":0}]',
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_grading_rules_tenant ON grading_rules(tenant_id);
CREATE INDEX IF NOT EXISTS idx_grading_rules_tenant_course ON grading_rules(tenant_id, course_id);

-- Apply updated_at trigger to grading_rules if trigger function exists
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM pg_proc WHERE proname = 'update_updated_at_column') THEN
        -- No updated_at column on grading_rules per spec
        NULL;
    END IF;
END
$$;
