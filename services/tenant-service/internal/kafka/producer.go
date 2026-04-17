package kafka

import (
	"context"
	"encoding/json"

	"github.com/rs/zerolog/log"
	"github.com/segmentio/kafka-go"
)

// Producer writes tenant lifecycle events to Kafka topics.
type Producer struct {
	writer *kafka.Writer
}

// NewProducer creates a new Kafka producer.
func NewProducer(brokers []string) *Producer {
	return &Producer{
		writer: &kafka.Writer{
			Addr:     kafka.TCP(brokers...),
			Balancer: &kafka.LeastBytes{},
		},
	}
}

func (p *Producer) produce(ctx context.Context, topic string, key string, value interface{}) {
	data, err := json.Marshal(value)
	if err != nil {
		log.Error().Err(err).Str("topic", topic).Msg("failed to marshal event")
		return
	}

	if err := p.writer.WriteMessages(ctx, kafka.Message{
		Topic: topic,
		Key:   []byte(key),
		Value: data,
	}); err != nil {
		log.Error().Err(err).Str("topic", topic).Msg("failed to produce event")
	}
}

// ProduceTenantProvisioned emits a tenant.provisioned event.
func (p *Producer) ProduceTenantProvisioned(ctx context.Context, event *TenantProvisionedEvent) {
	p.produce(ctx, "tenant.provisioned", event.TenantID, event)
}

// ProduceProvisionFailed emits a tenant.provision_failed event.
func (p *Producer) ProduceProvisionFailed(ctx context.Context, onboardingID, errMsg string) {
	p.produce(ctx, "tenant.provision_failed", onboardingID, &TenantProvisionFailedEvent{
		OnboardingID: onboardingID,
		Error:        errMsg,
	})
}

// ProduceTenantDisabled emits a tenant.disabled event.
func (p *Producer) ProduceTenantDisabled(ctx context.Context, tenantID string) {
	p.produce(ctx, "tenant.disabled", tenantID, &TenantLifecycleEvent{TenantID: tenantID})
}

// ProduceTenantEnabled emits a tenant.enabled event.
func (p *Producer) ProduceTenantEnabled(ctx context.Context, tenantID string) {
	p.produce(ctx, "tenant.enabled", tenantID, &TenantLifecycleEvent{TenantID: tenantID})
}

// ProduceTenantPlanUpdated emits a tenant.plan_updated event.
func (p *Producer) ProduceTenantPlanUpdated(ctx context.Context, tenantID string, plan interface{}) {
	p.produce(ctx, "tenant.plan_updated", tenantID, map[string]interface{}{
		"tenantId": tenantID,
		"plan":     plan,
	})
}

// Close shuts down the producer.
func (p *Producer) Close() error {
	return p.writer.Close()
}
