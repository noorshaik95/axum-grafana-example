package service

import (
	"context"
	"testing"

	"slate/services/assignment-grading-service/internal/models"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestAutoTester_SkipsWhenNoTestsFile(t *testing.T) {
	ctx := context.Background()
	assignmentRepo := new(MockAssignmentRepository)
	submissionRepo := new(MockSubmissionRepository)
	results := new(mockTestResultRepo)

	assignmentRepo.On("GetByID", ctx, "a1").Return(&models.Assignment{ID: "a1"}, nil)
	submissionRepo.On("GetByID", ctx, "s1").Return(&models.Submission{ID: "s1", AssignmentID: "a1"}, nil)

	at := NewAutoTester(assignmentRepo, submissionRepo, results)
	out, err := at.Run(ctx, "s1")
	assert.NoError(t, err)
	assert.Nil(t, out)
}

func TestAutoTester_StubPersistsStructuralPass(t *testing.T) {
	ctx := context.Background()
	assignmentRepo := new(MockAssignmentRepository)
	submissionRepo := new(MockSubmissionRepository)
	resultRepo := new(mockTestResultRepo)

	assignmentRepo.On("GetByID", ctx, "a1").Return(&models.Assignment{ID: "a1", TestsFilePath: "tests/a1/tests.py"}, nil)
	submissionRepo.On("GetByID", ctx, "s1").Return(&models.Submission{ID: "s1", AssignmentID: "a1", FileURLs: []string{"minio://s1.zip"}}, nil)
	resultRepo.On("DeleteBySubmission", ctx, "s1").Return(nil)
	resultRepo.On("CreateBatch", ctx, mock.AnythingOfType("[]*models.SubmissionTestResult")).Return(nil)

	at := NewAutoTester(assignmentRepo, submissionRepo, resultRepo)
	out, err := at.Run(ctx, "s1")
	assert.NoError(t, err)
	assert.Len(t, out, 1)
	assert.True(t, out[0].Passed, "submission has files → structural check passes")
	assert.Equal(t, "structural_submission_present", out[0].TestName)
}
