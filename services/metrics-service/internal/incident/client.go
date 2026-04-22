// Package incident provides a client abstraction for querying the
// incident-service. The metrics-service uses it to enrich the platform
// stats endpoint with active-incident counts.
//
// TODO(W12.4): replace StubClient with a real gRPC client once the
// incident.IncidentService stub in services/incident-service/api/proto is
// reachable from this module without introducing a go.mod cycle. Today we
// return zero active incidents so the platform stats endpoint degrades
// gracefully when the dependency is not wired.
package incident

import "context"

// Client reports the count of currently active incidents the platform should
// surface to admins.
type Client interface {
	ActiveCount(ctx context.Context) (int, error)
}

// StubClient always reports zero active incidents.
type StubClient struct{}

// NewStub returns a StubClient.
func NewStub() *StubClient { return &StubClient{} }

// ActiveCount returns 0.
func (*StubClient) ActiveCount(context.Context) (int, error) { return 0, nil }
