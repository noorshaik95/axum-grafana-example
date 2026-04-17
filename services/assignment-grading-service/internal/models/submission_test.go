package models

import (
	"testing"
	"time"
)

func TestSubmission_Validate(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name        string
		submission  Submission
		expectError bool
		errorMsg    string
	}{
		{
			name: "valid submission",
			submission: Submission{
				AssignmentID: "ASSIGN-001",
				StudentID:    "STUDENT-001",
				FilePath:     "/path/to/file.pdf",
				SubmittedAt:  now,
				Status:       StatusSubmitted,
				IsLate:       false,
				DaysLate:     0,
			},
			expectError: false,
		},
		{
			name: "missing assignment_id",
			submission: Submission{
				StudentID:   "STUDENT-001",
				FilePath:    "/path/to/file.pdf",
				SubmittedAt: now,
				Status:      StatusSubmitted,
			},
			expectError: true,
			errorMsg:    "assignment_id is required",
		},
		{
			name: "missing student_id",
			submission: Submission{
				AssignmentID: "ASSIGN-001",
				FilePath:     "/path/to/file.pdf",
				SubmittedAt:  now,
				Status:       StatusSubmitted,
			},
			expectError: true,
			errorMsg:    "student_id is required",
		},
		{
			name: "missing file_path and file_urls",
			submission: Submission{
				AssignmentID: "ASSIGN-001",
				StudentID:    "STUDENT-001",
				SubmittedAt:  now,
				Status:       StatusSubmitted,
			},
			expectError: true,
			errorMsg:    "file_path or file_urls is required",
		},
		{
			name: "valid with file_urls instead of file_path",
			submission: Submission{
				AssignmentID: "ASSIGN-001",
				StudentID:    "STUDENT-001",
				FileURLs:     []string{"s3://bucket/file.pdf"},
				SubmittedAt:  now,
				Status:       StatusSubmitted,
			},
			expectError: false,
		},
		{
			name: "zero submitted_at",
			submission: Submission{
				AssignmentID: "ASSIGN-001",
				StudentID:    "STUDENT-001",
				FilePath:     "/path/to/file.pdf",
				Status:       StatusSubmitted,
			},
			expectError: true,
			errorMsg:    "submitted_at is required",
		},
		{
			name: "invalid status",
			submission: Submission{
				AssignmentID: "ASSIGN-001",
				StudentID:    "STUDENT-001",
				FilePath:     "/path/to/file.pdf",
				SubmittedAt:  now,
				Status:       "invalid_status",
			},
			expectError: true,
			errorMsg:    "invalid status",
		},
		{
			name: "negative days_late",
			submission: Submission{
				AssignmentID: "ASSIGN-001",
				StudentID:    "STUDENT-001",
				FilePath:     "/path/to/file.pdf",
				SubmittedAt:  now,
				Status:       StatusSubmitted,
				DaysLate:     -1,
			},
			expectError: true,
			errorMsg:    "days_late must be non-negative",
		},
		{
			name: "late submission with positive days",
			submission: Submission{
				AssignmentID: "ASSIGN-001",
				StudentID:    "STUDENT-001",
				FilePath:     "/path/to/file.pdf",
				SubmittedAt:  now,
				Status:       StatusSubmitted,
				IsLate:       true,
				DaysLate:     2,
			},
			expectError: false,
		},
		{
			name: "graded status",
			submission: Submission{
				AssignmentID: "ASSIGN-001",
				StudentID:    "STUDENT-001",
				FilePath:     "/path/to/file.pdf",
				SubmittedAt:  now,
				Status:       StatusGraded,
			},
			expectError: false,
		},
		{
			name: "returned status",
			submission: Submission{
				AssignmentID: "ASSIGN-001",
				StudentID:    "STUDENT-001",
				FilePath:     "/path/to/file.pdf",
				SubmittedAt:  now,
				Status:       StatusReturned,
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.submission.Validate()
			if tt.expectError {
				if err == nil {
					t.Errorf("expected error but got nil")
				} else if err.Error() != tt.errorMsg {
					t.Errorf("expected error %q but got %q", tt.errorMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("expected no error but got: %v", err)
				}
			}
		})
	}
}

func TestSubmission_UpdateStatus(t *testing.T) {
	now := time.Now()
	submission := Submission{
		AssignmentID: "ASSIGN-001",
		StudentID:    "STUDENT-001",
		FilePath:     "/path/to/file.pdf",
		SubmittedAt:  now,
		Status:       StatusSubmitted,
		UpdatedAt:    now,
	}

	tests := []struct {
		name        string
		newStatus   string
		expectError bool
	}{
		{
			name:        "update to graded",
			newStatus:   StatusGraded,
			expectError: false,
		},
		{
			name:        "update to returned",
			newStatus:   StatusReturned,
			expectError: false,
		},
		{
			name:        "invalid status",
			newStatus:   "invalid_status",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := submission // Make a copy
			err := s.UpdateStatus(tt.newStatus)
			if tt.expectError {
				if err == nil {
					t.Errorf("expected error but got nil")
				}
			} else {
				if err != nil {
					t.Errorf("expected no error but got: %v", err)
				}
				if s.Status != tt.newStatus {
					t.Errorf("expected status %q but got %q", tt.newStatus, s.Status)
				}
				if !s.UpdatedAt.After(now) {
					t.Errorf("expected UpdatedAt to be updated")
				}
			}
		})
	}
}

func TestIsValidStatus(t *testing.T) {
	tests := []struct {
		status   string
		expected bool
	}{
		{StatusSubmitted, true},
		{StatusGraded, true},
		{StatusReturned, true},
		{StatusLate, true},
		{"invalid", false},
		{"", false},
		{"SUBMITTED", false}, // Case sensitive
	}

	for _, tt := range tests {
		t.Run(tt.status, func(t *testing.T) {
			result := isValidStatus(tt.status)
			if result != tt.expected {
				t.Errorf("isValidStatus(%q) = %v, want %v", tt.status, result, tt.expected)
			}
		})
	}
}
