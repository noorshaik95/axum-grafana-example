package repository

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"slate/services/tenant-service/internal/models"

	_ "github.com/lib/pq"
)

// Note: These are integration tests that require a running PostgreSQL instance
// To run: docker-compose up postgres -d && go test -v ./internal/repository/...

func setupTestDB(t *testing.T) *sql.DB {
	// Skip if not in integration test mode
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	dsn := "host=localhost port=5432 user=postgres password=postgres dbname=tenantdb_test sslmode=disable"
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		t.Fatalf("Failed to connect to test database: %v", err)
	}

	// Ping to verify connection
	if err := db.Ping(); err != nil {
		t.Skipf("Test database not available: %v", err)
	}

	// Clean up any existing test data
	cleanupTestDB(t, db)

	return db
}

func cleanupTestDB(t *testing.T, db *sql.DB) {
	tables := []string{
		"tenant_setup_tokens",
		"tenant_provisioning",
		"tenant_admins",
		"tenant_storage_files",
		"tenants",
		"subscription_tiers",
	}

	for _, table := range tables {
		_, err := db.Exec("DELETE FROM " + table + " WHERE id LIKE 'test-%'")
		if err != nil {
			t.Logf("Warning: Failed to clean up table %s: %v", table, err)
		}
	}
}

func createTestTier(t *testing.T, db *sql.DB) *models.SubscriptionTier {
	tier := &models.SubscriptionTier{
		ID:                "test-tier-001",
		Name:              "test-free",
		DisplayName:       "Test Free",
		TierLevel:         0,
		StorageQuotaBytes: 1073741824,
		MaxUsers:          10,
		MaxCourses:        5,
		APIRateLimit:      100,
		DedicatedDatabase: false,
		CustomDomain:      false,
		PrioritySupport:   false,
		PriceCentsMonthly: 0,
		CreatedAt:         time.Now(),
		UpdatedAt:         time.Now(),
	}

	query := `
		INSERT INTO subscription_tiers (
			id, name, display_name, tier_level, storage_quota_bytes,
			max_users, max_courses, api_rate_limit, dedicated_database,
			custom_domain, priority_support, price_cents_monthly,
			created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
		ON CONFLICT (id) DO NOTHING
	`

	_, err := db.Exec(query,
		tier.ID, tier.Name, tier.DisplayName, tier.TierLevel, tier.StorageQuotaBytes,
		tier.MaxUsers, tier.MaxCourses, tier.APIRateLimit, tier.DedicatedDatabase,
		tier.CustomDomain, tier.PrioritySupport, tier.PriceCentsMonthly,
		tier.CreatedAt, tier.UpdatedAt,
	)

	if err != nil {
		t.Fatalf("Failed to create test tier: %v", err)
	}

	return tier
}

func TestTenantRepository_Create(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	defer cleanupTestDB(t, db)

	tier := createTestTier(t, db)
	repo := NewTenantRepository(db)

	tenant := &models.Tenant{
		ID:                "test-tenant-001",
		Name:              "Test Tenant",
		Domain:            "test.example.com",
		TierID:            tier.ID,
		IsActive:          true,
		StorageQuotaBytes: 1073741824,
		StorageUsedBytes:  0,
		FileCount:         0,
		UserCount:         0,
		CourseCount:       0,
		CreatedAt:         time.Now(),
		UpdatedAt:         time.Now(),
	}

	err := repo.Create(context.Background(), tenant)
	if err != nil {
		t.Fatalf("Failed to create tenant: %v", err)
	}

	// Verify tenant was created
	retrieved, err := repo.GetByID(context.Background(), tenant.ID)
	if err != nil {
		t.Fatalf("Failed to retrieve tenant: %v", err)
	}

	if retrieved.Name != tenant.Name {
		t.Errorf("Expected name %s, got %s", tenant.Name, retrieved.Name)
	}

	if retrieved.Domain != tenant.Domain {
		t.Errorf("Expected domain %s, got %s", tenant.Domain, retrieved.Domain)
	}
}

func TestTenantRepository_GetByDomain(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	defer cleanupTestDB(t, db)

	tier := createTestTier(t, db)
	repo := NewTenantRepository(db)

	tenant := &models.Tenant{
		ID:                "test-tenant-002",
		Name:              "Domain Test",
		Domain:            "domain-test.example.com",
		TierID:            tier.ID,
		IsActive:          true,
		StorageQuotaBytes: 1073741824,
		CreatedAt:         time.Now(),
		UpdatedAt:         time.Now(),
	}

	if err := repo.Create(context.Background(), tenant); err != nil {
		t.Fatalf("Failed to create tenant: %v", err)
	}

	// Retrieve by domain
	retrieved, err := repo.GetByDomain(context.Background(), tenant.Domain)
	if err != nil {
		t.Fatalf("Failed to retrieve tenant by domain: %v", err)
	}

	if retrieved.ID != tenant.ID {
		t.Errorf("Expected ID %s, got %s", tenant.ID, retrieved.ID)
	}
}

func TestTenantRepository_Update(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	defer cleanupTestDB(t, db)

	tier := createTestTier(t, db)
	repo := NewTenantRepository(db)

	tenant := &models.Tenant{
		ID:                "test-tenant-003",
		Name:              "Update Test",
		Domain:            "update.example.com",
		TierID:            tier.ID,
		IsActive:          true,
		StorageQuotaBytes: 1073741824,
		UserCount:         5,
		CourseCount:       3,
		CreatedAt:         time.Now(),
		UpdatedAt:         time.Now(),
	}

	if err := repo.Create(context.Background(), tenant); err != nil {
		t.Fatalf("Failed to create tenant: %v", err)
	}

	// Update tenant
	tenant.Name = "Updated Name"
	tenant.UserCount = 10
	tenant.CourseCount = 7
	tenant.IsActive = false

	err := repo.Update(context.Background(), tenant)
	if err != nil {
		t.Fatalf("Failed to update tenant: %v", err)
	}

	// Verify update
	retrieved, _ := repo.GetByID(context.Background(), tenant.ID)

	if retrieved.Name != "Updated Name" {
		t.Errorf("Expected name 'Updated Name', got %s", retrieved.Name)
	}

	if retrieved.UserCount != 10 {
		t.Errorf("Expected user count 10, got %d", retrieved.UserCount)
	}

	if retrieved.IsActive != false {
		t.Error("Expected tenant to be inactive")
	}
}

func TestTenantRepository_List(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	defer cleanupTestDB(t, db)

	tier := createTestTier(t, db)
	repo := NewTenantRepository(db)

	// Create multiple tenants
	for i := 1; i <= 5; i++ {
		tenant := &models.Tenant{
			ID:                "test-tenant-list-" + string(rune('0'+i)),
			Name:              "List Test " + string(rune('0'+i)),
			Domain:            "list" + string(rune('0'+i)) + ".example.com",
			TierID:            tier.ID,
			IsActive:          i%2 == 0, // Alternate active/inactive
			StorageQuotaBytes: 1073741824,
			CreatedAt:         time.Now(),
			UpdatedAt:         time.Now(),
		}
		if err := repo.Create(context.Background(), tenant); err != nil {
			t.Fatalf("Failed to create tenant: %v", err)
		}
	}

	// List all
	tenants, total, err := repo.List(context.Background(), 1, 10, map[string]interface{}{})
	if err != nil {
		t.Fatalf("Failed to list tenants: %v", err)
	}

	if total < 5 {
		t.Errorf("Expected at least 5 tenants, got %d", total)
	}

	if len(tenants) < 5 {
		t.Errorf("Expected at least 5 tenants in result, got %d", len(tenants))
	}

	// List with filter (active only)
	activeTenants, activeTotal, _ := repo.List(context.Background(), 1, 10, map[string]interface{}{
		"is_active": true,
	})

	if activeTotal < 2 {
		t.Errorf("Expected at least 2 active tenants, got %d", activeTotal)
	}

	for _, tenant := range activeTenants {
		if !tenant.IsActive {
			t.Error("Found inactive tenant in active filter")
		}
	}
}

func TestTenantRepository_StorageQuota(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	defer cleanupTestDB(t, db)

	tier := createTestTier(t, db)
	repo := NewTenantRepository(db)

	tenant := &models.Tenant{
		ID:                "test-tenant-storage",
		Name:              "Storage Test",
		Domain:            "storage.example.com",
		TierID:            tier.ID,
		IsActive:          true,
		StorageQuotaBytes: 1073741824,
		StorageUsedBytes:  0,
		FileCount:         0,
		CreatedAt:         time.Now(),
		UpdatedAt:         time.Now(),
	}

	if err := repo.Create(context.Background(), tenant); err != nil {
		t.Fatalf("Failed to create tenant: %v", err)
	}

	// AC6: Update storage usage
	err := repo.UpdateStorageUsage(context.Background(), tenant.ID, 104857600, 10) // +100MB, +10 files
	if err != nil {
		t.Fatalf("Failed to update storage usage: %v", err)
	}

	// Verify
	quota, used, files, err := repo.GetStorageQuota(context.Background(), tenant.ID)
	if err != nil {
		t.Fatalf("Failed to get storage quota: %v", err)
	}

	if quota != 1073741824 {
		t.Errorf("Expected quota 1073741824, got %d", quota)
	}

	if used != 104857600 {
		t.Errorf("Expected used 104857600, got %d", used)
	}

	if files != 10 {
		t.Errorf("Expected 10 files, got %d", files)
	}
}

func TestTenantRepository_CreateAdmin(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	defer cleanupTestDB(t, db)

	tier := createTestTier(t, db)
	repo := NewTenantRepository(db)

	tenant := &models.Tenant{
		ID:                "test-tenant-admin",
		Name:              "Admin Test",
		Domain:            "admin.example.com",
		TierID:            tier.ID,
		IsActive:          true,
		StorageQuotaBytes: 1073741824,
		CreatedAt:         time.Now(),
		UpdatedAt:         time.Now(),
	}

	if err := repo.Create(context.Background(), tenant); err != nil {
		t.Fatalf("Failed to create tenant: %v", err)
	}

	// AC3: Create admin
	admin := &models.TenantAdmin{
		ID:        "test-admin-001",
		TenantID:  tenant.ID,
		UserID:    "user-123",
		Email:     "admin@test.com",
		FirstName: sql.NullString{String: "Admin", Valid: true},
		LastName:  sql.NullString{String: "User", Valid: true},
		IsPrimary: true,
		CreatedAt: time.Now(),
	}

	err := repo.CreateAdmin(context.Background(), admin)
	if err != nil {
		t.Fatalf("Failed to create admin: %v", err)
	}

	// Retrieve admin
	retrieved, err := repo.GetAdminByID(context.Background(), tenant.ID, admin.ID)
	if err != nil {
		t.Fatalf("Failed to retrieve admin: %v", err)
	}

	if retrieved.Email != admin.Email {
		t.Errorf("Expected email %s, got %s", admin.Email, retrieved.Email)
	}

	if !retrieved.IsPrimary {
		t.Error("Expected admin to be primary")
	}
}

func TestTenantRepository_Provisioning(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	defer cleanupTestDB(t, db)

	tier := createTestTier(t, db)
	repo := NewTenantRepository(db)

	tenant := &models.Tenant{
		ID:                "test-tenant-provision",
		Name:              "Provision Test",
		Domain:            "provision.example.com",
		TierID:            tier.ID,
		IsActive:          true,
		StorageQuotaBytes: 1073741824,
		CreatedAt:         time.Now(),
		UpdatedAt:         time.Now(),
	}

	if err := repo.Create(context.Background(), tenant); err != nil {
		t.Fatalf("Failed to create tenant: %v", err)
	}

	// AC7: Create provisioning record
	provisioning := &models.TenantProvisioning{
		ID:                 "test-provision-001",
		TenantID:           tenant.ID,
		Status:             "pending",
		ProgressPercentage: 0,
		StartedAt:          time.Now(),
	}

	err := repo.CreateProvisioning(context.Background(), provisioning)
	if err != nil {
		t.Fatalf("Failed to create provisioning: %v", err)
	}

	// Update provisioning
	provisioning.Status = "completed"
	provisioning.ProgressPercentage = 100
	provisioning.CompletedAt = sql.NullTime{Time: time.Now(), Valid: true}
	provisioning.DurationSeconds = sql.NullInt32{Int32: 85, Valid: true} // AC7: < 120 seconds

	err = repo.UpdateProvisioning(context.Background(), provisioning)
	if err != nil {
		t.Fatalf("Failed to update provisioning: %v", err)
	}

	// Retrieve and verify
	retrieved, err := repo.GetProvisioningByID(context.Background(), provisioning.ID)
	if err != nil {
		t.Fatalf("Failed to retrieve provisioning: %v", err)
	}

	if retrieved.Status != "completed" {
		t.Errorf("Expected status 'completed', got %s", retrieved.Status)
	}

	if retrieved.ProgressPercentage != 100 {
		t.Errorf("Expected progress 100, got %d", retrieved.ProgressPercentage)
	}

	// AC7: Verify duration < 120 seconds
	if retrieved.DurationSeconds.Valid && retrieved.DurationSeconds.Int32 >= 120 {
		t.Errorf("AC7 VIOLATION: Provisioning took %d seconds (must be < 120)", retrieved.DurationSeconds.Int32)
	}
}
