package http

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	stdhttp "net/http"
	"strings"
	"testing"
	"time"

	"slate/services/user-auth-service/internal/auth"
	"slate/services/user-auth-service/internal/models"

	"github.com/golang-jwt/jwt/v5"
)

func mustNewRSAKey(t *testing.T) *rsa.PrivateKey {
	t.Helper()
	k, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("rsa.GenerateKey: %v", err)
	}
	return k
}

// End-to-end proof for the admin-auth ↔ user-auth impersonation handshake
// requested by po-analyst. Simulates admin-auth signing an RS256 token with
// its platform private key, then walks it through user-auth's
// /auth/impersonate redirect-target handler (the browser-facing URL from
// plan/CONTRACTS.md admin_auth.redirect_url), and asserts each of the 5
// validation steps fires.

// Step 1: GET /auth/impersonate?token=<admin-auth RS256 JWT> succeeds with a
// tenant session JWT when the token is well-formed and unrevoked.
func TestImpersonation_AdminAuthRedirectHandshake_FullChain(t *testing.T) {
	srv, priv, audit, rev := newTestServer(t)
	defer srv.Close()

	// Mint a token shaped exactly like admin-auth's issuer would (W2):
	//   alg=RS256, type=impersonation, tenant_id, target_user_id,
	//   actor_id (admin sub), 5-min TTL, unique jti.
	claims := auth.ImpersonationClaims{
		ImpersonationID: "imp-e2e-1",
		Type:            "impersonation",
		TenantID:        "t1",
		TenantSlug:      "acme",
		TargetUserID:    "user-42",
		TargetEmail:     "target@acme.com",
		TargetRoles:     []string{"instructor"},
		ActorID:         "admin-007",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(5 * time.Minute)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	signed, err := jwt.NewWithClaims(jwt.SigningMethodRS256, claims).SignedString(priv)
	if err != nil {
		t.Fatalf("sign: %v", err)
	}

	// Hit the browser-redirect URL shape: /auth/impersonate?token=<jwt> (GET).
	resp, err := stdhttp.Get(srv.URL + "/auth/impersonate?token=" + signed)
	if err != nil {
		t.Fatalf("GET /auth/impersonate: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != stdhttp.StatusOK {
		t.Fatalf("want 200, got %d", resp.StatusCode)
	}

	// Step 5 response: tenant JWT + impersonation_id echoed back.
	var body struct {
		AccessToken     string `json:"access_token"`
		ExpiresIn       int64  `json:"expires_in"`
		ImpersonationID string `json:"impersonation_id"`
		TargetUserID    string `json:"target_user_id"`
		ActorID         string `json:"actor_id"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body.AccessToken == "" || body.ImpersonationID != "imp-e2e-1" || body.TargetUserID != "user-42" || body.ActorID != "admin-007" {
		t.Fatalf("bad response body: %+v", body)
	}

	// Steps 1-5 observable via audit row + revocation still clean.
	if starts := audit.ByAction(models.AuditActionImpersonationStart); len(starts) != 1 {
		t.Fatalf("want 1 impersonation_start audit, got %d", len(starts))
	} else if starts[0].ActorID != "admin-007" || starts[0].TargetID != "user-42" {
		t.Fatalf("audit row wrong: %+v", starts[0])
	}
	if on, _ := rev.IsRevoked(context.Background(), "acme", "imp-e2e-1"); on {
		t.Fatal("token must not be revoked after initial validation")
	}
}

// Step 2: type != "impersonation" is rejected (e.g. an admin access_token
// accidentally used as impersonation credential).
func TestImpersonation_WrongTypeRejected(t *testing.T) {
	srv, priv, _, _ := newTestServer(t)
	defer srv.Close()
	claims := auth.ImpersonationClaims{
		ImpersonationID: "imp-x",
		Type:            "access", // not impersonation
		TenantID:        "t1",
		TargetUserID:    "user-42",
		ActorID:         "admin",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(5 * time.Minute)),
		},
	}
	raw, _ := jwt.NewWithClaims(jwt.SigningMethodRS256, claims).SignedString(priv)
	resp, _ := stdhttp.Get(srv.URL + "/auth/impersonate?token=" + raw)
	defer resp.Body.Close()
	if resp.StatusCode != stdhttp.StatusForbidden {
		t.Fatalf("want 403 for wrong type, got %d", resp.StatusCode)
	}
}

// Step 3: expired token rejected.
func TestImpersonation_ExpiredRejected(t *testing.T) {
	srv, priv, _, _ := newTestServer(t)
	defer srv.Close()
	claims := auth.ImpersonationClaims{
		ImpersonationID: "imp-old",
		Type:            "impersonation",
		TenantID:        "t1",
		TargetUserID:    "user-42",
		ActorID:         "admin",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(-1 * time.Minute)),
		},
	}
	raw, _ := jwt.NewWithClaims(jwt.SigningMethodRS256, claims).SignedString(priv)
	resp, _ := stdhttp.Get(srv.URL + "/auth/impersonate?token=" + raw)
	defer resp.Body.Close()
	if resp.StatusCode != stdhttp.StatusUnauthorized {
		t.Fatalf("want 401 for expired, got %d", resp.StatusCode)
	}
}

// Step 3: wrong tenant rejected.
func TestImpersonation_WrongTenantRejected(t *testing.T) {
	srv, priv, _, _ := newTestServer(t)
	defer srv.Close()
	claims := auth.ImpersonationClaims{
		ImpersonationID: "imp-z",
		Type:            "impersonation",
		TenantID:        "some-other-tenant",
		TargetUserID:    "user-42",
		ActorID:         "admin",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(5 * time.Minute)),
		},
	}
	raw, _ := jwt.NewWithClaims(jwt.SigningMethodRS256, claims).SignedString(priv)
	resp, _ := stdhttp.Get(srv.URL + "/auth/impersonate?token=" + raw)
	defer resp.Body.Close()
	if resp.StatusCode != stdhttp.StatusForbidden {
		t.Fatalf("want 403 for wrong tenant, got %d", resp.StatusCode)
	}
}

// Step 4: revoked token rejected. Simulates admin-auth (or this service's
// DELETE /api/auth/impersonate/:id) having added the id to the shared Redis
// set `tenant:{slug}:revoked:{id}` per admin_auth contract line 275.
func TestImpersonation_RevokedTokenRejected(t *testing.T) {
	srv, priv, _, rev := newTestServer(t)
	defer srv.Close()
	// Pre-revoke the id (simulating cross-service Redis write).
	if err := rev.Revoke(context.Background(), "acme", "imp-already-dead", time.Hour); err != nil {
		t.Fatalf("pre-revoke: %v", err)
	}
	claims := auth.ImpersonationClaims{
		ImpersonationID: "imp-already-dead",
		Type:            "impersonation",
		TenantID:        "t1",
		TenantSlug:      "acme",
		TargetUserID:    "user-42",
		ActorID:         "admin",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(5 * time.Minute)),
		},
	}
	raw, _ := jwt.NewWithClaims(jwt.SigningMethodRS256, claims).SignedString(priv)
	resp, _ := stdhttp.Get(srv.URL + "/auth/impersonate?token=" + raw)
	defer resp.Body.Close()
	if resp.StatusCode != stdhttp.StatusForbidden {
		t.Fatalf("want 403 for revoked, got %d", resp.StatusCode)
	}
}

// Signature forgery attempt: signed with a DIFFERENT RSA key → rejected even
// if every other claim is correct.
func TestImpersonation_ForgedSignatureRejected(t *testing.T) {
	srv, _, _, _ := newTestServer(t)
	defer srv.Close()
	// A fresh key unrelated to the server's trusted public key.
	attackerKey := mustNewRSAKey(t)
	claims := auth.ImpersonationClaims{
		ImpersonationID: "imp-forged",
		Type:            "impersonation",
		TenantID:        "t1",
		TargetUserID:    "user-42",
		ActorID:         "mallory",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(5 * time.Minute)),
		},
	}
	raw, err := jwt.NewWithClaims(jwt.SigningMethodRS256, claims).SignedString(attackerKey)
	if err != nil {
		t.Fatalf("sign with attacker key: %v", err)
	}
	resp, _ := stdhttp.Get(srv.URL + "/auth/impersonate?token=" + raw)
	defer resp.Body.Close()
	if resp.StatusCode != stdhttp.StatusUnauthorized {
		t.Fatalf("want 401 for forged sig, got %d", resp.StatusCode)
	}
}

// DELETE /api/auth/impersonate/:id revokes the token. A subsequent validate
// call for the same id on a still-unexpired token then fails with 403. This
// demonstrates the full issue→validate→revoke→rejected cycle on the user-auth
// side (the admin-auth side has mirror coverage per CONTRACTS.md §admin_auth).
func TestImpersonation_IssueValidateRevokeRejectedCycle(t *testing.T) {
	srv, priv, _, _ := newTestServer(t)
	defer srv.Close()

	claims := auth.ImpersonationClaims{
		ImpersonationID: "imp-cycle",
		Type:            "impersonation",
		TenantID:        "t1",
		TenantSlug:      "acme",
		TargetUserID:    "user-42",
		TargetEmail:     "t@a.com",
		ActorID:         "admin",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(5 * time.Minute)),
		},
	}
	raw, _ := jwt.NewWithClaims(jwt.SigningMethodRS256, claims).SignedString(priv)

	// 1. First validate succeeds.
	r1, _ := stdhttp.Get(srv.URL + "/auth/impersonate?token=" + raw)
	r1.Body.Close()
	if r1.StatusCode != stdhttp.StatusOK {
		t.Fatalf("first validate want 200, got %d", r1.StatusCode)
	}

	// 2. Revoke via DELETE.
	req, _ := stdhttp.NewRequest("DELETE", srv.URL+"/api/auth/impersonate/imp-cycle", nil)
	req.Header.Set("X-Actor-Id", "admin")
	del, err := stdhttp.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("delete: %v", err)
	}
	del.Body.Close()
	if del.StatusCode != stdhttp.StatusNoContent {
		t.Fatalf("delete want 204, got %d", del.StatusCode)
	}

	// 3. Re-validate → 403 (revoked).
	r2, _ := stdhttp.Get(srv.URL + "/auth/impersonate?token=" + raw)
	r2.Body.Close()
	if r2.StatusCode != stdhttp.StatusForbidden {
		t.Fatalf("after revoke want 403, got %d", r2.StatusCode)
	}
}

// Sanity: the trace-propagation middleware is active on /auth/impersonate
// (X-Request-ID echoed).
func TestImpersonation_AuthImpersonateEchoesRequestID(t *testing.T) {
	srv, priv, _, _ := newTestServer(t)
	defer srv.Close()
	claims := auth.ImpersonationClaims{
		ImpersonationID: "imp-trace", Type: "impersonation", TenantID: "t1",
		TenantSlug: "acme", TargetUserID: "u", ActorID: "a",
		RegisteredClaims: jwt.RegisteredClaims{ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Minute))},
	}
	raw, _ := jwt.NewWithClaims(jwt.SigningMethodRS256, claims).SignedString(priv)
	req, _ := stdhttp.NewRequest("GET", srv.URL+"/auth/impersonate?token="+raw, nil)
	req.Header.Set("X-Request-Id", "trace-abc")
	resp, err := stdhttp.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	defer resp.Body.Close()
	if got := resp.Header.Get("X-Request-Id"); got != "trace-abc" {
		t.Fatalf("want echoed trace-abc, got %q", got)
	}
	if !strings.HasPrefix(resp.Header.Get("Content-Type"), "application/json") {
		t.Fatalf("want JSON response, got %s", resp.Header.Get("Content-Type"))
	}
}
