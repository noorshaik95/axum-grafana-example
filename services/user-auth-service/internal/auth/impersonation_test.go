package auth

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"testing"
	"time"

	"slate/services/user-auth-service/internal/models"
	"slate/services/user-auth-service/internal/repository"

	"github.com/golang-jwt/jwt/v5"
)

type fakeIssuer struct {
	calls int
}

func (f *fakeIssuer) IssueImpersonatedToken(userID, _ string, _ []string, actor string, ttl time.Duration) (string, int64, error) {
	f.calls++
	return "jwt-for-" + userID + "-by-" + actor, int64(ttl.Seconds()), nil
}

func generateKey(t *testing.T) *rsa.PrivateKey {
	t.Helper()
	k, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("rsa: %v", err)
	}
	return k
}

func signToken(t *testing.T, key *rsa.PrivateKey, claims ImpersonationClaims) string {
	t.Helper()
	tok := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	s, err := tok.SignedString(key)
	if err != nil {
		t.Fatalf("sign: %v", err)
	}
	return s
}

func newValidator(t *testing.T, priv *rsa.PrivateKey, tenantID, slug string, audit repository.AuditRepository, rev RevocationStore, iss ImpersonationIssuer) *ImpersonationValidator {
	t.Helper()
	v, err := NewImpersonationValidator(ValidatorOptions{
		PublicKey:       &priv.PublicKey,
		TenantID:        tenantID,
		TenantSlug:      slug,
		RevocationStore: rev,
		AuditRepo:       audit,
		TokenIssuer:     iss,
		TokenTTL:        30 * time.Minute,
	})
	if err != nil {
		t.Fatalf("validator: %v", err)
	}
	return v
}

func TestImpersonation_HappyPath(t *testing.T) {
	priv := generateKey(t)
	audit := repository.NewInMemoryAuditRepository()
	rev := NewInMemoryRevocationStore()
	iss := &fakeIssuer{}
	v := newValidator(t, priv, "tenant-1", "acme", audit, rev, iss)

	raw := signToken(t, priv, ImpersonationClaims{
		ImpersonationID: "imp-1",
		Type:            "impersonation",
		TenantID:        "tenant-1",
		TenantSlug:      "acme",
		TargetUserID:    "user-42",
		TargetEmail:     "alice@acme.com",
		ActorID:         "admin-9",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(5 * time.Minute)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	})
	res, err := v.Validate(context.Background(), raw, "1.2.3.4")
	if err != nil {
		t.Fatalf("validate: %v", err)
	}
	if res.AccessToken == "" || iss.calls != 1 {
		t.Fatalf("issuer not called: %+v", res)
	}
	if len(audit.ByAction(models.AuditActionImpersonationStart)) != 1 {
		t.Fatalf("want 1 impersonation_start audit row")
	}
}

func TestImpersonation_Expired(t *testing.T) {
	priv := generateKey(t)
	v := newValidator(t, priv, "tenant-1", "acme", repository.NewInMemoryAuditRepository(), NewInMemoryRevocationStore(), &fakeIssuer{})
	raw := signToken(t, priv, ImpersonationClaims{
		ImpersonationID: "imp", Type: "impersonation", TenantID: "tenant-1",
		TargetUserID: "u", ActorID: "a",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(-time.Minute)),
		},
	})
	if _, err := v.Validate(context.Background(), raw, ""); err == nil {
		t.Fatal("want expired error")
	}
}

func TestImpersonation_WrongTenant(t *testing.T) {
	priv := generateKey(t)
	v := newValidator(t, priv, "tenant-expected", "acme", repository.NewInMemoryAuditRepository(), NewInMemoryRevocationStore(), &fakeIssuer{})
	raw := signToken(t, priv, ImpersonationClaims{
		ImpersonationID: "imp", Type: "impersonation", TenantID: "tenant-other",
		TargetUserID: "u", ActorID: "a",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(5 * time.Minute)),
		},
	})
	if _, err := v.Validate(context.Background(), raw, ""); err != ErrImpersonationWrongTenant {
		t.Fatalf("want wrong tenant, got %v", err)
	}
}

func TestImpersonation_Revoked(t *testing.T) {
	priv := generateKey(t)
	audit := repository.NewInMemoryAuditRepository()
	rev := NewInMemoryRevocationStore()
	v := newValidator(t, priv, "tenant-1", "acme", audit, rev, &fakeIssuer{})

	_ = rev.Revoke(context.Background(), "acme", "imp-revoked", time.Hour)
	raw := signToken(t, priv, ImpersonationClaims{
		ImpersonationID: "imp-revoked", Type: "impersonation", TenantID: "tenant-1",
		TargetUserID: "u", ActorID: "a",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(5 * time.Minute)),
		},
	})
	if _, err := v.Validate(context.Background(), raw, ""); err != ErrImpersonationRevoked {
		t.Fatalf("want revoked error, got %v", err)
	}
}

func TestImpersonation_Revoke_AddsToSetAndAudits(t *testing.T) {
	priv := generateKey(t)
	audit := repository.NewInMemoryAuditRepository()
	rev := NewInMemoryRevocationStore()
	v := newValidator(t, priv, "tenant-1", "acme", audit, rev, &fakeIssuer{})
	if err := v.Revoke(context.Background(), "imp-7", "admin-1", "10.0.0.1"); err != nil {
		t.Fatalf("revoke: %v", err)
	}
	on, _ := rev.IsRevoked(context.Background(), "acme", "imp-7")
	if !on {
		t.Fatal("want impersonation_id in revocation set")
	}
	if len(audit.ByAction(models.AuditActionImpersonationEnd)) != 1 {
		t.Fatal("want impersonation_end audit row")
	}
}

func TestImpersonation_WrongType(t *testing.T) {
	priv := generateKey(t)
	v := newValidator(t, priv, "tenant-1", "acme", repository.NewInMemoryAuditRepository(), NewInMemoryRevocationStore(), &fakeIssuer{})
	raw := signToken(t, priv, ImpersonationClaims{
		ImpersonationID: "imp", Type: "access", TenantID: "tenant-1",
		TargetUserID: "u", ActorID: "a",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(5 * time.Minute)),
		},
	})
	if _, err := v.Validate(context.Background(), raw, ""); err != ErrImpersonationWrongType {
		t.Fatalf("want wrong type, got %v", err)
	}
}
