package docker

import (
	"context"
	"testing"
)

func TestNoopFlusher_NeverErrors(t *testing.T) {
	f := NewRedisFlusher("", "", 0)
	if err := f.FlushTenant(context.Background(), "eastfield"); err != nil {
		t.Fatalf("noop flusher should not error: %v", err)
	}
	if err := f.Close(); err != nil {
		t.Fatalf("noop flusher close should not error: %v", err)
	}
}

func TestFlushTenant_RejectsEmptySlug(t *testing.T) {
	// Using a non-empty addr forces the real flusher (the go-redis client
	// is constructed but not yet connected — we only exercise the slug
	// validation which short-circuits before the network call).
	f := NewRedisFlusher("localhost:1", "", 0)
	defer f.Close()
	if err := f.FlushTenant(context.Background(), ""); err == nil {
		t.Fatal("expected error for empty slug")
	}
}
