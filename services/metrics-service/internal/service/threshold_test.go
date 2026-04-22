package service

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
)

func TestThresholdEmitter_EmitShape(t *testing.T) {
	p := NewStubProducer()
	e := NewThresholdEmitter(p)

	tenant := "acme"
	if err := e.Emit(context.Background(), ThresholdBreachedEvent{
		TenantID:   &tenant,
		Service:    "course-service",
		MetricName: "p99_latency_ms",
		Threshold:  1000,
		Observed:   2500,
		Message:    "latency spike",
	}); err != nil {
		t.Fatal(err)
	}

	if p.Count("metrics.threshold_breached") != 1 {
		t.Fatalf("expected 1 message, got %d", p.Count("metrics.threshold_breached"))
	}

	var got ThresholdBreachedEvent
	if err := json.Unmarshal(p.Messages[0].Payload, &got); err != nil {
		t.Fatal(err)
	}
	if got.Service != "course-service" || got.Observed != 2500 || *got.TenantID != "acme" {
		t.Fatalf("unexpected payload: %+v", got)
	}
	if p.Messages[0].Key != "course-service" {
		t.Fatalf("expected key=service for partitioning, got %s", p.Messages[0].Key)
	}
}

func TestThresholdEmitter_SkipsMissingService(t *testing.T) {
	p := NewStubProducer()
	e := NewThresholdEmitter(p)
	if err := e.Emit(context.Background(), ThresholdBreachedEvent{Service: ""}); err != nil {
		t.Fatal(err)
	}
	if p.Count("metrics.threshold_breached") != 0 {
		t.Fatal("expected no emit when service missing")
	}
}

func TestThresholdEmitter_EvaluateAndEmit(t *testing.T) {
	p := NewStubProducer()
	e := NewThresholdEmitter(p)

	// Under threshold — no emit.
	emitted, err := e.EvaluateAndEmit(context.Background(), ThresholdBreachedEvent{
		Service: "svc", Threshold: 100, Observed: 50,
	})
	if err != nil || emitted {
		t.Fatalf("did not expect emit, got emitted=%v err=%v", emitted, err)
	}

	// Over threshold — emit.
	emitted, err = e.EvaluateAndEmit(context.Background(), ThresholdBreachedEvent{
		Service: "svc", Threshold: 100, Observed: 200,
	})
	if err != nil || !emitted {
		t.Fatalf("expected emit, got emitted=%v err=%v", emitted, err)
	}
	if p.Count("metrics.threshold_breached") != 1 {
		t.Fatalf("expected 1 message, got %d", p.Count("metrics.threshold_breached"))
	}

	// Equal to threshold — no emit (strictly greater).
	emitted, _ = e.EvaluateAndEmit(context.Background(), ThresholdBreachedEvent{
		Service: "svc", Threshold: 100, Observed: 100,
	})
	if emitted {
		t.Fatal("expected no emit when observed == threshold")
	}
}

type breakingProducer struct{}

func (breakingProducer) Publish(context.Context, string, string, []byte) error {
	return errors.New("broker down")
}

func TestThresholdEmitter_PublishError(t *testing.T) {
	e := NewThresholdEmitter(breakingProducer{})
	err := e.Emit(context.Background(), ThresholdBreachedEvent{Service: "svc"})
	if err == nil {
		t.Fatal("expected error to propagate")
	}
}
