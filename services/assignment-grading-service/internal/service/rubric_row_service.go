package service

import (
	"context"
	"fmt"

	"slate/services/assignment-grading-service/internal/models"
	"slate/services/assignment-grading-service/internal/repository"
)

// RubricRowService exposes business rules for rubric-row CRUD (W9.1).
type RubricRowService interface {
	Create(ctx context.Context, row *models.RubricRow) (*models.RubricRow, error)
	List(ctx context.Context, assignmentID string) ([]*models.RubricRow, error)
	Update(ctx context.Context, row *models.RubricRow) (*models.RubricRow, error)
	Delete(ctx context.Context, id string) error
}

type rubricRowService struct {
	repo repository.RubricRowRepository
}

// NewRubricRowService constructs the default rubric-row service.
func NewRubricRowService(repo repository.RubricRowRepository) RubricRowService {
	return &rubricRowService{repo: repo}
}

func (s *rubricRowService) Create(ctx context.Context, row *models.RubricRow) (*models.RubricRow, error) {
	if err := row.Validate(); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}
	if err := s.repo.Create(ctx, row); err != nil {
		return nil, err
	}
	return row, nil
}

func (s *rubricRowService) List(ctx context.Context, assignmentID string) ([]*models.RubricRow, error) {
	if assignmentID == "" {
		return nil, fmt.Errorf("assignment_id is required")
	}
	return s.repo.ListByAssignment(ctx, assignmentID)
}

func (s *rubricRowService) Update(ctx context.Context, row *models.RubricRow) (*models.RubricRow, error) {
	if row.ID == "" {
		return nil, fmt.Errorf("id is required")
	}
	if err := row.Validate(); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}
	if err := s.repo.Update(ctx, row); err != nil {
		return nil, err
	}
	return row, nil
}

func (s *rubricRowService) Delete(ctx context.Context, id string) error {
	if id == "" {
		return fmt.Errorf("id is required")
	}
	return s.repo.Delete(ctx, id)
}
