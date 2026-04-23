package tracing

import (
	"context"
	"testing"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	"google.golang.org/grpc/metadata"
)

func withW3C(t *testing.T) {
	t.Helper()
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))
}

func TestRequestIDAndTenantRoundTripGRPC(t *testing.T) {
	withW3C(t)

	// Outbound: attach correlation + traceparent onto gRPC metadata.
	ctx := WithRequestID(context.Background(), "req-abc")
	ctx = WithTenantSlug(ctx, "eastfield")
	ctx = InjectTraceparent(ctx)

	md, ok := metadata.FromOutgoingContext(ctx)
	if !ok {
		t.Fatalf("expected outgoing metadata to be set")
	}
	if got := md.Get(RequestIDHeader); len(got) == 0 || got[0] != "req-abc" {
		t.Fatalf("x-request-id missing, got %v", got)
	}
	if got := md.Get(TenantSlugHeader); len(got) == 0 || got[0] != "eastfield" {
		t.Fatalf("x-tenant-slug missing, got %v", got)
	}

	// Simulate the receiving service: metadata arrives on an incoming context.
	incoming := metadata.NewIncomingContext(context.Background(), md)
	restored := ExtractTraceparent(incoming)

	if got := RequestIDFromContext(restored); got != "req-abc" {
		t.Fatalf("expected request_id=req-abc, got %q", got)
	}
	if got := TenantSlugFromContext(restored); got != "eastfield" {
		t.Fatalf("expected tenant slug eastfield, got %q", got)
	}
}

func TestKafkaHeadersRoundTrip(t *testing.T) {
	withW3C(t)

	ctx := WithRequestID(context.Background(), "req-42")
	ctx = WithTenantSlug(ctx, "stanford")

	headers := KafkaHeadersFromContext(ctx)
	if len(headers) == 0 {
		t.Fatalf("expected non-empty kafka headers")
	}

	var sawReqID, sawTenant bool
	for _, h := range headers {
		if h.Key == RequestIDHeader && string(h.Value) == "req-42" {
			sawReqID = true
		}
		if h.Key == TenantSlugHeader && string(h.Value) == "stanford" {
			sawTenant = true
		}
	}
	if !sawReqID || !sawTenant {
		t.Fatalf("missing correlation headers in kafka payload: %+v", headers)
	}

	restored := ContextFromKafkaHeaders(context.Background(), headers)
	if got := RequestIDFromContext(restored); got != "req-42" {
		t.Fatalf("expected request_id=req-42, got %q", got)
	}
	if got := TenantSlugFromContext(restored); got != "stanford" {
		t.Fatalf("expected tenant slug stanford, got %q", got)
	}
}

func TestEnsureDefaultPropagator(t *testing.T) {
	EnsureDefaultPropagator()
	// If the call above panics, this test fails automatically.
	p := otel.GetTextMapPropagator()
	if p == nil {
		t.Fatalf("expected a propagator to be configured")
	}
}
