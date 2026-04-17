package service

import (
	"context"
	"database/sql"
	"fmt"

	"slate/services/assignment-grading-service/internal/grading"
	"slate/services/assignment-grading-service/internal/models"
	"slate/services/assignment-grading-service/internal/repository"
	"slate/services/assignment-grading-service/pkg/kafka"
)

type gradingService struct {
	assignmentRepo repository.AssignmentRepository
	submissionRepo repository.SubmissionRepository
	gradeRepo      repository.GradeRepository
	producer       kafka.EventPublisher
	latePolicyCalc *LatePolicyCalculator
	engine         *grading.Engine
	db             *sql.DB
}

// NewGradingService creates a new grading service
func NewGradingService(
	assignmentRepo repository.AssignmentRepository,
	submissionRepo repository.SubmissionRepository,
	gradeRepo repository.GradeRepository,
	producer kafka.EventPublisher,
) GradingService {
	return &gradingService{
		assignmentRepo: assignmentRepo,
		submissionRepo: submissionRepo,
		gradeRepo:      gradeRepo,
		producer:       producer,
		latePolicyCalc: NewLatePolicyCalculator(),
		engine:         grading.NewEngine(gradeRepo, producer),
	}
}

// NewGradingServiceWithDB creates a grading service with DB access for auto-grading
func NewGradingServiceWithDB(
	assignmentRepo repository.AssignmentRepository,
	submissionRepo repository.SubmissionRepository,
	gradeRepo repository.GradeRepository,
	producer kafka.EventPublisher,
	db *sql.DB,
) GradingService {
	return &gradingService{
		assignmentRepo: assignmentRepo,
		submissionRepo: submissionRepo,
		gradeRepo:      gradeRepo,
		producer:       producer,
		latePolicyCalc: NewLatePolicyCalculator(),
		engine:         grading.NewEngine(gradeRepo, producer),
		db:             db,
	}
}

// CreateGrade creates a new grade for a submission (simple version)
func (s *gradingService) CreateGrade(ctx context.Context, submissionID string, score float64, feedback, gradedBy string) (*models.Grade, error) {
	return s.CreateGradeFull(ctx, submissionID, score, feedback, gradedBy, nil, "", "")
}

// CreateGradeFull creates a grade with all fields
func (s *gradingService) CreateGradeFull(ctx context.Context, submissionID string, score float64, feedback, gradedBy string, rubricScores map[string]interface{}, courseID, tenantID string) (*models.Grade, error) {
	submission, err := s.submissionRepo.GetByID(ctx, submissionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get submission: %w", err)
	}

	assignment, err := s.assignmentRepo.GetByID(ctx, submission.AssignmentID)
	if err != nil {
		return nil, fmt.Errorf("failed to get assignment: %w", err)
	}

	if score < 0 || score > assignment.MaxPoints {
		return nil, fmt.Errorf("score must be between 0 and %f", assignment.MaxPoints)
	}

	adjustedScore := s.latePolicyCalc.ApplyPenalty(score, submission.DaysLate, assignment.LatePolicy)

	percentage := 0.0
	if assignment.MaxPoints > 0 {
		percentage = (adjustedScore / assignment.MaxPoints) * 100
	}

	letterGrade := grading.LetterGrade(percentage, nil)

	if courseID == "" {
		courseID = assignment.CourseID
	}
	if tenantID == "" {
		tenantID = assignment.TenantID
	}

	grade := &models.Grade{
		TenantID:     tenantID,
		SubmissionID: submissionID,
		StudentID:    submission.StudentID,
		AssignmentID: assignment.ID,
		CourseID:     courseID,
		Score:        score,
		MaxScore:     assignment.MaxPoints,
		AdjustedScore: adjustedScore,
		Percentage:   percentage,
		LetterGrade:  letterGrade,
		RubricScores: rubricScores,
		Feedback:     feedback,
		Status:       models.GradeStatusDraft,
		GradedBy:     gradedBy,
	}

	if err := grade.Validate(); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	if err := grade.ValidateScore(assignment.MaxPoints); err != nil {
		return nil, fmt.Errorf("score validation failed: %w", err)
	}

	if err := s.gradeRepo.Create(ctx, grade); err != nil {
		return nil, fmt.Errorf("failed to create grade: %w", err)
	}

	submission.Status = models.StatusGraded
	if err := s.submissionRepo.Update(ctx, submission); err != nil {
		fmt.Printf("Failed to update submission status: %v\n", err)
	}

	// Emit submission.graded event
	event := kafka.NewSubmissionGradedEvent(grade.ID, grade.AssignmentID, grade.StudentID, grade.Score, grade.TenantID)
	if err := s.producer.PublishEvent(ctx, event); err != nil {
		fmt.Printf("Failed to publish submission.graded event: %v\n", err)
	}

	return grade, nil
}

// UpdateGrade updates an existing grade (only draft grades can be updated)
func (s *gradingService) UpdateGrade(ctx context.Context, id string, score float64, feedback string) (*models.Grade, error) {
	grade, err := s.gradeRepo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get grade: %w", err)
	}

	if !grade.IsDraft() {
		return nil, fmt.Errorf("only draft grades can be updated")
	}

	submission, err := s.submissionRepo.GetByID(ctx, grade.SubmissionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get submission: %w", err)
	}

	assignment, err := s.assignmentRepo.GetByID(ctx, submission.AssignmentID)
	if err != nil {
		return nil, fmt.Errorf("failed to get assignment: %w", err)
	}

	if score < 0 || score > assignment.MaxPoints {
		return nil, fmt.Errorf("score must be between 0 and %f", assignment.MaxPoints)
	}

	adjustedScore := s.latePolicyCalc.ApplyPenalty(score, submission.DaysLate, assignment.LatePolicy)

	percentage := 0.0
	if assignment.MaxPoints > 0 {
		percentage = (adjustedScore / assignment.MaxPoints) * 100
	}

	grade.Score = score
	grade.AdjustedScore = adjustedScore
	grade.Feedback = feedback
	grade.Percentage = percentage
	grade.LetterGrade = grading.LetterGrade(percentage, nil)
	grade.MaxScore = assignment.MaxPoints

	if err := s.gradeRepo.Update(ctx, grade); err != nil {
		return nil, fmt.Errorf("failed to update grade: %w", err)
	}

	return grade, nil
}

// PublishGrade publishes a grade (makes it visible to student)
func (s *gradingService) PublishGrade(ctx context.Context, id string) (*models.Grade, error) {
	grade, err := s.gradeRepo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get grade: %w", err)
	}

	if publishErr := grade.Publish(); publishErr != nil {
		return nil, fmt.Errorf("failed to publish grade: %w", publishErr)
	}

	if updateErr := s.gradeRepo.Update(ctx, grade); updateErr != nil {
		return nil, fmt.Errorf("failed to update grade: %w", updateErr)
	}

	submission, err := s.submissionRepo.GetByID(ctx, grade.SubmissionID)
	if err == nil {
		submission.Status = models.StatusReturned
		if err := s.submissionRepo.Update(ctx, submission); err != nil {
			fmt.Printf("Failed to update submission status: %v\n", err)
		}
	}

	event := kafka.NewGradePublishedEvent(grade.ID, grade.AssignmentID, grade.StudentID, grade.Score, grade.AdjustedScore)
	if err := s.producer.PublishEvent(ctx, event); err != nil {
		fmt.Printf("Failed to publish grade.published event: %v\n", err)
	}

	return grade, nil
}

// GetGrade retrieves a grade by ID
func (s *gradingService) GetGrade(ctx context.Context, id string) (*models.Grade, error) {
	grade, err := s.gradeRepo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get grade: %w", err)
	}
	return grade, nil
}

// AutoGrade triggers percentile-based auto-grading for an assignment
func (s *gradingService) AutoGrade(ctx context.Context, assignmentID string) error {
	return s.engine.AutoGrade(ctx, s.db, assignmentID)
}
