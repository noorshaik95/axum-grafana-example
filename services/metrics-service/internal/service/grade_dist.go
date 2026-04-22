package service

import (
	"slate/services/metrics-service/internal/analytics"
)

// LetterBucket is a single letter-grade bucket in an assignment histogram.
type LetterBucket struct {
	Letter string `json:"letter"`
	Count  int    `json:"count"`
}

// LetterHistogram is the W12.2 grade distribution shape: a list of buckets
// in canonical letter order plus headline stats.
type LetterHistogram struct {
	AssignmentID  string         `json:"assignment_id"`
	Buckets       []LetterBucket `json:"buckets"`
	TotalStudents int            `json:"total_students"`
	Mean          float64        `json:"mean"`
	Median        float64        `json:"median"`
}

// letterOrder is the canonical order letter buckets are reported in. The
// order is flattened to coarse bands (A, B, C, D, F) so the histogram is easy
// to chart — the underlying LetterGrade analyser produces +/- variants which
// we collapse here.
var letterOrder = []string{"A", "B", "C", "D", "F"}

// BuildLetterHistogram buckets scores into A/B/C/D/F and returns a histogram
// suitable for the frontend. Scores ≥ 90 are A, ≥ 80 B, ≥ 70 C, ≥ 60 D, else F.
func BuildLetterHistogram(assignmentID string, scores []float64) LetterHistogram {
	counts := map[string]int{}
	for _, s := range scores {
		counts[coarseLetter(s)]++
	}

	buckets := make([]LetterBucket, 0, len(letterOrder))
	for _, l := range letterOrder {
		buckets = append(buckets, LetterBucket{Letter: l, Count: counts[l]})
	}

	h := LetterHistogram{
		AssignmentID:  assignmentID,
		Buckets:       buckets,
		TotalStudents: len(scores),
	}
	if len(scores) > 0 {
		dist := analytics.ComputeDistribution(assignmentID, append([]float64(nil), scores...))
		h.Mean = dist.Mean
		h.Median = dist.Median
	}
	return h
}

func coarseLetter(score float64) string {
	switch {
	case score >= 90:
		return "A"
	case score >= 80:
		return "B"
	case score >= 70:
		return "C"
	case score >= 60:
		return "D"
	default:
		return "F"
	}
}
