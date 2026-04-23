package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"slate/services/assignment-grading-service/internal/models"

	"github.com/google/uuid"
)

// TestResultRepository persists auto-test outcomes per submission (W9.3).
type TestResultRepository interface {
	Create(ctx context.Context, r *models.SubmissionTestResult) error
	CreateBatch(ctx context.Context, results []*models.SubmissionTestResult) error
	ListBySubmission(ctx context.Context, submissionID string) ([]*models.SubmissionTestResult, error)
	ListByAssignment(ctx context.Context, assignmentID string) ([]*models.SubmissionTestResult, error)
	DeleteBySubmission(ctx context.Context, submissionID string) error
}

type testResultRepository struct {
	db *sql.DB
}

// NewTestResultRepository constructs a SQL-backed test result repository.
func NewTestResultRepository(db *sql.DB) TestResultRepository {
	return &testResultRepository{db: db}
}

func (r *testResultRepository) Create(ctx context.Context, tr *models.SubmissionTestResult) error {
	if tr.ID == "" {
		tr.ID = uuid.New().String()
	}
	if tr.CreatedAt.IsZero() {
		tr.CreatedAt = time.Now()
	}
	query := `
		INSERT INTO submission_test_results (id, submission_id, assignment_id, test_name, passed, output, duration_ms, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`
	_, err := r.db.ExecContext(ctx, query, tr.ID, tr.SubmissionID, tr.AssignmentID, tr.TestName, tr.Passed, tr.Output, tr.DurationMS, tr.CreatedAt)
	if err != nil {
		return fmt.Errorf("failed to create test result: %w", err)
	}
	return nil
}

func (r *testResultRepository) CreateBatch(ctx context.Context, results []*models.SubmissionTestResult) error {
	if len(results) == 0 {
		return nil
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin tx: %w", err)
	}
	for _, tr := range results {
		if tr.ID == "" {
			tr.ID = uuid.New().String()
		}
		if tr.CreatedAt.IsZero() {
			tr.CreatedAt = time.Now()
		}
		_, err := tx.ExecContext(ctx, `
			INSERT INTO submission_test_results (id, submission_id, assignment_id, test_name, passed, output, duration_ms, created_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		`, tr.ID, tr.SubmissionID, tr.AssignmentID, tr.TestName, tr.Passed, tr.Output, tr.DurationMS, tr.CreatedAt)
		if err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("failed to insert test result: %w", err)
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit test results: %w", err)
	}
	return nil
}

func (r *testResultRepository) ListBySubmission(ctx context.Context, submissionID string) ([]*models.SubmissionTestResult, error) {
	query := `
		SELECT id, submission_id, assignment_id, test_name, passed, output, duration_ms, created_at
		FROM submission_test_results
		WHERE submission_id = $1
		ORDER BY test_name ASC
	`
	return r.scan(ctx, query, submissionID)
}

func (r *testResultRepository) ListByAssignment(ctx context.Context, assignmentID string) ([]*models.SubmissionTestResult, error) {
	query := `
		SELECT id, submission_id, assignment_id, test_name, passed, output, duration_ms, created_at
		FROM submission_test_results
		WHERE assignment_id = $1
		ORDER BY submission_id ASC, test_name ASC
	`
	return r.scan(ctx, query, assignmentID)
}

func (r *testResultRepository) DeleteBySubmission(ctx context.Context, submissionID string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM submission_test_results WHERE submission_id = $1`, submissionID)
	if err != nil {
		return fmt.Errorf("failed to delete test results: %w", err)
	}
	return nil
}

func (r *testResultRepository) scan(ctx context.Context, query string, args ...interface{}) ([]*models.SubmissionTestResult, error) {
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list test results: %w", err)
	}
	defer rows.Close()

	var out []*models.SubmissionTestResult
	for rows.Next() {
		tr := &models.SubmissionTestResult{}
		var output sql.NullString
		if err := rows.Scan(&tr.ID, &tr.SubmissionID, &tr.AssignmentID, &tr.TestName, &tr.Passed, &output, &tr.DurationMS, &tr.CreatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan test result: %w", err)
		}
		if output.Valid {
			tr.Output = output.String
		}
		out = append(out, tr)
	}
	return out, nil
}
