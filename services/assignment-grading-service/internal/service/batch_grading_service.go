package service

import (
	"context"
	"encoding/json"
	"fmt"

	"slate/libs/common-go/tracing"
	"slate/services/assignment-grading-service/internal/grading"
	"slate/services/assignment-grading-service/internal/models"
	"slate/services/assignment-grading-service/internal/repository"
	"slate/services/assignment-grading-service/pkg/kafka"
)

// RubricScoreInput is one row's score in a batch grading request.
type RubricScoreInput struct {
	RowID  string  `json:"row_id"`
	Points float64 `json:"points"`
}

// BatchGradeRequest is the payload accepted by POST /grades/batch (W9.5).
type BatchGradeRequest struct {
	PatternID        string             `json:"pattern_id"`
	AssignmentID     string             `json:"assignment_id"`
	SubmissionIDs    []string           `json:"submission_ids"`
	RubricScores     []RubricScoreInput `json:"rubric_scores"`
	FeedbackTemplate string             `json:"feedback_template"`
	GradedBy         string             `json:"graded_by"`
	InstructorID     string             `json:"instructor_id"`
}

// BatchGradeResult summarises a completed batch.
type BatchGradeResult struct {
	PatternID    string   `json:"pattern_id"`
	AssignmentID string   `json:"assignment_id"`
	GradedCount  int      `json:"graded_count"`
	GradeIDs     []string `json:"grade_ids"`
}

// BatchGradingService applies a single rubric + feedback template to every
// submission in a pattern and emits `grade.updated` per student.
type BatchGradingService interface {
	Apply(ctx context.Context, req BatchGradeRequest) (*BatchGradeResult, error)
}

type batchGradingService struct {
	assignmentRepo repository.AssignmentRepository
	submissionRepo repository.SubmissionRepository
	gradeRepo      repository.GradeRepository
	producer       kafka.EventPublisher
	latePolicyCalc *LatePolicyCalculator
}

// NewBatchGradingService constructs the default batch grader.
func NewBatchGradingService(
	assignmentRepo repository.AssignmentRepository,
	submissionRepo repository.SubmissionRepository,
	gradeRepo repository.GradeRepository,
	producer kafka.EventPublisher,
) BatchGradingService {
	return &batchGradingService{
		assignmentRepo: assignmentRepo,
		submissionRepo: submissionRepo,
		gradeRepo:      gradeRepo,
		producer:       producer,
		latePolicyCalc: NewLatePolicyCalculator(),
	}
}

// Apply creates (or updates) a grade for every submission_id in the request,
// applying the shared rubric scores + feedback template, and emitting
// grade.updated events. Returns the set of grade IDs written.
func (s *batchGradingService) Apply(ctx context.Context, req BatchGradeRequest) (*BatchGradeResult, error) {
	if req.AssignmentID == "" {
		return nil, fmt.Errorf("assignment_id is required")
	}
	if req.PatternID == "" {
		return nil, fmt.Errorf("pattern_id is required")
	}
	if len(req.SubmissionIDs) == 0 {
		return nil, fmt.Errorf("submission_ids is required")
	}
	if req.GradedBy == "" {
		req.GradedBy = req.InstructorID
	}
	if req.GradedBy == "" {
		return nil, fmt.Errorf("graded_by is required")
	}

	assignment, err := s.assignmentRepo.GetByID(ctx, req.AssignmentID)
	if err != nil {
		return nil, fmt.Errorf("get assignment: %w", err)
	}

	totalScore := 0.0
	rubricMap := map[string]interface{}{}
	for _, rs := range req.RubricScores {
		totalScore += rs.Points
		rubricMap[rs.RowID] = rs.Points
	}
	if totalScore > assignment.MaxPoints {
		return nil, fmt.Errorf("rubric total %.2f exceeds max_points %.2f", totalScore, assignment.MaxPoints)
	}

	result := &BatchGradeResult{
		PatternID:    req.PatternID,
		AssignmentID: req.AssignmentID,
	}

	for _, submissionID := range req.SubmissionIDs {
		sub, err := s.submissionRepo.GetByID(ctx, submissionID)
		if err != nil {
			return nil, fmt.Errorf("get submission %s: %w", submissionID, err)
		}
		if sub.AssignmentID != assignment.ID {
			return nil, fmt.Errorf("submission %s does not belong to assignment %s", submissionID, assignment.ID)
		}
		if sub.IsDraft {
			continue
		}

		adjusted := s.latePolicyCalc.ApplyPenalty(totalScore, sub.DaysLate, assignment.LatePolicy)
		percentage := 0.0
		if assignment.MaxPoints > 0 {
			percentage = (adjusted / assignment.MaxPoints) * 100
		}
		letter := grading.LetterGrade(percentage, nil)

		// If a grade already exists, update it; otherwise create.
		existing, _ := s.gradeRepo.GetBySubmission(ctx, sub.ID)
		var grade *models.Grade
		if existing != nil {
			existing.Score = totalScore
			existing.MaxScore = assignment.MaxPoints
			existing.AdjustedScore = adjusted
			existing.Percentage = percentage
			existing.LetterGrade = letter
			existing.Feedback = req.FeedbackTemplate
			existing.RubricScores = rubricMap
			existing.GradedBy = req.GradedBy
			if err := s.gradeRepo.Update(ctx, existing); err != nil {
				return nil, fmt.Errorf("update grade for %s: %w", sub.ID, err)
			}
			grade = existing
		} else {
			grade = &models.Grade{
				TenantID:      assignment.TenantID,
				SubmissionID:  sub.ID,
				StudentID:     sub.StudentID,
				AssignmentID:  assignment.ID,
				CourseID:      assignment.CourseID,
				Score:         totalScore,
				MaxScore:      assignment.MaxPoints,
				AdjustedScore: adjusted,
				Percentage:    percentage,
				LetterGrade:   letter,
				RubricScores:  rubricMap,
				Feedback:      req.FeedbackTemplate,
				Status:        models.GradeStatusDraft,
				GradedBy:      req.GradedBy,
			}
			if err := grade.Validate(); err != nil {
				return nil, fmt.Errorf("validate grade for %s: %w", sub.ID, err)
			}
			if err := s.gradeRepo.Create(ctx, grade); err != nil {
				return nil, fmt.Errorf("create grade for %s: %w", sub.ID, err)
			}
		}

		sub.Status = models.StatusGraded
		if err := s.submissionRepo.Update(ctx, sub); err != nil {
			return nil, fmt.Errorf("update submission %s: %w", sub.ID, err)
		}

		evt := kafka.NewGradeUpdatedEvent(kafka.GradeUpdatedPayload{
			GradeID:         grade.ID,
			AssignmentID:    grade.AssignmentID,
			AssignmentTitle: assignment.Title,
			StudentID:       grade.StudentID,
			TenantID:        grade.TenantID,
			TenantSlug:      tracing.TenantSlugFromContext(ctx),
			CourseID:        assignment.CourseID,
			Score:           grade.Score,
			MaxScore:        assignment.MaxPoints,
			AdjustedScore:   grade.AdjustedScore,
			PatternID:       req.PatternID,
		})
		if err := s.producer.PublishEvent(ctx, evt); err != nil {
			return nil, fmt.Errorf("publish grade.updated for %s: %w", sub.ID, err)
		}

		result.GradeIDs = append(result.GradeIDs, grade.ID)
		result.GradedCount++
	}

	return result, nil
}

// MarshalRubricScores is a small helper for handlers that want to echo back
// the request rubric scores as JSON.
func MarshalRubricScores(scores []RubricScoreInput) ([]byte, error) {
	return json.Marshal(scores)
}
