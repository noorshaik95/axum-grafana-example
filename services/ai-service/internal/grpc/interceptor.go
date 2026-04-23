package grpc

import (
	"context"

	"google.golang.org/grpc"

	commontracing "slate/libs/common-go/tracing"
)

// CorrelationInterceptor extracts W3C traceparent + x-request-id +
// x-tenant-slug from inbound gRPC metadata and stashes them on the context
// so the span tagging in claude.Invoke has the correlation fields to stamp.
//
// Without this, the existing common-go TracingUnaryInterceptor still
// extracts traceparent, but not the Slate-specific correlation headers the
// cost ledger needs.
func CorrelationInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, _ *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		ctx = commontracing.ExtractTraceparent(ctx)
		return handler(ctx, req)
	}
}
