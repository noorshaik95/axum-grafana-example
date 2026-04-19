package repository

import (
	"context"
	"testing"

	"slate/services/user-auth-service/internal/models"
)

// TestInMemoryAudit_AllSevenActions proves every action type specified in
// plan.md W14.3 can round-trip through the repository (plan.md lists 8 action
// types; the legacy plan wording said 7 but includes both
// impersonation_start and impersonation_end — we cover all of them).
func TestInMemoryAudit_AllActionsRoundTrip(t *testing.T) {
	repo := NewInMemoryAuditRepository()
	actions := []models.AuditAction{
		models.AuditActionLogin,
		models.AuditActionLogout,
		models.AuditActionFailedLogin,
		models.AuditActionPasswordChange,
		models.AuditActionMFAChange,
		models.AuditActionImpersonationStart,
		models.AuditActionImpersonationEnd,
		models.AuditActionSSOLogin,
	}
	for _, a := range actions {
		if err := repo.Append(context.Background(), &models.AuditEvent{
			ActorID:   "u",
			ActorType: models.AuditActorUser,
			Action:    a,
			TargetID:  "u",
			IPAddress: "1.2.3.4",
			Metadata:  map[string]interface{}{"source": "test"},
		}); err != nil {
			t.Fatalf("append %s: %v", a, err)
		}
	}
	if got := repo.Len(); got != len(actions) {
		t.Fatalf("want %d events, got %d", len(actions), got)
	}
	for _, a := range actions {
		if got := repo.ByAction(a); len(got) != 1 {
			t.Fatalf("action %s missing", a)
		}
	}
}

func TestInMemoryAudit_RequiresAction(t *testing.T) {
	repo := NewInMemoryAuditRepository()
	if err := repo.Append(context.Background(), &models.AuditEvent{ActorID: "u"}); err == nil {
		t.Fatal("want error when action missing")
	}
}

func TestInMemoryAudit_ListFiltersByActor(t *testing.T) {
	repo := NewInMemoryAuditRepository()
	_ = repo.Append(context.Background(), &models.AuditEvent{ActorID: "a", Action: models.AuditActionLogin})
	_ = repo.Append(context.Background(), &models.AuditEvent{ActorID: "b", Action: models.AuditActionLogin})
	out, _ := repo.List(context.Background(), "a", 10)
	if len(out) != 1 || out[0].ActorID != "a" {
		t.Fatalf("bad list result: %+v", out)
	}
}
