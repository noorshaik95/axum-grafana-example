package config

import (
	"os"
	"testing"
)

func TestLoad_Defaults(t *testing.T) {
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	if cfg.Server.Port != 8086 {
		t.Errorf("Server.Port = %d, want 8086", cfg.Server.Port)
	}
	if cfg.GRPC.Port != 50056 {
		t.Errorf("GRPC.Port = %d, want 50056", cfg.GRPC.Port)
	}
	if cfg.Database.DBName != "email_service" {
		t.Errorf("Database.DBName = %q, want %q", cfg.Database.DBName, "email_service")
	}
	if cfg.Kafka.GroupID != "email-service" {
		t.Errorf("Kafka.GroupID = %q, want %q", cfg.Kafka.GroupID, "email-service")
	}
}

func TestLoad_EnvOverride(t *testing.T) {
	os.Setenv("SERVER_PORT", "9999")
	os.Setenv("DB_NAME", "test_db")
	defer os.Unsetenv("SERVER_PORT")
	defer os.Unsetenv("DB_NAME")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	if cfg.Server.Port != 9999 {
		t.Errorf("Server.Port = %d, want 9999", cfg.Server.Port)
	}
	if cfg.Database.DBName != "test_db" {
		t.Errorf("Database.DBName = %q, want %q", cfg.Database.DBName, "test_db")
	}
}

func TestDSN(t *testing.T) {
	cfg := DatabaseConfig{
		Host:     "localhost",
		Port:     5432,
		User:     "postgres",
		Password: "pass",
		DBName:   "testdb",
		SSLMode:  "disable",
	}

	expected := "host=localhost port=5432 user=postgres password=pass dbname=testdb sslmode=disable"
	if got := cfg.DSN(); got != expected {
		t.Errorf("DSN() = %q, want %q", got, expected)
	}
}

func TestAddress(t *testing.T) {
	srv := ServerConfig{Host: "0.0.0.0", Port: 8086}
	if got := srv.Address(); got != "0.0.0.0:8086" {
		t.Errorf("Address() = %q, want %q", got, "0.0.0.0:8086")
	}
}
