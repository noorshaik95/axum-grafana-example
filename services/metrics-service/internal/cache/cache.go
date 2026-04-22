package cache

import (
	"sync"
	"time"
)

// Cache is the abstraction for an in-memory / Redis-backed cache used by the
// metrics service. The implementation in this package is an in-memory TTL
// store; the same interface can be satisfied by a Redis-backed wrapper when
// shared state across instances is required.
type Cache interface {
	Get(key string) ([]byte, bool)
	Set(key string, value []byte, ttl time.Duration)
	Delete(key string)
	DeletePrefix(prefix string)
}

type entry struct {
	value     []byte
	expiresAt time.Time
}

// Memory is an in-memory TTL cache safe for concurrent use.
type Memory struct {
	mu    sync.RWMutex
	items map[string]entry
	now   func() time.Time
}

// NewMemory returns a new empty in-memory cache.
func NewMemory() *Memory {
	return &Memory{
		items: make(map[string]entry),
		now:   time.Now,
	}
}

func (c *Memory) Get(key string) ([]byte, bool) {
	c.mu.RLock()
	e, ok := c.items[key]
	c.mu.RUnlock()
	if !ok {
		return nil, false
	}
	if c.now().After(e.expiresAt) {
		c.mu.Lock()
		delete(c.items, key)
		c.mu.Unlock()
		return nil, false
	}
	out := make([]byte, len(e.value))
	copy(out, e.value)
	return out, true
}

func (c *Memory) Set(key string, value []byte, ttl time.Duration) {
	buf := make([]byte, len(value))
	copy(buf, value)
	c.mu.Lock()
	c.items[key] = entry{value: buf, expiresAt: c.now().Add(ttl)}
	c.mu.Unlock()
}

func (c *Memory) Delete(key string) {
	c.mu.Lock()
	delete(c.items, key)
	c.mu.Unlock()
}

// DeletePrefix removes every entry whose key starts with prefix.
func (c *Memory) DeletePrefix(prefix string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	for k := range c.items {
		if len(k) >= len(prefix) && k[:len(prefix)] == prefix {
			delete(c.items, k)
		}
	}
}

// setClock is test-only to deterministically drive expiry.
func (c *Memory) setClock(now func() time.Time) {
	c.mu.Lock()
	c.now = now
	c.mu.Unlock()
}
