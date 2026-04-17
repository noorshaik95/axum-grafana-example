package analytics

import (
	"math"
	"testing"
)

func TestComputeDistribution_BasicScores(t *testing.T) {
	scores := []float64{95, 85, 75, 65, 55, 45, 35, 25, 15, 5}
	dist := ComputeDistribution("course-1", scores)

	if dist.TotalStudents != 10 {
		t.Errorf("expected 10 students, got %d", dist.TotalStudents)
	}

	// Each bucket should have exactly 1 student
	for i, b := range dist.Buckets {
		if b.Count != 1 {
			t.Errorf("bucket %d (%.0f-%.0f) expected count 1, got %d", i, b.Min, b.Max, b.Count)
		}
	}

	expectedMean := 50.0
	if math.Abs(dist.Mean-expectedMean) > 0.01 {
		t.Errorf("expected mean %.2f, got %.2f", expectedMean, dist.Mean)
	}
}

func TestComputeDistribution_EmptyScores(t *testing.T) {
	dist := ComputeDistribution("course-1", []float64{})

	if dist.TotalStudents != 0 {
		t.Errorf("expected 0 students, got %d", dist.TotalStudents)
	}
	if len(dist.Buckets) != 10 {
		t.Errorf("expected 10 buckets, got %d", len(dist.Buckets))
	}
}

func TestComputeDistribution_AllSameScore(t *testing.T) {
	scores := []float64{75, 75, 75, 75, 75}
	dist := ComputeDistribution("course-1", scores)

	if dist.Mean != 75.0 {
		t.Errorf("expected mean 75.0, got %.2f", dist.Mean)
	}
	if dist.Median != 75.0 {
		t.Errorf("expected median 75.0, got %.2f", dist.Median)
	}
	if dist.P25 != 75.0 {
		t.Errorf("expected p25 75.0, got %.2f", dist.P25)
	}
	if dist.P75 != 75.0 {
		t.Errorf("expected p75 75.0, got %.2f", dist.P75)
	}
}

func TestComputeDistribution_PerfectScores(t *testing.T) {
	scores := []float64{100, 100, 100}
	dist := ComputeDistribution("course-1", scores)

	// Score of 100 should go into bucket 9 (90-100)
	if dist.Buckets[9].Count != 3 {
		t.Errorf("expected bucket 9 count 3, got %d", dist.Buckets[9].Count)
	}
}

func TestPercentileValue(t *testing.T) {
	sorted := []float64{10, 20, 30, 40, 50, 60, 70, 80, 90, 100}

	tests := []struct {
		percentile float64
		expected   float64
	}{
		{0, 10},
		{50, 55},
		{100, 100},
		{25, 32.5},
		{75, 77.5},
	}

	for _, tt := range tests {
		result := PercentileValue(sorted, tt.percentile)
		if math.Abs(result-tt.expected) > 0.01 {
			t.Errorf("percentile %.0f: expected %.2f, got %.2f", tt.percentile, tt.expected, result)
		}
	}
}

func TestPercentileValue_SingleValue(t *testing.T) {
	result := PercentileValue([]float64{42}, 50)
	if result != 42 {
		t.Errorf("expected 42, got %.2f", result)
	}
}

func TestPercentileValue_Empty(t *testing.T) {
	result := PercentileValue([]float64{}, 50)
	if result != 0 {
		t.Errorf("expected 0, got %.2f", result)
	}
}

func TestComputeStudentPercentile(t *testing.T) {
	sorted := []float64{10, 20, 30, 40, 50, 60, 70, 80, 90, 100}

	pct := ComputeStudentPercentile(sorted, 75)
	if pct != 70.0 {
		t.Errorf("expected 70.0, got %.2f", pct)
	}

	pct = ComputeStudentPercentile(sorted, 100)
	if pct != 90.0 {
		t.Errorf("expected 90.0, got %.2f", pct)
	}

	pct = ComputeStudentPercentile(sorted, 5)
	if pct != 0.0 {
		t.Errorf("expected 0.0, got %.2f", pct)
	}
}

func TestLetterGrade(t *testing.T) {
	tests := []struct {
		score    float64
		expected string
	}{
		{95, "A"},
		{91, "A-"},
		{88, "B+"},
		{85, "B"},
		{80, "B-"},
		{77, "C+"},
		{73, "C"},
		{70, "C-"},
		{67, "D+"},
		{63, "D"},
		{60, "D-"},
		{55, "F"},
		{0, "F"},
		{100, "A"},
	}

	for _, tt := range tests {
		result := LetterGrade(tt.score)
		if result != tt.expected {
			t.Errorf("score %.0f: expected %s, got %s", tt.score, tt.expected, result)
		}
	}
}

func TestStdDev(t *testing.T) {
	scores := []float64{10, 20, 30, 40, 50}
	mean := 30.0
	sd := StdDev(scores, mean)

	// Population std dev of {10,20,30,40,50} with mean 30 = sqrt(200) ≈ 14.14
	expected := math.Sqrt(200)
	if math.Abs(sd-expected) > 0.01 {
		t.Errorf("expected std dev %.2f, got %.2f", expected, sd)
	}
}

func TestStdDev_SingleValue(t *testing.T) {
	sd := StdDev([]float64{50}, 50)
	if sd != 0 {
		t.Errorf("expected 0, got %.2f", sd)
	}
}
