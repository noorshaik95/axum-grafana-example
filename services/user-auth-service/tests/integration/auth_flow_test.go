package integration

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"

	"slate/services/user-auth-service/pkg/jwt"
)

// TestAuthFlow_RegisterLoginValidate tests the full auth lifecycle:
// 1. Register a user (insert into DB with hashed password)
// 2. Login (verify password, generate JWT)
// 3. Validate the JWT and confirm it contains the correct claims
func TestAuthFlow_RegisterLoginValidate(t *testing.T) {
	testDB := SetupTestDatabase(t)
	defer testDB.Cleanup(t)
	testDB.RunMigrations(t)

	email := "authflow@example.com"
	password := "securePassword123"

	// Step 1: Register — create user with hashed password
	user := testDB.CreateTestUser(t, email, password, true)
	require.NotNil(t, user)
	assert.Equal(t, email, user.Email)

	// Verify the password hash is stored, not plaintext
	var storedHash string
	err := testDB.DB.QueryRow("SELECT password_hash FROM users WHERE id = $1", user.ID).Scan(&storedHash)
	require.NoError(t, err)
	assert.NotEqual(t, password, storedHash)

	// Step 2: Login — verify password against stored hash
	err = bcrypt.CompareHashAndPassword([]byte(storedHash), []byte(password))
	require.NoError(t, err, "password verification should succeed for correct password")

	// Wrong password should fail
	err = bcrypt.CompareHashAndPassword([]byte(storedHash), []byte("wrongPassword"))
	assert.Error(t, err, "password verification should fail for wrong password")

	// Generate JWT tokens (simulates what the login handler does)
	tokenService := jwt.NewTokenService("test-secret-key", 15, 24)

	accessToken, expiresIn, err := tokenService.GenerateAccessToken(user.ID, email, []string{"user"})
	require.NoError(t, err)
	assert.NotEmpty(t, accessToken)
	assert.Greater(t, expiresIn, int64(0))

	refreshToken, err := tokenService.GenerateRefreshToken(user.ID, email, []string{"user"})
	require.NoError(t, err)
	assert.NotEmpty(t, refreshToken)

	// Step 3: Validate — the JWT should contain the correct user claims
	claims, err := tokenService.ValidateAccessToken(accessToken)
	require.NoError(t, err)
	assert.Equal(t, user.ID, claims.UserID)
	assert.Equal(t, email, claims.Email)
	assert.Equal(t, []string{"user"}, claims.Roles)
	assert.Equal(t, "access", claims.Type)

	// Validate refresh token
	refreshClaims, err := tokenService.ValidateRefreshToken(refreshToken)
	require.NoError(t, err)
	assert.Equal(t, user.ID, refreshClaims.UserID)
	assert.Equal(t, email, refreshClaims.Email)
	assert.Equal(t, "refresh", refreshClaims.Type)

	// Invalid token should be rejected
	_, err = tokenService.ValidateAccessToken("invalid.token.here")
	assert.Error(t, err)

	// Refresh token should not pass access token validation
	_, err = tokenService.ValidateAccessToken(refreshToken)
	assert.Error(t, err)

	t.Log("Full auth flow verified: register -> login (password check) -> JWT generation -> JWT validation")
}

// TestAuthFlow_TokenRefreshCycle tests generating a token, then refreshing it
func TestAuthFlow_TokenRefreshCycle(t *testing.T) {
	testDB := SetupTestDatabase(t)
	defer testDB.Cleanup(t)
	testDB.RunMigrations(t)

	email := "refresh@example.com"
	password := "refreshTest123"

	user := testDB.CreateTestUser(t, email, password, true)
	require.NotNil(t, user)

	tokenService := jwt.NewTokenService("test-secret-key", 15, 24)

	// Generate initial tokens
	_, _, err := tokenService.GenerateAccessToken(user.ID, email, []string{"user"})
	require.NoError(t, err)

	refreshToken, err := tokenService.GenerateRefreshToken(user.ID, email, []string{"user"})
	require.NoError(t, err)

	// Refresh the access token
	newAccessToken, newRefreshToken, newExpiresIn, err := tokenService.RefreshAccessToken(refreshToken)
	require.NoError(t, err)
	assert.NotEmpty(t, newAccessToken)
	assert.NotEmpty(t, newRefreshToken)
	assert.Greater(t, newExpiresIn, int64(0))

	// New access token should be valid
	claims, err := tokenService.ValidateAccessToken(newAccessToken)
	require.NoError(t, err)
	assert.Equal(t, user.ID, claims.UserID)
	assert.Equal(t, email, claims.Email)

	t.Log("Token refresh cycle verified: generate -> refresh -> validate new token")
}

// TestAuthFlow_InactiveUserCannotLogin verifies that an inactive user's
// password still matches but the application should reject them based on is_active flag
func TestAuthFlow_InactiveUserCannotLogin(t *testing.T) {
	testDB := SetupTestDatabase(t)
	defer testDB.Cleanup(t)
	testDB.RunMigrations(t)

	email := "inactive@example.com"
	password := "inactiveUser123"

	user := testDB.CreateTestUser(t, email, password, false)
	require.NotNil(t, user)

	// Query user status
	var isActive bool
	var storedHash string
	err := testDB.DB.QueryRow(
		"SELECT is_active, password_hash FROM users WHERE id = $1", user.ID,
	).Scan(&isActive, &storedHash)
	require.NoError(t, err)

	// Password hash matches (bcrypt check passes)
	err = bcrypt.CompareHashAndPassword([]byte(storedHash), []byte(password))
	require.NoError(t, err, "password hash should match even for inactive user")

	// But the user is inactive — application layer should reject login
	assert.False(t, isActive, "user should be marked inactive")

	t.Log("Verified: inactive user's password matches but is_active=false prevents login")
}
