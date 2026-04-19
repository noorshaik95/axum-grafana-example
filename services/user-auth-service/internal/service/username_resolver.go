// username_resolver.go — batched @handle → user_id lookup used by
// discussion-service CreatePost and email-service broadcast. Volume is low so
// a 30-second in-memory cache keyed by lowercased username is enough to
// protect the DB from repeat mention resolution during a discussion thread
// render.

package service

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"sync"
	"time"
)

// UsernameLookup is the subset of repository operations the resolver needs.
// Kept small so tests can inject a fake without touching the real user repo.
type UsernameLookup interface {
	LookupUsernames(ctx context.Context, usernames []string) (map[string]string, error)
}

// UsernameResolver resolves usernames to user_ids with a short positive-only
// TTL cache. Negative results (unknown usernames) are NOT cached so newly
// provisioned users become resolvable immediately.
type UsernameResolver struct {
	repo  UsernameLookup
	ttl   time.Duration
	now   func() time.Time
	mu    sync.Mutex
	cache map[string]cacheEntry
}

type cacheEntry struct {
	userID string
	expiry time.Time
}

// NewUsernameResolver builds a resolver. ttl<=0 defaults to 30 seconds.
func NewUsernameResolver(repo UsernameLookup, ttl time.Duration) *UsernameResolver {
	if ttl <= 0 {
		ttl = 30 * time.Second
	}
	return &UsernameResolver{repo: repo, ttl: ttl, now: time.Now, cache: make(map[string]cacheEntry)}
}

// Resolve returns a map of lowercased-username → user_id. Unknown usernames
// are omitted from the result (not an error). Input is deduplicated + lowered
// before the DB roundtrip.
func (r *UsernameResolver) Resolve(ctx context.Context, usernames []string) (map[string]string, error) {
	if len(usernames) == 0 {
		return map[string]string{}, nil
	}
	normalized := make(map[string]struct{}, len(usernames))
	for _, u := range usernames {
		trimmed := strings.ToLower(strings.TrimSpace(u))
		if trimmed == "" {
			continue
		}
		normalized[trimmed] = struct{}{}
	}
	if len(normalized) == 0 {
		return map[string]string{}, nil
	}

	result := make(map[string]string, len(normalized))
	var missing []string
	now := r.now()
	r.mu.Lock()
	for u := range normalized {
		if e, ok := r.cache[u]; ok && now.Before(e.expiry) {
			result[u] = e.userID
		} else {
			missing = append(missing, u)
		}
	}
	r.mu.Unlock()

	if len(missing) == 0 {
		return result, nil
	}
	found, err := r.repo.LookupUsernames(ctx, missing)
	if err != nil {
		return nil, fmt.Errorf("username lookup failed: %w", err)
	}
	expiry := now.Add(r.ttl)
	r.mu.Lock()
	for u, id := range found {
		r.cache[u] = cacheEntry{userID: id, expiry: expiry}
		result[u] = id
	}
	r.mu.Unlock()
	return result, nil
}

// SetClock overrides the internal clock. Tests only.
func (r *UsernameResolver) SetClock(now func() time.Time) { r.now = now }

// Invalidate drops a single username from the positive cache. Called on user
// update/rename; harmless if the username was never cached.
func (r *UsernameResolver) Invalidate(username string) {
	u := strings.ToLower(strings.TrimSpace(username))
	if u == "" {
		return
	}
	r.mu.Lock()
	delete(r.cache, u)
	r.mu.Unlock()
}

// SQLUsernameLookup implements UsernameLookup against the users table.
type SQLUsernameLookup struct {
	db *sql.DB
}

// NewSQLUsernameLookup wraps a *sql.DB for production use.
func NewSQLUsernameLookup(db *sql.DB) *SQLUsernameLookup { return &SQLUsernameLookup{db: db} }

// LookupUsernames runs a single batched query against `users` with
// case-insensitive matching. Unknown usernames are simply absent from the
// returned map.
func (l *SQLUsernameLookup) LookupUsernames(ctx context.Context, usernames []string) (map[string]string, error) {
	if len(usernames) == 0 {
		return map[string]string{}, nil
	}
	placeholders := make([]string, len(usernames))
	args := make([]interface{}, len(usernames))
	for i, u := range usernames {
		placeholders[i] = fmt.Sprintf("$%d", i+1)
		args[i] = strings.ToLower(u)
	}
	query := fmt.Sprintf("SELECT LOWER(username), id FROM users WHERE LOWER(username) IN (%s)", strings.Join(placeholders, ","))
	rows, err := l.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make(map[string]string, len(usernames))
	for rows.Next() {
		var lower, id string
		if err := rows.Scan(&lower, &id); err != nil {
			return nil, err
		}
		out[lower] = id
	}
	return out, rows.Err()
}
