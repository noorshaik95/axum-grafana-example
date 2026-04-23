package broadcast

import (
	"context"
	"errors"
	"testing"

	"slate/services/email-service/internal/email"
	"slate/services/email-service/internal/tenants"
)

func TestDispatch_MissingMessage(t *testing.T) {
	s := New(nil, &email.RecordingSender{}, tenants.NewStaticResolver(nil))
	_, err := s.Dispatch(context.Background(), Request{Targets: Targets{AllTenants: true}, Channels: []string{"in_app"}})
	var ve *ValidationError
	if !errors.As(err, &ve) {
		t.Fatalf("expected ValidationError, got %v", err)
	}
}

func TestDispatch_MutuallyExclusiveTargets(t *testing.T) {
	s := New(nil, &email.RecordingSender{}, tenants.NewStaticResolver(nil))
	_, err := s.Dispatch(context.Background(), Request{
		Message:  "x",
		Channels: []string{"in_app"},
		Targets:  Targets{AllTenants: true, TenantIDs: []string{"t1"}},
	})
	var ve *ValidationError
	if !errors.As(err, &ve) {
		t.Fatalf("expected ValidationError, got %v", err)
	}
}

func TestDispatch_EmailFanOut(t *testing.T) {
	sender := &email.RecordingSender{}
	resolver := tenants.NewStaticResolver([]tenants.Admin{
		{TenantID: "t1", Email: "a@e.com"},
		{TenantID: "t2", Email: "b@e.com"},
	})
	s := New(nil, sender, resolver)
	res, err := s.Dispatch(context.Background(), Request{
		AuthorID: "admin-1",
		Message:  "restart",
		Channels: []string{"email_admins"},
		Targets:  Targets{TenantIDs: []string{"t1", "t2"}},
	})
	if err != nil {
		t.Fatalf("Dispatch: %v", err)
	}
	if res.EmailsSent != 2 {
		t.Errorf("EmailsSent = %d, want 2", res.EmailsSent)
	}
	if len(sender.Sent) != 2 {
		t.Errorf("recorded = %d, want 2", len(sender.Sent))
	}
}

func TestDispatch_AllTenantsResolves(t *testing.T) {
	sender := &email.RecordingSender{}
	resolver := tenants.NewStaticResolver([]tenants.Admin{
		{TenantID: "x", Email: "x@e.com"},
		{TenantID: "y", Email: "y@e.com"},
		{TenantID: "z", Email: "z@e.com"},
	})
	s := New(nil, sender, resolver)
	res, err := s.Dispatch(context.Background(), Request{
		AuthorID: "admin",
		Message:  "ping",
		Channels: []string{"email_admins"},
		Targets:  Targets{AllTenants: true},
	})
	if err != nil {
		t.Fatalf("Dispatch: %v", err)
	}
	if res.EmailsSent != 3 {
		t.Errorf("EmailsSent = %d, want 3", res.EmailsSent)
	}
	if len(res.TenantIDs) != 3 {
		t.Errorf("TenantIDs len = %d, want 3", len(res.TenantIDs))
	}
}

func TestDispatch_UnknownChannel(t *testing.T) {
	s := New(nil, &email.RecordingSender{}, tenants.NewStaticResolver(nil))
	_, err := s.Dispatch(context.Background(), Request{
		Message:  "x",
		Channels: []string{"sms"},
		Targets:  Targets{TenantIDs: []string{"t1"}},
	})
	var ve *ValidationError
	if !errors.As(err, &ve) {
		t.Fatalf("expected ValidationError, got %v", err)
	}
}
