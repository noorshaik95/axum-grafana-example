package sso

import (
	"context"
	"fmt"
	"net/url"
	"strings"
)

// SAMLStrategy handles Shibboleth (and any SAML 2.0) IdP interaction.
// In production the IdPClient is backed by github.com/crewjam/saml — here the
// strategy only concerns itself with issuing AuthnRequests, state tracking,
// and normalizing assertion attributes to Identity.
type SAMLStrategy struct {
	cfg    *Config
	client IdPClient
	states *stateStore
}

// NewSAMLStrategy constructs a SAML strategy. The client is required; use a
// mock implementation in tests.
func NewSAMLStrategy(cfg *Config, client IdPClient) (*SAMLStrategy, error) {
	if cfg == nil || cfg.Kind != ProviderSAML || cfg.SAML == nil {
		return nil, fmt.Errorf("saml strategy: config must have kind=saml")
	}
	if client == nil {
		return nil, fmt.Errorf("saml strategy: idp client is required")
	}
	return &SAMLStrategy{cfg: cfg, client: client, states: newStateStore()}, nil
}

func (s *SAMLStrategy) Kind() ProviderKind { return ProviderSAML }

// Initiate returns the SSO URL (with RelayState = CSRF token) that the browser
// should be redirected to.
func (s *SAMLStrategy) Initiate(_ context.Context, _ string) (string, string, error) {
	state, err := s.states.issue()
	if err != nil {
		return "", "", err
	}
	u, err := url.Parse(s.cfg.SAML.SSOURL)
	if err != nil {
		return "", "", fmt.Errorf("invalid sso_url: %w", err)
	}
	q := u.Query()
	q.Set("RelayState", state)
	q.Set("entity_id", s.cfg.SAML.EntityID)
	u.RawQuery = q.Encode()
	return u.String(), state, nil
}

// Callback validates the SAML response via the injected IdPClient, checks the
// relay-state CSRF token, and returns a normalized Identity.
func (s *SAMLStrategy) Callback(ctx context.Context, p CallbackParams) (*Identity, error) {
	if p.SAMLResponse == "" {
		return nil, fmt.Errorf("saml_response is required")
	}
	relay := p.RelayState
	if relay == "" {
		relay = p.State
	}
	if err := s.states.consume(relay); err != nil {
		return nil, err
	}
	attrs, err := s.client.ExchangeSAMLResponse(ctx, p.SAMLResponse, s.cfg.SAML)
	if err != nil {
		return nil, fmt.Errorf("saml response rejected: %w", err)
	}
	return s.identity(attrs), nil
}

func (s *SAMLStrategy) TestConnection(ctx context.Context) error {
	return s.client.Probe(ctx, s.cfg)
}

func (s *SAMLStrategy) identity(attrs map[string]string) *Identity {
	emailKey := firstNonEmpty(s.cfg.SAML.EmailAttr, "email", "mail",
		"urn:oid:0.9.2342.19200300.100.1.3")
	firstKey := firstNonEmpty(s.cfg.SAML.FirstAttr, "first_name", "givenName",
		"urn:oid:2.5.4.42")
	lastKey := firstNonEmpty(s.cfg.SAML.LastAttr, "last_name", "sn",
		"urn:oid:2.5.4.4")
	roleKey := firstNonEmpty(s.cfg.SAML.RoleAttr, "role", "eduPersonAffiliation",
		"urn:oid:1.3.6.1.4.1.5923.1.1.1.1")

	id := &Identity{
		Email:     lookup(attrs, emailKey, "email", "mail"),
		FirstName: lookup(attrs, firstKey, "first_name", "givenName"),
		LastName:  lookup(attrs, lastKey, "last_name", "sn"),
		Raw:       attrs,
	}
	if v := lookup(attrs, roleKey, "role"); v != "" {
		id.Roles = splitCSV(v)
	}
	if len(id.Roles) == 0 && s.cfg.DefaultRole != "" {
		id.Roles = []string{s.cfg.DefaultRole}
	}
	id.Roles = mapRoles(id.Roles, s.cfg.RoleMap)
	id.Subject = firstNonEmpty(attrs["NameID"], id.Email)
	return id
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
}

func lookup(m map[string]string, keys ...string) string {
	for _, k := range keys {
		if v, ok := m[k]; ok && v != "" {
			return v
		}
	}
	return ""
}

func splitCSV(s string) []string {
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if v := strings.TrimSpace(p); v != "" {
			out = append(out, v)
		}
	}
	return out
}

func mapRoles(in []string, rm map[string]string) []string {
	if len(rm) == 0 {
		return in
	}
	out := make([]string, 0, len(in))
	for _, r := range in {
		if mapped, ok := rm[r]; ok {
			out = append(out, mapped)
		} else {
			out = append(out, r)
		}
	}
	return out
}
