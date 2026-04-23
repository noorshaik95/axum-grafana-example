package docker

import (
	"context"
	"fmt"

	"github.com/go-redis/redis/v8"
)

// RedisKeyFlusher deletes every key under `tenant:{slug}:*` before the
// DockerProvisioner tears down a tenant's containers (plan W16.4).
type RedisKeyFlusher interface {
	FlushTenant(ctx context.Context, slug string) error
	Close() error
}

// redisFlusher is the go-redis/v8 implementation. We use v8 because the rest
// of tenant-service already depends on it (ratelimit package).
type redisFlusher struct{ client *redis.Client }

// NewRedisFlusher returns a flusher backed by the given addr. Nil/empty addr
// yields a no-op flusher so unit tests + fake-provisioner runs don't require
// Redis to be up.
func NewRedisFlusher(addr, password string, db int) RedisKeyFlusher {
	if addr == "" {
		return &noopFlusher{}
	}
	return &redisFlusher{
		client: redis.NewClient(&redis.Options{Addr: addr, Password: password, DB: db}),
	}
}

func (f *redisFlusher) FlushTenant(ctx context.Context, slug string) error {
	if slug == "" {
		return fmt.Errorf("slug is required")
	}
	pattern := fmt.Sprintf("tenant:%s:*", slug)
	iter := f.client.Scan(ctx, 0, pattern, 256).Iterator()
	batch := make([]string, 0, 256)
	flush := func() error {
		if len(batch) == 0 {
			return nil
		}
		if err := f.client.Del(ctx, batch...).Err(); err != nil {
			return fmt.Errorf("redis del: %w", err)
		}
		batch = batch[:0]
		return nil
	}
	for iter.Next(ctx) {
		batch = append(batch, iter.Val())
		if len(batch) >= 256 {
			if err := flush(); err != nil {
				return err
			}
		}
	}
	if err := iter.Err(); err != nil {
		return fmt.Errorf("redis scan: %w", err)
	}
	return flush()
}

func (f *redisFlusher) Close() error {
	if f.client == nil {
		return nil
	}
	return f.client.Close()
}

// noopFlusher is returned when Redis isn't configured; it makes tenant
// operations succeed without hitting Redis.
type noopFlusher struct{}

func (noopFlusher) FlushTenant(context.Context, string) error { return nil }
func (noopFlusher) Close() error                              { return nil }
