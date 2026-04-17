package tracing

import (
	"context"
	"fmt"
	"log"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

// LoggingUnaryInterceptor logs incoming gRPC metadata and trace context for debugging
func LoggingUnaryInterceptor() grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req any,
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (any, error) {
		// Log to stderr for visibility
		log.Printf("[INTERCEPTOR] Called for method: %s", info.FullMethod)

		// Extract metadata from context
		md, ok := metadata.FromIncomingContext(ctx)
		if ok {
			log.Printf("[INTERCEPTOR] Metadata found with %d keys", len(md))

			// Log trace-related headers
			if traceparent := md.Get("traceparent"); len(traceparent) > 0 {
				log.Printf("[INTERCEPTOR] ✓ traceparent: %s", traceparent[0])
			} else {
				log.Printf("[INTERCEPTOR] ✗ NO traceparent header!")
			}

			if tracestate := md.Get("tracestate"); len(tracestate) > 0 {
				log.Printf("[INTERCEPTOR] ✓ tracestate: %s", tracestate[0])
			}

			// Log all metadata keys for debugging
			log.Printf("[INTERCEPTOR] All metadata keys:")
			for key, values := range md {
				log.Printf("[INTERCEPTOR]   - %s: %v", key, values)
			}
		} else {
			log.Printf("[INTERCEPTOR] ✗ NO metadata in context!")
		}

		// Check if there's already a span in the context (from otelgrpc)
		span := trace.SpanFromContext(ctx)
		if span.SpanContext().IsValid() {
			log.Printf("[INTERCEPTOR] ✓ Valid span found in context")
			log.Printf("[INTERCEPTOR]   TraceID: %s", span.SpanContext().TraceID().String())
			log.Printf("[INTERCEPTOR]   SpanID: %s", span.SpanContext().SpanID().String())
		} else {
			log.Printf("[INTERCEPTOR] ✗ NO valid span in context!")
		}

		// Create a new span for this operation
		tracer := otel.Tracer("user-auth-service")
		ctx, span = tracer.Start(ctx, fmt.Sprintf("grpc.%s", info.FullMethod))
		defer span.End()

		log.Printf("[INTERCEPTOR] Created new span: %s", span.SpanContext().SpanID().String())

		// Call the actual handler
		return handler(ctx, req)
	}
}
