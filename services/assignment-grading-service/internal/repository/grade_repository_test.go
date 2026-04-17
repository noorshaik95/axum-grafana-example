package repository

import (
	"testing"
)

func TestGradeStatisticsDefaults(t *testing.T) {
	stats := &GradeStatistics{}

	if stats.TotalSubmissions != 0 {
		t.Error("expected TotalSubmissions to be 0")
	}
	if stats.GradedCount != 0 {
		t.Error("expected GradedCount to be 0")
	}
	if stats.Mean != 0 {
		t.Error("expected Mean to be 0")
	}
}
