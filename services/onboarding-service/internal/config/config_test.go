package config

import (
	"os"
	"testing"
	"time"
)

func TestLoad_Defaults(t *testing.T) {
	// Clear environment variables
	clearEnv()

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}

	// Verify defaults
	if cfg.Server.Host != "0.0.0.0" {
		t.Errorf("Server.Host = %v, want 0.0.0.0", cfg.Server.Host)
	}
	if cfg.Server.Port != "8082" {
		t.Errorf("Server.Port = %v, want 8082", cfg.Server.Port)
	}
	if cfg.Database.MaxOpenConns != 50 {
		t.Errorf("Database.MaxOpenConns = %v, want 50", cfg.Database.MaxOpenConns)
	}
	if cfg.Worker.Concurrency != 10 {
		t.Errorf("Worker.Concurrency = %v, want 10", cfg.Worker.Concurrency)
	}
}

func TestLoad_CustomValues(t *testing.T) {
	// Set environment variables
	os.Setenv("SERVER_HOST", "127.0.0.1")
	os.Setenv("SERVER_PORT", "9000")
	os.Setenv("DB_MAX_OPEN_CONNS", "100")
	os.Setenv("WORKER_CONCURRENCY", "20")
	os.Setenv("KAFKA_BROKERS", "kafka1:9092,kafka2:9092")
	defer clearEnv()

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}

	if cfg.Server.Host != "127.0.0.1" {
		t.Errorf("Server.Host = %v, want 127.0.0.1", cfg.Server.Host)
	}
	if cfg.Server.Port != "9000" {
		t.Errorf("Server.Port = %v, want 9000", cfg.Server.Port)
	}
	if cfg.Database.MaxOpenConns != 100 {
		t.Errorf("Database.MaxOpenConns = %v, want 100", cfg.Database.MaxOpenConns)
	}
	if cfg.Worker.Concurrency != 20 {
		t.Errorf("Worker.Concurrency = %v, want 20", cfg.Worker.Concurrency)
	}
}

func TestLoad_DurationParsing(t *testing.T) {
	os.Setenv("DB_CONN_MAX_LIFETIME", "10m")
	os.Setenv("DB_CONN_MAX_IDLE_TIME", "2m")
	defer clearEnv()

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}

	if cfg.Database.ConnMaxLifetime != 10*time.Minute {
		t.Errorf("ConnMaxLifetime = %v, want 10m", cfg.Database.ConnMaxLifetime)
	}
	if cfg.Database.ConnMaxIdleTime != 2*time.Minute {
		t.Errorf("ConnMaxIdleTime = %v, want 2m", cfg.Database.ConnMaxIdleTime)
	}
}

func TestLoad_JWTSecretValidation_Production(t *testing.T) {
	os.Setenv("ENVIRONMENT", "production")
	os.Setenv("JWT_SECRET", "your-super-secret-jwt-key-change-in-production")
	defer clearEnv()

	_, err := Load()
	if err == nil {
		t.Error("Load() should fail in production with default JWT secret")
	}
}

func TestLoad_JWTSecretValidation_Development(t *testing.T) {
	os.Setenv("ENVIRONMENT", "development")
	os.Setenv("JWT_SECRET", "your-super-secret-jwt-key-change-in-production")
	defer clearEnv()

	_, err := Load()
	if err != nil {
		t.Errorf("Load() should succeed in development with default JWT secret: %v", err)
	}
}

func TestGetEnv(t *testing.T) {
	tests := []struct {
		name         string
		key          string
		defaultValue string
		envValue     string
		want         string
	}{
		{
			name:         "environment variable set",
			key:          "TEST_KEY",
			defaultValue: "default",
			envValue:     "custom",
			want:         "custom",
		},
		{
			name:         "environment variable not set",
			key:          "TEST_KEY_UNSET",
			defaultValue: "default",
			envValue:     "",
			want:         "default",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.envValue != "" {
				os.Setenv(tt.key, tt.envValue)
				defer os.Unsetenv(tt.key)
			}

			got := getEnv(tt.key, tt.defaultValue)
			if got != tt.want {
				t.Errorf("getEnv() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetEnvAsInt(t *testing.T) {
	tests := []struct {
		name         string
		key          string
		defaultValue int
		envValue     string
		want         int
	}{
		{
			name:         "valid integer",
			key:          "TEST_INT",
			defaultValue: 10,
			envValue:     "20",
			want:         20,
		},
		{
			name:         "invalid integer",
			key:          "TEST_INT_INVALID",
			defaultValue: 10,
			envValue:     "invalid",
			want:         10,
		},
		{
			name:         "not set",
			key:          "TEST_INT_UNSET",
			defaultValue: 10,
			envValue:     "",
			want:         10,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.envValue != "" {
				os.Setenv(tt.key, tt.envValue)
				defer os.Unsetenv(tt.key)
			}

			got := getEnvAsInt(tt.key, tt.defaultValue)
			if got != tt.want {
				t.Errorf("getEnvAsInt() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetEnvAsInt64(t *testing.T) {
	tests := []struct {
		name         string
		key          string
		defaultValue int64
		envValue     string
		want         int64
	}{
		{
			name:         "valid int64",
			key:          "TEST_INT64",
			defaultValue: 1000,
			envValue:     "2000",
			want:         2000,
		},
		{
			name:         "invalid int64",
			key:          "TEST_INT64_INVALID",
			defaultValue: 1000,
			envValue:     "invalid",
			want:         1000,
		},
		{
			name:         "large number",
			key:          "TEST_INT64_LARGE",
			defaultValue: 1000,
			envValue:     "104857600",
			want:         104857600,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.envValue != "" {
				os.Setenv(tt.key, tt.envValue)
				defer os.Unsetenv(tt.key)
			}

			got := getEnvAsInt64(tt.key, tt.defaultValue)
			if got != tt.want {
				t.Errorf("getEnvAsInt64() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetEnvAsDuration(t *testing.T) {
	tests := []struct {
		name         string
		key          string
		defaultValue time.Duration
		envValue     string
		want         time.Duration
	}{
		{
			name:         "valid duration",
			key:          "TEST_DURATION",
			defaultValue: 5 * time.Minute,
			envValue:     "10m",
			want:         10 * time.Minute,
		},
		{
			name:         "invalid duration",
			key:          "TEST_DURATION_INVALID",
			defaultValue: 5 * time.Minute,
			envValue:     "invalid",
			want:         5 * time.Minute,
		},
		{
			name:         "seconds duration",
			key:          "TEST_DURATION_SECONDS",
			defaultValue: 5 * time.Minute,
			envValue:     "30s",
			want:         30 * time.Second,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.envValue != "" {
				os.Setenv(tt.key, tt.envValue)
				defer os.Unsetenv(tt.key)
			}

			got := getEnvAsDuration(tt.key, tt.defaultValue)
			if got != tt.want {
				t.Errorf("getEnvAsDuration() = %v, want %v", got, tt.want)
			}
		})
	}
}

func clearEnv() {
	envVars := []string{
		"SERVER_HOST", "SERVER_PORT", "GRPC_HOST", "GRPC_PORT",
		"DB_HOST", "DB_PORT", "DB_USER", "DB_PASSWORD", "DB_NAME", "DB_SSLMODE",
		"DB_MAX_OPEN_CONNS", "DB_MAX_IDLE_CONNS", "DB_CONN_MAX_LIFETIME", "DB_CONN_MAX_IDLE_TIME",
		"KAFKA_BROKERS", "KAFKA_CONSUMER_GROUP",
		"REDIS_HOST", "REDIS_PORT",
		"USER_AUTH_SERVICE_ENDPOINT",
		"WEBSOCKET_HOST", "WEBSOCKET_PORT",
		"MAX_FILE_SIZE", "UPLOAD_DIR",
		"WORKER_CONCURRENCY", "BATCH_SIZE", "MAX_RETRIES", "RETRY_BACKOFF_MS",
		"JWT_SECRET", "OTEL_EXPORTER_OTLP_ENDPOINT", "LOG_LEVEL", "ENVIRONMENT",
	}

	for _, key := range envVars {
		os.Unsetenv(key)
	}
}
