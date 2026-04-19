package sso

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"sync"
	"time"
)

// stateStore is a simple in-memory CSRF state store used by every SSO
// strategy. A production deployment could swap this for a Redis-backed
// implementation, but the surface is small enough to share across tenants.
type stateStore struct {
	mu     sync.Mutex
	issued map[string]time.Time
	ttl    time.Duration
}

func newStateStore() *stateStore {
	return &stateStore{
		issued: make(map[string]time.Time),
		ttl:    10 * time.Minute,
	}
}

func (s *stateStore) issue() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("failed to generate state: %w", err)
	}
	state := base64.URLEncoding.EncodeToString(b)
	s.mu.Lock()
	s.issued[state] = time.Now()
	s.mu.Unlock()
	return state, nil
}

func (s *stateStore) consume(state string) error {
	if state == "" {
		return fmt.Errorf("state is required")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	issuedAt, ok := s.issued[state]
	if !ok {
		return fmt.Errorf("invalid or expired state")
	}
	delete(s.issued, state)
	if time.Since(issuedAt) > s.ttl {
		return fmt.Errorf("state expired")
	}
	return nil
}
