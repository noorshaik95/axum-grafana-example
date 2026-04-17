package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"slate/services/assignment-grading-service/internal/models"

	"github.com/google/uuid"
	"github.com/lib/pq"
)

const (
	orderDesc = "DESC"
)

// SubmissionRepository defines the interface for submission data access
type SubmissionRepository interface {
	Create(ctx context.Context, submission *models.Submission) error
	GetByID(ctx context.Context, id string) (*models.Submission, error)
	GetByAssignmentAndStudent(ctx context.Context, assignmentID, studentID string) (*models.Submission, error)
	Update(ctx context.Context, submission *models.Submission) error
	ListByAssignment(ctx context.Context, assignmentID, sortBy, order string) ([]*models.Submission, error)
	ListByAssignmentPaginated(ctx context.Context, assignmentID, tenantID string, page, pageSize int) ([]*models.Submission, int, error)
	ListByStudent(ctx context.Context, studentID, courseID string) ([]*models.Submission, error)
}

type submissionRepository struct {
	db *sql.DB
}

// NewSubmissionRepository creates a new submission repository
func NewSubmissionRepository(db *sql.DB) SubmissionRepository {
	return &submissionRepository{db: db}
}

// Create creates a new submission (or updates if exists due to UNIQUE constraint)
func (r *submissionRepository) Create(ctx context.Context, submission *models.Submission) error {
	if submission.ID == "" {
		submission.ID = uuid.New().String()
	}
	submission.CreatedAt = time.Now()
	submission.UpdatedAt = time.Now()

	query := `
		INSERT INTO submissions (
			id, tenant_id, assignment_id, student_id, file_path, file_urls,
			submitted_at, status, is_late, days_late, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		ON CONFLICT (assignment_id, student_id)
		DO UPDATE SET
			file_path = EXCLUDED.file_path,
			file_urls = EXCLUDED.file_urls,
			submitted_at = EXCLUDED.submitted_at,
			status = EXCLUDED.status,
			is_late = EXCLUDED.is_late,
			days_late = EXCLUDED.days_late,
			updated_at = EXCLUDED.updated_at
		RETURNING id
	`

	err := r.db.QueryRowContext(ctx, query,
		submission.ID, nilIfEmpty(submission.TenantID), submission.AssignmentID,
		submission.StudentID, submission.FilePath, pq.Array(submission.FileURLs),
		submission.SubmittedAt, submission.Status, submission.IsLate, submission.DaysLate,
		submission.CreatedAt, submission.UpdatedAt,
	).Scan(&submission.ID)

	if err != nil {
		return fmt.Errorf("failed to create submission: %w", err)
	}

	return nil
}

// GetByID retrieves a submission by ID
func (r *submissionRepository) GetByID(ctx context.Context, id string) (*models.Submission, error) {
	query := `
		SELECT id, tenant_id, assignment_id, student_id, file_path, file_urls,
			   submitted_at, status, is_late, days_late, created_at, updated_at
		FROM submissions
		WHERE id = $1
	`

	return r.scanSubmission(r.db.QueryRowContext(ctx, query, id))
}

// GetByAssignmentAndStudent retrieves a submission by assignment and student
func (r *submissionRepository) GetByAssignmentAndStudent(ctx context.Context, assignmentID, studentID string) (*models.Submission, error) {
	query := `
		SELECT id, tenant_id, assignment_id, student_id, file_path, file_urls,
			   submitted_at, status, is_late, days_late, created_at, updated_at
		FROM submissions
		WHERE assignment_id = $1 AND student_id = $2
	`

	sub, err := r.scanSubmission(r.db.QueryRowContext(ctx, query, assignmentID, studentID))
	if err != nil && err.Error() == "submission not found" {
		return nil, nil
	}
	return sub, err
}

// Update updates an existing submission
func (r *submissionRepository) Update(ctx context.Context, submission *models.Submission) error {
	submission.UpdatedAt = time.Now()

	query := `
		UPDATE submissions
		SET file_path = $2, file_urls = $3, status = $4, is_late = $5,
			days_late = $6, updated_at = $7
		WHERE id = $1
	`

	result, err := r.db.ExecContext(ctx, query,
		submission.ID, submission.FilePath, pq.Array(submission.FileURLs),
		submission.Status, submission.IsLate, submission.DaysLate, submission.UpdatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to update submission: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rows == 0 {
		return fmt.Errorf("submission not found")
	}

	return nil
}

// ListByAssignment lists submissions for an assignment with sorting
func (r *submissionRepository) ListByAssignment(ctx context.Context, assignmentID, sortBy, order string) ([]*models.Submission, error) {
	if sortBy == "" {
		sortBy = "submitted_at"
	}
	if order == "" {
		order = orderDesc
	}

	validSortFields := map[string]bool{
		"submitted_at": true,
		"student_id":   true,
		"status":       true,
	}
	if !validSortFields[sortBy] {
		sortBy = "submitted_at"
	}

	if order != "ASC" && order != orderDesc {
		order = orderDesc
	}

	// #nosec G201 - sortBy and order are validated against allowlists above
	query := fmt.Sprintf(`
		SELECT id, tenant_id, assignment_id, student_id, file_path, file_urls,
			   submitted_at, status, is_late, days_late, created_at, updated_at
		FROM submissions
		WHERE assignment_id = $1
		ORDER BY %s %s
	`, sortBy, order)

	return r.scanSubmissions(ctx, query, assignmentID)
}

// ListByAssignmentPaginated lists submissions with pagination and tenant scope
func (r *submissionRepository) ListByAssignmentPaginated(ctx context.Context, assignmentID, tenantID string, page, pageSize int) ([]*models.Submission, int, error) {
	baseWhere := "WHERE assignment_id = $1"
	args := []interface{}{assignmentID}
	argIdx := 2

	if tenantID != "" {
		baseWhere += fmt.Sprintf(" AND tenant_id = $%d", argIdx)
		args = append(args, tenantID)
		argIdx++
	}

	var total int
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM submissions %s", baseWhere)
	if err := r.db.QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("failed to count submissions: %w", err)
	}

	offset := (page - 1) * pageSize
	selectQuery := fmt.Sprintf(`
		SELECT id, tenant_id, assignment_id, student_id, file_path, file_urls,
			   submitted_at, status, is_late, days_late, created_at, updated_at
		FROM submissions
		%s
		ORDER BY submitted_at DESC
		LIMIT $%d OFFSET $%d
	`, baseWhere, argIdx, argIdx+1)

	args = append(args, pageSize, offset)

	subs, err := r.scanSubmissions(ctx, selectQuery, args...)
	if err != nil {
		return nil, 0, err
	}

	return subs, total, nil
}

// ListByStudent lists submissions for a student in a course
func (r *submissionRepository) ListByStudent(ctx context.Context, studentID, courseID string) ([]*models.Submission, error) {
	query := `
		SELECT s.id, s.tenant_id, s.assignment_id, s.student_id, s.file_path, s.file_urls,
			   s.submitted_at, s.status, s.is_late, s.days_late, s.created_at, s.updated_at
		FROM submissions s
		JOIN assignments a ON s.assignment_id = a.id
		WHERE s.student_id = $1 AND a.course_id = $2
		ORDER BY s.submitted_at DESC
	`

	return r.scanSubmissions(ctx, query, studentID, courseID)
}

// scanSubmission scans a single row
func (r *submissionRepository) scanSubmission(row *sql.Row) (*models.Submission, error) {
	submission := &models.Submission{}
	var tenantID sql.NullString
	var fileURLs pq.StringArray

	err := row.Scan(
		&submission.ID, &tenantID, &submission.AssignmentID, &submission.StudentID,
		&submission.FilePath, &fileURLs,
		&submission.SubmittedAt, &submission.Status, &submission.IsLate, &submission.DaysLate,
		&submission.CreatedAt, &submission.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("submission not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get submission: %w", err)
	}

	submission.TenantID = nullStringVal(tenantID)
	submission.FileURLs = []string(fileURLs)

	return submission, nil
}

// scanSubmissions scans multiple rows
func (r *submissionRepository) scanSubmissions(ctx context.Context, query string, args ...interface{}) ([]*models.Submission, error) {
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list submissions: %w", err)
	}
	defer rows.Close()

	var submissions []*models.Submission
	for rows.Next() {
		submission := &models.Submission{}
		var tenantID sql.NullString
		var fileURLs pq.StringArray

		err := rows.Scan(
			&submission.ID, &tenantID, &submission.AssignmentID, &submission.StudentID,
			&submission.FilePath, &fileURLs,
			&submission.SubmittedAt, &submission.Status, &submission.IsLate, &submission.DaysLate,
			&submission.CreatedAt, &submission.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan submission: %w", err)
		}

		submission.TenantID = nullStringVal(tenantID)
		submission.FileURLs = []string(fileURLs)

		submissions = append(submissions, submission)
	}

	return submissions, nil
}
