package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"slate/libs/common-go/tracing"
	"slate/services/user-auth-service/internal/models"
	"slate/services/user-auth-service/internal/repository"
	"slate/services/user-auth-service/pkg/logger"
	"slate/services/user-auth-service/pkg/validation"

	"go.opentelemetry.io/otel/attribute"
	"golang.org/x/crypto/bcrypt"
)

type UserService struct {
	userRepo        UserRepositoryInterface
	roleRepo        RoleRepositoryInterface
	tokenSvc        TokenServiceInterface
	tokenBlacklist  TokenBlacklistInterface
	validator       *validation.Validator
	logger          *logger.Logger
	metrics         MetricsInterface
	strategyManager StrategyManagerInterface // Optional: for OAuth/SAML support
}

func NewUserService(userRepo UserRepositoryInterface, roleRepo RoleRepositoryInterface, tokenSvc TokenServiceInterface, tokenBlacklist TokenBlacklistInterface, log *logger.Logger, metrics MetricsInterface, strategyManager StrategyManagerInterface) *UserService {
	return &UserService{
		userRepo:        userRepo,
		roleRepo:        roleRepo,
		tokenSvc:        tokenSvc,
		tokenBlacklist:  tokenBlacklist,
		validator:       validation.NewValidator(),
		logger:          log,
		metrics:         metrics,
		strategyManager: strategyManager,
	}
}

// Register registers a new user
func (s *UserService) Register(ctx context.Context, email, password, firstName, lastName, phone string) (user *models.User, tokens *models.TokenPair, err error) {
	ctx, span := tracing.StartSpan(ctx, "user_service.register",
		attribute.String("email", email))
	defer tracing.EndSpanWithError(span, &err)

	start := time.Now()
	var success bool
	defer func() {
		s.metrics.IncrementRegistrations(success)
		s.metrics.ObserveRequestDuration("register", time.Since(start).Seconds())
	}()

	// Validate email
	if err := s.validator.ValidateEmail(email); err != nil {
		return nil, nil, err
	}

	// Validate password
	if err := s.validator.ValidatePassword(password); err != nil {
		return nil, nil, err
	}

	// Sanitize and validate first name
	sanitizedFirstName, err := s.validator.SanitizeName(firstName)
	if err != nil {
		return nil, nil, err
	}

	// Sanitize and validate last name
	sanitizedLastName, err := s.validator.SanitizeName(lastName)
	if err != nil {
		return nil, nil, err
	}

	// Validate phone (optional field)
	if err := s.validator.ValidatePhone(phone); err != nil {
		return nil, nil, err
	}

	// Check if user already exists
	existingUser, _ := s.userRepo.GetByEmail(ctx, email)
	if existingUser != nil {
		return nil, nil, fmt.Errorf("user with email %s already exists", email)
	}

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to hash password: %w", err)
	}

	// Create user with sanitized names. Derive a unique username from the
	// email local-part (same convention migration 010 used for backfill:
	// lowercase, numeric suffix -N on collision). Admin can rename later
	// via the user API.
	user = models.NewUser(email, string(hashedPassword), sanitizedFirstName, sanitizedLastName, phone)
	base := deriveUsernameBase(email)
	for attempt := 0; attempt < 16; attempt++ {
		user.Username = base
		if attempt > 0 {
			user.Username = fmt.Sprintf("%s-%d", base, attempt+1)
		}
		if err = s.userRepo.Create(ctx, user); err == nil {
			break
		}
		if errors.Is(err, repository.ErrUsernameTaken) {
			continue
		}
		return nil, nil, err
	}
	if err != nil {
		return nil, nil, fmt.Errorf("unable to allocate username after retries: %w", err)
	}

	// Assign default "user" role
	if err := s.roleRepo.AssignRoleByName(ctx, user.ID, "user"); err != nil {
		// Log error but don't fail registration
		s.logger.WarnWithContext(ctx).
			Str("user_id", user.ID).
			Str("operation", "register").
			Str("error_type", "role_assignment_failed").
			Err(err).
			Msg("failed to assign default role")
	}

	// Reload user to get roles
	user, err = s.userRepo.GetByID(ctx, user.ID)
	if err != nil {
		s.logger.ErrorWithContext(ctx).
			Str("user_id", user.ID).
			Str("operation", "register").
			Str("error_type", "user_reload_failed").
			Err(err).
			Msg("failed to reload user after registration")
		return nil, nil, err
	}

	// Generate tokens
	tokens, err = s.generateTokens(user)
	if err != nil {
		s.logger.ErrorWithContext(ctx).
			Str("user_id", user.ID).
			Str("operation", "register").
			Str("error_type", "token_generation_failed").
			Err(err).
			Msg("failed to generate tokens")
		return nil, nil, err
	}

	s.logger.WithContext(ctx).
		Str("user_id", user.ID).
		Str("email", s.logger.RedactEmail(user.Email)).
		Str("operation", "register").
		Msg("user registered successfully")

	success = true
	return user, tokens, nil
}

// Login authenticates a user using normal (username/password) authentication.
// If OAuth or SAML authentication is configured, this method will return an error
// directing users to use the appropriate authentication method.
func (s *UserService) Login(ctx context.Context, email, password string) (user *models.User, tokens *models.TokenPair, err error) {
	ctx, span := tracing.StartSpan(ctx, "user_service.login",
		attribute.String("email", email))
	defer tracing.EndSpanWithError(span, &err)

	start := time.Now()
	var success bool
	defer func() {
		s.metrics.IncrementLogins(success)
		s.metrics.ObserveRequestDuration("login", time.Since(start).Seconds())
	}()

	// Check if OAuth/SAML authentication is configured
	if s.strategyManager != nil {
		activeAuthType := s.strategyManager.GetActiveAuthType()
		if activeAuthType == AuthTypeOAuth || activeAuthType == AuthTypeSAML {
			s.logger.WarnWithContext(ctx).
				Str("auth_type", string(activeAuthType)).
				Str("email", s.logger.RedactEmail(email)).
				Str("operation", "login").
				Msg("attempted normal login when OAuth/SAML is configured")
			return nil, nil, fmt.Errorf("this organization uses %s authentication. Please use the appropriate login method", activeAuthType)
		}
	}

	// Validate credentials
	user, err = s.validateCredentials(ctx, email, password)
	if err != nil {
		return nil, nil, fmt.Errorf("credential validation failed: %w", err)
	}

	// Generate tokens
	tokens, err = s.generateTokensWithSpan(ctx, user)
	if err != nil {
		return nil, nil, fmt.Errorf("token generation failed: %w", err)
	}

	s.logger.WithContext(ctx).
		Str("user_id", user.ID).
		Str("email", s.logger.RedactEmail(user.Email)).
		Str("operation", "login").
		Msg("user logged in successfully")

	success = true
	return user, tokens, nil
}

// validateCredentials validates user credentials and returns the user if valid
func (s *UserService) validateCredentials(ctx context.Context, email, password string) (user *models.User, err error) {
	ctx, span := tracing.StartSpan(ctx, "validate_credentials")
	defer tracing.EndSpanWithError(span, &err)

	user, err = s.userRepo.GetByEmail(ctx, email)
	if err != nil {
		// Security: Return generic "invalid credentials" error to prevent user enumeration.
		// This prevents attackers from determining which email addresses are registered.
		// Detailed error is logged server-side for debugging but not exposed to client.
		s.logger.WarnWithContext(ctx).
			Str("email", s.logger.RedactEmail(email)).
			Str("operation", "login").
			Str("error_type", "user_not_found").
			Msg("login attempt for non-existent user")
		return nil, fmt.Errorf("invalid credentials")
	}

	if !user.IsActive {
		s.logger.WarnWithContext(ctx).
			Str("user_id", user.ID).
			Str("email", s.logger.RedactEmail(email)).
			Str("operation", "login").
			Str("error_type", "inactive_account").
			Msg("login attempt for inactive account")
		return nil, fmt.Errorf("user account is inactive")
	}

	// Security: Use bcrypt.CompareHashAndPassword for timing-safe password comparison.
	// This prevents timing attacks where an attacker could determine password correctness
	// by measuring response time differences. bcrypt's comparison is constant-time.
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		// Security: Return the same generic "invalid credentials" error as user-not-found case.
		// This prevents attackers from distinguishing between "user doesn't exist" and "wrong password",
		// which would allow user enumeration attacks.
		s.logger.WarnWithContext(ctx).
			Str("user_id", user.ID).
			Str("email", s.logger.RedactEmail(email)).
			Str("operation", "login").
			Str("error_type", "invalid_password").
			Str("password_redacted", s.logger.RedactPassword(password)).
			Msg("login attempt with invalid password")
		return nil, fmt.Errorf("invalid credentials")
	}

	return user, nil
}

// generateTokensWithSpan generates access and refresh tokens for a user with tracing
func (s *UserService) generateTokensWithSpan(ctx context.Context, user *models.User) (tokens *models.TokenPair, err error) {
	ctx, span := tracing.StartSpan(ctx, "generate_tokens",
		attribute.String("user_id", user.ID))
	defer tracing.EndSpanWithError(span, &err)

	return s.generateTokens(user)
}

// Logout invalidates a user's access token by adding it to the blacklist
func (s *UserService) Logout(ctx context.Context, token string) (err error) {
	ctx, span := tracing.StartSpan(ctx, "user_service.logout")
	defer tracing.EndSpanWithError(span, &err)

	// Validate the token to get claims (including expiration)
	claims, err := s.tokenSvc.ValidateAccessToken(token)
	if err != nil {
		// Token is already invalid, no need to blacklist
		s.logger.WarnWithContext(ctx).
			Str("operation", "logout").
			Str("error_type", "invalid_token").
			Str("token_redacted", s.logger.RedactToken(token)).
			Err(err).
			Msg("logout attempt with invalid token")
		return nil
	}

	// Add token to blacklist with TTL matching token expiration
	if s.tokenBlacklist != nil {
		expiresAt := claims.ExpiresAt.Time
		err = s.tokenBlacklist.BlacklistToken(ctx, token, expiresAt)
		if err != nil {
			// Log error but don't fail logout (fail-open for logout)
			s.logger.ErrorWithContext(ctx).
				Str("user_id", claims.UserID).
				Str("operation", "logout").
				Str("error_type", "blacklist_failed").
				Err(err).
				Msg("failed to blacklist token, but allowing logout to proceed")
		} else {
			s.logger.WithContext(ctx).
				Str("user_id", claims.UserID).
				Str("operation", "logout").
				Msg("token blacklisted successfully")
		}
	}

	s.logger.WithContext(ctx).
		Str("user_id", claims.UserID).
		Str("operation", "logout").
		Msg("user logged out successfully")

	return nil
}

// ValidateToken validates a token and returns user info
func (s *UserService) ValidateToken(ctx context.Context, token string) (userID string, roles []string, err error) {
	ctx, span := tracing.StartSpan(ctx, "user_service.validate_token")
	defer tracing.EndSpanWithError(span, &err)

	claims, err := s.tokenSvc.ValidateAccessToken(token)
	if err != nil {
		return "", nil, err
	}

	// Check if token is blacklisted
	if s.tokenBlacklist != nil {
		isBlacklisted, err := s.tokenBlacklist.IsTokenBlacklisted(ctx, token, claims.UserID, claims.IssuedAt.Time)
		if err != nil {
			s.logger.ErrorWithContext(ctx).
				Str("user_id", claims.UserID).
				Str("operation", "validate_token").
				Str("error_type", "blacklist_check_failed").
				Err(err).
				Msg("failed to check token blacklist")
			// Fail-secure: if we can't check blacklist, reject the token
			return "", nil, fmt.Errorf("token revoked")
		}

		if isBlacklisted {
			s.logger.WarnWithContext(ctx).
				Str("user_id", claims.UserID).
				Str("operation", "validate_token").
				Str("error_type", "token_blacklisted").
				Str("token_redacted", s.logger.RedactToken(token)).
				Msg("attempt to use blacklisted token")
			return "", nil, fmt.Errorf("token revoked")
		}
	}

	return claims.UserID, claims.Roles, nil
}

// RefreshToken refreshes an access token
func (s *UserService) RefreshToken(ctx context.Context, refreshToken string) (tokens *models.TokenPair, err error) {
	ctx, span := tracing.StartSpan(ctx, "user_service.refresh_token")
	defer tracing.EndSpanWithError(span, &err)

	accessToken, newRefreshToken, expiresIn, err := s.tokenSvc.RefreshAccessToken(refreshToken)
	if err != nil {
		return nil, err
	}

	return &models.TokenPair{
		AccessToken:  accessToken,
		RefreshToken: newRefreshToken,
		ExpiresIn:    expiresIn,
	}, nil
}

// CreateUser creates a new user (admin operation)
func (s *UserService) CreateUser(ctx context.Context, email, password, firstName, lastName, phone string, roles []string) (*models.User, error) {
	// Validate email
	if err := s.validator.ValidateEmail(email); err != nil {
		return nil, err
	}

	// Validate password
	if err := s.validator.ValidatePassword(password); err != nil {
		return nil, err
	}

	// Sanitize and validate first name
	sanitizedFirstName, err := s.validator.SanitizeName(firstName)
	if err != nil {
		return nil, err
	}

	// Sanitize and validate last name
	sanitizedLastName, err := s.validator.SanitizeName(lastName)
	if err != nil {
		return nil, err
	}

	// Validate phone (optional field)
	if err := s.validator.ValidatePhone(phone); err != nil {
		return nil, err
	}

	// Check if user already exists
	existingUser, _ := s.userRepo.GetByEmail(ctx, email)
	if existingUser != nil {
		return nil, fmt.Errorf("user with email %s already exists", email)
	}

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	// Create user with sanitized names. Same username-derivation retry loop
	// as Register so admin-created users get a unique @handle too.
	user := models.NewUser(email, string(hashedPassword), sanitizedFirstName, sanitizedLastName, phone)
	base := deriveUsernameBase(email)
	var createErr error
	for attempt := 0; attempt < 16; attempt++ {
		user.Username = base
		if attempt > 0 {
			user.Username = fmt.Sprintf("%s-%d", base, attempt+1)
		}
		if createErr = s.userRepo.Create(ctx, user); createErr == nil {
			break
		}
		if errors.Is(createErr, repository.ErrUsernameTaken) {
			continue
		}
		return nil, createErr
	}
	if createErr != nil {
		return nil, fmt.Errorf("unable to allocate username after retries: %w", createErr)
	}

	// Assign roles
	for _, role := range roles {
		if err := s.roleRepo.AssignRoleByName(ctx, user.ID, role); err != nil {
			s.logger.Warn().
				Str("user_id", user.ID).
				Str("role", role).
				Str("operation", "create_user").
				Str("error_type", "role_assignment_failed").
				Err(err).
				Msg("failed to assign role")
		}
	}

	// Reload user to get roles
	return s.userRepo.GetByID(ctx, user.ID)
}

// GetUser retrieves a user by ID
func (s *UserService) GetUser(ctx context.Context, userID string) (*models.User, error) {
	return s.userRepo.GetByID(ctx, userID)
}

// UpdateUser updates a user
func (s *UserService) UpdateUser(ctx context.Context, userID string, email, firstName, lastName, phone *string, isActive *bool) (*models.User, error) {
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	// Update fields if provided with validation
	if email != nil {
		if err := s.validator.ValidateEmail(*email); err != nil {
			return nil, err
		}
		user.Email = *email
	}
	if firstName != nil {
		sanitizedFirstName, err := s.validator.SanitizeName(*firstName)
		if err != nil {
			return nil, err
		}
		user.FirstName = sanitizedFirstName
	}
	if lastName != nil {
		sanitizedLastName, err := s.validator.SanitizeName(*lastName)
		if err != nil {
			return nil, err
		}
		user.LastName = sanitizedLastName
	}
	if phone != nil {
		if err := s.validator.ValidatePhone(*phone); err != nil {
			return nil, err
		}
		user.Phone = *phone
	}
	if isActive != nil {
		user.IsActive = *isActive
	}

	user.UpdatedAt = time.Now()

	if err := s.userRepo.Update(ctx, user); err != nil {
		return nil, err
	}

	return s.userRepo.GetByID(ctx, userID)
}

// DeleteUser deletes a user (soft delete)
func (s *UserService) DeleteUser(ctx context.Context, userID string) error {
	return s.userRepo.Delete(ctx, userID)
}

// ListUsers retrieves a paginated list of users
func (s *UserService) ListUsers(ctx context.Context, page, pageSize int, search, role string, isActive *bool) ([]*models.User, int, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 10
	}

	return s.userRepo.List(ctx, page, pageSize, search, role, isActive)
}

// GetProfile retrieves a user's profile
func (s *UserService) GetProfile(ctx context.Context, userID string) (*models.Profile, error) {
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	return user.ToProfile(), nil
}

// UpdateProfile updates a user's profile
func (s *UserService) UpdateProfile(ctx context.Context, userID string, firstName, lastName, phone, avatarURL, bio, timezone *string) (*models.Profile, error) {
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	if firstName != nil {
		sanitizedFirstName, err := s.validator.SanitizeName(*firstName)
		if err != nil {
			return nil, err
		}
		user.FirstName = sanitizedFirstName
	}
	if lastName != nil {
		sanitizedLastName, err := s.validator.SanitizeName(*lastName)
		if err != nil {
			return nil, err
		}
		user.LastName = sanitizedLastName
	}
	if phone != nil {
		if err := s.validator.ValidatePhone(*phone); err != nil {
			return nil, err
		}
		user.Phone = *phone
	}
	if avatarURL != nil {
		user.AvatarURL = *avatarURL
	}
	if bio != nil {
		user.Bio = *bio
	}
	if timezone != nil {
		user.Timezone = *timezone
	}

	user.UpdatedAt = time.Now()

	if err := s.userRepo.Update(ctx, user); err != nil {
		return nil, err
	}

	user, err = s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	return user.ToProfile(), nil
}

// ChangePassword changes a user's password
func (s *UserService) ChangePassword(ctx context.Context, userID, oldPassword, newPassword string) error {
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return err
	}

	// Security: Use bcrypt.CompareHashAndPassword for timing-safe password comparison.
	// This prevents timing attacks where an attacker could determine password correctness
	// by measuring response time differences. bcrypt's comparison is constant-time.
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(oldPassword)); err != nil {
		return fmt.Errorf("invalid old password")
	}

	// Validate new password complexity before hashing
	if err := s.validator.ValidatePassword(newPassword); err != nil {
		return err
	}

	// Hash new password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}

	// Update password in database
	err = s.userRepo.UpdatePassword(ctx, userID, string(hashedPassword))
	if err != nil {
		return err
	}

	// Security: Invalidate all existing tokens for this user after password change
	// This ensures that if the password was compromised, all sessions are terminated
	if s.tokenBlacklist != nil {
		// Use maximum token lifetime (refresh token duration is typically longest)
		// This ensures all tokens are invalidated, even long-lived refresh tokens
		maxTokenLifetime := 7 * 24 * time.Hour // 7 days (typical refresh token lifetime)

		err = s.tokenBlacklist.BlacklistUserTokens(ctx, userID, maxTokenLifetime)
		if err != nil {
			// Log error but don't fail the password change
			// Password change is more critical than token invalidation
			s.logger.ErrorWithContext(ctx).
				Str("user_id", userID).
				Str("operation", "change_password").
				Str("error_type", "token_invalidation_failed").
				Err(err).
				Msg("failed to invalidate user tokens after password change")
		} else {
			s.logger.WithContext(ctx).
				Str("user_id", userID).
				Str("operation", "change_password").
				Msg("all user tokens invalidated after password change")
		}
	}

	s.logger.WithContext(ctx).
		Str("user_id", userID).
		Str("operation", "change_password").
		Msg("password changed successfully")

	return nil
}

// AssignRole assigns a role to a user
func (s *UserService) AssignRole(ctx context.Context, userID, role string) error {
	// Check if user exists
	if _, err := s.userRepo.GetByID(ctx, userID); err != nil {
		return err
	}

	return s.roleRepo.AssignRoleByName(ctx, userID, role)
}

// RemoveRole removes a role from a user
func (s *UserService) RemoveRole(ctx context.Context, userID, role string) error {
	return s.roleRepo.RemoveRoleByName(ctx, userID, role)
}

// GetUserRoles retrieves all roles for a user
func (s *UserService) GetUserRoles(ctx context.Context, userID string) ([]string, error) {
	return s.roleRepo.GetUserRoles(ctx, userID)
}

// CheckPermission checks if a user has a specific permission
func (s *UserService) CheckPermission(ctx context.Context, userID, permission string) (bool, error) {
	return s.roleRepo.CheckPermission(ctx, userID, permission)
}

// generateTokens generates access and refresh tokens for a user
func (s *UserService) generateTokens(user *models.User) (*models.TokenPair, error) {
	accessToken, expiresIn, err := s.tokenSvc.GenerateAccessToken(user.ID, user.Email, user.Roles)
	if err != nil {
		return nil, err
	}

	refreshToken, err := s.tokenSvc.GenerateRefreshToken(user.ID, user.Email, user.Roles)
	if err != nil {
		return nil, err
	}

	return &models.TokenPair{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    expiresIn,
	}, nil
}

// LoginWithAuthType initiates authentication using the specified authentication type.
// For normal authentication, it returns user and tokens immediately.
// For OAuth/SAML, it returns initiation data (authorization URL or SAML request).
func (s *UserService) LoginWithAuthType(ctx context.Context, authType AuthType, req *AuthRequest) (*AuthResult, error) {
	if s.strategyManager == nil {
		return nil, fmt.Errorf("authentication strategies not configured")
	}

	s.logger.WithContext(ctx).
		Str("auth_type", string(authType)).
		Str("operation", "login_with_auth_type").
		Msg("initiating authentication with strategy")

	strategy, err := s.strategyManager.GetStrategy(authType)
	if err != nil {
		s.logger.ErrorWithContext(ctx).
			Str("auth_type", string(authType)).
			Str("operation", "login_with_auth_type").
			Err(err).
			Msg("failed to get authentication strategy")
		return nil, fmt.Errorf("auth type not supported: %w", err)
	}

	result, err := strategy.Authenticate(ctx, req)
	if err != nil {
		s.logger.ErrorWithContext(ctx).
			Str("auth_type", string(authType)).
			Str("operation", "login_with_auth_type").
			Err(err).
			Msg("authentication failed")
		return nil, err
	}

	if result.Success {
		s.logger.WithContext(ctx).
			Str("auth_type", string(authType)).
			Str("user_id", result.User.ID).
			Str("operation", "login_with_auth_type").
			Msg("authentication successful")
	} else {
		s.logger.WithContext(ctx).
			Str("auth_type", string(authType)).
			Str("operation", "login_with_auth_type").
			Msg("authentication initiated, awaiting callback")
	}

	return result, nil
}

// HandleAuthCallback processes authentication callbacks from OAuth or SAML providers.
// This method completes the authentication flow after the user has been redirected
// back from the external identity provider.
func (s *UserService) HandleAuthCallback(ctx context.Context, authType AuthType, req *CallbackRequest) (*AuthResult, error) {
	if s.strategyManager == nil {
		return nil, fmt.Errorf("authentication strategies not configured")
	}

	s.logger.WithContext(ctx).
		Str("auth_type", string(authType)).
		Str("operation", "handle_auth_callback").
		Msg("processing authentication callback")

	strategy, err := s.strategyManager.GetStrategy(authType)
	if err != nil {
		s.logger.ErrorWithContext(ctx).
			Str("auth_type", string(authType)).
			Str("operation", "handle_auth_callback").
			Err(err).
			Msg("failed to get authentication strategy")
		return nil, fmt.Errorf("auth type not supported: %w", err)
	}

	result, err := strategy.HandleCallback(ctx, req)
	if err != nil {
		s.logger.ErrorWithContext(ctx).
			Str("auth_type", string(authType)).
			Str("operation", "handle_auth_callback").
			Err(err).
			Msg("callback processing failed")
		return nil, err
	}

	if result.Success && result.User != nil {
		s.logger.WithContext(ctx).
			Str("auth_type", string(authType)).
			Str("user_id", result.User.ID).
			Str("email", s.logger.RedactEmail(result.User.Email)).
			Str("operation", "handle_auth_callback").
			Msg("callback processed successfully")
	}

	return result, nil
}

// GetSupportedAuthTypes returns the list of authentication types that are currently
// registered and available for use. This is useful for clients to discover which
// authentication methods they can use.
func (s *UserService) GetSupportedAuthTypes() []AuthType {
	if s.strategyManager == nil {
		// If no strategy manager, only normal auth is supported
		return []AuthType{AuthTypeNormal}
	}

	// Get the active auth type from configuration
	activeAuthType := s.strategyManager.GetActiveAuthType()

	// Return the active auth type
	// Note: In the current implementation, only one auth type is active at a time
	return []AuthType{activeAuthType}
}

// deriveUsernameBase returns the lowercase local-part of the email with
// characters outside [a-z0-9_-.] stripped. Matches migration 010's backfill
// convention so fresh registrations share the same handle space as existing
// rows. Returns "user" as a last-resort fallback for pathological emails.
func deriveUsernameBase(email string) string {
	at := strings.IndexByte(email, '@')
	if at <= 0 {
		at = len(email)
	}
	raw := strings.ToLower(email[:at])
	var b strings.Builder
	b.Grow(len(raw))
	for _, r := range raw {
		switch {
		case r >= 'a' && r <= 'z',
			r >= '0' && r <= '9',
			r == '_', r == '-', r == '.':
			b.WriteRune(r)
		}
	}
	out := strings.Trim(b.String(), ".-_")
	if out == "" {
		return "user"
	}
	if len(out) > 60 {
		out = out[:60]
	}
	return out
}
