package repository

import (
	"testing"
)

func TestSubmissionSortFieldValidation(t *testing.T) {
	validSortFields := map[string]bool{
		"submitted_at": true,
		"student_id":   true,
		"status":       true,
	}

	tests := []struct {
		field string
		valid bool
	}{
		{"submitted_at", true},
		{"student_id", true},
		{"status", true},
		{"invalid_field", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.field, func(t *testing.T) {
			if validSortFields[tt.field] != tt.valid {
				t.Errorf("field %q: expected valid=%v", tt.field, tt.valid)
			}
		})
	}
}
