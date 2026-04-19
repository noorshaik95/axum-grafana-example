package sso

import (
	"context"
	"fmt"
	"net/url"
	"strings"
)

// GoogleStrategy implements the Google Workspace OAuth2 login flow. It is
// effectively a preconfigured OIDC flow aimed at accounts.google.com; hosted
// domain enforcement happens client-side via `hd` and server-side via the
// Hosted-Domain claim check in IdPClient.
type GoogleStrategy struct {
	cfg    *Config
	client IdPClient
	states *stateStore
}

// NewGoogleStrategy constructs a Google Workspace OAuth2 strategy.
func NewGoogleStrategy(cfg *Config, client IdPClient) (*GoogleStrategy, error) {
	if cfg == nil || cfg.Kind != ProviderGoogle || cfg.Google == nil {
		return nil, fmt.Errorf("google strategy: config must have kind=google")
	}
	if client == nil {
		return nil, fmt.Errorf("google strategy: idp client is required")
	}
	return &GoogleStrategy{cfg: cfg, client: client, states: newStateStore()}, nil
}

func (s *GoogleStrategy) Kind() ProviderKind { return ProviderGoogle }

func (s *GoogleStrategy) Initiate(_ context.Context, redirectBack string) (string, string, error) {
	state, err := s.states.issue()
	if err != nil {
		return "", "", err
	}
	u, _ := url.Parse("https://accounts.google.com/o/oauth2/v2/auth")
	scopes := s.cfg.Google.Scopes
	if len(scopes) == 0 {
		scopes = []string{"openid", "email", "profile"}
	}
	q := u.Query()
	q.Set("response_type", "code")
	q.Set("client_id", s.cfg.Google.ClientID)
	redirect := redirectBack
	if redirect == "" {
		redirect = s.cfg.Google.RedirectURI
	}
	q.Set("redirect_uri", redirect)
	q.Set("scope", strings.Join(scopes, " "))
	q.Set("state", state)
	if s.cfg.Google.HostedDomain != "" {
		q.Set("hd", s.cfg.Google.HostedDomain)
	}
	u.RawQuery = q.Encode()
	return u.String(), state, nil
}

func (s *GoogleStrategy) Callback(ctx context.Context, p CallbackParams) (*Identity, error) {
	if p.Code == "" {
		return nil, fmt.Errorf("code is required")
	}
	if err := s.states.consume(p.State); err != nil {
		return nil, err
	}
	claims, err := s.client.ExchangeGoogleCode(ctx, p.Code, s.cfg.Google)
	if err != nil {
		return nil, fmt.Errorf("google code exchange failed: %w", err)
	}
	if hd := s.cfg.Google.HostedDomain; hd != "" {
		if got := claims["hd"]; got != hd {
			return nil, fmt.Errorf("google: hosted domain mismatch (got %q, want %q)", got, hd)
		}
	}
	id := &Identity{
		Subject:   claims["sub"],
		Email:     claims["email"],
		FirstName: lookup(claims, "given_name", "first_name"),
		LastName:  lookup(claims, "family_name", "last_name"),
		Raw:       claims,
	}
	if len(id.Roles) == 0 && s.cfg.DefaultRole != "" {
		id.Roles = []string{s.cfg.DefaultRole}
	}
	id.Roles = mapRoles(id.Roles, s.cfg.RoleMap)
	return id, nil
}

func (s *GoogleStrategy) TestConnection(ctx context.Context) error {
	return s.client.Probe(ctx, s.cfg)
}
