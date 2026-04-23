package service

import (
	"math"
	"testing"

	aipb "slate/services/ai-service/api/proto"
)

const epsilon = 1e-4

func closeEnough(a, b float64) bool { return math.Abs(a-b) < epsilon }

// TestProject_HalfCompleteOnTarget: student has 100% on the first half, is
// targeting 90%. They should need >= 80% on the remaining half (feasible).
func TestProject_HalfCompleteOnTarget(t *testing.T) {
	req := &aipb.GradeProjectionRequest{
		TargetGrade: 0.9,
		Completed: []*aipb.GradeEntry{
			{Score: 100, MaxPoints: 100, Weight: 0.5},
		},
		Remaining: []*aipb.RemainingAssignment{
			{AssignmentId: "a2", MaxPoints: 100, Weight: 0.5},
		},
	}
	resp := computeGradeProjection(req)
	if !closeEnough(resp.CurrentGrade, 1.0) {
		t.Fatalf("current = %v want 1.0", resp.CurrentGrade)
	}
	if !closeEnough(resp.ProjectedGrade, 1.0) {
		t.Fatalf("projected = %v want 1.0 (assuming 100%% remaining)", resp.ProjectedGrade)
	}
	if !closeEnough(resp.RequiredAverageRemaining, 0.8) {
		t.Fatalf("required avg remaining = %v want 0.8", resp.RequiredAverageRemaining)
	}
	if !resp.TargetFeasible {
		t.Fatalf("expected target to be feasible")
	}
}

// TestProject_TargetInfeasible: a student who bombed the first half
// cannot hit an A any more. The math must mark that explicitly.
func TestProject_TargetInfeasible(t *testing.T) {
	req := &aipb.GradeProjectionRequest{
		TargetGrade: 0.9,
		Completed: []*aipb.GradeEntry{
			{Score: 40, MaxPoints: 100, Weight: 0.7},
		},
		Remaining: []*aipb.RemainingAssignment{
			{AssignmentId: "a2", MaxPoints: 100, Weight: 0.3},
		},
	}
	resp := computeGradeProjection(req)
	if resp.TargetFeasible {
		t.Fatalf("0.9 target should be infeasible, got %+v", resp)
	}
	// required = (0.9*1.0 - 0.4*0.7) / 0.3 = (0.9 - 0.28) / 0.3 = 2.0667
	if resp.RequiredAverageRemaining <= 1.0 {
		t.Fatalf("required avg remaining should exceed 1.0, got %v", resp.RequiredAverageRemaining)
	}
}

// TestProject_NoRemainingHitsTarget: no assignments left, current grade
// already meets target → feasible; current == projected.
func TestProject_NoRemainingHitsTarget(t *testing.T) {
	req := &aipb.GradeProjectionRequest{
		TargetGrade: 0.85,
		Completed: []*aipb.GradeEntry{
			{Score: 90, MaxPoints: 100, Weight: 0.5},
			{Score: 85, MaxPoints: 100, Weight: 0.5},
		},
	}
	resp := computeGradeProjection(req)
	if !closeEnough(resp.CurrentGrade, 0.875) {
		t.Fatalf("current = %v want 0.875", resp.CurrentGrade)
	}
	if !closeEnough(resp.ProjectedGrade, 0.875) {
		t.Fatalf("projected should equal current when nothing remains, got %v", resp.ProjectedGrade)
	}
	if !resp.TargetFeasible {
		t.Fatalf("target should be feasible when already exceeded")
	}
}

// TestProject_NoRemainingMissesTarget: if nothing remains and current is
// below target, feasibility must be false.
func TestProject_NoRemainingMissesTarget(t *testing.T) {
	req := &aipb.GradeProjectionRequest{
		TargetGrade: 0.95,
		Completed: []*aipb.GradeEntry{
			{Score: 80, MaxPoints: 100, Weight: 1.0},
		},
	}
	resp := computeGradeProjection(req)
	if resp.TargetFeasible {
		t.Fatalf("expected infeasible (no work left, under target), got %+v", resp)
	}
}

// TestProject_ZeroMaxPointsIgnored defensively skips malformed entries
// instead of dividing by zero.
func TestProject_ZeroMaxPointsIgnored(t *testing.T) {
	req := &aipb.GradeProjectionRequest{
		TargetGrade: 0.8,
		Completed: []*aipb.GradeEntry{
			{Score: 100, MaxPoints: 0, Weight: 0.5},    // skipped
			{Score: 90, MaxPoints: 100, Weight: 0.5},
		},
		Remaining: []*aipb.RemainingAssignment{
			{AssignmentId: "a2", MaxPoints: 100, Weight: 0.5},
		},
	}
	resp := computeGradeProjection(req)
	// Only one valid completed entry: 0.9 * 0.5 = 0.45 weighted, completed weight = 0.5
	if !closeEnough(resp.CurrentGrade, 0.9) {
		t.Fatalf("current = %v want 0.9", resp.CurrentGrade)
	}
}

// TestProject_PathMirrorsRequired asserts the per-assignment path fraction
// is the same required average (the UI uses per-assignment feasibility).
func TestProject_PathMirrorsRequired(t *testing.T) {
	req := &aipb.GradeProjectionRequest{
		TargetGrade: 0.85,
		Completed: []*aipb.GradeEntry{
			{Score: 85, MaxPoints: 100, Weight: 0.5},
		},
		Remaining: []*aipb.RemainingAssignment{
			{AssignmentId: "a", MaxPoints: 100, Weight: 0.25},
			{AssignmentId: "b", MaxPoints: 100, Weight: 0.25},
		},
	}
	resp := computeGradeProjection(req)
	if len(resp.Path) != 2 {
		t.Fatalf("expected 2 path entries, got %d", len(resp.Path))
	}
	for _, p := range resp.Path {
		if !closeEnough(p.RequiredFraction, resp.RequiredAverageRemaining) {
			t.Fatalf("path required=%v resp=%v", p.RequiredFraction, resp.RequiredAverageRemaining)
		}
	}
}
