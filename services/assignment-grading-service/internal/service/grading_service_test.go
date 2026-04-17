package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"slate/services/assignment-grading-service/internal/models"
	"slate/services/assignment-grading-service/internal/repository"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockGradeRepository is a mock implementation of GradeRepository
type MockGradeRepository struct {
	mock.Mock
}

func (m *MockGradeRepository) Create(ctx context.Context, grade *models.Grade) error {
	args := m.Called(ctx, grade)
	return args.Error(0)
}

func (m *MockGradeRepository) GetByID(ctx context.Context, id string) (*models.Grade, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Grade), args.Error(1)
}

func (m *MockGradeRepository) GetBySubmission(ctx context.Context, submissionID string) (*models.Grade, error) {
	args := m.Called(ctx, submissionID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Grade), args.Error(1)
}

func (m *MockGradeRepository) Update(ctx context.Context, grade *models.Grade) error {
	args := m.Called(ctx, grade)
	return args.Error(0)
}

func (m *MockGradeRepository) ListByStudent(ctx context.Context, studentID, courseID string) ([]*models.Grade, error) {
	args := m.Called(ctx, studentID, courseID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.Grade), args.Error(1)
}

func (m *MockGradeRepository) ListByCourse(ctx context.Context, courseID string) ([]*models.Grade, error) {
	args := m.Called(ctx, courseID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.Grade), args.Error(1)
}

func (m *MockGradeRepository) GetStatistics(ctx context.Context, assignmentID string) (*repository.GradeStatistics, error) {
	args := m.Called(ctx, assignmentID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*repository.GradeStatistics), args.Error(1)
}

func (m *MockGradeRepository) ListByStudentTenant(ctx context.Context, tenantID, studentID string) ([]*models.Grade, error) {
	args := m.Called(ctx, tenantID, studentID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.Grade), args.Error(1)
}

func (m *MockGradeRepository) ListByAssignment(ctx context.Context, assignmentID string) ([]*models.Grade, error) {
	args := m.Called(ctx, assignmentID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.Grade), args.Error(1)
}

func (m *MockGradeRepository) UpdatePercentile(ctx context.Context, gradeID string, percentile float64) error {
	args := m.Called(ctx, gradeID, percentile)
	return args.Error(0)
}

func (m *MockGradeRepository) GetScoresForAssignment(ctx context.Context, assignmentID string) ([]repository.GradeScore, error) {
	args := m.Called(ctx, assignmentID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]repository.GradeScore), args.Error(1)
}

func (m *MockGradeRepository) CreateEmpty(ctx context.Context, tenantID, assignmentID, studentID, courseID string) error {
	args := m.Called(ctx, tenantID, assignmentID, studentID, courseID)
	return args.Error(0)
}

func TestGradingService_CreateGrade(t *testing.T) {
	ctx := context.Background()

	t.Run("create grade for on-time submission", func(t *testing.T) {
		mockAssignmentRepo := new(MockAssignmentRepository)
		mockSubmissionRepo := new(MockSubmissionRepository)
		mockGradeRepo := new(MockGradeRepository)
		mockProducer := new(MockKafkaProducer)
		service := NewGradingService(mockAssignmentRepo, mockSubmissionRepo, mockGradeRepo, mockProducer)

		submission := &models.Submission{
			ID:           "submission-id",
			AssignmentID: "assignment-id",
			StudentID:    "STUDENT-001",
			IsLate:       false,
			DaysLate:     0,
		}

		assignment := &models.Assignment{
			ID:        "assignment-id",
			CourseID:  "COURSE-001",
			Title:     "Assignment 1",
			MaxPoints: 100.0,
			LatePolicy: models.LatePolicy{
				PenaltyPercentPerDay: 10,
				MaxLateDays:          3,
			},
		}

		mockSubmissionRepo.On("GetByID", ctx, "submission-id").Return(submission, nil)
		mockAssignmentRepo.On("GetByID", ctx, "assignment-id").Return(assignment, nil)
		mockGradeRepo.On("Create", ctx, mock.AnythingOfType("*models.Grade")).Return(nil)
		mockSubmissionRepo.On("Update", ctx, mock.AnythingOfType("*models.Submission")).Return(nil)
		mockProducer.On("PublishEvent", ctx, mock.Anything).Return(nil)

		grade, err := service.CreateGrade(ctx, "submission-id", 95.0, "Great work!", "INSTRUCTOR-001")

		assert.NoError(t, err)
		assert.NotNil(t, grade)
		assert.Equal(t, 95.0, grade.Score)
		assert.Equal(t, 95.0, grade.AdjustedScore) // No penalty for on-time
		assert.Equal(t, models.GradeStatusDraft, grade.Status)
		assert.Equal(t, "Great work!", grade.Feedback)
		mockSubmissionRepo.AssertExpectations(t)
		mockAssignmentRepo.AssertExpectations(t)
		mockGradeRepo.AssertExpectations(t)
	})

	t.Run("create grade with late penalty", func(t *testing.T) {
		mockAssignmentRepo := new(MockAssignmentRepository)
		mockSubmissionRepo := new(MockSubmissionRepository)
		mockGradeRepo := new(MockGradeRepository)
		mockProducer := new(MockKafkaProducer)
		service := NewGradingService(mockAssignmentRepo, mockSubmissionRepo, mockGradeRepo, mockProducer)

		submission := &models.Submission{
			ID:           "submission-id",
			AssignmentID: "assignment-id",
			StudentID:    "STUDENT-001",
			IsLate:       true,
			DaysLate:     1,
		}

		assignment := &models.Assignment{
			ID:        "assignment-id",
			CourseID:  "COURSE-001",
			Title:     "Assignment 1",
			MaxPoints: 100.0,
			LatePolicy: models.LatePolicy{
				PenaltyPercentPerDay: 10,
				MaxLateDays:          3,
			},
		}

		mockSubmissionRepo.On("GetByID", ctx, "submission-id").Return(submission, nil)
		mockAssignmentRepo.On("GetByID", ctx, "assignment-id").Return(assignment, nil)
		mockGradeRepo.On("Create", ctx, mock.AnythingOfType("*models.Grade")).Return(nil)
		mockSubmissionRepo.On("Update", ctx, mock.AnythingOfType("*models.Submission")).Return(nil)
		mockProducer.On("PublishEvent", ctx, mock.Anything).Return(nil)

		grade, err := service.CreateGrade(ctx, "submission-id", 90.0, "Good work, but late", "INSTRUCTOR-001")

		assert.NoError(t, err)
		assert.NotNil(t, grade)
		assert.Equal(t, 90.0, grade.Score)
		assert.Equal(t, 81.0, grade.AdjustedScore) // 90 - 10% = 81
		mockSubmissionRepo.AssertExpectations(t)
		mockAssignmentRepo.AssertExpectations(t)
		mockGradeRepo.AssertExpectations(t)
	})

	t.Run("score exceeds max points", func(t *testing.T) {
		mockAssignmentRepo := new(MockAssignmentRepository)
		mockSubmissionRepo := new(MockSubmissionRepository)
		mockGradeRepo := new(MockGradeRepository)
		mockProducer := new(MockKafkaProducer)
		service := NewGradingService(mockAssignmentRepo, mockSubmissionRepo, mockGradeRepo, mockProducer)

		submission := &models.Submission{
			ID:           "submission-id",
			AssignmentID: "assignment-id",
			StudentID:    "STUDENT-001",
			IsLate:       false,
			DaysLate:     0,
		}

		assignment := &models.Assignment{
			ID:        "assignment-id",
			CourseID:  "COURSE-001",
			Title:     "Assignment 1",
			MaxPoints: 100.0,
			LatePolicy: models.LatePolicy{
				PenaltyPercentPerDay: 10,
				MaxLateDays:          3,
			},
		}

		mockSubmissionRepo.On("GetByID", ctx, "submission-id").Return(submission, nil)
		mockAssignmentRepo.On("GetByID", ctx, "assignment-id").Return(assignment, nil)

		grade, err := service.CreateGrade(ctx, "submission-id", 150.0, "Invalid score", "INSTRUCTOR-001")

		assert.Error(t, err)
		assert.Nil(t, grade)
		assert.Contains(t, err.Error(), "score must be between")
		mockGradeRepo.AssertNotCalled(t, "Create")
	})

	t.Run("negative score", func(t *testing.T) {
		mockAssignmentRepo := new(MockAssignmentRepository)
		mockSubmissionRepo := new(MockSubmissionRepository)
		mockGradeRepo := new(MockGradeRepository)
		mockProducer := new(MockKafkaProducer)
		service := NewGradingService(mockAssignmentRepo, mockSubmissionRepo, mockGradeRepo, mockProducer)

		submission := &models.Submission{
			ID:           "submission-id",
			AssignmentID: "assignment-id",
			StudentID:    "STUDENT-001",
			IsLate:       false,
			DaysLate:     0,
		}

		assignment := &models.Assignment{
			ID:        "assignment-id",
			CourseID:  "COURSE-001",
			Title:     "Assignment 1",
			MaxPoints: 100.0,
			LatePolicy: models.LatePolicy{
				PenaltyPercentPerDay: 10,
				MaxLateDays:          3,
			},
		}

		mockSubmissionRepo.On("GetByID", ctx, "submission-id").Return(submission, nil)
		mockAssignmentRepo.On("GetByID", ctx, "assignment-id").Return(assignment, nil)

		grade, err := service.CreateGrade(ctx, "submission-id", -10.0, "Invalid", "INSTRUCTOR-001")

		assert.Error(t, err)
		assert.Nil(t, grade)
		assert.Contains(t, err.Error(), "score must be between")
		mockGradeRepo.AssertNotCalled(t, "Create")
	})

	t.Run("submission not found", func(t *testing.T) {
		mockAssignmentRepo := new(MockAssignmentRepository)
		mockSubmissionRepo := new(MockSubmissionRepository)
		mockGradeRepo := new(MockGradeRepository)
		mockProducer := new(MockKafkaProducer)
		service := NewGradingService(mockAssignmentRepo, mockSubmissionRepo, mockGradeRepo, mockProducer)

		mockSubmissionRepo.On("GetByID", ctx, "non-existent").Return(nil, errors.New("submission not found"))

		grade, err := service.CreateGrade(ctx, "non-existent", 90.0, "Test", "INSTRUCTOR-001")

		assert.Error(t, err)
		assert.Nil(t, grade)
		assert.Contains(t, err.Error(), "failed to get submission")
		mockGradeRepo.AssertNotCalled(t, "Create")
	})
}

func TestGradingService_UpdateGrade(t *testing.T) {
	ctx := context.Background()

	t.Run("update draft grade successfully", func(t *testing.T) {
		mockAssignmentRepo := new(MockAssignmentRepository)
		mockSubmissionRepo := new(MockSubmissionRepository)
		mockGradeRepo := new(MockGradeRepository)
		mockProducer := new(MockKafkaProducer)
		service := NewGradingService(mockAssignmentRepo, mockSubmissionRepo, mockGradeRepo, mockProducer)

		existingGrade := &models.Grade{
			ID:            "grade-id",
			SubmissionID:  "submission-id",
			StudentID:     "STUDENT-001",
			AssignmentID:  "assignment-id",
			Score:         80.0,
			AdjustedScore: 80.0,
			Status:        models.GradeStatusDraft,
			GradedBy:      "INSTRUCTOR-001",
		}

		submission := &models.Submission{
			ID:           "submission-id",
			AssignmentID: "assignment-id",
			StudentID:    "STUDENT-001",
			IsLate:       false,
			DaysLate:     0,
		}

		assignment := &models.Assignment{
			ID:        "assignment-id",
			CourseID:  "COURSE-001",
			MaxPoints: 100.0,
			LatePolicy: models.LatePolicy{
				PenaltyPercentPerDay: 10,
				MaxLateDays:          3,
			},
		}

		mockGradeRepo.On("GetByID", ctx, "grade-id").Return(existingGrade, nil)
		mockSubmissionRepo.On("GetByID", ctx, "submission-id").Return(submission, nil)
		mockAssignmentRepo.On("GetByID", ctx, "assignment-id").Return(assignment, nil)
		mockGradeRepo.On("Update", ctx, mock.AnythingOfType("*models.Grade")).Return(nil)

		grade, err := service.UpdateGrade(ctx, "grade-id", 95.0, "Updated feedback")

		assert.NoError(t, err)
		assert.NotNil(t, grade)
		assert.Equal(t, 95.0, grade.Score)
		assert.Equal(t, 95.0, grade.AdjustedScore)
		assert.Equal(t, "Updated feedback", grade.Feedback)
		mockGradeRepo.AssertExpectations(t)
	})

	t.Run("cannot update published grade", func(t *testing.T) {
		mockAssignmentRepo := new(MockAssignmentRepository)
		mockSubmissionRepo := new(MockSubmissionRepository)
		mockGradeRepo := new(MockGradeRepository)
		mockProducer := new(MockKafkaProducer)
		service := NewGradingService(mockAssignmentRepo, mockSubmissionRepo, mockGradeRepo, mockProducer)

		publishedGrade := &models.Grade{
			ID:            "grade-id",
			SubmissionID:  "submission-id",
			StudentID:     "STUDENT-001",
			AssignmentID:  "assignment-id",
			Score:         80.0,
			AdjustedScore: 80.0,
			Status:        models.GradeStatusPublished,
			GradedBy:      "INSTRUCTOR-001",
			PublishedAt:   &time.Time{},
		}

		mockGradeRepo.On("GetByID", ctx, "grade-id").Return(publishedGrade, nil)

		grade, err := service.UpdateGrade(ctx, "grade-id", 95.0, "Updated feedback")

		assert.Error(t, err)
		assert.Nil(t, grade)
		assert.Contains(t, err.Error(), "only draft grades can be updated")
		mockGradeRepo.AssertNotCalled(t, "Update")
	})

	t.Run("grade not found", func(t *testing.T) {
		mockAssignmentRepo := new(MockAssignmentRepository)
		mockSubmissionRepo := new(MockSubmissionRepository)
		mockGradeRepo := new(MockGradeRepository)
		mockProducer := new(MockKafkaProducer)
		service := NewGradingService(mockAssignmentRepo, mockSubmissionRepo, mockGradeRepo, mockProducer)

		mockGradeRepo.On("GetByID", ctx, "non-existent").Return(nil, errors.New("grade not found"))

		grade, err := service.UpdateGrade(ctx, "non-existent", 95.0, "Updated")

		assert.Error(t, err)
		assert.Nil(t, grade)
		assert.Contains(t, err.Error(), "failed to get grade")
	})
}

func TestGradingService_PublishGrade(t *testing.T) {
	ctx := context.Background()

	t.Run("publish draft grade successfully", func(t *testing.T) {
		mockAssignmentRepo := new(MockAssignmentRepository)
		mockSubmissionRepo := new(MockSubmissionRepository)
		mockGradeRepo := new(MockGradeRepository)
		mockProducer := new(MockKafkaProducer)
		service := NewGradingService(mockAssignmentRepo, mockSubmissionRepo, mockGradeRepo, mockProducer)

		draftGrade := &models.Grade{
			ID:            "grade-id",
			SubmissionID:  "submission-id",
			StudentID:     "STUDENT-001",
			AssignmentID:  "assignment-id",
			Score:         90.0,
			AdjustedScore: 90.0,
			Status:        models.GradeStatusDraft,
			GradedBy:      "INSTRUCTOR-001",
		}

		submission := &models.Submission{
			ID:           "submission-id",
			AssignmentID: "assignment-id",
			StudentID:    "STUDENT-001",
			Status:       models.StatusGraded,
		}

		mockGradeRepo.On("GetByID", ctx, "grade-id").Return(draftGrade, nil)
		mockGradeRepo.On("Update", ctx, mock.AnythingOfType("*models.Grade")).Return(nil)
		mockSubmissionRepo.On("GetByID", ctx, "submission-id").Return(submission, nil)
		mockSubmissionRepo.On("Update", ctx, mock.AnythingOfType("*models.Submission")).Return(nil)
		mockProducer.On("PublishEvent", ctx, mock.Anything).Return(nil)

		grade, err := service.PublishGrade(ctx, "grade-id")

		assert.NoError(t, err)
		assert.NotNil(t, grade)
		assert.Equal(t, models.GradeStatusPublished, grade.Status)
		assert.NotNil(t, grade.PublishedAt)
		mockGradeRepo.AssertExpectations(t)
		mockSubmissionRepo.AssertExpectations(t)
		mockProducer.AssertExpectations(t)
	})

	t.Run("cannot publish already published grade", func(t *testing.T) {
		mockAssignmentRepo := new(MockAssignmentRepository)
		mockSubmissionRepo := new(MockSubmissionRepository)
		mockGradeRepo := new(MockGradeRepository)
		mockProducer := new(MockKafkaProducer)
		service := NewGradingService(mockAssignmentRepo, mockSubmissionRepo, mockGradeRepo, mockProducer)

		publishedGrade := &models.Grade{
			ID:            "grade-id",
			SubmissionID:  "submission-id",
			StudentID:     "STUDENT-001",
			AssignmentID:  "assignment-id",
			Score:         90.0,
			AdjustedScore: 90.0,
			Status:        models.GradeStatusPublished,
			GradedBy:      "INSTRUCTOR-001",
			PublishedAt:   &time.Time{},
		}

		mockGradeRepo.On("GetByID", ctx, "grade-id").Return(publishedGrade, nil)

		grade, err := service.PublishGrade(ctx, "grade-id")

		assert.Error(t, err)
		assert.Nil(t, grade)
		assert.Contains(t, err.Error(), "failed to publish grade")
		mockGradeRepo.AssertCalled(t, "GetByID", ctx, "grade-id")
		mockGradeRepo.AssertNotCalled(t, "Update")
	})

	t.Run("grade not found", func(t *testing.T) {
		mockAssignmentRepo := new(MockAssignmentRepository)
		mockSubmissionRepo := new(MockSubmissionRepository)
		mockGradeRepo := new(MockGradeRepository)
		mockProducer := new(MockKafkaProducer)
		service := NewGradingService(mockAssignmentRepo, mockSubmissionRepo, mockGradeRepo, mockProducer)

		mockGradeRepo.On("GetByID", ctx, "non-existent").Return(nil, errors.New("grade not found"))

		grade, err := service.PublishGrade(ctx, "non-existent")

		assert.Error(t, err)
		assert.Nil(t, grade)
		assert.Contains(t, err.Error(), "failed to get grade")
	})
}

func TestGradingService_GetGrade(t *testing.T) {
	ctx := context.Background()

	t.Run("get existing grade", func(t *testing.T) {
		mockAssignmentRepo := new(MockAssignmentRepository)
		mockSubmissionRepo := new(MockSubmissionRepository)
		mockGradeRepo := new(MockGradeRepository)
		mockProducer := new(MockKafkaProducer)
		service := NewGradingService(mockAssignmentRepo, mockSubmissionRepo, mockGradeRepo, mockProducer)

		expectedGrade := &models.Grade{
			ID:            "grade-id",
			SubmissionID:  "submission-id",
			StudentID:     "STUDENT-001",
			AssignmentID:  "assignment-id",
			Score:         90.0,
			AdjustedScore: 90.0,
			Status:        models.GradeStatusPublished,
		}

		mockGradeRepo.On("GetByID", ctx, "grade-id").Return(expectedGrade, nil)

		grade, err := service.GetGrade(ctx, "grade-id")

		assert.NoError(t, err)
		assert.Equal(t, expectedGrade, grade)
		mockGradeRepo.AssertExpectations(t)
	})

	t.Run("grade not found", func(t *testing.T) {
		mockAssignmentRepo := new(MockAssignmentRepository)
		mockSubmissionRepo := new(MockSubmissionRepository)
		mockGradeRepo := new(MockGradeRepository)
		mockProducer := new(MockKafkaProducer)
		service := NewGradingService(mockAssignmentRepo, mockSubmissionRepo, mockGradeRepo, mockProducer)

		mockGradeRepo.On("GetByID", ctx, "non-existent").Return(nil, errors.New("grade not found"))

		grade, err := service.GetGrade(ctx, "non-existent")

		assert.Error(t, err)
		assert.Nil(t, grade)
		assert.Contains(t, err.Error(), "failed to get grade")
		mockGradeRepo.AssertExpectations(t)
	})
}
