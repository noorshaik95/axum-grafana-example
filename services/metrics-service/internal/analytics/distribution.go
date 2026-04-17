package analytics

import (
	"math"
	"sort"

	"slate/services/metrics-service/internal/models"
)

// ComputeDistribution computes a grade distribution from a slice of scores.
func ComputeDistribution(courseID string, scores []float64) models.GradeDistribution {
	if len(scores) == 0 {
		return models.GradeDistribution{
			CourseID:      courseID,
			Buckets:       emptyBuckets(),
			TotalStudents: 0,
		}
	}

	sort.Float64s(scores)

	buckets := make([]models.DistributionBucket, 10)
	for i := range buckets {
		buckets[i] = models.DistributionBucket{
			Min: float64(i * 10),
			Max: float64((i + 1) * 10),
		}
	}

	sum := 0.0
	for _, s := range scores {
		sum += s
		idx := int(s / 10)
		if idx >= 10 {
			idx = 9
		}
		if idx < 0 {
			idx = 0
		}
		buckets[idx].Count++
	}

	n := len(scores)
	return models.GradeDistribution{
		CourseID:      courseID,
		Buckets:       buckets,
		Mean:          sum / float64(n),
		Median:        PercentileValue(scores, 50),
		P25:           PercentileValue(scores, 25),
		P75:           PercentileValue(scores, 75),
		TotalStudents: n,
	}
}

// PercentileValue computes the p-th percentile from a sorted slice of values.
func PercentileValue(sorted []float64, p float64) float64 {
	if len(sorted) == 0 {
		return 0
	}
	if len(sorted) == 1 {
		return sorted[0]
	}

	// Use linear interpolation
	rank := (p / 100.0) * float64(len(sorted)-1)
	lower := int(math.Floor(rank))
	upper := int(math.Ceil(rank))
	if lower == upper {
		return sorted[lower]
	}
	frac := rank - float64(lower)
	return sorted[lower]*(1-frac) + sorted[upper]*frac
}

// ComputeStudentPercentile returns what percentile a student score falls at.
func ComputeStudentPercentile(sorted []float64, studentScore float64) float64 {
	if len(sorted) == 0 {
		return 0
	}
	below := 0
	for _, s := range sorted {
		if s < studentScore {
			below++
		}
	}
	return (float64(below) / float64(len(sorted))) * 100.0
}

// LetterGrade converts a numeric score (0-100) to a letter grade.
func LetterGrade(score float64) string {
	switch {
	case score >= 93:
		return "A"
	case score >= 90:
		return "A-"
	case score >= 87:
		return "B+"
	case score >= 83:
		return "B"
	case score >= 80:
		return "B-"
	case score >= 77:
		return "C+"
	case score >= 73:
		return "C"
	case score >= 70:
		return "C-"
	case score >= 67:
		return "D+"
	case score >= 63:
		return "D"
	case score >= 60:
		return "D-"
	default:
		return "F"
	}
}

// StdDev computes standard deviation from scores.
func StdDev(scores []float64, mean float64) float64 {
	if len(scores) < 2 {
		return 0
	}
	sumSq := 0.0
	for _, s := range scores {
		d := s - mean
		sumSq += d * d
	}
	return math.Sqrt(sumSq / float64(len(scores)))
}

func emptyBuckets() []models.DistributionBucket {
	buckets := make([]models.DistributionBucket, 10)
	for i := range buckets {
		buckets[i] = models.DistributionBucket{
			Min: float64(i * 10),
			Max: float64((i + 1) * 10),
		}
	}
	return buckets
}
