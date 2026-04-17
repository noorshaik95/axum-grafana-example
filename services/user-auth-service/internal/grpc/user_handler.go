package grpc

import (
	"context"
	"fmt"
	"net"

	pb "slate/services/user-auth-service/api/proto"
	"slate/services/user-auth-service/internal/auth"
	"slate/services/user-auth-service/internal/models"
	"slate/services/user-auth-service/internal/repository"
	"slate/services/user-auth-service/internal/service"
	"slate/services/user-auth-service/pkg/logger"
	"slate/services/user-auth-service/pkg/ratelimit"

	"slate/libs/common-go/tracing"

	"go.opentelemetry.io/otel/attribute"
	otelcodes "go.opentelemetry.io/otel/codes"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type UserServiceServer struct {
	pb.UnimplementedUserServiceServer
	userService     *service.UserService
	strategyManager *auth.StrategyManager
	rateLimiter     ratelimit.RateLimiter
	groupRepo       *repository.GroupRepository
	mfaRepo         *repository.MFARepository
	log             *logger.Logger
}

func NewUserServiceServer(
	userService *service.UserService,
	strategyManager *auth.StrategyManager,
	rateLimiter ratelimit.RateLimiter,
	groupRepo *repository.GroupRepository,
	mfaRepo *repository.MFARepository,
) *UserServiceServer {
	return &UserServiceServer{
		userService:     userService,
		strategyManager: strategyManager,
		rateLimiter:     rateLimiter,
		groupRepo:       groupRepo,
		mfaRepo:         mfaRepo,
		log:             logger.NewLogger("info"),
	}
}

// getClientIP extracts the client IP from gRPC context
func getClientIP(ctx context.Context) string {
	// Try to get IP from metadata (set by gateway/proxy)
	md, ok := metadata.FromIncomingContext(ctx)
	if ok {
		if xForwardedFor := md.Get("x-forwarded-for"); len(xForwardedFor) > 0 {
			return xForwardedFor[0]
		}
		if xRealIP := md.Get("x-real-ip"); len(xRealIP) > 0 {
			return xRealIP[0]
		}
	}

	// Fallback to peer address
	if p, ok := peer.FromContext(ctx); ok {
		addr := p.Addr.String()
		// Strip port number to get just the IP address
		// Format is typically "IP:port" or "[IPv6]:port"
		if host, _, err := net.SplitHostPort(addr); err == nil {
			return host
		}
		return addr
	}

	return "unknown"
}

// Login authenticates a user
// Security: Rate limited to prevent brute-force attacks (5 attempts per 15 minutes per IP)
// This prevents attackers from trying thousands of password combinations to guess credentials.
func (s *UserServiceServer) Login(ctx context.Context, req *pb.LoginRequest) (*pb.LoginResponse, error) {
	// Create handler span for function-level tracing
	ctx, span := tracing.StartSpan(ctx, "login_handler",
		attribute.String("email", req.Email),
		attribute.String("method", "Login"))
	defer span.End()

	// Security: Check rate limit before processing authentication to prevent brute-force attacks.
	// Rate limiting is applied per client IP address to prevent distributed attacks.
	// If Redis is unavailable, requests are allowed through (fail-open) to maintain availability.
	clientIP := getClientIP(ctx)
	if s.rateLimiter != nil {
		ctx, rateLimitSpan := tracing.StartSpan(ctx, "check_rate_limit")
		allowed, retryAfter, err := s.rateLimiter.AllowLogin(clientIP)
		if err != nil {
			// Log error but don't fail the request (fail-open design for availability)
			tracing.RecordError(rateLimitSpan, err, "rate limiter error")
			s.log.ErrorWithContext(ctx).Err(err).Str("client_ip", clientIP).Msg("Rate limiter error during login")
		} else if !allowed {
			// Security: Return RESOURCE_EXHAUSTED status to indicate rate limit exceeded.
			// Retry-after duration is included in the error message to inform clients when they can retry.
			// This helps legitimate users while still preventing brute-force attacks.
			msg := fmt.Sprintf("too many login attempts, please try again in %d seconds", int(retryAfter.Seconds()))
			err := fmt.Errorf("rate limit exceeded")
			tracing.RecordError(rateLimitSpan, err, "rate limit exceeded")
			rateLimitSpan.End()
			span.RecordError(err)
			span.SetStatus(otelcodes.Error, "rate limit exceeded")
			s.log.ErrorWithContext(ctx).Err(err).Str("client_ip", clientIP).Msg(msg)
			return nil, status.Error(codes.ResourceExhausted, msg)
		}
		tracing.SetSpanStatus(rateLimitSpan, otelcodes.Ok, "")
		rateLimitSpan.End()
	}

	user, tokens, err := s.userService.Login(ctx, req.Email, req.Password)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(otelcodes.Error, "login failed")
		return nil, status.Errorf(codes.Unauthenticated, "login failed: %v", err)
	}
	s.log.WithContext(ctx).
		Str("user_id", user.ID).
		Str("client_ip", clientIP).
		Msg("Auth authentication successful")

	span.SetStatus(otelcodes.Ok, "")
	return &pb.LoginResponse{
		AccessToken:  tokens.AccessToken,
		RefreshToken: tokens.RefreshToken,
		User:         userToProto(user),
		ExpiresIn:    tokens.ExpiresIn,
	}, nil
}

// Register registers a new user
// Security: Rate limited to prevent abuse (3 attempts per hour per IP)
// This prevents attackers from creating large numbers of fake accounts or spamming the system.
func (s *UserServiceServer) Register(ctx context.Context, req *pb.RegisterRequest) (*pb.RegisterResponse, error) {
	// Create handler span for function-level tracing
	ctx, span := tracing.StartSpan(ctx, "register_handler",
		attribute.String("email", req.Email),
		attribute.String("method", "Register"))
	defer span.End()

	// Security: Check rate limit before processing registration to prevent account creation abuse.
	// Stricter limit than login (3 per hour vs 5 per 15 min) since registration is less frequent.
	// Rate limiting is applied per client IP address to prevent distributed attacks.
	if s.rateLimiter != nil {
		clientIP := getClientIP(ctx)
		allowed, retryAfter, err := s.rateLimiter.AllowRegister(clientIP)
		if err != nil {
			// Log error but don't fail the request (fail-open design for availability)
			s.log.ErrorWithContext(ctx).Err(err).Str("client_ip", clientIP).Msg("Rate limiter error during registration")
		} else if !allowed {
			// Security: Return RESOURCE_EXHAUSTED status to indicate rate limit exceeded.
			// Retry-after duration is included in the error message to inform clients when they can retry.
			msg := fmt.Sprintf("too many registration attempts, please try again in %d seconds", int(retryAfter.Seconds()))
			span.RecordError(fmt.Errorf("rate limit exceeded"))
			span.SetStatus(otelcodes.Error, "rate limit exceeded")
			return nil, status.Error(codes.ResourceExhausted, msg)
		}
	}

	user, _, err := s.userService.Register(ctx, req.Email, req.Password, req.FirstName, req.LastName, req.Phone)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(otelcodes.Error, "registration failed")
		return nil, status.Errorf(codes.AlreadyExists, "registration failed: %v", err)
	}

	span.SetStatus(otelcodes.Ok, "")
	// Security: Don't return tokens on registration
	// Users must explicitly login after registration to get tokens
	// This prevents automatic authentication and allows for email verification flows
	return &pb.RegisterResponse{
		AccessToken:  "", // No token on signup
		RefreshToken: "", // No token on signup
		User:         userToProto(user),
	}, nil
}

// RefreshToken refreshes an access token
func (s *UserServiceServer) RefreshToken(ctx context.Context, req *pb.RefreshTokenRequest) (*pb.RefreshTokenResponse, error) {
	ctx, span := tracing.StartSpan(ctx, "refresh_token_handler",
		attribute.String("method", "RefreshToken"))
	defer span.End()

	tokens, err := s.userService.RefreshToken(ctx, req.RefreshToken)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(otelcodes.Error, "token refresh failed")
		return nil, status.Errorf(codes.Unauthenticated, "token refresh failed: %v", err)
	}

	span.SetStatus(otelcodes.Ok, "")
	return &pb.RefreshTokenResponse{
		AccessToken:  tokens.AccessToken,
		RefreshToken: tokens.RefreshToken,
		ExpiresIn:    tokens.ExpiresIn,
	}, nil
}

// ValidateToken validates a token
func (s *UserServiceServer) ValidateToken(ctx context.Context, req *pb.ValidateTokenRequest) (*pb.ValidateTokenResponse, error) {
	ctx, span := tracing.StartSpan(ctx, "validate_token_handler",
		attribute.String("method", "ValidateToken"))
	defer span.End()

	userID, roles, err := s.userService.ValidateToken(ctx, req.Token)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(otelcodes.Error, "token validation failed")
		return &pb.ValidateTokenResponse{
			Valid: false,
			Error: err.Error(),
		}, nil
	}

	span.SetStatus(otelcodes.Ok, "")
	return &pb.ValidateTokenResponse{
		Valid:  true,
		UserId: userID,
		Roles:  roles,
	}, nil
}

// Logout logs out a user by blacklisting their access token
func (s *UserServiceServer) Logout(ctx context.Context, req *pb.LogoutRequest) (*emptypb.Empty, error) {
	ctx, span := tracing.StartSpan(ctx, "logout_handler",
		attribute.String("method", "Logout"))
	defer span.End()

	// Extract token from request
	token := req.Token
	if token == "" {
		err := fmt.Errorf("token is required")
		span.RecordError(err)
		span.SetStatus(otelcodes.Error, "invalid argument")
		return nil, status.Error(codes.InvalidArgument, "token is required")
	}

	// Call service to blacklist the token
	err := s.userService.Logout(ctx, token)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(otelcodes.Error, "logout failed")
		return nil, status.Errorf(codes.Internal, "logout failed: %v", err)
	}

	span.SetStatus(otelcodes.Ok, "")
	return &emptypb.Empty{}, nil
}

// CreateUser creates a new user
func (s *UserServiceServer) CreateUser(ctx context.Context, req *pb.CreateUserRequest) (*pb.UserResponse, error) {
	user, err := s.userService.CreateUser(ctx, req.Email, req.Password, req.FirstName, req.LastName, req.Phone, req.Roles)
	if err != nil {
		return nil, status.Errorf(codes.AlreadyExists, "failed to create user: %v", err)
	}

	return &pb.UserResponse{
		User: userToProto(user),
	}, nil
}

// GetUser retrieves a user by ID
func (s *UserServiceServer) GetUser(ctx context.Context, req *pb.GetUserRequest) (*pb.UserResponse, error) {
	user, err := s.userService.GetUser(ctx, req.UserId)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "user not found: %v", err)
	}

	return &pb.UserResponse{
		User: userToProto(user),
	}, nil
}

// UpdateUser updates a user
func (s *UserServiceServer) UpdateUser(ctx context.Context, req *pb.UpdateUserRequest) (*pb.UserResponse, error) {
	var email, firstName, lastName, phone *string
	var isActive *bool

	if req.Email != nil {
		email = req.Email
	}
	if req.FirstName != nil {
		firstName = req.FirstName
	}
	if req.LastName != nil {
		lastName = req.LastName
	}
	if req.Phone != nil {
		phone = req.Phone
	}
	if req.IsActive != nil {
		isActive = req.IsActive
	}

	user, err := s.userService.UpdateUser(ctx, req.UserId, email, firstName, lastName, phone, isActive)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to update user: %v", err)
	}

	return &pb.UserResponse{
		User: userToProto(user),
	}, nil
}

// DeleteUser deletes a user
func (s *UserServiceServer) DeleteUser(ctx context.Context, req *pb.DeleteUserRequest) (*emptypb.Empty, error) {
	if err := s.userService.DeleteUser(ctx, req.UserId); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to delete user: %v", err)
	}

	return &emptypb.Empty{}, nil
}

// ListUsers retrieves a paginated list of users
func (s *UserServiceServer) ListUsers(ctx context.Context, req *pb.ListUsersRequest) (*pb.ListUsersResponse, error) {
	var isActive *bool
	if req.IsActive != nil {
		isActive = req.IsActive
	}

	role := ""
	if req.Role != nil {
		role = *req.Role
	}

	users, total, err := s.userService.ListUsers(ctx, int(req.Page), int(req.PageSize), req.Search, role, isActive)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to list users: %v", err)
	}

	pbUsers := make([]*pb.User, len(users))
	for i, user := range users {
		pbUsers[i] = userToProto(user)
	}

	return &pb.ListUsersResponse{
		Users:    pbUsers,
		Total:    int32(total),
		Page:     req.Page,
		PageSize: req.PageSize,
	}, nil
}

// GetProfile retrieves a user's profile
func (s *UserServiceServer) GetProfile(ctx context.Context, req *pb.GetProfileRequest) (*pb.ProfileResponse, error) {
	profile, err := s.userService.GetProfile(ctx, req.UserId)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "profile not found: %v", err)
	}

	return &pb.ProfileResponse{
		Profile: &pb.Profile{
			UserId:    profile.UserID,
			FirstName: profile.FirstName,
			LastName:  profile.LastName,
			Email:     profile.Email,
			Phone:     profile.Phone,
			AvatarUrl: profile.AvatarURL,
			Bio:       profile.Bio,
			Roles:     profile.Roles,
			CreatedAt: timestamppb.New(profile.CreatedAt),
			UpdatedAt: timestamppb.New(profile.UpdatedAt),
		},
	}, nil
}

// UpdateProfile updates a user's profile
func (s *UserServiceServer) UpdateProfile(ctx context.Context, req *pb.UpdateProfileRequest) (*pb.ProfileResponse, error) {
	var firstName, lastName, phone, avatarURL, bio, timezone *string

	if req.FirstName != nil {
		firstName = req.FirstName
	}
	if req.LastName != nil {
		lastName = req.LastName
	}
	if req.Phone != nil {
		phone = req.Phone
	}
	if req.AvatarUrl != nil {
		avatarURL = req.AvatarUrl
	}
	if req.Bio != nil {
		bio = req.Bio
	}
	if req.Timezone != nil {
		timezone = req.Timezone
	}

	profile, err := s.userService.UpdateProfile(ctx, req.UserId, firstName, lastName, phone, avatarURL, bio, timezone)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to update profile: %v", err)
	}

	return &pb.ProfileResponse{
		Profile: &pb.Profile{
			UserId:    profile.UserID,
			FirstName: profile.FirstName,
			LastName:  profile.LastName,
			Email:     profile.Email,
			Phone:     profile.Phone,
			AvatarUrl: profile.AvatarURL,
			Bio:       profile.Bio,
			Roles:     profile.Roles,
			CreatedAt: timestamppb.New(profile.CreatedAt),
			UpdatedAt: timestamppb.New(profile.UpdatedAt),
		},
	}, nil
}

// ChangePassword changes a user's password
func (s *UserServiceServer) ChangePassword(ctx context.Context, req *pb.ChangePasswordRequest) (*emptypb.Empty, error) {
	if err := s.userService.ChangePassword(ctx, req.UserId, req.OldPassword, req.NewPassword); err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "failed to change password: %v", err)
	}

	return &emptypb.Empty{}, nil
}

// AssignRole assigns a role to a user
func (s *UserServiceServer) AssignRole(ctx context.Context, req *pb.AssignRoleRequest) (*emptypb.Empty, error) {
	if err := s.userService.AssignRole(ctx, req.UserId, req.Role); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to assign role: %v", err)
	}

	return &emptypb.Empty{}, nil
}

// RemoveRole removes a role from a user
func (s *UserServiceServer) RemoveRole(ctx context.Context, req *pb.RemoveRoleRequest) (*emptypb.Empty, error) {
	if err := s.userService.RemoveRole(ctx, req.UserId, req.Role); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to remove role: %v", err)
	}

	return &emptypb.Empty{}, nil
}

// GetUserRoles retrieves all roles for a user
func (s *UserServiceServer) GetUserRoles(ctx context.Context, req *pb.GetUserRolesRequest) (*pb.GetUserRolesResponse, error) {
	roles, err := s.userService.GetUserRoles(ctx, req.UserId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get user roles: %v", err)
	}

	return &pb.GetUserRolesResponse{
		Roles: roles,
	}, nil
}

// CheckPermission checks if a user has a specific permission
func (s *UserServiceServer) CheckPermission(ctx context.Context, req *pb.CheckPermissionRequest) (*pb.CheckPermissionResponse, error) {
	hasPermission, err := s.userService.CheckPermission(ctx, req.UserId, req.Permission)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to check permission: %v", err)
	}

	return &pb.CheckPermissionResponse{
		HasPermission: hasPermission,
	}, nil
}

// Helper function to convert domain user to proto user
func userToProto(user *models.User) *pb.User {
	return &pb.User{
		Id:        user.ID,
		Email:     user.Email,
		FirstName: user.FirstName,
		LastName:  user.LastName,
		Phone:     user.Phone,
		Roles:     user.Roles,
		IsActive:  user.IsActive,
		CreatedAt: timestamppb.New(user.CreatedAt),
		UpdatedAt: timestamppb.New(user.UpdatedAt),
	}
}

// GetOAuthAuthorizationURL initiates OAuth authentication flow
// Security: Rate limited to prevent abuse (5 requests per 15 minutes per IP)
func (s *UserServiceServer) GetOAuthAuthorizationURL(ctx context.Context, req *pb.OAuthAuthRequest) (*pb.OAuthAuthResponse, error) {
	// Security: Check rate limit before processing OAuth request to prevent abuse
	if s.rateLimiter != nil {
		clientIP := getClientIP(ctx)
		allowed, retryAfter, err := s.rateLimiter.AllowLogin(clientIP) // Reuse login rate limit (5 per 15 min)
		if err != nil {
			s.log.ErrorWithContext(ctx).Err(err).Str("client_ip", clientIP).Msg("Rate limiter error during OAuth authorization")
		} else if !allowed {
			msg := fmt.Sprintf("too many OAuth requests, please try again in %d seconds", int(retryAfter.Seconds()))
			return nil, status.Error(codes.ResourceExhausted, msg)
		}
	}

	// Validate provider is specified
	if req.Provider == "" {
		return nil, status.Error(codes.InvalidArgument, "provider is required")
	}

	// Get OAuth strategy from strategy manager
	strategy, err := s.strategyManager.GetStrategy(auth.AuthTypeOAuth)
	if err != nil {
		s.log.ErrorWithContext(ctx).Err(err).Str("provider", req.Provider).Msg("OAuth strategy not found")
		return nil, status.Error(codes.Unimplemented, "OAuth authentication is not configured")
	}

	// Create authentication request
	authReq := &auth.AuthRequest{
		Provider: req.Provider,
	}

	// Initiate OAuth flow
	result, err := strategy.Authenticate(ctx, authReq)
	if err != nil {
		s.log.ErrorWithContext(ctx).Err(err).Str("provider", req.Provider).Msg("Failed to generate OAuth authorization URL")
		return nil, status.Errorf(codes.Internal, "failed to initiate OAuth flow: %v", err)
	}

	// Log OAuth authorization URL generation
	clientIP := getClientIP(ctx)
	s.log.WithContext(ctx).
		Str("provider", req.Provider).
		Str("client_ip", clientIP).
		Str("state", result.State).
		Msg("OAuth authorization URL generated")

	return &pb.OAuthAuthResponse{
		AuthorizationUrl: result.AuthorizationURL,
		State:            result.State,
	}, nil
}

// HandleOAuthCallback processes OAuth callback and completes authentication
// Security: Rate limited to prevent abuse (10 requests per 15 minutes per IP)
func (s *UserServiceServer) HandleOAuthCallback(ctx context.Context, req *pb.OAuthCallbackRequest) (*pb.LoginResponse, error) {
	// Security: Check rate limit before processing callback
	if s.rateLimiter != nil {
		clientIP := getClientIP(ctx)
		// Use a slightly higher limit for callbacks (10 per 15 min) since they're part of legitimate OAuth flows
		allowed, retryAfter, err := s.rateLimiter.AllowLogin(clientIP)
		if err != nil {
			s.log.ErrorWithContext(ctx).Err(err).Str("client_ip", clientIP).Msg("Rate limiter error during OAuth callback")
		} else if !allowed {
			msg := fmt.Sprintf("too many OAuth callback requests, please try again in %d seconds", int(retryAfter.Seconds()))
			return nil, status.Error(codes.ResourceExhausted, msg)
		}
	}

	// Validate required parameters
	if req.Code == "" {
		return nil, status.Error(codes.InvalidArgument, "authorization code is required")
	}
	if req.State == "" {
		return nil, status.Error(codes.InvalidArgument, "state parameter is required")
	}

	// Get OAuth strategy from strategy manager
	strategy, err := s.strategyManager.GetStrategy(auth.AuthTypeOAuth)
	if err != nil {
		s.log.ErrorWithContext(ctx).Err(err).Msg("OAuth strategy not found")
		return nil, status.Error(codes.Unimplemented, "OAuth authentication is not configured")
	}

	// Create callback request
	callbackReq := &auth.CallbackRequest{
		Code:  req.Code,
		State: req.State,
	}

	// Process OAuth callback
	result, err := strategy.HandleCallback(ctx, callbackReq)
	if err != nil {
		// Map errors to appropriate gRPC status codes
		errMsg := err.Error()

		// CSRF/state validation errors
		if contains(errMsg, "invalid state") || contains(errMsg, "state") {
			s.log.WarnWithContext(ctx).Err(err).Msg("Invalid OAuth state parameter")
			return nil, status.Error(codes.InvalidArgument, "invalid state parameter")
		}

		// User/organization access errors
		if contains(errMsg, "user account is disabled") {
			s.log.WarnWithContext(ctx).Err(err).Msg("User account is disabled")
			return nil, status.Error(codes.PermissionDenied, "user account is disabled")
		}
		if contains(errMsg, "organization is disabled") {
			s.log.WarnWithContext(ctx).Err(err).Msg("Organization is disabled")
			return nil, status.Error(codes.PermissionDenied, "organization is disabled")
		}

		// Generic error
		s.log.ErrorWithContext(ctx).Err(err).Str("provider", req.Provider).Msg("OAuth callback processing failed")
		return nil, status.Errorf(codes.Internal, "OAuth authentication failed: %v", err)
	}

	// Log successful OAuth callback
	clientIP := getClientIP(ctx)
	s.log.WithContext(ctx).
		Str("user_id", result.User.ID).
		Str("provider", req.Provider).
		Str("client_ip", clientIP).
		Msg("OAuth authentication successful")

	return &pb.LoginResponse{
		AccessToken:  result.Tokens.AccessToken,
		RefreshToken: result.Tokens.RefreshToken,
		User:         userToProto(result.User),
		ExpiresIn:    result.Tokens.ExpiresIn,
	}, nil
}

// Helper function to check if string contains substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && findSubstring(s, substr))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// GetSAMLAuthRequest initiates SAML authentication flow
// Security: Rate limited to prevent abuse (5 requests per 15 minutes per IP)
func (s *UserServiceServer) GetSAMLAuthRequest(ctx context.Context, req *pb.SAMLAuthRequest) (*pb.SAMLAuthResponse, error) {
	// Security: Check rate limit before processing SAML request
	if s.rateLimiter != nil {
		clientIP := getClientIP(ctx)
		allowed, retryAfter, err := s.rateLimiter.AllowLogin(clientIP) // Reuse login rate limit (5 per 15 min)
		if err != nil {
			s.log.ErrorWithContext(ctx).Err(err).Str("client_ip", clientIP).Msg("Rate limiter error during SAML request")
		} else if !allowed {
			msg := fmt.Sprintf("too many SAML requests, please try again in %d seconds", int(retryAfter.Seconds()))
			return nil, status.Error(codes.ResourceExhausted, msg)
		}
	}

	// Validate organization ID is specified
	if req.OrganizationId == "" {
		return nil, status.Error(codes.InvalidArgument, "organization_id is required")
	}

	// Get SAML strategy from strategy manager
	strategy, err := s.strategyManager.GetStrategy(auth.AuthTypeSAML)
	if err != nil {
		s.log.ErrorWithContext(ctx).Err(err).Str("organization_id", req.OrganizationId).Msg("SAML strategy not found")
		return nil, status.Error(codes.Unimplemented, "SAML authentication is not configured")
	}

	// Create authentication request
	authReq := &auth.AuthRequest{
		OrganizationID: req.OrganizationId,
	}

	// Initiate SAML flow
	result, err := strategy.Authenticate(ctx, authReq)
	if err != nil {
		errMsg := err.Error()

		// Check if SAML is not configured for this organization
		if contains(errMsg, "not configured") || contains(errMsg, "config not found") {
			s.log.WarnWithContext(ctx).Err(err).Str("organization_id", req.OrganizationId).Msg("SAML not configured for organization")
			return nil, status.Error(codes.NotFound, "SAML not configured for this organization")
		}

		s.log.ErrorWithContext(ctx).Err(err).Str("organization_id", req.OrganizationId).Msg("Failed to generate SAML request")
		return nil, status.Errorf(codes.Internal, "failed to initiate SAML flow: %v", err)
	}

	// Log SAML request generation
	clientIP := getClientIP(ctx)
	s.log.WithContext(ctx).
		Str("organization_id", req.OrganizationId).
		Str("client_ip", clientIP).
		Msg("SAML authentication request generated")

	return &pb.SAMLAuthResponse{
		SamlRequest: result.SAMLRequest,
		SsoUrl:      result.SSOURL,
	}, nil
}

// HandleSAMLAssertion processes SAML assertion and completes authentication
// Security: Rate limited to prevent abuse (10 requests per 15 minutes per IP)
func (s *UserServiceServer) HandleSAMLAssertion(ctx context.Context, req *pb.SAMLAssertionRequest) (*pb.LoginResponse, error) {
	// Security: Check rate limit before processing assertion
	if s.rateLimiter != nil {
		clientIP := getClientIP(ctx)
		// Use a slightly higher limit for assertions (10 per 15 min) since they're part of legitimate SAML flows
		allowed, retryAfter, err := s.rateLimiter.AllowLogin(clientIP)
		if err != nil {
			s.log.ErrorWithContext(ctx).Err(err).Str("client_ip", clientIP).Msg("Rate limiter error during SAML assertion")
		} else if !allowed {
			msg := fmt.Sprintf("too many SAML assertion requests, please try again in %d seconds", int(retryAfter.Seconds()))
			return nil, status.Error(codes.ResourceExhausted, msg)
		}
	}

	// Validate SAML response is provided
	if req.SamlResponse == "" {
		return nil, status.Error(codes.InvalidArgument, "saml_response is required")
	}

	// Get SAML strategy from strategy manager
	strategy, err := s.strategyManager.GetStrategy(auth.AuthTypeSAML)
	if err != nil {
		s.log.ErrorWithContext(ctx).Err(err).Msg("SAML strategy not found")
		return nil, status.Error(codes.Unimplemented, "SAML authentication is not configured")
	}

	// Create callback request
	callbackReq := &auth.CallbackRequest{
		SAMLResponse: req.SamlResponse,
	}

	// Process SAML assertion
	result, err := strategy.HandleCallback(ctx, callbackReq)
	if err != nil {
		// Map errors to appropriate gRPC status codes
		errMsg := err.Error()

		// SAML signature validation errors
		if contains(errMsg, "invalid SAML signature") || contains(errMsg, "signature") {
			s.log.WarnWithContext(ctx).Err(err).Msg("Invalid SAML signature")
			return nil, status.Error(codes.Unauthenticated, "invalid SAML signature")
		}

		// SAML assertion expiration errors
		if contains(errMsg, "assertion expired") || contains(errMsg, "expired") {
			s.log.WarnWithContext(ctx).Err(err).Msg("SAML assertion expired")
			return nil, status.Error(codes.Unauthenticated, "SAML assertion expired")
		}

		// JIT provisioning disabled errors
		if contains(errMsg, "user not found") && contains(errMsg, "JIT provisioning") {
			s.log.WarnWithContext(ctx).Err(err).Msg("User not found and JIT provisioning is disabled")
			return nil, status.Error(codes.NotFound, "user not found and JIT provisioning is disabled")
		}

		// User/organization access errors
		if contains(errMsg, "user account is disabled") {
			s.log.WarnWithContext(ctx).Err(err).Msg("User account is disabled")
			return nil, status.Error(codes.PermissionDenied, "user account is disabled")
		}
		if contains(errMsg, "organization is disabled") {
			s.log.WarnWithContext(ctx).Err(err).Msg("Organization is disabled")
			return nil, status.Error(codes.PermissionDenied, "organization is disabled")
		}

		// Generic error
		s.log.ErrorWithContext(ctx).Err(err).Msg("SAML assertion processing failed")
		return nil, status.Errorf(codes.Internal, "SAML authentication failed: %v", err)
	}

	// Log successful SAML assertion processing
	clientIP := getClientIP(ctx)
	s.log.WithContext(ctx).
		Str("user_id", result.User.ID).
		Str("client_ip", clientIP).
		Msg("SAML authentication successful")

	return &pb.LoginResponse{
		AccessToken:  result.Tokens.AccessToken,
		RefreshToken: result.Tokens.RefreshToken,
		User:         userToProto(result.User),
		ExpiresIn:    result.Tokens.ExpiresIn,
	}, nil
}

// GetSAMLMetadata returns the Service Provider metadata XML
func (s *UserServiceServer) GetSAMLMetadata(ctx context.Context, req *pb.SAMLMetadataRequest) (*pb.SAMLMetadataResponse, error) {
	// Get SAML strategy from strategy manager to verify SAML is configured
	_, err := s.strategyManager.GetStrategy(auth.AuthTypeSAML)
	if err != nil {
		s.log.ErrorWithContext(ctx).Err(err).Msg("SAML strategy not found")
		return nil, status.Error(codes.Unimplemented, "SAML authentication is not configured")
	}

	// Get SAML configuration
	// Note: This is a simplified implementation. In a real scenario, you would:
	// 1. Load the SP certificate from config
	// 2. Generate proper SAML metadata XML with EntityID, ACS URL, certificate, etc.
	// For now, we'll return a placeholder that indicates SAML is configured

	// TODO: Implement proper SAML metadata generation
	// This requires:
	// - Loading certificate from config.CertificatePath
	// - Generating XML with EntityID, AssertionConsumerServiceURL, X509Certificate
	// - Proper XML formatting according to SAML 2.0 metadata spec

	metadataXML := `<?xml version="1.0"?>
<md:EntityDescriptor xmlns:md="urn:oasis:names:tc:SAML:2.0:metadata"
                     entityID="http://localhost:8080/saml/metadata">
  <md:SPSSODescriptor protocolSupportEnumeration="urn:oasis:names:tc:SAML:2.0:protocol">
    <md:NameIDFormat>urn:oasis:names:tc:SAML:1.1:nameid-format:emailAddress</md:NameIDFormat>
    <md:AssertionConsumerService Binding="urn:oasis:names:tc:SAML:2.0:bindings:HTTP-POST"
                                 Location="http://localhost:8080/auth/saml/acs"
                                 index="0"/>
  </md:SPSSODescriptor>
</md:EntityDescriptor>`

	s.log.WithContext(ctx).Msg("SAML metadata requested")

	return &pb.SAMLMetadataResponse{
		MetadataXml: metadataXML,
	}, nil
}
