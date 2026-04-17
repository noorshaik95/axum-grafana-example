package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	kafkago "github.com/segmentio/kafka-go"
)

// EventHandler processes a consumed event
type EventHandler func(ctx context.Context, event Event) error

// Consumer reads messages from Kafka topics
type Consumer struct {
	readers  []*kafkago.Reader
	handlers map[string]EventHandler
	enabled  bool
}

// ConsumerConfig holds configuration for the consumer
type ConsumerConfig struct {
	Brokers []string
	GroupID string
	Topics  []string
	Enabled bool
}

// NewConsumer creates a new Kafka consumer
func NewConsumer(cfg ConsumerConfig) *Consumer {
	if !cfg.Enabled {
		return &Consumer{enabled: false, handlers: make(map[string]EventHandler)}
	}

	c := &Consumer{
		enabled:  true,
		handlers: make(map[string]EventHandler),
	}

	for _, topic := range cfg.Topics {
		reader := kafkago.NewReader(kafkago.ReaderConfig{
			Brokers:        cfg.Brokers,
			Topic:          topic,
			GroupID:        cfg.GroupID,
			MinBytes:       1,
			MaxBytes:       10e6,
			MaxWait:        1 * time.Second,
			CommitInterval: time.Second,
		})
		c.readers = append(c.readers, reader)
	}

	return c
}

// RegisterHandler registers a handler for a specific event type
func (c *Consumer) RegisterHandler(eventType string, handler EventHandler) {
	c.handlers[eventType] = handler
}

// Start begins consuming messages from all configured topics
func (c *Consumer) Start(ctx context.Context) {
	if !c.enabled {
		return
	}

	for _, reader := range c.readers {
		go c.consumeFromReader(ctx, reader)
	}
}

func (c *Consumer) consumeFromReader(ctx context.Context, reader *kafkago.Reader) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
			msg, err := reader.ReadMessage(ctx)
			if err != nil {
				if ctx.Err() != nil {
					return
				}
				fmt.Printf("Error reading message: %v\n", err)
				continue
			}

			var event Event
			if err := json.Unmarshal(msg.Value, &event); err != nil {
				fmt.Printf("Error unmarshaling event: %v\n", err)
				continue
			}

			handler, ok := c.handlers[event.Type]
			if !ok {
				continue
			}

			if err := handler(ctx, event); err != nil {
				fmt.Printf("Error handling event %s: %v\n", event.Type, err)
			}
		}
	}
}

// Close closes all readers
func (c *Consumer) Close() error {
	for _, reader := range c.readers {
		if err := reader.Close(); err != nil {
			return err
		}
	}
	return nil
}
