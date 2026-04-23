// Package service implements each W7 capability. W7.2 is pure math and has
// no external dependencies beyond the cache it reads/writes.
package service

import (
	"context"
	"errors"
	"math"
	"time"

	aipb "slate/services/ai-service/api/proto"
	"slate/services/ai-service/internal/cache"
)

const gradeProjectionTTL = 10 * time.Minute

type GradeProjector struct {
	cache *cache.Cache
}

func NewGradeProjector(c *cache.Cache) *GradeProjector {
	return &GradeProjector{cache: c}
}

// Project returns a W7.2 response without calling Claude. Weighting follows
// a standard weighted-average model:
//
//	current = Σ (score/max * weight) / Σ weights_completed
//	projected_max = current_weighted + Σ remaining_weights   (assume 100%)
//	required_avg_remaining = (target - current_weighted) / Σ remaining_weights
//
// The `path` field returns the per-assignment required fraction to hit the
// target, clamped to [0, +∞); feasibility is (required <= 1.0).
func (g *GradeProjector) Project(ctx context.Context, req *aipb.GradeProjectionRequest) (*aipb.GradeProjectionResponse, error) {
	if req == nil {
		return nil, errors.New("grade projection: nil request")
	}

	key := cache.GradeKey(req.TenantSlug, req.UserId, req.CourseId)
	var cached aipb.GradeProjectionResponse
	if hit, err := g.cache.GetJSON(ctx, key, &cached); err == nil && hit {
		return &cached, nil
	}

	resp := computeGradeProjection(req)
	_ = g.cache.SetJSON(ctx, key, resp, gradeProjectionTTL)
	return resp, nil
}

// computeGradeProjection is extracted for ease of testing — it takes only
// the request and has no external state.
func computeGradeProjection(req *aipb.GradeProjectionRequest) *aipb.GradeProjectionResponse {
	var completedWeighted, completedWeight, remainingWeight float64
	for _, e := range req.Completed {
		if e.MaxPoints <= 0 || e.Weight <= 0 {
			continue
		}
		frac := e.Score / e.MaxPoints
		completedWeighted += frac * e.Weight
		completedWeight += e.Weight
	}
	for _, r := range req.Remaining {
		if r.Weight <= 0 {
			continue
		}
		remainingWeight += r.Weight
	}

	// Current grade = weighted-average of completed work only.
	current := 0.0
	if completedWeight > 0 {
		current = completedWeighted / completedWeight
	}

	// Projected grade assumes every remaining item scores 100%.
	projected := completedWeighted + remainingWeight
	totalWeight := completedWeight + remainingWeight
	if totalWeight > 0 {
		projected /= totalWeight
	}

	// Required average on remaining work to hit target (against the full
	// course weight basis, which is target * totalWeight).
	var requiredAvgRemaining float64
	feasible := true
	if remainingWeight > 0 {
		needed := req.TargetGrade*totalWeight - completedWeighted
		requiredAvgRemaining = needed / remainingWeight
	} else {
		requiredAvgRemaining = 0
		feasible = current >= req.TargetGrade
	}
	if requiredAvgRemaining > 1.0 {
		feasible = false
	}

	path := make([]*aipb.PathToTargetEntry, 0, len(req.Remaining))
	for _, r := range req.Remaining {
		clamped := math.Max(0, requiredAvgRemaining)
		path = append(path, &aipb.PathToTargetEntry{
			AssignmentId:     r.AssignmentId,
			RequiredFraction: clamped,
			Feasible:         clamped <= 1.0,
		})
	}

	return &aipb.GradeProjectionResponse{
		CurrentGrade:             roundTo(current, 4),
		ProjectedGrade:           roundTo(projected, 4),
		RequiredAverageRemaining: roundTo(requiredAvgRemaining, 4),
		TargetFeasible:           feasible,
		Path:                     path,
	}
}

func roundTo(v float64, places int) float64 {
	factor := math.Pow(10, float64(places))
	return math.Round(v*factor) / factor
}

// InvalidateGrade is called by the Kafka consumer on `grade.updated`.
func (g *GradeProjector) InvalidateGrade(ctx context.Context, slug, userID, courseID string) error {
	return g.cache.Del(ctx, cache.GradeKey(slug, userID, courseID))
}
