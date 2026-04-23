package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"slate/services/incident-service/internal/models"
)

var (
	ErrNotFound          = errors.New("incident not found")
	ErrInvalidPriority   = errors.New("invalid priority")
	ErrInvalidStatus     = errors.New("invalid status")
	ErrInvalidTransition = errors.New("invalid status transition")
)

type Repository struct {
	db *sql.DB
}

func New(db *sql.DB) *Repository {
	return &Repository{db: db}
}

type CreateIncidentParams struct {
	TenantID    *string
	Service     *string
	Title       string
	Priority    string
	Impact      string // stored in description column
	CreatedBy   *string
}

func (r *Repository) CreateIncident(ctx context.Context, p CreateIncidentParams) (*models.Incident, error) {
	if !models.IsValidPriority(p.Priority) {
		return nil, ErrInvalidPriority
	}
	if strings.TrimSpace(p.Title) == "" {
		return nil, fmt.Errorf("title required")
	}

	inc := &models.Incident{
		TenantID:    p.TenantID,
		Service:     p.Service,
		Title:       p.Title,
		Description: p.Impact,
		Priority:    p.Priority,
		Status:      models.StatusOpen,
		CreatedBy:   p.CreatedBy,
	}

	const q = `
		INSERT INTO incidents (title, description, priority, status, tenant_id, service, created_by)
		VALUES ($1, $2, $3, 'open', $4, $5, $6)
		RETURNING id, created_at, updated_at
	`
	if err := r.db.QueryRowContext(ctx, q,
		p.Title, p.Impact, p.Priority, p.TenantID, p.Service, p.CreatedBy,
	).Scan(&inc.ID, &inc.CreatedAt, &inc.UpdatedAt); err != nil {
		return nil, fmt.Errorf("insert incident: %w", err)
	}
	return inc, nil
}

type UpdateIncidentParams struct {
	ID       string
	Status   *string
	Priority *string
	Title    *string
	Impact   *string
	ActorID  *string
}

// UpdateIncident applies a partial update. If Status transitions to resolved,
// resolved_at is set to NOW(). Returns the refreshed incident (without events).
// A status change also appends a status_change event inside the same transaction.
func (r *Repository) UpdateIncident(ctx context.Context, p UpdateIncidentParams) (*models.Incident, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	current, err := scanIncidentTx(ctx, tx, p.ID)
	if err != nil {
		return nil, err
	}

	newStatus := current.Status
	if p.Status != nil {
		if !models.IsValidStatus(*p.Status) {
			return nil, ErrInvalidStatus
		}
		if !models.IsValidTransition(current.Status, *p.Status) {
			return nil, ErrInvalidTransition
		}
		newStatus = *p.Status
	}

	newPriority := current.Priority
	if p.Priority != nil {
		if !models.IsValidPriority(*p.Priority) {
			return nil, ErrInvalidPriority
		}
		newPriority = *p.Priority
	}

	newTitle := current.Title
	if p.Title != nil {
		newTitle = *p.Title
	}
	newImpact := current.Description
	if p.Impact != nil {
		newImpact = *p.Impact
	}

	resolvedAtExpr := "resolved_at"
	args := []any{newStatus, newPriority, newTitle, newImpact, p.ID}
	if newStatus == models.StatusResolved && current.Status != models.StatusResolved {
		resolvedAtExpr = "NOW()"
	} else if newStatus != models.StatusResolved && current.Status == models.StatusResolved {
		// resolved -> something else is blocked by IsValidTransition, but defend anyway.
		resolvedAtExpr = "NULL"
	}

	q := fmt.Sprintf(`
		UPDATE incidents
		   SET status = $1, priority = $2, title = $3, description = $4,
		       resolved_at = %s, updated_at = NOW()
		 WHERE id = $5
		 RETURNING id, created_at, updated_at, resolved_at
	`, resolvedAtExpr)

	if err := tx.QueryRowContext(ctx, q, args...).
		Scan(&current.ID, &current.CreatedAt, &current.UpdatedAt, &current.ResolvedAt); err != nil {
		return nil, fmt.Errorf("update incident: %w", err)
	}

	if newStatus != current.Status {
		content := fmt.Sprintf("status %s -> %s", current.Status, newStatus)
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO incident_events (incident_id, actor_id, event_type, content)
			VALUES ($1, $2, 'status_change', $3)
		`, p.ID, p.ActorID, content); err != nil {
			return nil, fmt.Errorf("insert status_change event: %w", err)
		}
	}

	current.Status = newStatus
	current.Priority = newPriority
	current.Title = newTitle
	current.Description = newImpact

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit: %w", err)
	}
	return current, nil
}

func (r *Repository) GetIncident(ctx context.Context, id string) (*models.Incident, error) {
	inc, err := scanIncident(ctx, r.db, id)
	if err != nil {
		return nil, err
	}
	events, err := r.listEvents(ctx, id)
	if err != nil {
		return nil, err
	}
	inc.Events = events
	return inc, nil
}

type ListFilter struct {
	TenantID *string
	Status   *string
	Priority *string
	Service  *string
	Limit    int
}

func (r *Repository) ListIncidents(ctx context.Context, f ListFilter) ([]*models.Incident, error) {
	var (
		where []string
		args  []any
	)
	idx := 1
	if f.TenantID != nil {
		where = append(where, fmt.Sprintf("tenant_id = $%d", idx))
		args = append(args, *f.TenantID)
		idx++
	}
	if f.Status != nil {
		where = append(where, fmt.Sprintf("status = $%d", idx))
		args = append(args, *f.Status)
		idx++
	}
	if f.Priority != nil {
		where = append(where, fmt.Sprintf("priority = $%d", idx))
		args = append(args, *f.Priority)
		idx++
	}
	if f.Service != nil {
		where = append(where, fmt.Sprintf("service = $%d", idx))
		args = append(args, *f.Service)
		idx++
	}

	clause := ""
	if len(where) > 0 {
		clause = "WHERE " + strings.Join(where, " AND ")
	}
	limit := f.Limit
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	q := fmt.Sprintf(`
		SELECT id, title, description, priority, status, tenant_id, service, created_by,
		       resolved_at, created_at, updated_at
		  FROM incidents
		  %s
		 ORDER BY created_at DESC
		 LIMIT %d
	`, clause, limit)

	rows, err := r.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("query incidents: %w", err)
	}
	defer rows.Close()

	var out []*models.Incident
	for rows.Next() {
		inc, err := scanRow(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, inc)
	}
	return out, rows.Err()
}

// HasOpenIncidentFor returns true if there's already an open/watching incident
// for the given service+tenant combination. Used by the Kafka consumer to
// suppress duplicate auto-opens.
func (r *Repository) HasOpenIncidentFor(ctx context.Context, service string, tenantID *string) (bool, error) {
	q := `
		SELECT EXISTS(
			SELECT 1 FROM incidents
			 WHERE service = $1
			   AND status IN ('open','watching')
			   AND ($2::uuid IS NULL AND tenant_id IS NULL OR tenant_id = $2::uuid)
		)`
	var exists bool
	if err := r.db.QueryRowContext(ctx, q, service, tenantID).Scan(&exists); err != nil {
		return false, fmt.Errorf("check open incident: %w", err)
	}
	return exists, nil
}

func (r *Repository) AddEvent(ctx context.Context, incidentID string, actorID *string, eventType, content string) (*models.IncidentEvent, error) {
	if strings.TrimSpace(eventType) == "" {
		return nil, fmt.Errorf("event_type required")
	}
	ev := &models.IncidentEvent{
		IncidentID: incidentID,
		ActorID:    actorID,
		EventType:  eventType,
		Content:    content,
	}
	if err := r.db.QueryRowContext(ctx, `
		INSERT INTO incident_events (incident_id, actor_id, event_type, content)
		VALUES ($1, $2, $3, $4)
		RETURNING id, created_at
	`, incidentID, actorID, eventType, content).Scan(&ev.ID, &ev.CreatedAt); err != nil {
		return nil, fmt.Errorf("insert event: %w", err)
	}
	return ev, nil
}

// CountIncidentsSince returns how many incidents were opened in the last
// `since` window. Used by GetPublicStatus.
func (r *Repository) CountIncidentsSince(ctx context.Context, since time.Time) (int, error) {
	var n int
	if err := r.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM incidents WHERE created_at >= $1`, since,
	).Scan(&n); err != nil {
		return 0, fmt.Errorf("count incidents: %w", err)
	}
	return n, nil
}

// OpenIncidentsByService returns a map of service -> list of open+watching
// incidents, used to compute the public status page.
func (r *Repository) OpenIncidentsByService(ctx context.Context) (map[string][]*models.Incident, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, title, description, priority, status, tenant_id, service, created_by,
		       resolved_at, created_at, updated_at
		  FROM incidents
		 WHERE status IN ('open','watching')
		 ORDER BY priority ASC, created_at DESC
	`)
	if err != nil {
		return nil, fmt.Errorf("query open by service: %w", err)
	}
	defer rows.Close()

	out := make(map[string][]*models.Incident)
	for rows.Next() {
		inc, err := scanRow(rows)
		if err != nil {
			return nil, err
		}
		svc := ""
		if inc.Service != nil {
			svc = *inc.Service
		}
		out[svc] = append(out[svc], inc)
	}
	return out, rows.Err()
}

func (r *Repository) listEvents(ctx context.Context, incidentID string) ([]models.IncidentEvent, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, incident_id, actor_id, event_type, COALESCE(content, ''), created_at
		  FROM incident_events
		 WHERE incident_id = $1
		 ORDER BY created_at ASC
	`, incidentID)
	if err != nil {
		return nil, fmt.Errorf("list events: %w", err)
	}
	defer rows.Close()

	var out []models.IncidentEvent
	for rows.Next() {
		var ev models.IncidentEvent
		if err := rows.Scan(&ev.ID, &ev.IncidentID, &ev.ActorID, &ev.EventType, &ev.Content, &ev.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan event: %w", err)
		}
		out = append(out, ev)
	}
	return out, rows.Err()
}

type rowScanner interface {
	Scan(dest ...any) error
}

type queryRowAble interface {
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

func scanIncident(ctx context.Context, q queryRowAble, id string) (*models.Incident, error) {
	inc := &models.Incident{}
	err := q.QueryRowContext(ctx, `
		SELECT id, title, COALESCE(description, ''), priority, status, tenant_id, service,
		       created_by, resolved_at, created_at, updated_at
		  FROM incidents WHERE id = $1
	`, id).Scan(
		&inc.ID, &inc.Title, &inc.Description, &inc.Priority, &inc.Status,
		&inc.TenantID, &inc.Service, &inc.CreatedBy, &inc.ResolvedAt,
		&inc.CreatedAt, &inc.UpdatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("scan incident: %w", err)
	}
	return inc, nil
}

func scanIncidentTx(ctx context.Context, tx *sql.Tx, id string) (*models.Incident, error) {
	return scanIncident(ctx, tx, id)
}

func scanRow(r rowScanner) (*models.Incident, error) {
	inc := &models.Incident{}
	var desc sql.NullString
	if err := r.Scan(
		&inc.ID, &inc.Title, &desc, &inc.Priority, &inc.Status,
		&inc.TenantID, &inc.Service, &inc.CreatedBy,
		&inc.ResolvedAt, &inc.CreatedAt, &inc.UpdatedAt,
	); err != nil {
		return nil, fmt.Errorf("scan row: %w", err)
	}
	if desc.Valid {
		inc.Description = desc.String
	}
	return inc, nil
}
