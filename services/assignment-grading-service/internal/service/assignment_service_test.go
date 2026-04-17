package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"slate/services/assignment-grading-service/internal/models"
	"slate/services/assignment-grading-service/pkg/kafka"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockAssignmentRepository is a mock implementation of AssignmentRepository
type MockAssignmentRepository struct {
	mock.Mock
}

func (m *MockAssignmentRepository) Create(ctx context.Context, assignment *models.Assignment) error {
	args := m.Called(ctx, assignment)
	return args.Error(0)
}

func (m *MockAssignmentRepository) GetByID(ctx context.Context, id string) (*models.Assignment, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Assignment), args.Error(1)
}

func (m *MockAssignmentRepository) Update(ctx context.Context, assignment *models.Assignment) error {
	args := m.Called(ctx, assignment)
	return args.Error(0)
}

func (m *MockAssignmentRepository) Delete(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockAssignmentRepository) SoftDelete(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockAssignmentRepository) ListByCourse(ctx context.Context, courseID string, page, pageSize int) ([]*models.Assignment, int, error) {
	args := m.Called(ctx, courseID, page, pageSize)
	if args.Get(0) == nil {
		return nil, args.Int(1), args.Error(2)
	}
	return args.Get(0).([]*models.Assignment), args.Int(1), args.Error(2)
}

func (m *MockAssignmentRepository) ListFiltered(ctx context.Context, tenantID, courseID, instructorID string, page, pageSize int) ([]*models.Assignment, int, error) {
	args := m.Called(ctx, tenantID, courseID, instructorID, page, pageSize)
	if args.Get(0) == nil {
		return nil, args.Int(1), args.Error(2)
	}
	return args.Get(0).([]*models.Assignment), args.Int(1), args.Error(2)
}

func (m *MockAssignmentRepository) HasSubmissions(ctx context.Context, id string) (bool, error) {
	args := m.Called(ctx, id)
	return args.Bool(0), args.Error(1)
}

func (m *MockAssignmentRepository) SoftDeleteByCourse(ctx context.Context, courseID string) error {
	args := m.Called(ctx, courseID)
	return args.Error(0)
}

// MockKafkaProducer is a mock implementation of kafka.Producer
type MockKafkaProducer struct {
	mock.Mock
}

func (m *MockKafkaProducer) PublishEvent(ctx context.Context, event kafka.Event) error {
	args := m.Called(ctx, event)
	return args.Error(0)
}

func (m *MockKafkaProducer) Close() error {
	args := m.Called()
	return args.Error(0)
}

func TestAssignmentService_CreateAssignment(t *testing.T) {
	ctx := context.Background()

	t.Run("create valid assignment", func(t *testing.T) {
		mockRepo := new(MockAssignmentRepository)
		mockProducer := new(MockKafkaProducer)
		service := NewAssignmentService(mockRepo, mockProducer)

		dueDate := time.Now().Add(24 * time.Hour)
		latePolicy := models.LatePolicy{
			PenaltyPercentPerDay: 10,
			MaxLateDays:          3,
		}

		mockRepo.On("Create", ctx, mock.AnythingOfType("*models.Assignment")).Return(nil)
		mockProducer.On("PublishEvent", ctx, mock.AnythingOfType("kafka.Event")).Return(nil)

		assignment, err := service.CreateAssignment(ctx, "COURSE-001", "Assignment 1", "Test", 100.0, dueDate, latePolicy)

		assert.NoError(t, err)
		assert.NotNil(t, assignment)
		assert.Equal(t, "COURSE-001", assignment.CourseID)
		assert.Equal(t, "Assignment 1", assignment.Title)
		assert.Equal(t, 100.0, assignment.MaxPoints)
		mockRepo.AssertExpectations(t)
		mockProducer.AssertExpectations(t)
	})

	t.Run("validation fails - missing course_id", func(t *testing.T) {
		mockRepo := new(MockAssignmentRepository)
		mockProducer := new(MockKafkaProducer)
		service := NewAssignmentService(mockRepo, mockProducer)

		dueDate := time.Now().Add(24 * time.Hour)
		latePolicy := models.LatePolicy{
			PenaltyPercentPerDay: 10,
			MaxLateDays:          3,
		}

		assignment, err := service.CreateAssignment(ctx, "", "Assignment 1", "Test", 100.0, dueDate, latePolicy)

		assert.Error(t, err)
		assert.Nil(t, assignment)
		assert.Contains(t, err.Error(), "validation failed")
		mockRepo.AssertNotCalled(t, "Create")
		mockProducer.AssertNotCalled(t, "PublishEvent")
	})

	t.Run("repository error", func(t *testing.T) {
		mockRepo := new(MockAssignmentRepository)
		mockProducer := new(MockKafkaProducer)
		service := NewAssignmentService(mockRepo, mockProducer)

		dueDate := time.Now().Add(24 * time.Hour)
		latePolicy := models.LatePolicy{
			PenaltyPercentPerDay: 10,
			MaxLateDays:          3,
		}

		mockRepo.On("Create", ctx, mock.AnythingOfType("*models.Assignment")).Return(errors.New("database error"))

		assignment, err := service.CreateAssignment(ctx, "COURSE-001", "Assignment 1", "Test", 100.0, dueDate, latePolicy)

		assert.Error(t, err)
		assert.Nil(t, assignment)
		assert.Contains(t, err.Error(), "failed to create assignment")
		mockRepo.AssertExpectations(t)
		mockProducer.AssertNotCalled(t, "PublishEvent")
	})
}

func TestAssignmentService_GetAssignment(t *testing.T) {
	ctx := context.Background()

	t.Run("get existing assignment", func(t *testing.T) {
		mockRepo := new(MockAssignmentRepository)
		mockProducer := new(MockKafkaProducer)
		service := NewAssignmentService(mockRepo, mockProducer)

		expectedAssignment := &models.Assignment{
			ID:        "test-id",
			CourseID:  "COURSE-001",
			Title:     "Assignment 1",
			MaxPoints: 100.0,
		}

		mockRepo.On("GetByID", ctx, "test-id").Return(expectedAssignment, nil)

		assignment, err := service.GetAssignment(ctx, "test-id")

		assert.NoError(t, err)
		assert.Equal(t, expectedAssignment, assignment)
		mockRepo.AssertExpectations(t)
	})

	t.Run("assignment not found", func(t *testing.T) {
		mockRepo := new(MockAssignmentRepository)
		mockProducer := new(MockKafkaProducer)
		service := NewAssignmentService(mockRepo, mockProducer)

		mockRepo.On("GetByID", ctx, "non-existent").Return(nil, errors.New("assignment not found"))

		assignment, err := service.GetAssignment(ctx, "non-existent")

		assert.Error(t, err)
		assert.Nil(t, assignment)
		mockRepo.AssertExpectations(t)
	})
}

func TestAssignmentService_SoftDeleteAssignment(t *testing.T) {
	ctx := context.Background()

	t.Run("soft delete existing assignment", func(t *testing.T) {
		mockRepo := new(MockAssignmentRepository)
		mockProducer := new(MockKafkaProducer)
		service := NewAssignmentService(mockRepo, mockProducer)

		existingAssignment := &models.Assignment{
			ID:       "test-id",
			CourseID: "COURSE-001",
		}

		mockRepo.On("GetByID", ctx, "test-id").Return(existingAssignment, nil)
		mockRepo.On("SoftDelete", ctx, "test-id").Return(nil)
		mockProducer.On("PublishEvent", ctx, mock.AnythingOfType("kafka.Event")).Return(nil)

		err := service.SoftDeleteAssignment(ctx, "test-id")

		assert.NoError(t, err)
		mockRepo.AssertExpectations(t)
		mockProducer.AssertExpectations(t)
	})
}

func TestAssignmentService_ListAssignments(t *testing.T) {
	ctx := context.Background()

	t.Run("list assignments with pagination", func(t *testing.T) {
		mockRepo := new(MockAssignmentRepository)
		mockProducer := new(MockKafkaProducer)
		service := NewAssignmentService(mockRepo, mockProducer)

		expectedAssignments := []*models.Assignment{
			{ID: "id1", Title: "Assignment 1"},
			{ID: "id2", Title: "Assignment 2"},
		}

		mockRepo.On("ListByCourse", ctx, "COURSE-001", 1, 20).Return(expectedAssignments, 2, nil)

		assignments, total, err := service.ListAssignments(ctx, "COURSE-001", 1, 20)

		assert.NoError(t, err)
		assert.Equal(t, 2, total)
		assert.Len(t, assignments, 2)
		mockRepo.AssertExpectations(t)
	})

	t.Run("normalize invalid page and page_size", func(t *testing.T) {
		mockRepo := new(MockAssignmentRepository)
		mockProducer := new(MockKafkaProducer)
		service := NewAssignmentService(mockRepo, mockProducer)

		expectedAssignments := []*models.Assignment{}

		mockRepo.On("ListByCourse", ctx, "COURSE-001", 1, 20).Return(expectedAssignments, 0, nil)

		assignments, total, err := service.ListAssignments(ctx, "COURSE-001", 0, 200)

		assert.NoError(t, err)
		assert.Equal(t, 0, total)
		assert.Len(t, assignments, 0)
		mockRepo.AssertExpectations(t)
	})
}
