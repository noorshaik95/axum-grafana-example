package sso

import (
	"context"
	"fmt"
	"net/url"
	"strings"
)

// OIDCStrategy implements the OIDC Authorization Code flow for generic OIDC
// IdPs (Okta, Azure AD). Token verification is delegated to the IdPClient; in
// production that calls github.com/coreos/go-oidc to verify the id_token.
type OIDCStrategy struct {
	cfg    *Config
	client IdPClient
	states *stateStore
}

// NewOIDCStrategy constructs an OIDC strategy.
func NewOIDCStrategy(cfg *Config, client IdPClient) (*OIDCStrategy, error) {
	if cfg == nil || cfg.Kind != ProviderOIDC || cfg.OIDC == nil {
		return nil, fmt.Errorf("oidc strategy: config must have kind=oidc")
	}
	if client == nil {
		return nil, fmt.Errorf("oidc strategy: idp client is required")
	}
	return &OIDCStrategy{cfg: cfg, client: client, states: newStateStore()}, nil
}

func (s *OIDCStrategy) Kind() ProviderKind { return ProviderOIDC }

// Initiate builds the authorize URL (issuer/authorize) with the configured
// client_id, scopes, redirect_uri, and a CSRF state.
func (s *OIDCStrategy) Initiate(_ context.Context, redirectBack string) (string, string, error) {
	state, err := s.states.issue()
	if err != nil {
		return "", "", err
	}
	authorize := strings.TrimRight(s.cfg.OIDC.Issuer, "/") + "/v1/authorize"
	u, err := url.Parse(authorize)
	if err != nil {
		return "", "", fmt.Errorf("invalid issuer: %w", err)
	}
	scopes := s.cfg.OIDC.Scopes
	if len(scopes) == 0 {
		scopes = []string{"openid", "profile", "email"}
	}
	q := u.Query()
	q.Set("response_type", "code")
	q.Set("client_id", s.cfg.OIDC.ClientID)
	redirect := redirectBack
	if redirect == "" {
		redirect = s.cfg.OIDC.RedirectURI
	}
	q.Set("redirect_uri", redirect)
	q.Set("scope", strings.Join(scopes, " "))
	q.Set("state", state)
	u.RawQuery = q.Encode()
	return u.String(), state, nil
}

// Callback validates the state, exchanges the code with the IdPClient, and
// normalizes claims to Identity.
func (s *OIDCStrategy) Callback(ctx context.Context, p CallbackParams) (*Identity, error) {
	if p.Code == "" {
		return nil, fmt.Errorf("code is required")
	}
	if err := s.states.consume(p.State); err != nil {
		return nil, err
	}
	claims, err := s.client.ExchangeOIDCCode(ctx, p.Code, s.cfg.OIDC)
	if err != nil {
		return nil, fmt.Errorf("oidc code exchange failed: %w", err)
	}
	id := &Identity{
		Subject:   claims["sub"],
		Email:     lookup(claims, "email"),
		FirstName: lookup(claims, "given_name", "first_name"),
		LastName:  lookup(claims, "family_name", "last_name"),
		Raw:       claims,
	}
	roleClaim := s.cfg.OIDC.RoleClaim
	if roleClaim == "" {
		roleClaim = "roles"
	}
	if v, ok := claims[roleClaim]; ok && v != "" {
		id.Roles = splitCSV(v)
	}
	if len(id.Roles) == 0 && s.cfg.DefaultRole != "" {
		id.Roles = []string{s.cfg.DefaultRole}
	}
	id.Roles = mapRoles(id.Roles, s.cfg.RoleMap)
	return id, nil
}

func (s *OIDCStrategy) TestConnection(ctx context.Context) error {
	return s.client.Probe(ctx, s.cfg)
}
