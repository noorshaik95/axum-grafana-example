package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/segmentio/kafka-go"
)

const (
	// Kafka topics
	TopicOnboardingJobs     = "onboarding.jobs"
	TopicOnboardingProgress = "onboarding.progress"
	TopicOnboardingAudit    = "onboarding.audit"
)

// Producer handles Kafka message production
type Producer struct {
	writers map[string]*kafka.Writer
}

// NewProducer creates a new Kafka producer
func NewProducer(brokers []string) *Producer {
	return &Producer{
		writers: map[string]*kafka.Writer{
			TopicOnboardingJobs:     createWriter(brokers, TopicOnboardingJobs),
			TopicOnboardingProgress: createWriter(brokers, TopicOnboardingProgress),
			TopicOnboardingAudit:    createWriter(brokers, TopicOnboardingAudit),
		},
	}
}

func createWriter(brokers []string, topic string) *kafka.Writer {
	return &kafka.Writer{
		Addr:         kafka.TCP(brokers...),
		Topic:        topic,
		Balancer:     &kafka.Hash{}, // Use hash balancing for key-based partitioning
		BatchSize:    100,
		BatchTimeout: 10 * time.Millisecond,
		Compression:  kafka.Snappy,
		RequiredAcks: kafka.RequireOne,
		Async:        false, // Synchronous writes for reliability
	}
}

// PublishTask publishes an onboarding task to Kafka
func (p *Producer) PublishTask(ctx context.Context, key string, message interface{}) error {
	return p.publishMessage(ctx, TopicOnboardingJobs, key, message)
}

// PublishProgress publishes a progress update to Kafka
func (p *Producer) PublishProgress(ctx context.Context, key string, message interface{}) error {
	return p.publishMessage(ctx, TopicOnboardingProgress, key, message)
}

// PublishAudit publishes an audit event to Kafka
func (p *Producer) PublishAudit(ctx context.Context, key string, message interface{}) error {
	return p.publishMessage(ctx, TopicOnboardingAudit, key, message)
}

func (p *Producer) publishMessage(ctx context.Context, topic, key string, message interface{}) error {
	writer, ok := p.writers[topic]
	if !ok {
		return fmt.Errorf("no writer found for topic: %s", topic)
	}

	// Serialize message to JSON
	value, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	// Create Kafka message
	msg := kafka.Message{
		Key:   []byte(key),
		Value: value,
		Time:  time.Now(),
	}

	// Publish message
	err = writer.WriteMessages(ctx, msg)
	if err != nil {
		log.Error().
			Err(err).
			Str("topic", topic).
			Str("key", key).
			Msg("Failed to publish message to Kafka")
		return fmt.Errorf("failed to write message to Kafka: %w", err)
	}

	log.Debug().
		Str("topic", topic).
		Str("key", key).
		Msg("Message published to Kafka")

	return nil
}

// Close closes all Kafka writers
func (p *Producer) Close() error {
	for topic, writer := range p.writers {
		if err := writer.Close(); err != nil {
			log.Error().
				Err(err).
				Str("topic", topic).
				Msg("Failed to close Kafka writer")
			return err
		}
	}
	log.Info().Msg("All Kafka producers closed")
	return nil
}
