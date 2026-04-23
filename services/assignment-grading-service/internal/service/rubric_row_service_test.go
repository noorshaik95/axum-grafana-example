package service

import (
	"context"
	"testing"

	"slate/services/assignment-grading-service/internal/models"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// mockRubricRowRepo implements repository.RubricRowRepository for unit tests.
type mockRubricRowRepo struct {
	mock.Mock
}

func (m *mockRubricRowRepo) Create(ctx context.Context, row *models.RubricRow) error {
	args := m.Called(ctx, row)
	return args.Error(0)
}
func (m *mockRubricRowRepo) GetByID(ctx context.Context, id string) (*models.RubricRow, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.RubricRow), args.Error(1)
}
func (m *mockRubricRowRepo) Update(ctx context.Context, row *models.RubricRow) error {
	args := m.Called(ctx, row)
	return args.Error(0)
}
func (m *mockRubricRowRepo) Delete(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}
func (m *mockRubricRowRepo) ListByAssignment(ctx context.Context, assignmentID string) ([]*models.RubricRow, error) {
	args := m.Called(ctx, assignmentID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.RubricRow), args.Error(1)
}

func TestRubricRowService_CreateValidates(t *testing.T) {
	ctx := context.Background()
	t.Run("rejects missing title", func(t *testing.T) {
		repo := new(mockRubricRowRepo)
		svc := NewRubricRowService(repo)
		_, err := svc.Create(ctx, &models.RubricRow{AssignmentID: "a1", MaxPoints: 5})
		assert.Error(t, err)
	})

	t.Run("rejects non-positive max_points", func(t *testing.T) {
		repo := new(mockRubricRowRepo)
		svc := NewRubricRowService(repo)
		_, err := svc.Create(ctx, &models.RubricRow{AssignmentID: "a1", Title: "Correctness"})
		assert.Error(t, err)
	})

	t.Run("persists a valid rubric row", func(t *testing.T) {
		repo := new(mockRubricRowRepo)
		svc := NewRubricRowService(repo)
		row := &models.RubricRow{AssignmentID: "a1", Title: "Correctness", MaxPoints: 10, SortOrder: 1}
		repo.On("Create", ctx, row).Return(nil)
		out, err := svc.Create(ctx, row)
		assert.NoError(t, err)
		assert.NotNil(t, out)
		repo.AssertExpectations(t)
	})
}

func TestRubricRowService_UpdateAndList(t *testing.T) {
	ctx := context.Background()

	t.Run("list returns rows from repo", func(t *testing.T) {
		repo := new(mockRubricRowRepo)
		repo.On("ListByAssignment", ctx, "a1").Return([]*models.RubricRow{
			{ID: "r1", AssignmentID: "a1", Title: "Tests pass", MaxPoints: 20, SortOrder: 0},
			{ID: "r2", AssignmentID: "a1", Title: "Style", MaxPoints: 5, SortOrder: 1},
		}, nil)
		rows, err := NewRubricRowService(repo).List(ctx, "a1")
		assert.NoError(t, err)
		assert.Len(t, rows, 2)
	})

	t.Run("update requires id and valid fields", func(t *testing.T) {
		repo := new(mockRubricRowRepo)
		svc := NewRubricRowService(repo)
		_, err := svc.Update(ctx, &models.RubricRow{Title: "T", MaxPoints: 5, AssignmentID: "a1"})
		assert.Error(t, err)
	})

	t.Run("update persists changes", func(t *testing.T) {
		repo := new(mockRubricRowRepo)
		svc := NewRubricRowService(repo)
		row := &models.RubricRow{ID: "r1", AssignmentID: "a1", Title: "Correctness", MaxPoints: 15, SortOrder: 0}
		repo.On("Update", ctx, row).Return(nil)
		_, err := svc.Update(ctx, row)
		assert.NoError(t, err)
		repo.AssertExpectations(t)
	})

	t.Run("delete requires id", func(t *testing.T) {
		repo := new(mockRubricRowRepo)
		err := NewRubricRowService(repo).Delete(ctx, "")
		assert.Error(t, err)
	})
}
