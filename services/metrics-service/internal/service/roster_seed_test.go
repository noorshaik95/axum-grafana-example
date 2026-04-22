package service

import (
	"context"
	"testing"

	"slate/services/metrics-service/internal/cache"
)

// TestRoster_EastfieldSeedAssertion pins the classifier against the Eastfield
// seed shape documented in plan/EXECUTION.md:168 — "seed to produce 5 at_risk,
// 5 slipping, 10 healthy in roster health (W12.1)". The fixture below is
// deterministic: given these signals the classifier must always produce the
// 5/5/10 split. If anyone tweaks the risk algorithm this test will flag the
// divergence before the seed step does.
func TestRoster_EastfieldSeedAssertion(t *testing.T) {
	// 5 at_risk: each satisfies one of the three at_risk conditions.
	atRiskSignals := []RosterSignals{
		{UserID: "student01", MissedAssignments: 2},                                   // 2+ missed
		{UserID: "student02", MissedAssignments: 3},                                   // 2+ missed
		{UserID: "student03", GradeTrend: []float64{90, 85, 80, 70}},                  // 3-week decline
		{UserID: "student04", DaysSinceActive: 6},                                     // 5+ inactive
		{UserID: "student05", MissedAssignments: 2, GradeTrend: []float64{80, 75, 60}}, // 2+ missed
	}

	// 5 slipping: each satisfies one of the two slipping conditions and none of
	// the at_risk conditions.
	slippingSignals := []RosterSignals{
		{UserID: "student06", MissedAssignments: 1},                              // 1 missed
		{UserID: "student07", MissedAssignments: 1, DaysSinceActive: 2},          // 1 missed
		{UserID: "student08", GradeTrend: []float64{85, 80, 75}},                 // 2-week decline
		{UserID: "student09", GradeTrend: []float64{90, 88, 84}, DaysSinceActive: 3}, // 2-week decline
		{UserID: "student10", MissedAssignments: 1, GradeTrend: []float64{80}},   // 1 missed
	}

	// 10 healthy: no missed assignments, no decline streak ≥2, <5 days inactive.
	healthySignals := []RosterSignals{
		{UserID: "student11", GradeTrend: []float64{85, 88, 90}},
		{UserID: "student12", GradeTrend: []float64{90, 92}},
		{UserID: "student13", DaysSinceActive: 1},
		{UserID: "student14", GradeTrend: []float64{88, 88, 88}},
		{UserID: "student15", GradeTrend: []float64{80, 78, 82}}, // blip down then up
		{UserID: "student16"},
		{UserID: "student17", GradeTrend: []float64{95}},
		{UserID: "student18", DaysSinceActive: 4, GradeTrend: []float64{80}}, // <5 and flat
		{UserID: "student19", GradeTrend: []float64{82, 85}},
		{UserID: "student20", GradeTrend: []float64{90, 91, 92, 93}},
	}

	all := append(append(append([]RosterSignals{}, atRiskSignals...), slippingSignals...), healthySignals...)

	svc := NewRosterService(cache.NewMemory())
	entries, err := svc.GetRosterHealth(context.Background(), "eastfield", "CS101", "martinez",
		func(context.Context, string, string, string) ([]RosterSignals, error) {
			return all, nil
		})
	if err != nil {
		t.Fatal(err)
	}

	counts := map[string]int{}
	for _, e := range entries {
		counts[e.RiskLevel]++
	}
	if counts[RiskAtRisk] != 5 {
		t.Errorf("expected 5 at_risk, got %d", counts[RiskAtRisk])
	}
	if counts[RiskSlipping] != 5 {
		t.Errorf("expected 5 slipping, got %d", counts[RiskSlipping])
	}
	if counts[RiskHealthy] != 10 {
		t.Errorf("expected 10 healthy, got %d", counts[RiskHealthy])
	}
	if len(entries) != 20 {
		t.Errorf("expected 20 entries total, got %d", len(entries))
	}
}
