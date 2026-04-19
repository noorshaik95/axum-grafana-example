package main

// W14 wiring — SSO, impersonation, MFA reset, ResolveUsername.
//
// Each feature is env-gated so a tenant can run without SSO configured or
// without the platform public key mounted. The production IdP clients
// (crewjam/saml, coreos/go-oidc, golang.org/x/oauth2) are NOT yet vendored —
// when SSO_ENABLED=true and TENANT_SSO_CONFIG is valid, the service boots
// with a stub IdPClient that returns "not implemented" on exchange. The
// strategies, state stores, and audit logging are all live; swap the stub
// when EXECUTION.md §3 whitelists the deps.

import (
	"context"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/redis/go-redis/v9"

	"slate/services/user-auth-service/internal/auth"
	"slate/services/user-auth-service/internal/auth/sso"
	httpadmin "slate/services/user-auth-service/internal/http"
	"slate/services/user-auth-service/internal/repository"
	"slate/services/user-auth-service/internal/service"
	"slate/services/user-auth-service/pkg/jwt"
	"slate/services/user-auth-service/pkg/logger"
)

// wireW14Handlers builds the HTTP Handler (returns nil if every feature is
// disabled). Callers should mount it onto an http.ServeMux before Serve.
func wireW14Handlers(
	log *logger.Logger,
	auditRepo repository.AuditRepository,
	userRepo service.UserRepositoryInterface,
	roleRepo service.RoleRepositoryInterface,
	mfaStore auth.MFAStore,
	tokenSvc *jwt.TokenService,
	redisClient *redis.Client,
) *httpadmin.Handler {
	h := &httpadmin.Handler{}

	// SSO manager — gated on SSO_ENABLED + TENANT_SSO_CONFIG.
	if os.Getenv("SSO_ENABLED") == "true" {
		cfg, err := sso.ParseConfig(os.Getenv("TENANT_SSO_CONFIG"))
		if err != nil {
			log.Warn().Err(err).Msg("W14 SSO disabled: invalid TENANT_SSO_CONFIG")
		} else {
			// TODO: swap for crewjam/saml + coreos/go-oidc wiring once those
			// packages are whitelisted in EXECUTION.md §3. The stub returns
			// errors on exchange but satisfies IdPClient + keeps the server
			// alive so other W14 features don't break.
			client := &stubIdPClient{}
			tokens := ssoTokenAdapter{t: tokenSvc}
			mgr, mErr := sso.NewManager(sso.ManagerOptions{
				Config: cfg, Client: client, AuditRepo: auditRepo,
				UserRepo: userRepo, RoleRepo: roleRepo, TokenIssuer: tokens,
			})
			if mErr != nil {
				log.Warn().Err(mErr).Msg("W14 SSO disabled: manager init failed")
			} else {
				h.SSOManager = mgr
				log.Info().Str("kind", string(cfg.Kind)).Msg("W14 SSO enabled")
			}
		}
	} else {
		log.Info().Msg("W14 SSO disabled (SSO_ENABLED != true)")
	}

	// Impersonation validator — gated on PLATFORM_PUBLIC_KEY.
	if pubKey := loadPlatformPublicKey(log); pubKey != nil {
		tenantID := os.Getenv("TENANT_ID")
		slug := os.Getenv("TENANT_SLUG")
		if tenantID == "" {
			log.Warn().Msg("W14 impersonation disabled: TENANT_ID not set")
		} else {
			var revStore auth.RevocationStore
			if redisClient != nil {
				revStore = newRedisRevocationStore(redisClient)
			} else {
				log.Warn().Msg("W14 impersonation using in-memory revocation (Redis unavailable)")
				revStore = auth.NewInMemoryRevocationStore()
			}
			issuer := impersonationTokenAdapter{t: tokenSvc}
			v, err := auth.NewImpersonationValidator(auth.ValidatorOptions{
				PublicKey:       pubKey,
				TenantID:        tenantID,
				TenantSlug:      slug,
				RevocationStore: revStore,
				AuditRepo:       auditRepo,
				TokenIssuer:     issuer,
			})
			if err != nil {
				log.Warn().Err(err).Msg("W14 impersonation disabled: validator init failed")
			} else {
				h.ImpersonationValidator = v
				log.Info().Str("tenant_id", tenantID).Msg("W14 impersonation enabled")
			}
		}

		// MFA reset — reuses the same platform public key.
		if mfaStore != nil {
			h.MFAReset = auth.NewMFAResetService(mfaStore, auth.NewRSAPlatformVerifier(pubKey), auditRepo)
			log.Info().Msg("W14 MFA reset enabled")
		}
	} else {
		log.Info().Msg("W14 impersonation + MFA reset disabled (PLATFORM_PUBLIC_KEY not set)")
	}

	if h.SSOManager == nil && h.ImpersonationValidator == nil && h.MFAReset == nil {
		return nil
	}
	return h
}

// loadPlatformPublicKey reads PLATFORM_PUBLIC_KEY (PEM) from env or, if that
// is empty, from the file at PLATFORM_PUBLIC_KEY_PATH. Returns nil when
// neither is set.
func loadPlatformPublicKey(log *logger.Logger) *rsa.PublicKey {
	pem := []byte(os.Getenv("PLATFORM_PUBLIC_KEY"))
	if len(pem) == 0 {
		path := os.Getenv("PLATFORM_PUBLIC_KEY_PATH")
		if path == "" {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			log.Warn().Err(err).Str("path", path).Msg("W14: failed to read PLATFORM_PUBLIC_KEY_PATH")
			return nil
		}
		pem = data
	}
	key, err := parseRSAPublicKey(pem)
	if err != nil {
		log.Warn().Err(err).Msg("W14: failed to parse PLATFORM_PUBLIC_KEY")
		return nil
	}
	return key
}

func parseRSAPublicKey(pemBytes []byte) (*rsa.PublicKey, error) {
	block, _ := pem.Decode(pemBytes)
	if block == nil {
		return nil, fmt.Errorf("invalid PEM")
	}
	pub, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		pub2, err2 := x509.ParsePKCS1PublicKey(block.Bytes)
		if err2 != nil {
			return nil, fmt.Errorf("PKIX and PKCS1 both failed: %v | %v", err, err2)
		}
		return pub2, nil
	}
	rsaKey, ok := pub.(*rsa.PublicKey)
	if !ok {
		return nil, fmt.Errorf("not an RSA public key")
	}
	return rsaKey, nil
}

// ssoTokenAdapter adapts *jwt.TokenService to sso.TokenIssuer.
type ssoTokenAdapter struct{ t *jwt.TokenService }

func (a ssoTokenAdapter) GenerateAccessToken(userID, email string, roles []string) (string, int64, error) {
	return a.t.GenerateAccessToken(userID, email, roles)
}
func (a ssoTokenAdapter) GenerateRefreshToken(userID, email string, roles []string) (string, error) {
	return a.t.GenerateRefreshToken(userID, email, roles)
}

// impersonationTokenAdapter adapts *jwt.TokenService to auth.ImpersonationIssuer.
// The platform public key is used for *verifying* inbound platform tokens;
// the tenant-side JWT we issue back is HS256-signed via TokenService, and we
// stash `impersonated_by` into the roles slot because the existing TokenService
// shape doesn't expose an arbitrary claims API.
type impersonationTokenAdapter struct{ t *jwt.TokenService }

func (a impersonationTokenAdapter) IssueImpersonatedToken(userID, email string, roles []string, impersonatedBy string, _ time.Duration) (string, int64, error) {
	augmentedRoles := append([]string{}, roles...)
	augmentedRoles = append(augmentedRoles, "impersonated_by:"+impersonatedBy)
	return a.t.GenerateAccessToken(userID, email, augmentedRoles)
}

// redisRevocationStore is a thin wrapper around redis.Client that implements
// auth.RevocationStore with SADD / SISMEMBER over per-tenant keys.
type redisRevocationStore struct{ c *redis.Client }

func newRedisRevocationStore(c *redis.Client) *redisRevocationStore { return &redisRevocationStore{c: c} }

func (r *redisRevocationStore) IsRevoked(ctx context.Context, slug, id string) (bool, error) {
	key := "tenant:" + slug + ":revoked:" + id
	exists, err := r.c.Exists(ctx, key).Result()
	if err != nil {
		return false, err
	}
	return exists > 0, nil
}

func (r *redisRevocationStore) Revoke(ctx context.Context, slug, id string, ttl time.Duration) error {
	key := "tenant:" + slug + ":revoked:" + id
	return r.c.Set(ctx, key, "1", ttl).Err()
}

// stubIdPClient returns errors on every exchange call so SSO_ENABLED=true
// against un-vendored IdP libs surfaces a clear operator error rather than a
// silent success. Swap when crewjam/saml + coreos/go-oidc are whitelisted.
type stubIdPClient struct{}

func (stubIdPClient) ExchangeSAMLResponse(context.Context, string, *sso.SAMLConfig) (map[string]string, error) {
	return nil, fmt.Errorf("saml client not wired (awaiting crewjam/saml vendor approval)")
}
func (stubIdPClient) ExchangeOIDCCode(context.Context, string, *sso.OIDCConfig) (map[string]string, error) {
	return nil, fmt.Errorf("oidc client not wired (awaiting coreos/go-oidc vendor approval)")
}
func (stubIdPClient) ExchangeGoogleCode(context.Context, string, *sso.GoogleConfig) (map[string]string, error) {
	return nil, fmt.Errorf("google client not wired (awaiting golang.org/x/oauth2 vendor approval)")
}
func (stubIdPClient) Probe(context.Context, *sso.Config) error { return nil }

// registerOnMux mounts the W14 handler onto the supplied mux. Called by
// main.go after the health endpoint is wired so we reuse the same HTTP
// listener (SERVER_PORT).
func registerOnMux(mux *http.ServeMux, h *httpadmin.Handler) {
	if h == nil {
		return
	}
	h.Register(mux)
}
