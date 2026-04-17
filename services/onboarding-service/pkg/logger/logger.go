package logger

import (
	"context"
	"io"
	"os"
	"strings"
	"time"

	"github.com/rs/zerolog"
)

const (
	// tokenPrefixLength is the number of characters to show from the beginning of a token
	tokenPrefixLength = 8
	// expectedEmailParts is the expected number of parts when splitting an email by @
	expectedEmailParts = 2
	// emailLocalPartPrefixLength is the number of characters to show from the local part of an email
	emailLocalPartPrefixLength = 2
)

// Logger wraps zerolog for structured logging
type Logger struct {
	logger zerolog.Logger
}

// NewLogger creates a new structured logger with the specified level
func NewLogger(level string) *Logger {
	// Parse log level
	logLevel := parseLogLevel(level)

	// Configure zerolog for JSON output
	zerolog.TimeFieldFormat = time.RFC3339Nano

	// Create logger with JSON output to stdout
	logger := zerolog.New(os.Stdout).
		Level(logLevel).
		With().
		Timestamp().
		Logger()

	return &Logger{logger: logger}
}

// NewLoggerWithWriter creates a logger with a custom writer (useful for testing)
func NewLoggerWithWriter(level string, writer io.Writer) *Logger {
	logLevel := parseLogLevel(level)

	zerolog.TimeFieldFormat = time.RFC3339Nano

	logger := zerolog.New(writer).
		Level(logLevel).
		With().
		Timestamp().
		Logger()

	return &Logger{logger: logger}
}

// parseLogLevel converts string level to zerolog.Level
func parseLogLevel(level string) zerolog.Level {
	switch strings.ToLower(level) {
	case "debug":
		return zerolog.DebugLevel
	case "info":
		return zerolog.InfoLevel
	case "warn", "warning":
		return zerolog.WarnLevel
	case "error":
		return zerolog.ErrorLevel
	default:
		return zerolog.InfoLevel
	}
}

// Info returns a new info level event
func (l *Logger) Info() *zerolog.Event {
	return l.logger.Info()
}

// Error returns a new error level event
func (l *Logger) Error() *zerolog.Event {
	return l.logger.Error()
}

// Warn returns a new warning level event
func (l *Logger) Warn() *zerolog.Event {
	return l.logger.Warn()
}

// Debug returns a new debug level event
func (l *Logger) Debug() *zerolog.Event {
	return l.logger.Debug()
}

// WithContext returns a logger event with trace ID extracted from context
func (l *Logger) WithContext(ctx context.Context) *zerolog.Event {
	event := l.logger.Info()

	// Extract trace ID from context if available
	if traceID := extractTraceID(ctx); traceID != "" {
		event = event.Str("trace_id", traceID)
	}

	return event
}

// WithTraceID returns a logger event with the specified trace ID
func (l *Logger) WithTraceID(traceID string) *zerolog.Event {
	return l.logger.Info().Str("trace_id", traceID)
}

// ContextKey is a custom type for context keys to avoid collisions
type ContextKey string

// TraceIDKey is the context key for trace IDs
const TraceIDKey ContextKey = "trace_id"

// extractTraceID extracts trace ID from context
// This will be enhanced when we integrate with OpenTelemetry
func extractTraceID(ctx context.Context) string {
	// TODO: Extract from OpenTelemetry span context
	// For now, check for trace_id in context values
	// Support both typed key (preferred) and string key (legacy)
	if traceID, ok := ctx.Value(TraceIDKey).(string); ok {
		return traceID
	}
	if traceID, ok := ctx.Value("trace_id").(string); ok {
		return traceID
	}
	return ""
}

// RedactPassword redacts sensitive password data for logging
func (l *Logger) RedactPassword(password string) string {
	if password == "" {
		return ""
	}
	return "***REDACTED***"
}

// RedactToken redacts sensitive token data for logging
func (l *Logger) RedactToken(token string) string {
	if token == "" {
		return ""
	}
	// Show first tokenPrefixLength characters for debugging, redact the rest
	if len(token) > tokenPrefixLength {
		return token[:tokenPrefixLength] + "***REDACTED***"
	}
	return "***REDACTED***"
}

// RedactEmail partially redacts email for logging (keeps domain for debugging)
func (l *Logger) RedactEmail(email string) string {
	if email == "" {
		return ""
	}

	// Split email at @
	parts := strings.Split(email, "@")
	if len(parts) != expectedEmailParts {
		return "***REDACTED***"
	}

	// Show first emailLocalPartPrefixLength characters of local part, keep domain
	localPart := parts[0]
	if len(localPart) > emailLocalPartPrefixLength {
		return localPart[:emailLocalPartPrefixLength] + "***@" + parts[1]
	}
	return "***@" + parts[1]
}

// GetLevel returns the current log level
func (l *Logger) GetLevel() string {
	level := l.logger.GetLevel()
	switch level {
	case zerolog.DebugLevel:
		return "debug"
	case zerolog.InfoLevel:
		return "info"
	case zerolog.WarnLevel:
		return "warn"
	case zerolog.ErrorLevel:
		return "error"
	case zerolog.FatalLevel:
		return "fatal"
	case zerolog.PanicLevel:
		return "panic"
	case zerolog.TraceLevel:
		return "trace"
	case zerolog.NoLevel, zerolog.Disabled:
		return "disabled"
	default:
		return "info"
	}
}
