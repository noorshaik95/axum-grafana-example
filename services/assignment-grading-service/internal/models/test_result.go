package models

import "time"

// SubmissionTestResult is one auto-test outcome for a submission (W9.3).
type SubmissionTestResult struct {
	ID           string    `json:"id"`
	SubmissionID string    `json:"submission_id"`
	AssignmentID string    `json:"assignment_id"`
	TestName     string    `json:"test_name"`
	Passed       bool      `json:"passed"`
	Output       string    `json:"output,omitempty"`
	DurationMS   int       `json:"duration_ms"`
	CreatedAt    time.Time `json:"created_at"`
}
