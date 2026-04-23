// Package kafka emits audit events produced by admin-auth-service.
//
// Topics:
//   - audit.admin_action            — every admin RPC (login, register, list, logout)
//   - audit.impersonation_started   — additional event on impersonation mint success
//
// Trace propagation is handled by copying traceparent + x-request-id out of
// the request context onto each message header via libs/common-go/tracing.
package kafka

import (
	"context"
	"errors"
	"time"

	kgo "github.com/segmentio/kafka-go"

	commontracing "slate/libs/common-go/tracing"
)

// Topic names.
const (
	TopicAdminAction           = "audit.admin_action"
	TopicImpersonationStarted  = "audit.impersonation_started"
)

// Producer writes audit payloads to Kafka.
type Producer struct {
	writer *kgo.Writer
}

// NewProducer creates a producer over the given brokers. A nil or empty
// broker list returns a no-op producer that returns ErrDisabled from
// every produce call — callers are expected to tolerate that.
func NewProducer(brokers []string) *Producer {
	if len(brokers) == 0 {
		return &Producer{}
	}
	return &Producer{
		writer: &kgo.Writer{
			Addr:         kgo.TCP(brokers...),
			Balancer:     &kgo.LeastBytes{},
			BatchTimeout: 100 * time.Millisecond,
			RequiredAcks: kgo.RequireOne,
			Async:        true,
		},
	}
}

// ErrDisabled is returned when the producer has no brokers configured.
var ErrDisabled = errors.New("kafka producer disabled (no brokers configured)")

// ProduceAdminAction emits a single message to audit.admin_action.
func (p *Producer) ProduceAdminAction(ctx context.Context, key string, payload []byte) error {
	return p.produce(ctx, TopicAdminAction, key, payload)
}

// ProduceImpersonationStarted emits to audit.impersonation_started.
func (p *Producer) ProduceImpersonationStarted(ctx context.Context, key string, payload []byte) error {
	return p.produce(ctx, TopicImpersonationStarted, key, payload)
}

func (p *Producer) produce(ctx context.Context, topic, key string, payload []byte) error {
	if p.writer == nil {
		return ErrDisabled
	}
	headers := commonHeadersToKafka(commontracing.KafkaHeadersFromContext(ctx))
	return p.writer.WriteMessages(ctx, kgo.Message{
		Topic:   topic,
		Key:     []byte(key),
		Value:   payload,
		Headers: headers,
	})
}

// Close flushes and shuts down the writer.
func (p *Producer) Close() error {
	if p.writer == nil {
		return nil
	}
	return p.writer.Close()
}

// commonHeadersToKafka converts tracing.KafkaHeader → kafka-go Header.
func commonHeadersToKafka(hs []commontracing.KafkaHeader) []kgo.Header {
	if len(hs) == 0 {
		return nil
	}
	out := make([]kgo.Header, 0, len(hs))
	for _, h := range hs {
		out = append(out, kgo.Header{Key: h.Key, Value: h.Value})
	}
	return out
}
