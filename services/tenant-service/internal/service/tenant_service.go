package service

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"
	"time"

	"slate/services/tenant-service/internal/models"
	"slate/services/tenant-service/internal/repository"
	"slate/services/tenant-service/pkg/metrics"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type TenantService interface {
	// CreateTenant provisions a new tenant (AC1, AC2, AC3, AC4, AC5, AC6, AC7)
	CreateTenant(ctx context.Context, req *models.CreateTenantRequest) (*models.CreateTenantResponse, error)
	GetTenant(ctx context.Context, tenantID string) (*models.Tenant, error)
	UpdateTenant(ctx context.Context, tenantID string, updates map[string]interface{}) (*models.Tenant, error)
	DeleteTenant(ctx context.Context, tenantID string, forceDelete bool) error
	ListTenants(ctx context.Context, page, pageSize int32, filters map[string]interface{}) ([]*models.Tenant, int32, error)

	// Provisioning status
	GetProvisioningStatus(ctx context.Context, provisioningID string) (*models.TenantProvisioning, error)

	// Storage quota (AC6)
	GetStorageQuota(ctx context.Context, tenantID string) (int64, int64, int32, error)
	UpdateStorageUsage(ctx context.Context, tenantID string, bytesDelta int64, fileCountDelta int32) error

	// Tenant admin (AC3)
	CreateTenantAdmin(ctx context.Context, tenantID, email, firstName, lastName, password string) (*models.TenantAdmin, error)
	GetTenantAdmin(ctx context.Context, tenantID, adminID string) (*models.TenantAdmin, error)
}

type tenantService struct {
	repo              repository.TenantRepository
	userServiceClient UserServiceClient
	emailServiceClient EmailServiceClient
	metricsCollector  *metrics.MetricsCollector
	baseSetupURL      string
}

// UserServiceClient interface for creating users
type UserServiceClient interface {
	CreateUser(ctx context.Context, email, password, firstName, lastName string, roles []string) (string, error)
}

// EmailServiceClient interface for sending emails
type EmailServiceClient interface {
	SendWelcomeEmail(ctx context.Context, tenantName, adminEmail, adminName, setupURL, subdomain, tier string) error
}

func NewTenantService(
	repo repository.TenantRepository,
	userClient UserServiceClient,
	emailClient EmailServiceClient,
	metricsCollector *metrics.MetricsCollector,
	baseSetupURL string,
) TenantService {
	return &tenantService{
		repo:              repo,
		userServiceClient: userClient,
		emailServiceClient: emailClient,
		metricsCollector:  metricsCollector,
		baseSetupURL:      baseSetupURL,
	}
}

// CreateTenant provisions a new tenant (AC1-AC7)
func (s *tenantService) CreateTenant(ctx context.Context, req *models.CreateTenantRequest) (*models.CreateTenantResponse, error) {
	startTime := time.Now()

	// Record provisioning metric
	defer func() {
		duration := time.Since(startTime).Seconds()
		s.metricsCollector.RecordProvisioningDuration(duration)
	}()

	// Validate provisioning time (AC7: Must complete within 2 minutes)
	ctx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()

	// AC7: Track provisioning status
	provisioningID := uuid.New().String()
	provisioning := &models.TenantProvisioning{
		ID:                 provisioningID,
		Status:             models.StatusPending.String(),
		ProgressPercentage: 0,
		StartedAt:          time.Now(),
	}

	// AC1: Get tier information
	tier, err := s.getTierFromRequest(ctx, req.Tier)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid tier: %v", err)
	}

	// AC1: Create tenant record
	tenant := &models.Tenant{
		ID:                uuid.New().String(),
		Name:              req.Name,
		Domain:            req.Domain, // AC5: Custom subdomain
		TierID:            tier.ID,
		IsActive:          true,
		StorageQuotaBytes: tier.StorageQuotaBytes, // AC6: Storage quota set by tier
		StorageUsedBytes:  0,
		FileCount:         0,
		UserCount:         0,
		CourseCount:       0,
		CreatedAt:         time.Now(),
		UpdatedAt:         time.Now(),
	}

	provisioning.TenantID = tenant.ID

	// Create tenant in database
	if err := s.repo.Create(ctx, tenant); err != nil {
		s.metricsCollector.IncProvisioningErrors("create_tenant_failed")
		return nil, status.Errorf(codes.Internal, "failed to create tenant: %v", err)
	}

	// Create provisioning record
	if err := s.repo.CreateProvisioning(ctx, provisioning); err != nil {
		s.metricsCollector.IncProvisioningErrors("create_provisioning_failed")
		return nil, status.Errorf(codes.Internal, "failed to create provisioning record: %v", err)
	}

	// Start async provisioning
	go s.provisionTenantAsync(tenant, tier, req, provisioningID)

	// Increment successful provisioning counter
	s.metricsCollector.IncProvisioningTotal("initiated")

	return &models.CreateTenantResponse{
		Tenant:         tenant,
		ProvisioningID: provisioningID,
		Status:         models.StatusPending,
	}, nil
}

// provisionTenantAsync handles async tenant provisioning (AC2, AC3, AC4, AC6, AC7)
func (s *tenantService) provisionTenantAsync(
	tenant *models.Tenant,
	tier *models.SubscriptionTier,
	req *models.CreateTenantRequest,
	provisioningID string,
) {
	ctx := context.Background()
	startTime := time.Now()

	// Update provisioning progress
	updateProgress := func(status models.ProvisioningStatus, progress int32, errorMsg string) {
		provisioning, err := s.repo.GetProvisioningByID(ctx, provisioningID)
		if err != nil {
			return
		}

		provisioning.Status = status.String()
		provisioning.ProgressPercentage = progress
		if errorMsg != "" {
			provisioning.ErrorMessage = sql.NullString{String: errorMsg, Valid: true}
		}

		if status == models.StatusCompleted || status == models.StatusFailed {
			now := time.Now()
			provisioning.CompletedAt = sql.NullTime{Time: now, Valid: true}
			duration := int32(now.Sub(startTime).Seconds())
			provisioning.DurationSeconds = sql.NullInt32{Int32: duration, Valid: true}

			// AC7: Check if provisioning completed within 2 minutes
			if duration > 120 {
				s.metricsCollector.IncProvisioningErrors("timeout_exceeded")
			}
		}

		if err := s.repo.UpdateProvisioning(ctx, provisioning); err != nil {
		// Log error but don't fail - this is a progress update
		_ = err
	}
	}

	// AC2: Provision dedicated database (if Professional+ tier)
	if tier.DedicatedDatabase {
		updateProgress(models.StatusProvisioningDB, 20, "")

		// In production, this would create a new database
		// For now, we'll simulate this step
		time.Sleep(2 * time.Second)

		tenant.DatabaseName = sql.NullString{String: fmt.Sprintf("tenant_%s", tenant.ID), Valid: true}
		tenant.DatabaseHost = sql.NullString{String: "postgres", Valid: true}
		tenant.DatabasePort = sql.NullInt32{Int32: 5432, Valid: true}

		if err := s.repo.Update(ctx, tenant); err != nil {
			updateProgress(models.StatusFailed, 20, fmt.Sprintf("database provisioning failed: %v", err))
			s.metricsCollector.IncProvisioningErrors("database_provisioning_failed")
			return
		}
	}

	// AC3: Create default admin account
	updateProgress(models.StatusCreatingAdmin, 40, "")

	// Create user in user-auth-service
	userID, err := s.userServiceClient.CreateUser(
		ctx,
		req.AdminEmail,
		req.AdminPassword,
		req.AdminFirstName,
		req.AdminLastName,
		[]string{"admin", "tenant_admin"},
	)
	if err != nil {
		updateProgress(models.StatusFailed, 40, fmt.Sprintf("admin creation failed: %v", err))
		s.metricsCollector.IncProvisioningErrors("admin_creation_failed")
		return
	}

	// Create tenant admin record
	admin := &models.TenantAdmin{
		ID:        uuid.New().String(),
		TenantID:  tenant.ID,
		UserID:    userID,
		Email:     req.AdminEmail,
		FirstName: sql.NullString{String: req.AdminFirstName, Valid: true},
		LastName:  sql.NullString{String: req.AdminLastName, Valid: true},
		IsPrimary: true,
		CreatedAt: time.Now(),
	}

	if err := s.repo.CreateAdmin(ctx, admin); err != nil {
		updateProgress(models.StatusFailed, 40, fmt.Sprintf("admin record creation failed: %v", err))
		s.metricsCollector.IncProvisioningErrors("admin_record_failed")
		return
	}

	// AC6: Storage quota is already set by tier during tenant creation
	updateProgress(models.StatusSettingQuota, 60, "")

	// AC4: Create setup token and send welcome email
	updateProgress(models.StatusSendingEmail, 80, "")

	setupToken, err := s.createSetupToken(ctx, tenant.ID, admin.ID)
	if err != nil {
		updateProgress(models.StatusFailed, 80, fmt.Sprintf("setup token creation failed: %v", err))
		s.metricsCollector.IncProvisioningErrors("setup_token_failed")
		return
	}

	// AC4: Setup URL with token
	setupURL := fmt.Sprintf("%s/setup?token=%s", s.baseSetupURL, setupToken)

	// AC4: Send welcome email with setup link
	err = s.emailServiceClient.SendWelcomeEmail(
		ctx,
		tenant.Name,
		admin.Email,
		fmt.Sprintf("%s %s", req.AdminFirstName, req.AdminLastName),
		setupURL,
		tenant.Domain, // AC5: Custom subdomain
		tier.DisplayName,
	)
	if err != nil {
		// Don't fail provisioning if email fails, just log
		updateProgress(models.StatusCompleted, 100, fmt.Sprintf("provisioning completed but email failed: %v", err))
		s.metricsCollector.IncProvisioningErrors("email_send_failed")
		return
	}

	// AC7: Provisioning completed
	updateProgress(models.StatusCompleted, 100, "")
	s.metricsCollector.IncProvisioningTotal("completed")
}

// createSetupToken creates a setup token for tenant onboarding
func (s *tenantService) createSetupToken(ctx context.Context, tenantID, adminID string) (string, error) {
	// Generate random token
	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		return "", err
	}
	token := hex.EncodeToString(tokenBytes)

	setupToken := &models.TenantSetupToken{
		ID:        uuid.New().String(),
		TenantID:  tenantID,
		AdminID:   adminID,
		Token:     token,
		ExpiresAt: time.Now().Add(7 * 24 * time.Hour), // 7 days
		CreatedAt: time.Now(),
	}

	if err := s.repo.CreateSetupToken(ctx, setupToken); err != nil {
		return "", err
	}

	return token, nil
}

// getTierFromRequest converts tier enum to tier model
func (s *tenantService) getTierFromRequest(ctx context.Context, tierEnum models.TenantTier) (*models.SubscriptionTier, error) {
	tierName := tierEnum.String()
	return s.repo.GetTierByName(ctx, tierName)
}

// GetTenant retrieves a tenant by ID
func (s *tenantService) GetTenant(ctx context.Context, tenantID string) (*models.Tenant, error) {
	return s.repo.GetByID(ctx, tenantID)
}

// UpdateTenant updates a tenant
func (s *tenantService) UpdateTenant(ctx context.Context, tenantID string, updates map[string]interface{}) (*models.Tenant, error) {
	tenant, err := s.repo.GetByID(ctx, tenantID)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "tenant not found: %v", err)
	}

	// Apply updates
	if name, ok := updates["name"].(string); ok {
		tenant.Name = name
	}
	if domain, ok := updates["domain"].(string); ok {
		tenant.Domain = domain
	}
	if tierID, ok := updates["tier_id"].(string); ok {
		tenant.TierID = tierID
	}
	if isActive, ok := updates["is_active"].(bool); ok {
		tenant.IsActive = isActive
	}

	if err := s.repo.Update(ctx, tenant); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to update tenant: %v", err)
	}

	return tenant, nil
}

// DeleteTenant deletes a tenant
func (s *tenantService) DeleteTenant(ctx context.Context, tenantID string, forceDelete bool) error {
	tenant, err := s.repo.GetByID(ctx, tenantID)
	if err != nil {
		return status.Errorf(codes.NotFound, "tenant not found: %v", err)
	}

	if !forceDelete && (tenant.UserCount > 0 || tenant.CourseCount > 0) {
		return status.Errorf(codes.FailedPrecondition, "cannot delete tenant with active users or courses")
	}

	return s.repo.Delete(ctx, tenantID)
}

// ListTenants lists tenants with pagination and filters
func (s *tenantService) ListTenants(ctx context.Context, page, pageSize int32, filters map[string]interface{}) ([]*models.Tenant, int32, error) {
	return s.repo.List(ctx, page, pageSize, filters)
}

// GetProvisioningStatus retrieves provisioning status
func (s *tenantService) GetProvisioningStatus(ctx context.Context, provisioningID string) (*models.TenantProvisioning, error) {
	return s.repo.GetProvisioningByID(ctx, provisioningID)
}

// GetStorageQuota retrieves storage quota for a tenant (AC6)
func (s *tenantService) GetStorageQuota(ctx context.Context, tenantID string) (int64, int64, int32, error) {
	return s.repo.GetStorageQuota(ctx, tenantID)
}

// UpdateStorageUsage updates storage usage for a tenant (AC6)
func (s *tenantService) UpdateStorageUsage(ctx context.Context, tenantID string, bytesDelta int64, fileCountDelta int32) error {
	// Check quota before update
	quota, used, _, err := s.repo.GetStorageQuota(ctx, tenantID)
	if err != nil {
		return status.Errorf(codes.Internal, "failed to get storage quota: %v", err)
	}

	// AC6: Enforce storage quota
	if used+bytesDelta > quota && bytesDelta > 0 {
		return status.Errorf(codes.ResourceExhausted, "storage quota exceeded: %d/%d bytes used", used, quota)
	}

	return s.repo.UpdateStorageUsage(ctx, tenantID, bytesDelta, fileCountDelta)
}

// CreateTenantAdmin creates a new tenant admin (AC3)
func (s *tenantService) CreateTenantAdmin(ctx context.Context, tenantID, email, firstName, lastName, password string) (*models.TenantAdmin, error) {
	// Verify tenant exists
	_, err := s.repo.GetByID(ctx, tenantID)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "tenant not found: %v", err)
	}

	// Create user in user-auth-service
	userID, err := s.userServiceClient.CreateUser(ctx, email, password, firstName, lastName, []string{"tenant_admin"})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create user: %v", err)
	}

	// Create tenant admin record
	admin := &models.TenantAdmin{
		ID:        uuid.New().String(),
		TenantID:  tenantID,
		UserID:    userID,
		Email:     email,
		FirstName: sql.NullString{String: firstName, Valid: true},
		LastName:  sql.NullString{String: lastName, Valid: true},
		IsPrimary: false,
		CreatedAt: time.Now(),
	}

	if err := s.repo.CreateAdmin(ctx, admin); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create admin: %v", err)
	}

	return admin, nil
}

// GetTenantAdmin retrieves a tenant admin
func (s *tenantService) GetTenantAdmin(ctx context.Context, tenantID, adminID string) (*models.TenantAdmin, error) {
	return s.repo.GetAdminByID(ctx, tenantID, adminID)
}
