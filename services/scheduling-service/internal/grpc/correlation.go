package grpc

import (
	"context"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"

	commontracing "slate/libs/common-go/tracing"
)

// CorrelationUnaryInterceptor satisfies CONTRACTS.md "trace.propagation" point 1:
// pull `x-request-id` + `x-tenant-slug` off incoming gRPC metadata onto the context
// (so TagSpanWithCorrelation can stamp them on the span), and mint a fresh
// request_id when the caller didn't send one. The id is echoed on the response
// via grpc.SetHeader so the gateway can relay it back to the browser.
//
// Register this BEFORE common-go's TracingUnaryInterceptor so the span created
// by that interceptor already sees the correlation values on ctx.
func CorrelationUnaryInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, _ *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		md, _ := metadata.FromIncomingContext(ctx)

		reqID := firstMD(md, commontracing.RequestIDHeader)
		if reqID == "" {
			reqID = uuid.NewString()
		}
		ctx = commontracing.WithRequestID(ctx, reqID)

		if tenant := firstMD(md, commontracing.TenantSlugHeader); tenant != "" {
			ctx = commontracing.WithTenantSlug(ctx, tenant)
		}

		// Echo on response headers so the gateway / browser can correlate.
		_ = grpc.SetHeader(ctx, metadata.Pairs(commontracing.RequestIDHeader, reqID))

		// Stamp right away when a span already exists (e.g. otelgrpc StatsHandler
		// created one). The per-RPC handler will call TagSpanWithCorrelation again
		// after any domain tags are added — both are safe and idempotent.
		if span := trace.SpanFromContext(ctx); span.IsRecording() {
			commontracing.TagSpanWithCorrelation(ctx, span)
		}

		return handler(ctx, req)
	}
}

func firstMD(md metadata.MD, key string) string {
	if md == nil {
		return ""
	}
	vals := md.Get(key)
	if len(vals) == 0 {
		return ""
	}
	return vals[0]
}
