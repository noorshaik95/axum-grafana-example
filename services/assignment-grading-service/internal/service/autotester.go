package service

import (
	"context"
	"fmt"
	"time"

	"slate/services/assignment-grading-service/internal/models"
	"slate/services/assignment-grading-service/internal/repository"
)

// AutoTester runs an assignment's tests.py against a submission in a sandbox
// and records per-test outcomes. W9.3 ships a stub runner: it records a
// single structural pass/fail using the tests_file_path presence as the only
// signal. The contract (inputs, outputs, DB rows) matches what the real
// runner will produce so callers don't change.
type AutoTester struct {
	assignmentRepo repository.AssignmentRepository
	submissionRepo repository.SubmissionRepository
	resultRepo     repository.TestResultRepository
	runner         TestRunner
}

// TestRunner is the sandboxed execution boundary. Real impl shells out to a
// subprocess with CPU/memory/time limits; the default stub returns a fixed
// structural outcome. Swappable so tests stay hermetic.
type TestRunner interface {
	Run(ctx context.Context, assignment *models.Assignment, submission *models.Submission) ([]*models.SubmissionTestResult, error)
}

// NewAutoTester wires the repositories and defaults to StubRunner.
func NewAutoTester(ar repository.AssignmentRepository, sr repository.SubmissionRepository, rr repository.TestResultRepository) *AutoTester {
	return &AutoTester{
		assignmentRepo: ar,
		submissionRepo: sr,
		resultRepo:     rr,
		runner:         StubRunner{},
	}
}

// WithRunner swaps the sandbox implementation — used by tests and later by the
// real subprocess runner.
func (a *AutoTester) WithRunner(r TestRunner) *AutoTester {
	a.runner = r
	return a
}

// Run executes the assignment's auto-tests against a submission and persists
// per-test results. Returns (nil, nil) when the assignment has no tests file.
func (a *AutoTester) Run(ctx context.Context, submissionID string) ([]*models.SubmissionTestResult, error) {
	sub, err := a.submissionRepo.GetByID(ctx, submissionID)
	if err != nil {
		return nil, fmt.Errorf("get submission: %w", err)
	}
	assignment, err := a.assignmentRepo.GetByID(ctx, sub.AssignmentID)
	if err != nil {
		return nil, fmt.Errorf("get assignment: %w", err)
	}
	if assignment.TestsFilePath == "" {
		return nil, nil
	}

	results, err := a.runner.Run(ctx, assignment, sub)
	if err != nil {
		return nil, fmt.Errorf("runner: %w", err)
	}
	for _, r := range results {
		r.SubmissionID = sub.ID
		r.AssignmentID = assignment.ID
	}
	if err := a.resultRepo.DeleteBySubmission(ctx, sub.ID); err != nil {
		return nil, fmt.Errorf("clear prior results: %w", err)
	}
	if err := a.resultRepo.CreateBatch(ctx, results); err != nil {
		return nil, fmt.Errorf("persist results: %w", err)
	}
	return results, nil
}

// StubRunner is the first-pass auto-test runner. It records one structural
// test row ("structural_submission_present") that passes if the submission
// has any file attached. Real subprocess runner replaces this later.
type StubRunner struct{}

// Run is the stub implementation per W9.3 (mock the runner).
func (StubRunner) Run(ctx context.Context, assignment *models.Assignment, submission *models.Submission) ([]*models.SubmissionTestResult, error) {
	passed := submission.FilePath != "" || len(submission.FileURLs) > 0
	output := "stub: no tests executed — placeholder structural check"
	if !passed {
		output = "stub: no files attached to submission"
	}
	return []*models.SubmissionTestResult{{
		TestName:   "structural_submission_present",
		Passed:     passed,
		Output:     output,
		DurationMS: 0,
		CreatedAt:  time.Now(),
	}}, nil
}
