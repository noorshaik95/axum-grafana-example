// Package kafka publishes discussion events for downstream consumers (e.g.,
// email-service listens on discussion.mention to send notifications).
package kafka

import (
	"context"
	"encoding/json"
	"time"

	commontracing "slate/libs/common-go/tracing"

	kafkago "github.com/segmentio/kafka-go"
)

// Producer is a thin wrapper around kafka-go.Writer. When disabled, it no-ops.
type Producer struct {
	writer *kafkago.Writer
	mock   MockSink
}

// MockSink captures events in memory for tests.
type MockSink interface {
	Record(topic string, key []byte, value []byte)
}

func NewProducer(brokers []string, enabled bool) *Producer {
	if !enabled {
		return &Producer{}
	}
	w := &kafkago.Writer{
		Addr:         kafkago.TCP(brokers...),
		Balancer:     &kafkago.LeastBytes{},
		MaxAttempts:  3,
		BatchSize:    1,
		BatchTimeout: 10 * time.Millisecond,
		WriteTimeout: 10 * time.Second,
		RequiredAcks: kafkago.RequireOne,
	}
	return &Producer{writer: w}
}

// NewTestProducer captures events via the given sink instead of Kafka.
func NewTestProducer(sink MockSink) *Producer {
	return &Producer{mock: sink}
}

// MentionEvent is the payload of `discussion.mention`. Field names match the
// consumer contract ratified in plan/CONTRACTS.md §email.MessagingService
// (W15 email-expert). mentioned_by / mentioned_user carry user_ids (the
// producer does not hold a stable username at mention-materialization time).
// mentioned_email is populated when cheaply resolvable; otherwise empty, and
// email-service resolves via user-auth GetUser. post_id and course_id are
// additive internal fields preserved for metrics-service's engagement
// counters; email-service ignores them safely.
type MentionEvent struct {
	Type           string    `json:"type"`
	TenantID       string    `json:"tenant_id"`
	ThreadID       string    `json:"thread_id"`
	ThreadTitle    string    `json:"thread_title"`
	MentionedBy    string    `json:"mentioned_by"`
	MentionedUser  string    `json:"mentioned_user"`
	MentionedEmail string    `json:"mentioned_email"`
	Excerpt        string    `json:"excerpt"`
	PostID         string    `json:"post_id"`
	CourseID       string    `json:"course_id"`
	Timestamp      time.Time `json:"timestamp"`
}

// PublishMention emits `discussion.mention`. Failures are swallowed so a broken
// broker does not break the write path; the mention row is the source of truth.
//
// Kafka headers carry `traceparent` + `x-request-id` + `x-tenant-slug` so the
// consumer (email-service) can resume the caller's trace — satisfies the
// plan/CONTRACTS.md `trace.propagation` acceptance gate.
func (p *Producer) PublishMention(ctx context.Context, evt MentionEvent) error {
	evt.Type = "discussion.mention"
	evt.Timestamp = time.Now().UTC()
	data, err := json.Marshal(evt)
	if err != nil {
		return err
	}
	key := []byte(evt.MentionedUser)
	if p.mock != nil {
		p.mock.Record("discussion.mention", key, data)
		return nil
	}
	if p.writer == nil {
		return nil
	}

	propHeaders := commontracing.KafkaHeadersFromContext(ctx)
	hdrs := make([]kafkago.Header, 0, len(propHeaders))
	for _, h := range propHeaders {
		hdrs = append(hdrs, kafkago.Header{Key: h.Key, Value: h.Value})
	}
	return p.writer.WriteMessages(ctx, kafkago.Message{
		Topic:   "discussion.mention",
		Key:     key,
		Value:   data,
		Headers: hdrs,
	})
}

func (p *Producer) Close() error {
	if p.writer == nil {
		return nil
	}
	return p.writer.Close()
}
