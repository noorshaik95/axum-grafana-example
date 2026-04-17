// Package logging provides a structured logger with OpenTelemetry trace ID injection.
// It wraps zerolog and automatically attaches trace_id to log events when an active
// OTel span is present in the context.
package logging

import (
	"context"
	"io"
	"os"
	"strings"
	"time"

	"github.com/rs/zerolog"
	"go.opentelemetry.io/otel/trace"
)

// Logger wraps zerolog with trace ID injection from OTel span context.
type Logger struct {
	logger  zerolog.Logger
	service string
}

// NewLogger creates a JSON-formatted logger that emits to stdout.
// serviceName is added to every log line as the "service" field.
// level is one of "debug", "info", "warn", "error".
func NewLogger(serviceName, level string) *Logger {
	logLevel := parseLogLevel(level)
	zerolog.TimeFieldFormat = time.RFC3339Nano

	logger := zerolog.New(os.Stdout).
		Level(logLevel).
		With().
		Timestamp().
		Str("service", serviceName).
		Logger()

	return &Logger{logger: logger, service: serviceName}
}

// NewLoggerWithWriter creates a logger that writes to the supplied writer (useful for tests).
func NewLoggerWithWriter(serviceName, level string, w io.Writer) *Logger {
	logLevel := parseLogLevel(level)
	zerolog.TimeFieldFormat = time.RFC3339Nano

	logger := zerolog.New(w).
		Level(logLevel).
		With().
		Timestamp().
		Str("service", serviceName).
		Logger()

	return &Logger{logger: logger, service: serviceName}
}

// Info returns an info-level event.
func (l *Logger) Info() *zerolog.Event { return l.logger.Info() }

// Error returns an error-level event.
func (l *Logger) Error() *zerolog.Event { return l.logger.Error() }

// Warn returns a warn-level event.
func (l *Logger) Warn() *zerolog.Event { return l.logger.Warn() }

// Debug returns a debug-level event.
func (l *Logger) Debug() *zerolog.Event { return l.logger.Debug() }

// WithContext returns an info-level event enriched with trace_id from the OTel
// span stored in ctx. Use this for all log calls that have access to a context.
func (l *Logger) WithContext(ctx context.Context) *zerolog.Event {
	event := l.logger.Info()
	if id := extractTraceID(ctx); id != "" {
		event = event.Str("trace_id", id)
	}
	return event
}

// ErrorWithContext returns an error-level event with trace_id.
func (l *Logger) ErrorWithContext(ctx context.Context) *zerolog.Event {
	event := l.logger.Error()
	if id := extractTraceID(ctx); id != "" {
		event = event.Str("trace_id", id)
	}
	return event
}

// WarnWithContext returns a warn-level event with trace_id.
func (l *Logger) WarnWithContext(ctx context.Context) *zerolog.Event {
	event := l.logger.Warn()
	if id := extractTraceID(ctx); id != "" {
		event = event.Str("trace_id", id)
	}
	return event
}

// DebugWithContext returns a debug-level event with trace_id.
func (l *Logger) DebugWithContext(ctx context.Context) *zerolog.Event {
	event := l.logger.Debug()
	if id := extractTraceID(ctx); id != "" {
		event = event.Str("trace_id", id)
	}
	return event
}

// WithTraceID returns an info-level event with the given trace ID string.
func (l *Logger) WithTraceID(traceID string) *zerolog.Event {
	return l.logger.Info().Str("trace_id", traceID)
}

// Zerolog returns the underlying zerolog.Logger for callers that need it directly.
func (l *Logger) Zerolog() zerolog.Logger { return l.logger }

// extractTraceID pulls the hex trace ID from the OTel span context stored in ctx.
// Returns empty string when no valid span is present.
func extractTraceID(ctx context.Context) string {
	span := trace.SpanFromContext(ctx)
	if span.SpanContext().IsValid() {
		return span.SpanContext().TraceID().String()
	}
	return ""
}

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
