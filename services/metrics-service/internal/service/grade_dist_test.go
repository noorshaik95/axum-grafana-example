package service

import (
	"math"
	"testing"
)

func TestBuildLetterHistogram_BucketsAndStats(t *testing.T) {
	// 2 As, 3 Bs, 1 C, 1 D, 1 F
	scores := []float64{95, 92, 88, 85, 80, 72, 65, 50}
	h := BuildLetterHistogram("assn-1", scores)

	if h.TotalStudents != 8 {
		t.Fatalf("expected 8 students, got %d", h.TotalStudents)
	}

	want := map[string]int{"A": 2, "B": 3, "C": 1, "D": 1, "F": 1}
	for _, b := range h.Buckets {
		if b.Count != want[b.Letter] {
			t.Errorf("letter %s: expected %d, got %d", b.Letter, want[b.Letter], b.Count)
		}
	}

	// Canonical order A,B,C,D,F
	expectedOrder := []string{"A", "B", "C", "D", "F"}
	for i, b := range h.Buckets {
		if b.Letter != expectedOrder[i] {
			t.Fatalf("position %d: expected %s, got %s", i, expectedOrder[i], b.Letter)
		}
	}

	if math.Abs(h.Mean-78.375) > 0.01 {
		t.Fatalf("expected mean 78.375, got %.3f", h.Mean)
	}
}

func TestBuildLetterHistogram_Empty(t *testing.T) {
	h := BuildLetterHistogram("assn-1", nil)
	if h.TotalStudents != 0 {
		t.Fatalf("expected 0, got %d", h.TotalStudents)
	}
	if len(h.Buckets) != 5 {
		t.Fatalf("expected 5 buckets, got %d", len(h.Buckets))
	}
	for _, b := range h.Buckets {
		if b.Count != 0 {
			t.Errorf("expected 0 count, got %d", b.Count)
		}
	}
}

func TestBuildLetterHistogram_Boundaries(t *testing.T) {
	// 90 -> A, 89 -> B, 80 -> B, 79 -> C, 70 -> C, 69 -> D, 60 -> D, 59 -> F
	scores := []float64{90, 89, 80, 79, 70, 69, 60, 59}
	h := BuildLetterHistogram("b", scores)
	want := map[string]int{"A": 1, "B": 2, "C": 2, "D": 2, "F": 1}
	for _, b := range h.Buckets {
		if b.Count != want[b.Letter] {
			t.Errorf("letter %s: expected %d, got %d", b.Letter, want[b.Letter], b.Count)
		}
	}
}

func TestCoarseLetter(t *testing.T) {
	cases := map[float64]string{
		100: "A", 90: "A",
		89: "B", 80: "B",
		79: "C", 70: "C",
		69: "D", 60: "D",
		59: "F", 0: "F",
	}
	for score, want := range cases {
		if got := coarseLetter(score); got != want {
			t.Errorf("%.0f: want %s, got %s", score, want, got)
		}
	}
}
