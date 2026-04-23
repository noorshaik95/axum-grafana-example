package service

import (
	"context"
	"testing"

	"slate/services/assignment-grading-service/internal/models"
	"slate/services/assignment-grading-service/pkg/kafka"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestBatchGrading_ApplyPerPatternEmitsKafkaPerStudent(t *testing.T) {
	ctx := context.Background()
	assignmentRepo := new(MockAssignmentRepository)
	submissionRepo := new(MockSubmissionRepository)
	gradeRepo := new(MockGradeRepository)
	producer := kafka.NewMockPublisher()

	assignment := &models.Assignment{
		ID:        "a1",
		Title:     "PS1",
		TenantID:  "t1",
		CourseID:  "c1",
		MaxPoints: 100,
	}
	assignmentRepo.On("GetByID", ctx, "a1").Return(assignment, nil)

	s1 := &models.Submission{ID: "s1", AssignmentID: "a1", StudentID: "u1", Status: models.StatusSubmitted}
	s2 := &models.Submission{ID: "s2", AssignmentID: "a1", StudentID: "u2", Status: models.StatusSubmitted}
	submissionRepo.On("GetByID", ctx, "s1").Return(s1, nil)
	submissionRepo.On("GetByID", ctx, "s2").Return(s2, nil)

	// No pre-existing grades → create path.
	gradeRepo.On("GetBySubmission", ctx, "s1").Return(nil, nil)
	gradeRepo.On("GetBySubmission", ctx, "s2").Return(nil, nil)
	gradeRepo.On("Create", ctx, mock.AnythingOfType("*models.Grade")).Return(nil).Twice()

	submissionRepo.On("Update", ctx, mock.AnythingOfType("*models.Submission")).Return(nil).Twice()

	svc := NewBatchGradingService(assignmentRepo, submissionRepo, gradeRepo, producer)
	res, err := svc.Apply(ctx, BatchGradeRequest{
		PatternID:     "pattern-001",
		AssignmentID:  "a1",
		SubmissionIDs: []string{"s1", "s2"},
		RubricScores: []RubricScoreInput{
			{RowID: "r1", Points: 30},
			{RowID: "r2", Points: 20},
		},
		FeedbackTemplate: "Nice work on the happy path; revisit edge cases.",
		GradedBy:         "instr-1",
	})

	assert.NoError(t, err)
	assert.Equal(t, 2, res.GradedCount)
	assert.Len(t, res.GradeIDs, 2)

	events := producer.EventsOfType(kafka.EventTypeGradeUpdated)
	assert.Len(t, events, 2, "expected one grade.updated event per student")
	for _, e := range events {
		// Three-consumer parity guard: ai-service + metrics-service read
		// tenant_slug + course_id; ai-service reads user_id (alias of
		// student_id); email-service reads assignment_title + max_score.
		// Legacy fields retained for anyone still keyed on them.
		assert.Equal(t, "a1", e.Data["assignment_id"])
		assert.Equal(t, "PS1", e.Data["assignment_title"], "email-service requires assignment_title")
		assert.Equal(t, "pattern-001", e.Data["pattern_id"])
		assert.Equal(t, float64(50), e.Data["score"])
		assert.Equal(t, float64(100), e.Data["max_score"], "email-service requires max_score")
		assert.Equal(t, "c1", e.Data["course_id"], "metrics + ai consumers require course_id")
		assert.Contains(t, e.Data, "tenant_slug", "metrics + ai consumers require tenant_slug")
		assert.Equal(t, e.Data["student_id"], e.Data["user_id"], "ai-service reads user_id; must alias student_id")
		assert.Contains(t, e.Data, "student_email", "email-service expects student_email key (empty from producer; lazy lookup on consumer)")
	}
}

func TestBatchGrading_RejectsOverMaxPoints(t *testing.T) {
	ctx := context.Background()
	assignmentRepo := new(MockAssignmentRepository)
	submissionRepo := new(MockSubmissionRepository)
	gradeRepo := new(MockGradeRepository)
	producer := kafka.NewMockPublisher()

	assignmentRepo.On("GetByID", ctx, "a1").Return(&models.Assignment{ID: "a1", MaxPoints: 50}, nil)
	svc := NewBatchGradingService(assignmentRepo, submissionRepo, gradeRepo, producer)

	_, err := svc.Apply(ctx, BatchGradeRequest{
		PatternID:     "pattern-x",
		AssignmentID:  "a1",
		SubmissionIDs: []string{"s1"},
		RubricScores:  []RubricScoreInput{{RowID: "r1", Points: 80}},
		GradedBy:      "instr-1",
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "exceeds max_points")
}

func TestBatchGrading_UpdatesExistingGrade(t *testing.T) {
	ctx := context.Background()
	assignmentRepo := new(MockAssignmentRepository)
	submissionRepo := new(MockSubmissionRepository)
	gradeRepo := new(MockGradeRepository)
	producer := kafka.NewMockPublisher()

	assignmentRepo.On("GetByID", ctx, "a1").Return(&models.Assignment{ID: "a1", TenantID: "t1", CourseID: "c1", MaxPoints: 100}, nil)
	sub := &models.Submission{ID: "s1", AssignmentID: "a1", StudentID: "u1", Status: models.StatusSubmitted}
	submissionRepo.On("GetByID", ctx, "s1").Return(sub, nil)
	existing := &models.Grade{ID: "g-existing", SubmissionID: "s1", AssignmentID: "a1", StudentID: "u1", Status: models.GradeStatusDraft}
	gradeRepo.On("GetBySubmission", ctx, "s1").Return(existing, nil)
	gradeRepo.On("Update", ctx, existing).Return(nil)
	submissionRepo.On("Update", ctx, sub).Return(nil)

	svc := NewBatchGradingService(assignmentRepo, submissionRepo, gradeRepo, producer)
	res, err := svc.Apply(ctx, BatchGradeRequest{
		PatternID:        "pattern-y",
		AssignmentID:     "a1",
		SubmissionIDs:    []string{"s1"},
		RubricScores:     []RubricScoreInput{{RowID: "r1", Points: 75}},
		FeedbackTemplate: "Retry test 3",
		GradedBy:         "instr-1",
	})
	assert.NoError(t, err)
	assert.Equal(t, 1, res.GradedCount)
	assert.Equal(t, "g-existing", res.GradeIDs[0])
	assert.Equal(t, 75.0, existing.Score)
}
