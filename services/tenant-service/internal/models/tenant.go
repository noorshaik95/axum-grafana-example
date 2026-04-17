package models

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"
)

// V2 tenant status constants for the Docker-provisioned model.
const (
	StatusProvisioning = "provisioning"
	StatusActive       = "active"
	StatusSuspended    = "suspended"
	StatusDeleted      = "deleted"
)

// TenantV2 represents a tenant in the Docker-provisioned multi-tenant platform.
type TenantV2 struct {
	ID           string    `json:"id" db:"id"`
	Slug         string    `json:"slug" db:"slug"`
	Name         string    `json:"name" db:"name"`
	AdminEmail   string    `json:"adminEmail" db:"admin_email"`
	Status       string    `json:"status" db:"status"`
	Plan         Plan      `json:"plan" db:"plan"`
	Subdomain    string    `json:"subdomain" db:"subdomain"`
	ContainerIDs []string  `json:"containerIds" db:"container_ids"`
	CreatedAt    time.Time `json:"createdAt" db:"created_at"`
	UpdatedAt    time.Time `json:"updatedAt" db:"updated_at"`
}

// Plan defines resource quotas and feature flags for a tenant.
type Plan struct {
	StorageGB  int      `json:"storageGb"`
	MaxUsers   int      `json:"maxUsers"`
	MaxCourses int      `json:"maxCourses"`
	Features   []string `json:"features"`
}

// Value implements driver.Valuer for storing Plan as JSONB.
func (p Plan) Value() (driver.Value, error) {
	return json.Marshal(p)
}

// Scan implements sql.Scanner for reading Plan from JSONB.
func (p *Plan) Scan(src interface{}) error {
	if src == nil {
		return nil
	}
	switch v := src.(type) {
	case []byte:
		return json.Unmarshal(v, p)
	case string:
		return json.Unmarshal([]byte(v), p)
	default:
		return fmt.Errorf("unsupported type for Plan: %T", src)
	}
}

// StringArray is a helper for scanning PostgreSQL TEXT[] arrays.
type StringArray []string

// Value implements driver.Valuer for TEXT[].
func (a StringArray) Value() (driver.Value, error) {
	if a == nil {
		return "{}", nil
	}
	// Build PostgreSQL array literal: {"a","b","c"}
	result := "{"
	for i, s := range a {
		if i > 0 {
			result += ","
		}
		result += fmt.Sprintf(`"%s"`, s)
	}
	result += "}"
	return result, nil
}

// Scan implements sql.Scanner for TEXT[].
func (a *StringArray) Scan(src interface{}) error {
	if src == nil {
		*a = nil
		return nil
	}
	switch v := src.(type) {
	case []byte:
		return a.parseArray(string(v))
	case string:
		return a.parseArray(v)
	default:
		return fmt.Errorf("unsupported type for StringArray: %T", src)
	}
}

func (a *StringArray) parseArray(s string) error {
	// Handle PostgreSQL array format: {val1,val2} or {"val1","val2"}
	if s == "{}" || s == "" {
		*a = nil
		return nil
	}
	s = s[1 : len(s)-1] // strip { }
	var items []string
	inQuote := false
	current := ""
	for i := 0; i < len(s); i++ {
		ch := s[i]
		switch {
		case ch == '"':
			inQuote = !inQuote
		case ch == ',' && !inQuote:
			items = append(items, current)
			current = ""
		default:
			current += string(ch)
		}
	}
	if current != "" {
		items = append(items, current)
	}
	*a = items
	return nil
}

// ProvisionTenantRequest is the REST API request body for provisioning a new tenant.
type ProvisionTenantRequest struct {
	Name       string `json:"name"`
	Slug       string `json:"slug"`
	AdminEmail string `json:"adminEmail"`
	Plan       *Plan  `json:"plan,omitempty"`
}

// UpdateAccessRequest is the API request for enabling/disabling a tenant.
type UpdateAccessRequest struct {
	Enabled bool   `json:"enabled"`
	Reason  string `json:"reason,omitempty"`
}

// UpdatePlanRequest is the API request for changing a tenant's plan.
type UpdatePlanRequest struct {
	StorageGB  *int      `json:"storageGb,omitempty"`
	MaxUsers   *int      `json:"maxUsers,omitempty"`
	MaxCourses *int      `json:"maxCourses,omitempty"`
	Features   *[]string `json:"features,omitempty"`
}

// TenantUsage represents current resource usage for a tenant.
type TenantUsage struct {
	TenantID       string `json:"tenantId"`
	StorageUsedGB  float64 `json:"storageUsedGb"`
	StorageLimitGB int     `json:"storageLimitGb"`
	UserCount      int     `json:"userCount"`
	MaxUsers       int     `json:"maxUsers"`
	CourseCount    int     `json:"courseCount"`
	MaxCourses     int     `json:"maxCourses"`
	ContainerCount int     `json:"containerCount"`
}
