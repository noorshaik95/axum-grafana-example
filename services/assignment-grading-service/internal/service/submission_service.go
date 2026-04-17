package service

import (
	"bytes"
	"context"
	"fmt"
	"time"

	"slate/services/assignment-grading-service/internal/models"
	"slate/services/assignment-grading-service/internal/repository"
	"slate/services/assignment-grading-service/pkg/kafka"
	"slate/services/assignment-grading-service/pkg/storage"
)

type submissionService struct {
	assignmentRepo repository.AssignmentRepository
	submissionRepo repository.SubmissionRepository
	storage        storage.FileStorage
	producer       kafka.EventPublisher
	latePolicyCalc *LatePolicyCalculator
}

// NewSubmissionService creates a new submission service
func NewSubmissionService(
	assignmentRepo repository.AssignmentRepository,
	submissionRepo repository.SubmissionRepository,
	fileStorage storage.FileStorage,
	producer kafka.EventPublisher,
) SubmissionService {
	return &submissionService{
		assignmentRepo: assignmentRepo,
		submissionRepo: submissionRepo,
		storage:        fileStorage,
		producer:       producer,
		latePolicyCalc: NewLatePolicyCalculator(),
	}
}

// SubmitAssignment creates a new submission with file upload
func (s *submissionService) SubmitAssignment(ctx context.Context, assignmentID, studentID string, fileContent []byte, fileName, contentType string) (*models.Submission, error) {
	assignment, err := s.assignmentRepo.GetByID(ctx, assignmentID)
	if err != nil {
		return nil, fmt.Errorf("failed to get assignment: %w", err)
	}

	reader := bytes.NewReader(fileContent)
	filePath, err := s.storage.Save(fileName, reader, contentType)
	if err != nil {
		return nil, fmt.Errorf("failed to save file: %w", err)
	}

	submittedAt := time.Now()
	isLate, daysLate := s.latePolicyCalc.CalculatePenalty(submittedAt, assignment.DueDate, assignment.LatePolicy)

	submission := &models.Submission{
		TenantID:     assignment.TenantID,
		AssignmentID: assignmentID,
		StudentID:    studentID,
		FilePath:     filePath,
		SubmittedAt:  submittedAt,
		Status:       models.StatusSubmitted,
		IsLate:       isLate,
		DaysLate:     daysLate,
	}

	if err := submission.Validate(); err != nil {
		if delErr := s.storage.Delete(filePath); delErr != nil {
			fmt.Printf("Failed to delete file %s: %v\n", filePath, delErr)
		}
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	if err := s.submissionRepo.Create(ctx, submission); err != nil {
		if delErr := s.storage.Delete(filePath); delErr != nil {
			fmt.Printf("Failed to delete file %s: %v\n", filePath, delErr)
		}
		return nil, fmt.Errorf("failed to create submission: %w", err)
	}

	event := kafka.NewSubmissionCreatedEvent(submission.ID, assignmentID, studentID, isLate)
	if err := s.producer.PublishEvent(ctx, event); err != nil {
		fmt.Printf("Failed to publish submission.created event: %v\n", err)
	}

	return submission, nil
}

// SubmitAssignmentWithURLs creates a submission with MinIO file URLs
func (s *submissionService) SubmitAssignmentWithURLs(ctx context.Context, tenantID, assignmentID, studentID string, fileURLs []string) (*models.Submission, error) {
	assignment, err := s.assignmentRepo.GetByID(ctx, assignmentID)
	if err != nil {
		return nil, fmt.Errorf("failed to get assignment: %w", err)
	}

	submittedAt := time.Now()
	isLate, daysLate := s.latePolicyCalc.CalculatePenalty(submittedAt, assignment.DueDate, assignment.LatePolicy)

	submission := &models.Submission{
		TenantID:     tenantID,
		AssignmentID: assignmentID,
		StudentID:    studentID,
		FileURLs:     fileURLs,
		FilePath:     fileURLs[0], // primary file path for backwards compat
		SubmittedAt:  submittedAt,
		Status:       models.StatusSubmitted,
		IsLate:       isLate,
		DaysLate:     daysLate,
	}

	if err := submission.Validate(); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	if err := s.submissionRepo.Create(ctx, submission); err != nil {
		return nil, fmt.Errorf("failed to create submission: %w", err)
	}

	event := kafka.NewSubmissionCreatedEvent(submission.ID, assignmentID, studentID, isLate)
	if err := s.producer.PublishEvent(ctx, event); err != nil {
		fmt.Printf("Failed to publish submission.created event: %v\n", err)
	}

	return submission, nil
}

// GetSubmission retrieves a submission by ID
func (s *submissionService) GetSubmission(ctx context.Context, id string) (*models.Submission, error) {
	submission, err := s.submissionRepo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get submission: %w", err)
	}
	return submission, nil
}

// ListSubmissions lists submissions for an assignment
func (s *submissionService) ListSubmissions(ctx context.Context, assignmentID, sortBy, order string) ([]*models.Submission, error) {
	submissions, err := s.submissionRepo.ListByAssignment(ctx, assignmentID, sortBy, order)
	if err != nil {
		return nil, fmt.Errorf("failed to list submissions: %w", err)
	}
	return submissions, nil
}

// ListSubmissionsPaginated lists submissions with pagination
func (s *submissionService) ListSubmissionsPaginated(ctx context.Context, assignmentID, tenantID string, page, pageSize int) ([]*models.Submission, int, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	return s.submissionRepo.ListByAssignmentPaginated(ctx, assignmentID, tenantID, page, pageSize)
}

// ListStudentSubmissions lists submissions for a student in a course
func (s *submissionService) ListStudentSubmissions(ctx context.Context, studentID, courseID string) ([]*models.Submission, error) {
	submissions, err := s.submissionRepo.ListByStudent(ctx, studentID, courseID)
	if err != nil {
		return nil, fmt.Errorf("failed to list student submissions: %w", err)
	}
	return submissions, nil
}
