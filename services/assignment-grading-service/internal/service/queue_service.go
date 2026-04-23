package service

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"

	"slate/services/assignment-grading-service/internal/models"
	"slate/services/assignment-grading-service/internal/repository"
)

// PatternSubmission summarises one submission inside a pattern cluster.
type PatternSubmission struct {
	SubmissionID string `json:"submission_id"`
	StudentID    string `json:"student_id"`
	Status       string `json:"status"`
}

// PatternGroup is a cluster of submissions sharing a grading signal —
// either a test-result fingerprint or a rubric-score bucket (W9.4).
type PatternGroup struct {
	PatternID           string              `json:"pattern_id"`
	Description         string              `json:"description"`
	Count               int                 `json:"count"`
	AutoScoreSuggestion float64             `json:"auto_score_suggestion"`
	Submissions         []PatternSubmission `json:"submissions"`
}

// GradingQueueService computes pattern-grouped grading queues.
type GradingQueueService interface {
	GetGradingQueue(ctx context.Context, assignmentID, instructorID string) ([]PatternGroup, error)
}

type gradingQueueService struct {
	assignmentRepo repository.AssignmentRepository
	submissionRepo repository.SubmissionRepository
	gradeRepo      repository.GradeRepository
	testResultRepo repository.TestResultRepository
}

// NewGradingQueueService wires the pattern-clustering queue.
func NewGradingQueueService(
	assignmentRepo repository.AssignmentRepository,
	submissionRepo repository.SubmissionRepository,
	gradeRepo repository.GradeRepository,
	testResultRepo repository.TestResultRepository,
) GradingQueueService {
	return &gradingQueueService{
		assignmentRepo: assignmentRepo,
		submissionRepo: submissionRepo,
		gradeRepo:      gradeRepo,
		testResultRepo: testResultRepo,
	}
}

// GetGradingQueue clusters all non-draft submissions for the assignment.
// If the assignment has tests configured, it groups by test-result fingerprint
// (set of {test_name,passed}). Otherwise it buckets submissions by rubric
// score distribution once grading has started.
func (s *gradingQueueService) GetGradingQueue(ctx context.Context, assignmentID, instructorID string) ([]PatternGroup, error) {
	if assignmentID == "" {
		return nil, fmt.Errorf("assignment_id is required")
	}
	assignment, err := s.assignmentRepo.GetByID(ctx, assignmentID)
	if err != nil {
		return nil, fmt.Errorf("get assignment: %w", err)
	}
	_ = instructorID // reserved for future instructor-scoped filters

	subs, err := s.submissionRepo.ListByAssignment(ctx, assignmentID, "submitted_at", "DESC")
	if err != nil {
		return nil, fmt.Errorf("list submissions: %w", err)
	}
	if len(subs) == 0 {
		return []PatternGroup{}, nil
	}

	if assignment.TestsFilePath != "" {
		return s.clusterByTestFingerprint(ctx, assignment, subs)
	}
	return s.clusterByRubricScore(ctx, assignment, subs)
}

func (s *gradingQueueService) clusterByTestFingerprint(ctx context.Context, assignment *models.Assignment, subs []*models.Submission) ([]PatternGroup, error) {
	results, err := s.testResultRepo.ListByAssignment(ctx, assignment.ID)
	if err != nil {
		return nil, fmt.Errorf("list test results: %w", err)
	}
	bySubmission := map[string][]*models.SubmissionTestResult{}
	for _, r := range results {
		bySubmission[r.SubmissionID] = append(bySubmission[r.SubmissionID], r)
	}

	groups := map[string]*PatternGroup{}
	order := []string{}
	for _, sub := range subs {
		fp, desc, passRate := fingerprintFromResults(bySubmission[sub.ID])
		g, ok := groups[fp]
		if !ok {
			g = &PatternGroup{
				PatternID:           fp,
				Description:         desc,
				AutoScoreSuggestion: round2(passRate * assignment.MaxPoints),
			}
			groups[fp] = g
			order = append(order, fp)
		}
		g.Count++
		g.Submissions = append(g.Submissions, PatternSubmission{
			SubmissionID: sub.ID, StudentID: sub.StudentID, Status: sub.Status,
		})
	}

	out := make([]PatternGroup, 0, len(order))
	for _, key := range order {
		out = append(out, *groups[key])
	}
	return out, nil
}

func (s *gradingQueueService) clusterByRubricScore(ctx context.Context, assignment *models.Assignment, subs []*models.Submission) ([]PatternGroup, error) {
	grades, err := s.gradeRepo.ListByAssignment(ctx, assignment.ID)
	if err != nil {
		return nil, fmt.Errorf("list grades: %w", err)
	}
	bySub := map[string]*models.Grade{}
	for _, g := range grades {
		bySub[g.SubmissionID] = g
	}

	// Bucket boundaries by percentage of max points: 0, 60, 75, 90, 100.
	buckets := []struct {
		id    string
		label string
		low   float64
		high  float64
	}{
		{"bucket-ungraded", "Not yet graded", -1, 0},
		{"bucket-0-60", "Needs rework (0–60%)", 0, 60},
		{"bucket-60-75", "Partial credit (60–75%)", 60, 75},
		{"bucket-75-90", "Solid attempt (75–90%)", 75, 90},
		{"bucket-90-100", "Top tier (90–100%)", 90, 100.0001},
	}

	groups := map[string]*PatternGroup{}
	order := []string{}
	for _, b := range buckets {
		groups[b.id] = &PatternGroup{PatternID: b.id, Description: b.label}
		order = append(order, b.id)
	}

	for _, sub := range subs {
		g, ok := bySub[sub.ID]
		key := "bucket-ungraded"
		var pct float64
		if ok && g.Status == models.GradeStatusPublished {
			pct = g.Percentage
			for _, b := range buckets[1:] {
				if pct >= b.low && pct < b.high {
					key = b.id
					break
				}
			}
		}
		grp := groups[key]
		grp.Count++
		grp.Submissions = append(grp.Submissions, PatternSubmission{
			SubmissionID: sub.ID, StudentID: sub.StudentID, Status: sub.Status,
		})
	}

	out := make([]PatternGroup, 0, len(order))
	for _, key := range order {
		g := groups[key]
		if g.Count == 0 {
			continue
		}
		out = append(out, *g)
	}
	return out, nil
}

// fingerprintFromResults turns a submission's test outcomes into a stable
// pattern id plus a human-readable description and a pass-rate hint.
func fingerprintFromResults(results []*models.SubmissionTestResult) (string, string, float64) {
	if len(results) == 0 {
		return "pattern-no-tests", "No auto-test results yet", 0
	}
	sort.Slice(results, func(i, j int) bool { return results[i].TestName < results[j].TestName })
	parts := make([]string, 0, len(results))
	passed := 0
	for _, r := range results {
		flag := "fail"
		if r.Passed {
			flag = "pass"
			passed++
		}
		parts = append(parts, r.TestName+":"+flag)
	}
	sum := sha1.Sum([]byte(strings.Join(parts, "|"))) // #nosec G401 — fingerprint id, not security-sensitive
	id := "pattern-" + hex.EncodeToString(sum[:8])
	passRate := float64(passed) / float64(len(results))
	desc := fmt.Sprintf("%d/%d tests passing", passed, len(results))
	return id, desc, passRate
}

func round2(f float64) float64 {
	return float64(int(f*100+0.5)) / 100
}
