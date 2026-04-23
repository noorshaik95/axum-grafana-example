// Package kafka wires the invalidation consumers described in W7.2/W7.3:
//   - grade.updated      → drop tenant:{slug}:grades:{user}:{course}
//   - assignment.created → drop tenant:{slug}:study_plan:{user}
//
// Consumption is best-effort: a malformed message is logged and skipped.
package kafka

import (
	"context"
	"encoding/json"

	"github.com/segmentio/kafka-go"

	"slate/libs/common-go/logging"
	"slate/services/ai-service/internal/service"
)

type InvalidationConsumer struct {
	brokers       []string
	groupID       string
	gradeTopic    string
	planTopic     string
	grades        *service.GradeProjector
	studyPlan     *service.StudyPlanner
	log           *logging.Logger
}

func NewInvalidationConsumer(brokers []string, groupID string, grades *service.GradeProjector, planner *service.StudyPlanner, log *logging.Logger) *InvalidationConsumer {
	return &InvalidationConsumer{
		brokers:    brokers,
		groupID:    groupID,
		gradeTopic: "grade.updated",
		planTopic:  "assignment.created",
		grades:     grades,
		studyPlan:  planner,
		log:        log,
	}
}

// Start launches both topic readers in background goroutines. Call the
// returned cancel function on shutdown.
func (c *InvalidationConsumer) Start(ctx context.Context) context.CancelFunc {
	cctx, cancel := context.WithCancel(ctx)
	go c.consume(cctx, c.gradeTopic, c.handleGradeUpdated)
	go c.consume(cctx, c.planTopic, c.handleAssignmentCreated)
	return cancel
}

func (c *InvalidationConsumer) consume(ctx context.Context, topic string, handler func(context.Context, []byte)) {
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers: c.brokers,
		GroupID: c.groupID,
		Topic:   topic,
	})
	defer reader.Close()
	for {
		msg, err := reader.ReadMessage(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			c.log.Warn().Err(err).Str("topic", topic).Msg("kafka read error")
			continue
		}
		handler(ctx, msg.Value)
	}
}

type gradeUpdatedEvent struct {
	TenantSlug string `json:"tenant_slug"`
	UserID     string `json:"user_id"`
	CourseID   string `json:"course_id"`
}

func (c *InvalidationConsumer) handleGradeUpdated(ctx context.Context, payload []byte) {
	var e gradeUpdatedEvent
	if err := json.Unmarshal(payload, &e); err != nil {
		c.log.Warn().Err(err).Msg("grade.updated: bad payload")
		return
	}
	if c.grades == nil {
		return
	}
	if err := c.grades.InvalidateGrade(ctx, e.TenantSlug, e.UserID, e.CourseID); err != nil {
		c.log.Warn().Err(err).Msg("grade.updated: invalidate failed")
	}
}

type assignmentCreatedEvent struct {
	TenantSlug string `json:"tenant_slug"`
	UserID     string `json:"user_id"`
}

func (c *InvalidationConsumer) handleAssignmentCreated(ctx context.Context, payload []byte) {
	var e assignmentCreatedEvent
	if err := json.Unmarshal(payload, &e); err != nil {
		c.log.Warn().Err(err).Msg("assignment.created: bad payload")
		return
	}
	if c.studyPlan == nil {
		return
	}
	if err := c.studyPlan.InvalidateStudyPlan(ctx, e.TenantSlug, e.UserID); err != nil {
		c.log.Warn().Err(err).Msg("assignment.created: invalidate failed")
	}
}
