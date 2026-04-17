package tracing

import (
	"context"

	commontracing "slate/libs/common-go/tracing"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

// Config is an alias for the common-go tracing Config.
type Config = commontracing.Config

// InitTracer delegates to common-go's tracing initializer.
func InitTracer(cfg Config) (*sdktrace.TracerProvider, error) {
	return commontracing.InitTracer(cfg)
}

// Shutdown delegates to common-go's tracing shutdown.
func Shutdown(ctx context.Context, tp *sdktrace.TracerProvider) error {
	return commontracing.Shutdown(ctx, tp)
}
