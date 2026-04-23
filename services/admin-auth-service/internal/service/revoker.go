package service

import (
	"context"
	"sync"
	"time"
)

// MemoryRevoker is an in-process revocation store used when Redis is unavailable.
// It is safe for concurrent use and prunes expired entries on access.
type MemoryRevoker struct {
	mu      sync.Mutex
	entries map[string]time.Time
}

// NewMemoryRevoker returns a new MemoryRevoker.
func NewMemoryRevoker() *MemoryRevoker {
	return &MemoryRevoker{entries: map[string]time.Time{}}
}

// Revoke adds jti to the blacklist for the given ttl.
func (r *MemoryRevoker) Revoke(_ context.Context, jti string, ttl time.Duration) error {
	if jti == "" || ttl <= 0 {
		return nil
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.entries[jti] = time.Now().Add(ttl)
	return nil
}

// IsRevoked reports whether jti is currently blacklisted.
func (r *MemoryRevoker) IsRevoked(_ context.Context, jti string) (bool, error) {
	if jti == "" {
		return false, nil
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	exp, ok := r.entries[jti]
	if !ok {
		return false, nil
	}
	if time.Now().After(exp) {
		delete(r.entries, jti)
		return false, nil
	}
	return true, nil
}
