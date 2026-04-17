package tracing

import (
	"context"
	"time"
)

// InitTracer initialises an OTLP gRPC tracer pointing at otlpEndpoint and
// returns a shutdown function that flushes and closes the provider.
//
// This is the preferred entry-point for services that only need a service name
// and the Tempo/collector endpoint:
//
//	shutdown, err := tracing.InitTracer("my-service", "tempo:4317")
//	if err != nil { log.Fatal(err) }
//	defer shutdown()
//
// For finer control (sampling rate, TLS, version) use InitTracerWithConfig.
func InitTracer(serviceName, otlpEndpoint string) (func(), error) {
	tp, err := InitTracerWithConfig(Config{
		ServiceName:    serviceName,
		ServiceVersion: "unknown",
		OTLPEndpoint:   otlpEndpoint,
		OTLPInsecure:   true,
		SamplingRate:   1.0,
	})
	if err != nil {
		return func() {}, err
	}

	shutdown := func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = Shutdown(ctx, tp)
	}
	return shutdown, nil
}
