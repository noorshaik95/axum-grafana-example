// W14.2 — ResetMFA implementation.
//
// The top-level proto has not been regenerated to add the ResetMFA rpc yet
// (out of scope for this change, which must not break existing generated
// code). This file implements the reset logic behind an interface that the
// gRPC handler + HTTP admin route both call. A platform admin JWT
// (aud=platform) must be supplied; anything else is rejected.

package auth

import (
	"context"
	"errors"
	"fmt"
	"time"

	"slate/services/user-auth-service/internal/models"
	"slate/services/user-auth-service/internal/repository"

	"github.com/golang-jwt/jwt/v5"
)

// Errors returned by the MFA reset path. Handlers map these to gRPC / HTTP
// status codes.
var (
	ErrMFAResetTokenMissing = errors.New("platform admin token is required")
	ErrMFAResetTokenInvalid = errors.New("platform admin token invalid")
	ErrMFAResetNotPlatform  = errors.New("token audience must be 'platform'")
	ErrMFAResetUserRequired = errors.New("user_id is required")
)

// MFAStore is the subset of MFARepository that ResetMFA needs. Abstracted for
// tests.
type MFAStore interface {
	GetByUserID(ctx context.Context, userID string) ([]*models.UserMFA, error)
	Delete(ctx context.Context, userID, mfaType string) error
}

// PlatformTokenVerifier parses and validates a platform admin JWT. The
// default implementation verifies RS256 with the platform public key (same key
// used for impersonation tokens).
type PlatformTokenVerifier interface {
	VerifyPlatformAdmin(token string) (adminID string, err error)
}

// MFAResetService implements the ResetMFA operation.
type MFAResetService struct {
	mfa       MFAStore
	verifier  PlatformTokenVerifier
	auditRepo repository.AuditRepository
	now       func() time.Time
}

// NewMFAResetService wires the reset service.
func NewMFAResetService(mfa MFAStore, verifier PlatformTokenVerifier, auditRepo repository.AuditRepository) *MFAResetService {
	return &MFAResetService{mfa: mfa, verifier: verifier, auditRepo: auditRepo, now: time.Now}
}

// Reset validates the platform admin token, deletes all MFA configs for the
// user, and audits mfa_change.
func (s *MFAResetService) Reset(ctx context.Context, platformToken, userID, clientIP string) error {
	if userID == "" {
		return ErrMFAResetUserRequired
	}
	if platformToken == "" {
		return ErrMFAResetTokenMissing
	}
	if s.verifier == nil {
		return fmt.Errorf("mfa reset: platform verifier not configured")
	}
	adminID, err := s.verifier.VerifyPlatformAdmin(platformToken)
	if err != nil {
		return err
	}
	configs, err := s.mfa.GetByUserID(ctx, userID)
	if err != nil && !errors.Is(err, ErrNoMFAConfigs) {
		// Best-effort: fall through and try to delete the common types. The
		// Delete call is idempotent so no harm if nothing exists.
		configs = nil
	}
	if len(configs) == 0 {
		for _, t := range []string{"totp", "sms", "email"} {
			_ = s.mfa.Delete(ctx, userID, t)
		}
	} else {
		for _, cfg := range configs {
			if err := s.mfa.Delete(ctx, userID, cfg.MFAType); err != nil {
				return fmt.Errorf("mfa reset: delete %s: %w", cfg.MFAType, err)
			}
		}
	}
	if s.auditRepo != nil {
		_ = s.auditRepo.Append(ctx, &models.AuditEvent{
			ActorID:    adminID,
			ActorType:  models.AuditActorAdmin,
			Action:     models.AuditActionMFAChange,
			TargetID:   userID,
			TargetType: "user",
			IPAddress:  clientIP,
			Metadata: map[string]interface{}{
				"op": "reset",
			},
			CreatedAt: s.now().UTC(),
		})
	}
	return nil
}

// SetClock overrides the clock for tests.
func (s *MFAResetService) SetClock(now func() time.Time) { s.now = now }

// ErrNoMFAConfigs is returned by stores that distinguish "no rows" from real
// errors. Compared via errors.Is in Reset.
var ErrNoMFAConfigs = errors.New("no mfa configurations")

// RSAPlatformVerifier verifies RS256 platform admin JWTs and checks aud=platform.
type RSAPlatformVerifier struct {
	pubKey interface{}
	now    func() time.Time
}

// NewRSAPlatformVerifier builds a verifier with the given public key (either
// an *rsa.PublicKey or anything jwt accepts).
func NewRSAPlatformVerifier(pubKey interface{}) *RSAPlatformVerifier {
	return &RSAPlatformVerifier{pubKey: pubKey, now: time.Now}
}

// VerifyPlatformAdmin parses, signs-checks, and enforces aud=platform on the
// token. Returns the admin's sub claim.
func (v *RSAPlatformVerifier) VerifyPlatformAdmin(tokenStr string) (string, error) {
	if tokenStr == "" {
		return "", ErrMFAResetTokenMissing
	}
	claims := jwt.MapClaims{}
	tok, err := jwt.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return v.pubKey, nil
	})
	if err != nil || tok == nil || !tok.Valid {
		return "", ErrMFAResetTokenInvalid
	}
	if exp, ok := claims["exp"].(float64); ok {
		if time.Unix(int64(exp), 0).Before(v.now()) {
			return "", ErrMFAResetTokenInvalid
		}
	}
	aud, _ := claims["aud"].(string)
	if aud != "platform" {
		// Also tolerate aud claim encoded as slice
		if arr, ok := claims["aud"].([]interface{}); ok {
			found := false
			for _, a := range arr {
				if s, _ := a.(string); s == "platform" {
					found = true
					break
				}
			}
			if !found {
				return "", ErrMFAResetNotPlatform
			}
		} else {
			return "", ErrMFAResetNotPlatform
		}
	}
	sub, _ := claims["sub"].(string)
	if sub == "" {
		return "", ErrMFAResetTokenInvalid
	}
	return sub, nil
}
