package repository

import (
	"database/sql"
	"testing"
)

func sqlNullString(valid bool, value string) sql.NullString {
	return sql.NullString{String: value, Valid: valid}
}

func TestNilIfEmpty(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		isNil    bool
	}{
		{"empty string returns nil", "", true},
		{"non-empty string returns value", "hello", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := nilIfEmpty(tt.input)
			if tt.isNil && result != nil {
				t.Errorf("expected nil but got %v", result)
			}
			if !tt.isNil && result == nil {
				t.Errorf("expected non-nil but got nil")
			}
			if !tt.isNil && result != tt.input {
				t.Errorf("expected %q but got %v", tt.input, result)
			}
		})
	}
}

func TestNullStringVal(t *testing.T) {
	tests := []struct {
		name     string
		valid    bool
		value    string
		expected string
	}{
		{"valid string", true, "test", "test"},
		{"null string", false, "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ns := sqlNullString(tt.valid, tt.value)
			result := nullStringVal(ns)
			if result != tt.expected {
				t.Errorf("expected %q but got %q", tt.expected, result)
			}
		})
	}
}
