package budget

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
)

// fakeClock is a pinned UTC clock used to make Redis keys deterministic.
type fakeClock struct{ t time.Time }

func (c fakeClock) Now() time.Time { return c.t }

// inMemoryRedis is a tiny subset of redis.UniversalClient backed by a map.
// Only the calls Tracker actually uses are implemented; every other method
// panics so a future regression that reaches for unimplemented behavior is
// caught loudly in tests rather than silently returning zero values.
type inMemoryRedis struct {
	mu   sync.Mutex
	data map[string]int64
	redis.UniversalClient
}

func newInMemoryRedis() *inMemoryRedis {
	return &inMemoryRedis{data: map[string]int64{}}
}

func (r *inMemoryRedis) Get(ctx context.Context, key string) *redis.StringCmd {
	r.mu.Lock()
	defer r.mu.Unlock()
	cmd := redis.NewStringCmd(ctx, "get", key)
	v, ok := r.data[key]
	if !ok {
		cmd.SetErr(redis.Nil)
		return cmd
	}
	cmd.SetVal(strconv.FormatInt(v, 10))
	return cmd
}

// TxPipeline returns a minimal pipeline that accumulates IncrBy / Expire
// calls and applies them atomically-enough for tests on Exec.
func (r *inMemoryRedis) TxPipeline() redis.Pipeliner {
	return &fakePipeline{r: r}
}

type fakePipeline struct {
	r  *inMemoryRedis
	ops []func()
	redis.Pipeliner
}

func (p *fakePipeline) IncrBy(ctx context.Context, key string, value int64) *redis.IntCmd {
	cmd := redis.NewIntCmd(ctx, "incrby", key, value)
	p.ops = append(p.ops, func() {
		p.r.mu.Lock()
		p.r.data[key] += value
		cmd.SetVal(p.r.data[key])
		p.r.mu.Unlock()
	})
	return cmd
}

func (p *fakePipeline) Expire(ctx context.Context, key string, expiration time.Duration) *redis.BoolCmd {
	cmd := redis.NewBoolCmd(ctx, "expire", key, expiration.Seconds())
	p.ops = append(p.ops, func() { cmd.SetVal(true) })
	return cmd
}

func (p *fakePipeline) Exec(ctx context.Context) ([]redis.Cmder, error) {
	for _, op := range p.ops {
		op()
	}
	return nil, nil
}

func newTracker(t *testing.T, budget int64) (*Tracker, *inMemoryRedis) {
	t.Helper()
	r := newInMemoryRedis()
	tr := NewTracker(r, budget).WithClock(fakeClock{t: time.Date(2026, 4, 18, 12, 0, 0, 0, time.UTC)})
	return tr, r
}

// TestKeyFormat pins the Redis key format from the W7.5 spec so it can't
// drift silently under refactors (other services inspect this key for
// billing reconciliation).
func TestKeyFormat(t *testing.T) {
	tr, _ := newTracker(t, 1000)
	got := tr.key("acme")
	want := "tenant:acme:ai:tokens_used:2026-04"
	if got != want {
		t.Fatalf("key = %q want %q", got, want)
	}
}

// TestCheck_UnderBudget proves normal operation lets traffic through.
func TestCheck_UnderBudget(t *testing.T) {
	tr, _ := newTracker(t, 1000)
	if err := tr.Check(context.Background(), "acme"); err != nil {
		t.Fatalf("fresh tenant should not exceed: %v", err)
	}
	_ = tr.Record(context.Background(), "acme", 500)
	if err := tr.Check(context.Background(), "acme"); err != nil {
		t.Fatalf("halfway should not exceed: %v", err)
	}
}

// TestCheck_AtLimit is the load-bearing W7.5 cost-containment test:
// once usage hits the budget exactly, further calls MUST be rejected.
func TestCheck_AtLimit(t *testing.T) {
	tr, _ := newTracker(t, 1000)
	_ = tr.Record(context.Background(), "acme", 1000)
	err := tr.Check(context.Background(), "acme")
	if !errors.Is(err, ErrBudgetExceeded) {
		t.Fatalf("at-limit should return ErrBudgetExceeded, got %v", err)
	}
}

// TestCheck_OverLimit covers the >= semantics for safety overshoot.
func TestCheck_OverLimit(t *testing.T) {
	tr, _ := newTracker(t, 1000)
	_ = tr.Record(context.Background(), "acme", 1500)
	err := tr.Check(context.Background(), "acme")
	if !errors.Is(err, ErrBudgetExceeded) {
		t.Fatalf("over-limit should return ErrBudgetExceeded, got %v", err)
	}
}

// TestRecord_Accumulates proves multiple concurrent turns accumulate
// correctly into the monthly counter.
func TestRecord_Accumulates(t *testing.T) {
	tr, _ := newTracker(t, 10000)
	ctx := context.Background()
	for i := 0; i < 4; i++ {
		if err := tr.Record(ctx, "acme", 250); err != nil {
			t.Fatalf("record: %v", err)
		}
	}
	used, err := tr.Used(ctx, "acme")
	if err != nil {
		t.Fatalf("used: %v", err)
	}
	if used != 1000 {
		t.Fatalf("used = %d want 1000", used)
	}
}

// TestRecord_ZeroOrNegativeIsNoop guards the defensive path so callers can
// pass raw SDK token counts without pre-filtering.
func TestRecord_ZeroOrNegativeIsNoop(t *testing.T) {
	tr, _ := newTracker(t, 1000)
	ctx := context.Background()
	if err := tr.Record(ctx, "acme", 0); err != nil {
		t.Fatalf("record(0): %v", err)
	}
	if err := tr.Record(ctx, "acme", -5); err != nil {
		t.Fatalf("record(-5): %v", err)
	}
	if used, _ := tr.Used(ctx, "acme"); used != 0 {
		t.Fatalf("used = %d want 0", used)
	}
}

// TestDisabledBudgetAllowsEverything confirms a non-positive budget
// disables the gate (useful when running without budget enforcement).
func TestDisabledBudgetAllowsEverything(t *testing.T) {
	tr, _ := newTracker(t, 0)
	_ = tr.Record(context.Background(), "acme", 1_000_000)
	if err := tr.Check(context.Background(), "acme"); err != nil {
		t.Fatalf("disabled budget should never reject: %v", err)
	}
}

// TestKeyMonthBoundary ensures counter rolls over at month boundaries.
func TestKeyMonthBoundary(t *testing.T) {
	r := newInMemoryRedis()
	tr := NewTracker(r, 1000).WithClock(fakeClock{t: time.Date(2026, 4, 30, 23, 30, 0, 0, time.UTC)})
	if !strings.HasSuffix(tr.key("acme"), "2026-04") {
		t.Fatalf("late-april key wrong: %s", tr.key("acme"))
	}
	tr2 := NewTracker(r, 1000).WithClock(fakeClock{t: time.Date(2026, 5, 1, 0, 30, 0, 0, time.UTC)})
	if !strings.HasSuffix(tr2.key("acme"), "2026-05") {
		t.Fatalf("early-may key wrong: %s", tr2.key("acme"))
	}
}
