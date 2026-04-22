package service

import (
	"context"
	"errors"
	"testing"

	"slate/services/metrics-service/internal/cache"
)

func TestClassify_Healthy(t *testing.T) {
	e := classify(RosterSignals{
		UserID:          "u1",
		MissedAssignments: 0,
		DaysSinceActive:   1,
		GradeTrend:        []float64{80, 82, 85},
	})
	if e.RiskLevel != RiskHealthy {
		t.Fatalf("expected healthy, got %s", e.RiskLevel)
	}
}

func TestClassify_SlippingOneMissed(t *testing.T) {
	e := classify(RosterSignals{
		MissedAssignments: 1,
		DaysSinceActive:   2,
		GradeTrend:        []float64{85, 88},
	})
	if e.RiskLevel != RiskSlipping {
		t.Fatalf("expected slipping, got %s", e.RiskLevel)
	}
}

func TestClassify_SlippingTwoWeekDown(t *testing.T) {
	e := classify(RosterSignals{
		MissedAssignments: 0,
		DaysSinceActive:   1,
		GradeTrend:        []float64{80, 78, 75}, // 2-step down streak
	})
	if e.RiskLevel != RiskSlipping {
		t.Fatalf("expected slipping, got %s", e.RiskLevel)
	}
}

func TestClassify_AtRiskTwoMissed(t *testing.T) {
	e := classify(RosterSignals{
		MissedAssignments: 2,
		GradeTrend:        []float64{80, 82},
	})
	if e.RiskLevel != RiskAtRisk {
		t.Fatalf("expected at_risk, got %s", e.RiskLevel)
	}
}

func TestClassify_AtRiskThreeWeekDown(t *testing.T) {
	e := classify(RosterSignals{
		GradeTrend: []float64{90, 85, 80, 70}, // 3-step down streak
	})
	if e.RiskLevel != RiskAtRisk {
		t.Fatalf("expected at_risk, got %s", e.RiskLevel)
	}
}

func TestClassify_AtRiskInactive(t *testing.T) {
	e := classify(RosterSignals{
		DaysSinceActive: 7,
		GradeTrend:      []float64{80},
	})
	if e.RiskLevel != RiskAtRisk {
		t.Fatalf("expected at_risk, got %s", e.RiskLevel)
	}
}

func TestTrailingDownStreak(t *testing.T) {
	cases := []struct {
		name string
		in   []float64
		want int
	}{
		{"empty", nil, 0},
		{"single", []float64{80}, 0},
		{"flat", []float64{80, 80, 80}, 0},
		{"up", []float64{80, 82, 85}, 0},
		{"two-down", []float64{80, 78, 75}, 2},
		{"three-down", []float64{90, 85, 80, 70}, 3},
		{"recovers", []float64{90, 85, 88}, 0},
		{"down-then-flat", []float64{90, 85, 85}, 0},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := trailingDownStreak(c.in); got != c.want {
				t.Fatalf("%s: expected %d, got %d", c.name, c.want, got)
			}
		})
	}
}

func TestSortByRisk(t *testing.T) {
	entries := []RosterEntry{
		{UserID: "h", DisplayName: "Healthy H", RiskLevel: RiskHealthy},
		{UserID: "a1", DisplayName: "At Risk A", RiskLevel: RiskAtRisk, MissedAssignments: 3},
		{UserID: "s", DisplayName: "Slipping S", RiskLevel: RiskSlipping},
		{UserID: "a2", DisplayName: "At Risk B", RiskLevel: RiskAtRisk, MissedAssignments: 5},
	}
	sortByRisk(entries)

	if entries[0].UserID != "a2" {
		t.Fatalf("expected a2 first, got %s", entries[0].UserID)
	}
	if entries[1].UserID != "a1" {
		t.Fatalf("expected a1 second, got %s", entries[1].UserID)
	}
	if entries[2].RiskLevel != RiskSlipping {
		t.Fatalf("expected slipping third, got %s", entries[2].RiskLevel)
	}
	if entries[3].RiskLevel != RiskHealthy {
		t.Fatalf("expected healthy last, got %s", entries[3].RiskLevel)
	}
}

func TestGetRosterHealth_ClassifiesAllBandsAndCaches(t *testing.T) {
	c := cache.NewMemory()
	svc := NewRosterService(c)

	signals := []RosterSignals{
		{UserID: "u1", DisplayName: "Alice", GradeTrend: []float64{90, 92}},
		{UserID: "u2", DisplayName: "Bob", MissedAssignments: 1, GradeTrend: []float64{80}},
		{UserID: "u3", DisplayName: "Carol", MissedAssignments: 3, DaysSinceActive: 6},
	}
	calls := 0
	fetch := func(ctx context.Context, tenant, course, instr string) ([]RosterSignals, error) {
		calls++
		return signals, nil
	}

	got, err := svc.GetRosterHealth(context.Background(), "acme", "c1", "i1", fetch)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 3 {
		t.Fatalf("expected 3, got %d", len(got))
	}
	if got[0].RiskLevel != RiskAtRisk {
		t.Fatalf("expected at_risk first, got %s", got[0].RiskLevel)
	}
	if got[1].RiskLevel != RiskSlipping {
		t.Fatalf("expected slipping second, got %s", got[1].RiskLevel)
	}
	if got[2].RiskLevel != RiskHealthy {
		t.Fatalf("expected healthy last, got %s", got[2].RiskLevel)
	}

	// Second call should be cached — fetch not invoked again.
	_, err = svc.GetRosterHealth(context.Background(), "acme", "c1", "i1", fetch)
	if err != nil {
		t.Fatal(err)
	}
	if calls != 1 {
		t.Fatalf("expected fetch called once, got %d", calls)
	}
}

func TestGetRosterHealth_InvalidateClearsCache(t *testing.T) {
	c := cache.NewMemory()
	svc := NewRosterService(c)
	calls := 0
	fetch := func(ctx context.Context, tenant, course, instr string) ([]RosterSignals, error) {
		calls++
		return []RosterSignals{{UserID: "u1", DisplayName: "A"}}, nil
	}
	_, _ = svc.GetRosterHealth(context.Background(), "acme", "c1", "i1", fetch)
	svc.InvalidateRoster("acme", "c1")
	_, _ = svc.GetRosterHealth(context.Background(), "acme", "c1", "i1", fetch)

	if calls != 2 {
		t.Fatalf("expected 2 fetches after invalidation, got %d", calls)
	}
}

func TestGetRosterHealth_InvalidateAllCourses(t *testing.T) {
	c := cache.NewMemory()
	svc := NewRosterService(c)
	fetch := func(ctx context.Context, tenant, course, instr string) ([]RosterSignals, error) {
		return []RosterSignals{{UserID: "u1"}}, nil
	}
	_, _ = svc.GetRosterHealth(context.Background(), "acme", "c1", "i1", fetch)
	_, _ = svc.GetRosterHealth(context.Background(), "acme", "c2", "i1", fetch)

	svc.InvalidateRoster("acme", "")

	if _, ok := c.Get(RosterCacheKey("acme", "c1")); ok {
		t.Fatal("expected c1 evicted")
	}
	if _, ok := c.Get(RosterCacheKey("acme", "c2")); ok {
		t.Fatal("expected c2 evicted")
	}
}

func TestGetRosterHealth_FetcherError(t *testing.T) {
	svc := NewRosterService(cache.NewMemory())
	fetchErr := errors.New("boom")
	_, err := svc.GetRosterHealth(context.Background(), "acme", "c1", "i1",
		func(ctx context.Context, tenant, course, instr string) ([]RosterSignals, error) {
			return nil, fetchErr
		})
	if !errors.Is(err, fetchErr) {
		t.Fatalf("expected fetch error, got %v", err)
	}
}
