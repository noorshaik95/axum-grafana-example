// Package cache wraps Redis with typed JSON helpers used by every W7.2-W7.4
// handler. Cache keys are namespaced by tenant slug to keep multi-tenant
// isolation intact even if two tenants share a Redis instance.
package cache

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

type Cache struct {
	rdb redis.UniversalClient
}

func New(rdb redis.UniversalClient) *Cache { return &Cache{rdb: rdb} }

// GetJSON populates dst if the key is present. Returns (true, nil) on hit,
// (false, nil) on miss, (false, err) on I/O errors.
func (c *Cache) GetJSON(ctx context.Context, key string, dst any) (bool, error) {
	if c == nil || c.rdb == nil {
		return false, nil
	}
	b, err := c.rdb.Get(ctx, key).Bytes()
	if errors.Is(err, redis.Nil) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if err := json.Unmarshal(b, dst); err != nil {
		return false, err
	}
	return true, nil
}

// SetJSON stores src under key with the given TTL. A TTL of 0 persists the
// key indefinitely — callers should always pass a positive duration to
// avoid orphan entries.
func (c *Cache) SetJSON(ctx context.Context, key string, src any, ttl time.Duration) error {
	if c == nil || c.rdb == nil {
		return nil
	}
	b, err := json.Marshal(src)
	if err != nil {
		return err
	}
	return c.rdb.Set(ctx, key, b, ttl).Err()
}

// Del removes the given keys. Used by Kafka invalidation consumers.
func (c *Cache) Del(ctx context.Context, keys ...string) error {
	if c == nil || c.rdb == nil || len(keys) == 0 {
		return nil
	}
	return c.rdb.Del(ctx, keys...).Err()
}

// SHA1Hex returns a 40-char lowercase hex SHA-1 digest of s.
// Used as the query-hash component of the command-palette cache key.
func SHA1Hex(s string) string {
	sum := sha1.Sum([]byte(s))
	return hex.EncodeToString(sum[:])
}

// GradeKey: tenant:{slug}:grades:{user_id}:{course_id}
func GradeKey(slug, userID, courseID string) string {
	return fmt.Sprintf("tenant:%s:grades:%s:%s", slug, userID, courseID)
}

// StudyPlanKey: tenant:{slug}:study_plan:{user_id}
func StudyPlanKey(slug, userID string) string {
	return fmt.Sprintf("tenant:%s:study_plan:%s", slug, userID)
}

// CmdPaletteKey: tenant:{slug}:cmd_palette:{user_id}:{sha1(query)}
func CmdPaletteKey(slug, userID, query string) string {
	return fmt.Sprintf("tenant:%s:cmd_palette:%s:%s", slug, userID, SHA1Hex(query))
}
