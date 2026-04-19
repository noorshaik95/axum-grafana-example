package sso

import (
	"context"
	"errors"
	"net/url"
	"strings"
	"testing"

	"slate/services/user-auth-service/internal/models"
	"slate/services/user-auth-service/internal/repository"
)

// mockIdPClient lets each test stage IdP responses without a live server.
type mockIdPClient struct {
	samlAttrs map[string]string
	oidcClaims map[string]string
	googleClaims map[string]string
	probeErr  error
	samlErr   error
	oidcErr   error
	googleErr error

	sawSAML   bool
	sawOIDC   bool
	sawGoogle bool
}

func (m *mockIdPClient) ExchangeSAMLResponse(_ context.Context, _ string, _ *SAMLConfig) (map[string]string, error) {
	m.sawSAML = true
	if m.samlErr != nil {
		return nil, m.samlErr
	}
	return m.samlAttrs, nil
}

func (m *mockIdPClient) ExchangeOIDCCode(_ context.Context, _ string, _ *OIDCConfig) (map[string]string, error) {
	m.sawOIDC = true
	if m.oidcErr != nil {
		return nil, m.oidcErr
	}
	return m.oidcClaims, nil
}

func (m *mockIdPClient) ExchangeGoogleCode(_ context.Context, _ string, _ *GoogleConfig) (map[string]string, error) {
	m.sawGoogle = true
	if m.googleErr != nil {
		return nil, m.googleErr
	}
	return m.googleClaims, nil
}

func (m *mockIdPClient) Probe(_ context.Context, _ *Config) error { return m.probeErr }

func TestParseConfig_SAML_OIDC_Google(t *testing.T) {
	cases := []struct {
		name string
		raw  string
		kind ProviderKind
	}{
		{"saml", `{"kind":"saml","tenant_slug":"acme","default_role":"student","saml":{"entity_id":"urn:acme","sso_url":"https://idp.example/sso"}}`, ProviderSAML},
		{"oidc", `{"kind":"oidc","tenant_slug":"acme","oidc":{"issuer":"https://okta","client_id":"cid","redirect_uri":"https://acme/cb"}}`, ProviderOIDC},
		{"google", `{"kind":"google","tenant_slug":"acme","google":{"client_id":"cid","redirect_uri":"https://acme/cb","hosted_domain":"acme.com"}}`, ProviderGoogle},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cfg, err := ParseConfig(tc.raw)
			if err != nil {
				t.Fatalf("parse: %v", err)
			}
			if cfg.Kind != tc.kind {
				t.Fatalf("want kind=%s got %s", tc.kind, cfg.Kind)
			}
		})
	}
}

func TestParseConfig_Invalid(t *testing.T) {
	if _, err := ParseConfig(""); err == nil {
		t.Fatal("empty config should fail")
	}
	if _, err := ParseConfig(`{"kind":"mystery"}`); err == nil {
		t.Fatal("unknown kind should fail")
	}
	if _, err := ParseConfig(`{"kind":"saml","saml":{}}`); err == nil {
		t.Fatal("missing saml fields should fail")
	}
}

func buildManager(t *testing.T, cfg *Config, client IdPClient, audit repository.AuditRepository) *Manager {
	t.Helper()
	m, err := NewManager(ManagerOptions{Config: cfg, Client: client, AuditRepo: audit})
	if err != nil {
		t.Fatalf("new manager: %v", err)
	}
	return m
}

func TestSAMLStrategy_CallbackMockIdP(t *testing.T) {
	audit := repository.NewInMemoryAuditRepository()
	cfg := &Config{
		Kind:        ProviderSAML,
		TenantSlug:  "acme",
		DefaultRole: "student",
		SAML: &SAMLConfig{
			EntityID:  "urn:acme",
			SSOURL:    "https://idp.example/sso",
			EmailAttr: "mail",
			FirstAttr: "givenName",
			LastAttr:  "sn",
			RoleAttr:  "eduPersonAffiliation",
		},
	}
	client := &mockIdPClient{samlAttrs: map[string]string{
		"mail":                 "alice@example.edu",
		"givenName":            "Alice",
		"sn":                   "Smith",
		"eduPersonAffiliation": "student,member",
	}}
	mgr := buildManager(t, cfg, client, audit)

	authURL, state, err := mgr.Initiate(context.Background(), "")
	if err != nil {
		t.Fatalf("initiate: %v", err)
	}
	u, err := url.Parse(authURL)
	if err != nil || u.Query().Get("RelayState") != state {
		t.Fatalf("auth URL missing RelayState: %s", authURL)
	}

	res, err := mgr.Callback(context.Background(), CallbackParams{SAMLResponse: "<xml/>", RelayState: state}, "10.0.0.1")
	if err != nil {
		t.Fatalf("callback: %v", err)
	}
	if !client.sawSAML {
		t.Fatal("idp client should have been called")
	}
	if res.User.Email != "alice@example.edu" {
		t.Fatalf("wrong email: %+v", res.User)
	}
	if len(audit.ByAction(models.AuditActionSSOLogin)) != 1 {
		t.Fatalf("want 1 sso_login audit row, got %d", len(audit.ByAction(models.AuditActionSSOLogin)))
	}
}

func TestSAMLStrategy_RejectsUnknownState(t *testing.T) {
	cfg := &Config{Kind: ProviderSAML, SAML: &SAMLConfig{EntityID: "urn:x", SSOURL: "https://idp/sso"}}
	mgr := buildManager(t, cfg, &mockIdPClient{}, nil)
	_, err := mgr.Callback(context.Background(), CallbackParams{SAMLResponse: "<x/>", RelayState: "not-issued"}, "")
	if err == nil {
		t.Fatal("want invalid state error")
	}
}

func TestOIDCStrategy_Callback(t *testing.T) {
	audit := repository.NewInMemoryAuditRepository()
	cfg := &Config{
		Kind:       ProviderOIDC,
		TenantSlug: "acme",
		OIDC: &OIDCConfig{
			Issuer:      "https://okta.example.com/",
			ClientID:    "cid",
			RedirectURI: "https://acme/cb",
			RoleClaim:   "roles",
		},
	}
	client := &mockIdPClient{oidcClaims: map[string]string{
		"sub":         "okta|alice",
		"email":       "alice@acme.com",
		"given_name":  "Alice",
		"family_name": "Smith",
		"roles":       "student",
	}}
	mgr := buildManager(t, cfg, client, audit)
	authURL, state, err := mgr.Initiate(context.Background(), "")
	if err != nil {
		t.Fatalf("initiate: %v", err)
	}
	if !strings.Contains(authURL, "/v1/authorize") {
		t.Fatalf("unexpected oidc authorize url: %s", authURL)
	}
	parsed, err := url.Parse(authURL)
	if err != nil || parsed.Query().Get("state") != state {
		t.Fatalf("oidc authorize url missing state: %s", authURL)
	}
	res, err := mgr.Callback(context.Background(), CallbackParams{Code: "code-xyz", State: state}, "")
	if err != nil {
		t.Fatalf("callback: %v", err)
	}
	if !client.sawOIDC {
		t.Fatal("expected OIDC client call")
	}
	if res.User.Email != "alice@acme.com" || len(res.User.Roles) == 0 {
		t.Fatalf("wrong identity: %+v", res.User)
	}
}

func TestGoogleStrategy_Callback_AndHostedDomainMismatch(t *testing.T) {
	cfg := &Config{
		Kind: ProviderGoogle,
		TenantSlug: "acme",
		DefaultRole: "student",
		Google: &GoogleConfig{
			ClientID:     "cid",
			RedirectURI:  "https://acme/cb",
			HostedDomain: "acme.com",
		},
	}
	good := &mockIdPClient{googleClaims: map[string]string{
		"sub": "gid", "email": "alice@acme.com", "given_name": "A", "family_name": "S", "hd": "acme.com",
	}}
	audit := repository.NewInMemoryAuditRepository()
	mgr := buildManager(t, cfg, good, audit)
	_, state, err := mgr.Initiate(context.Background(), "")
	if err != nil {
		t.Fatalf("initiate: %v", err)
	}
	res, err := mgr.Callback(context.Background(), CallbackParams{Code: "c", State: state}, "")
	if err != nil {
		t.Fatalf("google callback: %v", err)
	}
	if res.User.Email != "alice@acme.com" {
		t.Fatalf("wrong email: %+v", res.User)
	}

	// Hosted domain mismatch
	wrong := &mockIdPClient{googleClaims: map[string]string{
		"sub": "gid", "email": "alice@other.com", "hd": "other.com",
	}}
	mgr2 := buildManager(t, cfg, wrong, audit)
	_, state2, _ := mgr2.Initiate(context.Background(), "")
	_, err = mgr2.Callback(context.Background(), CallbackParams{Code: "c", State: state2}, "")
	if err == nil {
		t.Fatal("want hosted domain mismatch error")
	}
}

func TestOIDC_CodeExchangeError(t *testing.T) {
	cfg := &Config{
		Kind: ProviderOIDC,
		OIDC: &OIDCConfig{Issuer: "https://i", ClientID: "c", RedirectURI: "r"},
	}
	client := &mockIdPClient{oidcErr: errors.New("boom")}
	mgr := buildManager(t, cfg, client, nil)
	_, state, _ := mgr.Initiate(context.Background(), "")
	_, err := mgr.Callback(context.Background(), CallbackParams{Code: "c", State: state}, "")
	if err == nil || !strings.Contains(err.Error(), "boom") {
		t.Fatalf("want exchange error, got %v", err)
	}
}

func TestTestConnection_DelegatesToProbe(t *testing.T) {
	cfg := &Config{Kind: ProviderOIDC, OIDC: &OIDCConfig{Issuer: "i", ClientID: "c", RedirectURI: "r"}}
	client := &mockIdPClient{probeErr: errors.New("unreachable")}
	mgr := buildManager(t, cfg, client, nil)
	if err := mgr.TestConnection(context.Background()); err == nil {
		t.Fatal("want probe error")
	}
}
