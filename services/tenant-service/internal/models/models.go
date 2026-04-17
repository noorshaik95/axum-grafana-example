package models

import (
	"database/sql"
	"time"
)

// TenantTier represents subscription tiers
type TenantTier int32

const (
	TierUnspecified TenantTier = 0
	TierFree        TenantTier = 1
	TierBasic       TenantTier = 2
	TierProfessional TenantTier = 3
	TierEnterprise  TenantTier = 4
)

func (t TenantTier) String() string {
	switch t {
	case TierFree:
		return "free"
	case TierBasic:
		return "basic"
	case TierProfessional:
		return "professional"
	case TierEnterprise:
		return "enterprise"
	default:
		return "unspecified"
	}
}

// ProvisioningStatus represents the current status of tenant provisioning
type ProvisioningStatus int32

const (
	StatusUnspecified      ProvisioningStatus = 0
	StatusPending          ProvisioningStatus = 1
	StatusProvisioningDB   ProvisioningStatus = 2
	StatusCreatingAdmin    ProvisioningStatus = 3
	StatusSettingQuota     ProvisioningStatus = 4
	StatusSendingEmail     ProvisioningStatus = 5
	StatusCompleted        ProvisioningStatus = 6
	StatusFailed           ProvisioningStatus = 7
)

func (s ProvisioningStatus) String() string {
	switch s {
	case StatusPending:
		return "pending"
	case StatusProvisioningDB:
		return "provisioning_database"
	case StatusCreatingAdmin:
		return "creating_admin"
	case StatusSettingQuota:
		return "setting_quota"
	case StatusSendingEmail:
		return "sending_email"
	case StatusCompleted:
		return "completed"
	case StatusFailed:
		return "failed"
	default:
		return "unspecified"
	}
}

// SubscriptionTier represents a subscription tier configuration
type SubscriptionTier struct {
	ID                 string
	Name               string
	DisplayName        string
	TierLevel          int32
	StorageQuotaBytes  int64
	MaxUsers           int32
	MaxCourses         int32
	APIRateLimit       int32
	DedicatedDatabase  bool
	CustomDomain       bool
	PrioritySupport    bool
	PriceCentsMonthly  int32
	Features           map[string]interface{}
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

// Tenant represents a tenant/organization in the system
type Tenant struct {
	ID                       string
	Name                     string
	Domain                   string
	TierID                   string
	IsActive                 bool
	DatabaseName             sql.NullString
	DatabaseHost             sql.NullString
	DatabasePort             sql.NullInt32
	DatabaseUsername         sql.NullString
	DatabasePasswordEncrypted sql.NullString
	DatabaseConnectionString sql.NullString
	StorageQuotaBytes        int64
	StorageUsedBytes         int64
	FileCount                int32
	UserCount                int32
	CourseCount              int32
	CreatedAt                time.Time
	UpdatedAt                time.Time
	LastActiveAt             sql.NullTime
}

// TenantAdmin represents an admin user for a tenant
type TenantAdmin struct {
	ID        string
	TenantID  string
	UserID    string
	Email     string
	FirstName sql.NullString
	LastName  sql.NullString
	IsPrimary bool
	CreatedAt time.Time
}

// TenantProvisioning represents the provisioning tracking record
type TenantProvisioning struct {
	ID                 string
	TenantID           string
	Status             string
	CurrentStep        sql.NullString
	ProgressPercentage int32
	ErrorMessage       sql.NullString
	StartedAt          time.Time
	CompletedAt        sql.NullTime
	DurationSeconds    sql.NullInt32
}

// TenantSetupToken represents a setup token for tenant onboarding
type TenantSetupToken struct {
	ID        string
	TenantID  string
	AdminID   string
	Token     string
	ExpiresAt time.Time
	UsedAt    sql.NullTime
	CreatedAt time.Time
}

// StorageFile represents a file in tenant storage
type StorageFile struct {
	ID           string
	TenantID     string
	UserID       string
	Filename     string
	FilePath     string
	FileSizeBytes int64
	MimeType     sql.NullString
	Checksum     sql.NullString
	CreatedAt    time.Time
	UpdatedAt    time.Time
	DeletedAt    sql.NullTime
}

// CreateTenantRequest represents the request to create a new tenant
type CreateTenantRequest struct {
	Name           string
	Domain         string
	Tier           TenantTier
	AdminEmail     string
	AdminFirstName string
	AdminLastName  string
	AdminPassword  string
}

// CreateTenantResponse represents the response from creating a tenant
type CreateTenantResponse struct {
	Tenant         *Tenant
	Admin          *TenantAdmin
	SetupURL       string
	Status         ProvisioningStatus
	ProvisioningID string
}
