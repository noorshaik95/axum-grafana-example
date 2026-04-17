package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"slate/services/tenant-service/internal/models"

	"github.com/google/uuid"
	"github.com/lib/pq"
)

// TenantCRUDRepository defines the new CRUD operations for the upgraded tenant model.
type TenantCRUDRepository interface {
	CreateTenant(ctx context.Context, tenant *models.TenantV2) error
	GetTenantByID(ctx context.Context, id string) (*models.TenantV2, error)
	GetBySlug(ctx context.Context, slug string) (*models.TenantV2, error)
	ListTenants(ctx context.Context, page, pageSize int, search string) ([]*models.TenantV2, int, error)
	UpdateTenantStatus(ctx context.Context, id string, status string) error
	UpdateTenantPlan(ctx context.Context, id string, plan models.Plan) error
	UpdateTenantAfterProvision(ctx context.Context, tenant *models.TenantV2) error
	SoftDeleteTenant(ctx context.Context, id string) error
}

type tenantCRUDRepository struct {
	db *sql.DB
}

// NewTenantCRUDRepository creates a repository for the new tenant model.
func NewTenantCRUDRepository(db *sql.DB) TenantCRUDRepository {
	return &tenantCRUDRepository{db: db}
}

func (r *tenantCRUDRepository) CreateTenant(ctx context.Context, tenant *models.TenantV2) error {
	if tenant.ID == "" {
		tenant.ID = uuid.New().String()
	}
	now := time.Now()
	tenant.CreatedAt = now
	tenant.UpdatedAt = now

	query := `
		INSERT INTO tenants_v2 (id, slug, name, admin_email, status, plan, subdomain, container_ids, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`
	_, err := r.db.ExecContext(ctx, query,
		tenant.ID, tenant.Slug, tenant.Name, tenant.AdminEmail, tenant.Status,
		tenant.Plan, tenant.Subdomain,
		pq.Array(models.StringArray(tenant.ContainerIDs)),
		tenant.CreatedAt, tenant.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to create tenant: %w", err)
	}
	return nil
}

func (r *tenantCRUDRepository) GetTenantByID(ctx context.Context, id string) (*models.TenantV2, error) {
	query := `
		SELECT id, slug, name, admin_email, status, plan, subdomain, container_ids, created_at, updated_at
		FROM tenants_v2
		WHERE id = $1 AND status != 'deleted'
	`
	tenant := &models.TenantV2{}
	var containerIDs models.StringArray
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&tenant.ID, &tenant.Slug, &tenant.Name, &tenant.AdminEmail, &tenant.Status,
		&tenant.Plan, &tenant.Subdomain, pq.Array(&containerIDs),
		&tenant.CreatedAt, &tenant.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("tenant not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get tenant: %w", err)
	}
	tenant.ContainerIDs = containerIDs
	return tenant, nil
}

func (r *tenantCRUDRepository) GetBySlug(ctx context.Context, slug string) (*models.TenantV2, error) {
	query := `
		SELECT id, slug, name, admin_email, status, plan, subdomain, container_ids, created_at, updated_at
		FROM tenants_v2
		WHERE slug = $1 AND status != 'deleted'
	`
	tenant := &models.TenantV2{}
	var containerIDs models.StringArray
	err := r.db.QueryRowContext(ctx, query, slug).Scan(
		&tenant.ID, &tenant.Slug, &tenant.Name, &tenant.AdminEmail, &tenant.Status,
		&tenant.Plan, &tenant.Subdomain, pq.Array(&containerIDs),
		&tenant.CreatedAt, &tenant.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("tenant not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get tenant: %w", err)
	}
	tenant.ContainerIDs = containerIDs
	return tenant, nil
}

func (r *tenantCRUDRepository) ListTenants(ctx context.Context, page, pageSize int, search string) ([]*models.TenantV2, int, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize

	countQuery := `SELECT COUNT(*) FROM tenants_v2 WHERE status != 'deleted'`
	listQuery := `
		SELECT id, slug, name, admin_email, status, plan, subdomain, container_ids, created_at, updated_at
		FROM tenants_v2
		WHERE status != 'deleted'
	`

	args := []interface{}{}
	argIdx := 1

	if search != "" {
		filter := fmt.Sprintf(" AND (name ILIKE $%d OR slug ILIKE $%d)", argIdx, argIdx)
		countQuery += filter
		listQuery += filter
		args = append(args, "%"+search+"%")
		argIdx++
	}

	var total int
	err := r.db.QueryRowContext(ctx, countQuery, args...).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count tenants: %w", err)
	}

	listQuery += fmt.Sprintf(" ORDER BY created_at DESC LIMIT $%d OFFSET $%d", argIdx, argIdx+1)
	args = append(args, pageSize, offset)

	rows, err := r.db.QueryContext(ctx, listQuery, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list tenants: %w", err)
	}
	defer rows.Close()

	var tenants []*models.TenantV2
	for rows.Next() {
		tenant := &models.TenantV2{}
		var containerIDs models.StringArray
		if err := rows.Scan(
			&tenant.ID, &tenant.Slug, &tenant.Name, &tenant.AdminEmail, &tenant.Status,
			&tenant.Plan, &tenant.Subdomain, pq.Array(&containerIDs),
			&tenant.CreatedAt, &tenant.UpdatedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("failed to scan tenant: %w", err)
		}
		tenant.ContainerIDs = containerIDs
		tenants = append(tenants, tenant)
	}

	return tenants, total, nil
}

func (r *tenantCRUDRepository) UpdateTenantStatus(ctx context.Context, id string, status string) error {
	query := `UPDATE tenants_v2 SET status = $1, updated_at = $2 WHERE id = $3`
	_, err := r.db.ExecContext(ctx, query, status, time.Now(), id)
	if err != nil {
		return fmt.Errorf("failed to update tenant status: %w", err)
	}
	return nil
}

func (r *tenantCRUDRepository) UpdateTenantPlan(ctx context.Context, id string, plan models.Plan) error {
	query := `UPDATE tenants_v2 SET plan = $1, updated_at = $2 WHERE id = $3`
	_, err := r.db.ExecContext(ctx, query, plan, time.Now(), id)
	if err != nil {
		return fmt.Errorf("failed to update tenant plan: %w", err)
	}
	return nil
}

func (r *tenantCRUDRepository) UpdateTenantAfterProvision(ctx context.Context, tenant *models.TenantV2) error {
	query := `UPDATE tenants_v2 SET status = $1, container_ids = $2, updated_at = $3 WHERE id = $4`
	_, err := r.db.ExecContext(ctx, query,
		tenant.Status,
		pq.Array(models.StringArray(tenant.ContainerIDs)),
		time.Now(), tenant.ID,
	)
	if err != nil {
		return fmt.Errorf("failed to update tenant after provision: %w", err)
	}
	return nil
}

func (r *tenantCRUDRepository) SoftDeleteTenant(ctx context.Context, id string) error {
	query := `UPDATE tenants_v2 SET status = 'deleted', updated_at = $1 WHERE id = $2`
	_, err := r.db.ExecContext(ctx, query, time.Now(), id)
	if err != nil {
		return fmt.Errorf("failed to soft delete tenant: %w", err)
	}
	return nil
}
