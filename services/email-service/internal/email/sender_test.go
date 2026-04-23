package email

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"slate/libs/common-go/tracing"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
)

func TestRecordingSenderCapturesPayload(t *testing.T) {
	rs := &RecordingSender{}
	if err := rs.Send(context.Background(), Payload{To: "a@b.com", Subject: "s"}); err != nil {
		t.Fatalf("Send: %v", err)
	}
	if len(rs.Sent) != 1 {
		t.Fatalf("sent = %d, want 1", len(rs.Sent))
	}
	if rs.Sent[0].To != "a@b.com" {
		t.Errorf("To = %q, want a@b.com", rs.Sent[0].To)
	}
}

func TestHTTPSenderEmptyEndpointIsNoOp(t *testing.T) {
	s := NewHTTPSender("")
	if err := s.Send(context.Background(), Payload{To: "a@b.com"}); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestHTTPSenderInjectsTraceparent(t *testing.T) {
	tracing.EnsureDefaultPropagator()

	var sawTraceparent bool
	srv := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		if r.Header.Get("traceparent") != "" || r.Header.Get("Traceparent") != "" {
			sawTraceparent = true
		}
	}))
	defer srv.Close()

	carrier := propagation.MapCarrier{"traceparent": "00-0af7651916cd43dd8448eb211c80319c-b7ad6b7169203331-01"}
	ctx := otel.GetTextMapPropagator().Extract(context.Background(), carrier)

	s := NewHTTPSender(srv.URL)
	if err := s.Send(ctx, Payload{To: "x@y.com", Subject: "s", Body: "b"}); err != nil {
		t.Fatalf("Send: %v", err)
	}
	if !sawTraceparent {
		t.Errorf("traceparent header missing on outbound HTTP request")
	}
}

func TestHTTPSenderPropagatesRelayError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	s := NewHTTPSender(srv.URL)
	if err := s.Send(context.Background(), Payload{To: "x@y.com"}); err == nil {
		t.Errorf("expected error for 500 response")
	}
}
