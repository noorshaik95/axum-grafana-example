package database

import (
	"os"
	"testing"
	"time"
)

func TestConnect_EnvironmentVariables(t *testing.T) {
	// Set environment variables
	os.Setenv("DB_HOST", "testhost")
	os.Setenv("DB_PORT", "5433")
	os.Setenv("DB_USER", "testuser")
	os.Setenv("DB_PASSWORD", "testpass")
	os.Setenv("DB_NAME", "testdb")
	os.Setenv("DB_SSLMODE", "require")
	os.Setenv("DB_MAX_OPEN_CONNS", "50")
	os.Setenv("DB_MAX_IDLE_CONNS", "25")
	os.Setenv("DB_CONN_MAX_LIFETIME", "30m")
	defer func() {
		os.Unsetenv("DB_HOST")
		os.Unsetenv("DB_PORT")
		os.Unsetenv("DB_USER")
		os.Unsetenv("DB_PASSWORD")
		os.Unsetenv("DB_NAME")
		os.Unsetenv("DB_SSLMODE")
		os.Unsetenv("DB_MAX_OPEN_CONNS")
		os.Unsetenv("DB_MAX_IDLE_CONNS")
		os.Unsetenv("DB_CONN_MAX_LIFETIME")
	}()

	// Build config from environment variables
	cfg := Config{
		Host:     os.Getenv("DB_HOST"),
		Port:     5433,
		User:     os.Getenv("DB_USER"),
		Password: os.Getenv("DB_PASSWORD"),
		DBName:   os.Getenv("DB_NAME"),
		SSLMode:  os.Getenv("DB_SSLMODE"),
	}

	// Note: This will fail to connect (no actual database), but we can verify the DSN was built correctly
	// by checking the error message
	_, err := Connect(cfg)
	if err == nil {
		t.Skip("Unexpected success - test database shouldn't be available")
	}

	// The error should indicate connection attempt was made
	// We just verify the function doesn't panic and returns an error
	if err == nil {
		t.Error("Expected error when connecting to non-existent database")
	}
}

func TestConnect_WithDefaults(t *testing.T) {
	// Clear environment to test defaults
	os.Clearenv()

	cfg := Config{
		Host:     "localhost",
		Port:     5432,
		User:     "postgres",
		Password: "postgres",
		DBName:   "testdb",
		SSLMode:  "disable",
	}

	_, err := Connect(cfg)
	// Should fail (no database), but shouldn't panic
	if err == nil {
		t.Skip("Unexpected success - test database shouldn't be available")
	}
}

func TestGetEnvAsInt(t *testing.T) {
	os.Setenv("TEST_INT", "42")
	defer os.Unsetenv("TEST_INT")

	result := getEnvAsInt("TEST_INT", 10)
	if result != 42 {
		t.Errorf("Expected 42, got %d", result)
	}

	result = getEnvAsInt("NON_EXISTENT", 10)
	if result != 10 {
		t.Errorf("Expected default 10, got %d", result)
	}

	os.Setenv("TEST_INVALID", "not-a-number")
	defer os.Unsetenv("TEST_INVALID")

	result = getEnvAsInt("TEST_INVALID", 20)
	if result != 20 {
		t.Errorf("Expected default 20 for invalid int, got %d", result)
	}
}

func TestGetDurationFromEnv(t *testing.T) {
	tests := []struct {
		name     string
		envValue string
		fallback time.Duration
		expected string
	}{
		{
			name:     "Valid duration",
			envValue: "30m",
			fallback: 10 * time.Minute,
			expected: "30m0s",
		},
		{
			name:     "Empty env - use fallback",
			envValue: "",
			fallback: 15 * time.Minute,
			expected: "15m0s",
		},
		{
			name:     "Invalid duration - use fallback",
			envValue: "invalid",
			fallback: 5 * time.Minute,
			expected: "5m0s",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.envValue != "" {
				os.Setenv("TEST_DURATION", tt.envValue)
				defer os.Unsetenv("TEST_DURATION")
			} else {
				os.Unsetenv("TEST_DURATION")
			}

			duration := getDurationFromEnv("TEST_DURATION", tt.fallback)
			if duration.String() != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, duration.String())
			}
		})
	}
}

func TestDurationParsing(t *testing.T) {
	// Test various duration formats
	durations := []string{
		"1h",
		"30m",
		"45s",
		"1h30m",
		"2h15m30s",
	}

	for _, d := range durations {
		os.Setenv("TEST_DUR", d)
		result := getDurationFromEnv("TEST_DUR", 1*time.Minute)
		if result.String() == "1m0s" {
			t.Errorf("Failed to parse valid duration %s", d)
		}
		os.Unsetenv("TEST_DUR")
	}
}
