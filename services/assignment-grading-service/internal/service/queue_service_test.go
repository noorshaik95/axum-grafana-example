package service

import (
	"context"
	"testing"

	"slate/services/assignment-grading-service/internal/models"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// mockTestResultRepo implements repository.TestResultRepository.
type mockTestResultRepo struct {
	mock.Mock
}

func (m *mockTestResultRepo) Create(ctx context.Context, r *models.SubmissionTestResult) error {
	args := m.Called(ctx, r)
	return args.Error(0)
}
func (m *mockTestResultRepo) CreateBatch(ctx context.Context, results []*models.SubmissionTestResult) error {
	args := m.Called(ctx, results)
	return args.Error(0)
}
func (m *mockTestResultRepo) ListBySubmission(ctx context.Context, submissionID string) ([]*models.SubmissionTestResult, error) {
	args := m.Called(ctx, submissionID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.SubmissionTestResult), args.Error(1)
}
func (m *mockTestResultRepo) ListByAssignment(ctx context.Context, assignmentID string) ([]*models.SubmissionTestResult, error) {
	args := m.Called(ctx, assignmentID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.SubmissionTestResult), args.Error(1)
}
func (m *mockTestResultRepo) DeleteBySubmission(ctx context.Context, submissionID string) error {
	args := m.Called(ctx, submissionID)
	return args.Error(0)
}

func TestGradingQueue_ClustersByTestFingerprint(t *testing.T) {
	ctx := context.Background()
	assignmentRepo := new(MockAssignmentRepository)
	submissionRepo := new(MockSubmissionRepository)
	gradeRepo := new(MockGradeRepository)
	testRepo := new(mockTestResultRepo)

	assignment := &models.Assignment{
		ID:            "a1",
		TenantID:      "t1",
		CourseID:      "c1",
		MaxPoints:     100,
		TestsFilePath: "tests/a1/tests.py",
	}
	assignmentRepo.On("GetByID", ctx, "a1").Return(assignment, nil)

	// 4 submissions across 2 patterns:
	//   pattern A: [s1, s2] — test1 pass, test2 fail  → 50%
	//   pattern B: [s3, s4] — both pass              → 100%
	subs := []*models.Submission{
		{ID: "s1", AssignmentID: "a1", StudentID: "u1", Status: models.StatusSubmitted},
		{ID: "s2", AssignmentID: "a1", StudentID: "u2", Status: models.StatusSubmitted},
		{ID: "s3", AssignmentID: "a1", StudentID: "u3", Status: models.StatusSubmitted},
		{ID: "s4", AssignmentID: "a1", StudentID: "u4", Status: models.StatusSubmitted},
	}
	submissionRepo.On("ListByAssignment", ctx, "a1", "submitted_at", "DESC").Return(subs, nil)

	results := []*models.SubmissionTestResult{
		{SubmissionID: "s1", AssignmentID: "a1", TestName: "test1", Passed: true},
		{SubmissionID: "s1", AssignmentID: "a1", TestName: "test2", Passed: false},
		{SubmissionID: "s2", AssignmentID: "a1", TestName: "test1", Passed: true},
		{SubmissionID: "s2", AssignmentID: "a1", TestName: "test2", Passed: false},
		{SubmissionID: "s3", AssignmentID: "a1", TestName: "test1", Passed: true},
		{SubmissionID: "s3", AssignmentID: "a1", TestName: "test2", Passed: true},
		{SubmissionID: "s4", AssignmentID: "a1", TestName: "test1", Passed: true},
		{SubmissionID: "s4", AssignmentID: "a1", TestName: "test2", Passed: true},
	}
	testRepo.On("ListByAssignment", ctx, "a1").Return(results, nil)

	svc := NewGradingQueueService(assignmentRepo, submissionRepo, gradeRepo, testRepo)
	groups, err := svc.GetGradingQueue(ctx, "a1", "instr1")
	assert.NoError(t, err)
	assert.Len(t, groups, 2, "expected two patterns")

	for _, g := range groups {
		assert.Equal(t, 2, g.Count)
		assert.Len(t, g.Submissions, 2)
	}

	// Verify suggestions — pattern B should suggest 100, pattern A should suggest 50.
	var aSuggest, bSuggest float64
	for _, g := range groups {
		if g.Description == "1/2 tests passing" {
			aSuggest = g.AutoScoreSuggestion
		}
		if g.Description == "2/2 tests passing" {
			bSuggest = g.AutoScoreSuggestion
		}
	}
	assert.InDelta(t, 50.0, aSuggest, 0.1)
	assert.InDelta(t, 100.0, bSuggest, 0.1)
}

func TestGradingQueue_ClustersByRubricScoreWhenNoTests(t *testing.T) {
	ctx := context.Background()
	assignmentRepo := new(MockAssignmentRepository)
	submissionRepo := new(MockSubmissionRepository)
	gradeRepo := new(MockGradeRepository)
	testRepo := new(mockTestResultRepo)

	assignment := &models.Assignment{
		ID:        "a2",
		MaxPoints: 100,
		// no TestsFilePath — forces rubric clustering
	}
	assignmentRepo.On("GetByID", ctx, "a2").Return(assignment, nil)

	subs := []*models.Submission{
		{ID: "s1", AssignmentID: "a2", StudentID: "u1", Status: models.StatusGraded},
		{ID: "s2", AssignmentID: "a2", StudentID: "u2", Status: models.StatusGraded},
		{ID: "s3", AssignmentID: "a2", StudentID: "u3", Status: models.StatusSubmitted},
		{ID: "s4", AssignmentID: "a2", StudentID: "u4", Status: models.StatusGraded},
	}
	submissionRepo.On("ListByAssignment", ctx, "a2", "submitted_at", "DESC").Return(subs, nil)

	grades := []*models.Grade{
		{ID: "g1", SubmissionID: "s1", Percentage: 95, Status: models.GradeStatusPublished},
		{ID: "g2", SubmissionID: "s2", Percentage: 92, Status: models.GradeStatusPublished},
		{ID: "g4", SubmissionID: "s4", Percentage: 40, Status: models.GradeStatusPublished},
	}
	gradeRepo.On("ListByAssignment", ctx, "a2").Return(grades, nil)

	svc := NewGradingQueueService(assignmentRepo, submissionRepo, gradeRepo, testRepo)
	groups, err := svc.GetGradingQueue(ctx, "a2", "")
	assert.NoError(t, err)

	// Expect three buckets: ungraded (s3), 0-60 (s4), 90-100 (s1, s2).
	byID := map[string]PatternGroup{}
	for _, g := range groups {
		byID[g.PatternID] = g
	}
	assert.Contains(t, byID, "bucket-ungraded")
	assert.Equal(t, 1, byID["bucket-ungraded"].Count)
	assert.Contains(t, byID, "bucket-0-60")
	assert.Equal(t, 1, byID["bucket-0-60"].Count)
	assert.Contains(t, byID, "bucket-90-100")
	assert.Equal(t, 2, byID["bucket-90-100"].Count)
}
