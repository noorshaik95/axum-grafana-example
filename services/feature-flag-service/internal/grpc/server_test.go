package grpc

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	grpccodes "google.golang.org/grpc/codes"
	grpcstatus "google.golang.org/grpc/status"

	pb "slate/services/feature-flag-service/api/proto"
	"slate/services/feature-flag-service/internal/cache"
	"slate/services/feature-flag-service/internal/repository"
	"slate/services/feature-flag-service/internal/service"
)

// fakeStore is an in-memory repository.Store used to verify the server's
// cache interaction without standing up Postgres.
type fakeStore struct {
	mu       sync.Mutex
	flags    map[string]repository.Flag
	listCalls int
}

func newFakeStore(initial ...repository.Flag) *fakeStore {
	m := make(map[string]repository.Flag, len(initial))
	for _, f := range initial {
		if f.ID == uuid.Nil {
			f.ID = uuid.New()
		}
		if f.UpdatedAt.IsZero() {
			f.UpdatedAt = time.Now()
		}
		m[f.Key] = f
	}
	return &fakeStore{flags: m}
}

func (s *fakeStore) List(ctx context.Context) ([]repository.Flag, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.listCalls++
	out := make([]repository.Flag, 0, len(s.flags))
	for _, f := range s.flags {
		out = append(out, f)
	}
	return out, nil
}

func (s *fakeStore) GetByKey(ctx context.Context, key string) (*repository.Flag, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	f, ok := s.flags[key]
	if !ok {
		return nil, repository.ErrNotFound
	}
	return &f, nil
}

func (s *fakeStore) Upsert(ctx context.Context, key string, enabled bool, rules []repository.Rule) (*repository.Flag, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	f, ok := s.flags[key]
	if !ok {
		f = repository.Flag{ID: uuid.New(), Key: key}
	}
	f.Enabled = enabled
	f.Rules = append([]repository.Rule(nil), rules...)
	f.UpdatedAt = time.Now()
	s.flags[key] = f
	return &f, nil
}

func (s *fakeStore) Delete(ctx context.Context, key string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.flags[key]; !ok {
		return repository.ErrNotFound
	}
	delete(s.flags, key)
	return nil
}

func newServerWithRedis(t *testing.T, store repository.Store) (*Server, *miniredis.Miniredis) {
	t.Helper()
	mr := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = client.Close() })
	return NewServer(store, service.NewEvaluator(), cache.NewRedisCache(client, time.Minute)), mr
}

func TestEvaluateFlags_CacheMissThenHit(t *testing.T) {
	store := newFakeStore(
		repository.Flag{Key: "mobile_push", Enabled: true},
		repository.Flag{Key: "new_grading_queue", Enabled: false},
	)
	srv, _ := newServerWithRedis(t, store)
	ctx := context.Background()

	req := &pb.EvaluateFlagsRequest{
		TenantId:   "tenant-1",
		TenantSlug: "eastfield",
		UserId:     "user-1",
		Roles:      []string{"student"},
	}

	resp, err := srv.EvaluateFlags(ctx, req)
	if err != nil {
		t.Fatal(err)
	}
	if !resp.Flags["mobile_push"] || resp.Flags["new_grading_queue"] {
		t.Fatalf("unexpected evaluation: %+v", resp.Flags)
	}
	if store.listCalls != 1 {
		t.Fatalf("expected 1 list call on miss, got %d", store.listCalls)
	}

	// Second call should be served from cache — store should not be hit again.
	if _, err := srv.EvaluateFlags(ctx, req); err != nil {
		t.Fatal(err)
	}
	if store.listCalls != 1 {
		t.Fatalf("expected cache hit (still 1 list call), got %d", store.listCalls)
	}
}

func TestUpdateFlag_InvalidatesCache(t *testing.T) {
	store := newFakeStore(repository.Flag{Key: "mobile_push", Enabled: false})
	srv, _ := newServerWithRedis(t, store)
	ctx := context.Background()

	req := &pb.EvaluateFlagsRequest{TenantId: "tenant-1", UserId: "u", Roles: []string{"student"}}
	if _, err := srv.EvaluateFlags(ctx, req); err != nil {
		t.Fatal(err)
	}
	if _, err := srv.EvaluateFlags(ctx, req); err != nil {
		t.Fatal(err)
	}
	if store.listCalls != 1 {
		t.Fatalf("expected cache hit (1 list call), got %d", store.listCalls)
	}

	if _, err := srv.UpdateFlag(ctx, &pb.UpdateFlagRequest{
		Key:   "mobile_push",
		State: true,
		Targets: []*pb.TargetRule{{
			RuleType:      "role",
			RuleValueJson: `{"roles":["admin"]}`,
		}},
	}); err != nil {
		t.Fatal(err)
	}

	if _, err := srv.EvaluateFlags(ctx, req); err != nil {
		t.Fatal(err)
	}
	if store.listCalls != 2 {
		t.Fatalf("expected UpdateFlag to invalidate cache (2 list calls), got %d", store.listCalls)
	}
}

func TestDeleteFlag_NotFound(t *testing.T) {
	store := newFakeStore()
	srv, _ := newServerWithRedis(t, store)
	ctx := context.Background()

	_, err := srv.DeleteFlag(ctx, &pb.DeleteFlagRequest{Key: "missing"})
	if err == nil {
		t.Fatal("expected error on missing flag")
	}
}

func TestListFlags_ReturnsAllWithRules(t *testing.T) {
	store := newFakeStore(
		repository.Flag{Key: "a", Enabled: true, Rules: []repository.Rule{
			{RuleType: "role", RuleValue: `{"roles":["instructor"]}`},
		}},
		repository.Flag{Key: "b", Enabled: false},
	)
	srv, _ := newServerWithRedis(t, store)

	resp, err := srv.ListFlags(context.Background(), &pb.ListFlagsRequest{})
	if err != nil {
		t.Fatal(err)
	}
	if len(resp.Flags) != 2 {
		t.Fatalf("expected 2 flags, got %d", len(resp.Flags))
	}

	byKey := map[string]*pb.Flag{}
	for _, f := range resp.Flags {
		byKey[f.Key] = f
	}
	if byKey["a"] == nil || len(byKey["a"].Targets) != 1 {
		t.Fatalf("expected flag a with 1 target, got %+v", byKey["a"])
	}
	if byKey["b"] == nil || byKey["b"].State {
		t.Fatalf("expected flag b disabled, got %+v", byKey["b"])
	}
}

func TestEvaluateFlags_RequiresTenantID(t *testing.T) {
	srv, _ := newServerWithRedis(t, newFakeStore())
	_, err := srv.EvaluateFlags(context.Background(), &pb.EvaluateFlagsRequest{})
	if err == nil {
		t.Fatal("expected InvalidArgument for missing tenant_id")
	}
}

func TestGetFlag_ok(t *testing.T) {
	store := newFakeStore(repository.Flag{
		Key:     "mobile_push",
		Enabled: true,
		Rules: []repository.Rule{
			{RuleType: "role", RuleValue: `{"roles":["student"]}`},
		},
	})
	srv, _ := newServerWithRedis(t, store)

	resp, err := srv.GetFlag(context.Background(), &pb.GetFlagRequest{Key: "mobile_push"})
	if err != nil {
		t.Fatal(err)
	}
	if resp.Key != "mobile_push" || !resp.State {
		t.Fatalf("unexpected flag: %+v", resp)
	}
	if len(resp.Targets) != 1 || resp.Targets[0].RuleType != "role" {
		t.Fatalf("unexpected targets: %+v", resp.Targets)
	}
}

func TestGetFlag_notFound(t *testing.T) {
	srv, _ := newServerWithRedis(t, newFakeStore())

	_, err := srv.GetFlag(context.Background(), &pb.GetFlagRequest{Key: "missing"})
	if err == nil {
		t.Fatal("expected error")
	}
	if code := grpcstatus.Code(err); code != grpccodes.NotFound {
		t.Fatalf("expected NotFound, got %v", code)
	}
}

func TestGetFlag_emptyKey(t *testing.T) {
	srv, _ := newServerWithRedis(t, newFakeStore())

	_, err := srv.GetFlag(context.Background(), &pb.GetFlagRequest{})
	if err == nil {
		t.Fatal("expected error")
	}
	if code := grpcstatus.Code(err); code != grpccodes.InvalidArgument {
		t.Fatalf("expected InvalidArgument, got %v", code)
	}
}

// compile-time check that our fake satisfies the interface.
var _ repository.Store = (*fakeStore)(nil)

var _ = errors.New
