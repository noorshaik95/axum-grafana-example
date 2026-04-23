package config

import (
	"fmt"
	"os"
	"strconv"
)

type Config struct {
	GRPC          GRPCConfig
	Redis         RedisConfig
	Kafka         KafkaConfig
	Anthropic     AnthropicConfig
	Budget        BudgetConfig
	Observability ObservabilityConfig
	TenantSlug    string
}

type GRPCConfig struct {
	Host string
	Port int
}

type RedisConfig struct {
	Addr     string
	Password string
	DB       int
}

type KafkaConfig struct {
	Brokers []string
	Enabled bool
	GroupID string
}

type AnthropicConfig struct {
	APIKey           string
	DefaultModel     string
	StudyPlanModel   string
	WelcomeMaxTokens int
	CmdPaletteMaxTokens int
	StudyPlanMaxTokens  int
	FeedbackMaxTokens   int
}

type BudgetConfig struct {
	MonthlyTokenBudget int64
}

type ObservabilityConfig struct {
	OTLPEndpoint string
	OTLPInsecure bool
}

func Load() (*Config, error) {
	return &Config{
		GRPC: GRPCConfig{
			Host: getEnv("GRPC_HOST", "0.0.0.0"),
			Port: getEnvAsInt("GRPC_PORT", 50064),
		},
		Redis: RedisConfig{
			Addr:     getEnv("REDIS_ADDR", "localhost:6379"),
			Password: getEnv("REDIS_PASSWORD", ""),
			DB:       getEnvAsInt("REDIS_DB", 0),
		},
		Kafka: KafkaConfig{
			Brokers: []string{getEnv("KAFKA_BROKERS", "localhost:9092")},
			Enabled: getEnvAsBool("KAFKA_ENABLED", true),
			GroupID: getEnv("KAFKA_GROUP_ID", "ai-service"),
		},
		Anthropic: AnthropicConfig{
			APIKey:              getEnv("ANTHROPIC_API_KEY", ""),
			DefaultModel:        getEnv("ANTHROPIC_DEFAULT_MODEL", "claude-sonnet-4-6"),
			StudyPlanModel:      getEnv("ANTHROPIC_STUDY_PLAN_MODEL", "claude-opus-4-7"),
			WelcomeMaxTokens:    getEnvAsInt("ANTHROPIC_WELCOME_MAX_TOKENS", 120),
			CmdPaletteMaxTokens: getEnvAsInt("ANTHROPIC_CMD_PALETTE_MAX_TOKENS", 400),
			StudyPlanMaxTokens:  getEnvAsInt("ANTHROPIC_STUDY_PLAN_MAX_TOKENS", 8000),
			FeedbackMaxTokens:   getEnvAsInt("ANTHROPIC_FEEDBACK_MAX_TOKENS", 1200),
		},
		Budget: BudgetConfig{
			MonthlyTokenBudget: int64(getEnvAsInt("TENANT_AI_TOKEN_BUDGET", 1_000_000)),
		},
		Observability: ObservabilityConfig{
			OTLPEndpoint: getEnv("OTEL_EXPORTER_OTLP_ENDPOINT", "tempo:4317"),
			OTLPInsecure: getEnvAsBool("OTEL_EXPORTER_OTLP_INSECURE", true),
		},
		TenantSlug: getEnv("TENANT_SLUG", "default"),
	}, nil
}

func (c *GRPCConfig) Address() string {
	return fmt.Sprintf("%s:%d", c.Host, c.Port)
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvAsInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

func getEnvAsBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if boolValue, err := strconv.ParseBool(value); err == nil {
			return boolValue
		}
	}
	return defaultValue
}
