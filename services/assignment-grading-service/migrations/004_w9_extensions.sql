-- Migration 004: W9 assignment-grading extensions
-- - rubric_rows table (W9.1)
-- - submissions.is_draft (W9.2)
-- - submission_test_results + assignments.tests_file_path (W9.3)
-- - assignment_attachments (W9.6)

-- ============================================================
-- 1. Rubric rows (W9.1)
-- ============================================================
CREATE TABLE IF NOT EXISTS rubric_rows (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    assignment_id UUID NOT NULL REFERENCES assignments(id) ON DELETE CASCADE,
    title VARCHAR(500) NOT NULL,
    max_points DECIMAL(10, 2) NOT NULL CHECK (max_points > 0),
    sort_order INT NOT NULL DEFAULT 0,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_rubric_rows_assignment_id ON rubric_rows(assignment_id);
CREATE INDEX IF NOT EXISTS idx_rubric_rows_assignment_order ON rubric_rows(assignment_id, sort_order);

-- ============================================================
-- 2. Draft submissions (W9.2)
-- ============================================================
ALTER TABLE submissions ADD COLUMN IF NOT EXISTS is_draft BOOLEAN NOT NULL DEFAULT FALSE;

-- Allow 'draft' in submissions.status
ALTER TABLE submissions DROP CONSTRAINT IF EXISTS submissions_status_check;
ALTER TABLE submissions ADD CONSTRAINT submissions_status_check
    CHECK (status IN ('submitted', 'graded', 'returned', 'late', 'draft'));

CREATE INDEX IF NOT EXISTS idx_submissions_draft ON submissions(assignment_id, student_id, is_draft);

-- ============================================================
-- 3. Auto-test integration (W9.3)
-- ============================================================
ALTER TABLE assignments ADD COLUMN IF NOT EXISTS tests_file_path VARCHAR(1000);

CREATE TABLE IF NOT EXISTS submission_test_results (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    submission_id UUID NOT NULL REFERENCES submissions(id) ON DELETE CASCADE,
    assignment_id UUID NOT NULL REFERENCES assignments(id) ON DELETE CASCADE,
    test_name VARCHAR(500) NOT NULL,
    passed BOOLEAN NOT NULL,
    output TEXT,
    duration_ms INT NOT NULL DEFAULT 0,
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_submission_test_results_submission ON submission_test_results(submission_id);
CREATE INDEX IF NOT EXISTS idx_submission_test_results_assignment ON submission_test_results(assignment_id);

-- ============================================================
-- 4. Starter code / attachments (W9.6)
-- ============================================================
CREATE TABLE IF NOT EXISTS assignment_attachments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    assignment_id UUID NOT NULL REFERENCES assignments(id) ON DELETE CASCADE,
    file_path VARCHAR(1000) NOT NULL,
    file_name VARCHAR(500) NOT NULL,
    content_type VARCHAR(255),
    size_bytes BIGINT NOT NULL DEFAULT 0,
    kind VARCHAR(50) NOT NULL DEFAULT 'attachment',
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

ALTER TABLE assignment_attachments DROP CONSTRAINT IF EXISTS assignment_attachments_kind_check;
ALTER TABLE assignment_attachments ADD CONSTRAINT assignment_attachments_kind_check
    CHECK (kind IN ('attachment', 'starter_code', 'instructions'));

CREATE INDEX IF NOT EXISTS idx_assignment_attachments_assignment ON assignment_attachments(assignment_id);
