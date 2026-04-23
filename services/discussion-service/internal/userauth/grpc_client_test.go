package userauth

import (
	"context"
	"errors"
	"testing"

	userpb "slate/services/user-auth-service/api/proto"

	"google.golang.org/grpc"
)

// fakeUserClient implements userpb.UserServiceClient for the methods we touch
// (only ResolveUsername). All other methods panic — tests should never hit
// them.
type fakeUserClient struct {
	userpb.UserServiceClient

	resolveFn func(ctx context.Context, in *userpb.ResolveUsernameRequest) (*userpb.ResolveUsernameResponse, error)
}

func (f *fakeUserClient) ResolveUsername(ctx context.Context, in *userpb.ResolveUsernameRequest, _ ...grpc.CallOption) (*userpb.ResolveUsernameResponse, error) {
	return f.resolveFn(ctx, in)
}

func TestGRPCResolverReKeysByOriginalCasing(t *testing.T) {
	fake := &fakeUserClient{
		resolveFn: func(_ context.Context, in *userpb.ResolveUsernameRequest) (*userpb.ResolveUsernameResponse, error) {
			if in.TenantId != "t1" {
				t.Errorf("tenant_id = %q, want t1", in.TenantId)
			}
			// Server normalizes keys to lowercase, per CONTRACTS.md.
			return &userpb.ResolveUsernameResponse{
				UserIdsByUsername: map[string]string{
					"alice":     "user-alice",
					"prof.bob":  "user-bob",
				},
			}, nil
		},
	}
	r := NewGRPCResolverWithClient(fake)

	got, err := r.ResolveUsernames(context.Background(), "t1", []string{"Alice", "prof.bob", "ghost"})
	if err != nil {
		t.Fatalf("ResolveUsernames: %v", err)
	}
	// Preserves caller's original casing; omits unknowns.
	if got["Alice"] != "user-alice" {
		t.Errorf("Alice → %q, want user-alice", got["Alice"])
	}
	if got["prof.bob"] != "user-bob" {
		t.Errorf("prof.bob → %q, want user-bob", got["prof.bob"])
	}
	if _, ok := got["ghost"]; ok {
		t.Errorf("ghost should be omitted; got %q", got["ghost"])
	}
	if len(got) != 2 {
		t.Errorf("want 2 resolved users, got %d: %v", len(got), got)
	}
}

func TestGRPCResolverEmptyInputShortCircuits(t *testing.T) {
	called := false
	fake := &fakeUserClient{
		resolveFn: func(context.Context, *userpb.ResolveUsernameRequest) (*userpb.ResolveUsernameResponse, error) {
			called = true
			return nil, nil
		},
	}
	r := NewGRPCResolverWithClient(fake)
	got, err := r.ResolveUsernames(context.Background(), "t1", nil)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("want empty map, got %v", got)
	}
	if called {
		t.Errorf("server should not be called for empty usernames")
	}
}

func TestGRPCResolverSurfacesServerErrors(t *testing.T) {
	fake := &fakeUserClient{
		resolveFn: func(context.Context, *userpb.ResolveUsernameRequest) (*userpb.ResolveUsernameResponse, error) {
			return nil, errors.New("upstream unavailable")
		},
	}
	r := NewGRPCResolverWithClient(fake)
	if _, err := r.ResolveUsernames(context.Background(), "t1", []string{"alice"}); err == nil {
		t.Fatalf("expected error, got nil")
	}
}
