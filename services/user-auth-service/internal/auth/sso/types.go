// Package sso implements the W14.1 tenant-aware SSO strategies (SAML, OIDC,
// Google Workspace). Each strategy consumes a JSON config from the
// TENANT_SSO_CONFIG env var injected at provisioning time.
//
// The strategies are structured around a small IdPClient interface so unit
// tests can inject mocked identity-provider responses. Production wiring would
// swap the mock for github.com/crewjam/saml, github.com/coreos/go-oidc, and
// golang.org/x/oauth2 respectively; the public Strategy surface stays the same.
package sso

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

// ProviderKind identifies which SSO strategy should handle a request.
type ProviderKind string

const (
	ProviderSAML   ProviderKind = "saml"
	ProviderOIDC   ProviderKind = "oidc"
	ProviderGoogle ProviderKind = "google"
)

// Config is the JSON payload read from TENANT_SSO_CONFIG. Exactly one of the
// per-provider blocks is required — Kind picks which one.
type Config struct {
	Kind        ProviderKind  `json:"kind"`
	TenantSlug  string        `json:"tenant_slug"`
	DefaultRole string        `json:"default_role"`
	RoleMap     map[string]string `json:"role_map,omitempty"`
	SAML        *SAMLConfig   `json:"saml,omitempty"`
	OIDC        *OIDCConfig   `json:"oidc,omitempty"`
	Google      *GoogleConfig `json:"google,omitempty"`
}

// SAMLConfig captures the subset of Shibboleth/SAML settings needed to
// generate an AuthnRequest and verify an incoming response.
type SAMLConfig struct {
	EntityID     string `json:"entity_id"`
	SSOURL       string `json:"sso_url"`
	ACSURL       string `json:"acs_url"`
	IdPMetadata  string `json:"idp_metadata,omitempty"`
	Certificate  string `json:"certificate,omitempty"`
	RoleAttr     string `json:"role_attribute,omitempty"`
	EmailAttr    string `json:"email_attribute,omitempty"`
	FirstAttr    string `json:"first_name_attribute,omitempty"`
	LastAttr     string `json:"last_name_attribute,omitempty"`
}

// OIDCConfig captures the OIDC settings (Okta, Azure AD, generic OIDC).
type OIDCConfig struct {
	Issuer       string   `json:"issuer"`
	ClientID     string   `json:"client_id"`
	ClientSecret string   `json:"client_secret"`
	RedirectURI  string   `json:"redirect_uri"`
	Scopes       []string `json:"scopes,omitempty"`
	RoleClaim    string   `json:"role_claim,omitempty"`
}

// GoogleConfig captures the Google Workspace OAuth2 settings.
type GoogleConfig struct {
	ClientID     string   `json:"client_id"`
	ClientSecret string   `json:"client_secret"`
	RedirectURI  string   `json:"redirect_uri"`
	HostedDomain string   `json:"hosted_domain,omitempty"`
	Scopes       []string `json:"scopes,omitempty"`
}

// ParseConfig decodes a JSON blob (typically the TENANT_SSO_CONFIG env var)
// and validates that the selected provider block is populated.
func ParseConfig(raw string) (*Config, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, fmt.Errorf("TENANT_SSO_CONFIG is empty")
	}
	var c Config
	if err := json.Unmarshal([]byte(raw), &c); err != nil {
		return nil, fmt.Errorf("failed to parse TENANT_SSO_CONFIG: %w", err)
	}
	if err := c.Validate(); err != nil {
		return nil, err
	}
	return &c, nil
}

// Validate ensures the required provider-specific block is present.
func (c *Config) Validate() error {
	switch c.Kind {
	case ProviderSAML:
		if c.SAML == nil {
			return fmt.Errorf("saml config missing")
		}
		if c.SAML.EntityID == "" || c.SAML.SSOURL == "" {
			return fmt.Errorf("saml: entity_id and sso_url are required")
		}
	case ProviderOIDC:
		if c.OIDC == nil {
			return fmt.Errorf("oidc config missing")
		}
		if c.OIDC.Issuer == "" || c.OIDC.ClientID == "" || c.OIDC.RedirectURI == "" {
			return fmt.Errorf("oidc: issuer, client_id, and redirect_uri are required")
		}
	case ProviderGoogle:
		if c.Google == nil {
			return fmt.Errorf("google config missing")
		}
		if c.Google.ClientID == "" || c.Google.RedirectURI == "" {
			return fmt.Errorf("google: client_id and redirect_uri are required")
		}
	default:
		return fmt.Errorf("unsupported sso kind: %q", c.Kind)
	}
	return nil
}

// Identity is the normalized view of a user returned by any strategy after a
// successful callback.
type Identity struct {
	Subject    string
	Email      string
	FirstName  string
	LastName   string
	Roles      []string
	Raw        map[string]string
}

// Strategy is the interface every SSO provider implements.
type Strategy interface {
	Kind() ProviderKind
	// Initiate returns the IdP URL the browser should be redirected to and the
	// CSRF state token the caller must echo back on callback.
	Initiate(ctx context.Context, redirectBack string) (authURL, state string, err error)
	// Callback parses the IdP response (OIDC/Google: `code` + `state`; SAML:
	// `saml_response` + `state`) and returns a normalized Identity.
	Callback(ctx context.Context, params CallbackParams) (*Identity, error)
	// TestConnection performs a non-authenticating probe against the IdP so the
	// admin onboarding wizard can verify the tenant SSO config is reachable.
	TestConnection(ctx context.Context) error
}

// CallbackParams carries the query/body values received from the IdP.
type CallbackParams struct {
	Code         string
	State        string
	SAMLResponse string
	RelayState   string
}
