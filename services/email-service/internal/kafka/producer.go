package kafka

import (
	"context"
	"encoding/json"
	"log"
	"time"

	kafkago "github.com/segmentio/kafka-go"
)

type Producer struct {
	writer  *kafkago.Writer
	enabled bool
}

func NewProducer(brokers []string, enabled bool) *Producer {
	if !enabled {
		return &Producer{enabled: false}
	}

	writer := &kafkago.Writer{
		Addr:         kafkago.TCP(brokers...),
		Balancer:     &kafkago.LeastBytes{},
		MaxAttempts:  3,
		BatchSize:    1,
		BatchTimeout: 10 * time.Millisecond,
		WriteTimeout: 10 * time.Second,
		RequiredAcks: kafkago.RequireOne,
	}

	return &Producer{writer: writer, enabled: true}
}

func (p *Producer) PublishMessageSent(ctx context.Context, messageID, fromUserID string, recipientIDs []string, subject, tenantID string) {
	if !p.enabled {
		return
	}

	event := map[string]interface{}{
		"type":          "message.sent",
		"message_id":    messageID,
		"from_user_id":  fromUserID,
		"recipient_ids": recipientIDs,
		"subject":       subject,
		"tenant_id":     tenantID,
		"timestamp":     time.Now().UTC(),
	}

	data, err := json.Marshal(event)
	if err != nil {
		log.Printf("ERROR marshal message.sent event: %v", err)
		return
	}

	err = p.writer.WriteMessages(ctx, kafkago.Message{
		Topic: "message.sent",
		Key:   []byte(messageID),
		Value: data,
	})
	if err != nil {
		log.Printf("ERROR publish message.sent: %v", err)
	}
}

func (p *Producer) PublishMessageRead(ctx context.Context, messageID, userID, tenantID string) {
	if !p.enabled {
		return
	}

	event := map[string]interface{}{
		"type":       "message.read",
		"message_id": messageID,
		"user_id":    userID,
		"tenant_id":  tenantID,
		"timestamp":  time.Now().UTC(),
	}

	data, err := json.Marshal(event)
	if err != nil {
		log.Printf("ERROR marshal message.read event: %v", err)
		return
	}

	err = p.writer.WriteMessages(ctx, kafkago.Message{
		Topic: "message.read",
		Key:   []byte(messageID),
		Value: data,
	})
	if err != nil {
		log.Printf("ERROR publish message.read: %v", err)
	}
}

func (p *Producer) Close() error {
	if p.writer != nil {
		return p.writer.Close()
	}
	return nil
}
