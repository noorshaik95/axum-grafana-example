package kafka

import (
	"context"
	"encoding/json"
	"log"
	"time"

	kafkago "github.com/segmentio/kafka-go"

	commontracing "slate/libs/common-go/tracing"
	"slate/services/incident-service/internal/models"
)

const (
	TopicIncidentOpened   = "incident.opened"
	TopicIncidentResolved = "incident.resolved"
)

type Producer struct {
	writer  *kafkago.Writer
	enabled bool
}

func NewProducer(brokers []string, enabled bool) *Producer {
	if !enabled {
		return &Producer{enabled: false}
	}
	return &Producer{
		writer: &kafkago.Writer{
			Addr:         kafkago.TCP(brokers...),
			Balancer:     &kafkago.LeastBytes{},
			MaxAttempts:  3,
			BatchSize:    1,
			BatchTimeout: 10 * time.Millisecond,
			WriteTimeout: 10 * time.Second,
			RequiredAcks: kafkago.RequireOne,
		},
		enabled: true,
	}
}

func (p *Producer) Close() error {
	if p.writer != nil {
		return p.writer.Close()
	}
	return nil
}

type incidentOpenedEvent struct {
	Type      string  `json:"type"`
	ID        string  `json:"id"`
	TenantID  *string `json:"tenant_id,omitempty"`
	Service   *string `json:"service,omitempty"`
	Priority  string  `json:"priority"`
	Title     string  `json:"title"`
	Impact    string  `json:"impact"`
	OpenedAt  int64   `json:"opened_at_unix_ms"`
	Timestamp string  `json:"timestamp"`
}

type incidentResolvedEvent struct {
	Type       string  `json:"type"`
	ID         string  `json:"id"`
	TenantID   *string `json:"tenant_id,omitempty"`
	Service    *string `json:"service,omitempty"`
	Priority   string  `json:"priority"`
	Title      string  `json:"title"`
	OpenedAt   int64   `json:"opened_at_unix_ms"`
	ResolvedAt int64   `json:"resolved_at_unix_ms"`
	Timestamp  string  `json:"timestamp"`
}

func (p *Producer) PublishIncidentOpened(ctx context.Context, inc *models.Incident) {
	if !p.enabled || inc == nil {
		return
	}
	ev := incidentOpenedEvent{
		Type:      "incident.opened",
		ID:        inc.ID,
		TenantID:  inc.TenantID,
		Service:   inc.Service,
		Priority:  inc.Priority,
		Title:     inc.Title,
		Impact:    inc.Description,
		OpenedAt:  inc.CreatedAt.UnixMilli(),
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}
	p.publish(ctx, TopicIncidentOpened, inc.ID, ev)
}

func (p *Producer) PublishIncidentResolved(ctx context.Context, inc *models.Incident) {
	if !p.enabled || inc == nil || inc.ResolvedAt == nil {
		return
	}
	ev := incidentResolvedEvent{
		Type:       "incident.resolved",
		ID:         inc.ID,
		TenantID:   inc.TenantID,
		Service:    inc.Service,
		Priority:   inc.Priority,
		Title:      inc.Title,
		OpenedAt:   inc.CreatedAt.UnixMilli(),
		ResolvedAt: inc.ResolvedAt.UnixMilli(),
		Timestamp:  time.Now().UTC().Format(time.RFC3339),
	}
	p.publish(ctx, TopicIncidentResolved, inc.ID, ev)
}

func (p *Producer) publish(ctx context.Context, topic, key string, payload any) {
	data, err := json.Marshal(payload)
	if err != nil {
		log.Printf("ERROR marshal %s: %v", topic, err)
		return
	}

	msg := kafkago.Message{
		Topic: topic,
		Key:   []byte(key),
		Value: data,
	}
	for _, h := range commontracing.KafkaHeadersFromContext(ctx) {
		msg.Headers = append(msg.Headers, kafkago.Header{Key: h.Key, Value: h.Value})
	}

	if err := p.writer.WriteMessages(ctx, msg); err != nil {
		log.Printf("ERROR publish %s: %v", topic, err)
	}
}
