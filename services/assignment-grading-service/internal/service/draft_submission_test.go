package service

import (
	"context"
	"testing"
	"time"

	"slate/services/assignment-grading-service/internal/models"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestSubmissionService_UpsertDraftIgnoresDueDate(t *testing.T) {
	ctx := context.Background()
	assignmentRepo := new(MockAssignmentRepository)
	submissionRepo := new(MockSubmissionRepository)
	storage := new(MockFileStorage)
	producer := new(MockKafkaProducer)
	svc := NewSubmissionService(assignmentRepo, submissionRepo, storage, producer)

	// Due date long in the past — a real submission would be late. The draft
	// flow must ignore that and not set IsLate / DaysLate.
	past := time.Now().Add(-72 * time.Hour)
	assignment := &models.Assignment{
		ID:       "a1",
		TenantID: "t1",
		CourseID: "c1",
		DueDate:  past,
		LatePolicy: models.LatePolicy{
			PenaltyPercentPerDay: 25,
			MaxLateDays:          5,
		},
	}
	assignmentRepo.On("GetByID", ctx, "a1").Return(assignment, nil)
	submissionRepo.On("UpsertDraft", ctx, mock.AnythingOfType("*models.Submission")).Return(nil)

	draft, err := svc.UpsertDraft(ctx, "", "a1", "s1", []string{"url://file-1"})
	assert.NoError(t, err)
	assert.NotNil(t, draft)
	assert.True(t, draft.IsDraft)
	assert.Equal(t, models.StatusDraft, draft.Status)
	assert.False(t, draft.IsLate, "drafts must not be marked late even past due date")
	assert.Equal(t, 0, draft.DaysLate)
	assert.Equal(t, "t1", draft.TenantID, "tenant falls back to assignment when caller omits")

	assignmentRepo.AssertExpectations(t)
	submissionRepo.AssertExpectations(t)
}

func TestSubmissionService_GetDraft(t *testing.T) {
	ctx := context.Background()
	t.Run("returns draft when present", func(t *testing.T) {
		submissionRepo := new(MockSubmissionRepository)
		svc := NewSubmissionService(new(MockAssignmentRepository), submissionRepo, new(MockFileStorage), new(MockKafkaProducer))
		draft := &models.Submission{ID: "s1", AssignmentID: "a1", StudentID: "u1", IsDraft: true, Status: models.StatusDraft}
		submissionRepo.On("GetDraft", ctx, "a1", "u1").Return(draft, nil)

		got, err := svc.GetDraft(ctx, "a1", "u1")
		assert.NoError(t, err)
		assert.Equal(t, "s1", got.ID)
	})

	t.Run("returns error when missing", func(t *testing.T) {
		submissionRepo := new(MockSubmissionRepository)
		svc := NewSubmissionService(new(MockAssignmentRepository), submissionRepo, new(MockFileStorage), new(MockKafkaProducer))
		submissionRepo.On("GetDraft", ctx, "a1", "u1").Return(nil, nil)

		_, err := svc.GetDraft(ctx, "a1", "u1")
		assert.Error(t, err)
	})
}
