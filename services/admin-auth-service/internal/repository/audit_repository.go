package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"slate/services/admin-auth-service/internal/models"
)

// AuditFilter describes the optional predicates applied by Query.
// Empty-string / zero-value fields are treated as unset.
type AuditFilter struct {
	ActorID     string
	Action      string
	TargetID    string
	TargetType  string
	From        time.Time // inclusive; zero = no lower bound
	To          time.Time // exclusive; zero = no upper bound
	Limit       int       // default 50, max 500
	BeforeID    string    // keyset cursor: return rows with id < BeforeID (older rows)
	BeforeTime  time.Time // keyset cursor companion; matched with BeforeID for stable ordering
}

// AuditRepository persists admin action records to platform_audit.
type AuditRepository struct {
	db *sql.DB
}

// NewAuditRepository wires the repository to the given database handle.
func NewAuditRepository(db *sql.DB) *AuditRepository {
	return &AuditRepository{db: db}
}

// Insert writes one audit entry. Metadata is serialised as JSONB.
func (r *AuditRepository) Insert(ctx context.Context, e *models.AuditEntry) error {
	metaBytes := []byte("null")
	if e.Metadata != nil {
		b, err := json.Marshal(e.Metadata)
		if err != nil {
			return fmt.Errorf("marshal audit metadata: %w", err)
		}
		metaBytes = b
	}
	const q = `
INSERT INTO platform_audit (actor_id, action, target_id, target_type, metadata, request_id)
VALUES ($1, $2, $3, $4, $5::jsonb, $6)
RETURNING id, created_at`
	if err := r.db.QueryRowContext(ctx, q,
		e.ActorID, e.Action,
		nullIfEmpty(e.TargetID),
		nullIfEmpty(e.TargetType),
		string(metaBytes),
		nullIfEmpty(e.RequestID),
	).Scan(&e.ID, &e.CreatedAt); err != nil {
		return fmt.Errorf("insert audit: %w", err)
	}
	return nil
}

func nullIfEmpty(s string) any {
	if s == "" {
		return nil
	}
	return s
}

// Query returns audit rows matching the given filter, newest-first, with an
// optional keyset cursor. platform_admins is LEFT JOINed to populate admin_email.
func (r *AuditRepository) Query(ctx context.Context, f AuditFilter) ([]*models.AuditQueryRow, error) {
	limit := f.Limit
	if limit <= 0 {
		limit = 50
	}
	// Service layer clamps caller limit to 500 and adds +1 for over-fetch,
	// so 501 is the legitimate maximum here.
	if limit > 501 {
		limit = 501
	}
	var (
		clauses []string
		args    []any
	)
	// addEq appends `col = $N` with its value; assumes a single-placeholder clause.
	addEq := func(col string, v any) {
		args = append(args, v)
		clauses = append(clauses, fmt.Sprintf("%s = $%d", col, len(args)))
	}
	if f.ActorID != "" {
		addEq("a.actor_id", f.ActorID)
	}
	if f.Action != "" {
		addEq("a.action", f.Action)
	}
	if f.TargetID != "" {
		addEq("a.target_id", f.TargetID)
	}
	if f.TargetType != "" {
		addEq("a.target_type", f.TargetType)
	}
	if !f.From.IsZero() {
		args = append(args, f.From)
		clauses = append(clauses, fmt.Sprintf("a.created_at >= $%d", len(args)))
	}
	if !f.To.IsZero() {
		args = append(args, f.To)
		clauses = append(clauses, fmt.Sprintf("a.created_at < $%d", len(args)))
	}
	if f.BeforeID != "" && !f.BeforeTime.IsZero() {
		// keyset: (created_at, id) strictly less than (BeforeTime, BeforeID) in our DESC ordering.
		args = append(args, f.BeforeTime, f.BeforeTime, f.BeforeID)
		p1, p2, p3 := len(args)-2, len(args)-1, len(args)
		clauses = append(clauses, fmt.Sprintf(
			"(a.created_at < $%d OR (a.created_at = $%d AND a.id < $%d))", p1, p2, p3,
		))
	}

	where := ""
	if len(clauses) > 0 {
		where = "WHERE " + strings.Join(clauses, " AND ")
	}
	args = append(args, limit)

	q := fmt.Sprintf(`
SELECT a.id, a.actor_id, a.action, COALESCE(a.target_id, ''), COALESCE(a.target_type, ''),
       a.metadata, COALESCE(a.request_id, ''), a.created_at,
       COALESCE(u.email, '') AS admin_email
FROM platform_audit a
LEFT JOIN platform_admins u ON u.id::text = a.actor_id
%s
ORDER BY a.created_at DESC, a.id DESC
LIMIT $%d`, where, len(args))

	rows, err := r.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("query audit: %w", err)
	}
	defer rows.Close()

	var out []*models.AuditQueryRow
	for rows.Next() {
		var (
			row        models.AuditQueryRow
			metaBytes  []byte
			metaParsed map[string]any
		)
		if err := rows.Scan(
			&row.ID, &row.ActorID, &row.Action, &row.TargetID, &row.TargetType,
			&metaBytes, &row.RequestID, &row.CreatedAt, &row.AdminEmail,
		); err != nil {
			return nil, fmt.Errorf("scan audit: %w", err)
		}
		if len(metaBytes) > 0 && string(metaBytes) != "null" {
			_ = json.Unmarshal(metaBytes, &metaParsed)
		}
		row.Metadata = metaParsed
		out = append(out, &row)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iter audit: %w", err)
	}
	return out, nil
}
