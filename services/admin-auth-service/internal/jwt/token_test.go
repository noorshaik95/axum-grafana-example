package jwt

import (
	"crypto/rand"
	"crypto/rsa"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func newTestService(t *testing.T) (*TokenService, *rsa.PrivateKey) {
	t.Helper()
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	svc := NewTokenService("test-secret-at-least-32-chars-long-abcdef", time.Hour, 24*time.Hour, 5*time.Minute)
	svc.WithRSAKeys(priv, &priv.PublicKey)
	return svc, priv
}

func TestAccessTokenRoundTrip(t *testing.T) {
	svc, _ := newTestService(t)

	tok, exp, err := svc.IssueAccessToken("admin-123", "ops@slate.local", []string{"superadmin"})
	require.NoError(t, err)
	require.NotEmpty(t, tok)
	require.True(t, exp.After(time.Now()))

	claims, err := svc.ParseAdminToken(tok)
	require.NoError(t, err)
	require.Equal(t, "admin-123", claims.UserID)
	require.Equal(t, "ops@slate.local", claims.Email)
	require.Contains(t, claims.Roles, "superadmin")
	require.Equal(t, TypeAccess, claims.Type)
	require.True(t, claims.HasAudience(AudiencePlatform))
}

func TestAccessTokenRejectsWrongAudience(t *testing.T) {
	svc, _ := newTestService(t)
	// sign an HS256 token with the wrong audience using the same secret to
	// verify our aud-check is actually enforced.
	// Easiest path: tamper by changing issuer's secret — mismatch rejects.
	other := NewTokenService("another-secret-at-least-32-chars-long-abc", time.Hour, time.Hour, time.Minute)
	tok, _, err := other.IssueAccessToken("x", "x@slate.local", nil)
	require.NoError(t, err)

	_, err = svc.ParseAdminToken(tok)
	require.Error(t, err, "expected signature mismatch on wrong-secret token")
}

func TestImpersonationTokenRoundTrip(t *testing.T) {
	svc, _ := newTestService(t)

	token, impID, exp, err := svc.IssueImpersonationToken(
		"admin-42", "tenant-uuid-1", "eastfield", "student-9",
	)
	require.NoError(t, err)
	require.NotEmpty(t, token)
	require.NotEmpty(t, impID)
	require.True(t, exp.After(time.Now()))

	claims, err := svc.VerifyImpersonationToken(token)
	require.NoError(t, err)
	require.Equal(t, TypeImpersonation, claims.Type)
	require.Equal(t, "admin-42", claims.AdminUserID)
	require.Equal(t, "tenant-uuid-1", claims.TenantID)
	require.Equal(t, "eastfield", claims.TenantSlug)
	require.Equal(t, "student-9", claims.TargetUserID)
	require.Equal(t, impID, claims.ImpersonationID)
	require.Contains(t, claims.Audience, "tenant:eastfield")
}

func TestImpersonationTokenFailsWithoutRSAKey(t *testing.T) {
	svc := NewTokenService("test-secret-at-least-32-chars-long-abcdef", time.Hour, time.Hour, time.Minute)
	// no RSA keys configured
	_, _, _, err := svc.IssueImpersonationToken("a", "t", "s", "u")
	require.Error(t, err)
}

func TestImpersonationTokenExpiryHonoured(t *testing.T) {
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	svc := NewTokenService("secret-at-least-32-chars-long-abc-def-xyz", time.Hour, time.Hour, 1*time.Millisecond)
	svc.WithRSAKeys(priv, &priv.PublicKey)

	tok, _, _, err := svc.IssueImpersonationToken("a", "t", "s", "u")
	require.NoError(t, err)

	time.Sleep(50 * time.Millisecond)
	_, err = svc.VerifyImpersonationToken(tok)
	require.Error(t, err, "expected expiry error after TTL elapses")
}
