package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/segmentio/kafka-go"
)

// Consumer handles Kafka message consumption
type Consumer struct {
	reader *kafka.Reader
}

// MessageHandler is a function that processes Kafka messages
type MessageHandler func(ctx context.Context, key, value []byte) error

// NewConsumer creates a new Kafka consumer
func NewConsumer(brokers []string, groupID, topic string) *Consumer {
	return &Consumer{
		reader: kafka.NewReader(kafka.ReaderConfig{
			Brokers:        brokers,
			GroupID:        groupID,
			Topic:          topic,
			MinBytes:       1,           // 1 byte
			MaxBytes:       10e6,        // 10MB
			CommitInterval: time.Second, // Auto-commit every second
			StartOffset:    kafka.LastOffset,
			MaxWait:        500 * time.Millisecond,
			Logger:         kafka.LoggerFunc(log.Printf),
			ErrorLogger:    kafka.LoggerFunc(log.Printf),
		}),
	}
}

// Consume starts consuming messages and processing them with the handler
func (c *Consumer) Consume(ctx context.Context, handler MessageHandler) error {
	log.Info().
		Str("topic", c.reader.Config().Topic).
		Str("group_id", c.reader.Config().GroupID).
		Msg("Starting Kafka consumer")

	for {
		select {
		case <-ctx.Done():
			log.Info().Msg("Consumer context cancelled, shutting down")
			return ctx.Err()
		default:
			// Read message with timeout
			msg, err := c.reader.FetchMessage(ctx)
			if err != nil {
				if err == context.Canceled || err == context.DeadlineExceeded {
					return err
				}
				log.Error().
					Err(err).
					Msg("Failed to fetch message from Kafka")
				continue
			}

			// Process message
			log.Debug().
				Str("topic", msg.Topic).
				Int("partition", msg.Partition).
				Int64("offset", msg.Offset).
				Str("key", string(msg.Key)).
				Msg("Processing Kafka message")

			// Call handler
			if err := handler(ctx, msg.Key, msg.Value); err != nil {
				log.Error().
					Err(err).
					Str("topic", msg.Topic).
					Int64("offset", msg.Offset).
					Msg("Failed to process message")
				// Continue processing other messages (at-least-once delivery)
				continue
			}

			// Commit message
			if err := c.reader.CommitMessages(ctx, msg); err != nil {
				log.Error().
					Err(err).
					Int64("offset", msg.Offset).
					Msg("Failed to commit message")
			} else {
				log.Debug().
					Int64("offset", msg.Offset).
					Msg("Message committed successfully")
			}
		}
	}
}

// Close closes the Kafka consumer
func (c *Consumer) Close() error {
	if err := c.reader.Close(); err != nil {
		log.Error().
			Err(err).
			Msg("Failed to close Kafka consumer")
		return err
	}
	log.Info().Msg("Kafka consumer closed")
	return nil
}

// UnmarshalMessage unmarshals a JSON Kafka message value
func UnmarshalMessage(value []byte, target interface{}) error {
	if err := json.Unmarshal(value, target); err != nil {
		return fmt.Errorf("failed to unmarshal message: %w", err)
	}
	return nil
}
