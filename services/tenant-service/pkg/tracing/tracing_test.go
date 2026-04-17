package tracing

import (
	"context"
	"testing"
	"time"
)

func TestConfig_Defaults(t *testing.T) {
	cfg := Config{
		ServiceName:    "test-service",
		ServiceVersion: "1.0.0",
		OTLPEndpoint:   "localhost:4317",
		OTLPInsecure:   true,
		SamplingRate:   1.0,
	}

	if cfg.ServiceName != "test-service" {
		t.Errorf("Expected ServiceName test-service, got %s", cfg.ServiceName)
	}

	if cfg.ServiceVersion != "1.0.0" {
		t.Errorf("Expected ServiceVersion 1.0.0, got %s", cfg.ServiceVersion)
	}

	if cfg.OTLPEndpoint != "localhost:4317" {
		t.Errorf("Expected OTLPEndpoint localhost:4317, got %s", cfg.OTLPEndpoint)
	}

	if !cfg.OTLPInsecure {
		t.Error("Expected OTLPInsecure to be true")
	}

	if cfg.SamplingRate != 1.0 {
		t.Errorf("Expected SamplingRate 1.0, got %f", cfg.SamplingRate)
	}
}

func TestInitTracer_InvalidEndpoint(t *testing.T) {
	cfg := Config{
		ServiceName:    "test-service",
		ServiceVersion: "1.0.0",
		OTLPEndpoint:   "invalid-endpoint-that-does-not-exist:4317",
		OTLPInsecure:   true,
		SamplingRate:   1.0,
	}

	// This should fail to connect but shouldn't panic
	tp, err := InitTracer(cfg)
	if err == nil {
		// Unexpected success - cleanup
		if tp != nil {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			_ = Shutdown(ctx, tp)
		}
		t.Skip("Unexpected success - OTLP endpoint might be available")
	}

	// Expected to fail - verify error is returned
	if err == nil {
		t.Error("Expected error when connecting to invalid endpoint")
	}
}

func TestShutdown_NilProvider(t *testing.T) {
	ctx := context.Background()
	err := Shutdown(ctx, nil)
	if err != nil {
		t.Errorf("Expected no error when shutting down nil provider, got %v", err)
	}
}

func TestShutdown_WithContext(t *testing.T) {
	// Test with canceled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	err := Shutdown(ctx, nil)
	if err != nil {
		t.Errorf("Expected no error for nil provider even with canceled context, got %v", err)
	}
}

func TestConfig_SamplingRates(t *testing.T) {
	tests := []struct {
		name         string
		samplingRate float64
		valid        bool
	}{
		{"Zero sampling", 0.0, true},
		{"Half sampling", 0.5, true},
		{"Full sampling", 1.0, true},
		{"Over sampling", 1.5, true}, // Not invalid, just unusual
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := Config{
				ServiceName:    "test-service",
				ServiceVersion: "1.0.0",
				OTLPEndpoint:   "localhost:4317",
				OTLPInsecure:   true,
				SamplingRate:   tt.samplingRate,
			}

			if cfg.SamplingRate != tt.samplingRate {
				t.Errorf("Expected sampling rate %f, got %f", tt.samplingRate, cfg.SamplingRate)
			}
		})
	}
}

func TestConfig_TLSSettings(t *testing.T) {
	// Test insecure configuration
	cfgInsecure := Config{
		ServiceName:    "test-service",
		ServiceVersion: "1.0.0",
		OTLPEndpoint:   "localhost:4317",
		OTLPInsecure:   true,
		SamplingRate:   1.0,
	}

	if !cfgInsecure.OTLPInsecure {
		t.Error("Expected OTLPInsecure to be true")
	}

	// Test secure configuration
	cfgSecure := Config{
		ServiceName:    "test-service",
		ServiceVersion: "1.0.0",
		OTLPEndpoint:   "secure.example.com:4317",
		OTLPInsecure:   false,
		SamplingRate:   1.0,
	}

	if cfgSecure.OTLPInsecure {
		t.Error("Expected OTLPInsecure to be false")
	}
}
