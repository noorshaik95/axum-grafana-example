package jwt

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type Claims struct {
	UserID string   `json:"user_id"`
	Email  string   `json:"email"`
	Roles  []string `json:"roles"`
	Type   string   `json:"type"` // "access" or "refresh"
	jwt.RegisteredClaims
}

type TokenService struct {
	secretKey            []byte
	accessTokenDuration  time.Duration
	refreshTokenDuration time.Duration
}

func NewTokenService(secretKey string, accessTokenDuration, refreshTokenDuration int) *TokenService {
	return &TokenService{
		secretKey:            []byte(secretKey),
		accessTokenDuration:  time.Duration(accessTokenDuration) * time.Minute,
		refreshTokenDuration: time.Duration(refreshTokenDuration) * time.Hour,
	}
}

// GenerateAccessToken generates a new access token
func (s *TokenService) GenerateAccessToken(userID, email string, roles []string) (string, int64, error) {
	expiresAt := time.Now().Add(s.accessTokenDuration)
	claims := Claims{
		UserID: userID,
		Email:  email,
		Roles:  roles,
		Type:   "access",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(s.secretKey)
	if err != nil {
		return "", 0, err
	}

	return tokenString, int64(s.accessTokenDuration.Seconds()), nil
}

// GenerateRefreshToken generates a new refresh token
func (s *TokenService) GenerateRefreshToken(userID, email string, roles []string) (string, error) {
	expiresAt := time.Now().Add(s.refreshTokenDuration)
	claims := Claims{
		UserID: userID,
		Email:  email,
		Roles:  roles,
		Type:   "refresh",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(s.secretKey)
	if err != nil {
		return "", err
	}

	return tokenString, nil
}

// ValidateToken validates a token and returns the claims
func (s *TokenService) ValidateToken(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return s.secretKey, nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to parse token: %w", err)
	}

	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		return claims, nil
	}

	return nil, errors.New("invalid token")
}

// ValidateAccessToken validates an access token
func (s *TokenService) ValidateAccessToken(tokenString string) (*Claims, error) {
	claims, err := s.ValidateToken(tokenString)
	if err != nil {
		return nil, err
	}

	if claims.Type != "access" {
		return nil, errors.New("invalid token type: expected access token")
	}

	return claims, nil
}

// ValidateRefreshToken validates a refresh token
func (s *TokenService) ValidateRefreshToken(tokenString string) (*Claims, error) {
	claims, err := s.ValidateToken(tokenString)
	if err != nil {
		return nil, err
	}

	if claims.Type != "refresh" {
		return nil, errors.New("invalid token type: expected refresh token")
	}

	return claims, nil
}

// RefreshAccessToken generates a new access token from a refresh token
func (s *TokenService) RefreshAccessToken(refreshToken string) (string, string, int64, error) {
	claims, err := s.ValidateRefreshToken(refreshToken)
	if err != nil {
		return "", "", 0, err
	}

	// Generate new access token
	accessToken, expiresIn, err := s.GenerateAccessToken(claims.UserID, claims.Email, claims.Roles)
	if err != nil {
		return "", "", 0, err
	}

	// Generate new refresh token
	newRefreshToken, err := s.GenerateRefreshToken(claims.UserID, claims.Email, claims.Roles)
	if err != nil {
		return "", "", 0, err
	}

	return accessToken, newRefreshToken, expiresIn, nil
}
