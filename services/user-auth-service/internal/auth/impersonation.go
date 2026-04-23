// Package auth — impersonation.go implements W14.4 tenant-side validation of
// short-lived impersonation tokens minted by the platform.
//
// The platform issues an RS256-signed JWT of type=impersonation for a specific
// tenant. The tenant user-auth service:
//  1. Verifies the RSA signature against PLATFORM_PUBLIC_KEY.
//  2. Checks type=="impersonation", tenant_id matches this tenant, exp not past.
//  3. Checks Redis revocation set tenant:{slug}:revoked:{impersonation_id}.
//  4. Appends impersonation_start to audit_events.
//  5. Issues a 30-min tenant JWT carrying an impersonated_by claim.
//
// On DELETE, the impersonation_id is added to the revocation set with a 2h
// TTL (longer than the token's lifetime) and impersonation_end is audited.
package auth

import (
	"context"
	"crypto/rsa"
	"errors"
	"fmt"
	"time"

	"slate/services/user-auth-service/internal/models"
	"slate/services/user-auth-service/internal/repository"

	"github.com/golang-jwt/jwt/v5"
)

// Common impersonation errors. Handlers map these to HTTP statuses.
var (
	ErrImpersonationTokenMissing   = errors.New("impersonation token is required")
	ErrImpersonationTokenInvalid   = errors.New("impersonation token invalid")
	ErrImpersonationTokenExpired   = errors.New("impersonation token expired")
	ErrImpersonationWrongTenant    = errors.New("impersonation token not for this tenant")
	ErrImpersonationWrongType      = errors.New("token is not an impersonation token")
	ErrImpersonationRevoked        = errors.New("impersonation token has been revoked")
	ErrImpersonationConfigMissing  = errors.New("impersonation not configured on this tenant")
)

// RevocationStore abstracts the Redis set used to track revoked impersonation
// IDs. An in-memory implementation is provided for tests; the production
// wiring is a thin wrapper around redis.Client SADD/SISMEMBER.
type RevocationStore interface {
	IsRevoked(ctx context.Context, tenantSlug, impersonationID string) (bool, error)
	Revoke(ctx context.Context, tenantSlug, impersonationID string, ttl time.Duration) error
}

// ImpersonationIssuer issues the 30-min tenant JWT once validation passes.
// Declared as an interface so the tenant JWT service can be swapped in tests.
type ImpersonationIssuer interface {
	IssueImpersonatedToken(userID, email string, roles []string, impersonatedBy string, ttl time.Duration) (string, int64, error)
}

// ImpersonationClaims are the platform-issued claims we verify.
type ImpersonationClaims struct {
	ImpersonationID string   `json:"jti"`
	Type            string   `json:"type"`
	TenantID        string   `json:"tenant_id"`
	TenantSlug      string   `json:"tenant_slug,omitempty"`
	TargetUserID    string   `json:"target_user_id"`
	TargetEmail     string   `json:"target_email,omitempty"`
	TargetRoles     []string `json:"target_roles,omitempty"`
	ActorID         string   `json:"actor_id"`
	jwt.RegisteredClaims
}

// ImpersonationValidator validates incoming platform tokens and issues tenant
// JWTs. All operations audit through the injected AuditRepository.
type ImpersonationValidator struct {
	publicKey   *rsa.PublicKey
	tenantID    string
	tenantSlug  string
	revokeStore RevocationStore
	auditRepo   repository.AuditRepository
	issuer      ImpersonationIssuer
	// tokenTTL is how long the issued tenant JWT lives (defaults 30m).
	tokenTTL time.Duration
	// revocationTTL is how long a revoked ID stays in the set (defaults 2h).
	revocationTTL time.Duration
	// now is overridable in tests.
	now func() time.Time
}

// ValidatorOptions bundles the wiring required by ImpersonationValidator.
type ValidatorOptions struct {
	PublicKey       *rsa.PublicKey
	TenantID        string
	TenantSlug      string
	RevocationStore RevocationStore
	AuditRepo       repository.AuditRepository
	TokenIssuer     ImpersonationIssuer
	TokenTTL        time.Duration
	RevocationTTL   time.Duration
}

// NewImpersonationValidator constructs a validator, applying defaults for TTLs
// and a real-clock `now`.
func NewImpersonationValidator(opts ValidatorOptions) (*ImpersonationValidator, error) {
	if opts.PublicKey == nil {
		return nil, ErrImpersonationConfigMissing
	}
	if opts.TenantID == "" {
		return nil, fmt.Errorf("impersonation: tenant_id is required")
	}
	ttl := opts.TokenTTL
	if ttl <= 0 {
		ttl = 30 * time.Minute
	}
	revTTL := opts.RevocationTTL
	if revTTL <= 0 {
		revTTL = 2 * time.Hour
	}
	return &ImpersonationValidator{
		publicKey:     opts.PublicKey,
		tenantID:      opts.TenantID,
		tenantSlug:    opts.TenantSlug,
		revokeStore:   opts.RevocationStore,
		auditRepo:     opts.AuditRepo,
		issuer:        opts.TokenIssuer,
		tokenTTL:      ttl,
		revocationTTL: revTTL,
		now:           time.Now,
	}, nil
}

// ImpersonationResult is returned to the HTTP handler on success.
type ImpersonationResult struct {
	AccessToken     string
	ExpiresIn       int64
	ImpersonationID string
	TargetUserID    string
	TargetEmail     string
	ActorID         string
	TenantSlug      string
}

// Validate performs the full W14.4 chain: parse → signature → type → tenant →
// expiration → revocation → audit → issue.
func (v *ImpersonationValidator) Validate(ctx context.Context, rawToken, clientIP string) (*ImpersonationResult, error) {
	if rawToken == "" {
		return nil, ErrImpersonationTokenMissing
	}
	claims := &ImpersonationClaims{}
	token, err := jwt.ParseWithClaims(rawToken, claims, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return v.publicKey, nil
	})
	if err != nil || token == nil || !token.Valid {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, ErrImpersonationTokenExpired
		}
		return nil, ErrImpersonationTokenInvalid
	}
	if claims.Type != "impersonation" {
		return nil, ErrImpersonationWrongType
	}
	if claims.TenantID != v.tenantID {
		return nil, ErrImpersonationWrongTenant
	}
	if claims.ExpiresAt == nil || claims.ExpiresAt.Time.Before(v.now()) {
		return nil, ErrImpersonationTokenExpired
	}
	if v.revokeStore != nil {
		revoked, err := v.revokeStore.IsRevoked(ctx, v.tenantSlug, claims.ImpersonationID)
		if err == nil && revoked {
			return nil, ErrImpersonationRevoked
		}
	}

	var accessToken string
	var expiresIn int64
	if v.issuer != nil {
		accessToken, expiresIn, err = v.issuer.IssueImpersonatedToken(
			claims.TargetUserID,
			claims.TargetEmail,
			claims.TargetRoles,
			claims.ActorID,
			v.tokenTTL,
		)
		if err != nil {
			return nil, fmt.Errorf("impersonation: issue tenant JWT: %w", err)
		}
	}

	if v.auditRepo != nil {
		_ = v.auditRepo.Append(ctx, &models.AuditEvent{
			ActorID:    claims.ActorID,
			ActorType:  models.AuditActorAdmin,
			Action:     models.AuditActionImpersonationStart,
			TargetID:   claims.TargetUserID,
			TargetType: "user",
			IPAddress:  clientIP,
			Metadata: map[string]interface{}{
				"impersonation_id": claims.ImpersonationID,
				"tenant_slug":      v.tenantSlug,
			},
			CreatedAt: v.now().UTC(),
		})
	}

	// Prefer the slug embedded in the platform-issued JWT claim; fall back to
	// the locally-configured tenantSlug so the handler can always build the
	// fragment redirect URL without a round-trip.
	slug := claims.TenantSlug
	if slug == "" {
		slug = v.tenantSlug
	}
	return &ImpersonationResult{
		AccessToken:     accessToken,
		ExpiresIn:       expiresIn,
		ImpersonationID: claims.ImpersonationID,
		TargetUserID:    claims.TargetUserID,
		TargetEmail:     claims.TargetEmail,
		ActorID:         claims.ActorID,
		TenantSlug:      slug,
	}, nil
}

// Revoke adds an impersonation_id to the Redis set with the configured TTL
// and logs impersonation_end.
func (v *ImpersonationValidator) Revoke(ctx context.Context, impersonationID, actorID, clientIP string) error {
	if impersonationID == "" {
		return fmt.Errorf("impersonation_id is required")
	}
	if v.revokeStore == nil {
		return fmt.Errorf("impersonation: revocation store not configured")
	}
	if err := v.revokeStore.Revoke(ctx, v.tenantSlug, impersonationID, v.revocationTTL); err != nil {
		return fmt.Errorf("impersonation: revoke failed: %w", err)
	}
	if v.auditRepo != nil {
		_ = v.auditRepo.Append(ctx, &models.AuditEvent{
			ActorID:   actorID,
			ActorType: models.AuditActorAdmin,
			Action:    models.AuditActionImpersonationEnd,
			TargetID:  impersonationID,
			IPAddress: clientIP,
			Metadata: map[string]interface{}{
				"impersonation_id": impersonationID,
				"tenant_slug":      v.tenantSlug,
			},
			CreatedAt: v.now().UTC(),
		})
	}
	return nil
}

// SetClock replaces the internal clock. Tests only.
func (v *ImpersonationValidator) SetClock(now func() time.Time) {
	v.now = now
}

// InMemoryRevocationStore is a test-friendly RevocationStore. Keys are
// `{slug}:{id}`.
type InMemoryRevocationStore struct {
	items map[string]time.Time
	now   func() time.Time
}

// NewInMemoryRevocationStore returns an empty in-memory store.
func NewInMemoryRevocationStore() *InMemoryRevocationStore {
	return &InMemoryRevocationStore{items: make(map[string]time.Time), now: time.Now}
}

func key(slug, id string) string { return slug + ":" + id }

// IsRevoked returns true if the (slug,id) pair is in the store and its TTL has
// not expired.
func (s *InMemoryRevocationStore) IsRevoked(_ context.Context, slug, id string) (bool, error) {
	exp, ok := s.items[key(slug, id)]
	if !ok {
		return false, nil
	}
	if s.now().After(exp) {
		delete(s.items, key(slug, id))
		return false, nil
	}
	return true, nil
}

// Revoke records the pair with an absolute expiration.
func (s *InMemoryRevocationStore) Revoke(_ context.Context, slug, id string, ttl time.Duration) error {
	s.items[key(slug, id)] = s.now().Add(ttl)
	return nil
}
