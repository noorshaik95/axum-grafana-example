package models

import (
	"time"

	"github.com/google/uuid"
)

type User struct {
	ID             string    `json:"id"`
	Email          string    `json:"email"`
	PasswordHash   string    `json:"-"` // Never expose password hash in JSON
	FirstName      string    `json:"first_name"`
	LastName       string    `json:"last_name"`
	Phone          string    `json:"phone"`
	Timezone       string    `json:"timezone"`
	AvatarURL      string    `json:"avatar_url,omitempty"`
	Bio            string    `json:"bio,omitempty"`
	OrganizationID string    `json:"organization_id,omitempty"`
	// Username — unique case-insensitive @handle (W14.x). Backfilled by
	// migration 010 from the email local-part; NOT NULL in the DB.
	Username  string    `json:"username,omitempty"`
	IsActive  bool      `json:"is_active"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Roles     []string  `json:"roles,omitempty"`
}

type Profile struct {
	UserID    string    `json:"user_id"`
	FirstName string    `json:"first_name"`
	LastName  string    `json:"last_name"`
	Email     string    `json:"email"`
	Phone     string    `json:"phone"`
	AvatarURL string    `json:"avatar_url"`
	Bio       string    `json:"bio"`
	Timezone  string    `json:"timezone"`
	Roles     []string  `json:"roles"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type Role struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Permissions []string  `json:"permissions"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type UserRole struct {
	UserID    string    `json:"user_id"`
	RoleID    string    `json:"role_id"`
	AssignedAt time.Time `json:"assigned_at"`
}

type TokenPair struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int64  `json:"expires_in"`
}

// NewUser creates a new user with a generated UUID and a best-effort default
// username derived from the email local-part (migration 010 backfill
// convention). Callers that care about collision-safe allocation should
// overwrite `Username` with a suffix-retry loop before insert.
func NewUser(email, passwordHash, firstName, lastName, phone string) *User {
	now := time.Now()
	return &User{
		ID:           uuid.New().String(),
		Email:        email,
		PasswordHash: passwordHash,
		FirstName:    firstName,
		LastName:     lastName,
		Phone:        phone,
		Timezone:     "UTC", // Default timezone
		Username:     defaultUsernameFromEmail(email),
		IsActive:     true,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
}

// defaultUsernameFromEmail is the model-level fallback so every freshly-minted
// user has a non-empty Username even when callers (SSO/OAuth/SAML JIT paths)
// skip the explicit derivation. Collision-safe retry is a caller concern.
func defaultUsernameFromEmail(email string) string {
	at := -1
	for i := 0; i < len(email); i++ {
		if email[i] == '@' {
			at = i
			break
		}
	}
	if at <= 0 {
		at = len(email)
	}
	// Lowercase + strip anything outside [a-z0-9_.-]
	buf := make([]byte, 0, at)
	for i := 0; i < at; i++ {
		c := email[i]
		if c >= 'A' && c <= 'Z' {
			c += 'a' - 'A'
		}
		if (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') || c == '_' || c == '-' || c == '.' {
			buf = append(buf, c)
		}
	}
	if len(buf) == 0 {
		return "user"
	}
	if len(buf) > 60 {
		buf = buf[:60]
	}
	return string(buf)
}

// FullName returns the user's full name
func (u *User) FullName() string {
	return u.FirstName + " " + u.LastName
}

// HasRole checks if the user has a specific role
func (u *User) HasRole(role string) bool {
	for _, r := range u.Roles {
		if r == role {
			return true
		}
	}
	return false
}

// IsAdmin checks if the user has admin role
func (u *User) IsAdmin() bool {
	return u.HasRole("admin")
}

// ToProfile converts a User to Profile
func (u *User) ToProfile() *Profile {
	return &Profile{
		UserID:    u.ID,
		FirstName: u.FirstName,
		LastName:  u.LastName,
		Email:     u.Email,
		Phone:     u.Phone,
		AvatarURL: u.AvatarURL,
		Bio:       u.Bio,
		Timezone:  u.Timezone,
		Roles:     u.Roles,
		CreatedAt: u.CreatedAt,
		UpdatedAt: u.UpdatedAt,
	}
}
