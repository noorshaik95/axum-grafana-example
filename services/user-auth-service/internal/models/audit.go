package models

import "time"

// AuditAction enumerates the action types appended to the audit_events table
// per plan.md W14.3. New action types must be added here and covered by tests.
type AuditAction string

const (
	AuditActionLogin               AuditAction = "login"
	AuditActionLogout              AuditAction = "logout"
	AuditActionFailedLogin         AuditAction = "failed_login"
	AuditActionPasswordChange      AuditAction = "password_change"
	AuditActionMFAChange           AuditAction = "mfa_change"
	AuditActionImpersonationStart  AuditAction = "impersonation_start"
	AuditActionImpersonationEnd    AuditAction = "impersonation_end"
	AuditActionSSOLogin            AuditAction = "sso_login"
)

// AuditActorType identifies what kind of principal performed the action.
type AuditActorType string

const (
	AuditActorUser   AuditActorType = "user"
	AuditActorAdmin  AuditActorType = "admin"
	AuditActorSystem AuditActorType = "system"
)

// AuditEvent is a single row in audit_events. Metadata is JSON-serializable.
type AuditEvent struct {
	ID         string                 `json:"id"`
	ActorID    string                 `json:"actor_id"`
	ActorType  AuditActorType         `json:"actor_type"`
	Action     AuditAction            `json:"action"`
	TargetID   string                 `json:"target_id"`
	TargetType string                 `json:"target_type"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
	IPAddress  string                 `json:"ip_address,omitempty"`
	CreatedAt  time.Time              `json:"created_at"`
}
