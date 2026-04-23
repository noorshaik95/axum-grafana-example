// Package cache — Redis-backed cache of generated office-hour slots.
//
// Key layout per docs/plan.md W6.3:
//   tenant:{slug}:oh_slots:{instructor_id}
//
// Callers encode the full query (date range) into the cached payload so that a
// wider request cannot be silently answered from a narrower cached response —
// on range mismatch we treat it as a miss. This keeps Set/Get tenant-scoped and
// single-key, which makes InvalidateInstructor O(1).
package cache

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

const DefaultTTL = 5 * time.Minute

// Slot mirrors the wire shape the gRPC server serves.
type Slot struct {
	ScheduleID    string `json:"schedule_id"`
	InstructorID  string `json:"instructor_id"`
	SlotDate      string `json:"slot_date"`
	SlotStartTime string `json:"slot_start_time"`
	SlotEndTime   string `json:"slot_end_time"`
	Format        string `json:"format"`
	Location      string `json:"location,omitempty"`
}

// entry bundles the cached slots with the query range so we can detect
// range-mismatch misses.
type entry struct {
	DateFrom string `json:"date_from"`
	DateTo   string `json:"date_to"`
	Slots    []Slot `json:"slots"`
}

// Cache is the abstraction consumed by the service layer. RedisCache for prod,
// NoopCache for unit tests that don't exercise caching behaviour.
type Cache interface {
	Get(ctx context.Context, tenantSlug, instructorID, dateFrom, dateTo string) ([]Slot, bool, error)
	Set(ctx context.Context, tenantSlug, instructorID, dateFrom, dateTo string, slots []Slot) error
	InvalidateInstructor(ctx context.Context, tenantSlug, instructorID string) error
}

type RedisCache struct {
	client redis.UniversalClient
	ttl    time.Duration
}

func NewRedisCache(client redis.UniversalClient, ttl time.Duration) *RedisCache {
	if ttl == 0 {
		ttl = DefaultTTL
	}
	return &RedisCache{client: client, ttl: ttl}
}

func Key(tenantSlug, instructorID string) string {
	return fmt.Sprintf("tenant:%s:oh_slots:%s", tenantSlug, instructorID)
}

func (c *RedisCache) Get(ctx context.Context, tenantSlug, instructorID, dateFrom, dateTo string) ([]Slot, bool, error) {
	raw, err := c.client.Get(ctx, Key(tenantSlug, instructorID)).Bytes()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil, false, nil
		}
		return nil, false, fmt.Errorf("redis get: %w", err)
	}
	var e entry
	if err := json.Unmarshal(raw, &e); err != nil {
		return nil, false, fmt.Errorf("redis unmarshal: %w", err)
	}
	if e.DateFrom != dateFrom || e.DateTo != dateTo {
		// Different window → treat as miss. A lookup for the same exact window hits.
		return nil, false, nil
	}
	return e.Slots, true, nil
}

func (c *RedisCache) Set(ctx context.Context, tenantSlug, instructorID, dateFrom, dateTo string, slots []Slot) error {
	raw, err := json.Marshal(entry{DateFrom: dateFrom, DateTo: dateTo, Slots: slots})
	if err != nil {
		return fmt.Errorf("redis marshal: %w", err)
	}
	if err := c.client.Set(ctx, Key(tenantSlug, instructorID), raw, c.ttl).Err(); err != nil {
		return fmt.Errorf("redis set: %w", err)
	}
	return nil
}

func (c *RedisCache) InvalidateInstructor(ctx context.Context, tenantSlug, instructorID string) error {
	if err := c.client.Del(ctx, Key(tenantSlug, instructorID)).Err(); err != nil {
		return fmt.Errorf("redis del: %w", err)
	}
	return nil
}

// NoopCache — satisfies Cache when Redis is disabled.
type NoopCache struct{}

func (NoopCache) Get(context.Context, string, string, string, string) ([]Slot, bool, error) {
	return nil, false, nil
}
func (NoopCache) Set(context.Context, string, string, string, string, []Slot) error { return nil }
func (NoopCache) InvalidateInstructor(context.Context, string, string) error        { return nil }
