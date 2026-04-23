package grpc

import (
	"testing"

	pb "slate/services/incident-service/api/proto"
	"slate/services/incident-service/internal/models"
)

func svc(s string) *string { return &s }

func TestAggregatePublicStatus_OperationalWhenEmpty(t *testing.T) {
	comps, overall := aggregatePublicStatus(map[string][]*models.Incident{})
	if len(comps) != 0 {
		t.Errorf("expected 0 components, got %d", len(comps))
	}
	if overall != pb.OverallStatus_OPERATIONAL {
		t.Errorf("expected OPERATIONAL when no open incidents, got %v", overall)
	}
}

func TestAggregatePublicStatus_RedFromP0(t *testing.T) {
	byService := map[string][]*models.Incident{
		"course-service": {
			{ID: "1", Priority: models.PriorityP0, Service: svc("course-service")},
			{ID: "2", Priority: models.PriorityP3, Service: svc("course-service")},
		},
	}
	comps, overall := aggregatePublicStatus(byService)
	if overall != pb.OverallStatus_OUTAGE {
		t.Errorf("expected OUTAGE, got %v", overall)
	}
	if len(comps) != 1 {
		t.Fatalf("expected 1 component, got %d", len(comps))
	}
	if comps[0].GetHealth() != pb.ComponentHealth_RED {
		t.Errorf("expected RED, got %v", comps[0].GetHealth())
	}
	if comps[0].GetOpenIncidents() != 2 {
		t.Errorf("expected 2 open, got %d", comps[0].GetOpenIncidents())
	}
	if comps[0].HighestPriority == nil || *comps[0].HighestPriority != pb.Priority_P0 {
		t.Errorf("expected highest P0, got %v", comps[0].HighestPriority)
	}
}

func TestAggregatePublicStatus_AmberFromP2(t *testing.T) {
	byService := map[string][]*models.Incident{
		"email-service": {
			{ID: "1", Priority: models.PriorityP2, Service: svc("email-service")},
		},
	}
	comps, overall := aggregatePublicStatus(byService)
	if overall != pb.OverallStatus_DEGRADED {
		t.Errorf("expected DEGRADED, got %v", overall)
	}
	if comps[0].GetHealth() != pb.ComponentHealth_AMBER {
		t.Errorf("expected AMBER, got %v", comps[0].GetHealth())
	}
}

func TestAggregatePublicStatus_GreenFromP4(t *testing.T) {
	byService := map[string][]*models.Incident{
		"metrics-service": {
			{ID: "1", Priority: models.PriorityP4, Service: svc("metrics-service")},
		},
	}
	comps, overall := aggregatePublicStatus(byService)
	if overall != pb.OverallStatus_OPERATIONAL {
		t.Errorf("expected OPERATIONAL (P4 doesn't degrade), got %v", overall)
	}
	if comps[0].GetHealth() != pb.ComponentHealth_GREEN {
		t.Errorf("expected GREEN, got %v", comps[0].GetHealth())
	}
}

func TestAggregatePublicStatus_RedBeatsAmberAcrossServices(t *testing.T) {
	byService := map[string][]*models.Incident{
		"course-service": {{ID: "1", Priority: models.PriorityP2, Service: svc("course-service")}},
		"email-service":  {{ID: "2", Priority: models.PriorityP1, Service: svc("email-service")}},
	}
	_, overall := aggregatePublicStatus(byService)
	if overall != pb.OverallStatus_OUTAGE {
		t.Errorf("expected OUTAGE when any service RED, got %v", overall)
	}
}

func TestAggregatePublicStatus_SkipsEmptyServiceKey(t *testing.T) {
	byService := map[string][]*models.Incident{
		"": {{ID: "1", Priority: models.PriorityP0}},
	}
	comps, overall := aggregatePublicStatus(byService)
	if len(comps) != 0 {
		t.Errorf("incidents with no service should be skipped, got %d components", len(comps))
	}
	if overall != pb.OverallStatus_OPERATIONAL {
		t.Errorf("expected OPERATIONAL with only unassigned incidents, got %v", overall)
	}
}

func TestPriorityConversionRoundTrip(t *testing.T) {
	for _, p := range []string{"P0", "P1", "P2", "P3", "P4"} {
		proto := priorityToProto(p)
		back, ok := priorityFromProto(proto)
		if !ok || back != p {
			t.Errorf("roundtrip %q: got %q ok=%v", p, back, ok)
		}
	}
}

func TestStatusConversionRoundTrip(t *testing.T) {
	for _, s := range []string{"open", "watching", "resolved"} {
		proto := statusToProto(s)
		back, ok := statusFromProto(proto)
		if !ok || back != s {
			t.Errorf("roundtrip %q: got %q ok=%v", s, back, ok)
		}
	}
}
