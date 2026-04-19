package sso

import "context"

// IdPClient abstracts IdP interaction so the strategies can be unit-tested
// without making network calls. Production builds wire this to
// github.com/crewjam/saml, github.com/coreos/go-oidc, and golang.org/x/oauth2
// respectively (see cmd/server for the real wiring).
type IdPClient interface {
	// ExchangeSAMLResponse validates the SAML response XML/bytes and returns
	// the attributes keyed by the SAML attribute name.
	ExchangeSAMLResponse(ctx context.Context, rawResponse string, cfg *SAMLConfig) (map[string]string, error)

	// ExchangeOIDCCode swaps an authorization code for an id_token's claims.
	ExchangeOIDCCode(ctx context.Context, code string, cfg *OIDCConfig) (map[string]string, error)

	// ExchangeGoogleCode swaps an authorization code for a Google Workspace
	// userinfo response.
	ExchangeGoogleCode(ctx context.Context, code string, cfg *GoogleConfig) (map[string]string, error)

	// Probe performs a cheap reachability check against the IdP for
	// TestConnection endpoints. Returns nil on success.
	Probe(ctx context.Context, cfg *Config) error
}
