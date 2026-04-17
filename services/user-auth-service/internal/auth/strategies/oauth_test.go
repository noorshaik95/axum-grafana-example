package strategies

import (
	"context"
	"testing"
	"time"

	"slate/services/user-auth-service/internal/auth"
	"slate/services/user-auth-service/internal/auth/services"
	"slate/services/user-auth-service/internal/config"
	"slate/services/user-auth-service/internal/models"
	"slate/services/user-auth-service/internal/service"
	"slate/services/user-auth-service/pkg/logger"

	"go.opentelemetry.io/otel/trace/noop"
)

// Mock implementations
type mockUserRepository struct {
	users map[string]*models.User
}

func (m *mockUserRepository) Create(ctx context.Context, user *models.User) error {
	m.users[user.ID] = user
	return nil
}

func (m *mockUserRepository) GetByID(ctx context.Context, id string) (*models.User, error) {
	user, exists := m.users[id]
	if !exists {
		return nil, nil
	}
	return user, nil
}

func (m *mockUserRepository) GetByEmail(ctx context.Context, email string) (*models.User, error) {
	for _, user := range m.users {
		if user.Email == email {
			return user, nil
		}
	}
	return nil, nil
}

func (m *mockUserRepository) Update(ctx context.Context, user *models.User) error {
	m.users[user.ID] = user
	return nil
}

func (m *mockUserRepository) Delete(ctx context.Context, id string) error {
	delete(m.users, id)
	return nil
}

func (m *mockUserRepository) List(ctx context.Context, page, pageSize int, search, role string, isActive *bool) ([]*models.User, int, error) {
	return nil, 0, nil
}

func (m *mockUserRepository) UpdatePassword(ctx context.Context, userID, passwordHash string) error {
	return nil
}

type mockOAuthRepository struct {
	providers map[string]*models.OAuthProvider
}

func (m *mockOAuthRepository) CreateOrUpdate(ctx context.Context, provider *models.OAuthProvider) error {
	key := provider.Provider + ":" + provider.ProviderUserID
	m.providers[key] = provider
	return nil
}

func (m *mockOAuthRepository) GetByProviderAndUserID(ctx context.Context, provider, providerUserID string) (*models.OAuthProvider, error) {
	key := provider + ":" + providerUserID
	p, exists := m.providers[key]
	if !exists {
		return nil, nil
	}
	return p, nil
}

func (m *mockOAuthRepository) GetByUserID(ctx context.Context, userID string) ([]*models.OAuthProvider, error) {
	return nil, nil
}

func (m *mockOAuthRepository) Delete(ctx context.Context, id string) error {
	return nil
}

type mockTokenService struct{}

func (m *mockTokenService) GenerateAccessToken(userID, email string, roles []string) (string, int64, error) {
	return "access_token", 3600, nil
}

func (m *mockTokenService) GenerateRefreshToken(userID, email string, roles []string) (string, error) {
	return "refresh_token", nil
}

func (m *mockTokenService) ValidateAccessToken(token string) (*service.TokenClaims, error) {
	return nil, nil
}

func (m *mockTokenService) RefreshAccessToken(refreshToken string) (string, string, int64, error) {
	return "new_access_token", "new_refresh_token", 3600, nil
}

func TestOAuthAuthStrategy_GetType(t *testing.T) {
	strategy := &OAuthAuthStrategy{}

	if strategy.GetType() != auth.AuthTypeOAuth {
		t.Errorf("Expected AuthTypeOAuth, got %s", strategy.GetType())
	}
}

func TestOAuthAuthStrategy_ValidateConfig_Success(t *testing.T) {
	config := &config.OAuthConfig{
		Providers: map[string]config.OAuthProviderConfig{
			"google": {
				Type:         "google",
				ClientID:     "test-client-id",
				ClientSecret: "test-secret",
			},
		},
	}

	strategy := &OAuthAuthStrategy{
		config:      config,
		userRepo:    &mockUserRepository{users: make(map[string]*models.User)},
		oauthRepo:   &mockOAuthRepository{providers: make(map[string]*models.OAuthProvider)},
		userService: &service.UserService{}, // Needs to be non-nil for validation
		tokenSvc:    &mockTokenService{},
		sessionMgr:  &services.SessionManager{},
		tracer:      noop.NewTracerProvider().Tracer("test"),
		logger:      logger.NewLogger("debug"),
	}

	err := strategy.ValidateConfig()
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
}

func TestOAuthAuthStrategy_ValidateConfig_NoProviders(t *testing.T) {
	config := &config.OAuthConfig{
		Providers: map[string]config.OAuthProviderConfig{},
	}

	strategy := &OAuthAuthStrategy{
		config:      config,
		userRepo:    &mockUserRepository{users: make(map[string]*models.User)},
		oauthRepo:   &mockOAuthRepository{providers: make(map[string]*models.OAuthProvider)},
		userService: nil,
		tokenSvc:    &mockTokenService{},
		sessionMgr:  &services.SessionManager{},
		tracer:      noop.NewTracerProvider().Tracer("test"),
		logger:      logger.NewLogger("debug"),
	}

	err := strategy.ValidateConfig()
	if err == nil {
		t.Error("Expected error for no providers, got nil")
	}
}

func TestOAuthAuthStrategy_ValidateConfig_MissingDependency(t *testing.T) {
	config := &config.OAuthConfig{
		Providers: map[string]config.OAuthProviderConfig{
			"google": {},
		},
	}

	strategy := &OAuthAuthStrategy{
		config:      config,
		userRepo:    nil, // Missing dependency
		oauthRepo:   &mockOAuthRepository{providers: make(map[string]*models.OAuthProvider)},
		userService: nil,
		tokenSvc:    &mockTokenService{},
		sessionMgr:  &services.SessionManager{},
		tracer:      noop.NewTracerProvider().Tracer("test"),
		logger:      logger.NewLogger("debug"),
	}

	err := strategy.ValidateConfig()
	if err == nil {
		t.Error("Expected error for missing dependency, got nil")
	}
}

func TestOAuthAuthStrategy_GenerateState(t *testing.T) {
	strategy := &OAuthAuthStrategy{
		stateStore: make(map[string]time.Time),
		logger:     logger.NewLogger("debug"),
	}

	state1 := strategy.generateState()
	state2 := strategy.generateState()

	if state1 == "" {
		t.Error("Expected non-empty state")
	}

	if state1 == state2 {
		t.Error("Expected different states, got same")
	}

	// Check that states are stored
	if _, exists := strategy.stateStore[state1]; !exists {
		t.Error("State1 not stored")
	}
	if _, exists := strategy.stateStore[state2]; !exists {
		t.Error("State2 not stored")
	}
}

func TestOAuthAuthStrategy_ValidateState_Success(t *testing.T) {
	strategy := &OAuthAuthStrategy{
		stateStore: make(map[string]time.Time),
		logger:     logger.NewLogger("debug"),
	}

	state := strategy.generateState()

	err := strategy.validateState(state)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	// State should be removed after validation
	if _, exists := strategy.stateStore[state]; exists {
		t.Error("State should be removed after validation")
	}
}

func TestOAuthAuthStrategy_ValidateState_Expired(t *testing.T) {
	strategy := &OAuthAuthStrategy{
		stateStore: make(map[string]time.Time),
		logger:     logger.NewLogger("debug"),
	}

	state := "expired-state"
	strategy.stateStore[state] = time.Now().Add(-11 * time.Minute)

	err := strategy.validateState(state)
	if err == nil {
		t.Error("Expected error for expired state, got nil")
	}
}

func TestOAuthAuthStrategy_ValidateState_Invalid(t *testing.T) {
	strategy := &OAuthAuthStrategy{
		stateStore: make(map[string]time.Time),
		logger:     logger.NewLogger("debug"),
	}

	err := strategy.validateState("invalid-state")
	if err == nil {
		t.Error("Expected error for invalid state, got nil")
	}
}

func TestOAuthAuthStrategy_ValidateState_OneTimeUse(t *testing.T) {
	strategy := &OAuthAuthStrategy{
		stateStore: make(map[string]time.Time),
		logger:     logger.NewLogger("debug"),
	}

	state := strategy.generateState()

	// First validation should succeed
	err := strategy.validateState(state)
	if err != nil {
		t.Errorf("Expected no error on first validation, got %v", err)
	}

	// Second validation should fail (one-time use)
	err = strategy.validateState(state)
	if err == nil {
		t.Error("Expected error on second validation (one-time use), got nil")
	}
}

// Tests for Authenticate method
func TestOAuthAuthStrategy_Authenticate_Success(t *testing.T) {
	config := &config.OAuthConfig{
		Providers: map[string]config.OAuthProviderConfig{
			"google": {
				Type:         "google",
				ClientID:     "test-client-id",
				ClientSecret: "test-secret",
				RedirectURI:  "http://localhost/callback",
				Scopes:       []string{"openid", "profile", "email"},
			},
		},
	}

	strategy := NewOAuthAuthStrategy(
		config,
		&mockUserRepository{users: make(map[string]*models.User)},
		&mockOAuthRepository{providers: make(map[string]*models.OAuthProvider)},
		&service.UserService{},
		&mockTokenService{},
		&services.SessionManager{},
		noop.NewTracerProvider().Tracer("test"),
		logger.NewLogger("debug"),
		"test",
	)

	req := &auth.AuthRequest{
		Provider: "google",
	}

	result, err := strategy.Authenticate(context.Background(), req)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if result.Success {
		t.Error("Expected Success=false for OAuth initiation")
	}

	if result.AuthorizationURL == "" {
		t.Error("Expected authorization URL to be set")
	}

	if result.State == "" {
		t.Error("Expected state to be set")
	}
}

func TestOAuthAuthStrategy_Authenticate_MissingProvider(t *testing.T) {
	config := &config.OAuthConfig{
		Providers: map[string]config.OAuthProviderConfig{
			"google": {},
		},
	}

	strategy := NewOAuthAuthStrategy(
		config,
		&mockUserRepository{users: make(map[string]*models.User)},
		&mockOAuthRepository{providers: make(map[string]*models.OAuthProvider)},
		&service.UserService{},
		&mockTokenService{},
		&services.SessionManager{},
		noop.NewTracerProvider().Tracer("test"),
		logger.NewLogger("debug"),
		"test",
	)

	req := &auth.AuthRequest{
		Provider: "", // Missing provider
	}

	_, err := strategy.Authenticate(context.Background(), req)
	if err == nil {
		t.Error("Expected error for missing provider, got nil")
	}
}

func TestOAuthAuthStrategy_Authenticate_UnconfiguredProvider(t *testing.T) {
	config := &config.OAuthConfig{
		Providers: map[string]config.OAuthProviderConfig{
			"google": {},
		},
	}

	strategy := NewOAuthAuthStrategy(
		config,
		&mockUserRepository{users: make(map[string]*models.User)},
		&mockOAuthRepository{providers: make(map[string]*models.OAuthProvider)},
		&service.UserService{},
		&mockTokenService{},
		&services.SessionManager{},
		noop.NewTracerProvider().Tracer("test"),
		logger.NewLogger("debug"),
		"test",
	)

	req := &auth.AuthRequest{
		Provider: "github", // Not configured
	}

	_, err := strategy.Authenticate(context.Background(), req)
	if err == nil {
		t.Error("Expected error for unconfigured provider, got nil")
	}
}

func TestOAuthAuthStrategy_Authenticate_Microsoft(t *testing.T) {
	config := &config.OAuthConfig{
		Providers: map[string]config.OAuthProviderConfig{
			"microsoft": {
				Type:         "microsoft",
				ClientID:     "test-client-id",
				ClientSecret: "test-secret",
				RedirectURI:  "http://localhost/callback",
				Scopes:       []string{"openid", "profile", "email"},
			},
		},
	}

	strategy := NewOAuthAuthStrategy(
		config,
		&mockUserRepository{users: make(map[string]*models.User)},
		&mockOAuthRepository{providers: make(map[string]*models.OAuthProvider)},
		&service.UserService{},
		&mockTokenService{},
		&services.SessionManager{},
		noop.NewTracerProvider().Tracer("test"),
		logger.NewLogger("debug"),
		"test",
	)

	req := &auth.AuthRequest{
		Provider: "microsoft",
	}

	result, err := strategy.Authenticate(context.Background(), req)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if result.Success {
		t.Error("Expected Success=false for OAuth initiation")
	}

	if result.AuthorizationURL == "" {
		t.Error("Expected authorization URL to be set")
	}
}

func TestOAuthAuthStrategy_Authenticate_Custom(t *testing.T) {
	config := &config.OAuthConfig{
		Providers: map[string]config.OAuthProviderConfig{
			"custom": {
				Type:         "custom",
				ClientID:     "test-client-id",
				ClientSecret: "test-secret",
				RedirectURI:  "http://localhost/callback",
				AuthURL:      "https://custom.example.com/oauth/authorize",
				TokenURL:     "https://custom.example.com/oauth/token",
				UserInfoURL:  "https://custom.example.com/oauth/userinfo",
				Scopes:       []string{"openid", "profile", "email"},
			},
		},
	}

	strategy := NewOAuthAuthStrategy(
		config,
		&mockUserRepository{users: make(map[string]*models.User)},
		&mockOAuthRepository{providers: make(map[string]*models.OAuthProvider)},
		&service.UserService{},
		&mockTokenService{},
		&services.SessionManager{},
		noop.NewTracerProvider().Tracer("test"),
		logger.NewLogger("debug"),
		"test",
	)

	req := &auth.AuthRequest{
		Provider: "custom",
	}

	result, err := strategy.Authenticate(context.Background(), req)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if result.Success {
		t.Error("Expected Success=false for OAuth initiation")
	}

	if result.AuthorizationURL == "" {
		t.Error("Expected authorization URL to be set")
	}
}

// Tests for HandleCallback method
func TestOAuthAuthStrategy_HandleCallback_InvalidState(t *testing.T) {
	config := &config.OAuthConfig{
		Providers: map[string]config.OAuthProviderConfig{
			"google": {
				Type:         "google",
				ClientID:     "test-client-id",
				ClientSecret: "test-secret",
			},
		},
	}

	strategy := NewOAuthAuthStrategy(
		config,
		&mockUserRepository{users: make(map[string]*models.User)},
		&mockOAuthRepository{providers: make(map[string]*models.OAuthProvider)},
		&service.UserService{},
		&mockTokenService{},
		&services.SessionManager{},
		noop.NewTracerProvider().Tracer("test"),
		logger.NewLogger("debug"),
		"test",
	)

	req := &auth.CallbackRequest{
		Code:  "auth-code",
		State: "invalid-state",
	}

	_, err := strategy.HandleCallback(context.Background(), req)
	if err == nil {
		t.Error("Expected error for invalid state, got nil")
	}
}

func TestOAuthAuthStrategy_HandleCallback_MissingCode(t *testing.T) {
	config := &config.OAuthConfig{
		Providers: map[string]config.OAuthProviderConfig{
			"google": {},
		},
	}

	strategy := NewOAuthAuthStrategy(
		config,
		&mockUserRepository{users: make(map[string]*models.User)},
		&mockOAuthRepository{providers: make(map[string]*models.OAuthProvider)},
		&service.UserService{},
		&mockTokenService{},
		&services.SessionManager{},
		noop.NewTracerProvider().Tracer("test"),
		logger.NewLogger("debug"),
		"test",
	)

	req := &auth.CallbackRequest{
		Code:  "", // Missing code
		State: "some-state",
	}

	_, err := strategy.HandleCallback(context.Background(), req)
	if err == nil {
		t.Error("Expected error for missing code, got nil")
	}
}

func TestOAuthAuthStrategy_HandleCallback_MissingState(t *testing.T) {
	config := &config.OAuthConfig{
		Providers: map[string]config.OAuthProviderConfig{
			"google": {},
		},
	}

	strategy := NewOAuthAuthStrategy(
		config,
		&mockUserRepository{users: make(map[string]*models.User)},
		&mockOAuthRepository{providers: make(map[string]*models.OAuthProvider)},
		&service.UserService{},
		&mockTokenService{},
		&services.SessionManager{},
		noop.NewTracerProvider().Tracer("test"),
		logger.NewLogger("debug"),
		"test",
	)

	req := &auth.CallbackRequest{
		Code:  "auth-code",
		State: "", // Missing state
	}

	_, err := strategy.HandleCallback(context.Background(), req)
	if err == nil {
		t.Error("Expected error for missing state, got nil")
	}
}

// Note: Full integration tests for HandleCallback would require mocking HTTP responses
// from OAuth providers, which is beyond the scope of these unit tests.
// The core validation logic is tested above, and the token exchange/user creation
// logic would be tested in integration tests with actual HTTP mocking.
