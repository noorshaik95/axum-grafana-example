// Package email provides the outbound email abstraction used by the broadcast
// handler and Kafka consumers. Tests substitute a recording sender; production
// wires an HTTPSender that calls the platform's SMTP/relay endpoint.
package email

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
)

// Payload is the outbound email body delivered to the relay.
type Payload struct {
	To       string            `json:"to"`
	Subject  string            `json:"subject"`
	Body     string            `json:"body"`
	Template string            `json:"template,omitempty"`
	TenantID string            `json:"tenant_id,omitempty"`
	Data     map[string]string `json:"data,omitempty"`
}

// Sender sends a single email. Implementations must inject W3C traceparent
// onto any outbound HTTP calls so requests stay stitched to the caller's span.
type Sender interface {
	Send(ctx context.Context, p Payload) error
}

// HTTPSender posts payloads to the configured relay endpoint. traceparent is
// injected via the global OTel TextMapPropagator on every request.
type HTTPSender struct {
	Endpoint string
	Client   *http.Client
}

// NewHTTPSender constructs an HTTPSender with a default 5-second client.
func NewHTTPSender(endpoint string) *HTTPSender {
	return &HTTPSender{
		Endpoint: endpoint,
		Client:   &http.Client{Timeout: 5 * time.Second},
	}
}

// Send delivers p and injects traceparent onto the HTTP request.
func (s *HTTPSender) Send(ctx context.Context, p Payload) error {
	if s.Endpoint == "" {
		return nil // no relay configured — act as a no-op in dev
	}
	body, err := json.Marshal(p)
	if err != nil {
		return fmt.Errorf("marshal email payload: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.Endpoint, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	// Inject traceparent + tracestate onto outbound HTTP headers.
	otel.GetTextMapPropagator().Inject(ctx, propagation.HeaderCarrier(req.Header))

	resp, err := s.Client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return fmt.Errorf("email relay returned status %d", resp.StatusCode)
	}
	return nil
}

// RecordingSender captures sends for assertions in tests.
type RecordingSender struct {
	Sent []Payload
}

// Send records p and returns nil.
func (s *RecordingSender) Send(_ context.Context, p Payload) error {
	s.Sent = append(s.Sent, p)
	return nil
}
