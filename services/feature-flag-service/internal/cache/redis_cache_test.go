package cache

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

func newTestCache(t *testing.T) (*RedisCache, *miniredis.Miniredis) {
	t.Helper()
	mr := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = client.Close() })
	return NewRedisCache(client, time.Minute), mr
}

func TestRedisCache_MissThenHit(t *testing.T) {
	c, _ := newTestCache(t)
	ctx := context.Background()

	if _, hit, err := c.Get(ctx, "tenant-1", "user-1", []string{"student"}); err != nil || hit {
		t.Fatalf("expected miss, got hit=%v err=%v", hit, err)
	}

	want := map[string]bool{"new_grading_queue": true, "mobile_push": false}
	if err := c.Set(ctx, "tenant-1", "user-1", []string{"student"}, want); err != nil {
		t.Fatal(err)
	}

	got, hit, err := c.Get(ctx, "tenant-1", "user-1", []string{"student"})
	if err != nil || !hit {
		t.Fatalf("expected hit, got hit=%v err=%v", hit, err)
	}
	if len(got) != len(want) || got["new_grading_queue"] != want["new_grading_queue"] || got["mobile_push"] != want["mobile_push"] {
		t.Fatalf("unexpected payload: %+v", got)
	}
}

func TestRedisCache_RolesOrderDoesNotAffectKey(t *testing.T) {
	c, _ := newTestCache(t)
	ctx := context.Background()

	payload := map[string]bool{"x": true}
	if err := c.Set(ctx, "t", "u", []string{"admin", "instructor"}, payload); err != nil {
		t.Fatal(err)
	}
	got, hit, err := c.Get(ctx, "t", "u", []string{"instructor", "admin"})
	if err != nil || !hit {
		t.Fatalf("expected hit with re-ordered roles, got hit=%v err=%v", hit, err)
	}
	if !got["x"] {
		t.Fatalf("expected payload preserved, got %+v", got)
	}
}

func TestRedisCache_InvalidateTenant(t *testing.T) {
	c, _ := newTestCache(t)
	ctx := context.Background()

	if err := c.Set(ctx, "tenant-a", "u1", []string{"student"}, map[string]bool{"f": true}); err != nil {
		t.Fatal(err)
	}
	if err := c.Set(ctx, "tenant-a", "u2", []string{"admin"}, map[string]bool{"f": true}); err != nil {
		t.Fatal(err)
	}
	if err := c.Set(ctx, "tenant-b", "u3", []string{"student"}, map[string]bool{"f": true}); err != nil {
		t.Fatal(err)
	}

	if err := c.InvalidateTenant(ctx, "tenant-a"); err != nil {
		t.Fatal(err)
	}

	if _, hit, _ := c.Get(ctx, "tenant-a", "u1", []string{"student"}); hit {
		t.Fatal("expected tenant-a u1 evicted")
	}
	if _, hit, _ := c.Get(ctx, "tenant-a", "u2", []string{"admin"}); hit {
		t.Fatal("expected tenant-a u2 evicted")
	}
	if _, hit, _ := c.Get(ctx, "tenant-b", "u3", []string{"student"}); !hit {
		t.Fatal("expected tenant-b u3 preserved")
	}
}

func TestRedisCache_InvalidateAll(t *testing.T) {
	c, _ := newTestCache(t)
	ctx := context.Background()

	_ = c.Set(ctx, "tenant-a", "u1", nil, map[string]bool{"f": true})
	_ = c.Set(ctx, "tenant-b", "u2", nil, map[string]bool{"f": true})

	if err := c.InvalidateTenant(ctx, ""); err != nil {
		t.Fatal(err)
	}
	if _, hit, _ := c.Get(ctx, "tenant-a", "u1", nil); hit {
		t.Fatal("expected tenant-a cleared on global invalidate")
	}
	if _, hit, _ := c.Get(ctx, "tenant-b", "u2", nil); hit {
		t.Fatal("expected tenant-b cleared on global invalidate")
	}
}
