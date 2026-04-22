package service

import (
	"context"
	"encoding/json"
)

// ThresholdBreachedEvent is the payload metrics-service publishes on
// `metrics.threshold_breached`. Shape matches the consumer contract in
// services/incident-service/internal/kafka/consumer.go:ThresholdBreachedEvent —
// deviating here would silently break the auto-P1 pipeline.
type ThresholdBreachedEvent struct {
	TenantID   *string `json:"tenant_id,omitempty"`
	Service    string  `json:"service"`
	MetricName string  `json:"metric_name,omitempty"`
	Threshold  float64 `json:"threshold,omitempty"`
	Observed   float64 `json:"observed,omitempty"`
	Message    string  `json:"message,omitempty"`
}

// ThresholdEmitter publishes `metrics.threshold_breached` events. It is kept
// tiny: callers supply the service name + observed/threshold values; the
// emitter owns Kafka serialisation and header injection (via the producer).
type ThresholdEmitter struct {
	producer EventProducer
}

// NewThresholdEmitter wires an emitter to the given Kafka producer.
func NewThresholdEmitter(p EventProducer) *ThresholdEmitter {
	return &ThresholdEmitter{producer: p}
}

// Emit publishes a threshold-breach event. Returns an error if serialisation
// or publish fails; the caller decides whether to retry.
func (e *ThresholdEmitter) Emit(ctx context.Context, ev ThresholdBreachedEvent) error {
	if e.producer == nil || ev.Service == "" {
		// Service is required per consumer contract; drop silently rather
		// than poison the topic with invalid messages.
		return nil
	}
	buf, err := json.Marshal(ev)
	if err != nil {
		return err
	}
	return e.producer.Publish(ctx, "metrics.threshold_breached", ev.Service, buf)
}

// EvaluateAndEmit checks `observed` against `threshold` and emits only when
// the threshold is strictly exceeded. Returns whether an emit occurred.
func (e *ThresholdEmitter) EvaluateAndEmit(ctx context.Context, ev ThresholdBreachedEvent) (bool, error) {
	if ev.Observed <= ev.Threshold {
		return false, nil
	}
	return true, e.Emit(ctx, ev)
}
