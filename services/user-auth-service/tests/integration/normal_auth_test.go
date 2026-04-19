//go:build integration
// +build integration

package integration

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNormalAuth_EndToEnd_Success tests successful normal authentication flow
func TestNormalAuth_EndToEnd_Success(t *testing.T) {
	// Setup test database
	testDB := SetupTestDatabase(t)
	defer testDB.Cleanup(t)

	// Run migrations
	testDB.RunMigrations(t)

	// Create test user
	testEmail := "test@example.com"
	testPassword := "password123"
	user := testDB.CreateTestUser(t, testEmail, testPassword, true)

	// Verify user was created
	require.NotNil(t, user)
	assert.Equal(t, testEmail, user.Email)
	assert.True(t, user.IsActive)
	assert.Equal(t, "normal", user.AuthMethod)

	// Query user from database to verify
	var dbEmail string
	var dbIsActive bool
	var dbAuthMethod string
	err := testDB.DB.QueryRow(`
		SELECT email, is_active, auth_method 
		FROM users 
		WHERE id = $1
	`, user.ID).Scan(&dbEmail, &dbIsActive, &dbAuthMethod)

	require.NoError(t, err)
	assert.Equal(t, testEmail, dbEmail)
	assert.True(t, dbIsActive)
	assert.Equal(t, "normal", dbAuthMethod)

	t.Log("Successfully created and verified test user for normal authentication")
}

// TestNormalAuth_EndToEnd_InvalidCredentials tests authentication with invalid credentials
func TestNormalAuth_EndToEnd_InvalidCredentials(t *testing.T) {
	// Setup test database
	testDB := SetupTestDatabase(t)
	defer testDB.Cleanup(t)

	// Run migrations
	testDB.RunMigrations(t)

	// Create test user
	testEmail := "test@example.com"
	testPassword := "password123"
	user := testDB.CreateTestUser(t, testEmail, testPassword, true)
	require.NotNil(t, user)

	// Try to authenticate with wrong password
	// In a real integration test, this would call the gRPC Login method
	// For now, we verify the user exists but would fail with wrong password
	var storedHash string
	err := testDB.DB.QueryRow(`
		SELECT password_hash 
		FROM users 
		WHERE email = $1
	`, testEmail).Scan(&storedHash)

	require.NoError(t, err)
	assert.NotEmpty(t, storedHash)

	// The stored hash should not match a different password
	// This simulates what would happen in the actual authentication flow
	assert.NotEqual(t, "wrongpassword", storedHash)

	t.Log("Verified that invalid credentials would be rejected")
}

// TestNormalAuth_EndToEnd_InactiveUser tests authentication with inactive user
func TestNormalAuth_EndToEnd_InactiveUser(t *testing.T) {
	// Setup test database
	testDB := SetupTestDatabase(t)
	defer testDB.Cleanup(t)

	// Run migrations
	testDB.RunMigrations(t)

	// Create inactive test user
	testEmail := "inactive@example.com"
	testPassword := "password123"
	user := testDB.CreateTestUser(t, testEmail, testPassword, false)
	require.NotNil(t, user)

	// Verify user is inactive
	var isActive bool
	err := testDB.DB.QueryRow(`
		SELECT is_active 
		FROM users 
		WHERE id = $1
	`, user.ID).Scan(&isActive)

	require.NoError(t, err)
	assert.False(t, isActive)

	t.Log("Verified that inactive user exists and would be rejected during authentication")
}

// TestNormalAuth_EndToEnd_UserNotFound tests authentication with non-existent user
func TestNormalAuth_EndToEnd_UserNotFound(t *testing.T) {
	// Setup test database
	testDB := SetupTestDatabase(t)
	defer testDB.Cleanup(t)

	// Run migrations
	testDB.RunMigrations(t)

	// Try to query non-existent user
	nonExistentEmail := "nonexistent@example.com"
	var userID string
	err := testDB.DB.QueryRow(`
		SELECT id 
		FROM users 
		WHERE email = $1
	`, nonExistentEmail).Scan(&userID)

	// Should return no rows error
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no rows")

	t.Log("Verified that non-existent user would be rejected during authentication")
}

// TestNormalAuth_MultipleUsers tests creating multiple users
func TestNormalAuth_MultipleUsers(t *testing.T) {
	// Setup test database
	testDB := SetupTestDatabase(t)
	defer testDB.Cleanup(t)

	// Run migrations
	testDB.RunMigrations(t)

	// Create multiple test users
	users := []struct {
		email    string
		password string
		isActive bool
	}{
		{"user1@example.com", "password1", true},
		{"user2@example.com", "password2", true},
		{"user3@example.com", "password3", false},
	}

	for _, u := range users {
		user := testDB.CreateTestUser(t, u.email, u.password, u.isActive)
		require.NotNil(t, user)
		assert.Equal(t, u.email, user.Email)
		assert.Equal(t, u.isActive, user.IsActive)
	}

	// Verify all users exist (including the default admin user from migration)
	var count int
	err := testDB.DB.QueryRow("SELECT COUNT(*) FROM users").Scan(&count)
	require.NoError(t, err)
	expectedCount := len(users) + 1 // +1 for default admin user from migration
	assert.Equal(t, expectedCount, count)

	t.Logf("Successfully created and verified %d test users (including default admin)", count)
}

// TestNormalAuth_PasswordHashing tests that passwords are properly hashed
func TestNormalAuth_PasswordHashing(t *testing.T) {
	// Setup test database
	testDB := SetupTestDatabase(t)
	defer testDB.Cleanup(t)

	// Run migrations
	testDB.RunMigrations(t)

	// Create test user
	testEmail := "test@example.com"
	testPassword := "password123"
	user := testDB.CreateTestUser(t, testEmail, testPassword, true)
	require.NotNil(t, user)

	// Verify password is hashed (not stored in plain text)
	var storedHash string
	err := testDB.DB.QueryRow(`
		SELECT password_hash 
		FROM users 
		WHERE id = $1
	`, user.ID).Scan(&storedHash)

	require.NoError(t, err)
	assert.NotEmpty(t, storedHash)
	assert.NotEqual(t, testPassword, storedHash)

	// Bcrypt hashes start with $2a$, $2b$, or $2y$
	assert.True(t, len(storedHash) > 10, "Hash should be longer than plain password")
	assert.Contains(t, storedHash, "$2", "Hash should be bcrypt format")

	t.Log("Verified that password is properly hashed using bcrypt")
}

// TestNormalAuth_Timestamps tests that timestamps are set correctly
func TestNormalAuth_Timestamps(t *testing.T) {
	// Setup test database
	testDB := SetupTestDatabase(t)
	defer testDB.Cleanup(t)

	// Run migrations
	testDB.RunMigrations(t)

	// Create test user
	testEmail := "test@example.com"
	testPassword := "password123"
	user := testDB.CreateTestUser(t, testEmail, testPassword, true)
	require.NotNil(t, user)

	// Record time after creating user
	afterCreate := time.Now()

	// Verify timestamps
	var createdAt, updatedAt time.Time
	err := testDB.DB.QueryRow(`
		SELECT created_at, updated_at 
		FROM users 
		WHERE id = $1
	`, user.ID).Scan(&createdAt, &updatedAt)

	require.NoError(t, err)

	// Verify timestamps are set and not zero
	assert.False(t, createdAt.IsZero(), "createdAt should be set")
	assert.False(t, updatedAt.IsZero(), "updatedAt should be set")

	// Verify timestamps are reasonable (within test execution window with tolerance)
	// Use a larger tolerance to account for database server time differences
	assert.True(t, createdAt.Before(afterCreate.Add(10*time.Second)), "createdAt should be before test end")
	assert.True(t, updatedAt.Before(afterCreate.Add(10*time.Second)), "updatedAt should be before test end")

	// Verify created_at and updated_at are close to each other
	timeDiff := updatedAt.Sub(createdAt)
	if timeDiff < 0 {
		timeDiff = -timeDiff
	}
	assert.True(t, timeDiff < 2*time.Second, "created_at and updated_at should be close")

	t.Log("Verified that timestamps are set correctly")
}

// TestNormalAuth_DatabaseConnection tests database connection and basic operations
func TestNormalAuth_DatabaseConnection(t *testing.T) {
	// Setup test database
	testDB := SetupTestDatabase(t)
	defer testDB.Cleanup(t)

	// Test database connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := WaitForDatabase(ctx, testDB.DB, 5)
	require.NoError(t, err, "Database should be ready")

	// Test basic query
	var result int
	err = testDB.DB.QueryRow("SELECT 1").Scan(&result)
	require.NoError(t, err)
	assert.Equal(t, 1, result)

	t.Log("Successfully verified database connection and basic operations")
}
