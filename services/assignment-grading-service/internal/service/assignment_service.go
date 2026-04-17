package service

import (
	"context"
	"fmt"
	"time"

	"slate/services/assignment-grading-service/internal/models"
	"slate/services/assignment-grading-service/internal/repository"
	"slate/services/assignment-grading-service/pkg/kafka"
)

type assignmentService struct {
	repo     repository.AssignmentRepository
	producer kafka.EventPublisher
}

// NewAssignmentService creates a new assignment service
func NewAssignmentService(repo repository.AssignmentRepository, producer kafka.EventPublisher) AssignmentService {
	return &assignmentService{
		repo:     repo,
		producer: producer,
	}
}

// CreateAssignment creates a new assignment (simple version for gRPC compat)
func (s *assignmentService) CreateAssignment(ctx context.Context, courseID, title, description string, maxPoints float64, dueDate time.Time, latePolicy models.LatePolicy) (*models.Assignment, error) {
	assignment := &models.Assignment{
		CourseID:    courseID,
		Title:       title,
		Description: description,
		MaxPoints:   maxPoints,
		DueDate:     dueDate,
		LatePolicy:  latePolicy,
	}
	return s.CreateAssignmentFull(ctx, assignment)
}

// CreateAssignmentFull creates a new assignment with all fields
func (s *assignmentService) CreateAssignmentFull(ctx context.Context, assignment *models.Assignment) (*models.Assignment, error) {
	if err := assignment.Validate(); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	if err := s.repo.Create(ctx, assignment); err != nil {
		return nil, fmt.Errorf("failed to create assignment: %w", err)
	}

	event := kafka.NewAssignmentCreatedEvent(assignment.ID, assignment.CourseID, assignment.Title)
	if err := s.producer.PublishEvent(ctx, event); err != nil {
		fmt.Printf("Failed to publish assignment.created event: %v\n", err)
	}

	return assignment, nil
}

// GetAssignment retrieves an assignment by ID
func (s *assignmentService) GetAssignment(ctx context.Context, id string) (*models.Assignment, error) {
	assignment, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get assignment: %w", err)
	}
	return assignment, nil
}

// UpdateAssignment updates an existing assignment (simple version)
func (s *assignmentService) UpdateAssignment(ctx context.Context, id, title, description string, maxPoints float64, dueDate time.Time, latePolicy models.LatePolicy) (*models.Assignment, error) {
	assignment, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get assignment: %w", err)
	}

	oldDueDate := assignment.DueDate
	assignment.Title = title
	assignment.Description = description
	assignment.MaxPoints = maxPoints
	assignment.DueDate = dueDate
	assignment.LatePolicy = latePolicy

	if err := assignment.Validate(); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	if err := s.repo.Update(ctx, assignment); err != nil {
		return nil, fmt.Errorf("failed to update assignment: %w", err)
	}

	// Check if due date changed
	if !oldDueDate.Equal(dueDate) {
		event := kafka.NewAssignmentDeadlineUpdatedEvent(assignment.ID, assignment.CourseID, dueDate)
		if err := s.producer.PublishEvent(ctx, event); err != nil {
			fmt.Printf("Failed to publish assignment.deadline_updated event: %v\n", err)
		}
	}

	event := kafka.NewAssignmentUpdatedEvent(assignment.ID, assignment.CourseID)
	if err := s.producer.PublishEvent(ctx, event); err != nil {
		fmt.Printf("Failed to publish assignment.updated event: %v\n", err)
	}

	return assignment, nil
}

// UpdateAssignmentFull updates an assignment with all fields
func (s *assignmentService) UpdateAssignmentFull(ctx context.Context, assignment *models.Assignment) (*models.Assignment, error) {
	existing, err := s.repo.GetByID(ctx, assignment.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to get assignment: %w", err)
	}

	oldDueDate := existing.DueDate

	if err := assignment.Validate(); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	if err := s.repo.Update(ctx, assignment); err != nil {
		return nil, fmt.Errorf("failed to update assignment: %w", err)
	}

	if !oldDueDate.Equal(assignment.DueDate) {
		event := kafka.NewAssignmentDeadlineUpdatedEvent(assignment.ID, assignment.CourseID, assignment.DueDate)
		if err := s.producer.PublishEvent(ctx, event); err != nil {
			fmt.Printf("Failed to publish assignment.deadline_updated event: %v\n", err)
		}
	}

	event := kafka.NewAssignmentUpdatedEvent(assignment.ID, assignment.CourseID)
	if err := s.producer.PublishEvent(ctx, event); err != nil {
		fmt.Printf("Failed to publish assignment.updated event: %v\n", err)
	}

	return assignment, nil
}

// DeleteAssignment hard-deletes an assignment
func (s *assignmentService) DeleteAssignment(ctx context.Context, id string) error {
	assignment, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to get assignment: %w", err)
	}

	if err := s.repo.Delete(ctx, id); err != nil {
		return fmt.Errorf("failed to delete assignment: %w", err)
	}

	event := kafka.NewAssignmentDeletedEvent(assignment.ID, assignment.CourseID)
	if err := s.producer.PublishEvent(ctx, event); err != nil {
		fmt.Printf("Failed to publish assignment.deleted event: %v\n", err)
	}

	return nil
}

// SoftDeleteAssignment soft-deletes an assignment
func (s *assignmentService) SoftDeleteAssignment(ctx context.Context, id string) error {
	assignment, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to get assignment: %w", err)
	}

	if err := s.repo.SoftDelete(ctx, id); err != nil {
		return fmt.Errorf("failed to soft delete assignment: %w", err)
	}

	event := kafka.NewAssignmentDeletedEvent(assignment.ID, assignment.CourseID)
	if err := s.producer.PublishEvent(ctx, event); err != nil {
		fmt.Printf("Failed to publish assignment.deleted event: %v\n", err)
	}

	return nil
}

// ListAssignments lists assignments for a course with pagination
func (s *assignmentService) ListAssignments(ctx context.Context, courseID string, page, pageSize int) ([]*models.Assignment, int, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	assignments, total, err := s.repo.ListByCourse(ctx, courseID, page, pageSize)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list assignments: %w", err)
	}

	return assignments, total, nil
}

// ListAssignmentsFiltered lists assignments with filters
func (s *assignmentService) ListAssignmentsFiltered(ctx context.Context, tenantID, courseID, instructorID string, page, pageSize int) ([]*models.Assignment, int, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	assignments, total, err := s.repo.ListFiltered(ctx, tenantID, courseID, instructorID, page, pageSize)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list assignments: %w", err)
	}

	return assignments, total, nil
}
