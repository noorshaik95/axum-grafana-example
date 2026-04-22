package grpc

import (
	"context"
	"errors"
	"testing"
	"time"

	pb "slate/services/assignment-grading-service/api/proto"
	"slate/services/assignment-grading-service/internal/grading"
	"slate/services/assignment-grading-service/internal/models"
	"slate/services/assignment-grading-service/internal/repository"
	"slate/services/assignment-grading-service/internal/service"
)

// --- Stubs ---

type stubSubmissionService struct {
	upsertFn  func(ctx context.Context, tenantID, assignmentID, studentID string, fileURLs []string) (*models.Submission, error)
	getDraft  func(ctx context.Context, assignmentID, studentID string) (*models.Submission, error)
}

func (s *stubSubmissionService) SubmitAssignment(ctx context.Context, assignmentID, studentID string, fileContent []byte, fileName, contentType string) (*models.Submission, error) {
	return nil, errors.New("not used")
}
func (s *stubSubmissionService) SubmitAssignmentWithURLs(ctx context.Context, tenantID, assignmentID, studentID string, fileURLs []string) (*models.Submission, error) {
	return nil, errors.New("not used")
}
func (s *stubSubmissionService) GetSubmission(ctx context.Context, id string) (*models.Submission, error) {
	return nil, errors.New("not used")
}
func (s *stubSubmissionService) ListSubmissions(ctx context.Context, assignmentID, sortBy, order string) ([]*models.Submission, error) {
	return nil, errors.New("not used")
}
func (s *stubSubmissionService) ListSubmissionsPaginated(ctx context.Context, assignmentID, tenantID string, page, pageSize int) ([]*models.Submission, int, error) {
	return nil, 0, errors.New("not used")
}
func (s *stubSubmissionService) ListStudentSubmissions(ctx context.Context, studentID, courseID string) ([]*models.Submission, error) {
	return nil, errors.New("not used")
}
func (s *stubSubmissionService) UpsertDraft(ctx context.Context, tenantID, assignmentID, studentID string, fileURLs []string) (*models.Submission, error) {
	return s.upsertFn(ctx, tenantID, assignmentID, studentID, fileURLs)
}
func (s *stubSubmissionService) GetDraft(ctx context.Context, assignmentID, studentID string) (*models.Submission, error) {
	return s.getDraft(ctx, assignmentID, studentID)
}

type stubGradingService struct {
	publishFn func(ctx context.Context, id string) (*models.Grade, error)
}

func (s *stubGradingService) CreateGrade(ctx context.Context, submissionID string, score float64, feedback, gradedBy string) (*models.Grade, error) {
	return nil, errors.New("not used")
}
func (s *stubGradingService) CreateGradeFull(ctx context.Context, submissionID string, score float64, feedback, gradedBy string, rubricScores map[string]interface{}, courseID, tenantID string) (*models.Grade, error) {
	return nil, errors.New("not used")
}
func (s *stubGradingService) UpdateGrade(ctx context.Context, id string, score float64, feedback string) (*models.Grade, error) {
	return nil, errors.New("not used")
}
func (s *stubGradingService) PublishGrade(ctx context.Context, id string) (*models.Grade, error) {
	return s.publishFn(ctx, id)
}
func (s *stubGradingService) GetGrade(ctx context.Context, id string) (*models.Grade, error) {
	return nil, errors.New("not used")
}
func (s *stubGradingService) AutoGrade(ctx context.Context, assignmentID string) error {
	return errors.New("not used")
}

type stubQueueService struct {
	groups []service.PatternGroup
	err    error
}

func (s *stubQueueService) GetGradingQueue(ctx context.Context, assignmentID, instructorID string) ([]service.PatternGroup, error) {
	return s.groups, s.err
}

// compile-time interface checks
var _ service.SubmissionService = (*stubSubmissionService)(nil)
var _ service.GradingService = (*stubGradingService)(nil)
var _ service.GradingQueueService = (*stubQueueService)(nil)

// --- Tests ---

func TestSaveDraft_ok(t *testing.T) {
	draft := &models.Submission{
		ID: "s1", AssignmentID: "a1", StudentID: "u1",
		Status: models.StatusDraft, IsDraft: true,
		FilePath: "url://f1",
	}
	stub := &stubSubmissionService{
		upsertFn: func(ctx context.Context, tenantID, assignmentID, studentID string, fileURLs []string) (*models.Submission, error) {
			if assignmentID != "a1" || studentID != "u1" || len(fileURLs) != 1 {
				t.Fatalf("unexpected args: %s %s %v", assignmentID, studentID, fileURLs)
			}
			return draft, nil
		},
	}
	srv := NewSubmissionServiceServer(stub)
	resp, err := srv.SaveDraft(context.Background(), &pb.SaveDraftRequest{
		TenantId: "t1", AssignmentId: "a1", StudentId: "u1", FileUrls: []string{"url://f1"},
	})
	if err != nil {
		t.Fatalf("SaveDraft error: %v", err)
	}
	if resp.Submission.Id != "s1" {
		t.Fatalf("expected submission id s1, got %q", resp.Submission.Id)
	}
	if resp.Submission.Status != models.StatusDraft {
		t.Fatalf("expected draft status, got %q", resp.Submission.Status)
	}
}

func TestGetDraft_ok(t *testing.T) {
	draft := &models.Submission{
		ID: "s1", AssignmentID: "a1", StudentID: "u1",
		Status: models.StatusDraft, IsDraft: true,
	}
	stub := &stubSubmissionService{
		getDraft: func(ctx context.Context, assignmentID, studentID string) (*models.Submission, error) {
			if assignmentID != "a1" || studentID != "u1" {
				t.Fatalf("unexpected args")
			}
			return draft, nil
		},
	}
	srv := NewSubmissionServiceServer(stub)
	resp, err := srv.GetDraft(context.Background(), &pb.GetDraftRequest{
		AssignmentId: "a1", StudentId: "u1",
	})
	if err != nil {
		t.Fatalf("GetDraft error: %v", err)
	}
	if resp.Submission.Id != "s1" {
		t.Fatalf("expected id s1, got %q", resp.Submission.Id)
	}
}

func TestGetGradingQueue_ok(t *testing.T) {
	groups := []service.PatternGroup{
		{
			PatternID: "p1", Description: "pattern 1", Count: 2, AutoScoreSuggestion: 8.5,
			Submissions: []service.PatternSubmission{
				{SubmissionID: "s1", StudentID: "u1", Status: "submitted"},
				{SubmissionID: "s2", StudentID: "u2", Status: "submitted"},
			},
		},
		{PatternID: "p2", Description: "pattern 2", Count: 1},
	}
	srv := NewGradingServiceServerFull(&stubGradingService{}, &stubQueueService{groups: groups})
	resp, err := srv.GetGradingQueue(context.Background(), &pb.GetGradingQueueRequest{
		AssignmentId: "a1", InstructorId: "i1",
	})
	if err != nil {
		t.Fatalf("GetGradingQueue error: %v", err)
	}
	if len(resp.Groups) != 2 {
		t.Fatalf("expected 2 groups, got %d", len(resp.Groups))
	}
	if resp.Groups[0].PatternId != "p1" || resp.Groups[0].Count != 2 {
		t.Fatalf("group 0 mismatch: %+v", resp.Groups[0])
	}
	if len(resp.Groups[0].Submissions) != 2 {
		t.Fatalf("expected 2 submissions in group 0, got %d", len(resp.Groups[0].Submissions))
	}
}

func TestGetGradingQueueCount_ok(t *testing.T) {
	groups := []service.PatternGroup{
		{PatternID: "p1", Count: 3},
		{PatternID: "p2", Count: 2},
	}
	srv := NewGradingServiceServerFull(&stubGradingService{}, &stubQueueService{groups: groups})
	resp, err := srv.GetGradingQueueCount(context.Background(), &pb.GetGradingQueueCountRequest{
		AssignmentId: "a1",
	})
	if err != nil {
		t.Fatalf("GetGradingQueueCount error: %v", err)
	}
	if resp.Count != 5 {
		t.Fatalf("expected count 5, got %d", resp.Count)
	}
}

func TestBatchPublishGrades_ok(t *testing.T) {
	now := time.Now()
	calls := 0
	stub := &stubGradingService{
		publishFn: func(ctx context.Context, id string) (*models.Grade, error) {
			calls++
			if id == "bad" {
				return nil, errors.New("not found")
			}
			return &models.Grade{ID: id, Status: models.GradeStatusPublished, PublishedAt: &now}, nil
		},
	}
	srv := NewGradingServiceServerFull(stub, &stubQueueService{})
	resp, err := srv.BatchPublishGrades(context.Background(), &pb.BatchPublishGradesRequest{
		GradeIds: []string{"g1", "g2", "bad", "g3"},
	})
	if err != nil {
		t.Fatalf("BatchPublishGrades error: %v", err)
	}
	if calls != 4 {
		t.Fatalf("expected 4 publish calls, got %d", calls)
	}
	if resp.PublishedCount != 3 {
		t.Fatalf("expected 3 published, got %d", resp.PublishedCount)
	}
	if len(resp.FailedIds) != 1 || resp.FailedIds[0] != "bad" {
		t.Fatalf("expected [bad] failed, got %v", resp.FailedIds)
	}
}

// keep imports used (grading / repository packages compile-check the service stub)
var (
	_ grading.GradeEstimate
	_ repository.GradeStatistics
)
