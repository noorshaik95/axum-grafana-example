package service

import (
	"encoding/json"
	"testing"

	"slate/services/assignment-grading-service/pkg/kafka"

	"github.com/stretchr/testify/assert"
)

// These three struct literals mirror the HEAD shapes of the downstream
// grade.updated consumers. If any consumer changes its shape, the
// corresponding struct here must be updated in lockstep and this test will
// catch producer drift.

// aiServiceGradeUpdatedEvent mirrors
// services/ai-service/internal/kafka/consumer.go::gradeUpdatedEvent.
type aiServiceGradeUpdatedEvent struct {
	TenantSlug string `json:"tenant_slug"`
	UserID     string `json:"user_id"`
	CourseID   string `json:"course_id"`
}

// metricsServiceGradeUpdatedEvent mirrors
// services/metrics-service/internal/kafka/consumer.go::gradeUpdatedEvent.
type metricsServiceGradeUpdatedEvent struct {
	TenantSlug string `json:"tenant_slug"`
	TenantID   string `json:"tenant_id"`
	CourseID   string `json:"course_id"`
}

// emailServiceGradeUpdatedEvent mirrors
// services/email-service/internal/kafka/consumer.go::GradeUpdatedEvent.
// student_email intentionally stays empty from this producer —
// email-service resolves it lazily via user-auth when
// EMAIL_GRADE_NOTIFICATIONS=true (decision recorded in pkg/kafka/events.go).
type emailServiceGradeUpdatedEvent struct {
	TenantID        string  `json:"tenant_id"`
	StudentID       string  `json:"student_id"`
	StudentEmail    string  `json:"student_email"`
	AssignmentTitle string  `json:"assignment_title"`
	Score           float64 `json:"score"`
	MaxScore        float64 `json:"max_score"`
}

func TestGradeUpdatedEvent_ConsumerShapeParity(t *testing.T) {
	evt := kafka.NewGradeUpdatedEvent(kafka.GradeUpdatedPayload{
		GradeID:         "g1",
		AssignmentID:    "a1",
		AssignmentTitle: "CS101 PS4: Binary Trees",
		StudentID:       "student-42",
		TenantID:        "tenant-7",
		TenantSlug:      "eastfield",
		CourseID:        "course-CS101",
		Score:           85,
		MaxScore:        100,
		AdjustedScore:   80,
		PatternID:       "pattern-abc",
	})

	// The producer ships `Event.Data` as JSON on the wire. Round-trip
	// through Data → JSON → consumer struct, same as the actual consumers
	// perform on kafka.Message.Value.
	raw, err := json.Marshal(evt.Data)
	assert.NoError(t, err)

	t.Run("ai-service grade cache invalidator resolves required fields", func(t *testing.T) {
		var ai aiServiceGradeUpdatedEvent
		assert.NoError(t, json.Unmarshal(raw, &ai))
		assert.Equal(t, "eastfield", ai.TenantSlug, "ai-service keys tenant:{slug}:grades:{user}:{course}")
		assert.Equal(t, "student-42", ai.UserID, "ai-service keys on user_id (student alias)")
		assert.Equal(t, "course-CS101", ai.CourseID)
	})

	t.Run("metrics-service roster-health invalidator resolves required fields", func(t *testing.T) {
		var ms metricsServiceGradeUpdatedEvent
		assert.NoError(t, json.Unmarshal(raw, &ms))
		assert.True(t, ms.TenantSlug != "" || ms.TenantID != "", "metrics-service falls back to tenant_id when tenant_slug missing")
		assert.Equal(t, "eastfield", ms.TenantSlug)
		assert.Equal(t, "tenant-7", ms.TenantID)
		assert.Equal(t, "course-CS101", ms.CourseID)
	})

	t.Run("email-service outbound-email gate resolves required fields", func(t *testing.T) {
		var em emailServiceGradeUpdatedEvent
		assert.NoError(t, json.Unmarshal(raw, &em))
		assert.Equal(t, "tenant-7", em.TenantID)
		assert.Equal(t, "student-42", em.StudentID)
		assert.Equal(t, "CS101 PS4: Binary Trees", em.AssignmentTitle)
		assert.Equal(t, 85.0, em.Score)
		assert.Equal(t, 100.0, em.MaxScore)
		// student_email is emitted as "" on purpose: the email-service
		// consumer gate `event.StudentEmail == ""` falls through to the
		// user-auth lookup path per the coupling decision recorded in
		// pkg/kafka/events.go::GradeUpdatedPayload doc comment.
		assert.Equal(t, "", em.StudentEmail)
	})
}
