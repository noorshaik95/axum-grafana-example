package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// Rule types per plan W3.2.
const (
	RuleTypeAll        = "all"
	RuleTypeTenant     = "tenant"
	RuleTypeRole       = "role"
	RuleTypePercentage = "percentage"
	RuleTypeUser       = "user"
)

// ErrNotFound indicates the requested flag does not exist.
var ErrNotFound = errors.New("flag not found")

// Rule is a single targeting rule stored in flag_rules.
type Rule struct {
	ID        uuid.UUID
	FlagID    uuid.UUID
	RuleType  string
	RuleValue string // raw JSON as stored in JSONB
	CreatedAt time.Time
}

// Flag is the aggregate domain object — a flag plus its targeting rules.
type Flag struct {
	ID          uuid.UUID
	Key         string
	Description string
	Enabled     bool
	Rules       []Rule
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// Store is the flag persistence surface. Mockable for tests that don't need real postgres.
type Store interface {
	List(ctx context.Context) ([]Flag, error)
	GetByKey(ctx context.Context, key string) (*Flag, error)
	Upsert(ctx context.Context, key string, enabled bool, rules []Rule) (*Flag, error)
	Delete(ctx context.Context, key string) error
}

// PostgresStore is the sql.DB-backed Store.
type PostgresStore struct{ db *sql.DB }

func NewPostgresStore(db *sql.DB) *PostgresStore { return &PostgresStore{db: db} }

func (s *PostgresStore) List(ctx context.Context) ([]Flag, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, key, COALESCE(description, ''), enabled, created_at, updated_at
		FROM flags ORDER BY key ASC
	`)
	if err != nil {
		return nil, fmt.Errorf("list flags: %w", err)
	}
	defer rows.Close()

	var flags []Flag
	for rows.Next() {
		var f Flag
		if err := rows.Scan(&f.ID, &f.Key, &f.Description, &f.Enabled, &f.CreatedAt, &f.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan flag: %w", err)
		}
		flags = append(flags, f)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	for i := range flags {
		rules, err := s.rulesFor(ctx, flags[i].ID)
		if err != nil {
			return nil, err
		}
		flags[i].Rules = rules
	}
	return flags, nil
}

func (s *PostgresStore) GetByKey(ctx context.Context, key string) (*Flag, error) {
	var f Flag
	err := s.db.QueryRowContext(ctx, `
		SELECT id, key, COALESCE(description, ''), enabled, created_at, updated_at
		FROM flags WHERE key = $1
	`, key).Scan(&f.ID, &f.Key, &f.Description, &f.Enabled, &f.CreatedAt, &f.UpdatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("get flag: %w", err)
	}
	rules, err := s.rulesFor(ctx, f.ID)
	if err != nil {
		return nil, err
	}
	f.Rules = rules
	return &f, nil
}

func (s *PostgresStore) rulesFor(ctx context.Context, flagID uuid.UUID) ([]Rule, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, flag_id, rule_type, rule_value::text, created_at
		FROM flag_rules WHERE flag_id = $1 ORDER BY created_at ASC
	`, flagID)
	if err != nil {
		return nil, fmt.Errorf("list rules: %w", err)
	}
	defer rows.Close()

	var rules []Rule
	for rows.Next() {
		var r Rule
		if err := rows.Scan(&r.ID, &r.FlagID, &r.RuleType, &r.RuleValue, &r.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan rule: %w", err)
		}
		rules = append(rules, r)
	}
	return rules, rows.Err()
}

// Upsert updates the flag's enabled state and replaces all of its rules atomically.
// If the flag does not exist a new one is created.
func (s *PostgresStore) Upsert(ctx context.Context, key string, enabled bool, rules []Rule) (*Flag, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	var id uuid.UUID
	err = tx.QueryRowContext(ctx, `
		INSERT INTO flags (key, enabled, updated_at)
		VALUES ($1, $2, NOW())
		ON CONFLICT (key) DO UPDATE
		   SET enabled = EXCLUDED.enabled,
		       updated_at = NOW()
		RETURNING id
	`, key, enabled).Scan(&id)
	if err != nil {
		return nil, fmt.Errorf("upsert flag: %w", err)
	}

	if _, err := tx.ExecContext(ctx, `DELETE FROM flag_rules WHERE flag_id = $1`, id); err != nil {
		return nil, fmt.Errorf("clear rules: %w", err)
	}
	for _, r := range rules {
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO flag_rules (flag_id, rule_type, rule_value)
			VALUES ($1, $2, $3::jsonb)
		`, id, r.RuleType, r.RuleValue); err != nil {
			return nil, fmt.Errorf("insert rule: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit: %w", err)
	}
	return s.GetByKey(ctx, key)
}

func (s *PostgresStore) Delete(ctx context.Context, key string) error {
	res, err := s.db.ExecContext(ctx, `DELETE FROM flags WHERE key = $1`, key)
	if err != nil {
		return fmt.Errorf("delete flag: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrNotFound
	}
	return nil
}
