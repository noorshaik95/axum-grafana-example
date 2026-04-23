package repository

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/lib/pq"

	"slate/services/admin-auth-service/internal/models"
)

// RoleRepository reads platform_roles (W2.5).
type RoleRepository struct {
	db *sql.DB
}

// NewRoleRepository wires a RoleRepository.
func NewRoleRepository(db *sql.DB) *RoleRepository {
	return &RoleRepository{db: db}
}

// List returns platform_roles paginated by `name ASC` with a keyset cursor.
// afterName — return rows where name > afterName; empty = first page.
func (r *RoleRepository) List(ctx context.Context, afterName string, limit int) ([]*models.PlatformRole, int, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	var total int
	if err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM platform_roles`).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count roles: %w", err)
	}

	const q = `
SELECT name, description, permissions, created_at
FROM platform_roles
WHERE ($1 = '' OR name > $1)
ORDER BY name ASC
LIMIT $2`
	rows, err := r.db.QueryContext(ctx, q, afterName, limit)
	if err != nil {
		return nil, 0, fmt.Errorf("list roles: %w", err)
	}
	defer rows.Close()

	var out []*models.PlatformRole
	for rows.Next() {
		var (
			role models.PlatformRole
			desc sql.NullString
			perm pq.StringArray
		)
		if err := rows.Scan(&role.Name, &desc, &perm, &role.CreatedAt); err != nil {
			return nil, 0, fmt.Errorf("scan role: %w", err)
		}
		if desc.Valid {
			role.Description = desc.String
		}
		role.Permissions = []string(perm)
		out = append(out, &role)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iter roles: %w", err)
	}
	return out, total, nil
}
