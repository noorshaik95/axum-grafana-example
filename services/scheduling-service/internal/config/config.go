// Package config loads runtime settings for scheduling-service from the environment.
package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

type Config struct {
	Server        ServerConfig
	Database      DatabaseConfig
	GRPC          GRPCConfig
	Redis         RedisConfig
	Kafka         KafkaConfig
	Tenant        TenantConfig
	Observability ObservabilityConfig
}

type ServerConfig struct {
	Host string
	Port int
}

type DatabaseConfig struct {
	Host     string
	Port     int
	User     string
	Password string
	DBName   string
	SSLMode  string
}

type GRPCConfig struct {
	Host string
	Port int
}

type RedisConfig struct {
	Addr     string
	Password string
	DB       int
	Enabled  bool
}

type KafkaConfig struct {
	Brokers []string
	Enabled bool
	GroupID string
}

type TenantConfig struct {
	Slug string
}

type ObservabilityConfig struct {
	OTLPEndpoint string
	OTLPInsecure bool
}

func Load() (*Config, error) {
	return &Config{
		Server: ServerConfig{
			Host: getEnv("SERVER_HOST", "0.0.0.0"),
			Port: getEnvAsInt("SERVER_PORT", 8093),
		},
		Database: DatabaseConfig{
			Host:     getEnv("DB_HOST", "localhost"),
			Port:     getEnvAsInt("DB_PORT", 5432),
			User:     getEnv("DB_USER", "postgres"),
			Password: getEnv("DB_PASSWORD", "postgres"),
			DBName:   getEnv("DB_NAME", "scheduling"),
			SSLMode:  getEnv("DB_SSLMODE", "disable"),
		},
		GRPC: GRPCConfig{
			Host: getEnv("GRPC_HOST", "0.0.0.0"),
			Port: getEnvAsInt("GRPC_PORT", 50063),
		},
		Redis: RedisConfig{
			Addr:     getEnv("REDIS_ADDR", "redis:6379"),
			Password: getEnv("REDIS_PASSWORD", ""),
			DB:       getEnvAsInt("REDIS_DB", 0),
			Enabled:  getEnvAsBool("REDIS_ENABLED", true),
		},
		Kafka: KafkaConfig{
			Brokers: getEnvAsSlice("KAFKA_BROKERS", []string{"kafka:9092"}),
			Enabled: getEnvAsBool("KAFKA_ENABLED", false),
			GroupID: getEnv("KAFKA_GROUP_ID", "scheduling-service"),
		},
		Tenant: TenantConfig{
			Slug: getEnv("TENANT_SLUG", "default"),
		},
		Observability: ObservabilityConfig{
			OTLPEndpoint: getEnv("OTEL_EXPORTER_OTLP_ENDPOINT", "tempo:4317"),
			OTLPInsecure: getEnvAsBool("OTEL_EXPORTER_OTLP_INSECURE", true),
		},
	}, nil
}

func (c *DatabaseConfig) DSN() string {
	return fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		c.Host, c.Port, c.User, c.Password, c.DBName, c.SSLMode)
}

func (c *GRPCConfig) Address() string  { return fmt.Sprintf("%s:%d", c.Host, c.Port) }
func (c *ServerConfig) Address() string { return fmt.Sprintf("%s:%d", c.Host, c.Port) }

func getEnv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func getEnvAsInt(key string, def int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return def
}

func getEnvAsBool(key string, def bool) bool {
	if v := os.Getenv(key); v != "" {
		if b, err := strconv.ParseBool(v); err == nil {
			return b
		}
	}
	return def
}

func getEnvAsSlice(key string, def []string) []string {
	if v := os.Getenv(key); v != "" {
		parts := strings.Split(v, ",")
		out := make([]string, 0, len(parts))
		for _, p := range parts {
			p = strings.TrimSpace(p)
			if p != "" {
				out = append(out, p)
			}
		}
		if len(out) > 0 {
			return out
		}
	}
	return def
}
