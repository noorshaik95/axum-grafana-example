package service

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"slate/services/tenant-service/internal/models"
	"slate/services/tenant-service/pkg/metrics"
)

// Mock repository
type mockTenantRepository struct {
	tenants     map[string]*models.Tenant
	tiers       map[string]*models.SubscriptionTier
	admins      map[string]*models.TenantAdmin
	provisioning map[string]*models.TenantProvisioning
	setupTokens map[string]*models.TenantSetupToken
}

func newMockRepository() *mockTenantRepository {
	return &mockTenantRepository{
		tenants:     make(map[string]*models.Tenant),
		tiers:       make(map[string]*models.SubscriptionTier),
		admins:      make(map[string]*models.TenantAdmin),
		provisioning: make(map[string]*models.TenantProvisioning),
		setupTokens: make(map[string]*models.TenantSetupToken),
	}
}

func (m *mockTenantRepository) Create(ctx context.Context, tenant *models.Tenant) error {
	if _, exists := m.tenants[tenant.ID]; exists {
		return errors.New("tenant already exists")
	}
	m.tenants[tenant.ID] = tenant
	return nil
}

func (m *mockTenantRepository) GetByID(ctx context.Context, id string) (*models.Tenant, error) {
	tenant, exists := m.tenants[id]
	if !exists {
		return nil, errors.New("tenant not found")
	}
	return tenant, nil
}

func (m *mockTenantRepository) GetByDomain(ctx context.Context, domain string) (*models.Tenant, error) {
	for _, tenant := range m.tenants {
		if tenant.Domain == domain {
			return tenant, nil
		}
	}
	return nil, errors.New("tenant not found")
}

func (m *mockTenantRepository) Update(ctx context.Context, tenant *models.Tenant) error {
	if _, exists := m.tenants[tenant.ID]; !exists {
		return errors.New("tenant not found")
	}
	m.tenants[tenant.ID] = tenant
	return nil
}

func (m *mockTenantRepository) Delete(ctx context.Context, id string) error {
	delete(m.tenants, id)
	return nil
}

func (m *mockTenantRepository) List(ctx context.Context, page, pageSize int32, filters map[string]interface{}) ([]*models.Tenant, int32, error) {
	var result []*models.Tenant
	for _, tenant := range m.tenants {
		result = append(result, tenant)
	}
	return result, int32(len(result)), nil
}

func (m *mockTenantRepository) GetTierByID(ctx context.Context, tierID string) (*models.SubscriptionTier, error) {
	tier, exists := m.tiers[tierID]
	if !exists {
		return nil, errors.New("tier not found")
	}
	return tier, nil
}

func (m *mockTenantRepository) GetTierByName(ctx context.Context, name string) (*models.SubscriptionTier, error) {
	for _, tier := range m.tiers {
		if tier.Name == name {
			return tier, nil
		}
	}
	return nil, errors.New("tier not found")
}

func (m *mockTenantRepository) ListTiers(ctx context.Context) ([]*models.SubscriptionTier, error) {
	var result []*models.SubscriptionTier
	for _, tier := range m.tiers {
		result = append(result, tier)
	}
	return result, nil
}

func (m *mockTenantRepository) CreateAdmin(ctx context.Context, admin *models.TenantAdmin) error {
	m.admins[admin.ID] = admin
	return nil
}

func (m *mockTenantRepository) GetAdminByID(ctx context.Context, tenantID, adminID string) (*models.TenantAdmin, error) {
	admin, exists := m.admins[adminID]
	if !exists || admin.TenantID != tenantID {
		return nil, errors.New("admin not found")
	}
	return admin, nil
}

func (m *mockTenantRepository) GetAdminsByTenant(ctx context.Context, tenantID string) ([]*models.TenantAdmin, error) {
	var result []*models.TenantAdmin
	for _, admin := range m.admins {
		if admin.TenantID == tenantID {
			result = append(result, admin)
		}
	}
	return result, nil
}

func (m *mockTenantRepository) CreateProvisioning(ctx context.Context, provisioning *models.TenantProvisioning) error {
	m.provisioning[provisioning.ID] = provisioning
	return nil
}

func (m *mockTenantRepository) UpdateProvisioning(ctx context.Context, provisioning *models.TenantProvisioning) error {
	m.provisioning[provisioning.ID] = provisioning
	return nil
}

func (m *mockTenantRepository) GetProvisioningByID(ctx context.Context, provisioningID string) (*models.TenantProvisioning, error) {
	prov, exists := m.provisioning[provisioningID]
	if !exists {
		return nil, errors.New("provisioning not found")
	}
	return prov, nil
}

func (m *mockTenantRepository) CreateSetupToken(ctx context.Context, token *models.TenantSetupToken) error {
	m.setupTokens[token.ID] = token
	return nil
}

func (m *mockTenantRepository) GetSetupTokenByToken(ctx context.Context, token string) (*models.TenantSetupToken, error) {
	for _, setupToken := range m.setupTokens {
		if setupToken.Token == token {
			return setupToken, nil
		}
	}
	return nil, errors.New("token not found")
}

func (m *mockTenantRepository) MarkSetupTokenUsed(ctx context.Context, tokenID string) error {
	if token, exists := m.setupTokens[tokenID]; exists {
		token.UsedAt = sql.NullTime{Time: time.Now(), Valid: true}
		return nil
	}
	return errors.New("token not found")
}

func (m *mockTenantRepository) UpdateStorageUsage(ctx context.Context, tenantID string, bytesDelta int64, fileCountDelta int32) error {
	tenant, exists := m.tenants[tenantID]
	if !exists {
		return errors.New("tenant not found")
	}
	tenant.StorageUsedBytes += bytesDelta
	tenant.FileCount += fileCountDelta
	return nil
}

func (m *mockTenantRepository) GetStorageQuota(ctx context.Context, tenantID string) (int64, int64, int32, error) {
	tenant, exists := m.tenants[tenantID]
	if !exists {
		return 0, 0, 0, errors.New("tenant not found")
	}
	return tenant.StorageQuotaBytes, tenant.StorageUsedBytes, tenant.FileCount, nil
}

// Mock user service client
type mockUserServiceClient struct {
	shouldFail bool
	createdUsers []string
}

func (m *mockUserServiceClient) CreateUser(ctx context.Context, email, password, firstName, lastName string, roles []string) (string, error) {
	if m.shouldFail {
		return "", errors.New("user service unavailable")
	}
	userID := "user_" + email
	m.createdUsers = append(m.createdUsers, userID)
	return userID, nil
}

// Mock email service client
type mockEmailServiceClient struct {
	shouldFail bool
	sentEmails []string
}

func (m *mockEmailServiceClient) SendWelcomeEmail(ctx context.Context, tenantName, adminEmail, adminName, setupURL, subdomain, tier string) error {
	if m.shouldFail {
		return errors.New("email service unavailable")
	}
	m.sentEmails = append(m.sentEmails, adminEmail)
	return nil
}

// Tests

func TestCreateTenant_Success(t *testing.T) {
	// Setup
	repo := newMockRepository()
	userClient := &mockUserServiceClient{}
	emailClient := &mockEmailServiceClient{}
	metricsCollector := metrics.NewMetricsCollectorWithRegistry(prometheus.NewRegistry())

	// Add test tier
	repo.tiers["tier-free-001"] = &models.SubscriptionTier{
		ID:                "tier-free-001",
		Name:              "free",
		DisplayName:       "Free",
		StorageQuotaBytes: 1073741824, // 1GB
		DedicatedDatabase: false,
	}

	service := NewTenantService(repo, userClient, emailClient, metricsCollector, "http://localhost:8080")

	// Test AC1: Create tenant
	req := &models.CreateTenantRequest{
		Name:           "Test Tenant",
		Domain:         "test.slate.local",
		Tier:           models.TierFree,
		AdminEmail:     "admin@test.com",
		AdminFirstName: "Admin",
		AdminLastName:  "User",
		AdminPassword:  "SecurePass123!",
	}

	resp, err := service.CreateTenant(context.Background(), req)

	// Assertions
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if resp.Tenant == nil {
		t.Fatal("Expected tenant to be created")
	}

	if resp.Tenant.Name != req.Name {
		t.Errorf("Expected tenant name %s, got %s", req.Name, resp.Tenant.Name)
	}

	if resp.Tenant.Domain != req.Domain {
		t.Errorf("Expected domain %s, got %s", req.Domain, resp.Tenant.Domain)
	}

	// AC6: Storage quota set by tier
	if resp.Tenant.StorageQuotaBytes != 1073741824 {
		t.Errorf("Expected storage quota 1073741824, got %d", resp.Tenant.StorageQuotaBytes)
	}

	if resp.ProvisioningID == "" {
		t.Error("Expected provisioning ID to be set")
	}
}

func TestCreateTenant_ProfessionalTier(t *testing.T) {
	// Setup
	repo := newMockRepository()
	userClient := &mockUserServiceClient{}
	emailClient := &mockEmailServiceClient{}
	metricsCollector := metrics.NewMetricsCollectorWithRegistry(prometheus.NewRegistry())

	// Add Professional tier (AC2: Dedicated database)
	repo.tiers["tier-pro-001"] = &models.SubscriptionTier{
		ID:                "tier-pro-001",
		Name:              "professional",
		DisplayName:       "Professional",
		StorageQuotaBytes: 107374182400, // 100GB
		DedicatedDatabase: true,          // AC2
	}

	service := NewTenantService(repo, userClient, emailClient, metricsCollector, "http://localhost:8080")

	req := &models.CreateTenantRequest{
		Name:           "Pro Tenant",
		Domain:         "pro.slate.local",
		Tier:           models.TierProfessional,
		AdminEmail:     "admin@pro.com",
		AdminFirstName: "Pro",
		AdminLastName:  "Admin",
		AdminPassword:  "SecurePass123!",
	}

	resp, err := service.CreateTenant(context.Background(), req)

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// AC2: Professional tier should have larger quota
	if resp.Tenant.StorageQuotaBytes != 107374182400 {
		t.Errorf("Expected storage quota 107374182400, got %d", resp.Tenant.StorageQuotaBytes)
	}
}

func TestGetStorageQuota_Success(t *testing.T) {
	// Setup
	repo := newMockRepository()
	service := NewTenantService(repo, nil, nil, metrics.NewMetricsCollectorWithRegistry(prometheus.NewRegistry()), "")

	// Create tenant
	tenant := &models.Tenant{
		ID:                "tenant-1",
		StorageQuotaBytes: 1073741824, // 1GB
		StorageUsedBytes:  536870912,  // 512MB
		FileCount:         100,
	}
	repo.tenants[tenant.ID] = tenant

	// Test AC6: Get storage quota
	quota, used, files, err := service.GetStorageQuota(context.Background(), tenant.ID)

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if quota != 1073741824 {
		t.Errorf("Expected quota 1073741824, got %d", quota)
	}

	if used != 536870912 {
		t.Errorf("Expected used 536870912, got %d", used)
	}

	if files != 100 {
		t.Errorf("Expected 100 files, got %d", files)
	}
}

func TestUpdateStorageUsage_WithinQuota(t *testing.T) {
	// Setup
	repo := newMockRepository()
	service := NewTenantService(repo, nil, nil, metrics.NewMetricsCollectorWithRegistry(prometheus.NewRegistry()), "")

	tenant := &models.Tenant{
		ID:                "tenant-1",
		StorageQuotaBytes: 1073741824, // 1GB
		StorageUsedBytes:  536870912,  // 512MB
		FileCount:         100,
	}
	repo.tenants[tenant.ID] = tenant

	// Test AC6: Update storage usage (within quota)
	err := service.UpdateStorageUsage(context.Background(), tenant.ID, 104857600, 10) // +100MB, +10 files

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Verify update
	_, used, files, _ := service.GetStorageQuota(context.Background(), tenant.ID)
	if used != 641728512 { // 512MB + 100MB
		t.Errorf("Expected used 641728512, got %d", used)
	}

	if files != 110 {
		t.Errorf("Expected 110 files, got %d", files)
	}
}

func TestUpdateStorageUsage_ExceedsQuota(t *testing.T) {
	// Setup
	repo := newMockRepository()
	service := NewTenantService(repo, nil, nil, metrics.NewMetricsCollectorWithRegistry(prometheus.NewRegistry()), "")

	tenant := &models.Tenant{
		ID:                "tenant-1",
		StorageQuotaBytes: 1073741824, // 1GB
		StorageUsedBytes:  1000000000, // 953MB
		FileCount:         100,
	}
	repo.tenants[tenant.ID] = tenant

	// Test AC6: Update storage usage (exceeds quota)
	err := service.UpdateStorageUsage(context.Background(), tenant.ID, 200000000, 10) // +190MB (would exceed)

	if err == nil {
		t.Fatal("Expected error for quota exceeded, got nil")
	}

	// Verify usage didn't change
	_, used, _, _ := service.GetStorageQuota(context.Background(), tenant.ID)
	if used != 1000000000 {
		t.Errorf("Expected usage unchanged at 1000000000, got %d", used)
	}
}

func TestListTenants_WithFilters(t *testing.T) {
	// Setup
	repo := newMockRepository()
	service := NewTenantService(repo, nil, nil, metrics.NewMetricsCollectorWithRegistry(prometheus.NewRegistry()), "")

	// Create multiple tenants
	tenants := []*models.Tenant{
		{ID: "t1", Name: "Tenant 1", IsActive: true},
		{ID: "t2", Name: "Tenant 2", IsActive: true},
		{ID: "t3", Name: "Tenant 3", IsActive: false},
	}

	for _, tenant := range tenants {
		repo.tenants[tenant.ID] = tenant
	}

	// Test listing
	result, total, err := service.ListTenants(context.Background(), 1, 10, map[string]interface{}{})

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if total != 3 {
		t.Errorf("Expected total 3, got %d", total)
	}

	if len(result) != 3 {
		t.Errorf("Expected 3 tenants, got %d", len(result))
	}
}

func TestDeleteTenant_WithActiveUsers(t *testing.T) {
	// Setup
	repo := newMockRepository()
	service := NewTenantService(repo, nil, nil, metrics.NewMetricsCollectorWithRegistry(prometheus.NewRegistry()), "")

	tenant := &models.Tenant{
		ID:         "tenant-1",
		Name:       "Test Tenant",
		UserCount:  10,
		IsActive:   true,
	}
	repo.tenants[tenant.ID] = tenant

	// Test delete without force (should fail)
	err := service.DeleteTenant(context.Background(), tenant.ID, false)

	if err == nil {
		t.Fatal("Expected error when deleting tenant with active users, got nil")
	}

	// Verify tenant still exists
	if _, exists := repo.tenants[tenant.ID]; !exists {
		t.Error("Tenant should not be deleted")
	}
}

func TestDeleteTenant_ForceDelete(t *testing.T) {
	// Setup
	repo := newMockRepository()
	service := NewTenantService(repo, nil, nil, metrics.NewMetricsCollectorWithRegistry(prometheus.NewRegistry()), "")

	tenant := &models.Tenant{
		ID:         "tenant-1",
		Name:       "Test Tenant",
		UserCount:  10,
		IsActive:   true,
	}
	repo.tenants[tenant.ID] = tenant

	// Test force delete
	err := service.DeleteTenant(context.Background(), tenant.ID, true)

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Verify tenant deleted
	if _, exists := repo.tenants[tenant.ID]; exists {
		t.Error("Tenant should be deleted")
	}
}

func TestUpdateTenant_Success(t *testing.T) {
	// Setup
	repo := newMockRepository()
	service := NewTenantService(repo, nil, nil, metrics.NewMetricsCollectorWithRegistry(prometheus.NewRegistry()), "")

	tenant := &models.Tenant{
		ID:       "tenant-1",
		Name:     "Old Name",
		Domain:   "old.domain.com",
		IsActive: true,
	}
	repo.tenants[tenant.ID] = tenant

	// Test update
	updates := map[string]interface{}{
		"name":      "New Name",
		"domain":    "new.domain.com",
		"is_active": false,
	}

	updated, err := service.UpdateTenant(context.Background(), tenant.ID, updates)

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if updated.Name != "New Name" {
		t.Errorf("Expected name 'New Name', got '%s'", updated.Name)
	}

	if updated.Domain != "new.domain.com" {
		t.Errorf("Expected domain 'new.domain.com', got '%s'", updated.Domain)
	}

	if updated.IsActive != false {
		t.Error("Expected tenant to be inactive")
	}
}

func TestGetTenant_NotFound(t *testing.T) {
	// Setup
	repo := newMockRepository()
	service := NewTenantService(repo, nil, nil, metrics.NewMetricsCollectorWithRegistry(prometheus.NewRegistry()), "")

	// Test get non-existent tenant
	_, err := service.GetTenant(context.Background(), "non-existent")

	if err == nil {
		t.Fatal("Expected error for non-existent tenant, got nil")
	}
}

func TestGetProvisioningStatus_Success(t *testing.T) {
	// Setup
	repo := newMockRepository()
	service := NewTenantService(repo, nil, nil, metrics.NewMetricsCollectorWithRegistry(prometheus.NewRegistry()), "")

	// Create a provisioning record
	provisioning := &models.TenantProvisioning{
		ID:                 "test-prov-001",
		TenantID:           "test-tenant-001",
		Status:             models.StatusCompleted.String(),
		ProgressPercentage: 100,
		StartedAt:          time.Now(),
	}
	repo.provisioning[provisioning.ID] = provisioning

	// Test get provisioning status
	result, err := service.GetProvisioningStatus(context.Background(), provisioning.ID)

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if result == nil {
		t.Fatal("Expected provisioning record, got nil")
	}

	if result.ID != provisioning.ID {
		t.Errorf("Expected ID %s, got %s", provisioning.ID, result.ID)
	}

	if result.Status != provisioning.Status {
		t.Errorf("Expected status %s, got %s", provisioning.Status, result.Status)
	}
}

func TestGetProvisioningStatus_NotFound(t *testing.T) {
	// Setup
	repo := newMockRepository()
	service := NewTenantService(repo, nil, nil, metrics.NewMetricsCollectorWithRegistry(prometheus.NewRegistry()), "")

	// Test get non-existent provisioning
	_, err := service.GetProvisioningStatus(context.Background(), "non-existent")

	if err == nil {
		t.Fatal("Expected error for non-existent provisioning, got nil")
	}
}

func TestCreateTenantAdmin_Success(t *testing.T) {
	// Setup
	repo := newMockRepository()
	userClient := &mockUserServiceClient{}
	service := NewTenantService(repo, userClient, nil, metrics.NewMetricsCollectorWithRegistry(prometheus.NewRegistry()), "")

	// Create a tenant first
	tenant := &models.Tenant{
		ID:                "test-tenant-001",
		Name:              "Test Tenant",
		Domain:            "test-domain",
		TierID:            "tier-free-001",
		IsActive:          true,
		StorageQuotaBytes: 1073741824,
		CreatedAt:         time.Now(),
		UpdatedAt:         time.Now(),
	}
	repo.tenants[tenant.ID] = tenant

	// Test create admin
	admin, err := service.CreateTenantAdmin(context.Background(), tenant.ID, "admin@test.com", "John", "Doe", "password123")

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if admin == nil {
		t.Fatal("Expected admin to be created, got nil")
	}

	if admin.Email != "admin@test.com" {
		t.Errorf("Expected email admin@test.com, got %s", admin.Email)
	}

	if admin.TenantID != tenant.ID {
		t.Errorf("Expected tenant ID %s, got %s", tenant.ID, admin.TenantID)
	}

	// Verify admin was stored
	if _, exists := repo.admins[admin.ID]; !exists {
		t.Error("Admin was not stored in repository")
	}
}

func TestCreateTenantAdmin_TenantNotFound(t *testing.T) {
	// Setup
	repo := newMockRepository()
	userClient := &mockUserServiceClient{}
	service := NewTenantService(repo, userClient, nil, metrics.NewMetricsCollectorWithRegistry(prometheus.NewRegistry()), "")

	// Test create admin for non-existent tenant
	_, err := service.CreateTenantAdmin(context.Background(), "non-existent", "admin@test.com", "John", "Doe", "password123")

	if err == nil {
		t.Fatal("Expected error for non-existent tenant, got nil")
	}
}

func TestGetTenantAdmin_Success(t *testing.T) {
	// Setup
	repo := newMockRepository()
	service := NewTenantService(repo, nil, nil, metrics.NewMetricsCollectorWithRegistry(prometheus.NewRegistry()), "")

	// Create tenant and admin
	tenant := &models.Tenant{
		ID:                "test-tenant-001",
		Name:              "Test Tenant",
		Domain:            "test-domain",
		TierID:            "tier-free-001",
		IsActive:          true,
		StorageQuotaBytes: 1073741824,
		CreatedAt:         time.Now(),
		UpdatedAt:         time.Now(),
	}
	repo.tenants[tenant.ID] = tenant

	admin := &models.TenantAdmin{
		ID:        "test-admin-001",
		TenantID:  tenant.ID,
		UserID:    "user-001",
		Email:     "admin@test.com",
		FirstName: sql.NullString{String: "John", Valid: true},
		LastName:  sql.NullString{String: "Doe", Valid: true},
		IsPrimary: true,
		CreatedAt: time.Now(),
	}
	repo.admins[admin.ID] = admin

	// Test get admin
	result, err := service.GetTenantAdmin(context.Background(), tenant.ID, admin.ID)

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if result == nil {
		t.Fatal("Expected admin, got nil")
	}

	if result.ID != admin.ID {
		t.Errorf("Expected ID %s, got %s", admin.ID, result.ID)
	}

	if result.Email != admin.Email {
		t.Errorf("Expected email %s, got %s", admin.Email, result.Email)
	}
}

func TestGetTenantAdmin_NotFound(t *testing.T) {
	// Setup
	repo := newMockRepository()
	service := NewTenantService(repo, nil, nil, metrics.NewMetricsCollectorWithRegistry(prometheus.NewRegistry()), "")

	// Create tenant
	tenant := &models.Tenant{
		ID:                "test-tenant-001",
		Name:              "Test Tenant",
		Domain:            "test-domain",
		TierID:            "tier-free-001",
		IsActive:          true,
		StorageQuotaBytes: 1073741824,
		CreatedAt:         time.Now(),
		UpdatedAt:         time.Now(),
	}
	repo.tenants[tenant.ID] = tenant

	// Test get non-existent admin
	_, err := service.GetTenantAdmin(context.Background(), tenant.ID, "non-existent")

	if err == nil {
		t.Fatal("Expected error for non-existent admin, got nil")
	}
}

