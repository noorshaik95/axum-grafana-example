package service

import (
	"context"
	"errors"
	"io"
	"testing"
	"time"

	"slate/services/assignment-grading-service/internal/models"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockSubmissionRepository is a mock implementation of SubmissionRepository
type MockSubmissionRepository struct {
	mock.Mock
}

func (m *MockSubmissionRepository) Create(ctx context.Context, submission *models.Submission) error {
	args := m.Called(ctx, submission)
	return args.Error(0)
}

func (m *MockSubmissionRepository) GetByID(ctx context.Context, id string) (*models.Submission, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Submission), args.Error(1)
}

func (m *MockSubmissionRepository) GetByAssignmentAndStudent(ctx context.Context, assignmentID, studentID string) (*models.Submission, error) {
	args := m.Called(ctx, assignmentID, studentID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Submission), args.Error(1)
}

func (m *MockSubmissionRepository) Update(ctx context.Context, submission *models.Submission) error {
	args := m.Called(ctx, submission)
	return args.Error(0)
}

func (m *MockSubmissionRepository) ListByAssignment(ctx context.Context, assignmentID, sortBy, order string) ([]*models.Submission, error) {
	args := m.Called(ctx, assignmentID, sortBy, order)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.Submission), args.Error(1)
}

func (m *MockSubmissionRepository) ListByStudent(ctx context.Context, studentID, courseID string) ([]*models.Submission, error) {
	args := m.Called(ctx, studentID, courseID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.Submission), args.Error(1)
}

func (m *MockSubmissionRepository) ListByAssignmentPaginated(ctx context.Context, assignmentID, tenantID string, page, pageSize int) ([]*models.Submission, int, error) {
	args := m.Called(ctx, assignmentID, tenantID, page, pageSize)
	if args.Get(0) == nil {
		return nil, args.Int(1), args.Error(2)
	}
	return args.Get(0).([]*models.Submission), args.Int(1), args.Error(2)
}

// MockFileStorage is a mock implementation of FileStorage
type MockFileStorage struct {
	mock.Mock
}

func (m *MockFileStorage) Save(fileName string, reader io.Reader, contentType string) (string, error) {
	args := m.Called(fileName, reader, contentType)
	return args.String(0), args.Error(1)
}

func (m *MockFileStorage) Get(filePath string) (io.ReadCloser, error) {
	args := m.Called(filePath)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(io.ReadCloser), args.Error(1)
}

func (m *MockFileStorage) Delete(filePath string) error {
	args := m.Called(filePath)
	return args.Error(0)
}

func (m *MockFileStorage) Exists(filePath string) (bool, error) {
	args := m.Called(filePath)
	return args.Bool(0), args.Error(1)
}

func TestSubmissionService_SubmitAssignment(t *testing.T) {
	ctx := context.Background()

	t.Run("submit on-time assignment successfully", func(t *testing.T) {
		mockAssignmentRepo := new(MockAssignmentRepository)
		mockSubmissionRepo := new(MockSubmissionRepository)
		mockStorage := new(MockFileStorage)
		mockProducer := new(MockKafkaProducer)
		service := NewSubmissionService(mockAssignmentRepo, mockSubmissionRepo, mockStorage, mockProducer)

		dueDate := time.Now().Add(24 * time.Hour) // Due tomorrow
		assignment := &models.Assignment{
			ID:        "assignment-id",
			CourseID:  "COURSE-001",
			Title:     "Assignment 1",
			MaxPoints: 100.0,
			DueDate:   dueDate,
			LatePolicy: models.LatePolicy{
				PenaltyPercentPerDay: 10,
				MaxLateDays:          3,
			},
		}

		fileContent := []byte("submission content")

		mockAssignmentRepo.On("GetByID", ctx, "assignment-id").Return(assignment, nil)
		mockStorage.On("Save", "homework.pdf", mock.Anything, "application/pdf").Return("submissions/file123.pdf", nil)
		mockSubmissionRepo.On("Create", ctx, mock.AnythingOfType("*models.Submission")).Return(nil)
		mockProducer.On("PublishEvent", ctx, mock.Anything).Return(nil)

		submission, err := service.SubmitAssignment(ctx, "assignment-id", "STUDENT-001", fileContent, "homework.pdf", "application/pdf")

		assert.NoError(t, err)
		assert.NotNil(t, submission)
		assert.Equal(t, "assignment-id", submission.AssignmentID)
		assert.Equal(t, "STUDENT-001", submission.StudentID)
		assert.Equal(t, "submissions/file123.pdf", submission.FilePath)
		assert.False(t, submission.IsLate)
		assert.Equal(t, 0, submission.DaysLate)
		mockAssignmentRepo.AssertExpectations(t)
		mockStorage.AssertExpectations(t)
		mockSubmissionRepo.AssertExpectations(t)
		mockProducer.AssertExpectations(t)
	})

	t.Run("submit late assignment with penalty", func(t *testing.T) {
		mockAssignmentRepo := new(MockAssignmentRepository)
		mockSubmissionRepo := new(MockSubmissionRepository)
		mockStorage := new(MockFileStorage)
		mockProducer := new(MockKafkaProducer)
		service := NewSubmissionService(mockAssignmentRepo, mockSubmissionRepo, mockStorage, mockProducer)

		dueDate := time.Now().Add(-25 * time.Hour) // Due 25 hours ago (2 days late with rounding)
		assignment := &models.Assignment{
			ID:        "assignment-id",
			CourseID:  "COURSE-001",
			Title:     "Assignment 1",
			MaxPoints: 100.0,
			DueDate:   dueDate,
			LatePolicy: models.LatePolicy{
				PenaltyPercentPerDay: 10,
				MaxLateDays:          3,
			},
		}

		fileContent := []byte("submission content")

		mockAssignmentRepo.On("GetByID", ctx, "assignment-id").Return(assignment, nil)
		mockStorage.On("Save", "homework.pdf", mock.Anything, "application/pdf").Return("submissions/file123.pdf", nil)
		mockSubmissionRepo.On("Create", ctx, mock.AnythingOfType("*models.Submission")).Return(nil)
		mockProducer.On("PublishEvent", ctx, mock.Anything).Return(nil)

		submission, err := service.SubmitAssignment(ctx, "assignment-id", "STUDENT-001", fileContent, "homework.pdf", "application/pdf")

		assert.NoError(t, err)
		assert.NotNil(t, submission)
		assert.True(t, submission.IsLate)
		assert.Equal(t, 2, submission.DaysLate)
		mockAssignmentRepo.AssertExpectations(t)
		mockStorage.AssertExpectations(t)
		mockSubmissionRepo.AssertExpectations(t)
		mockProducer.AssertExpectations(t)
	})

	t.Run("assignment not found", func(t *testing.T) {
		mockAssignmentRepo := new(MockAssignmentRepository)
		mockSubmissionRepo := new(MockSubmissionRepository)
		mockStorage := new(MockFileStorage)
		mockProducer := new(MockKafkaProducer)
		service := NewSubmissionService(mockAssignmentRepo, mockSubmissionRepo, mockStorage, mockProducer)

		fileContent := []byte("submission content")

		mockAssignmentRepo.On("GetByID", ctx, "non-existent").Return(nil, errors.New("assignment not found"))

		submission, err := service.SubmitAssignment(ctx, "non-existent", "STUDENT-001", fileContent, "homework.pdf", "application/pdf")

		assert.Error(t, err)
		assert.Nil(t, submission)
		assert.Contains(t, err.Error(), "failed to get assignment")
		mockAssignmentRepo.AssertExpectations(t)
		mockStorage.AssertNotCalled(t, "Save")
		mockSubmissionRepo.AssertNotCalled(t, "Create")
	})

	t.Run("file storage fails", func(t *testing.T) {
		mockAssignmentRepo := new(MockAssignmentRepository)
		mockSubmissionRepo := new(MockSubmissionRepository)
		mockStorage := new(MockFileStorage)
		mockProducer := new(MockKafkaProducer)
		service := NewSubmissionService(mockAssignmentRepo, mockSubmissionRepo, mockStorage, mockProducer)

		dueDate := time.Now().Add(24 * time.Hour)
		assignment := &models.Assignment{
			ID:        "assignment-id",
			CourseID:  "COURSE-001",
			Title:     "Assignment 1",
			MaxPoints: 100.0,
			DueDate:   dueDate,
			LatePolicy: models.LatePolicy{
				PenaltyPercentPerDay: 10,
				MaxLateDays:          3,
			},
		}

		fileContent := []byte("submission content")

		mockAssignmentRepo.On("GetByID", ctx, "assignment-id").Return(assignment, nil)
		mockStorage.On("Save", "homework.pdf", mock.Anything, "application/pdf").Return("", errors.New("storage error"))

		submission, err := service.SubmitAssignment(ctx, "assignment-id", "STUDENT-001", fileContent, "homework.pdf", "application/pdf")

		assert.Error(t, err)
		assert.Nil(t, submission)
		assert.Contains(t, err.Error(), "failed to save file")
		mockAssignmentRepo.AssertExpectations(t)
		mockStorage.AssertExpectations(t)
		mockSubmissionRepo.AssertNotCalled(t, "Create")
	})

	t.Run("validation fails - clean up file", func(t *testing.T) {
		mockAssignmentRepo := new(MockAssignmentRepository)
		mockSubmissionRepo := new(MockSubmissionRepository)
		mockStorage := new(MockFileStorage)
		mockProducer := new(MockKafkaProducer)
		service := NewSubmissionService(mockAssignmentRepo, mockSubmissionRepo, mockStorage, mockProducer)

		dueDate := time.Now().Add(24 * time.Hour)
		assignment := &models.Assignment{
			ID:        "assignment-id",
			CourseID:  "COURSE-001",
			Title:     "Assignment 1",
			MaxPoints: 100.0,
			DueDate:   dueDate,
			LatePolicy: models.LatePolicy{
				PenaltyPercentPerDay: 10,
				MaxLateDays:          3,
			},
		}

		fileContent := []byte("submission content")

		mockAssignmentRepo.On("GetByID", ctx, "assignment-id").Return(assignment, nil)
		mockStorage.On("Save", "homework.pdf", mock.Anything, "application/pdf").Return("submissions/file123.pdf", nil)
		mockStorage.On("Delete", "submissions/file123.pdf").Return(nil)

		// Pass empty student ID to trigger validation error
		submission, err := service.SubmitAssignment(ctx, "assignment-id", "", fileContent, "homework.pdf", "application/pdf")

		assert.Error(t, err)
		assert.Nil(t, submission)
		assert.Contains(t, err.Error(), "validation failed")
		mockStorage.AssertCalled(t, "Delete", "submissions/file123.pdf") // File should be deleted
		mockSubmissionRepo.AssertNotCalled(t, "Create")
	})

	t.Run("repository error - clean up file", func(t *testing.T) {
		mockAssignmentRepo := new(MockAssignmentRepository)
		mockSubmissionRepo := new(MockSubmissionRepository)
		mockStorage := new(MockFileStorage)
		mockProducer := new(MockKafkaProducer)
		service := NewSubmissionService(mockAssignmentRepo, mockSubmissionRepo, mockStorage, mockProducer)

		dueDate := time.Now().Add(24 * time.Hour)
		assignment := &models.Assignment{
			ID:        "assignment-id",
			CourseID:  "COURSE-001",
			Title:     "Assignment 1",
			MaxPoints: 100.0,
			DueDate:   dueDate,
			LatePolicy: models.LatePolicy{
				PenaltyPercentPerDay: 10,
				MaxLateDays:          3,
			},
		}

		fileContent := []byte("submission content")

		mockAssignmentRepo.On("GetByID", ctx, "assignment-id").Return(assignment, nil)
		mockStorage.On("Save", "homework.pdf", mock.Anything, "application/pdf").Return("submissions/file123.pdf", nil)
		mockSubmissionRepo.On("Create", ctx, mock.AnythingOfType("*models.Submission")).Return(errors.New("database error"))
		mockStorage.On("Delete", "submissions/file123.pdf").Return(nil)

		submission, err := service.SubmitAssignment(ctx, "assignment-id", "STUDENT-001", fileContent, "homework.pdf", "application/pdf")

		assert.Error(t, err)
		assert.Nil(t, submission)
		assert.Contains(t, err.Error(), "failed to create submission")
		mockStorage.AssertCalled(t, "Delete", "submissions/file123.pdf") // File should be deleted
		mockSubmissionRepo.AssertExpectations(t)
	})
}

func TestSubmissionService_GetSubmission(t *testing.T) {
	ctx := context.Background()

	t.Run("get existing submission", func(t *testing.T) {
		mockAssignmentRepo := new(MockAssignmentRepository)
		mockSubmissionRepo := new(MockSubmissionRepository)
		mockStorage := new(MockFileStorage)
		mockProducer := new(MockKafkaProducer)
		service := NewSubmissionService(mockAssignmentRepo, mockSubmissionRepo, mockStorage, mockProducer)

		expectedSubmission := &models.Submission{
			ID:           "submission-id",
			AssignmentID: "assignment-id",
			StudentID:    "STUDENT-001",
			FilePath:     "submissions/file123.pdf",
			Status:       models.StatusSubmitted,
		}

		mockSubmissionRepo.On("GetByID", ctx, "submission-id").Return(expectedSubmission, nil)

		submission, err := service.GetSubmission(ctx, "submission-id")

		assert.NoError(t, err)
		assert.Equal(t, expectedSubmission, submission)
		mockSubmissionRepo.AssertExpectations(t)
	})

	t.Run("submission not found", func(t *testing.T) {
		mockAssignmentRepo := new(MockAssignmentRepository)
		mockSubmissionRepo := new(MockSubmissionRepository)
		mockStorage := new(MockFileStorage)
		mockProducer := new(MockKafkaProducer)
		service := NewSubmissionService(mockAssignmentRepo, mockSubmissionRepo, mockStorage, mockProducer)

		mockSubmissionRepo.On("GetByID", ctx, "non-existent").Return(nil, errors.New("submission not found"))

		submission, err := service.GetSubmission(ctx, "non-existent")

		assert.Error(t, err)
		assert.Nil(t, submission)
		assert.Contains(t, err.Error(), "failed to get submission")
		mockSubmissionRepo.AssertExpectations(t)
	})
}

func TestSubmissionService_ListSubmissions(t *testing.T) {
	ctx := context.Background()

	t.Run("list submissions successfully", func(t *testing.T) {
		mockAssignmentRepo := new(MockAssignmentRepository)
		mockSubmissionRepo := new(MockSubmissionRepository)
		mockStorage := new(MockFileStorage)
		mockProducer := new(MockKafkaProducer)
		service := NewSubmissionService(mockAssignmentRepo, mockSubmissionRepo, mockStorage, mockProducer)

		expectedSubmissions := []*models.Submission{
			{ID: "sub1", StudentID: "STUDENT-001"},
			{ID: "sub2", StudentID: "STUDENT-002"},
		}

		mockSubmissionRepo.On("ListByAssignment", ctx, "assignment-id", "submitted_at", "DESC").Return(expectedSubmissions, nil)

		submissions, err := service.ListSubmissions(ctx, "assignment-id", "submitted_at", "DESC")

		assert.NoError(t, err)
		assert.Len(t, submissions, 2)
		assert.Equal(t, expectedSubmissions, submissions)
		mockSubmissionRepo.AssertExpectations(t)
	})

	t.Run("repository error", func(t *testing.T) {
		mockAssignmentRepo := new(MockAssignmentRepository)
		mockSubmissionRepo := new(MockSubmissionRepository)
		mockStorage := new(MockFileStorage)
		mockProducer := new(MockKafkaProducer)
		service := NewSubmissionService(mockAssignmentRepo, mockSubmissionRepo, mockStorage, mockProducer)

		mockSubmissionRepo.On("ListByAssignment", ctx, "assignment-id", "submitted_at", "DESC").Return(nil, errors.New("database error"))

		submissions, err := service.ListSubmissions(ctx, "assignment-id", "submitted_at", "DESC")

		assert.Error(t, err)
		assert.Nil(t, submissions)
		assert.Contains(t, err.Error(), "failed to list submissions")
		mockSubmissionRepo.AssertExpectations(t)
	})
}

func TestSubmissionService_ListStudentSubmissions(t *testing.T) {
	ctx := context.Background()

	t.Run("list student submissions successfully", func(t *testing.T) {
		mockAssignmentRepo := new(MockAssignmentRepository)
		mockSubmissionRepo := new(MockSubmissionRepository)
		mockStorage := new(MockFileStorage)
		mockProducer := new(MockKafkaProducer)
		service := NewSubmissionService(mockAssignmentRepo, mockSubmissionRepo, mockStorage, mockProducer)

		expectedSubmissions := []*models.Submission{
			{ID: "sub1", AssignmentID: "assignment1"},
			{ID: "sub2", AssignmentID: "assignment2"},
		}

		mockSubmissionRepo.On("ListByStudent", ctx, "STUDENT-001", "COURSE-001").Return(expectedSubmissions, nil)

		submissions, err := service.ListStudentSubmissions(ctx, "STUDENT-001", "COURSE-001")

		assert.NoError(t, err)
		assert.Len(t, submissions, 2)
		assert.Equal(t, expectedSubmissions, submissions)
		mockSubmissionRepo.AssertExpectations(t)
	})

	t.Run("repository error", func(t *testing.T) {
		mockAssignmentRepo := new(MockAssignmentRepository)
		mockSubmissionRepo := new(MockSubmissionRepository)
		mockStorage := new(MockFileStorage)
		mockProducer := new(MockKafkaProducer)
		service := NewSubmissionService(mockAssignmentRepo, mockSubmissionRepo, mockStorage, mockProducer)

		mockSubmissionRepo.On("ListByStudent", ctx, "STUDENT-001", "COURSE-001").Return(nil, errors.New("database error"))

		submissions, err := service.ListStudentSubmissions(ctx, "STUDENT-001", "COURSE-001")

		assert.Error(t, err)
		assert.Nil(t, submissions)
		assert.Contains(t, err.Error(), "failed to list student submissions")
		mockSubmissionRepo.AssertExpectations(t)
	})
}
