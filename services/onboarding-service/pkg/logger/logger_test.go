package logger

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewLogger(t *testing.T) {
	tests := []struct {
		name          string
		level         string
		expectedLevel string
	}{
		{"debug level", "debug", "debug"},
		{"info level", "info", "info"},
		{"warn level", "warn", "warn"},
		{"error level", "error", "error"},
		{"default level", "invalid", "info"},
		{"empty level", "", "info"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := NewLogger(tt.level)
			assert.NotNil(t, logger)
			assert.Equal(t, tt.expectedLevel, logger.GetLevel())
		})
	}
}

func TestLoggerMethods(t *testing.T) {
	var buf bytes.Buffer
	logger := NewLoggerWithWriter("debug", &buf)

	t.Run("Info logging", func(t *testing.T) {
		buf.Reset()
		logger.Info().Str("key", "value").Msg("test info message")

		var logEntry map[string]interface{}
		err := json.Unmarshal(buf.Bytes(), &logEntry)
		require.NoError(t, err)

		assert.Equal(t, "info", logEntry["level"])
		assert.Equal(t, "test info message", logEntry["message"])
		assert.Equal(t, "value", logEntry["key"])
	})

	t.Run("Error logging", func(t *testing.T) {
		buf.Reset()
		logger.Error().Str("error_type", "test_error").Msg("test error message")

		var logEntry map[string]interface{}
		err := json.Unmarshal(buf.Bytes(), &logEntry)
		require.NoError(t, err)

		assert.Equal(t, "error", logEntry["level"])
		assert.Equal(t, "test error message", logEntry["message"])
		assert.Equal(t, "test_error", logEntry["error_type"])
	})

	t.Run("Warn logging", func(t *testing.T) {
		buf.Reset()
		logger.Warn().Str("warning_type", "test_warning").Msg("test warning message")

		var logEntry map[string]interface{}
		err := json.Unmarshal(buf.Bytes(), &logEntry)
		require.NoError(t, err)

		assert.Equal(t, "warn", logEntry["level"])
		assert.Equal(t, "test warning message", logEntry["message"])
		assert.Equal(t, "test_warning", logEntry["warning_type"])
	})

	t.Run("Debug logging", func(t *testing.T) {
		buf.Reset()
		logger.Debug().Str("debug_info", "test_debug").Msg("test debug message")

		var logEntry map[string]interface{}
		err := json.Unmarshal(buf.Bytes(), &logEntry)
		require.NoError(t, err)

		assert.Equal(t, "debug", logEntry["level"])
		assert.Equal(t, "test debug message", logEntry["message"])
		assert.Equal(t, "test_debug", logEntry["debug_info"])
	})
}

func TestWithContext(t *testing.T) {
	var buf bytes.Buffer
	logger := NewLoggerWithWriter("info", &buf)

	t.Run("context with trace ID", func(t *testing.T) {
		buf.Reset()
		ctx := context.WithValue(context.Background(), TraceIDKey, "test-trace-123")
		logger.WithContext(ctx).Msg("test message with trace")

		var logEntry map[string]interface{}
		err := json.Unmarshal(buf.Bytes(), &logEntry)
		require.NoError(t, err)

		assert.Equal(t, "test-trace-123", logEntry["trace_id"])
		assert.Equal(t, "test message with trace", logEntry["message"])
	})

	t.Run("context without trace ID", func(t *testing.T) {
		buf.Reset()
		ctx := context.Background()
		logger.WithContext(ctx).Msg("test message without trace")

		var logEntry map[string]interface{}
		err := json.Unmarshal(buf.Bytes(), &logEntry)
		require.NoError(t, err)

		_, hasTraceID := logEntry["trace_id"]
		assert.False(t, hasTraceID)
		assert.Equal(t, "test message without trace", logEntry["message"])
	})
}

func TestWithTraceID(t *testing.T) {
	var buf bytes.Buffer
	logger := NewLoggerWithWriter("info", &buf)

	buf.Reset()
	logger.WithTraceID("explicit-trace-456").Msg("test message with explicit trace")

	var logEntry map[string]interface{}
	err := json.Unmarshal(buf.Bytes(), &logEntry)
	require.NoError(t, err)

	assert.Equal(t, "explicit-trace-456", logEntry["trace_id"])
	assert.Equal(t, "test message with explicit trace", logEntry["message"])
}

func TestRedactPassword(t *testing.T) {
	logger := NewLogger("info")

	tests := []struct {
		name     string
		password string
		expected string
	}{
		{"non-empty password", "MySecretPassword123!", "***REDACTED***"},
		{"empty password", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := logger.RedactPassword(tt.password)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestRedactToken(t *testing.T) {
	logger := NewLogger("info")

	tests := []struct {
		name     string
		token    string
		expected string
	}{
		{"long token", "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIn0", "eyJhbGci***REDACTED***"},
		{"short token", "abc123", "***REDACTED***"},
		{"empty token", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := logger.RedactToken(tt.token)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestRedactEmail(t *testing.T) {
	logger := NewLogger("info")

	tests := []struct {
		name     string
		email    string
		expected string
	}{
		{"normal email", "user@example.com", "us***@example.com"},
		{"short email", "a@example.com", "***@example.com"},
		{"invalid email", "notanemail", "***REDACTED***"},
		{"empty email", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := logger.RedactEmail(tt.email)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestJSONOutput(t *testing.T) {
	var buf bytes.Buffer
	logger := NewLoggerWithWriter("info", &buf)

	logger.Info().
		Str("user_id", "123").
		Str("operation", "login").
		Str("error_type", "validation_error").
		Int("status_code", 400).
		Msg("user login failed")

	var logEntry map[string]interface{}
	err := json.Unmarshal(buf.Bytes(), &logEntry)
	require.NoError(t, err)

	// Verify JSON structure
	assert.Equal(t, "info", logEntry["level"])
	assert.Equal(t, "user login failed", logEntry["message"])
	assert.Equal(t, "123", logEntry["user_id"])
	assert.Equal(t, "login", logEntry["operation"])
	assert.Equal(t, "validation_error", logEntry["error_type"])
	assert.Equal(t, float64(400), logEntry["status_code"])

	// Verify timestamp exists
	_, hasTimestamp := logEntry["time"]
	assert.True(t, hasTimestamp)
}

func TestLogLevelFiltering(t *testing.T) {
	var buf bytes.Buffer
	logger := NewLoggerWithWriter("warn", &buf)

	// Debug and Info should not be logged
	logger.Debug().Msg("debug message")
	logger.Info().Msg("info message")
	assert.Empty(t, buf.String())

	// Warn should be logged
	buf.Reset()
	logger.Warn().Msg("warn message")
	assert.True(t, strings.Contains(buf.String(), "warn message"))

	// Error should be logged
	buf.Reset()
	logger.Error().Msg("error message")
	assert.True(t, strings.Contains(buf.String(), "error message"))
}
