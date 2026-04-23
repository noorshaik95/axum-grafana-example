package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"slate/services/assignment-grading-service/internal/models"

	"github.com/google/uuid"
)

// RubricRowRepository persists per-assignment rubric rows (W9.1).
type RubricRowRepository interface {
	Create(ctx context.Context, row *models.RubricRow) error
	GetByID(ctx context.Context, id string) (*models.RubricRow, error)
	Update(ctx context.Context, row *models.RubricRow) error
	Delete(ctx context.Context, id string) error
	ListByAssignment(ctx context.Context, assignmentID string) ([]*models.RubricRow, error)
}

type rubricRowRepository struct {
	db *sql.DB
}

// NewRubricRowRepository constructs a SQL-backed rubric row repository.
func NewRubricRowRepository(db *sql.DB) RubricRowRepository {
	return &rubricRowRepository{db: db}
}

func (r *rubricRowRepository) Create(ctx context.Context, row *models.RubricRow) error {
	if row.ID == "" {
		row.ID = uuid.New().String()
	}
	now := time.Now()
	row.CreatedAt = now
	row.UpdatedAt = now

	query := `
		INSERT INTO rubric_rows (id, assignment_id, title, max_points, sort_order, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`
	_, err := r.db.ExecContext(ctx, query,
		row.ID, row.AssignmentID, row.Title, row.MaxPoints, row.SortOrder,
		row.CreatedAt, row.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to create rubric row: %w", err)
	}
	return nil
}

func (r *rubricRowRepository) GetByID(ctx context.Context, id string) (*models.RubricRow, error) {
	query := `
		SELECT id, assignment_id, title, max_points, sort_order, created_at, updated_at
		FROM rubric_rows WHERE id = $1
	`
	row := r.db.QueryRowContext(ctx, query, id)
	rr := &models.RubricRow{}
	if err := row.Scan(&rr.ID, &rr.AssignmentID, &rr.Title, &rr.MaxPoints, &rr.SortOrder, &rr.CreatedAt, &rr.UpdatedAt); err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("rubric row not found")
		}
		return nil, fmt.Errorf("failed to get rubric row: %w", err)
	}
	return rr, nil
}

func (r *rubricRowRepository) Update(ctx context.Context, row *models.RubricRow) error {
	row.UpdatedAt = time.Now()
	query := `
		UPDATE rubric_rows
		SET title = $2, max_points = $3, sort_order = $4, updated_at = $5
		WHERE id = $1
	`
	result, err := r.db.ExecContext(ctx, query, row.ID, row.Title, row.MaxPoints, row.SortOrder, row.UpdatedAt)
	if err != nil {
		return fmt.Errorf("failed to update rubric row: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("rubric row not found")
	}
	return nil
}

func (r *rubricRowRepository) Delete(ctx context.Context, id string) error {
	result, err := r.db.ExecContext(ctx, `DELETE FROM rubric_rows WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("failed to delete rubric row: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("rubric row not found")
	}
	return nil
}

func (r *rubricRowRepository) ListByAssignment(ctx context.Context, assignmentID string) ([]*models.RubricRow, error) {
	query := `
		SELECT id, assignment_id, title, max_points, sort_order, created_at, updated_at
		FROM rubric_rows
		WHERE assignment_id = $1
		ORDER BY sort_order ASC, created_at ASC
	`
	rows, err := r.db.QueryContext(ctx, query, assignmentID)
	if err != nil {
		return nil, fmt.Errorf("failed to list rubric rows: %w", err)
	}
	defer rows.Close()

	var out []*models.RubricRow
	for rows.Next() {
		rr := &models.RubricRow{}
		if err := rows.Scan(&rr.ID, &rr.AssignmentID, &rr.Title, &rr.MaxPoints, &rr.SortOrder, &rr.CreatedAt, &rr.UpdatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan rubric row: %w", err)
		}
		out = append(out, rr)
	}
	return out, nil
}
