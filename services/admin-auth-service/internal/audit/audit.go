// Package audit records admin actions and emits the matching Kafka event.
package audit

import (
	"context"
	"encoding/json"
	"time"

	commontracing "slate/libs/common-go/tracing"
	"slate/services/admin-auth-service/internal/kafka"
	"slate/services/admin-auth-service/internal/models"
	"slate/services/admin-auth-service/internal/repository"
)

// Action constants for the action column.
const (
	ActionAdminLogin            = "admin.login"
	ActionAdminLoginFailed      = "admin.login_failed"
	ActionAdminLogout           = "admin.logout"
	ActionAdminRegister         = "admin.register"
	ActionImpersonationStarted  = "impersonation.started"
	ActionImpersonationRejected = "impersonation.rejected"
	ActionListAdminUsers        = "admin.list_users"
	ActionAdminTokenRefresh     = "admin.token_refresh"
)

// Event is the on-wire Kafka payload for audit.admin_action.
type Event struct {
	ID         string         `json:"id"`
	ActorID    string         `json:"actor_id"`
	Action     string         `json:"action"`
	TargetID   string         `json:"target_id,omitempty"`
	TargetType string         `json:"target_type,omitempty"`
	Metadata   map[string]any `json:"metadata,omitempty"`
	RequestID  string         `json:"request_id,omitempty"`
	CreatedAt  time.Time      `json:"created_at"`
}

// Recorder persists an audit row and emits the matching Kafka event.
type Recorder struct {
	repo     *repository.AuditRepository
	producer Producer
}

// Producer is the subset of kafka.Producer used by Recorder. Interface lets
// us swap in a fake in tests.
type Producer interface {
	ProduceAdminAction(ctx context.Context, key string, payload []byte) error
	ProduceImpersonationStarted(ctx context.Context, key string, payload []byte) error
	Close() error
}

// NewRecorder wires a Recorder with a repository and a Kafka producer.
// producer may be nil — in which case audit rows are written but no event
// is emitted. That keeps the service functional in environments where
// Kafka is unavailable (dev bring-up, tests).
func NewRecorder(repo *repository.AuditRepository, producer Producer) *Recorder {
	return &Recorder{repo: repo, producer: producer}
}

// Record inserts an audit row and emits the matching Kafka event.
// Failure to emit Kafka does not fail the caller — it's logged by the
// producer and the row is already persisted. Failure to insert the row
// returns an error since the audit ledger is authoritative.
func (r *Recorder) Record(ctx context.Context, entry *models.AuditEntry) error {
	if entry.RequestID == "" {
		entry.RequestID = commontracing.RequestIDFromContext(ctx)
	}
	if err := r.repo.Insert(ctx, entry); err != nil {
		return err
	}
	if r.producer == nil {
		return nil
	}
	ev := Event{
		ID:         entry.ID,
		ActorID:    entry.ActorID,
		Action:     entry.Action,
		TargetID:   entry.TargetID,
		TargetType: entry.TargetType,
		Metadata:   entry.Metadata,
		RequestID:  entry.RequestID,
		CreatedAt:  entry.CreatedAt,
	}
	payload, err := json.Marshal(ev)
	if err != nil {
		// Marshalling shouldn't fail — but if it does, swallow after
		// row is persisted so we don't fail the admin action.
		return nil
	}
	if entry.Action == ActionImpersonationStarted {
		_ = r.producer.ProduceImpersonationStarted(ctx, entry.ActorID, payload)
	}
	_ = r.producer.ProduceAdminAction(ctx, entry.ActorID, payload)
	return nil
}

// Compile-time check that *kafka.Producer satisfies Producer.
var _ Producer = (*kafka.Producer)(nil)
