package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"

	"slate/services/metrics-service/internal/analytics"
	"slate/services/metrics-service/internal/models"
	"slate/services/metrics-service/internal/repository"

	"github.com/rs/zerolog/log"
	"github.com/segmentio/kafka-go"
)

// Consumer handles Kafka message consumption for metrics events.
type Consumer struct {
	brokers []string
	groupID string
	repo    *repository.Repository
}

// NewConsumer creates a new Kafka consumer.
func NewConsumer(brokers []string, groupID string, repo *repository.Repository) *Consumer {
	return &Consumer{
		brokers: brokers,
		groupID: groupID,
		repo:    repo,
	}
}

// Run starts consuming from all relevant topics. Blocks until ctx is cancelled.
func (c *Consumer) Run(ctx context.Context) error {
	errCh := make(chan error, 5)

	go func() { errCh <- c.consumeTopic(ctx, "lesson.completed", c.handleLessonCompleted) }()
	go func() { errCh <- c.consumeTopic(ctx, "submission.uploaded", c.handleSubmissionUploaded) }()
	go func() { errCh <- c.consumeTopic(ctx, "submission.graded", c.handleSubmissionGraded) }()
	go func() { errCh <- c.consumeTopic(ctx, "room.ended", c.handleRoomEnded) }()
	go func() { errCh <- c.consumeTopic(ctx, "message.sent", c.handleMessageSent) }()

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (c *Consumer) consumeTopic(ctx context.Context, topic string, handler func(ctx context.Context, msg []byte) error) error {
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers: c.brokers,
		Topic:   topic,
		GroupID: c.groupID,
	})
	defer reader.Close()

	log.Info().Str("topic", topic).Msg("started consuming")

	for {
		msg, err := reader.ReadMessage(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return nil
			}
			log.Error().Err(err).Str("topic", topic).Msg("error reading message")
			continue
		}

		if err := handler(ctx, msg.Value); err != nil {
			log.Error().Err(err).Str("topic", topic).Msg("error handling message")
		}
	}
}

func (c *Consumer) handleLessonCompleted(ctx context.Context, data []byte) error {
	var event models.LessonCompletedEvent
	if err := json.Unmarshal(data, &event); err != nil {
		return fmt.Errorf("unmarshal lesson.completed: %w", err)
	}

	// Upsert student progress
	if err := c.repo.UpsertStudentProgress(ctx,
		event.TenantID, event.StudentID, event.CourseID,
		1, event.CompletionPct, event.DurationMinutes,
	); err != nil {
		return fmt.Errorf("upsert student progress: %w", err)
	}

	// Log to event_log
	metadata := map[string]interface{}{
		"lessonId":        event.LessonID,
		"completionPct":   event.CompletionPct,
		"durationMinutes": event.DurationMinutes,
	}
	if err := c.repo.InsertEvent(ctx,
		event.TenantID, "lesson.completed",
		&event.StudentID, &event.CourseID, nil, metadata,
	); err != nil {
		return fmt.Errorf("insert event: %w", err)
	}

	log.Debug().Str("studentId", event.StudentID).Str("courseId", event.CourseID).Msg("processed lesson.completed")
	return nil
}

func (c *Consumer) handleSubmissionUploaded(ctx context.Context, data []byte) error {
	var event models.SubmissionEvent
	if err := json.Unmarshal(data, &event); err != nil {
		return fmt.Errorf("unmarshal submission.uploaded: %w", err)
	}

	if err := c.repo.InsertEvent(ctx,
		event.TenantID, "submission.uploaded",
		&event.StudentID, &event.CourseID, &event.AssignmentID, nil,
	); err != nil {
		return fmt.Errorf("insert event: %w", err)
	}

	log.Debug().Str("studentId", event.StudentID).Str("assignmentId", event.AssignmentID).Msg("processed submission.uploaded")
	return nil
}

func (c *Consumer) handleSubmissionGraded(ctx context.Context, data []byte) error {
	var event models.GradeEvent
	if err := json.Unmarshal(data, &event); err != nil {
		return fmt.Errorf("unmarshal submission.graded: %w", err)
	}

	// Normalize score to 0-100
	score := event.Score
	if event.MaxScore > 0 && event.MaxScore != 100 {
		score = (event.Score / event.MaxScore) * 100
	}

	// Log to event_log with score metadata
	metadata := map[string]interface{}{
		"score":    score,
		"maxScore": event.MaxScore,
		"rawScore": event.Score,
	}
	if err := c.repo.InsertEvent(ctx,
		event.TenantID, "submission.graded",
		&event.StudentID, &event.CourseID, &event.AssignmentID, metadata,
	); err != nil {
		return fmt.Errorf("insert event: %w", err)
	}

	// Recompute grade stats for assignment
	scores, err := c.repo.GetScoresForAssignment(ctx, event.TenantID, event.CourseID, event.AssignmentID)
	if err != nil {
		return fmt.Errorf("get scores for assignment: %w", err)
	}

	if len(scores) > 0 {
		sort.Float64s(scores)
		dist := analytics.ComputeDistribution(event.CourseID, scores)
		stats := &models.GradeStats{
			TenantID:     event.TenantID,
			CourseID:     event.CourseID,
			AssignmentID: event.AssignmentID,
			MeanScore:    dist.Mean,
			MedianScore:  dist.Median,
			P25:          dist.P25,
			P75:          dist.P75,
			StdDev:       analytics.StdDev(scores, dist.Mean),
			StudentCount: dist.TotalStudents,
		}
		if err := c.repo.UpsertGradeStats(ctx, event.TenantID, event.CourseID, &event.AssignmentID, stats); err != nil {
			return fmt.Errorf("upsert grade stats: %w", err)
		}
	}

	// Also recompute course-level stats
	courseScores, err := c.repo.GetScoresForCourse(ctx, event.TenantID, event.CourseID)
	if err != nil {
		return fmt.Errorf("get scores for course: %w", err)
	}

	if len(courseScores) > 0 {
		sort.Float64s(courseScores)
		dist := analytics.ComputeDistribution(event.CourseID, courseScores)
		stats := &models.GradeStats{
			TenantID:     event.TenantID,
			CourseID:     event.CourseID,
			MeanScore:    dist.Mean,
			MedianScore:  dist.Median,
			P25:          dist.P25,
			P75:          dist.P75,
			StdDev:       analytics.StdDev(courseScores, dist.Mean),
			StudentCount: dist.TotalStudents,
		}
		if err := c.repo.UpsertGradeStats(ctx, event.TenantID, event.CourseID, nil, stats); err != nil {
			return fmt.Errorf("upsert course grade stats: %w", err)
		}
	}

	log.Debug().Str("studentId", event.StudentID).Str("assignmentId", event.AssignmentID).Float64("score", score).Msg("processed submission.graded")
	return nil
}

func (c *Consumer) handleRoomEnded(ctx context.Context, data []byte) error {
	var event models.RoomEndedEvent
	if err := json.Unmarshal(data, &event); err != nil {
		return fmt.Errorf("unmarshal room.ended: %w", err)
	}

	// Log attendance for each participant
	for _, participantID := range event.Participants {
		pid := participantID
		metadata := map[string]interface{}{
			"sessionId":       event.RoomID,
			"durationMinutes": event.DurationMinutes,
		}
		if err := c.repo.InsertEvent(ctx,
			event.TenantID, "attendance.recorded",
			&pid, &event.CourseID, nil, metadata,
		); err != nil {
			log.Error().Err(err).Str("participantId", pid).Msg("failed to log attendance")
		}
	}

	log.Debug().Str("roomId", event.RoomID).Int("participants", len(event.Participants)).Msg("processed room.ended")
	return nil
}

func (c *Consumer) handleMessageSent(ctx context.Context, data []byte) error {
	var event models.MessageSentEvent
	if err := json.Unmarshal(data, &event); err != nil {
		return fmt.Errorf("unmarshal message.sent: %w", err)
	}

	courseID := event.CourseID
	var coursePtr *string
	if courseID != "" {
		coursePtr = &courseID
	}

	if err := c.repo.InsertEvent(ctx,
		event.TenantID, "message.sent",
		&event.UserID, coursePtr, nil, nil,
	); err != nil {
		return fmt.Errorf("insert event: %w", err)
	}

	log.Debug().Str("userId", event.UserID).Msg("processed message.sent")
	return nil
}
