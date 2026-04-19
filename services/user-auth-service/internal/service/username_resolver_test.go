package service

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"
)

// fakeLookup records calls and returns the staged map + err.
type fakeLookup struct {
	mu      sync.Mutex
	calls   int
	staged  map[string]string
	err     error
	seenArg [][]string
}

func (f *fakeLookup) LookupUsernames(_ context.Context, usernames []string) (map[string]string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls++
	cpy := make([]string, len(usernames))
	copy(cpy, usernames)
	f.seenArg = append(f.seenArg, cpy)
	if f.err != nil {
		return nil, f.err
	}
	out := make(map[string]string)
	for _, u := range usernames {
		if id, ok := f.staged[u]; ok {
			out[u] = id
		}
	}
	return out, nil
}

func TestResolve_HappyPath(t *testing.T) {
	f := &fakeLookup{staged: map[string]string{"alice": "u-1", "bob": "u-2"}}
	r := NewUsernameResolver(f, time.Second)
	got, err := r.Resolve(context.Background(), []string{"alice", "BOB"})
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if got["alice"] != "u-1" || got["bob"] != "u-2" {
		t.Fatalf("got %+v", got)
	}
	if f.calls != 1 {
		t.Fatalf("want 1 db call, got %d", f.calls)
	}
}

func TestResolve_PartialMatch(t *testing.T) {
	f := &fakeLookup{staged: map[string]string{"alice": "u-1"}}
	r := NewUsernameResolver(f, time.Second)
	got, _ := r.Resolve(context.Background(), []string{"alice", "mallory"})
	if len(got) != 1 || got["alice"] != "u-1" {
		t.Fatalf("expected only alice, got %+v", got)
	}
	if _, ok := got["mallory"]; ok {
		t.Fatal("unknown username must not appear in result")
	}
}

func TestResolve_UnknownOnly(t *testing.T) {
	f := &fakeLookup{staged: map[string]string{}}
	r := NewUsernameResolver(f, time.Second)
	got, err := r.Resolve(context.Background(), []string{"nobody"})
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("want empty map, got %+v", got)
	}
}

func TestResolve_EmptyInputNoDBCall(t *testing.T) {
	f := &fakeLookup{}
	r := NewUsernameResolver(f, time.Second)
	if _, err := r.Resolve(context.Background(), nil); err != nil {
		t.Fatalf("nil input: %v", err)
	}
	if f.calls != 0 {
		t.Fatal("empty input must skip DB call")
	}
}

func TestResolve_CacheHitSkipsDB(t *testing.T) {
	f := &fakeLookup{staged: map[string]string{"alice": "u-1"}}
	r := NewUsernameResolver(f, time.Minute)
	if _, err := r.Resolve(context.Background(), []string{"alice"}); err != nil {
		t.Fatalf("first: %v", err)
	}
	if _, err := r.Resolve(context.Background(), []string{"alice"}); err != nil {
		t.Fatalf("second: %v", err)
	}
	if f.calls != 1 {
		t.Fatalf("want 1 db call (cache hit), got %d", f.calls)
	}
}

func TestResolve_CacheExpiry(t *testing.T) {
	f := &fakeLookup{staged: map[string]string{"alice": "u-1"}}
	r := NewUsernameResolver(f, 100*time.Millisecond)
	fixed := time.Now()
	r.SetClock(func() time.Time { return fixed })
	_, _ = r.Resolve(context.Background(), []string{"alice"})
	fixed = fixed.Add(time.Second)
	_, _ = r.Resolve(context.Background(), []string{"alice"})
	if f.calls != 2 {
		t.Fatalf("want cache expiry to trigger 2nd call, got %d", f.calls)
	}
}

func TestResolve_LookupError(t *testing.T) {
	f := &fakeLookup{err: errors.New("db down")}
	r := NewUsernameResolver(f, time.Second)
	if _, err := r.Resolve(context.Background(), []string{"x"}); err == nil {
		t.Fatal("want lookup error to propagate")
	}
}

func TestInvalidate_DropsCache(t *testing.T) {
	f := &fakeLookup{staged: map[string]string{"alice": "u-1"}}
	r := NewUsernameResolver(f, time.Hour)
	_, _ = r.Resolve(context.Background(), []string{"alice"})
	r.Invalidate("Alice")
	_, _ = r.Resolve(context.Background(), []string{"alice"})
	if f.calls != 2 {
		t.Fatalf("want invalidate to trigger 2nd lookup, got %d", f.calls)
	}
}
