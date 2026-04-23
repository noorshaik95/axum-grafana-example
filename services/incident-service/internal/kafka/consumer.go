package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	kafkago "github.com/segmentio/kafka-go"

	commontracing "slate/libs/common-go/tracing"
	"slate/services/incident-service/internal/cache"
	"slate/services/incident-service/internal/models"
	"slate/services/incident-service/internal/repository"
)

const TopicMetricsThresholdBreached = "metrics.threshold_breached"

// ThresholdBreachedEvent is the payload published by metrics-service when a
// service exceeds a configured threshold. Fields are optional except service.
type ThresholdBreachedEvent struct {
	TenantID   *string `json:"tenant_id,omitempty"`
	Service    string  `json:"service"`
	MetricName string  `json:"metric_name,omitempty"`
	Threshold  float64 `json:"threshold,omitempty"`
	Observed   float64 `json:"observed,omitempty"`
	Message    string  `json:"message,omitempty"`
}

// consumerRepo is the subset of the repository used by the Kafka consumer.
// Defined as an interface so tests can swap in a fake.
type consumerRepo interface {
	HasOpenIncidentFor(ctx context.Context, service string, tenantID *string) (bool, error)
	CreateIncident(ctx context.Context, p repository.CreateIncidentParams) (*models.Incident, error)
	AddEvent(ctx context.Context, incidentID string, actorID *string, eventType, content string) (*models.IncidentEvent, error)
	ListIncidents(ctx context.Context, f repository.ListFilter) ([]*models.Incident, error)
}

type incidentPublisher interface {
	PublishIncidentOpened(ctx context.Context, inc *models.Incident)
}

type activeCache interface {
	SetActive(ctx context.Context, incidents []*models.Incident) error
}

type Consumer struct {
	reader   *kafkago.Reader
	repo     consumerRepo
	producer incidentPublisher
	cache    activeCache
}

func NewConsumer(brokers []string, groupID string, repo *repository.Repository, producer *Producer, c *cache.Cache) *Consumer {
	reader := kafkago.NewReader(kafkago.ReaderConfig{
		Brokers:  brokers,
		Topic:    TopicMetricsThresholdBreached,
		GroupID:  groupID,
		MinBytes: 1,
		MaxBytes: 10e6,
	})
	cons := &Consumer{reader: reader, repo: repo, producer: producer}
	if c != nil {
		cons.cache = c
	}
	return cons
}

// NewConsumerForTest builds a Consumer with interface-level collaborators. Used
// by unit tests to drive HandleThresholdBreached without real Kafka/DB.
func NewConsumerForTest(repo consumerRepo, producer incidentPublisher, c activeCache) *Consumer {
	return &Consumer{repo: repo, producer: producer, cache: c}
}

func (c *Consumer) Start(ctx context.Context) {
	go c.consume(ctx)
}

func (c *Consumer) Close() error {
	if c.reader != nil {
		return c.reader.Close()
	}
	return nil
}

func (c *Consumer) consume(ctx context.Context) {
	log.Printf("Kafka consumer started for topic: %s", TopicMetricsThresholdBreached)
	for {
		msg, err := c.reader.ReadMessage(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			log.Printf("ERROR reading %s: %v", TopicMetricsThresholdBreached, err)
			continue
		}

		headers := make([]commontracing.KafkaHeader, 0, len(msg.Headers))
		for _, h := range msg.Headers {
			headers = append(headers, commontracing.KafkaHeader{Key: h.Key, Value: h.Value})
		}
		msgCtx := commontracing.ContextFromKafkaHeaders(ctx, headers)

		if err := c.HandleThresholdBreached(msgCtx, msg.Value); err != nil {
			log.Printf("ERROR handle %s: %v", TopicMetricsThresholdBreached, err)
		}
	}
}

// HandleThresholdBreached auto-opens a P1 incident for the service+tenant
// combination unless one is already open. Exported so tests can drive it
// directly without spinning up Kafka.
func (c *Consumer) HandleThresholdBreached(ctx context.Context, data []byte) error {
	var ev ThresholdBreachedEvent
	if err := json.Unmarshal(data, &ev); err != nil {
		return fmt.Errorf("unmarshal threshold_breached: %w", err)
	}
	if ev.Service == "" {
		return fmt.Errorf("service required on threshold_breached event")
	}

	exists, err := c.repo.HasOpenIncidentFor(ctx, ev.Service, ev.TenantID)
	if err != nil {
		return err
	}
	if exists {
		return nil
	}

	title := fmt.Sprintf("[auto] %s threshold breached", ev.Service)
	impact := ev.Message
	if impact == "" && ev.MetricName != "" {
		impact = fmt.Sprintf("%s observed=%.2f threshold=%.2f",
			ev.MetricName, ev.Observed, ev.Threshold)
	}

	svc := ev.Service
	inc, err := c.repo.CreateIncident(ctx, repository.CreateIncidentParams{
		TenantID: ev.TenantID,
		Service:  &svc,
		Title:    title,
		Priority: models.PriorityP1,
		Impact:   impact,
	})
	if err != nil {
		return fmt.Errorf("create auto incident: %w", err)
	}

	if _, err := c.repo.AddEvent(ctx, inc.ID, nil, models.EventAutoOpen,
		"auto-opened from metrics.threshold_breached"); err != nil {
		log.Printf("WARN add auto_open event: %v", err)
	}

	c.producer.PublishIncidentOpened(ctx, inc)

	// Refresh Redis cache from DB to avoid staleness from concurrent writes.
	if c.cache != nil {
		active, err := c.repo.ListIncidents(ctx, repository.ListFilter{
			Status: strPtr(models.StatusOpen),
			Limit:  500,
		})
		if err == nil {
			_ = c.cache.SetActive(ctx, active)
		}
	}
	return nil
}

func strPtr(s string) *string { return &s }
