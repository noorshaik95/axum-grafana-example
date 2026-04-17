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

// AssignmentRepository defines the interface for assignment data access
type AssignmentRepository interface {
	Create(ctx context.Context, assignment *models.Assignment) error
	GetByID(ctx context.Context, id string) (*models.Assignment, error)
	Update(ctx context.Context, assignment *models.Assignment) error
	Delete(ctx context.Context, id string) error
	SoftDelete(ctx context.Context, id string) error
	ListByCourse(ctx context.Context, courseID string, page, pageSize int) ([]*models.Assignment, int, error)
	ListFiltered(ctx context.Context, tenantID, courseID, instructorID string, page, pageSize int) ([]*models.Assignment, int, error)
	HasSubmissions(ctx context.Context, id string) (bool, error)
	SoftDeleteByCourse(ctx context.Context, courseID string) error
}

type assignmentRepository struct {
	db *sql.DB
}

// NewAssignmentRepository creates a new assignment repository
func NewAssignmentRepository(db *sql.DB) AssignmentRepository {
	return &assignmentRepository{db: db}
}

// Create creates a new assignment
func (r *assignmentRepository) Create(ctx context.Context, assignment *models.Assignment) error {
	assignment.ID = uuid.New().String()
	assignment.CreatedAt = time.Now()
	assignment.UpdatedAt = time.Now()

	rubricJSON, err := assignment.RubricJSON()
	if err != nil {
		return fmt.Errorf("failed to marshal rubric: %w", err)
	}

	query := `
		INSERT INTO assignments (
			id, tenant_id, course_id, instructor_id, title, description,
			max_points, rubric, assignment_type, due_date, max_file_size_mb,
			allowed_file_types, is_deleted, late_penalty_percent, max_late_days,
			created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17)
	`

	_, err = r.db.ExecContext(ctx, query,
		assignment.ID, nilIfEmpty(assignment.TenantID), assignment.CourseID,
		nilIfEmpty(assignment.InstructorID), assignment.Title, assignment.Description,
		assignment.MaxPoints, rubricJSON, nilIfEmpty(assignment.AssignmentType),
		assignment.DueDate, assignment.MaxFileSizeMB,
		pq.Array(assignment.AllowedFileTypes), assignment.IsDeleted,
		assignment.LatePolicy.PenaltyPercentPerDay, assignment.LatePolicy.MaxLateDays,
		assignment.CreatedAt, assignment.UpdatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to create assignment: %w", err)
	}

	return nil
}

// GetByID retrieves an assignment by ID
func (r *assignmentRepository) GetByID(ctx context.Context, id string) (*models.Assignment, error) {
	query := `
		SELECT id, tenant_id, course_id, instructor_id, title, description,
			   max_points, rubric, assignment_type, due_date, max_file_size_mb,
			   allowed_file_types, is_deleted, late_penalty_percent, max_late_days,
			   created_at, updated_at
		FROM assignments
		WHERE id = $1 AND is_deleted = false
	`

	return r.scanAssignment(r.db.QueryRowContext(ctx, query, id))
}

// Update updates an existing assignment
func (r *assignmentRepository) Update(ctx context.Context, assignment *models.Assignment) error {
	assignment.UpdatedAt = time.Now()

	rubricJSON, err := assignment.RubricJSON()
	if err != nil {
		return fmt.Errorf("failed to marshal rubric: %w", err)
	}

	query := `
		UPDATE assignments
		SET title = $2, description = $3, max_points = $4, due_date = $5,
			late_penalty_percent = $6, max_late_days = $7, updated_at = $8,
			rubric = $9, assignment_type = $10, instructor_id = $11,
			max_file_size_mb = $12, allowed_file_types = $13
		WHERE id = $1 AND is_deleted = false
	`

	result, err := r.db.ExecContext(ctx, query,
		assignment.ID, assignment.Title, assignment.Description, assignment.MaxPoints,
		assignment.DueDate, assignment.LatePolicy.PenaltyPercentPerDay,
		assignment.LatePolicy.MaxLateDays, assignment.UpdatedAt,
		rubricJSON, nilIfEmpty(assignment.AssignmentType),
		nilIfEmpty(assignment.InstructorID), assignment.MaxFileSizeMB,
		pq.Array(assignment.AllowedFileTypes),
	)

	if err != nil {
		return fmt.Errorf("failed to update assignment: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rows == 0 {
		return fmt.Errorf("assignment not found")
	}

	return nil
}

// Delete hard-deletes an assignment
func (r *assignmentRepository) Delete(ctx context.Context, id string) error {
	query := `DELETE FROM assignments WHERE id = $1`

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete assignment: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rows == 0 {
		return fmt.Errorf("assignment not found")
	}

	return nil
}

// SoftDelete sets is_deleted=true
func (r *assignmentRepository) SoftDelete(ctx context.Context, id string) error {
	query := `UPDATE assignments SET is_deleted = true, updated_at = NOW() WHERE id = $1 AND is_deleted = false`

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to soft delete assignment: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rows == 0 {
		return fmt.Errorf("assignment not found")
	}

	return nil
}

// ListByCourse lists assignments for a course with pagination
func (r *assignmentRepository) ListByCourse(ctx context.Context, courseID string, page, pageSize int) ([]*models.Assignment, int, error) {
	// Get total count
	var total int
	countQuery := `SELECT COUNT(*) FROM assignments WHERE course_id = $1 AND is_deleted = false`
	if err := r.db.QueryRowContext(ctx, countQuery, courseID).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("failed to count assignments: %w", err)
	}

	// Get paginated results
	offset := (page - 1) * pageSize
	query := `
		SELECT id, tenant_id, course_id, instructor_id, title, description,
			   max_points, rubric, assignment_type, due_date, max_file_size_mb,
			   allowed_file_types, is_deleted, late_penalty_percent, max_late_days,
			   created_at, updated_at
		FROM assignments
		WHERE course_id = $1 AND is_deleted = false
		ORDER BY due_date DESC
		LIMIT $2 OFFSET $3
	`

	rows, err := r.db.QueryContext(ctx, query, courseID, pageSize, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list assignments: %w", err)
	}
	defer rows.Close()

	var assignments []*models.Assignment
	for rows.Next() {
		assignment, err := r.scanAssignmentRow(rows)
		if err != nil {
			return nil, 0, err
		}
		assignments = append(assignments, assignment)
	}

	return assignments, total, nil
}

// ListFiltered lists assignments with multiple filter criteria
func (r *assignmentRepository) ListFiltered(ctx context.Context, tenantID, courseID, instructorID string, page, pageSize int) ([]*models.Assignment, int, error) {
	baseWhere := "WHERE is_deleted = false"
	args := []interface{}{}
	argIdx := 1

	if tenantID != "" {
		baseWhere += fmt.Sprintf(" AND tenant_id = $%d", argIdx)
		args = append(args, tenantID)
		argIdx++
	}
	if courseID != "" {
		baseWhere += fmt.Sprintf(" AND course_id = $%d", argIdx)
		args = append(args, courseID)
		argIdx++
	}
	if instructorID != "" {
		baseWhere += fmt.Sprintf(" AND instructor_id = $%d", argIdx)
		args = append(args, instructorID)
		argIdx++
	}

	// Count
	var total int
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM assignments %s", baseWhere)
	if err := r.db.QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("failed to count assignments: %w", err)
	}

	// Paginated results
	offset := (page - 1) * pageSize
	selectQuery := fmt.Sprintf(`
		SELECT id, tenant_id, course_id, instructor_id, title, description,
			   max_points, rubric, assignment_type, due_date, max_file_size_mb,
			   allowed_file_types, is_deleted, late_penalty_percent, max_late_days,
			   created_at, updated_at
		FROM assignments
		%s
		ORDER BY due_date DESC
		LIMIT $%d OFFSET $%d
	`, baseWhere, argIdx, argIdx+1)

	args = append(args, pageSize, offset)

	rows, err := r.db.QueryContext(ctx, selectQuery, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list assignments: %w", err)
	}
	defer rows.Close()

	var assignments []*models.Assignment
	for rows.Next() {
		assignment, err := r.scanAssignmentRow(rows)
		if err != nil {
			return nil, 0, err
		}
		assignments = append(assignments, assignment)
	}

	return assignments, total, nil
}

// HasSubmissions checks if an assignment has any submissions
func (r *assignmentRepository) HasSubmissions(ctx context.Context, id string) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM submissions WHERE assignment_id = $1)`

	var exists bool
	err := r.db.QueryRowContext(ctx, query, id).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check submissions: %w", err)
	}

	return exists, nil
}

// SoftDeleteByCourse soft-deletes all assignments for a course (Kafka consumer: course.deleted)
func (r *assignmentRepository) SoftDeleteByCourse(ctx context.Context, courseID string) error {
	query := `UPDATE assignments SET is_deleted = true, updated_at = NOW() WHERE course_id = $1 AND is_deleted = false`
	_, err := r.db.ExecContext(ctx, query, courseID)
	if err != nil {
		return fmt.Errorf("failed to soft delete assignments for course: %w", err)
	}
	return nil
}

// scanAssignment scans a single row into an Assignment
func (r *assignmentRepository) scanAssignment(row *sql.Row) (*models.Assignment, error) {
	assignment := &models.Assignment{}
	var tenantID, instructorID, assignmentType sql.NullString
	var rubricJSON []byte
	var allowedFileTypes pq.StringArray
	var maxFileSizeMB sql.NullInt32

	err := row.Scan(
		&assignment.ID, &tenantID, &assignment.CourseID, &instructorID,
		&assignment.Title, &assignment.Description, &assignment.MaxPoints,
		&rubricJSON, &assignmentType, &assignment.DueDate, &maxFileSizeMB,
		&allowedFileTypes, &assignment.IsDeleted,
		&assignment.LatePolicy.PenaltyPercentPerDay, &assignment.LatePolicy.MaxLateDays,
		&assignment.CreatedAt, &assignment.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("assignment not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get assignment: %w", err)
	}

	assignment.TenantID = nullStringVal(tenantID)
	assignment.InstructorID = nullStringVal(instructorID)
	assignment.AssignmentType = nullStringVal(assignmentType)
	assignment.AllowedFileTypes = []string(allowedFileTypes)
	if maxFileSizeMB.Valid {
		assignment.MaxFileSizeMB = int(maxFileSizeMB.Int32)
	}
	if rubricJSON != nil {
		if err := assignment.SetRubricFromJSON(rubricJSON); err != nil {
			return nil, fmt.Errorf("failed to parse rubric: %w", err)
		}
	}

	return assignment, nil
}

// scanAssignmentRow scans from sql.Rows into an Assignment
func (r *assignmentRepository) scanAssignmentRow(rows *sql.Rows) (*models.Assignment, error) {
	assignment := &models.Assignment{}
	var tenantID, instructorID, assignmentType sql.NullString
	var rubricJSON []byte
	var allowedFileTypes pq.StringArray
	var maxFileSizeMB sql.NullInt32

	err := rows.Scan(
		&assignment.ID, &tenantID, &assignment.CourseID, &instructorID,
		&assignment.Title, &assignment.Description, &assignment.MaxPoints,
		&rubricJSON, &assignmentType, &assignment.DueDate, &maxFileSizeMB,
		&allowedFileTypes, &assignment.IsDeleted,
		&assignment.LatePolicy.PenaltyPercentPerDay, &assignment.LatePolicy.MaxLateDays,
		&assignment.CreatedAt, &assignment.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to scan assignment: %w", err)
	}

	assignment.TenantID = nullStringVal(tenantID)
	assignment.InstructorID = nullStringVal(instructorID)
	assignment.AssignmentType = nullStringVal(assignmentType)
	assignment.AllowedFileTypes = []string(allowedFileTypes)
	if maxFileSizeMB.Valid {
		assignment.MaxFileSizeMB = int(maxFileSizeMB.Int32)
	}
	if rubricJSON != nil {
		if err := assignment.SetRubricFromJSON(rubricJSON); err != nil {
			return nil, fmt.Errorf("failed to parse rubric: %w", err)
		}
	}

	return assignment, nil
}

func nilIfEmpty(s string) interface{} {
	if s == "" {
		return nil
	}
	return s
}

func nullStringVal(ns sql.NullString) string {
	if ns.Valid {
		return ns.String
	}
	return ""
}
