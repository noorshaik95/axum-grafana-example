package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"slate/services/tenant-service/internal/models"
)

type TenantRepository interface {
	// Tenant operations
	Create(ctx context.Context, tenant *models.Tenant) error
	GetByID(ctx context.Context, id string) (*models.Tenant, error)
	GetByDomain(ctx context.Context, domain string) (*models.Tenant, error)
	Update(ctx context.Context, tenant *models.Tenant) error
	Delete(ctx context.Context, id string) error
	List(ctx context.Context, page, pageSize int32, filters map[string]interface{}) ([]*models.Tenant, int32, error)

	// Tier operations
	GetTierByID(ctx context.Context, tierID string) (*models.SubscriptionTier, error)
	GetTierByName(ctx context.Context, name string) (*models.SubscriptionTier, error)
	ListTiers(ctx context.Context) ([]*models.SubscriptionTier, error)

	// Admin operations
	CreateAdmin(ctx context.Context, admin *models.TenantAdmin) error
	GetAdminByID(ctx context.Context, tenantID, adminID string) (*models.TenantAdmin, error)
	GetAdminsByTenant(ctx context.Context, tenantID string) ([]*models.TenantAdmin, error)

	// Provisioning operations
	CreateProvisioning(ctx context.Context, provisioning *models.TenantProvisioning) error
	UpdateProvisioning(ctx context.Context, provisioning *models.TenantProvisioning) error
	GetProvisioningByID(ctx context.Context, provisioningID string) (*models.TenantProvisioning, error)

	// Setup token operations
	CreateSetupToken(ctx context.Context, token *models.TenantSetupToken) error
	GetSetupTokenByToken(ctx context.Context, token string) (*models.TenantSetupToken, error)
	MarkSetupTokenUsed(ctx context.Context, tokenID string) error

	// Storage operations
	UpdateStorageUsage(ctx context.Context, tenantID string, bytesDelta int64, fileCountDelta int32) error
	GetStorageQuota(ctx context.Context, tenantID string) (int64, int64, int32, error) // quota, used, files
}

type tenantRepository struct {
	db *sql.DB
}

func NewTenantRepository(db *sql.DB) TenantRepository {
	return &tenantRepository{db: db}
}

// Create creates a new tenant
func (r *tenantRepository) Create(ctx context.Context, tenant *models.Tenant) error {
	query := `
		INSERT INTO tenants (
			id, name, domain, tier_id, is_active, database_name, database_host,
			database_port, database_username, database_password_encrypted,
			database_connection_string, storage_quota_bytes, storage_used_bytes,
			file_count, user_count, course_count, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18)
	`

	_, err := r.db.ExecContext(ctx, query,
		tenant.ID, tenant.Name, tenant.Domain, tenant.TierID, tenant.IsActive,
		tenant.DatabaseName, tenant.DatabaseHost, tenant.DatabasePort,
		tenant.DatabaseUsername, tenant.DatabasePasswordEncrypted,
		tenant.DatabaseConnectionString, tenant.StorageQuotaBytes,
		tenant.StorageUsedBytes, tenant.FileCount, tenant.UserCount,
		tenant.CourseCount, tenant.CreatedAt, tenant.UpdatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to create tenant: %w", err)
	}

	return nil
}

// GetByID retrieves a tenant by ID
func (r *tenantRepository) GetByID(ctx context.Context, id string) (*models.Tenant, error) {
	query := `
		SELECT id, name, domain, tier_id, is_active, database_name, database_host,
			   database_port, database_username, database_password_encrypted,
			   database_connection_string, storage_quota_bytes, storage_used_bytes,
			   file_count, user_count, course_count, created_at, updated_at, last_active_at
		FROM tenants
		WHERE id = $1
	`

	tenant := &models.Tenant{}
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&tenant.ID, &tenant.Name, &tenant.Domain, &tenant.TierID, &tenant.IsActive,
		&tenant.DatabaseName, &tenant.DatabaseHost, &tenant.DatabasePort,
		&tenant.DatabaseUsername, &tenant.DatabasePasswordEncrypted,
		&tenant.DatabaseConnectionString, &tenant.StorageQuotaBytes,
		&tenant.StorageUsedBytes, &tenant.FileCount, &tenant.UserCount,
		&tenant.CourseCount, &tenant.CreatedAt, &tenant.UpdatedAt, &tenant.LastActiveAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("tenant not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get tenant: %w", err)
	}

	return tenant, nil
}

// GetByDomain retrieves a tenant by domain
func (r *tenantRepository) GetByDomain(ctx context.Context, domain string) (*models.Tenant, error) {
	query := `
		SELECT id, name, domain, tier_id, is_active, database_name, database_host,
			   database_port, database_username, database_password_encrypted,
			   database_connection_string, storage_quota_bytes, storage_used_bytes,
			   file_count, user_count, course_count, created_at, updated_at, last_active_at
		FROM tenants
		WHERE domain = $1
	`

	tenant := &models.Tenant{}
	err := r.db.QueryRowContext(ctx, query, domain).Scan(
		&tenant.ID, &tenant.Name, &tenant.Domain, &tenant.TierID, &tenant.IsActive,
		&tenant.DatabaseName, &tenant.DatabaseHost, &tenant.DatabasePort,
		&tenant.DatabaseUsername, &tenant.DatabasePasswordEncrypted,
		&tenant.DatabaseConnectionString, &tenant.StorageQuotaBytes,
		&tenant.StorageUsedBytes, &tenant.FileCount, &tenant.UserCount,
		&tenant.CourseCount, &tenant.CreatedAt, &tenant.UpdatedAt, &tenant.LastActiveAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("tenant not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get tenant: %w", err)
	}

	return tenant, nil
}

// Update updates a tenant
func (r *tenantRepository) Update(ctx context.Context, tenant *models.Tenant) error {
	query := `
		UPDATE tenants
		SET name = $1, domain = $2, tier_id = $3, is_active = $4,
			storage_quota_bytes = $5, user_count = $6, course_count = $7,
			updated_at = $8, last_active_at = $9
		WHERE id = $10
	`

	_, err := r.db.ExecContext(ctx, query,
		tenant.Name, tenant.Domain, tenant.TierID, tenant.IsActive,
		tenant.StorageQuotaBytes, tenant.UserCount, tenant.CourseCount,
		time.Now(), tenant.LastActiveAt, tenant.ID,
	)

	if err != nil {
		return fmt.Errorf("failed to update tenant: %w", err)
	}

	return nil
}

// Delete deletes a tenant
func (r *tenantRepository) Delete(ctx context.Context, id string) error {
	query := `DELETE FROM tenants WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete tenant: %w", err)
	}
	return nil
}

// List retrieves tenants with pagination and filters
func (r *tenantRepository) List(ctx context.Context, page, pageSize int32, filters map[string]interface{}) ([]*models.Tenant, int32, error) {
	offset := (page - 1) * pageSize

	// Build query with filters
	query := `
		SELECT id, name, domain, tier_id, is_active, storage_quota_bytes,
			   storage_used_bytes, file_count, user_count, course_count,
			   created_at, updated_at, last_active_at
		FROM tenants
		WHERE 1=1
	`
	countQuery := `SELECT COUNT(*) FROM tenants WHERE 1=1`

	args := []interface{}{}
	argPos := 1

	if search, ok := filters["search"].(string); ok && search != "" {
		query += fmt.Sprintf(" AND (name ILIKE $%d OR domain ILIKE $%d)", argPos, argPos)
		countQuery += fmt.Sprintf(" AND (name ILIKE $%d OR domain ILIKE $%d)", argPos, argPos)
		args = append(args, "%"+search+"%")
		argPos++
	}

	if isActive, ok := filters["is_active"].(bool); ok {
		query += fmt.Sprintf(" AND is_active = $%d", argPos)
		countQuery += fmt.Sprintf(" AND is_active = $%d", argPos)
		args = append(args, isActive)
		argPos++
	}

	if tierID, ok := filters["tier_id"].(string); ok && tierID != "" {
		query += fmt.Sprintf(" AND tier_id = $%d", argPos)
		countQuery += fmt.Sprintf(" AND tier_id = $%d", argPos)
		args = append(args, tierID)
		argPos++
	}

	// Add ordering and pagination
	sortBy := filters["sort_by"]
	if sortBy == nil {
		sortBy = "created_at"
	}
	sortOrder := filters["sort_order"]
	if sortOrder == nil {
		sortOrder = "desc"
	}

	query += fmt.Sprintf(" ORDER BY %s %s LIMIT $%d OFFSET $%d", sortBy, sortOrder, argPos, argPos+1)
	args = append(args, pageSize, offset)

	// Get total count
	var total int32
	err := r.db.QueryRowContext(ctx, countQuery, args[:len(args)-2]...).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count tenants: %w", err)
	}

	// Get tenants
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list tenants: %w", err)
	}
	defer rows.Close()

	var tenants []*models.Tenant
	for rows.Next() {
		tenant := &models.Tenant{}
		err := rows.Scan(
			&tenant.ID, &tenant.Name, &tenant.Domain, &tenant.TierID, &tenant.IsActive,
			&tenant.StorageQuotaBytes, &tenant.StorageUsedBytes, &tenant.FileCount,
			&tenant.UserCount, &tenant.CourseCount, &tenant.CreatedAt, &tenant.UpdatedAt,
			&tenant.LastActiveAt,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan tenant: %w", err)
		}
		tenants = append(tenants, tenant)
	}

	return tenants, total, nil
}

// GetTierByID retrieves a tier by ID
func (r *tenantRepository) GetTierByID(ctx context.Context, tierID string) (*models.SubscriptionTier, error) {
	query := `
		SELECT id, name, display_name, tier_level, storage_quota_bytes, max_users,
			   max_courses, api_rate_limit, dedicated_database, custom_domain,
			   priority_support, price_cents_monthly, features, created_at, updated_at
		FROM subscription_tiers
		WHERE id = $1
	`

	tier := &models.SubscriptionTier{}
	var featuresJSON []byte

	err := r.db.QueryRowContext(ctx, query, tierID).Scan(
		&tier.ID, &tier.Name, &tier.DisplayName, &tier.TierLevel, &tier.StorageQuotaBytes,
		&tier.MaxUsers, &tier.MaxCourses, &tier.APIRateLimit, &tier.DedicatedDatabase,
		&tier.CustomDomain, &tier.PrioritySupport, &tier.PriceCentsMonthly,
		&featuresJSON, &tier.CreatedAt, &tier.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("tier not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get tier: %w", err)
	}

	if len(featuresJSON) > 0 {
		if err := json.Unmarshal(featuresJSON, &tier.Features); err != nil {
			return nil, fmt.Errorf("failed to unmarshal tier features: %w", err)
		}
	}

	return tier, nil
}

// GetTierByName retrieves a tier by name
func (r *tenantRepository) GetTierByName(ctx context.Context, name string) (*models.SubscriptionTier, error) {
	query := `
		SELECT id, name, display_name, tier_level, storage_quota_bytes, max_users,
			   max_courses, api_rate_limit, dedicated_database, custom_domain,
			   priority_support, price_cents_monthly, features, created_at, updated_at
		FROM subscription_tiers
		WHERE name = $1
	`

	tier := &models.SubscriptionTier{}
	var featuresJSON []byte

	err := r.db.QueryRowContext(ctx, query, name).Scan(
		&tier.ID, &tier.Name, &tier.DisplayName, &tier.TierLevel, &tier.StorageQuotaBytes,
		&tier.MaxUsers, &tier.MaxCourses, &tier.APIRateLimit, &tier.DedicatedDatabase,
		&tier.CustomDomain, &tier.PrioritySupport, &tier.PriceCentsMonthly,
		&featuresJSON, &tier.CreatedAt, &tier.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("tier not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get tier: %w", err)
	}

	if len(featuresJSON) > 0 {
		if err := json.Unmarshal(featuresJSON, &tier.Features); err != nil {
			return nil, fmt.Errorf("failed to unmarshal tier features: %w", err)
		}
	}

	return tier, nil
}

// ListTiers retrieves all available tiers
func (r *tenantRepository) ListTiers(ctx context.Context) ([]*models.SubscriptionTier, error) {
	query := `
		SELECT id, name, display_name, tier_level, storage_quota_bytes, max_users,
			   max_courses, api_rate_limit, dedicated_database, custom_domain,
			   priority_support, price_cents_monthly, features, created_at, updated_at
		FROM subscription_tiers
		ORDER BY tier_level ASC
	`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to list tiers: %w", err)
	}
	defer rows.Close()

	var tiers []*models.SubscriptionTier
	for rows.Next() {
		tier := &models.SubscriptionTier{}
		var featuresJSON []byte

		err := rows.Scan(
			&tier.ID, &tier.Name, &tier.DisplayName, &tier.TierLevel, &tier.StorageQuotaBytes,
			&tier.MaxUsers, &tier.MaxCourses, &tier.APIRateLimit, &tier.DedicatedDatabase,
			&tier.CustomDomain, &tier.PrioritySupport, &tier.PriceCentsMonthly,
			&featuresJSON, &tier.CreatedAt, &tier.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan tier: %w", err)
		}

		if len(featuresJSON) > 0 {
			if err := json.Unmarshal(featuresJSON, &tier.Features); err != nil {
				return nil, fmt.Errorf("failed to unmarshal tier features: %w", err)
			}
		}

		tiers = append(tiers, tier)
	}

	return tiers, nil
}

// CreateAdmin creates a new tenant admin
func (r *tenantRepository) CreateAdmin(ctx context.Context, admin *models.TenantAdmin) error {
	query := `
		INSERT INTO tenant_admins (id, tenant_id, user_id, email, first_name, last_name, is_primary, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`

	_, err := r.db.ExecContext(ctx, query,
		admin.ID, admin.TenantID, admin.UserID, admin.Email,
		admin.FirstName, admin.LastName, admin.IsPrimary, admin.CreatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to create admin: %w", err)
	}

	return nil
}

// GetAdminByID retrieves an admin by tenant ID and admin ID
func (r *tenantRepository) GetAdminByID(ctx context.Context, tenantID, adminID string) (*models.TenantAdmin, error) {
	query := `
		SELECT id, tenant_id, user_id, email, first_name, last_name, is_primary, created_at
		FROM tenant_admins
		WHERE tenant_id = $1 AND id = $2
	`

	admin := &models.TenantAdmin{}
	err := r.db.QueryRowContext(ctx, query, tenantID, adminID).Scan(
		&admin.ID, &admin.TenantID, &admin.UserID, &admin.Email,
		&admin.FirstName, &admin.LastName, &admin.IsPrimary, &admin.CreatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("admin not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get admin: %w", err)
	}

	return admin, nil
}

// GetAdminsByTenant retrieves all admins for a tenant
func (r *tenantRepository) GetAdminsByTenant(ctx context.Context, tenantID string) ([]*models.TenantAdmin, error) {
	query := `
		SELECT id, tenant_id, user_id, email, first_name, last_name, is_primary, created_at
		FROM tenant_admins
		WHERE tenant_id = $1
		ORDER BY is_primary DESC, created_at ASC
	`

	rows, err := r.db.QueryContext(ctx, query, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to list admins: %w", err)
	}
	defer rows.Close()

	var admins []*models.TenantAdmin
	for rows.Next() {
		admin := &models.TenantAdmin{}
		err := rows.Scan(
			&admin.ID, &admin.TenantID, &admin.UserID, &admin.Email,
			&admin.FirstName, &admin.LastName, &admin.IsPrimary, &admin.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan admin: %w", err)
		}
		admins = append(admins, admin)
	}

	return admins, nil
}

// CreateProvisioning creates a new provisioning record
func (r *tenantRepository) CreateProvisioning(ctx context.Context, provisioning *models.TenantProvisioning) error {
	query := `
		INSERT INTO tenant_provisioning (id, tenant_id, status, current_step, progress_percentage, started_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`

	_, err := r.db.ExecContext(ctx, query,
		provisioning.ID, provisioning.TenantID, provisioning.Status,
		provisioning.CurrentStep, provisioning.ProgressPercentage, provisioning.StartedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to create provisioning: %w", err)
	}

	return nil
}

// UpdateProvisioning updates a provisioning record
func (r *tenantRepository) UpdateProvisioning(ctx context.Context, provisioning *models.TenantProvisioning) error {
	query := `
		UPDATE tenant_provisioning
		SET status = $1, current_step = $2, progress_percentage = $3,
			error_message = $4, completed_at = $5, duration_seconds = $6
		WHERE id = $7
	`

	_, err := r.db.ExecContext(ctx, query,
		provisioning.Status, provisioning.CurrentStep, provisioning.ProgressPercentage,
		provisioning.ErrorMessage, provisioning.CompletedAt, provisioning.DurationSeconds,
		provisioning.ID,
	)

	if err != nil {
		return fmt.Errorf("failed to update provisioning: %w", err)
	}

	return nil
}

// GetProvisioningByID retrieves a provisioning record by ID
func (r *tenantRepository) GetProvisioningByID(ctx context.Context, provisioningID string) (*models.TenantProvisioning, error) {
	query := `
		SELECT id, tenant_id, status, current_step, progress_percentage,
			   error_message, started_at, completed_at, duration_seconds
		FROM tenant_provisioning
		WHERE id = $1
	`

	provisioning := &models.TenantProvisioning{}
	err := r.db.QueryRowContext(ctx, query, provisioningID).Scan(
		&provisioning.ID, &provisioning.TenantID, &provisioning.Status,
		&provisioning.CurrentStep, &provisioning.ProgressPercentage,
		&provisioning.ErrorMessage, &provisioning.StartedAt,
		&provisioning.CompletedAt, &provisioning.DurationSeconds,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("provisioning record not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get provisioning: %w", err)
	}

	return provisioning, nil
}

// CreateSetupToken creates a new setup token
func (r *tenantRepository) CreateSetupToken(ctx context.Context, token *models.TenantSetupToken) error {
	query := `
		INSERT INTO tenant_setup_tokens (id, tenant_id, admin_id, token, expires_at, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`

	_, err := r.db.ExecContext(ctx, query,
		token.ID, token.TenantID, token.AdminID, token.Token,
		token.ExpiresAt, token.CreatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to create setup token: %w", err)
	}

	return nil
}

// GetSetupTokenByToken retrieves a setup token by token string
func (r *tenantRepository) GetSetupTokenByToken(ctx context.Context, token string) (*models.TenantSetupToken, error) {
	query := `
		SELECT id, tenant_id, admin_id, token, expires_at, used_at, created_at
		FROM tenant_setup_tokens
		WHERE token = $1
	`

	setupToken := &models.TenantSetupToken{}
	err := r.db.QueryRowContext(ctx, query, token).Scan(
		&setupToken.ID, &setupToken.TenantID, &setupToken.AdminID,
		&setupToken.Token, &setupToken.ExpiresAt, &setupToken.UsedAt, &setupToken.CreatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("setup token not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get setup token: %w", err)
	}

	return setupToken, nil
}

// MarkSetupTokenUsed marks a setup token as used
func (r *tenantRepository) MarkSetupTokenUsed(ctx context.Context, tokenID string) error {
	query := `UPDATE tenant_setup_tokens SET used_at = $1 WHERE id = $2`
	_, err := r.db.ExecContext(ctx, query, time.Now(), tokenID)
	if err != nil {
		return fmt.Errorf("failed to mark token as used: %w", err)
	}
	return nil
}

// UpdateStorageUsage updates storage usage for a tenant
func (r *tenantRepository) UpdateStorageUsage(ctx context.Context, tenantID string, bytesDelta int64, fileCountDelta int32) error {
	query := `
		UPDATE tenants
		SET storage_used_bytes = storage_used_bytes + $1,
			file_count = file_count + $2,
			updated_at = $3
		WHERE id = $4
	`

	_, err := r.db.ExecContext(ctx, query, bytesDelta, fileCountDelta, time.Now(), tenantID)
	if err != nil {
		return fmt.Errorf("failed to update storage usage: %w", err)
	}

	return nil
}

// GetStorageQuota retrieves storage quota information for a tenant
func (r *tenantRepository) GetStorageQuota(ctx context.Context, tenantID string) (int64, int64, int32, error) {
	query := `
		SELECT storage_quota_bytes, storage_used_bytes, file_count
		FROM tenants
		WHERE id = $1
	`

	var quota, used int64
	var fileCount int32

	err := r.db.QueryRowContext(ctx, query, tenantID).Scan(&quota, &used, &fileCount)
	if err == sql.ErrNoRows {
		return 0, 0, 0, fmt.Errorf("tenant not found")
	}
	if err != nil {
		return 0, 0, 0, fmt.Errorf("failed to get storage quota: %w", err)
	}

	return quota, used, fileCount, nil
}
