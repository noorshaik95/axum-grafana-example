package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"slate/services/email-service/internal/repository"

	kafkago "github.com/segmentio/kafka-go"
)

type Consumer struct {
	readers []*kafkago.Reader
	repo    *repository.MessageRepository
}

type AssignmentGradedEvent struct {
	TenantID        string  `json:"tenant_id"`
	StudentID       string  `json:"student_id"`
	AssignmentTitle string  `json:"assignment_title"`
	Score           float64 `json:"score"`
	MaxScore        float64 `json:"max_score"`
	Feedback        string  `json:"feedback"`
}

type AnnouncementPostedEvent struct {
	TenantID           string   `json:"tenant_id"`
	InstructorID       string   `json:"instructor_id"`
	Title              string   `json:"title"`
	Body               string   `json:"body"`
	EnrolledStudentIDs []string `json:"enrolled_student_ids"`
}

type RoomCreatedEvent struct {
	TenantID          string   `json:"tenant_id"`
	RoomName          string   `json:"room_name"`
	InstructorID      string   `json:"instructor_id"`
	InvitedStudentIDs []string `json:"invited_student_ids"`
	StartTime         string   `json:"start_time"`
}

const systemUserID = "00000000-0000-0000-0000-000000000000"

func NewConsumer(brokers []string, groupID string, repo *repository.MessageRepository) *Consumer {
	topics := []string{"assignment.graded", "announcement.posted", "room.created"}

	var readers []*kafkago.Reader
	for _, topic := range topics {
		reader := kafkago.NewReader(kafkago.ReaderConfig{
			Brokers:  brokers,
			Topic:    topic,
			GroupID:  groupID,
			MinBytes: 1,
			MaxBytes: 10e6,
		})
		readers = append(readers, reader)
	}

	return &Consumer{readers: readers, repo: repo}
}

func (c *Consumer) Start(ctx context.Context) {
	for _, reader := range c.readers {
		go c.consume(ctx, reader)
	}
}

func (c *Consumer) consume(ctx context.Context, reader *kafkago.Reader) {
	topic := reader.Config().Topic
	log.Printf("Kafka consumer started for topic: %s", topic)

	for {
		msg, err := reader.ReadMessage(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			log.Printf("ERROR reading from %s: %v", topic, err)
			continue
		}

		if err := c.handleMessage(ctx, topic, msg.Value); err != nil {
			log.Printf("ERROR handling %s message: %v", topic, err)
		}
	}
}

func (c *Consumer) handleMessage(ctx context.Context, topic string, data []byte) error {
	switch topic {
	case "assignment.graded":
		return c.handleAssignmentGraded(ctx, data)
	case "announcement.posted":
		return c.handleAnnouncementPosted(ctx, data)
	case "room.created":
		return c.handleRoomCreated(ctx, data)
	default:
		return fmt.Errorf("unknown topic: %s", topic)
	}
}

func (c *Consumer) handleAssignmentGraded(ctx context.Context, data []byte) error {
	var event AssignmentGradedEvent
	if err := json.Unmarshal(data, &event); err != nil {
		return fmt.Errorf("unmarshal assignment.graded: %w", err)
	}

	subject := fmt.Sprintf("Your assignment '%s' has been graded", event.AssignmentTitle)
	body := fmt.Sprintf("Score: %.1f/%.1f\n\nFeedback: %s", event.Score, event.MaxScore, event.Feedback)

	_, err := c.repo.SendMessage(ctx, event.TenantID, systemUserID, subject, body, []string{event.StudentID})
	if err != nil {
		return fmt.Errorf("send grade notification: %w", err)
	}

	log.Printf("Sent grade notification to student %s for assignment '%s'", event.StudentID, event.AssignmentTitle)
	return nil
}

func (c *Consumer) handleAnnouncementPosted(ctx context.Context, data []byte) error {
	var event AnnouncementPostedEvent
	if err := json.Unmarshal(data, &event); err != nil {
		return fmt.Errorf("unmarshal announcement.posted: %w", err)
	}

	if len(event.EnrolledStudentIDs) == 0 {
		return nil
	}

	// One message, N recipients
	_, err := c.repo.SendMessage(ctx, event.TenantID, event.InstructorID, event.Title, event.Body, event.EnrolledStudentIDs)
	if err != nil {
		return fmt.Errorf("send announcement: %w", err)
	}

	log.Printf("Distributed announcement '%s' to %d students", event.Title, len(event.EnrolledStudentIDs))
	return nil
}

func (c *Consumer) handleRoomCreated(ctx context.Context, data []byte) error {
	var event RoomCreatedEvent
	if err := json.Unmarshal(data, &event); err != nil {
		return fmt.Errorf("unmarshal room.created: %w", err)
	}

	if len(event.InvitedStudentIDs) == 0 {
		return nil
	}

	subject := fmt.Sprintf("You're invited to session: %s", event.RoomName)
	body := fmt.Sprintf("You have been invited to a live session.\n\nSession: %s\nScheduled: %s\n\nJoin from your dashboard when the session begins.",
		event.RoomName, event.StartTime)

	_, err := c.repo.SendMessage(ctx, event.TenantID, event.InstructorID, subject, body, event.InvitedStudentIDs)
	if err != nil {
		return fmt.Errorf("send room invite: %w", err)
	}

	log.Printf("Sent room invite for '%s' to %d students", event.RoomName, len(event.InvitedStudentIDs))
	return nil
}

func (c *Consumer) Close() error {
	for _, reader := range c.readers {
		if err := reader.Close(); err != nil {
			log.Printf("ERROR closing reader: %v", err)
		}
	}
	return nil
}
