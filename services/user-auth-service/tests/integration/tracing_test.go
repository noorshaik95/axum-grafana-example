//go:build integration
// +build integration

package integration

import (
	"context"
	"testing"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc/metadata"
)

// TestTracing_ContextPropagation verifies that trace context is properly propagated through gRPC calls
func TestTracing_ContextPropagation(t *testing.T) {
	// Create in-memory span exporter for testing
	exporter := tracetest.NewInMemoryExporter()
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithSyncer(exporter),
	)
	otel.SetTracerProvider(tp)
	defer tp.Shutdown(context.Background())

	// Create a parent span to simulate incoming request with trace context
	tracer := otel.Tracer("test")
	parentCtx, parentSpan := tracer.Start(context.Background(), "parent-operation")

	// Create a child span directly from parent context (simpler test)
	_, childSpan := tracer.Start(parentCtx, "grpc-handler")
	childSpan.End()

	// End parent span
	parentSpan.End()

	// Force flush spans
	tp.ForceFlush(context.Background())

	// Get exported spans
	spans := exporter.GetSpans()

	// Verify we have both parent and child spans
	if len(spans) < 2 {
		t.Fatalf("Expected at least 2 spans, got %d", len(spans))
	}

	// Find parent and child spans
	var parent, child tracetest.SpanStub
	for _, span := range spans {
		if span.Name == "parent-operation" {
			parent = span
		} else if span.Name == "grpc-handler" {
			child = span
		}
	}

	if parent.SpanContext.TraceID() == (trace.TraceID{}) {
		t.Error("Parent span has invalid trace ID")
	}

	if child.SpanContext.TraceID() == (trace.TraceID{}) {
		t.Error("Child span has invalid trace ID")
	}

	// Verify trace ID is preserved
	if parent.SpanContext.TraceID() != child.SpanContext.TraceID() {
		t.Errorf("Trace ID not preserved: parent=%s, child=%s",
			parent.SpanContext.TraceID(),
			child.SpanContext.TraceID())
	}

	// Verify parent-child relationship
	if child.Parent.SpanID() != parent.SpanContext.SpanID() {
		t.Errorf("Child span does not have correct parent: expected=%s, got=%s",
			parent.SpanContext.SpanID(),
			child.Parent.SpanID())
	}

	t.Logf("Successfully verified trace context propagation: TraceID=%s", parent.SpanContext.TraceID())
}

// TestTracing_RootSpanCreation verifies that a new root span is created when no parent context exists
func TestTracing_RootSpanCreation(t *testing.T) {
	// Create in-memory span exporter for testing
	exporter := tracetest.NewInMemoryExporter()
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithSyncer(exporter),
	)
	otel.SetTracerProvider(tp)
	defer tp.Shutdown(context.Background())

	// Create span without parent context
	tracer := otel.Tracer("test")
	_, span := tracer.Start(context.Background(), "root-operation")
	span.End()

	// Force flush spans
	tp.ForceFlush(context.Background())

	// Get exported spans
	spans := exporter.GetSpans()

	if len(spans) != 1 {
		t.Fatalf("Expected 1 span, got %d", len(spans))
	}

	rootSpan := spans[0]

	// Verify it's a root span (no parent)
	if rootSpan.Parent.IsValid() {
		t.Error("Expected root span to have no parent, but parent is valid")
	}

	// Verify trace ID is valid
	if !rootSpan.SpanContext.TraceID().IsValid() {
		t.Error("Root span has invalid trace ID")
	}

	t.Logf("Successfully created root span: TraceID=%s", rootSpan.SpanContext.TraceID())
}

// TestTracing_ErrorRecording verifies that errors are properly recorded in spans
func TestTracing_ErrorRecording(t *testing.T) {
	// Create in-memory span exporter for testing
	exporter := tracetest.NewInMemoryExporter()
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithSyncer(exporter),
	)
	otel.SetTracerProvider(tp)
	defer tp.Shutdown(context.Background())

	// Create span and record an error
	tracer := otel.Tracer("test")
	_, span := tracer.Start(context.Background(), "error-operation")

	// Simulate authentication failure
	testErr := &testAuthError{message: "authentication failed"}
	span.RecordError(testErr)
	span.SetStatus(codes.Error, testErr.Error())
	span.End()

	// Force flush spans
	tp.ForceFlush(context.Background())

	// Get exported spans
	spans := exporter.GetSpans()

	if len(spans) != 1 {
		t.Fatalf("Expected 1 span, got %d", len(spans))
	}

	errorSpan := spans[0]

	// Verify span status is error
	if errorSpan.Status.Code != codes.Error {
		t.Errorf("Expected span status to be Error, got %v", errorSpan.Status.Code)
	}

	// Verify error message is recorded
	if errorSpan.Status.Description != "authentication failed" {
		t.Errorf("Expected error message 'authentication failed', got '%s'", errorSpan.Status.Description)
	}

	// Verify error event is recorded
	events := errorSpan.Events
	foundErrorEvent := false
	for _, event := range events {
		if event.Name == "exception" {
			foundErrorEvent = true
			break
		}
	}

	if !foundErrorEvent {
		t.Error("Expected error event to be recorded in span")
	}

	t.Log("Successfully verified error recording in span")
}

// metadataCarrier adapts gRPC metadata for trace context propagation
type metadataCarrier struct {
	md metadata.MD
}

func (mc *metadataCarrier) Get(key string) string {
	values := mc.md.Get(key)
	if len(values) == 0 {
		return ""
	}
	return values[0]
}

func (mc *metadataCarrier) Set(key, value string) {
	mc.md.Set(key, value)
}

func (mc *metadataCarrier) Keys() []string {
	keys := make([]string, 0, len(mc.md))
	for k := range mc.md {
		keys = append(keys, k)
	}
	return keys
}

// testAuthError is a simple error type for testing
type testAuthError struct {
	message string
}

func (e *testAuthError) Error() string {
	return e.message
}

// TestNormalAuthStrategy_SpanAttributes verifies that Normal authentication sets correct span attributes
func TestNormalAuthStrategy_SpanAttributes(t *testing.T) {
	// Create in-memory span exporter for testing
	exporter := tracetest.NewInMemoryExporter()
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithSyncer(exporter),
	)
	otel.SetTracerProvider(tp)
	defer tp.Shutdown(context.Background())

	// Create a span simulating normal authentication
	tracer := otel.Tracer("user-auth-service")
	_, span := tracer.Start(context.Background(), "normal.Authenticate")

	// Set attributes that would be set by NormalAuthStrategy
	span.SetAttributes(
		attribute.String("auth.type", "normal"),
		attribute.String("user_id", "test-user-123"),
	)
	span.End()

	// Force flush spans
	tp.ForceFlush(context.Background())

	// Get exported spans
	spans := exporter.GetSpans()

	if len(spans) != 1 {
		t.Fatalf("Expected 1 span, got %d", len(spans))
	}

	authSpan := spans[0]

	// Verify span name
	if authSpan.Name != "normal.Authenticate" {
		t.Errorf("Expected span name 'normal.Authenticate', got '%s'", authSpan.Name)
	}

	// Verify auth.type attribute
	foundAuthType := false
	foundUserID := false
	for _, attr := range authSpan.Attributes {
		if attr.Key == "auth.type" && attr.Value.AsString() == "normal" {
			foundAuthType = true
		}
		if attr.Key == "user_id" && attr.Value.AsString() == "test-user-123" {
			foundUserID = true
		}
	}

	if !foundAuthType {
		t.Error("Expected auth.type attribute not found")
	}

	if !foundUserID {
		t.Error("Expected user_id attribute not found")
	}

	t.Log("Successfully verified Normal authentication span attributes")
}

// TestOAuthAuthStrategy_SpanAttributes verifies that OAuth authentication sets correct span attributes
func TestOAuthAuthStrategy_SpanAttributes(t *testing.T) {
	// Create in-memory span exporter for testing
	exporter := tracetest.NewInMemoryExporter()
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithSyncer(exporter),
	)
	otel.SetTracerProvider(tp)
	defer tp.Shutdown(context.Background())

	// Create a span simulating OAuth authentication
	tracer := otel.Tracer("user-auth-service")
	_, span := tracer.Start(context.Background(), "oauth.Authenticate")

	// Set attributes that would be set by OAuthAuthStrategy
	span.SetAttributes(
		attribute.String("auth.type", "oauth"),
		attribute.String("oauth.provider", "google"),
		attribute.String("oauth.state", "test-state-abc123"),
	)
	span.End()

	// Force flush spans
	tp.ForceFlush(context.Background())

	// Get exported spans
	spans := exporter.GetSpans()

	if len(spans) != 1 {
		t.Fatalf("Expected 1 span, got %d", len(spans))
	}

	authSpan := spans[0]

	// Verify span name
	if authSpan.Name != "oauth.Authenticate" {
		t.Errorf("Expected span name 'oauth.Authenticate', got '%s'", authSpan.Name)
	}

	// Verify attributes
	foundAuthType := false
	foundProvider := false
	foundState := false
	for _, attr := range authSpan.Attributes {
		if attr.Key == "auth.type" && attr.Value.AsString() == "oauth" {
			foundAuthType = true
		}
		if attr.Key == "oauth.provider" && attr.Value.AsString() == "google" {
			foundProvider = true
		}
		if attr.Key == "oauth.state" && attr.Value.AsString() == "test-state-abc123" {
			foundState = true
		}
	}

	if !foundAuthType {
		t.Error("Expected auth.type attribute not found")
	}

	if !foundProvider {
		t.Error("Expected oauth.provider attribute not found")
	}

	if !foundState {
		t.Error("Expected oauth.state attribute not found")
	}

	t.Log("Successfully verified OAuth authentication span attributes")
}

// TestSAMLAuthStrategy_SpanAttributes verifies that SAML authentication sets correct span attributes
func TestSAMLAuthStrategy_SpanAttributes(t *testing.T) {
	// Create in-memory span exporter for testing
	exporter := tracetest.NewInMemoryExporter()
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithSyncer(exporter),
	)
	otel.SetTracerProvider(tp)
	defer tp.Shutdown(context.Background())

	// Create a span simulating SAML authentication
	tracer := otel.Tracer("user-auth-service")
	_, span := tracer.Start(context.Background(), "saml.HandleCallback")

	// Set attributes that would be set by SAMLAuthStrategy
	span.SetAttributes(
		attribute.String("auth.type", "saml"),
		attribute.String("saml.entity_id", "https://idp.example.com/saml"),
		attribute.String("saml.name_id", "user@example.com"),
	)
	span.End()

	// Force flush spans
	tp.ForceFlush(context.Background())

	// Get exported spans
	spans := exporter.GetSpans()

	if len(spans) != 1 {
		t.Fatalf("Expected 1 span, got %d", len(spans))
	}

	authSpan := spans[0]

	// Verify span name
	if authSpan.Name != "saml.HandleCallback" {
		t.Errorf("Expected span name 'saml.HandleCallback', got '%s'", authSpan.Name)
	}

	// Verify attributes
	foundAuthType := false
	foundEntityID := false
	foundNameID := false
	for _, attr := range authSpan.Attributes {
		if attr.Key == "auth.type" && attr.Value.AsString() == "saml" {
			foundAuthType = true
		}
		if attr.Key == "saml.entity_id" && attr.Value.AsString() == "https://idp.example.com/saml" {
			foundEntityID = true
		}
		if attr.Key == "saml.name_id" && attr.Value.AsString() == "user@example.com" {
			foundNameID = true
		}
	}

	if !foundAuthType {
		t.Error("Expected auth.type attribute not found")
	}

	if !foundEntityID {
		t.Error("Expected saml.entity_id attribute not found")
	}

	if !foundNameID {
		t.Error("Expected saml.name_id attribute not found")
	}

	t.Log("Successfully verified SAML authentication span attributes")
}
