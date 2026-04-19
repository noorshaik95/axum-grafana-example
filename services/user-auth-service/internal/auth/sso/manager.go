package sso

import (
	"context"
	"fmt"
	"os"
	"sync"
	"time"

	"slate/services/user-auth-service/internal/models"
	"slate/services/user-auth-service/internal/repository"
	"slate/services/user-auth-service/internal/service"

	"golang.org/x/crypto/bcrypt"
)

// UserProvisioner upserts users seen from an SSO IdP. Split behind an
// interface so tests can avoid a real database.
type UserProvisioner interface {
	UpsertFromSSO(ctx context.Context, id *Identity, kind ProviderKind) (*models.User, error)
}

// TokenIssuer issues JWT access/refresh tokens. Provided so the SSO callback
// can return a JWT without depending on the top-level UserService.
type TokenIssuer interface {
	GenerateAccessToken(userID, email string, roles []string) (string, int64, error)
	GenerateRefreshToken(userID, email string, roles []string) (string, error)
}

// Manager routes incoming SSO requests to the right Strategy. In practice each
// tenant container provisions a single Config so Manager resolves by Kind.
type Manager struct {
	mu          sync.RWMutex
	cfg         *Config
	strategies  map[ProviderKind]Strategy
	client      IdPClient
	auditRepo   repository.AuditRepository
	userRepo    service.UserRepositoryInterface
	roleRepo    service.RoleRepositoryInterface
	tokens      TokenIssuer
}

// ManagerOptions bundles the wiring pieces Manager needs. Fields are allowed
// to be nil if a given feature is not exercised (e.g. auditRepo nil in tests
// that only verify strategy mechanics).
type ManagerOptions struct {
	Config      *Config
	Client      IdPClient
	AuditRepo   repository.AuditRepository
	UserRepo    service.UserRepositoryInterface
	RoleRepo    service.RoleRepositoryInterface
	TokenIssuer TokenIssuer
}

// NewManager constructs and validates the SSO Manager. Strategies are
// registered lazily based on cfg.Kind; ErrUnsupported is returned for unknown
// kinds.
func NewManager(opts ManagerOptions) (*Manager, error) {
	if opts.Config == nil {
		return nil, fmt.Errorf("sso: config is required")
	}
	if opts.Client == nil {
		return nil, fmt.Errorf("sso: idp client is required")
	}
	m := &Manager{
		cfg:        opts.Config,
		client:     opts.Client,
		strategies: make(map[ProviderKind]Strategy),
		auditRepo:  opts.AuditRepo,
		userRepo:   opts.UserRepo,
		roleRepo:   opts.RoleRepo,
		tokens:     opts.TokenIssuer,
	}
	switch opts.Config.Kind {
	case ProviderSAML:
		s, err := NewSAMLStrategy(opts.Config, opts.Client)
		if err != nil {
			return nil, err
		}
		m.strategies[ProviderSAML] = s
	case ProviderOIDC:
		s, err := NewOIDCStrategy(opts.Config, opts.Client)
		if err != nil {
			return nil, err
		}
		m.strategies[ProviderOIDC] = s
	case ProviderGoogle:
		s, err := NewGoogleStrategy(opts.Config, opts.Client)
		if err != nil {
			return nil, err
		}
		m.strategies[ProviderGoogle] = s
	default:
		return nil, fmt.Errorf("sso: unsupported kind %q", opts.Config.Kind)
	}
	return m, nil
}

// NewManagerFromEnv reads TENANT_SSO_CONFIG and builds a Manager. Meant for
// cmd/server bootstrap; returns an error if the env is unset or invalid.
func NewManagerFromEnv(client IdPClient, auditRepo repository.AuditRepository, userRepo service.UserRepositoryInterface, roleRepo service.RoleRepositoryInterface, tokens TokenIssuer) (*Manager, error) {
	raw := os.Getenv("TENANT_SSO_CONFIG")
	cfg, err := ParseConfig(raw)
	if err != nil {
		return nil, err
	}
	return NewManager(ManagerOptions{
		Config:      cfg,
		Client:      client,
		AuditRepo:   auditRepo,
		UserRepo:    userRepo,
		RoleRepo:    roleRepo,
		TokenIssuer: tokens,
	})
}

// Kind returns the configured provider kind.
func (m *Manager) Kind() ProviderKind { return m.cfg.Kind }

// Strategy returns the Strategy for the currently configured provider.
func (m *Manager) Strategy() Strategy {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.strategies[m.cfg.Kind]
}

// Initiate starts an SSO login, returning the IdP URL + state.
func (m *Manager) Initiate(ctx context.Context, redirectBack string) (string, string, error) {
	return m.Strategy().Initiate(ctx, redirectBack)
}

// CallbackResult is what the HTTP callback handler returns to the caller.
type CallbackResult struct {
	User         *models.User
	AccessToken  string
	RefreshToken string
	ExpiresIn    int64
}

// Callback runs the SSO callback end-to-end: validate state, exchange code /
// SAML response, upsert user, assign default role, audit, issue JWT.
func (m *Manager) Callback(ctx context.Context, p CallbackParams, clientIP string) (*CallbackResult, error) {
	id, err := m.Strategy().Callback(ctx, p)
	if err != nil {
		return nil, err
	}
	if id.Email == "" {
		return nil, fmt.Errorf("sso: identity missing email")
	}
	user, err := m.upsert(ctx, id)
	if err != nil {
		return nil, err
	}
	result := &CallbackResult{User: user}
	if m.tokens != nil {
		access, expires, err := m.tokens.GenerateAccessToken(user.ID, user.Email, user.Roles)
		if err != nil {
			return nil, fmt.Errorf("sso: access token: %w", err)
		}
		refresh, err := m.tokens.GenerateRefreshToken(user.ID, user.Email, user.Roles)
		if err != nil {
			return nil, fmt.Errorf("sso: refresh token: %w", err)
		}
		result.AccessToken = access
		result.RefreshToken = refresh
		result.ExpiresIn = expires
	}
	if m.auditRepo != nil {
		_ = m.auditRepo.Append(ctx, &models.AuditEvent{
			ActorID:   user.ID,
			ActorType: models.AuditActorUser,
			Action:    models.AuditActionSSOLogin,
			TargetID:  user.ID,
			Metadata: map[string]interface{}{
				"kind":        string(m.cfg.Kind),
				"tenant_slug": m.cfg.TenantSlug,
			},
			IPAddress: clientIP,
			CreatedAt: time.Now().UTC(),
		})
	}
	return result, nil
}

// TestConnection probes the configured IdP.
func (m *Manager) TestConnection(ctx context.Context) error {
	return m.Strategy().TestConnection(ctx)
}

func (m *Manager) upsert(ctx context.Context, id *Identity) (*models.User, error) {
	if m.userRepo == nil {
		// Test-only path: return a synthesized user so strategy mechanics are
		// exercisable without a repo.
		return &models.User{
			ID:        "sso-" + id.Email,
			Email:     id.Email,
			FirstName: id.FirstName,
			LastName:  id.LastName,
			IsActive:  true,
			Roles:     id.Roles,
		}, nil
	}
	user, err := m.userRepo.GetByEmail(ctx, id.Email)
	if err != nil || user == nil {
		placeholder, hashErr := bcrypt.GenerateFromPassword([]byte(fmt.Sprintf("sso-%d", time.Now().UnixNano())), bcrypt.DefaultCost)
		if hashErr != nil {
			return nil, fmt.Errorf("sso: hash placeholder password: %w", hashErr)
		}
		user = models.NewUser(id.Email, string(placeholder), id.FirstName, id.LastName, "")
		if err := m.userRepo.Create(ctx, user); err != nil {
			return nil, fmt.Errorf("sso: create user: %w", err)
		}
	} else {
		changed := false
		if id.FirstName != "" && user.FirstName != id.FirstName {
			user.FirstName = id.FirstName
			changed = true
		}
		if id.LastName != "" && user.LastName != id.LastName {
			user.LastName = id.LastName
			changed = true
		}
		if changed {
			user.UpdatedAt = time.Now().UTC()
			_ = m.userRepo.Update(ctx, user)
		}
	}
	roles := id.Roles
	if len(roles) == 0 && m.cfg.DefaultRole != "" {
		roles = []string{m.cfg.DefaultRole}
	}
	if m.roleRepo != nil {
		for _, r := range roles {
			_ = m.roleRepo.AssignRoleByName(ctx, user.ID, r)
		}
	}
	if len(user.Roles) == 0 {
		user.Roles = roles
	}
	return user, nil
}
