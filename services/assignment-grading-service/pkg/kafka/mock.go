package kafka

import (
	"context"
	"sync"
)

// MockPublisher captures published events in memory. Unit tests use this to
// assert that business logic emits the expected Kafka events without spinning
// up a broker. Safe for concurrent use.
type MockPublisher struct {
	mu     sync.Mutex
	Events []Event
}

// NewMockPublisher constructs an empty in-memory publisher.
func NewMockPublisher() *MockPublisher {
	return &MockPublisher{}
}

// PublishEvent records the event and never errors.
func (m *MockPublisher) PublishEvent(_ context.Context, event Event) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Events = append(m.Events, event)
	return nil
}

// Close is a no-op for the mock.
func (m *MockPublisher) Close() error { return nil }

// EventsOfType returns every captured event with the given type.
func (m *MockPublisher) EventsOfType(t string) []Event {
	m.mu.Lock()
	defer m.mu.Unlock()
	var out []Event
	for _, e := range m.Events {
		if e.Type == t {
			out = append(out, e)
		}
	}
	return out
}

// Reset clears captured events (useful between sub-tests).
func (m *MockPublisher) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Events = nil
}
