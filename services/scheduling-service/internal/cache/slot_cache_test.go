package cache

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

func newCache(t *testing.T) (Cache, *miniredis.Miniredis) {
	t.Helper()
	mr := miniredis.RunT(t)
	return NewRedisCache(redis.NewClient(&redis.Options{Addr: mr.Addr()}), time.Minute), mr
}

func TestRedisCache_SetGetInvalidate(t *testing.T) {
	ctx := context.Background()
	c, mr := newCache(t)

	if _, hit, err := c.Get(ctx, "acme", "inst-1", "2026-05-04", "2026-05-04"); err != nil || hit {
		t.Fatalf("empty: err=%v hit=%v", err, hit)
	}

	want := []Slot{{ScheduleID: "s1", InstructorID: "inst-1", SlotDate: "2026-05-04", SlotStartTime: "09:00", SlotEndTime: "09:15", Format: "online"}}
	if err := c.Set(ctx, "acme", "inst-1", "2026-05-04", "2026-05-04", want); err != nil {
		t.Fatalf("set: %v", err)
	}

	got, hit, err := c.Get(ctx, "acme", "inst-1", "2026-05-04", "2026-05-04")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if !hit {
		t.Fatal("want cache hit")
	}
	if len(got) != 1 || got[0].SlotStartTime != "09:00" {
		t.Fatalf("bad payload: %+v", got)
	}

	if err := c.InvalidateInstructor(ctx, "acme", "inst-1"); err != nil {
		t.Fatalf("invalidate: %v", err)
	}
	if _, hit, _ := c.Get(ctx, "acme", "inst-1", "2026-05-04", "2026-05-04"); hit {
		t.Fatal("expected miss after invalidate")
	}
	// Sanity — underlying key must be gone.
	if mr.Exists(Key("acme", "inst-1")) {
		t.Fatal("key still present after invalidate")
	}
}

// Cache is tenant-scoped: invalidating one tenant must not affect another.
func TestRedisCache_TenantIsolation(t *testing.T) {
	ctx := context.Background()
	c, _ := newCache(t)

	slots := []Slot{{ScheduleID: "s1", InstructorID: "inst-1"}}
	_ = c.Set(ctx, "acme", "inst-1", "d", "d", slots)
	_ = c.Set(ctx, "globex", "inst-1", "d", "d", slots)

	_ = c.InvalidateInstructor(ctx, "acme", "inst-1")

	if _, hit, _ := c.Get(ctx, "globex", "inst-1", "d", "d"); !hit {
		t.Fatal("globex cache leaked invalidation from acme")
	}
	if _, hit, _ := c.Get(ctx, "acme", "inst-1", "d", "d"); hit {
		t.Fatal("acme cache should be invalidated")
	}
}
