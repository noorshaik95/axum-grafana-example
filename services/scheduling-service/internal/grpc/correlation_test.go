package grpc

import (
	"context"
	"testing"

	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"

	commontracing "slate/libs/common-go/tracing"
)

// When incoming gRPC metadata carries x-request-id + x-tenant-slug, the
// interceptor must pin both onto ctx via the common-go context helpers so
// TagSpanWithCorrelation can read them later.
func TestCorrelationInterceptor_ExtractsHeaders(t *testing.T) {
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(
		commontracing.RequestIDHeader, "req-123",
		commontracing.TenantSlugHeader, "acme",
	))

	var captured context.Context
	handler := func(c context.Context, _ any) (any, error) {
		captured = c
		return "ok", nil
	}

	if _, err := CorrelationUnaryInterceptor()(ctx, nil, &grpc.UnaryServerInfo{}, handler); err != nil {
		t.Fatalf("interceptor: %v", err)
	}
	if got := commontracing.RequestIDFromContext(captured); got != "req-123" {
		t.Errorf("request_id on ctx = %q, want %q", got, "req-123")
	}
	if got := commontracing.TenantSlugFromContext(captured); got != "acme" {
		t.Errorf("tenant.slug on ctx = %q, want %q", got, "acme")
	}
}

// No x-request-id on the way in → interceptor mints a new UUID and pins it.
// This matches CONTRACTS.md point 1 ("If X-Request-ID missing, generate one").
func TestCorrelationInterceptor_GeneratesRequestID(t *testing.T) {
	ctx := metadata.NewIncomingContext(context.Background(), metadata.New(nil))

	var captured context.Context
	handler := func(c context.Context, _ any) (any, error) {
		captured = c
		return nil, nil
	}

	if _, err := CorrelationUnaryInterceptor()(ctx, nil, &grpc.UnaryServerInfo{}, handler); err != nil {
		t.Fatalf("interceptor: %v", err)
	}
	got := commontracing.RequestIDFromContext(captured)
	if got == "" {
		t.Fatal("expected generated request_id, got empty")
	}
	// UUIDs are 36 chars.
	if len(got) != 36 {
		t.Errorf("generated request_id %q looks malformed (len %d)", got, len(got))
	}
}
