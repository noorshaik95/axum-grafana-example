//go:build integration
// +build integration

package integration

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	_ "github.com/lib/pq"
	"golang.org/x/crypto/bcrypt"
)

// TestDatabase represents a test database connection
type TestDatabase struct {
	DB     *sql.DB
	DBName string
}

// SetupTestDatabase creates a test database and runs migrations. Only
// compiled under `go test -tags=integration` (see build tag at top of file);
// a live Postgres test server is required at DB_HOST:DB_PORT.
func SetupTestDatabase(t *testing.T) *TestDatabase {
	t.Helper()

	// Generate unique database name for this test
	dbName := fmt.Sprintf("userauth_test_%s", uuid.New().String()[:8])

	// Connect to postgres database to create test database
	adminConnStr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=postgres sslmode=disable",
		getEnvOrDefault("DB_HOST", "localhost"),
		getEnvOrDefault("DB_PORT", "5432"),
		getEnvOrDefault("DB_USER", "postgres"),
		getEnvOrDefault("DB_PASSWORD", "postgres"),
	)

	adminDB, err := sql.Open("postgres", adminConnStr)
	if err != nil {
		t.Fatalf("Failed to connect to admin database: %v", err)
	}
	defer adminDB.Close()

	// Create test database
	_, err = adminDB.Exec(fmt.Sprintf("CREATE DATABASE %s", dbName))
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}

	// Connect to test database
	testConnStr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		getEnvOrDefault("DB_HOST", "localhost"),
		getEnvOrDefault("DB_PORT", "5432"),
		getEnvOrDefault("DB_USER", "postgres"),
		getEnvOrDefault("DB_PASSWORD", "postgres"),
		dbName,
	)

	testDB, err := sql.Open("postgres", testConnStr)
	if err != nil {
		t.Fatalf("Failed to connect to test database: %v", err)
	}

	if err := testDB.Ping(); err != nil {
		t.Fatalf("Failed to ping test database: %v", err)
	}

	return &TestDatabase{
		DB:     testDB,
		DBName: dbName,
	}
}

// RunMigrations executes all migration files in the migrations directory
func (td *TestDatabase) RunMigrations(t *testing.T) {
	t.Helper()

	// Create migrations table
	_, err := td.DB.Exec(`
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version VARCHAR(255) PRIMARY KEY,
			applied_at TIMESTAMP NOT NULL DEFAULT NOW()
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create migrations table: %v", err)
	}

	// Get migrations directory
	migrationsDir := "../../migrations"

	// Read migration files
	files, err := os.ReadDir(migrationsDir)
	if err != nil {
		t.Fatalf("Failed to read migrations directory: %v", err)
	}

	// Apply each migration
	for _, file := range files {
		name := file.Name()
		// Only process .sql files, skip .down.sql files
		if file.IsDir() || len(name) < 5 || name[len(name)-4:] != ".sql" {
			continue
		}
		if len(name) > 9 && name[len(name)-9:] == ".down.sql" {
			continue
		}

		version := name[:len(name)-4]

		// Check if already applied
		var applied bool
		err := td.DB.QueryRow("SELECT EXISTS(SELECT 1 FROM schema_migrations WHERE version = $1)", version).Scan(&applied)
		if err != nil {
			t.Fatalf("Failed to check migration status: %v", err)
		}

		if applied {
			continue
		}

		// Read and execute migration
		content, err := os.ReadFile(fmt.Sprintf("%s/%s", migrationsDir, name))
		if err != nil {
			t.Fatalf("Failed to read migration file %s: %v", name, err)
		}

		tx, err := td.DB.Begin()
		if err != nil {
			t.Fatalf("Failed to begin transaction: %v", err)
		}

		_, err = tx.Exec(string(content))
		if err != nil {
			tx.Rollback()
			t.Fatalf("Failed to execute migration %s: %v", name, err)
		}

		_, err = tx.Exec("INSERT INTO schema_migrations (version) VALUES ($1)", version)
		if err != nil {
			tx.Rollback()
			t.Fatalf("Failed to record migration %s: %v", name, err)
		}

		if err := tx.Commit(); err != nil {
			t.Fatalf("Failed to commit migration %s: %v", name, err)
		}
	}
}

// Cleanup drops the test database
func (td *TestDatabase) Cleanup(t *testing.T) {
	t.Helper()

	// Close connection to test database
	td.DB.Close()

	// Connect to postgres database to drop test database
	adminConnStr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=postgres sslmode=disable",
		getEnvOrDefault("DB_HOST", "localhost"),
		getEnvOrDefault("DB_PORT", "5432"),
		getEnvOrDefault("DB_USER", "postgres"),
		getEnvOrDefault("DB_PASSWORD", "postgres"),
	)

	adminDB, err := sql.Open("postgres", adminConnStr)
	if err != nil {
		t.Logf("Failed to connect to admin database for cleanup: %v", err)
		return
	}
	defer adminDB.Close()

	// Terminate connections to test database
	_, err = adminDB.Exec(fmt.Sprintf(`
		SELECT pg_terminate_backend(pg_stat_activity.pid)
		FROM pg_stat_activity
		WHERE pg_stat_activity.datname = '%s'
		AND pid <> pg_backend_pid()
	`, td.DBName))
	if err != nil {
		t.Logf("Failed to terminate connections: %v", err)
	}

	// Drop test database
	_, err = adminDB.Exec(fmt.Sprintf("DROP DATABASE IF EXISTS %s", td.DBName))
	if err != nil {
		t.Logf("Failed to drop test database: %v", err)
	}
}

// TestUser represents a test user
type TestUser struct {
	ID           string
	Email        string
	PasswordHash string
	FirstName    string
	LastName     string
	IsActive     bool
	AuthMethod   string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// CreateTestUser creates a test user in the database
func (td *TestDatabase) CreateTestUser(t *testing.T, email, password string, isActive bool) *TestUser {
	t.Helper()

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("Failed to hash password: %v", err)
	}

	user := &TestUser{
		ID:           uuid.New().String(),
		Email:        email,
		PasswordHash: string(hashedPassword),
		FirstName:    "Test",
		LastName:     "User",
		IsActive:     isActive,
		AuthMethod:   "normal",
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	_, err = td.DB.Exec(`
		INSERT INTO users (id, email, password_hash, first_name, last_name, is_active, auth_method, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`, user.ID, user.Email, user.PasswordHash, user.FirstName, user.LastName, user.IsActive, user.AuthMethod, user.CreatedAt, user.UpdatedAt)
	if err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}

	return user
}

// MockOAuthProvider creates a mock OAuth provider server
type MockOAuthProvider struct {
	Server       *httptest.Server
	AuthURL      string
	TokenURL     string
	UserInfoURL  string
	ClientID     string
	ClientSecret string
	AccessToken  string
	UserInfo     map[string]interface{}
}

// NewMockOAuthProvider creates a new mock OAuth provider
func NewMockOAuthProvider() *MockOAuthProvider {
	mock := &MockOAuthProvider{
		ClientID:     "test-client-id",
		ClientSecret: "test-client-secret",
		AccessToken:  "test-access-token",
		UserInfo: map[string]interface{}{
			"id":          "oauth-user-123",
			"email":       "oauth@example.com",
			"given_name":  "OAuth",
			"family_name": "User",
			"picture":     "https://example.com/avatar.jpg",
		},
	}

	mux := http.NewServeMux()

	// Token endpoint
	mux.HandleFunc("/token", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		if err := r.ParseForm(); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		// Validate client credentials
		if r.FormValue("client_id") != mock.ClientID || r.FormValue("client_secret") != mock.ClientSecret {
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte(`{"error": "invalid_client"}`))
			return
		}

		// Return access token
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{
			"access_token": "%s",
			"token_type": "Bearer",
			"expires_in": 3600,
			"refresh_token": "test-refresh-token"
		}`, mock.AccessToken)
	})

	// UserInfo endpoint
	mux.HandleFunc("/userinfo", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		// Validate access token
		authHeader := r.Header.Get("Authorization")
		expectedAuth := fmt.Sprintf("Bearer %s", mock.AccessToken)
		if authHeader != expectedAuth {
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte(`{"error": "invalid_token"}`))
			return
		}

		// Return user info
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{
			"id": "%s",
			"email": "%s",
			"given_name": "%s",
			"family_name": "%s",
			"picture": "%s"
		}`,
			mock.UserInfo["id"],
			mock.UserInfo["email"],
			mock.UserInfo["given_name"],
			mock.UserInfo["family_name"],
			mock.UserInfo["picture"],
		)
	})

	// Authorization endpoint
	mux.HandleFunc("/authorize", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		// In a real OAuth flow, this would redirect to login page
		// For testing, we just return the redirect URI with code
		redirectURI := r.URL.Query().Get("redirect_uri")
		state := r.URL.Query().Get("state")

		redirectURL := fmt.Sprintf("%s?code=test-auth-code&state=%s", redirectURI, state)
		w.Header().Set("Location", redirectURL)
		w.WriteHeader(http.StatusFound)
	})

	mock.Server = httptest.NewServer(mux)
	mock.AuthURL = mock.Server.URL + "/authorize"
	mock.TokenURL = mock.Server.URL + "/token"
	mock.UserInfoURL = mock.Server.URL + "/userinfo"

	return mock
}

// Close shuts down the mock OAuth provider server
func (m *MockOAuthProvider) Close() {
	m.Server.Close()
}

// MockSAMLProvider creates a mock SAML IdP
type MockSAMLProvider struct {
	EntityID    string
	SSOURL      string
	Certificate string
	PrivateKey  string
}

// NewMockSAMLProvider creates a new mock SAML provider
func NewMockSAMLProvider() *MockSAMLProvider {
	return &MockSAMLProvider{
		EntityID:    "https://idp.example.com/saml",
		SSOURL:      "https://idp.example.com/sso",
		Certificate: "test-certificate",
		PrivateKey:  "test-private-key",
	}
}

// GenerateMockSAMLAssertion generates a mock SAML assertion for testing
func (m *MockSAMLProvider) GenerateMockSAMLAssertion(nameID, email, firstName, lastName string) string {
	// This is a simplified mock SAML assertion for testing
	// In production, this would be properly signed and formatted
	return fmt.Sprintf(`<?xml version="1.0"?>
<saml:Assertion xmlns:saml="urn:oasis:names:tc:SAML:2.0:assertion" ID="test-assertion-id" IssueInstant="%s">
	<saml:Issuer>%s</saml:Issuer>
	<saml:Subject>
		<saml:NameID>%s</saml:NameID>
	</saml:Subject>
	<saml:AttributeStatement>
		<saml:Attribute Name="email">
			<saml:AttributeValue>%s</saml:AttributeValue>
		</saml:Attribute>
		<saml:Attribute Name="firstName">
			<saml:AttributeValue>%s</saml:AttributeValue>
		</saml:Attribute>
		<saml:Attribute Name="lastName">
			<saml:AttributeValue>%s</saml:AttributeValue>
		</saml:Attribute>
	</saml:AttributeStatement>
</saml:Assertion>`,
		time.Now().Format(time.RFC3339),
		m.EntityID,
		nameID,
		email,
		firstName,
		lastName,
	)
}

// getEnvOrDefault returns environment variable value or default
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// WaitForDatabase waits for database to be ready
func WaitForDatabase(ctx context.Context, db *sql.DB, maxRetries int) error {
	for i := 0; i < maxRetries; i++ {
		if err := db.PingContext(ctx); err == nil {
			return nil
		}
		time.Sleep(time.Second)
	}
	return fmt.Errorf("database not ready after %d retries", maxRetries)
}
