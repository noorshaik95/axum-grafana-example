// Package tracing — propagation helpers.
//
// These helpers bridge gRPC metadata and Kafka headers with W3C Trace Context
// (traceparent + tracestate) and the Slate-specific X-Request-ID correlation
// header. Use the *FromContext helpers on outbound calls and the
// ContextFrom* helpers on inbound paths to keep a single trace stitched
// across services.
package tracing

import (
	"context"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc/metadata"
)

// Standard header/metadata keys. gRPC metadata keys are lowercased per the
// gRPC spec, HTTP/Kafka implementations should also use the lowercased form.
const (
	TraceparentHeader = "traceparent"
	TracestateHeader  = "tracestate"
	RequestIDHeader   = "x-request-id"
	TenantSlugHeader  = "x-tenant-slug"
)

// ----- gRPC metadata propagation ------------------------------------------------

// metadataCarrier adapts gRPC metadata.MD to the OTel TextMapCarrier interface.
type metadataCarrier struct {
	md metadata.MD
}

func (m metadataCarrier) Get(key string) string {
	vals := m.md.Get(key)
	if len(vals) == 0 {
		return ""
	}
	return vals[0]
}

func (m metadataCarrier) Set(key, value string) {
	m.md.Set(key, value)
}

func (m metadataCarrier) Keys() []string {
	keys := make([]string, 0, len(m.md))
	for k := range m.md {
		keys = append(keys, k)
	}
	return keys
}

// InjectTraceparent writes the current OTel span's traceparent + tracestate
// onto outgoing gRPC metadata. The returned context has the metadata attached;
// pass it to the gRPC client stub (it will merge automatically).
func InjectTraceparent(ctx context.Context) context.Context {
	md, ok := metadata.FromOutgoingContext(ctx)
	if !ok {
		md = metadata.New(nil)
	} else {
		md = md.Copy()
	}

	propagator := otel.GetTextMapPropagator()
	propagator.Inject(ctx, metadataCarrier{md: md})

	// Also forward x-request-id if set on the incoming call.
	if reqID := RequestIDFromContext(ctx); reqID != "" {
		md.Set(RequestIDHeader, reqID)
	}
	if tenant := TenantSlugFromContext(ctx); tenant != "" {
		md.Set(TenantSlugHeader, tenant)
	}

	return metadata.NewOutgoingContext(ctx, md)
}

// ExtractTraceparent reads a W3C traceparent from incoming gRPC metadata and
// returns a context bound to the extracted SpanContext. If no traceparent is
// present, the original context is returned unchanged.
func ExtractTraceparent(ctx context.Context) context.Context {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return ctx
	}
	propagator := otel.GetTextMapPropagator()
	ctx = propagator.Extract(ctx, metadataCarrier{md: md})

	if vals := md.Get(RequestIDHeader); len(vals) > 0 && vals[0] != "" {
		ctx = WithRequestID(ctx, vals[0])
	}
	if vals := md.Get(TenantSlugHeader); len(vals) > 0 && vals[0] != "" {
		ctx = WithTenantSlug(ctx, vals[0])
	}

	return ctx
}

// ----- Kafka header propagation ------------------------------------------------

// KafkaHeader is a minimal shape compatible with segmentio/kafka-go,
// confluent-kafka-go, and Sarama (all expose Key string + Value []byte).
type KafkaHeader struct {
	Key   string
	Value []byte
}

// kafkaCarrier implements TextMapCarrier over a slice of KafkaHeader.
type kafkaCarrier struct {
	headers *[]KafkaHeader
}

func (c kafkaCarrier) Get(key string) string {
	for _, h := range *c.headers {
		if h.Key == key {
			return string(h.Value)
		}
	}
	return ""
}

func (c kafkaCarrier) Set(key, value string) {
	// Replace existing header with the same key, else append.
	for i, h := range *c.headers {
		if h.Key == key {
			(*c.headers)[i].Value = []byte(value)
			return
		}
	}
	*c.headers = append(*c.headers, KafkaHeader{Key: key, Value: []byte(value)})
}

func (c kafkaCarrier) Keys() []string {
	out := make([]string, 0, len(*c.headers))
	for _, h := range *c.headers {
		out = append(out, h.Key)
	}
	return out
}

// KafkaHeadersFromContext converts the trace + correlation state on ctx into
// Kafka headers ready to attach to a produced message. The returned slice
// always contains traceparent (and x-request-id / x-tenant-slug when set).
func KafkaHeadersFromContext(ctx context.Context) []KafkaHeader {
	var headers []KafkaHeader
	carrier := kafkaCarrier{headers: &headers}
	otel.GetTextMapPropagator().Inject(ctx, carrier)

	if reqID := RequestIDFromContext(ctx); reqID != "" {
		carrier.Set(RequestIDHeader, reqID)
	}
	if tenant := TenantSlugFromContext(ctx); tenant != "" {
		carrier.Set(TenantSlugHeader, tenant)
	}
	return headers
}

// ContextFromKafkaHeaders restores trace + correlation state from a consumed
// Kafka message. The returned context is bound to the extracted SpanContext.
func ContextFromKafkaHeaders(ctx context.Context, headers []KafkaHeader) context.Context {
	carrier := kafkaCarrier{headers: &headers}
	ctx = otel.GetTextMapPropagator().Extract(ctx, carrier)

	if v := carrier.Get(RequestIDHeader); v != "" {
		ctx = WithRequestID(ctx, v)
	}
	if v := carrier.Get(TenantSlugHeader); v != "" {
		ctx = WithTenantSlug(ctx, v)
	}
	return ctx
}

// ----- Context keys: request-id + tenant-slug ---------------------------------

type requestIDKey struct{}
type tenantSlugKey struct{}

// WithRequestID stashes a request_id (from incoming HTTP header / gRPC metadata)
// on the context so downstream spans can tag it and outbound calls can forward it.
func WithRequestID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, requestIDKey{}, id)
}

// RequestIDFromContext returns the correlation id, or "" if none is attached.
func RequestIDFromContext(ctx context.Context) string {
	if v, ok := ctx.Value(requestIDKey{}).(string); ok {
		return v
	}
	return ""
}

// WithTenantSlug records the active tenant on the context.
func WithTenantSlug(ctx context.Context, slug string) context.Context {
	return context.WithValue(ctx, tenantSlugKey{}, slug)
}

// TenantSlugFromContext returns the tenant slug, or "" if none is attached.
func TenantSlugFromContext(ctx context.Context) string {
	if v, ok := ctx.Value(tenantSlugKey{}).(string); ok {
		return v
	}
	return ""
}

// TagSpanWithCorrelation writes `request_id` and `tenant.slug` onto the given
// span when those values are present on the context. Call this once per handler
// so traces in Tempo always carry the correlation fields without forcing each
// caller to re-apply them.
func TagSpanWithCorrelation(ctx context.Context, span trace.Span) {
	if span == nil {
		return
	}
	if v := RequestIDFromContext(ctx); v != "" {
		span.SetAttributes(attribute.String("request_id", v))
	}
	if v := TenantSlugFromContext(ctx); v != "" {
		span.SetAttributes(attribute.String("tenant.slug", v))
	}
}

// ensurePropagator — callers who haven't configured the global propagator can
// call this once on service startup to default to W3C Trace Context + Baggage.
func EnsureDefaultPropagator() {
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))
}
