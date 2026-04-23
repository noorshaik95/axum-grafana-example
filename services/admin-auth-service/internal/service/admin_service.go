// Package service implements platform-admin business logic for admin-auth-service.
package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"

	commontracing "slate/libs/common-go/tracing"
	"slate/services/admin-auth-service/internal/audit"
	jwtpkg "slate/services/admin-auth-service/internal/jwt"
	"slate/services/admin-auth-service/internal/models"
	"slate/services/admin-auth-service/internal/repository"
)

// Errors returned by AdminService are mapped to gRPC codes in the handler.
var (
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrAccountDisabled    = errors.New("account is disabled")
	ErrForbidden          = errors.New("forbidden")
	ErrInvalidInput       = errors.New("invalid input")
)

// Revoker is the subset of token-blacklist behaviour AdminService needs.
// A full Redis-backed implementation lives in cmd/server; tests use a memory fake.
type Revoker interface {
	Revoke(ctx context.Context, jti string, ttl time.Duration) error
	IsRevoked(ctx context.Context, jti string) (bool, error)
}

// AdminService bundles the domain use cases behind AdminAuthService.
type AdminService struct {
	admins     *repository.AdminRepository
	audits     *audit.Recorder
	auditReads *repository.AuditRepository
	roles      *repository.RoleRepository
	tokens     *jwtpkg.TokenService
	revoker    Revoker
	redirect   string
}

// NewAdminService builds an AdminService.
// redirectTemplate should contain {slug} and {token} placeholders.
func NewAdminService(
	admins *repository.AdminRepository,
	audits *audit.Recorder,
	tokens *jwtpkg.TokenService,
	revoker Revoker,
	redirectTemplate string,
) *AdminService {
	return &AdminService{
		admins:   admins,
		audits:   audits,
		tokens:   tokens,
		revoker:  revoker,
		redirect: redirectTemplate,
	}
}

// WithReads attaches the W2.5 read-path repositories (roles + audit).
// Optional — services built before W2.5 keep working without calling this.
func (s *AdminService) WithReads(roles *repository.RoleRepository, auditReads *repository.AuditRepository) *AdminService {
	s.roles = roles
	s.auditReads = auditReads
	return s
}

// LoginResult bundles the tokens issued by Login.
type LoginResult struct {
	AccessToken  string
	RefreshToken string
	ExpiresAt    time.Time
	User         *models.PlatformAdmin
}

// Login authenticates email+password and issues an admin access+refresh token pair.
func (s *AdminService) Login(ctx context.Context, email, password string) (*LoginResult, error) {
	email = normalizeEmail(email)
	if email == "" || password == "" {
		return nil, ErrInvalidInput
	}
	admin, err := s.admins.GetByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			s.auditLoginFailed(ctx, email, "not_found")
			return nil, ErrInvalidCredentials
		}
		return nil, err
	}
	if admin.Disabled {
		s.auditLoginFailed(ctx, admin.ID, "disabled")
		return nil, ErrAccountDisabled
	}
	if err := bcrypt.CompareHashAndPassword([]byte(admin.PasswordHash), []byte(password)); err != nil {
		s.auditLoginFailed(ctx, admin.ID, "bad_password")
		return nil, ErrInvalidCredentials
	}
	access, exp, err := s.tokens.IssueAccessToken(admin.ID, admin.Email, admin.Roles)
	if err != nil {
		return nil, err
	}
	refresh, _, err := s.tokens.IssueRefreshToken(admin.ID, admin.Email, admin.Roles)
	if err != nil {
		return nil, err
	}
	_ = s.admins.UpdateLastLogin(ctx, admin.ID)

	_ = s.audits.Record(ctx, &models.AuditEntry{
		ActorID:    admin.ID,
		Action:     audit.ActionAdminLogin,
		TargetID:   admin.ID,
		TargetType: "admin",
	})
	return &LoginResult{
		AccessToken:  access,
		RefreshToken: refresh,
		ExpiresAt:    exp,
		User:         admin,
	}, nil
}

// Logout revokes the given access token by JTI for the remainder of its TTL.
func (s *AdminService) Logout(ctx context.Context, rawToken string) error {
	claims, err := s.tokens.ParseAdminToken(rawToken)
	if err != nil {
		return ErrInvalidInput
	}
	if s.revoker != nil && claims.ID != "" {
		ttl := time.Until(claims.ExpiresAt.Time)
		if ttl > 0 {
			_ = s.revoker.Revoke(ctx, claims.ID, ttl)
		}
	}
	_ = s.audits.Record(ctx, &models.AuditEntry{
		ActorID:    claims.UserID,
		Action:     audit.ActionAdminLogout,
		TargetID:   claims.UserID,
		TargetType: "admin",
	})
	return nil
}

// ValidateAdminTokenResult describes a validated admin token for callers.
type ValidateAdminTokenResult struct {
	Valid    bool
	UserID   string
	Email    string
	Roles    []string
	Audience string
	Error    string
}

// ValidateAdminToken checks signature, aud=platform, and revocation.
func (s *AdminService) ValidateAdminToken(ctx context.Context, rawToken string) *ValidateAdminTokenResult {
	claims, err := s.tokens.ParseAdminToken(rawToken)
	if err != nil {
		return &ValidateAdminTokenResult{Valid: false, Error: err.Error()}
	}
	if s.revoker != nil && claims.ID != "" {
		revoked, err := s.revoker.IsRevoked(ctx, claims.ID)
		if err == nil && revoked {
			return &ValidateAdminTokenResult{Valid: false, Error: "token revoked"}
		}
	}
	aud := ""
	if len(claims.Audience) > 0 {
		aud = claims.Audience[0]
	}
	return &ValidateAdminTokenResult{
		Valid:    true,
		UserID:   claims.UserID,
		Email:    claims.Email,
		Roles:    claims.Roles,
		Audience: aud,
	}
}

// Register creates a new platform admin.
// createdBy is the actor for the audit ledger; "" implies bootstrap / CLI.
func (s *AdminService) Register(ctx context.Context, email, password, fullName string, roles []string, createdBy string) (*models.PlatformAdmin, error) {
	email = normalizeEmail(email)
	if email == "" || len(password) < 8 {
		return nil, ErrInvalidInput
	}
	if len(roles) == 0 {
		roles = []string{"readonly"}
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("hash password: %w", err)
	}
	admin := &models.PlatformAdmin{
		Email:        email,
		PasswordHash: string(hash),
		FullName:     fullName,
		Roles:        roles,
	}
	if err := s.admins.Create(ctx, admin); err != nil {
		return nil, err
	}
	actor := createdBy
	if actor == "" {
		actor = admin.ID // bootstrap case: admin audits themselves
	}
	_ = s.audits.Record(ctx, &models.AuditEntry{
		ActorID:    actor,
		Action:     audit.ActionAdminRegister,
		TargetID:   admin.ID,
		TargetType: "admin",
		Metadata:   map[string]any{"email": admin.Email, "roles": admin.Roles},
	})
	return admin, nil
}

// ListAdmins returns a page of platform admins.
func (s *AdminService) ListAdmins(ctx context.Context, actorID string, limit, offset int) ([]*models.PlatformAdmin, int, error) {
	admins, total, err := s.admins.List(ctx, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	_ = s.audits.Record(ctx, &models.AuditEntry{
		ActorID:    firstNonEmpty(actorID, "unknown"),
		Action:     audit.ActionListAdminUsers,
		TargetType: "admin_collection",
		Metadata:   map[string]any{"returned": len(admins), "total": total},
	})
	return admins, total, nil
}

// ImpersonateResult bundles the fields the RPC returns to the caller.
type ImpersonateResult struct {
	Token           string
	RedirectURL     string
	ImpersonationID string
	ExpiresAt       time.Time
}

// Impersonate mints an impersonation token if the admin has the required role.
func (s *AdminService) Impersonate(ctx context.Context, adminUserID, tenantID, tenantSlug, targetUserID string) (*ImpersonateResult, error) {
	if adminUserID == "" || tenantID == "" || tenantSlug == "" || targetUserID == "" {
		return nil, ErrInvalidInput
	}
	admin, err := s.admins.GetByID(ctx, adminUserID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ErrInvalidInput
		}
		return nil, err
	}
	if admin.Disabled {
		return nil, ErrAccountDisabled
	}
	if !admin.CanImpersonate() {
		_ = s.audits.Record(ctx, &models.AuditEntry{
			ActorID:    admin.ID,
			Action:     audit.ActionImpersonationRejected,
			TargetID:   targetUserID,
			TargetType: "tenant_user",
			Metadata: map[string]any{
				"tenant_id":   tenantID,
				"tenant_slug": tenantSlug,
				"reason":      "insufficient_role",
				"roles":       admin.Roles,
			},
		})
		return nil, ErrForbidden
	}
	token, impID, exp, err := s.tokens.IssueImpersonationToken(admin.ID, tenantID, tenantSlug, targetUserID)
	if err != nil {
		return nil, err
	}
	redirect := strings.ReplaceAll(s.redirect, "{slug}", tenantSlug)
	redirect = strings.ReplaceAll(redirect, "{token}", token)

	_ = s.audits.Record(ctx, &models.AuditEntry{
		ActorID:    admin.ID,
		Action:     audit.ActionImpersonationStarted,
		TargetID:   targetUserID,
		TargetType: "tenant_user",
		Metadata: map[string]any{
			"tenant_id":        tenantID,
			"tenant_slug":      tenantSlug,
			"impersonation_id": impID,
			"expires_at":       exp.UnixMilli(),
		},
	})
	return &ImpersonateResult{
		Token:           token,
		RedirectURL:     redirect,
		ImpersonationID: impID,
		ExpiresAt:       exp,
	}, nil
}

func (s *AdminService) auditLoginFailed(ctx context.Context, actor, reason string) {
	_ = s.audits.Record(ctx, &models.AuditEntry{
		ActorID:    firstNonEmpty(actor, "unknown"),
		Action:     audit.ActionAdminLoginFailed,
		TargetID:   actor,
		TargetType: "admin",
		Metadata:   map[string]any{"reason": reason, "request_id": commontracing.RequestIDFromContext(ctx)},
	})
}

func normalizeEmail(s string) string {
	return strings.TrimSpace(strings.ToLower(s))
}

func firstNonEmpty(a, b string) string {
	if a != "" {
		return a
	}
	return b
}

