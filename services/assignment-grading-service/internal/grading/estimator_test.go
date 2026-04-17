package grading

import (
	"math"
	"testing"
)

func TestAverage(t *testing.T) {
	tests := []struct {
		name     string
		values   []float64
		expected float64
	}{
		{name: "empty", values: []float64{}, expected: 0},
		{name: "single", values: []float64{85}, expected: 85},
		{name: "multiple", values: []float64{80, 90, 100}, expected: 90},
		{name: "decimals", values: []float64{75.5, 82.5}, expected: 79},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := average(tt.values)
			if math.Abs(result-tt.expected) > 0.01 {
				t.Errorf("expected %.2f but got %.2f", tt.expected, result)
			}
		})
	}
}
