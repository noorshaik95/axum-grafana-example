// Package models defines DB-backed record types for admin-auth-service.
package models

import "time"

// PlatformAdmin is a row of platform_admins.
type PlatformAdmin struct {
	ID           string
	Email        string
	PasswordHash string
	FullName     string
	Roles        []string
	Disabled     bool
	CreatedAt    time.Time
	UpdatedAt    time.Time
	LastLoginAt  *time.Time
}

// HasRole returns true if the admin holds the named role.
func (p *PlatformAdmin) HasRole(name string) bool {
	for _, r := range p.Roles {
		if r == name {
			return true
		}
	}
	return false
}

// CanImpersonate reports whether this admin is allowed to mint impersonation tokens.
func (p *PlatformAdmin) CanImpersonate() bool {
	return p.HasRole("superadmin") || p.HasRole("support")
}

// AuditEntry is a row of platform_audit.
type AuditEntry struct {
	ID         string
	ActorID    string
	Action     string
	TargetID   string
	TargetType string
	Metadata   map[string]any
	RequestID  string
	CreatedAt  time.Time
}

// PlatformRole is a row of platform_roles (W2.5).
type PlatformRole struct {
	Name        string
	Description string
	Permissions []string
	CreatedAt   time.Time
}

// AuditQueryRow is the shape returned by the paginated audit read path.
// AdminEmail is populated via LEFT JOIN on platform_admins.
type AuditQueryRow struct {
	AuditEntry
	AdminEmail string
}
