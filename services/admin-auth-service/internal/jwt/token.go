// Package jwt issues and verifies tokens for admin-auth-service.
// Two kinds of tokens are produced:
//  1. Admin access/refresh tokens — HS256, aud="platform".
//  2. Impersonation tokens — RS256, type="impersonation", aud="tenant:{slug}", 5m TTL.
package jwt

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/hex"
	"encoding/pem"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// AudiencePlatform is the required aud claim on admin tokens.
const AudiencePlatform = "platform"

// TokenType constants.
const (
	TypeAccess        = "access"
	TypeRefresh       = "refresh"
	TypeImpersonation = "impersonation"
)

// AccessClaims are embedded in admin access/refresh tokens (HS256).
type AccessClaims struct {
	UserID string   `json:"user_id"`
	Email  string   `json:"email"`
	Roles  []string `json:"roles"`
	Type   string   `json:"type"`
	jwt.RegisteredClaims
}

// ImpersonationClaims are embedded in impersonation tokens (RS256).
type ImpersonationClaims struct {
	AdminUserID     string `json:"admin_user_id"`
	TenantID        string `json:"tenant_id"`
	TenantSlug      string `json:"tenant_slug"`
	TargetUserID    string `json:"target_user_id"`
	ImpersonationID string `json:"impersonation_id"`
	Type            string `json:"type"`
	jwt.RegisteredClaims
}

// TokenService issues and parses admin tokens.
type TokenService struct {
	accessSecret    []byte
	accessTTL       time.Duration
	refreshTTL      time.Duration
	rsaPrivateKey   *rsa.PrivateKey
	rsaPublicKey    *rsa.PublicKey
	impersonateTTL  time.Duration
	issuer          string
}

// NewTokenService builds a TokenService. rsaPrivateKey may be nil — in that
// case impersonation issuance will fail loudly. Loading from disk is done
// by LoadRSAFromPath for production configs.
func NewTokenService(accessSecret string, accessTTL, refreshTTL, impersonateTTL time.Duration) *TokenService {
	return &TokenService{
		accessSecret:   []byte(accessSecret),
		accessTTL:      accessTTL,
		refreshTTL:     refreshTTL,
		impersonateTTL: impersonateTTL,
		issuer:         "admin-auth-service",
	}
}

// WithRSAKeys attaches an RSA keypair for impersonation signing/verify.
// Either can be nil on the verify or sign side depending on caller role.
func (s *TokenService) WithRSAKeys(priv *rsa.PrivateKey, pub *rsa.PublicKey) {
	s.rsaPrivateKey = priv
	s.rsaPublicKey = pub
}

// IssueAccessToken mints an access JWT for the given admin.
func (s *TokenService) IssueAccessToken(userID, email string, roles []string) (string, time.Time, error) {
	return s.issueHS256(userID, email, roles, TypeAccess, s.accessTTL)
}

// IssueRefreshToken mints a refresh JWT for the given admin.
func (s *TokenService) IssueRefreshToken(userID, email string, roles []string) (string, time.Time, error) {
	return s.issueHS256(userID, email, roles, TypeRefresh, s.refreshTTL)
}

func (s *TokenService) issueHS256(userID, email string, roles []string, typ string, ttl time.Duration) (string, time.Time, error) {
	now := time.Now()
	exp := now.Add(ttl)
	claims := AccessClaims{
		UserID: userID,
		Email:  email,
		Roles:  roles,
		Type:   typ,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    s.issuer,
			Subject:   userID,
			Audience:  jwt.ClaimStrings{AudiencePlatform},
			ExpiresAt: jwt.NewNumericDate(exp),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			ID:        randomID(),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString(s.accessSecret)
	if err != nil {
		return "", exp, fmt.Errorf("sign access token: %w", err)
	}
	return signed, exp, nil
}

// ParseAdminToken parses an HS256 admin token and enforces aud=platform.
func (s *TokenService) ParseAdminToken(raw string) (*AccessClaims, error) {
	var claims AccessClaims
	tok, err := jwt.ParseWithClaims(raw, &claims, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return s.accessSecret, nil
	})
	if err != nil {
		return nil, err
	}
	if !tok.Valid {
		return nil, errors.New("token invalid")
	}
	if !claims.HasAudience(AudiencePlatform) {
		return nil, errors.New("token audience is not 'platform'")
	}
	return &claims, nil
}

// HasAudience reports whether aud contains target.
func (c *AccessClaims) HasAudience(target string) bool {
	for _, a := range c.Audience {
		if a == target {
			return true
		}
	}
	return false
}

// IssueImpersonationToken mints a short-lived RS256 impersonation JWT.
// aud is set to "tenant:{slug}" so the tenant user-auth-service can enforce
// scoped validation. Returns (signed, impersonationID, expires, error).
func (s *TokenService) IssueImpersonationToken(adminUserID, tenantID, tenantSlug, targetUserID string) (string, string, time.Time, error) {
	if s.rsaPrivateKey == nil {
		return "", "", time.Time{}, errors.New("platform private key not configured")
	}
	now := time.Now()
	exp := now.Add(s.impersonateTTL)
	impID := randomID()
	claims := ImpersonationClaims{
		AdminUserID:     adminUserID,
		TenantID:        tenantID,
		TenantSlug:      tenantSlug,
		TargetUserID:    targetUserID,
		ImpersonationID: impID,
		Type:            TypeImpersonation,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    s.issuer,
			Subject:   targetUserID,
			Audience:  jwt.ClaimStrings{"tenant:" + tenantSlug},
			ExpiresAt: jwt.NewNumericDate(exp),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			ID:        impID,
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	signed, err := token.SignedString(s.rsaPrivateKey)
	if err != nil {
		return "", "", exp, fmt.Errorf("sign impersonation token: %w", err)
	}
	return signed, impID, exp, nil
}

// VerifyImpersonationToken validates an impersonation token using the public key.
// Primarily used by tests and by tenant-side user-auth-service (W14.4).
func (s *TokenService) VerifyImpersonationToken(raw string) (*ImpersonationClaims, error) {
	if s.rsaPublicKey == nil {
		return nil, errors.New("platform public key not configured")
	}
	var claims ImpersonationClaims
	tok, err := jwt.ParseWithClaims(raw, &claims, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return s.rsaPublicKey, nil
	})
	if err != nil {
		return nil, err
	}
	if !tok.Valid {
		return nil, errors.New("impersonation token invalid")
	}
	if claims.Type != TypeImpersonation {
		return nil, errors.New("not an impersonation token")
	}
	return &claims, nil
}

// LoadRSAPrivateKeyFromPath reads a PKCS#1 or PKCS#8 PEM private key.
func LoadRSAPrivateKeyFromPath(path string) (*rsa.PrivateKey, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read private key: %w", err)
	}
	return ParseRSAPrivateKey(data)
}

// LoadRSAPublicKeyFromPath reads a PKIX PEM public key.
func LoadRSAPublicKeyFromPath(path string) (*rsa.PublicKey, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read public key: %w", err)
	}
	return ParseRSAPublicKey(data)
}

// ParseRSAPrivateKey decodes an RSA private key from PEM bytes.
func ParseRSAPrivateKey(pemBytes []byte) (*rsa.PrivateKey, error) {
	block, _ := pem.Decode(pemBytes)
	if block == nil {
		return nil, errors.New("no PEM block found in private key")
	}
	if k, err := x509.ParsePKCS1PrivateKey(block.Bytes); err == nil {
		return k, nil
	}
	k, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("parse private key: %w", err)
	}
	rsaKey, ok := k.(*rsa.PrivateKey)
	if !ok {
		return nil, errors.New("private key is not RSA")
	}
	return rsaKey, nil
}

// ParseRSAPublicKey decodes an RSA public key from PEM bytes.
func ParseRSAPublicKey(pemBytes []byte) (*rsa.PublicKey, error) {
	block, _ := pem.Decode(pemBytes)
	if block == nil {
		return nil, errors.New("no PEM block found in public key")
	}
	if k, err := x509.ParsePKIXPublicKey(block.Bytes); err == nil {
		if rsaKey, ok := k.(*rsa.PublicKey); ok {
			return rsaKey, nil
		}
	}
	k, err := x509.ParsePKCS1PublicKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("parse public key: %w", err)
	}
	return k, nil
}

func randomID() string {
	var b [16]byte
	_, _ = rand.Read(b[:])
	return hex.EncodeToString(b[:])
}
