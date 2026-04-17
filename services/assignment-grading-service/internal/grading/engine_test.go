package grading

import (
	"math"
	"testing"
	"time"

	"slate/services/assignment-grading-service/internal/models"
)

func TestComputePercentile(t *testing.T) {
	tests := []struct {
		name      string
		score     float64
		allScores []float64
		expected  float64
	}{
		{
			name:      "empty scores",
			score:     85,
			allScores: []float64{},
			expected:  0,
		},
		{
			name:      "single score",
			score:     85,
			allScores: []float64{85},
			expected:  100,
		},
		{
			name:      "highest score",
			score:     100,
			allScores: []float64{60, 70, 80, 90, 100},
			expected:  80, // 4 out of 5 below
		},
		{
			name:      "lowest score",
			score:     60,
			allScores: []float64{60, 70, 80, 90, 100},
			expected:  0, // 0 out of 5 below
		},
		{
			name:      "middle score",
			score:     80,
			allScores: []float64{60, 70, 80, 90, 100},
			expected:  40, // 2 out of 5 below
		},
		{
			name:      "ties - same score",
			score:     80,
			allScores: []float64{80, 80, 80},
			expected:  0, // no scores strictly below
		},
		{
			name:      "two scores",
			score:     90,
			allScores: []float64{70, 90},
			expected:  50, // 1 out of 2 below
		},
		{
			name:      "all same scores",
			score:     75,
			allScores: []float64{75, 75, 75, 75},
			expected:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ComputePercentile(tt.score, tt.allScores)
			if math.Abs(result-tt.expected) > 0.01 {
				t.Errorf("expected %.2f but got %.2f", tt.expected, result)
			}
		})
	}
}

func TestLetterGrade(t *testing.T) {
	tests := []struct {
		name     string
		pct      float64
		scale    []models.GradeScale
		expected string
	}{
		{name: "A grade default", pct: 95, scale: nil, expected: "A"},
		{name: "A boundary", pct: 90, scale: nil, expected: "A"},
		{name: "B grade default", pct: 85, scale: nil, expected: "B"},
		{name: "B boundary", pct: 80, scale: nil, expected: "B"},
		{name: "C grade default", pct: 75, scale: nil, expected: "C"},
		{name: "D grade default", pct: 65, scale: nil, expected: "D"},
		{name: "F grade default", pct: 50, scale: nil, expected: "F"},
		{name: "zero percent", pct: 0, scale: nil, expected: "F"},
		{name: "100 percent", pct: 100, scale: nil, expected: "A"},
		{
			name: "custom scale",
			pct:  85,
			scale: []models.GradeScale{
				{Grade: "S", Min: 70},
				{Grade: "U", Min: 0},
			},
			expected: "S",
		},
		{
			name: "custom scale below threshold",
			pct:  50,
			scale: []models.GradeScale{
				{Grade: "S", Min: 70},
				{Grade: "U", Min: 0},
			},
			expected: "U",
		},
		{name: "empty scale falls back to default", pct: 92, scale: []models.GradeScale{}, expected: "A"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := LetterGrade(tt.pct, tt.scale)
			if result != tt.expected {
				t.Errorf("expected %s but got %s", tt.expected, result)
			}
		})
	}
}

func TestApplyLatePenalty(t *testing.T) {
	dueDate := time.Date(2024, 6, 15, 23, 59, 59, 0, time.UTC)

	tests := []struct {
		name        string
		score       float64
		submittedAt time.Time
		dueDate     time.Time
		rule        models.GradingRule
		expected    float64
	}{
		{
			name:        "on time - no penalty",
			score:       100,
			submittedAt: dueDate.Add(-1 * time.Hour),
			dueDate:     dueDate,
			rule:        models.GradingRule{LatePenaltyPerDay: 10, MaxLatePenalty: 50},
			expected:    100,
		},
		{
			name:        "1 day late",
			score:       100,
			submittedAt: dueDate.Add(24 * time.Hour),
			dueDate:     dueDate,
			rule:        models.GradingRule{LatePenaltyPerDay: 10, MaxLatePenalty: 50},
			expected:    90,
		},
		{
			name:        "3 days late",
			score:       100,
			submittedAt: dueDate.Add(72 * time.Hour),
			dueDate:     dueDate,
			rule:        models.GradingRule{LatePenaltyPerDay: 10, MaxLatePenalty: 50},
			expected:    70,
		},
		{
			name:        "penalty capped at max",
			score:       100,
			submittedAt: dueDate.Add(10 * 24 * time.Hour),
			dueDate:     dueDate,
			rule:        models.GradingRule{LatePenaltyPerDay: 10, MaxLatePenalty: 50},
			expected:    50,
		},
		{
			name:        "zero due date - no penalty",
			score:       85,
			submittedAt: time.Now(),
			dueDate:     time.Time{},
			rule:        models.GradingRule{LatePenaltyPerDay: 10, MaxLatePenalty: 50},
			expected:    85,
		},
		{
			name:        "penalty cannot go below zero",
			score:       20,
			submittedAt: dueDate.Add(5 * 24 * time.Hour),
			dueDate:     dueDate,
			rule:        models.GradingRule{LatePenaltyPerDay: 10, MaxLatePenalty: 100},
			expected:    0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ApplyLatePenalty(tt.score, tt.submittedAt, tt.dueDate, tt.rule)
			if math.Abs(result-tt.expected) > 0.01 {
				t.Errorf("expected %.2f but got %.2f", tt.expected, result)
			}
		})
	}
}

func TestComputeDistribution(t *testing.T) {
	grades := []*models.Grade{
		{Percentage: 95},
		{Percentage: 92},
		{Percentage: 85},
		{Percentage: 82},
		{Percentage: 75},
		{Percentage: 65},
		{Percentage: 55},
		{Percentage: 45},
	}

	dist := ComputeDistribution(grades, nil)

	if len(dist) != 5 {
		t.Fatalf("expected 5 buckets, got %d", len(dist))
	}

	// A: 95, 92 = 2
	if dist[0].LetterGrade != "A" || dist[0].Count != 2 {
		t.Errorf("expected A=2, got %s=%d", dist[0].LetterGrade, dist[0].Count)
	}

	// B: 85, 82 = 2
	if dist[1].LetterGrade != "B" || dist[1].Count != 2 {
		t.Errorf("expected B=2, got %s=%d", dist[1].LetterGrade, dist[1].Count)
	}

	// C: 75 = 1
	if dist[2].LetterGrade != "C" || dist[2].Count != 1 {
		t.Errorf("expected C=1, got %s=%d", dist[2].LetterGrade, dist[2].Count)
	}

	// D: 65 = 1
	if dist[3].LetterGrade != "D" || dist[3].Count != 1 {
		t.Errorf("expected D=1, got %s=%d", dist[3].LetterGrade, dist[3].Count)
	}

	// F: 55, 45 = 2
	if dist[4].LetterGrade != "F" || dist[4].Count != 2 {
		t.Errorf("expected F=2, got %s=%d", dist[4].LetterGrade, dist[4].Count)
	}
}
