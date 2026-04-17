package config

import (
	"os"
	"testing"
)

func TestLoad_Success(t *testing.T) {
	// Setup environment variables
	os.Setenv("SERVER_PORT", "8080")
	os.Setenv("DB_HOST", "localhost")
	os.Setenv("DB_PORT", "5432")
	os.Setenv("DB_USER", "testuser")
	os.Setenv("DB_PASSWORD", "testpass")
	os.Setenv("DB_NAME", "testdb")
	defer func() {
		os.Unsetenv("SERVER_PORT")
		os.Unsetenv("DB_HOST")
		os.Unsetenv("DB_PORT")
		os.Unsetenv("DB_USER")
		os.Unsetenv("DB_PASSWORD")
		os.Unsetenv("DB_NAME")
	}()

	// Test Load
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if cfg == nil {
		t.Fatal("Expected config to be loaded, got nil")
	}

	if cfg.Server.Port != 8080 {
		t.Errorf("Expected port 8080, got %d", cfg.Server.Port)
	}

	if cfg.Database.Host != "localhost" {
		t.Errorf("Expected host localhost, got %s", cfg.Database.Host)
	}

	if cfg.Database.Port != 5432 {
		t.Errorf("Expected port 5432, got %d", cfg.Database.Port)
	}

	if cfg.Database.User != "testuser" {
		t.Errorf("Expected user testuser, got %s", cfg.Database.User)
	}

	if cfg.Database.Password != "testpass" {
		t.Errorf("Expected password testpass, got %s", cfg.Database.Password)
	}

	if cfg.Database.DBName != "testdb" {
		t.Errorf("Expected dbname testdb, got %s", cfg.Database.DBName)
	}
}

func TestLoad_WithDefaults(t *testing.T) {
	// Clear all relevant env vars to test defaults
	os.Clearenv()

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if cfg == nil {
		t.Fatal("Expected config to be loaded, got nil")
	}

	// Check defaults
	if cfg.Server.Port != 8083 {
		t.Errorf("Expected default port 8083, got %d", cfg.Server.Port)
	}

	if cfg.Database.Host != "localhost" {
		t.Errorf("Expected default host localhost, got %s", cfg.Database.Host)
	}

	if cfg.Database.Port != 5432 {
		t.Errorf("Expected default port 5432, got %d", cfg.Database.Port)
	}
}

func TestDatabaseConfig_DSN(t *testing.T) {
	cfg := &DatabaseConfig{
		Host:     "testhost",
		Port:     5432,
		User:     "testuser",
		Password: "testpass",
		DBName:   "testdb",
		SSLMode:  "disable",
	}

	dsn := cfg.DSN()

	expected := "host=testhost port=5432 user=testuser password=testpass dbname=testdb sslmode=disable"
	if dsn != expected {
		t.Errorf("Expected DSN %s, got %s", expected, dsn)
	}
}

func TestGRPCConfig_Address(t *testing.T) {
	cfg := &GRPCConfig{
		Host: "grpc-host",
		Port: 50053,
	}

	addr := cfg.Address()

	expected := "grpc-host:50053"
	if addr != expected {
		t.Errorf("Expected address %s, got %s", expected, addr)
	}
}

func TestServerConfig_Address(t *testing.T) {
	cfg := &ServerConfig{
		Host: "server-host",
		Port: 8080,
	}

	addr := cfg.Address()

	expected := "server-host:8080"
	if addr != expected {
		t.Errorf("Expected address %s, got %s", expected, addr)
	}
}

func TestGetEnv(t *testing.T) {
	os.Setenv("TEST_VAR", "test-value")
	defer os.Unsetenv("TEST_VAR")

	result := getEnv("TEST_VAR", "default")
	if result != "test-value" {
		t.Errorf("Expected test-value, got %s", result)
	}

	result = getEnv("NON_EXISTENT", "default")
	if result != "default" {
		t.Errorf("Expected default, got %s", result)
	}
}

func TestGetEnvAsInt(t *testing.T) {
	os.Setenv("TEST_INT", "123")
	defer os.Unsetenv("TEST_INT")

	result := getEnvAsInt("TEST_INT", 456)
	if result != 123 {
		t.Errorf("Expected 123, got %d", result)
	}

	result = getEnvAsInt("NON_EXISTENT", 456)
	if result != 456 {
		t.Errorf("Expected default 456, got %d", result)
	}

	os.Setenv("TEST_INVALID", "not-a-number")
	defer os.Unsetenv("TEST_INVALID")

	result = getEnvAsInt("TEST_INVALID", 789)
	if result != 789 {
		t.Errorf("Expected default 789 for invalid int, got %d", result)
	}
}

func TestGetEnvAsBool(t *testing.T) {
	os.Setenv("TEST_BOOL_TRUE", "true")
	os.Setenv("TEST_BOOL_FALSE", "false")
	os.Setenv("TEST_BOOL_INVALID", "invalid")
	defer func() {
		os.Unsetenv("TEST_BOOL_TRUE")
		os.Unsetenv("TEST_BOOL_FALSE")
		os.Unsetenv("TEST_BOOL_INVALID")
	}()

	if !getEnvAsBool("TEST_BOOL_TRUE", false) {
		t.Error("Expected true for TEST_BOOL_TRUE")
	}

	if getEnvAsBool("TEST_BOOL_FALSE", true) {
		t.Error("Expected false for TEST_BOOL_FALSE")
	}

	if !getEnvAsBool("NON_EXISTENT", true) {
		t.Error("Expected default true for non-existent var")
	}

	if !getEnvAsBool("TEST_BOOL_INVALID", true) {
		t.Error("Expected default true for invalid bool")
	}
}
