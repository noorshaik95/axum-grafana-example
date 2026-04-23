// Package grpc wires the AdminAuthService proto interface to the domain service.
package grpc

import (
	"context"
	"database/sql"
	"errors"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	commontracing "slate/libs/common-go/tracing"
	pb "slate/services/admin-auth-service/api/proto/adminauthpb"
	"slate/services/admin-auth-service/internal/models"
	"slate/services/admin-auth-service/internal/service"
)

// AdminAuthServer implements pb.AdminAuthServiceServer.
type AdminAuthServer struct {
	pb.UnimplementedAdminAuthServiceServer
	svc      *service.AdminService
	// tenantDB is an optional handle to the shared tenant database used to
	// resolve tenant_id → slug when the caller does not supply tenant_slug.
	// May be nil when TENANT_DB_DSN is not configured.
	tenantDB *sql.DB
}

// NewAdminAuthServer wires the server to an AdminService.
func NewAdminAuthServer(svc *service.AdminService) *AdminAuthServer {
	return &AdminAuthServer{svc: svc}
}

// WithTenantDB attaches a tenant database handle for slug lookups.
func (s *AdminAuthServer) WithTenantDB(db *sql.DB) *AdminAuthServer {
	s.tenantDB = db
	return s
}

// lookupTenantSlug resolves a tenant UUID to its slug by querying tenants_v2.
// Returns "" when the DB is not configured or the row is not found.
func (s *AdminAuthServer) lookupTenantSlug(ctx context.Context, tenantID string) string {
	if s.tenantDB == nil || tenantID == "" {
		return ""
	}
	var slug string
	err := s.tenantDB.QueryRowContext(ctx,
		`SELECT slug FROM tenants_v2 WHERE id = $1 AND status != 'deleted' LIMIT 1`,
		tenantID,
	).Scan(&slug)
	if err != nil {
		return ""
	}
	return slug
}


// AdminLogin handles POST /api/admin/auth/login.
func (s *AdminAuthServer) AdminLogin(ctx context.Context, req *pb.LoginRequest) (*pb.LoginResponse, error) {
	tagSpan(ctx)
	res, err := s.svc.Login(ctx, req.GetEmail(), req.GetPassword())
	if err != nil {
		return nil, toStatusErr(err)
	}
	return &pb.LoginResponse{
		AccessToken:     res.AccessToken,
		RefreshToken:    res.RefreshToken,
		ExpiresAtUnixMs: res.ExpiresAt.UnixMilli(),
		User:            adminToProto(res.User),
	}, nil
}

// AdminLogout handles POST /api/admin/auth/logout.
func (s *AdminAuthServer) AdminLogout(ctx context.Context, req *pb.LogoutRequest) (*pb.LogoutResponse, error) {
	tagSpan(ctx)
	if err := s.svc.Logout(ctx, req.GetAccessToken()); err != nil {
		return nil, toStatusErr(err)
	}
	return &pb.LogoutResponse{Success: true}, nil
}

// ValidateAdminToken validates an HS256 admin JWT with aud=platform.
func (s *AdminAuthServer) ValidateAdminToken(ctx context.Context, req *pb.ValidateTokenRequest) (*pb.ValidateTokenResponse, error) {
	tagSpan(ctx)
	res := s.svc.ValidateAdminToken(ctx, req.GetToken())
	return &pb.ValidateTokenResponse{
		Valid:    res.Valid,
		UserId:   res.UserID,
		Email:    res.Email,
		Roles:    res.Roles,
		Audience: res.Audience,
		Error:    res.Error,
	}, nil
}

// ListAdminUsers returns the paginated list of platform admins.
func (s *AdminAuthServer) ListAdminUsers(ctx context.Context, req *pb.ListAdminUsersRequest) (*pb.ListAdminUsersResponse, error) {
	tagSpan(ctx)
	actor := actorFromCtx(ctx)
	admins, total, err := s.svc.ListAdmins(ctx, actor, int(req.GetLimit()), int(req.GetOffset()))
	if err != nil {
		return nil, toStatusErr(err)
	}
	out := make([]*pb.AdminUser, 0, len(admins))
	for _, a := range admins {
		out = append(out, adminToProto(a))
	}
	return &pb.ListAdminUsersResponse{Users: out, Total: int32(total)}, nil
}

// Register creates a new platform admin.
func (s *AdminAuthServer) Register(ctx context.Context, req *pb.RegisterRequest) (*pb.RegisterResponse, error) {
	tagSpan(ctx)
	admin, err := s.svc.Register(ctx,
		req.GetEmail(), req.GetPassword(), req.GetFullName(), req.GetRoles(), req.GetCreatedBy(),
	)
	if err != nil {
		return nil, toStatusErr(err)
	}
	return &pb.RegisterResponse{User: adminToProto(admin)}, nil
}

// Impersonate mints an impersonation JWT and redirect URL.
//
// admin_user_id resolution (in order of preference):
//  1. req.AdminUserId if present (explicit / backward-compat)
//  2. Actor from gRPC context — populated by the actorExtractUnary interceptor
//     that parses the "authorization" Bearer JWT forwarded by the gateway.
//
// tenant_slug resolution (in order of preference):
//  1. req.TenantSlug if present (explicit / backward-compat)
//  2. DB lookup against tenants_v2.slug WHERE id = req.TenantId, when
//     TENANT_DB_DSN is configured.
func (s *AdminAuthServer) Impersonate(ctx context.Context, req *pb.ImpersonateRequest) (*pb.ImpersonateResponse, error) {
	tagSpan(ctx)

	// Resolve admin_user_id.
	adminUserID := req.GetAdminUserId()
	if adminUserID == "" {
		adminUserID = actorFromCtx(ctx)
	}

	// Resolve tenant_slug.
	tenantSlug := req.GetTenantSlug()
	if tenantSlug == "" {
		tenantSlug = s.lookupTenantSlug(ctx, req.GetTenantId())
	}

	res, err := s.svc.Impersonate(ctx,
		adminUserID, req.GetTenantId(), tenantSlug, req.GetTargetUserId(),
	)
	if err != nil {
		return nil, toStatusErr(err)
	}
	return &pb.ImpersonateResponse{
		Token:           res.Token,
		RedirectUrl:     res.RedirectURL,
		ImpersonationId: res.ImpersonationID,
		ExpiresAtUnixMs: res.ExpiresAt.UnixMilli(),
	}, nil
}

// GenerateImpersonationToken is an alias for Impersonate (plan W2.2 naming).
func (s *AdminAuthServer) GenerateImpersonationToken(ctx context.Context, req *pb.ImpersonateRequest) (*pb.ImpersonateResponse, error) {
	return s.Impersonate(ctx, req)
}

// ---- helpers ------------------------------------------------------------

func tagSpan(ctx context.Context) {
	// TagSpanWithCorrelation is safe to call with the active span.
	span := spanFromContext(ctx)
	commontracing.TagSpanWithCorrelation(ctx, span)
}

func toStatusErr(err error) error {
	switch {
	case errors.Is(err, service.ErrInvalidCredentials):
		return status.Error(codes.Unauthenticated, err.Error())
	case errors.Is(err, service.ErrAccountDisabled):
		return status.Error(codes.PermissionDenied, err.Error())
	case errors.Is(err, service.ErrForbidden):
		return status.Error(codes.PermissionDenied, err.Error())
	case errors.Is(err, service.ErrInvalidInput):
		return status.Error(codes.InvalidArgument, err.Error())
	default:
		return status.Errorf(codes.Internal, "internal error: %v", err)
	}
}

func adminToProto(a *models.PlatformAdmin) *pb.AdminUser {
	if a == nil {
		return nil
	}
	return &pb.AdminUser{
		Id:              a.ID,
		Email:           a.Email,
		FullName:        a.FullName,
		Roles:           a.Roles,
		CreatedAtUnixMs: a.CreatedAt.UnixMilli(),
		UpdatedAtUnixMs: a.UpdatedAt.UnixMilli(),
		Disabled:        a.Disabled,
	}
}

// actorFromCtx returns the UserID claimed on the admin token stored in ctx,
// or "" if none is present. Populated by the unary interceptor that parses
// incoming auth metadata.
func actorFromCtx(ctx context.Context) string {
	if v, ok := ctx.Value(ctxActorKey{}).(string); ok {
		return v
	}
	return ""
}

type ctxActorKey struct{}

// WithActor attaches actor id to ctx; used by the server interceptor.
func WithActor(ctx context.Context, actor string) context.Context {
	return context.WithValue(ctx, ctxActorKey{}, actor)
}
