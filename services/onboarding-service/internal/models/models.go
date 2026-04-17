package models

import (
	"database/sql"
	"encoding/json"
	"time"
)

// Job represents a bulk onboarding job
type Job struct {
	ID              string         `json:"id"`
	TenantID        string         `json:"tenant_id"`
	Name            string         `json:"name"`
	Description     string         `json:"description"`
	SourceType      string         `json:"source_type"`
	SourceReference string         `json:"source_reference"`
	Status          string         `json:"status"`
	TotalUsers      int            `json:"total_users"`
	ProcessedUsers  int            `json:"processed_users"`
	SuccessfulUsers int            `json:"successful_users"`
	FailedUsers     int            `json:"failed_users"`
	ErrorSummary    sql.NullString `json:"error_summary,omitempty"`
	CreatedBy       string         `json:"created_by"`
	StartedAt       sql.NullTime   `json:"started_at,omitempty"`
	CompletedAt     sql.NullTime   `json:"completed_at,omitempty"`
	CreatedAt       time.Time      `json:"created_at"`
	UpdatedAt       time.Time      `json:"updated_at"`
}

// Task represents an individual user onboarding task
type Task struct {
	ID             string         `json:"id"`
	JobID          string         `json:"job_id"`
	TenantID       string         `json:"tenant_id"`
	Email          string         `json:"email"`
	FirstName      string         `json:"first_name"`
	LastName       string         `json:"last_name"`
	Role           string         `json:"role"`
	StudentID      string         `json:"student_id,omitempty"`
	Department     string         `json:"department,omitempty"`
	CourseCodes    []string       `json:"course_codes,omitempty"`
	GraduationYear sql.NullInt32  `json:"graduation_year,omitempty"`
	Phone          string         `json:"phone,omitempty"`
	PreferredLang  string         `json:"preferred_language"`
	CustomFields   sql.NullString `json:"custom_fields,omitempty"`
	Status         string         `json:"status"`
	UserID         sql.NullString `json:"user_id,omitempty"`
	RetryCount     int            `json:"retry_count"`
	ErrorMessage   sql.NullString `json:"error_message,omitempty"`
	ErrorDetails   sql.NullString `json:"error_details,omitempty"`
	CreatedAt      time.Time      `json:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at"`
	ProcessedAt    sql.NullTime   `json:"processed_at,omitempty"`
}

// IntegrationConfig represents configuration for external integrations
type IntegrationConfig struct {
	ID              string       `json:"id"`
	TenantID        string       `json:"tenant_id"`
	IntegrationType string       `json:"integration_type"`
	Name            string       `json:"name"`
	Config          string       `json:"config"` // JSON encrypted
	IsActive        bool         `json:"is_active"`
	LastSyncAt      sql.NullTime `json:"last_sync_at,omitempty"`
	CreatedAt       time.Time    `json:"created_at"`
	UpdatedAt       time.Time    `json:"updated_at"`
}

// AuditLog represents an immutable audit trail entry
type AuditLog struct {
	ID          string         `json:"id"`
	TenantID    string         `json:"tenant_id"`
	JobID       sql.NullString `json:"job_id,omitempty"`
	TaskID      sql.NullString `json:"task_id,omitempty"`
	EventType   string         `json:"event_type"`
	EventData   string         `json:"event_data"` // JSON
	PerformedBy string         `json:"performed_by"`
	IPAddress   string         `json:"ip_address,omitempty"`
	UserAgent   string         `json:"user_agent,omitempty"`
	CreatedAt   time.Time      `json:"created_at"`
}

// JobProgress represents real-time progress tracking
type JobProgress struct {
	JobID                   string       `json:"job_id"`
	CurrentStage            string       `json:"current_stage"`
	ProgressPercentage      float64      `json:"progress_percentage"`
	EstimatedCompletionTime sql.NullTime `json:"estimated_completion_time,omitempty"`
	CurrentTaskID           string       `json:"current_task_id,omitempty"`
	Metrics                 string       `json:"metrics"` // JSON
	UpdatedAt               time.Time    `json:"updated_at"`
}

// Tenant represents a multi-tenant organization
type Tenant struct {
	ID        string       `json:"id"`
	Name      string       `json:"name"`
	Domain    string       `json:"domain"`
	IsActive  bool         `json:"is_active"`
	Settings  string       `json:"settings"` // JSON
	CreatedAt time.Time    `json:"created_at"`
	UpdatedAt time.Time    `json:"updated_at"`
	DeletedAt sql.NullTime `json:"deleted_at,omitempty"`
}

// Kafka Message Types

// OnboardingTaskMessage represents a task message for Kafka
type OnboardingTaskMessage struct {
	TaskID   string          `json:"task_id"`
	JobID    string          `json:"job_id"`
	TenantID string          `json:"tenant_id"`
	UserData UserDataPayload `json:"user_data"`
	Attempt  int             `json:"attempt"`
}

// ProgressUpdateMessage represents a progress update for Kafka
type ProgressUpdateMessage struct {
	JobID              string  `json:"job_id"`
	TenantID           string  `json:"tenant_id"`
	CurrentStage       string  `json:"current_stage"`
	ProgressPercentage float64 `json:"progress_percentage"`
	ProcessedCount     int     `json:"processed_count"`
	TotalCount         int     `json:"total_count"`
	SuccessCount       int     `json:"success_count"`
	FailedCount        int     `json:"failed_count"`
}

// AuditEventMessage represents an audit event for Kafka
type AuditEventMessage struct {
	TenantID    string                 `json:"tenant_id"`
	JobID       string                 `json:"job_id,omitempty"`
	TaskID      string                 `json:"task_id,omitempty"`
	EventType   string                 `json:"event_type"`
	EventData   map[string]interface{} `json:"event_data"`
	PerformedBy string                 `json:"performed_by"`
	IPAddress   string                 `json:"ip_address,omitempty"`
	UserAgent   string                 `json:"user_agent,omitempty"`
}

// UserDataPayload represents user data for onboarding
type UserDataPayload struct {
	Email          string            `json:"email"`
	FirstName      string            `json:"first_name"`
	LastName       string            `json:"last_name"`
	Role           string            `json:"role"`
	StudentID      string            `json:"student_id,omitempty"`
	Department     string            `json:"department,omitempty"`
	CourseCodes    []string          `json:"course_codes,omitempty"`
	GraduationYear int               `json:"graduation_year,omitempty"`
	Phone          string            `json:"phone,omitempty"`
	PreferredLang  string            `json:"preferred_language,omitempty"`
	CustomFields   map[string]string `json:"custom_fields,omitempty"`
}

// Helper methods

func (u *UserDataPayload) ToJSON() (string, error) {
	data, err := json.Marshal(u)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func (u *UserDataPayload) FromJSON(data string) error {
	return json.Unmarshal([]byte(data), u)
}

// Constants

const (
	// Job statuses
	JobStatusPending    = "pending"
	JobStatusProcessing = "processing"
	JobStatusCompleted  = "completed"
	JobStatusFailed     = "failed"
	JobStatusCancelled  = "cancelled"

	// Task statuses
	TaskStatusPending    = "pending"
	TaskStatusProcessing = "processing"
	TaskStatusCompleted  = "completed"
	TaskStatusFailed     = "failed"
	TaskStatusSkipped    = "skipped"

	// Source types
	SourceTypeCSV       = "csv"
	SourceTypeLDAP      = "ldap"
	SourceTypeSAML      = "saml"
	SourceTypeGoogle    = "google"
	SourceTypeMicrosoft = "microsoft"
	SourceTypeAPI       = "api"

	// Integration types
	IntegrationTypeLDAP      = "ldap"
	IntegrationTypeSAML      = "saml"
	IntegrationTypeGoogle    = "google"
	IntegrationTypeMicrosoft = "microsoft"

	// User roles
	RoleStudent    = "student"
	RoleInstructor = "instructor"
	RoleStaff      = "staff"
	RoleAdmin      = "admin"

	// Event types
	EventJobCreated     = "job_created"
	EventJobStarted     = "job_started"
	EventJobCompleted   = "job_completed"
	EventJobFailed      = "job_failed"
	EventJobCancelled   = "job_cancelled"
	EventTaskProcessed  = "task_processed"
	EventTaskCompleted  = "task_completed"
	EventTaskFailed     = "task_failed"
	EventTaskRetried    = "task_retried"
	EventUserCreated    = "user_created"
	EventRoleAssigned   = "role_assigned"
	EventCourseEnrolled = "course_enrolled"
	EventEmailSent      = "email_sent"
)
