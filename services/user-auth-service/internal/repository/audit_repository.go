package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"slate/services/user-auth-service/internal/models"
)

// AuditRepository persists audit events. All Append calls are non-blocking to
// avoid turning an audit failure into an auth failure — callers log repo errors
// and continue.
type AuditRepository interface {
	Append(ctx context.Context, event *models.AuditEvent) error
	List(ctx context.Context, actorID string, limit int) ([]*models.AuditEvent, error)
}

type sqlAuditRepository struct {
	db *sql.DB
}

// NewAuditRepository returns a PostgreSQL-backed AuditRepository.
func NewAuditRepository(db *sql.DB) AuditRepository {
	return &sqlAuditRepository{db: db}
}

func (r *sqlAuditRepository) Append(ctx context.Context, event *models.AuditEvent) error {
	if event == nil {
		return fmt.Errorf("audit event is nil")
	}
	if event.Action == "" {
		return fmt.Errorf("audit action is required")
	}
	if event.CreatedAt.IsZero() {
		event.CreatedAt = time.Now().UTC()
	}

	metadataJSON := []byte("null")
	if event.Metadata != nil {
		b, err := json.Marshal(event.Metadata)
		if err != nil {
			return fmt.Errorf("failed to marshal audit metadata: %w", err)
		}
		metadataJSON = b
	}

	var ip interface{}
	if event.IPAddress != "" {
		ip = event.IPAddress
	}

	query := `
		INSERT INTO audit_events (actor_id, actor_type, action, target_id, target_type, metadata, ip_address, created_at)
		VALUES ($1, $2, $3, $4, $5, $6::jsonb, $7, $8)
	`
	_, err := r.db.ExecContext(ctx, query,
		nullable(event.ActorID),
		string(event.ActorType),
		string(event.Action),
		nullable(event.TargetID),
		nullable(event.TargetType),
		string(metadataJSON),
		ip,
		event.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to append audit event: %w", err)
	}
	return nil
}

func (r *sqlAuditRepository) List(ctx context.Context, actorID string, limit int) ([]*models.AuditEvent, error) {
	if limit <= 0 || limit > 1000 {
		limit = 100
	}
	rows, err := r.db.QueryContext(ctx, `
		SELECT id::text, COALESCE(actor_id,''), COALESCE(actor_type,''), action, COALESCE(target_id,''), COALESCE(target_type,''), COALESCE(metadata::text,'null'), COALESCE(host(ip_address),''), created_at
		FROM audit_events WHERE ($1 = '' OR actor_id = $1)
		ORDER BY created_at DESC LIMIT $2
	`, actorID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []*models.AuditEvent
	for rows.Next() {
		var e models.AuditEvent
		var actorType, action, metadataRaw string
		if err := rows.Scan(&e.ID, &e.ActorID, &actorType, &action, &e.TargetID, &e.TargetType, &metadataRaw, &e.IPAddress, &e.CreatedAt); err != nil {
			return nil, err
		}
		e.ActorType = models.AuditActorType(actorType)
		e.Action = models.AuditAction(action)
		if metadataRaw != "" && metadataRaw != "null" {
			_ = json.Unmarshal([]byte(metadataRaw), &e.Metadata)
		}
		events = append(events, &e)
	}
	return events, rows.Err()
}

func nullable(s string) interface{} {
	if s == "" {
		return nil
	}
	return s
}

// InMemoryAuditRepository is a thread-safe in-memory AuditRepository. Used in
// tests where a real database is not available; production code wires the SQL
// implementation.
type InMemoryAuditRepository struct {
	mu     sync.Mutex
	events []*models.AuditEvent
}

// NewInMemoryAuditRepository returns a new in-memory audit repository.
func NewInMemoryAuditRepository() *InMemoryAuditRepository {
	return &InMemoryAuditRepository{}
}

func (r *InMemoryAuditRepository) Append(_ context.Context, event *models.AuditEvent) error {
	if event == nil {
		return fmt.Errorf("audit event is nil")
	}
	if event.Action == "" {
		return fmt.Errorf("audit action is required")
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if event.CreatedAt.IsZero() {
		event.CreatedAt = time.Now().UTC()
	}
	cpy := *event
	if event.Metadata != nil {
		cpy.Metadata = make(map[string]interface{}, len(event.Metadata))
		for k, v := range event.Metadata {
			cpy.Metadata[k] = v
		}
	}
	r.events = append(r.events, &cpy)
	return nil
}

func (r *InMemoryAuditRepository) List(_ context.Context, actorID string, limit int) ([]*models.AuditEvent, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var out []*models.AuditEvent
	for i := len(r.events) - 1; i >= 0 && (limit <= 0 || len(out) < limit); i-- {
		if actorID == "" || r.events[i].ActorID == actorID {
			e := *r.events[i]
			out = append(out, &e)
		}
	}
	return out, nil
}

// ByAction returns every recorded event with the given action. Intended for
// test assertions.
func (r *InMemoryAuditRepository) ByAction(action models.AuditAction) []*models.AuditEvent {
	r.mu.Lock()
	defer r.mu.Unlock()
	var out []*models.AuditEvent
	for _, e := range r.events {
		if e.Action == action {
			cpy := *e
			out = append(out, &cpy)
		}
	}
	return out
}

// Len returns the total number of events (thread-safe).
func (r *InMemoryAuditRepository) Len() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.events)
}
