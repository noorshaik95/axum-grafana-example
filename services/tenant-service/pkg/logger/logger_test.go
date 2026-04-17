package logger

import (
	"os"
	"testing"
)

func TestInitLogger(t *testing.T) {
	// Test production environment
	os.Setenv("ENVIRONMENT", "production")
	InitLogger()
	logger := GetLogger()
	if logger == nil {
		t.Fatal("Expected logger to be initialized")
	}
	os.Unsetenv("ENVIRONMENT")

	// Test development environment
	os.Setenv("ENVIRONMENT", "development")
	InitLogger()
	logger = GetLogger()
	if logger == nil {
		t.Fatal("Expected logger to be initialized in development mode")
	}
	os.Unsetenv("ENVIRONMENT")

	// Test default (no environment set)
	InitLogger()
	logger = GetLogger()
	if logger == nil {
		t.Fatal("Expected logger to be initialized with defaults")
	}
}

func TestGetLogger(t *testing.T) {
	// Initialize logger first
	InitLogger()

	logger := GetLogger()
	if logger == nil {
		t.Fatal("Expected logger instance, got nil")
	}

	// Get logger again - should return the same instance
	logger2 := GetLogger()
	if logger2 == nil {
		t.Fatal("Expected logger instance on second call, got nil")
	}
}

func TestLoggerLevels(t *testing.T) {
	// Test debug level
	os.Setenv("LOG_LEVEL", "debug")
	InitLogger()
	logger := GetLogger()
	if logger == nil {
		t.Fatal("Expected logger with debug level")
	}
	os.Unsetenv("LOG_LEVEL")

	// Test info level
	os.Setenv("LOG_LEVEL", "info")
	InitLogger()
	logger = GetLogger()
	if logger == nil {
		t.Fatal("Expected logger with info level")
	}
	os.Unsetenv("LOG_LEVEL")

	// Test warn level
	os.Setenv("LOG_LEVEL", "warn")
	InitLogger()
	logger = GetLogger()
	if logger == nil {
		t.Fatal("Expected logger with warn level")
	}
	os.Unsetenv("LOG_LEVEL")

	// Test error level
	os.Setenv("LOG_LEVEL", "error")
	InitLogger()
	logger = GetLogger()
	if logger == nil {
		t.Fatal("Expected logger with error level")
	}
	os.Unsetenv("LOG_LEVEL")
}

func TestLoggerBasicUsage(t *testing.T) {
	InitLogger()
	logger := GetLogger()

	// Test that basic logging operations don't panic
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("Logger panicked: %v", r)
		}
	}()

	logger.Info().Msg("Test info message")
	logger.Debug().Msg("Test debug message")
	logger.Warn().Msg("Test warn message")
	logger.Error().Msg("Test error message")
	logger.Info().Str("key", "value").Msg("Test with fields")
}
