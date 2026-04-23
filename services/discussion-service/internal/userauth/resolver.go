// Package userauth exposes a narrow interface for resolving @username tokens
// to canonical user IDs via the user-auth-service.
//
// TODO(po-analyst): user-auth-service does not yet expose a ResolveUsername RPC.
// Today the remote stub returns no matches; unit tests inject a fake resolver.
// When the RPC lands, wire the real client in NewGRPCResolver.
package userauth

import "context"

// Resolver maps usernames (case-insensitive) within a tenant to user IDs.
// Unknown usernames MUST be omitted from the returned map — callers treat
// missing entries as "no such user" and simply skip them.
type Resolver interface {
	ResolveUsernames(ctx context.Context, tenantID string, usernames []string) (map[string]string, error)
}

// NoopResolver returns an empty map. It is the default production stub until
// user-auth-service ships a ResolveUsername RPC (see package doc).
type NoopResolver struct{}

func (NoopResolver) ResolveUsernames(_ context.Context, _ string, _ []string) (map[string]string, error) {
	return map[string]string{}, nil
}

// StaticResolver is a test helper: map of lowercased username → user_id.
type StaticResolver struct{ Users map[string]string }

func (s StaticResolver) ResolveUsernames(_ context.Context, _ string, usernames []string) (map[string]string, error) {
	out := make(map[string]string, len(usernames))
	for _, u := range usernames {
		if id, ok := s.Users[lower(u)]; ok {
			out[u] = id
		}
	}
	return out, nil
}

func lower(s string) string {
	b := make([]byte, len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c >= 'A' && c <= 'Z' {
			c += 'a' - 'A'
		}
		b[i] = c
	}
	return string(b)
}
