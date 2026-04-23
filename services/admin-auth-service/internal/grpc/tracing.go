package grpc

import (
	"context"

	"go.opentelemetry.io/otel/trace"
)

// spanFromContext returns the current span bound to ctx.
// Wrapped here so handler.go doesn't need the OTel import directly.
func spanFromContext(ctx context.Context) trace.Span {
	return trace.SpanFromContext(ctx)
}
