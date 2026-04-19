package http

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	stdhttp "net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"slate/services/user-auth-service/internal/auth"
	"slate/services/user-auth-service/internal/auth/sso"
	"slate/services/user-auth-service/internal/models"
	"slate/services/user-auth-service/internal/repository"

	"github.com/golang-jwt/jwt/v5"
)

type okIdP struct{}

func (okIdP) ExchangeSAMLResponse(_ context.Context, _ string, _ *sso.SAMLConfig) (map[string]string, error) {
	return map[string]string{"mail": "alice@example.edu", "givenName": "Alice", "sn": "Smith"}, nil
}
func (okIdP) ExchangeOIDCCode(_ context.Context, _ string, _ *sso.OIDCConfig) (map[string]string, error) {
	return map[string]string{"sub": "alice", "email": "alice@acme.com", "given_name": "Alice", "family_name": "Smith"}, nil
}
func (okIdP) ExchangeGoogleCode(_ context.Context, _ string, _ *sso.GoogleConfig) (map[string]string, error) {
	return map[string]string{"sub": "gid", "email": "alice@acme.com", "hd": "acme.com"}, nil
}
func (okIdP) Probe(_ context.Context, _ *sso.Config) error { return nil }

type fakeIssuer struct{}

func (fakeIssuer) IssueImpersonatedToken(userID, _ string, _ []string, actor string, ttl time.Duration) (string, int64, error) {
	return "jwt-" + userID + "-by-" + actor, int64(ttl.Seconds()), nil
}

func newTestServer(t *testing.T) (*httptest.Server, *rsa.PrivateKey, *repository.InMemoryAuditRepository, *auth.InMemoryRevocationStore) {
	t.Helper()
	cfg := &sso.Config{
		Kind:        sso.ProviderOIDC,
		TenantSlug:  "acme",
		DefaultRole: "student",
		OIDC: &sso.OIDCConfig{
			Issuer: "https://idp.example", ClientID: "c", RedirectURI: "https://acme/cb",
		},
	}
	audit := repository.NewInMemoryAuditRepository()
	mgr, err := sso.NewManager(sso.ManagerOptions{Config: cfg, Client: okIdP{}, AuditRepo: audit})
	if err != nil {
		t.Fatalf("new manager: %v", err)
	}

	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("rsa: %v", err)
	}
	rev := auth.NewInMemoryRevocationStore()
	validator, err := auth.NewImpersonationValidator(auth.ValidatorOptions{
		PublicKey: &priv.PublicKey, TenantID: "t1", TenantSlug: "acme",
		RevocationStore: rev, AuditRepo: audit, TokenIssuer: fakeIssuer{},
	})
	if err != nil {
		t.Fatalf("validator: %v", err)
	}

	h := &Handler{SSOManager: mgr, ImpersonationValidator: validator}
	mux := stdhttp.NewServeMux()
	h.Register(mux)
	return httptest.NewServer(mux), priv, audit, rev
}

func TestHTTP_SSOInitiate_ReturnsAuthURLAndState(t *testing.T) {
	srv, _, _, _ := newTestServer(t)
	defer srv.Close()
	resp, err := stdhttp.Get(srv.URL + "/api/auth/sso/initiate")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Fatalf("status %d", resp.StatusCode)
	}
	var body map[string]string
	_ = json.NewDecoder(resp.Body).Decode(&body)
	if !strings.Contains(body["authorize_url"], "state=") || body["state"] == "" {
		t.Fatalf("bad body: %+v", body)
	}
}

func TestHTTP_SSOCallback_FullCycle(t *testing.T) {
	srv, _, audit, _ := newTestServer(t)
	defer srv.Close()

	resp, err := stdhttp.Get(srv.URL + "/api/auth/sso/initiate")
	if err != nil {
		t.Fatalf("initiate: %v", err)
	}
	var body map[string]string
	_ = json.NewDecoder(resp.Body).Decode(&body)
	resp.Body.Close()

	cb, err := stdhttp.Get(srv.URL + "/api/auth/sso/callback?code=xyz&state=" + body["state"])
	if err != nil {
		t.Fatalf("callback: %v", err)
	}
	if cb.StatusCode != 200 {
		t.Fatalf("callback status %d", cb.StatusCode)
	}
	cb.Body.Close()
	if len(audit.ByAction(models.AuditActionSSOLogin)) != 1 {
		t.Fatal("want sso_login audit")
	}
}

func TestHTTP_TestSSO(t *testing.T) {
	srv, _, _, _ := newTestServer(t)
	defer srv.Close()
	resp, err := stdhttp.Post(srv.URL+"/api/admin/auth/test-sso", "application/json", nil)
	if err != nil {
		t.Fatalf("post: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Fatalf("status %d", resp.StatusCode)
	}
}

func TestHTTP_Impersonate_AndRevoke(t *testing.T) {
	srv, priv, audit, rev := newTestServer(t)
	defer srv.Close()

	claims := auth.ImpersonationClaims{
		ImpersonationID: "imp-99", Type: "impersonation", TenantID: "t1", TenantSlug: "acme",
		TargetUserID: "u", TargetEmail: "u@acme.com", ActorID: "admin-1",
		RegisteredClaims: jwt.RegisteredClaims{ExpiresAt: jwt.NewNumericDate(time.Now().Add(5 * time.Minute))},
	}
	tok := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	raw, err := tok.SignedString(priv)
	if err != nil {
		t.Fatalf("sign: %v", err)
	}
	body, _ := json.Marshal(map[string]string{"token": raw})
	resp, err := stdhttp.Post(srv.URL+"/api/auth/impersonate", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("post: %v", err)
	}
	if resp.StatusCode != 200 {
		t.Fatalf("status %d", resp.StatusCode)
	}
	resp.Body.Close()

	if n := len(audit.ByAction(models.AuditActionImpersonationStart)); n != 1 {
		t.Fatalf("want impersonation_start audit, got %d", n)
	}

	req, _ := stdhttp.NewRequest("DELETE", srv.URL+"/api/auth/impersonate/imp-99", nil)
	req.Header.Set("X-Actor-Id", "admin-1")
	del, err := stdhttp.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("delete: %v", err)
	}
	del.Body.Close()
	if del.StatusCode != 204 {
		t.Fatalf("delete status %d", del.StatusCode)
	}
	on, _ := rev.IsRevoked(context.Background(), "acme", "imp-99")
	if !on {
		t.Fatal("want revoked")
	}
}
