// Package config loads runtime configuration for admin-auth-service from env.
package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

// Config is the full runtime configuration.
type Config struct {
	Service                 string
	Environment             string
	LogLevel                string
	GRPCPort                int
	HTTPPort                int
	DatabaseDSN             string
	RedisURL                string
	KafkaBrokers            []string
	JWTSecret               string
	AccessTokenTTL          time.Duration
	RefreshTokenTTL         time.Duration
	PlatformPrivateKeyPath  string // RSA PEM for impersonation signing
	PlatformPublicKeyPath   string // RSA PEM for verify (test helper)
	ImpersonationTTL        time.Duration
	ImpersonationRedirectFn string // "http://{slug}.slate.local/auth/impersonate?token={token}"
	// TenantDBDSN is the DSN of the shared tenant database (tenants_v2 table).
	// Used by the Impersonate handler to resolve a tenant_id → slug when the
	// caller does not supply tenant_slug explicitly. Optional — if empty the
	// slug lookup is skipped and the caller MUST supply tenant_slug.
	TenantDBDSN  string
	OTLPEndpoint string
}

// Load reads config from the environment, applying safe defaults for dev.
func Load() (*Config, error) {
	cfg := &Config{
		Service:                 "admin-auth-service",
		Environment:             getenv("ENVIRONMENT", "development"),
		LogLevel:                getenv("LOG_LEVEL", "info"),
		GRPCPort:                mustAtoi(getenv("GRPC_PORT", "50060")),
		HTTPPort:                mustAtoi(getenv("HTTP_PORT", "8090")),
		DatabaseDSN:             getenv("DB_DSN", "postgres://postgres:postgres@postgres:5432/adminauth?sslmode=disable"),
		RedisURL:                getenv("REDIS_URL", "redis://redis:6379"),
		KafkaBrokers:            splitCsv(getenv("KAFKA_BROKERS", "kafka:9092")),
		JWTSecret:               os.Getenv("JWT_SECRET"),
		AccessTokenTTL:          mustDuration(getenv("ACCESS_TOKEN_TTL", "1h")),
		RefreshTokenTTL:         mustDuration(getenv("REFRESH_TOKEN_TTL", "168h")),
		PlatformPrivateKeyPath:  getenv("PLATFORM_PRIVATE_KEY_PATH", "/run/secrets/platform_private_key"),
		PlatformPublicKeyPath:   os.Getenv("PLATFORM_PUBLIC_KEY_PATH"),
		ImpersonationTTL:        mustDuration(getenv("IMPERSONATION_TTL", "5m")),
		ImpersonationRedirectFn: getenv("IMPERSONATION_REDIRECT_TEMPLATE", "http://{slug}.slate.local/auth/impersonate?token={token}"),
		TenantDBDSN:             os.Getenv("TENANT_DB_DSN"),
		OTLPEndpoint:            getenv("OTEL_EXPORTER_OTLP_ENDPOINT", "tempo:4317"),
	}

	if cfg.JWTSecret == "" {
		// Default only in dev; production must set this explicitly.
		if cfg.Environment == "production" {
			return nil, fmt.Errorf("JWT_SECRET is required in production")
		}
		cfg.JWTSecret = "dev-admin-auth-secret-do-not-use-in-prod-123456789"
	}
	return cfg, nil
}

func (c *Config) GRPCAddress() string { return fmt.Sprintf(":%d", c.GRPCPort) }
func (c *Config) HTTPAddress() string { return fmt.Sprintf(":%d", c.HTTPPort) }

func getenv(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

func mustAtoi(s string) int {
	n, err := strconv.Atoi(s)
	if err != nil {
		return 0
	}
	return n
}

func mustDuration(s string) time.Duration {
	d, err := time.ParseDuration(s)
	if err != nil {
		return 0
	}
	return d
}

func splitCsv(s string) []string {
	if s == "" {
		return nil
	}
	out := []string{}
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == ',' {
			if i > start {
				out = append(out, s[start:i])
			}
			start = i + 1
		}
	}
	if start < len(s) {
		out = append(out, s[start:])
	}
	return out
}
