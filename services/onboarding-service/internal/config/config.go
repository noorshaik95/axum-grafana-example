package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

// Config holds all configuration for the onboarding service
type Config struct {
	Server    ServerConfig
	Database  DatabaseConfig
	Kafka     KafkaConfig
	Redis     RedisConfig
	UserAuth  UserAuthConfig
	WebSocket WebSocketConfig
	Upload    UploadConfig
	Worker    WorkerConfig
	JWT       JWTConfig
	Telemetry TelemetryConfig
	LogLevel  string
}

type ServerConfig struct {
	Host     string
	Port     string
	GRPCHost string
	GRPCPort string
}

type DatabaseConfig struct {
	Host            string
	Port            string
	User            string
	Password        string
	DBName          string
	SSLMode         string
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
	ConnMaxIdleTime time.Duration
}

type KafkaConfig struct {
	Brokers       []string
	ConsumerGroup string
}

type RedisConfig struct {
	Host string
	Port string
}

type UserAuthConfig struct {
	Endpoint string
}

type WebSocketConfig struct {
	Host string
	Port string
}

type UploadConfig struct {
	MaxFileSize int64
	UploadDir   string
}

type WorkerConfig struct {
	Concurrency    int
	BatchSize      int
	MaxRetries     int
	RetryBackoffMS int
}

type JWTConfig struct {
	Secret string
}

type TelemetryConfig struct {
	OTLPEndpoint string
}

// Default configuration values
const (
	// Database defaults
	defaultMaxOpenConns       = 50
	defaultMaxIdleConns       = 10
	defaultConnMaxLifetimeMin = 5
	defaultConnMaxIdleTimeMin = 1

	// Upload defaults
	defaultMaxFileSize = 104857600 // 100MB in bytes

	// Worker defaults
	defaultWorkerConcurrency = 10
	defaultBatchSize         = 100
	defaultMaxRetries        = 3
	defaultRetryBackoffMS    = 1000

	// ParseInt parameters
	decimalBase  = 10
	int64BitSize = 64
)

// Load loads configuration from environment variables
func Load() (*Config, error) {
	cfg := &Config{
		Server: ServerConfig{
			Host:     getEnv("SERVER_HOST", "0.0.0.0"),
			Port:     getEnv("SERVER_PORT", "8082"),
			GRPCHost: getEnv("GRPC_HOST", "0.0.0.0"),
			GRPCPort: getEnv("GRPC_PORT", "50052"),
		},
		Database: DatabaseConfig{
			Host:            getEnv("DB_HOST", "localhost"),
			Port:            getEnv("DB_PORT", "5432"),
			User:            getEnv("DB_USER", "postgres"),
			Password:        getEnv("DB_PASSWORD", "postgres"),
			DBName:          getEnv("DB_NAME", "onboarding"),
			SSLMode:         getEnv("DB_SSLMODE", "disable"),
			MaxOpenConns:    getEnvAsInt("DB_MAX_OPEN_CONNS", defaultMaxOpenConns),
			MaxIdleConns:    getEnvAsInt("DB_MAX_IDLE_CONNS", defaultMaxIdleConns),
			ConnMaxLifetime: getEnvAsDuration("DB_CONN_MAX_LIFETIME", defaultConnMaxLifetimeMin*time.Minute),
			ConnMaxIdleTime: getEnvAsDuration("DB_CONN_MAX_IDLE_TIME", defaultConnMaxIdleTimeMin*time.Minute),
		},
		Kafka: KafkaConfig{
			Brokers:       []string{getEnv("KAFKA_BROKERS", "localhost:9092")},
			ConsumerGroup: getEnv("KAFKA_CONSUMER_GROUP", "onboarding-service"),
		},
		Redis: RedisConfig{
			Host: getEnv("REDIS_HOST", "localhost"),
			Port: getEnv("REDIS_PORT", "6379"),
		},
		UserAuth: UserAuthConfig{
			Endpoint: getEnv("USER_AUTH_SERVICE_ENDPOINT", "localhost:50051"),
		},
		WebSocket: WebSocketConfig{
			Host: getEnv("WEBSOCKET_HOST", "0.0.0.0"),
			Port: getEnv("WEBSOCKET_PORT", "8083"),
		},
		Upload: UploadConfig{
			MaxFileSize: getEnvAsInt64("MAX_FILE_SIZE", defaultMaxFileSize),
			UploadDir:   getEnv("UPLOAD_DIR", "/tmp/uploads"),
		},
		Worker: WorkerConfig{
			Concurrency:    getEnvAsInt("WORKER_CONCURRENCY", defaultWorkerConcurrency),
			BatchSize:      getEnvAsInt("BATCH_SIZE", defaultBatchSize),
			MaxRetries:     getEnvAsInt("MAX_RETRIES", defaultMaxRetries),
			RetryBackoffMS: getEnvAsInt("RETRY_BACKOFF_MS", defaultRetryBackoffMS),
		},
		JWT: JWTConfig{
			Secret: getEnv("JWT_SECRET", "your-super-secret-jwt-key-change-in-production"),
		},
		Telemetry: TelemetryConfig{
			OTLPEndpoint: getEnv("OTEL_EXPORTER_OTLP_ENDPOINT", "localhost:4317"),
		},
		LogLevel: getEnv("LOG_LEVEL", "info"),
	}

	// Validate required fields
	if cfg.JWT.Secret == "your-super-secret-jwt-key-change-in-production" && getEnv("ENVIRONMENT", "development") == "production" {
		return nil, fmt.Errorf("JWT_SECRET must be set in production")
	}

	return cfg, nil
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvAsInt(key string, defaultValue int) int {
	valueStr := getEnv(key, "")
	if valueStr == "" {
		return defaultValue
	}
	value, err := strconv.Atoi(valueStr)
	if err != nil {
		return defaultValue
	}
	return value
}

func getEnvAsInt64(key string, defaultValue int64) int64 {
	valueStr := getEnv(key, "")
	if valueStr == "" {
		return defaultValue
	}
	value, err := strconv.ParseInt(valueStr, decimalBase, int64BitSize)
	if err != nil {
		return defaultValue
	}
	return value
}

func getEnvAsDuration(key string, defaultValue time.Duration) time.Duration {
	valueStr := getEnv(key, "")
	if valueStr == "" {
		return defaultValue
	}
	value, err := time.ParseDuration(valueStr)
	if err != nil {
		return defaultValue
	}
	return value
}
