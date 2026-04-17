// Package grpc provides reusable gRPC server interceptors for distributed tracing
// and structured logging across all Slate LMS Go services.
package grpc

import (
	"context"
	"unicode/utf8"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	semconv "go.opentelemetry.io/otel/semconv/v1.17.0"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/peer"
	grpcstatus "google.golang.org/grpc/status"

	"slate/libs/common-go/logging"
)

const maxMetaValueLen = 1000

// TracingUnaryInterceptor extracts W3C Trace Context from incoming gRPC metadata,
// creates a server-side span, and injects the span into the request context so
// downstream code (handlers, DB queries) can create child spans.
//
// serviceName should be the OTel service name (e.g. "email-service").
func TracingUnaryInterceptor(serviceName string) grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req any,
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (any, error) {
		md, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			md = metadata.New(nil)
		}

		// Extract W3C traceparent / tracestate from gRPC metadata.
		propagator := otel.GetTextMapPropagator()
		extractedCtx := propagator.Extract(ctx, metadataCarrier{md: md})

		tracer := otel.Tracer(serviceName)
		spanCtx, span := tracer.Start(
			extractedCtx,
			info.FullMethod,
			trace.WithSpanKind(trace.SpanKindServer),
		)
		defer span.End()

		span.SetAttributes(
			semconv.RPCSystemGRPC,
			semconv.RPCService(info.FullMethod),
			semconv.RPCMethod(info.FullMethod),
		)
		if p, ok := peer.FromContext(ctx); ok {
			span.SetAttributes(attribute.String("net.peer.ip", p.Addr.String()))
		}

		resp, err := handler(spanCtx, req)
		if err != nil {
			st, _ := grpcstatus.FromError(err)
			span.SetAttributes(attribute.String("rpc.grpc.status_code", st.Code().String()))
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
		} else {
			span.SetAttributes(attribute.String("rpc.grpc.status_code", "OK"))
			span.SetStatus(codes.Ok, "")
		}

		return resp, err
	}
}

// LoggingUnaryInterceptor logs gRPC method name, metadata keys, and trace context
// at debug level. Pass a *logging.Logger so the interceptor respects global log level.
func LoggingUnaryInterceptor(log *logging.Logger) grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req any,
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (any, error) {
		md, ok := metadata.FromIncomingContext(ctx)
		if ok {
			log.Debug().
				Str("method", info.FullMethod).
				Int("metadata_keys", len(md)).
				Msg("gRPC request received")

			if vals := md.Get("traceparent"); len(vals) > 0 {
				log.Debug().
					Str("traceparent", sanitize(vals[0])).
					Msg("W3C traceparent propagated")
			} else {
				log.Warn().
					Str("method", info.FullMethod).
					Msg("No traceparent in incoming gRPC metadata")
			}
		}

		span := trace.SpanFromContext(ctx)
		if span.SpanContext().IsValid() {
			log.Debug().
				Str("trace_id", span.SpanContext().TraceID().String()).
				Str("span_id", span.SpanContext().SpanID().String()).
				Msg("Active span found in gRPC context")
		}

		return handler(ctx, req)
	}
}

// metadataCarrier adapts gRPC metadata to otel TextMapCarrier for propagation.
type metadataCarrier struct{ md metadata.MD }

func (c metadataCarrier) Get(key string) string {
	v := c.md.Get(key)
	if len(v) == 0 {
		return ""
	}
	return v[0]
}
func (c metadataCarrier) Set(key, value string) { c.md.Set(key, value) }
func (c metadataCarrier) Keys() []string {
	keys := make([]string, 0, len(c.md))
	for k := range c.md {
		keys = append(keys, k)
	}
	return keys
}

func sanitize(s string) string {
	if !utf8.ValidString(s) {
		return "[invalid UTF-8]"
	}
	if len(s) > maxMetaValueLen {
		return s[:maxMetaValueLen] + "...[truncated]"
	}
	return s
}
