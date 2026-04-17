package ratelimit

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/go-redis/redis/v8"
)

// RateLimiter interface for rate limiting operations
type RateLimiter interface {
	AllowCreateTenant(clientIP string) (bool, int, error)
	AllowOperation(clientIP string, operation string) (bool, int, error)
}

// Config holds rate limiter configuration
type Config struct {
	// Create tenant limits (stricter)
	CreateTenantLimit   int // requests per window
	CreateTenantWindow  int // window in seconds

	// General operation limits
	OperationLimit  int
	OperationWindow int

	// Redis config
	RedisAddr     string
	RedisPassword string
	RedisDB       int
}

type rateLimiter struct {
	config      *Config
	redisClient *redis.Client
	fallback    *memoryRateLimiter
}

// memoryRateLimiter is a fallback in-memory rate limiter
type memoryRateLimiter struct {
	limits map[string]*limitEntry
	mu     sync.RWMutex
}

type limitEntry struct {
	count      int
	windowEnd  time.Time
	mu         sync.Mutex
}

// NewRateLimiter creates a new rate limiter with Redis and in-memory fallback
func NewRateLimiter(config *Config) (RateLimiter, error) {
	rl := &rateLimiter{
		config: config,
		fallback: &memoryRateLimiter{
			limits: make(map[string]*limitEntry),
		},
	}

	// Try to connect to Redis
	if config.RedisAddr != "" {
		rl.redisClient = redis.NewClient(&redis.Options{
			Addr:     config.RedisAddr,
			Password: config.RedisPassword,
			DB:       config.RedisDB,
		})

		// Test connection
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()

		if err := rl.redisClient.Ping(ctx).Err(); err != nil {
			// Redis not available, will use fallback
			rl.redisClient = nil
		}
	}

	// Start cleanup goroutine for memory limiter
	go rl.fallback.cleanup()

	return rl, nil
}

// AllowCreateTenant checks if tenant creation is allowed for the client
func (r *rateLimiter) AllowCreateTenant(clientIP string) (bool, int, error) {
	key := fmt.Sprintf("ratelimit:create_tenant:%s", clientIP)
	return r.checkLimit(key, r.config.CreateTenantLimit, r.config.CreateTenantWindow)
}

// AllowOperation checks if a general operation is allowed for the client
func (r *rateLimiter) AllowOperation(clientIP string, operation string) (bool, int, error) {
	key := fmt.Sprintf("ratelimit:%s:%s", operation, clientIP)
	return r.checkLimit(key, r.config.OperationLimit, r.config.OperationWindow)
}

// checkLimit checks rate limit using Redis or fallback to memory
func (r *rateLimiter) checkLimit(key string, limit int, windowSeconds int) (bool, int, error) {
	// Try Redis first
	if r.redisClient != nil {
		allowed, retryAfter, err := r.checkRedisLimit(key, limit, windowSeconds)
		if err == nil {
			return allowed, retryAfter, nil
		}
		// Redis failed, fall back to memory
	}

	// Use in-memory fallback
	allowed, retryAfter := r.fallback.checkLimit(key, limit, windowSeconds)
	return allowed, retryAfter, nil
}

// checkRedisLimit uses Redis for distributed rate limiting
func (r *rateLimiter) checkRedisLimit(key string, limit int, windowSeconds int) (bool, int, error) {
	ctx := context.Background()

	// Use Redis INCR with EXPIRE for simple sliding window
	pipe := r.redisClient.Pipeline()
	incr := pipe.Incr(ctx, key)
	pipe.Expire(ctx, key, time.Duration(windowSeconds)*time.Second)
	_, err := pipe.Exec(ctx)

	if err != nil {
		return false, 0, err
	}

	count := incr.Val()
	if count > int64(limit) {
		// Get TTL to calculate retry-after
		ttl, _ := r.redisClient.TTL(ctx, key).Result()
		retryAfter := int(ttl.Seconds())
		return false, retryAfter, nil
	}

	return true, 0, nil
}

// memoryRateLimiter methods

// checkLimit checks rate limit using in-memory storage
// Returns (allowed, retryAfter)
func (m *memoryRateLimiter) checkLimit(key string, limit int, windowSeconds int) (bool, int) {
	m.mu.Lock()
	entry, exists := m.limits[key]
	if !exists {
		entry = &limitEntry{
			count:     0,
			windowEnd: time.Now().Add(time.Duration(windowSeconds) * time.Second),
		}
		m.limits[key] = entry
	}
	m.mu.Unlock()

	entry.mu.Lock()
	defer entry.mu.Unlock()

	now := time.Now()

	// Reset if window expired
	if now.After(entry.windowEnd) {
		entry.count = 0
		entry.windowEnd = now.Add(time.Duration(windowSeconds) * time.Second)
	}

	// Check limit
	if entry.count >= limit {
		// Calculate retry-after as seconds until window expires
		retryAfter := int(entry.windowEnd.Sub(now).Seconds())
		if retryAfter < 0 {
			retryAfter = 0
		}
		return false, retryAfter
	}

	entry.count++
	return true, 0
}

// cleanup periodically removes expired entries
func (m *memoryRateLimiter) cleanup() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		m.mu.Lock()
		now := time.Now()
		for key, entry := range m.limits {
			entry.mu.Lock()
			if now.After(entry.windowEnd) {
				delete(m.limits, key)
			}
			entry.mu.Unlock()
		}
		m.mu.Unlock()
	}
}
