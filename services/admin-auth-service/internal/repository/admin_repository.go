// Package repository provides PostgreSQL-backed storage for platform admins.
package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/lib/pq"

	"slate/services/admin-auth-service/internal/models"
)

// ErrNotFound is returned when no row matches the query.
var ErrNotFound = errors.New("admin not found")

// ErrDuplicateEmail is returned when a Create collides on the email unique index.
var ErrDuplicateEmail = errors.New("admin with this email already exists")

// AdminRepository executes queries against platform_admins.
type AdminRepository struct {
	db *sql.DB
}

// NewAdminRepository wires an AdminRepository to the given database handle.
func NewAdminRepository(db *sql.DB) *AdminRepository {
	return &AdminRepository{db: db}
}

// Create inserts a new platform admin row.
func (r *AdminRepository) Create(ctx context.Context, a *models.PlatformAdmin) error {
	const q = `
INSERT INTO platform_admins (email, password_hash, full_name, roles, disabled)
VALUES ($1, $2, $3, $4, $5)
RETURNING id, created_at, updated_at`

	if len(a.Roles) == 0 {
		a.Roles = []string{"readonly"}
	}
	err := r.db.QueryRowContext(ctx, q,
		a.Email, a.PasswordHash, a.FullName, pq.Array(a.Roles), a.Disabled,
	).Scan(&a.ID, &a.CreatedAt, &a.UpdatedAt)
	if err != nil {
		if pqErr, ok := err.(*pq.Error); ok && pqErr.Code == "23505" {
			return ErrDuplicateEmail
		}
		return fmt.Errorf("create admin: %w", err)
	}
	return nil
}

// GetByEmail loads an admin row by email (used by AdminLogin).
func (r *AdminRepository) GetByEmail(ctx context.Context, email string) (*models.PlatformAdmin, error) {
	const q = `
SELECT id, email, password_hash, full_name, roles, disabled, created_at, updated_at, last_login_at
FROM platform_admins WHERE email = $1`
	return r.scanOne(r.db.QueryRowContext(ctx, q, email))
}

// GetByID loads an admin row by id (used by ValidateAdminToken / Impersonate).
func (r *AdminRepository) GetByID(ctx context.Context, id string) (*models.PlatformAdmin, error) {
	const q = `
SELECT id, email, password_hash, full_name, roles, disabled, created_at, updated_at, last_login_at
FROM platform_admins WHERE id = $1`
	return r.scanOne(r.db.QueryRowContext(ctx, q, id))
}

// List returns up to limit admins, ordered by created_at DESC.
func (r *AdminRepository) List(ctx context.Context, limit, offset int) ([]*models.PlatformAdmin, int, error) {
	if limit <= 0 || limit > 500 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}
	const countQ = `SELECT COUNT(*) FROM platform_admins`
	var total int
	if err := r.db.QueryRowContext(ctx, countQ).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count admins: %w", err)
	}
	const q = `
SELECT id, email, password_hash, full_name, roles, disabled, created_at, updated_at, last_login_at
FROM platform_admins
ORDER BY created_at DESC
LIMIT $1 OFFSET $2`
	rows, err := r.db.QueryContext(ctx, q, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("list admins: %w", err)
	}
	defer rows.Close()

	var out []*models.PlatformAdmin
	for rows.Next() {
		a, err := r.scanRow(rows)
		if err != nil {
			return nil, 0, err
		}
		out = append(out, a)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("list admins iter: %w", err)
	}
	return out, total, nil
}

// UpdateLastLogin stamps last_login_at = NOW() for the given admin.
func (r *AdminRepository) UpdateLastLogin(ctx context.Context, id string) error {
	const q = `UPDATE platform_admins SET last_login_at = NOW(), updated_at = NOW() WHERE id = $1`
	_, err := r.db.ExecContext(ctx, q, id)
	if err != nil {
		return fmt.Errorf("update last_login: %w", err)
	}
	return nil
}

type rowScanner interface {
	Scan(dest ...any) error
}

func (r *AdminRepository) scanOne(row rowScanner) (*models.PlatformAdmin, error) {
	a, err := r.scanRow(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return a, nil
}

func (r *AdminRepository) scanRow(row rowScanner) (*models.PlatformAdmin, error) {
	var (
		a           models.PlatformAdmin
		lastLogin   sql.NullTime
		rolesPQ     pq.StringArray
	)
	if err := row.Scan(
		&a.ID, &a.Email, &a.PasswordHash, &a.FullName,
		&rolesPQ, &a.Disabled, &a.CreatedAt, &a.UpdatedAt, &lastLogin,
	); err != nil {
		return nil, err
	}
	a.Roles = []string(rolesPQ)
	if lastLogin.Valid {
		t := lastLogin.Time
		a.LastLoginAt = &t
	}
	return &a, nil
}

// PingContext is exposed for the health-check path.
func (r *AdminRepository) PingContext(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	return r.db.PingContext(ctx)
}
