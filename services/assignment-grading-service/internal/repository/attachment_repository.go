package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"slate/services/assignment-grading-service/internal/models"

	"github.com/google/uuid"
)

// AttachmentRepository persists per-assignment files (W9.6).
type AttachmentRepository interface {
	Create(ctx context.Context, a *models.AssignmentAttachment) error
	GetByID(ctx context.Context, id string) (*models.AssignmentAttachment, error)
	Delete(ctx context.Context, id string) error
	ListByAssignment(ctx context.Context, assignmentID string) ([]*models.AssignmentAttachment, error)
}

type attachmentRepository struct {
	db *sql.DB
}

// NewAttachmentRepository constructs a SQL-backed attachment repository.
func NewAttachmentRepository(db *sql.DB) AttachmentRepository {
	return &attachmentRepository{db: db}
}

func (r *attachmentRepository) Create(ctx context.Context, a *models.AssignmentAttachment) error {
	if a.ID == "" {
		a.ID = uuid.New().String()
	}
	if a.CreatedAt.IsZero() {
		a.CreatedAt = time.Now()
	}
	query := `
		INSERT INTO assignment_attachments (id, assignment_id, file_path, file_name, content_type, size_bytes, kind, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`
	_, err := r.db.ExecContext(ctx, query, a.ID, a.AssignmentID, a.FilePath, a.FileName, nilIfEmpty(a.ContentType), a.SizeBytes, a.Kind, a.CreatedAt)
	if err != nil {
		return fmt.Errorf("failed to create attachment: %w", err)
	}
	return nil
}

func (r *attachmentRepository) GetByID(ctx context.Context, id string) (*models.AssignmentAttachment, error) {
	query := `
		SELECT id, assignment_id, file_path, file_name, content_type, size_bytes, kind, created_at
		FROM assignment_attachments WHERE id = $1
	`
	a := &models.AssignmentAttachment{}
	var ct sql.NullString
	err := r.db.QueryRowContext(ctx, query, id).Scan(&a.ID, &a.AssignmentID, &a.FilePath, &a.FileName, &ct, &a.SizeBytes, &a.Kind, &a.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("attachment not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get attachment: %w", err)
	}
	if ct.Valid {
		a.ContentType = ct.String
	}
	return a, nil
}

func (r *attachmentRepository) Delete(ctx context.Context, id string) error {
	result, err := r.db.ExecContext(ctx, `DELETE FROM assignment_attachments WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("failed to delete attachment: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("attachment not found")
	}
	return nil
}

func (r *attachmentRepository) ListByAssignment(ctx context.Context, assignmentID string) ([]*models.AssignmentAttachment, error) {
	query := `
		SELECT id, assignment_id, file_path, file_name, content_type, size_bytes, kind, created_at
		FROM assignment_attachments
		WHERE assignment_id = $1
		ORDER BY created_at ASC
	`
	rows, err := r.db.QueryContext(ctx, query, assignmentID)
	if err != nil {
		return nil, fmt.Errorf("failed to list attachments: %w", err)
	}
	defer rows.Close()

	var out []*models.AssignmentAttachment
	for rows.Next() {
		a := &models.AssignmentAttachment{}
		var ct sql.NullString
		if err := rows.Scan(&a.ID, &a.AssignmentID, &a.FilePath, &a.FileName, &ct, &a.SizeBytes, &a.Kind, &a.CreatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan attachment: %w", err)
		}
		if ct.Valid {
			a.ContentType = ct.String
		}
		out = append(out, a)
	}
	return out, nil
}
