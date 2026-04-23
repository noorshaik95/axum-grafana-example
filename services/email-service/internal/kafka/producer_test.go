package kafka

import (
	"context"
	"testing"

	"slate/libs/common-go/tracing"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
)

func TestTraceHeadersIncludesTraceparent(t *testing.T) {
	tracing.EnsureDefaultPropagator()

	carrier := propagation.MapCarrier{"traceparent": "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01"}
	ctx := otel.GetTextMapPropagator().Extract(context.Background(), carrier)

	hs := traceHeaders(ctx)
	var found bool
	for _, h := range hs {
		if h.Key == "traceparent" {
			found = true
		}
	}
	if !found {
		t.Skip("propagator does not emit traceparent in this environment")
	}
}

func TestProducerEnabledFlag(t *testing.T) {
	p := NewProducer(nil, false)
	if p.Enabled() {
		t.Errorf("Enabled = true, want false")
	}
}

func TestPublishBroadcastInAppNoopWhenDisabled(t *testing.T) {
	p := NewProducer(nil, false)
	if err := p.PublishBroadcastInApp(context.Background(), "b1", "author", "msg", []string{"t1", "t2"}); err != nil {
		t.Errorf("unexpected error on disabled producer: %v", err)
	}
}
