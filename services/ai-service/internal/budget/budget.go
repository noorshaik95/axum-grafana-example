// Package budget implements the W7.5 per-tenant monthly Claude token budget.
//
// Every Claude call must route through Check() before dispatch. If the
// tenant's cumulative monthly tokens have already breached the configured
// ceiling, Check returns ErrBudgetExceeded and the caller must not hit the
// Anthropic API. Record() is called after the response to add the actual
// input+output token counts to the Redis counter.
package budget

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// ErrBudgetExceeded is returned when the tenant has exhausted the current
// month's Claude token allowance. Callers surface this as a graceful
// degradation to the client.
var ErrBudgetExceeded = errors.New("tenant AI token budget exceeded for current month")

// Clock is injected so tests can pin the month window.
type Clock interface {
	Now() time.Time
}

type realClock struct{}

func (realClock) Now() time.Time { return time.Now().UTC() }

// Tracker enforces a monthly token ceiling per tenant slug. It is
// concurrency-safe because all mutation happens via atomic Redis ops.
type Tracker struct {
	rdb        redis.UniversalClient
	budget     int64
	clock      Clock
}

// NewTracker builds a Tracker with the given Redis client and monthly
// ceiling (in total input+output tokens).
func NewTracker(rdb redis.UniversalClient, monthlyBudget int64) *Tracker {
	return &Tracker{rdb: rdb, budget: monthlyBudget, clock: realClock{}}
}

// WithClock returns a tracker that uses the supplied clock. Intended for
// tests; the production code-path always uses the real clock.
func (t *Tracker) WithClock(c Clock) *Tracker {
	clone := *t
	clone.clock = c
	return &clone
}

// key composes the Redis counter key per the W7.5 spec:
//   tenant:{slug}:ai:tokens_used:{YYYY-MM}
func (t *Tracker) key(tenantSlug string) string {
	now := t.clock.Now()
	return fmt.Sprintf("tenant:%s:ai:tokens_used:%04d-%02d", tenantSlug, now.Year(), int(now.Month()))
}

// endOfMonthTTL computes the TTL from "now" to the end of the current
// calendar month. The Redis counter is reset implicitly by TTL expiry.
func (t *Tracker) endOfMonthTTL() time.Duration {
	now := t.clock.Now()
	firstOfNext := time.Date(now.Year(), now.Month()+1, 1, 0, 0, 0, 0, time.UTC)
	return firstOfNext.Sub(now)
}

// Check rejects the request if the current month's counter already exceeds
// the configured budget. A nil Tracker or non-positive budget disables the
// check (useful in tests that don't care about budget).
func (t *Tracker) Check(ctx context.Context, tenantSlug string) error {
	if t == nil || t.budget <= 0 {
		return nil
	}
	val, err := t.rdb.Get(ctx, t.key(tenantSlug)).Int64()
	if err != nil && !errors.Is(err, redis.Nil) {
		// Fail-closed on Redis errors would DoS the whole service if Redis
		// has a hiccup. Fail-open is the safer default — the next Record()
		// will still accrue usage and the subsequent Check() will catch up.
		return nil
	}
	if val >= t.budget {
		return ErrBudgetExceeded
	}
	return nil
}

// Record increments the monthly counter by the given number of tokens and
// refreshes the TTL to end-of-month. Safe to call with zero or negative
// token counts (no-op), so callers can pass raw SDK values without
// pre-validation.
func (t *Tracker) Record(ctx context.Context, tenantSlug string, tokens int64) error {
	if t == nil || tokens <= 0 {
		return nil
	}
	k := t.key(tenantSlug)
	pipe := t.rdb.TxPipeline()
	pipe.IncrBy(ctx, k, tokens)
	pipe.Expire(ctx, k, t.endOfMonthTTL())
	_, err := pipe.Exec(ctx)
	return err
}

// Used returns the current month's token count for the tenant (0 if none).
// Primarily for observability / admin endpoints.
func (t *Tracker) Used(ctx context.Context, tenantSlug string) (int64, error) {
	if t == nil {
		return 0, nil
	}
	val, err := t.rdb.Get(ctx, t.key(tenantSlug)).Int64()
	if errors.Is(err, redis.Nil) {
		return 0, nil
	}
	return val, err
}

// Budget returns the configured monthly token ceiling.
func (t *Tracker) Budget() int64 {
	if t == nil {
		return 0
	}
	return t.budget
}
