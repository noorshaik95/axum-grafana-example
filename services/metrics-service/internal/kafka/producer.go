package kafka

import (
	"context"
	"sync"

	"slate/libs/common-go/tracing"

	"github.com/segmentio/kafka-go"
)

// Producer publishes messages to Kafka topics. It lazily opens a writer per
// topic and closes all of them on Close. Satisfies service.EventProducer.
type Producer struct {
	brokers []string
	mu      sync.Mutex
	writers map[string]*kafka.Writer
}

// NewProducer returns a Producer that will connect to the given brokers.
func NewProducer(brokers []string) *Producer {
	return &Producer{
		brokers: brokers,
		writers: make(map[string]*kafka.Writer),
	}
}

// Publish writes a single message to the given topic. Trace context is
// injected into message headers per plan/CONTRACTS.md `trace.propagation` so
// the consumer span links back to this caller.
func (p *Producer) Publish(ctx context.Context, topic, key string, payload []byte) error {
	w := p.writer(topic)
	return w.WriteMessages(ctx, kafka.Message{
		Key:     []byte(key),
		Value:   payload,
		Headers: traceHeaders(ctx),
	})
}

func traceHeaders(ctx context.Context) []kafka.Header {
	hs := tracing.KafkaHeadersFromContext(ctx)
	out := make([]kafka.Header, len(hs))
	for i, h := range hs {
		out[i] = kafka.Header{Key: h.Key, Value: h.Value}
	}
	return out
}

func (p *Producer) writer(topic string) *kafka.Writer {
	p.mu.Lock()
	defer p.mu.Unlock()
	if w, ok := p.writers[topic]; ok {
		return w
	}
	w := &kafka.Writer{
		Addr:     kafka.TCP(p.brokers...),
		Topic:    topic,
		Balancer: &kafka.Hash{},
	}
	p.writers[topic] = w
	return w
}

// Close closes all underlying writers.
func (p *Producer) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	var firstErr error
	for _, w := range p.writers {
		if err := w.Close(); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}
