// Package cache — Redis-backed cache of evaluated flag maps.
//
// Key layout per plan W3.3 and CONTRACTS.md:
//   platform:flags:{tenant_id}:{sha1(user_id|roles)}
//
// TTL: 5 minutes. On UpdateFlag/DeleteFlag the evaluator invalidates the whole
// tenant namespace via SCAN + DEL (SCAN is cursor-safe for large keyspaces).
package cache

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

// DefaultTTL is the cache TTL for a single evaluation map.
const DefaultTTL = 5 * time.Minute

// Cache is the abstraction consumed by the gRPC server. Satisfied by
// *RedisCache for production and by a noop in tests / local dev without Redis.
type Cache interface {
	Get(ctx context.Context, tenantID, userID string, roles []string) (map[string]bool, bool, error)
	Set(ctx context.Context, tenantID, userID string, roles []string, flags map[string]bool) error
	InvalidateTenant(ctx context.Context, tenantID string) error
}

// RedisCache stores evaluated flag maps in Redis.
type RedisCache struct {
	client redis.UniversalClient
	ttl    time.Duration
}

// NewRedisCache returns a cache backed by the given go-redis client.
func NewRedisCache(client redis.UniversalClient, ttl time.Duration) *RedisCache {
	if ttl == 0 {
		ttl = DefaultTTL
	}
	return &RedisCache{client: client, ttl: ttl}
}

// Key returns the Redis key for the given evaluation identity.
func Key(tenantID, userID string, roles []string) string {
	sorted := append([]string(nil), roles...)
	sort.Strings(sorted)
	raw := userID + "|" + strings.Join(sorted, ",")
	sum := sha1.Sum([]byte(raw))
	return fmt.Sprintf("platform:flags:%s:%s", tenantID, hex.EncodeToString(sum[:]))
}

func tenantPattern(tenantID string) string {
	return fmt.Sprintf("platform:flags:%s:*", tenantID)
}

func (c *RedisCache) Get(ctx context.Context, tenantID, userID string, roles []string) (map[string]bool, bool, error) {
	raw, err := c.client.Get(ctx, Key(tenantID, userID, roles)).Bytes()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil, false, nil
		}
		return nil, false, fmt.Errorf("redis get: %w", err)
	}
	out := map[string]bool{}
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, false, fmt.Errorf("redis unmarshal: %w", err)
	}
	return out, true, nil
}

func (c *RedisCache) Set(ctx context.Context, tenantID, userID string, roles []string, flags map[string]bool) error {
	raw, err := json.Marshal(flags)
	if err != nil {
		return fmt.Errorf("redis marshal: %w", err)
	}
	if err := c.client.Set(ctx, Key(tenantID, userID, roles), raw, c.ttl).Err(); err != nil {
		return fmt.Errorf("redis set: %w", err)
	}
	return nil
}

// InvalidateTenant deletes every cached evaluation for the tenant. A missing
// tenant_id short-circuits to DEL all platform:flags:* (admin global edits).
func (c *RedisCache) InvalidateTenant(ctx context.Context, tenantID string) error {
	pattern := tenantPattern(tenantID)
	if tenantID == "" {
		pattern = "platform:flags:*"
	}
	iter := c.client.Scan(ctx, 0, pattern, 256).Iterator()
	var batch []string
	flush := func() error {
		if len(batch) == 0 {
			return nil
		}
		if err := c.client.Del(ctx, batch...).Err(); err != nil {
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

// NoopCache satisfies Cache when Redis is disabled (e.g. unit tests that don't
// care about caching behaviour or local dev with REDIS_ENABLED=false).
type NoopCache struct{}

func (NoopCache) Get(context.Context, string, string, []string) (map[string]bool, bool, error) {
	return nil, false, nil
}
func (NoopCache) Set(context.Context, string, string, []string, map[string]bool) error { return nil }
func (NoopCache) InvalidateTenant(context.Context, string) error                       { return nil }
