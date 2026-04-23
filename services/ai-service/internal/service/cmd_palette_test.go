package service

import (
	"context"
	"errors"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"

	aipb "slate/services/ai-service/api/proto"
	"slate/services/ai-service/internal/cache"
	"slate/services/ai-service/internal/claude"
)

// stubInvoker is a deterministic Invoker for tests — it records the last
// prompt it saw (so we can assert prompt_caching fields are wired) and
// returns a canned JSON response.
type stubInvoker struct {
	calls    int32
	lastPrompt claude.Prompt
	response   string
}

func (s *stubInvoker) Invoke(_ context.Context, _ string, p claude.Prompt) (claude.Result, error) {
	atomic.AddInt32(&s.calls, 1)
	s.lastPrompt = p
	return claude.Result{Text: s.response, InputTokens: 10, OutputTokens: 5, Model: "claude-sonnet-4-6"}, nil
}

// inMemCache is a minimal redis-shaped store for caching tests.
type inMemCache struct {
	data map[string][]byte
	redis.UniversalClient
}

func (r *inMemCache) Get(ctx context.Context, key string) *redis.StringCmd {
	cmd := redis.NewStringCmd(ctx, "get", key)
	v, ok := r.data[key]
	if !ok {
		cmd.SetErr(redis.Nil)
		return cmd
	}
	cmd.SetVal(string(v))
	return cmd
}
func (r *inMemCache) Set(ctx context.Context, key string, value any, _ time.Duration) *redis.StatusCmd {
	cmd := redis.NewStatusCmd(ctx, "set", key, value)
	switch v := value.(type) {
	case []byte:
		r.data[key] = append([]byte(nil), v...)
	case string:
		r.data[key] = []byte(v)
	}
	cmd.SetVal("OK")
	return cmd
}
func (r *inMemCache) Del(ctx context.Context, keys ...string) *redis.IntCmd {
	cmd := redis.NewIntCmd(ctx, "del", keys)
	var n int64
	for _, k := range keys {
		if _, ok := r.data[k]; ok {
			delete(r.data, k)
			n++
		}
	}
	cmd.SetVal(n)
	return cmd
}

func newCache() (*cache.Cache, *inMemCache) {
	m := &inMemCache{data: map[string][]byte{}}
	return cache.New(m), m
}

// TestCmdPalette_CachesResult verifies that W7.4's 30s cache key is
// populated on a MISS and reused on a HIT — the exact cost-reduction path.
func TestCmdPalette_CachesResult(t *testing.T) {
	stub := &stubInvoker{response: `{"results":[{"label":"View assignments","route":"/courses/x/assignments","icon":"book-open"}]}`}
	c, _ := newCache()
	p := NewCmdPalette(stub, c, "claude-sonnet-4-6", 400)

	req := &aipb.CmdPaletteRequest{
		TenantSlug: "acme", UserId: "u1", Query: "my assignments",
		RouteCatalog: []*aipb.CmdPaletteRoute{
			{Label: "Assignments", Route: "/courses/x/assignments"},
		},
	}

	resp, err := p.Resolve(context.Background(), req)
	if err != nil {
		t.Fatalf("first call: %v", err)
	}
	if resp.FromCache {
		t.Fatalf("first call should be a MISS")
	}
	if len(resp.Results) != 1 || resp.Results[0].Route != "/courses/x/assignments" {
		t.Fatalf("unexpected results: %+v", resp.Results)
	}

	resp2, err := p.Resolve(context.Background(), req)
	if err != nil {
		t.Fatalf("second call: %v", err)
	}
	if !resp2.FromCache {
		t.Fatalf("second call should be a HIT")
	}
	if atomic.LoadInt32(&stub.calls) != 1 {
		t.Fatalf("claude should have been called exactly once, got %d", stub.calls)
	}
}

// TestCmdPalette_FiltersHallucinatedRoutes is the bounded-routes
// guarantee from W7.4 — routes NOT in the catalog must be dropped.
func TestCmdPalette_FiltersHallucinatedRoutes(t *testing.T) {
	stub := &stubInvoker{response: `{"results":[
		{"label":"Real","route":"/real","icon":"a"},
		{"label":"Hallucinated","route":"/does-not-exist","icon":"b"}
	]}`}
	c, _ := newCache()
	p := NewCmdPalette(stub, c, "claude-sonnet-4-6", 400)

	req := &aipb.CmdPaletteRequest{
		TenantSlug: "acme", UserId: "u1", Query: "anything",
		RouteCatalog: []*aipb.CmdPaletteRoute{{Label: "R", Route: "/real"}},
	}
	resp, err := p.Resolve(context.Background(), req)
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if len(resp.Results) != 1 || resp.Results[0].Route != "/real" {
		t.Fatalf("hallucinated route leaked: %+v", resp.Results)
	}
}

// TestCmdPalette_StablePromptForCaching enforces the prompt-caching
// invariant: the system prompt must NEVER vary between requests with the
// same purpose — that's what lets cache_control actually land a hit.
func TestCmdPalette_StablePromptForCaching(t *testing.T) {
	stub := &stubInvoker{response: `{"results":[]}`}
	c, _ := newCache()
	p := NewCmdPalette(stub, c, "claude-sonnet-4-6", 400)

	req1 := &aipb.CmdPaletteRequest{TenantSlug: "acme", UserId: "u1", Query: "one", RouteCatalog: []*aipb.CmdPaletteRoute{{Route: "/a"}}}
	req2 := &aipb.CmdPaletteRequest{TenantSlug: "acme", UserId: "u2", Query: "two", RouteCatalog: []*aipb.CmdPaletteRoute{{Route: "/b"}}}

	if _, err := p.Resolve(context.Background(), req1); err != nil {
		t.Fatalf("first: %v", err)
	}
	first := stub.lastPrompt.SystemPrompt
	if _, err := p.Resolve(context.Background(), req2); err != nil {
		t.Fatalf("second: %v", err)
	}
	second := stub.lastPrompt.SystemPrompt
	if first != second {
		t.Fatalf("system prompt must be stable for prompt caching; got divergence:\n%s\n---\n%s", first, second)
	}
	if stub.lastPrompt.Purpose != "cmd_palette" {
		t.Fatalf("purpose must be set so the span gets tagged; got %q", stub.lastPrompt.Purpose)
	}
}

// TestCmdPalette_EmptyQueryRejected defends the API from obviously-bad
// inputs; we don't want to burn tokens on "   ".
func TestCmdPalette_EmptyQueryRejected(t *testing.T) {
	stub := &stubInvoker{response: ""}
	c, _ := newCache()
	p := NewCmdPalette(stub, c, "claude-sonnet-4-6", 400)

	_, err := p.Resolve(context.Background(), &aipb.CmdPaletteRequest{TenantSlug: "acme", UserId: "u1", Query: "   "})
	if err == nil || !strings.Contains(err.Error(), "query required") {
		t.Fatalf("expected query-required error, got %v", err)
	}
}

// errInvoker simulates the budget-exceeded path reaching the handler.
type errInvoker struct{ err error }

func (e errInvoker) Invoke(context.Context, string, claude.Prompt) (claude.Result, error) {
	return claude.Result{}, e.err
}

// TestCmdPalette_PropagatesBudgetError proves the handler passes the
// sentinel upstream unchanged so the gRPC layer can map it.
func TestCmdPalette_PropagatesBudgetError(t *testing.T) {
	sentinel := errors.New("budget boom")
	c, _ := newCache()
	p := NewCmdPalette(errInvoker{err: sentinel}, c, "claude-sonnet-4-6", 400)
	_, err := p.Resolve(context.Background(), &aipb.CmdPaletteRequest{
		TenantSlug: "acme", UserId: "u1", Query: "anything",
		RouteCatalog: []*aipb.CmdPaletteRoute{{Route: "/x"}},
	})
	if !errors.Is(err, sentinel) {
		t.Fatalf("expected sentinel, got %v", err)
	}
}
