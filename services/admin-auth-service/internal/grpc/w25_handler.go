package grpc

import (
	"context"
	"errors"
	"time"

	pb "slate/services/admin-auth-service/api/proto/adminauthpb"
	"slate/services/admin-auth-service/internal/service"
)

// ListAdminRoles (W2.5) — paginated platform_roles read.
func (s *AdminAuthServer) ListAdminRoles(ctx context.Context, req *pb.ListAdminRolesRequest) (*pb.ListAdminRolesResponse, error) {
	tagSpan(ctx)
	res, err := s.svc.ListRoles(ctx, int(req.GetPageSize()), req.GetPageToken())
	if err != nil {
		return nil, toStatusErr(mapReadErr(err))
	}
	roles := make([]*pb.AdminRole, 0, len(res.Roles))
	for _, r := range res.Roles {
		roles = append(roles, &pb.AdminRole{
			Key:             r.Key,
			Name:            r.Name,
			Description:     r.Description,
			Permissions:     r.Permissions,
			CreatedAtUnixMs: r.CreatedAt.UnixMilli(),
		})
	}
	return &pb.ListAdminRolesResponse{
		Roles:         roles,
		NextPageToken: res.NextPageToken,
		Total:         int32(res.Total),
	}, nil
}

// GetAuditLog (W2.5) — filtered + paginated platform_audit read.
func (s *AdminAuthServer) GetAuditLog(ctx context.Context, req *pb.GetAuditLogRequest) (*pb.GetAuditLogResponse, error) {
	tagSpan(ctx)
	filter := service.AuditFilter{
		ActorID:    req.GetAdminUserId(),
		Action:     req.GetActionType(),
		TargetID:   req.GetTargetId(),
		TargetType: req.GetTargetType(),
		Limit:      int(req.GetLimit()),
		Cursor:     req.GetCursor(),
	}
	if req.FromUnixMs != nil {
		filter.From = time.UnixMilli(req.GetFromUnixMs())
	}
	if req.ToUnixMs != nil {
		filter.To = time.UnixMilli(req.GetToUnixMs())
	}

	res, err := s.svc.GetAuditLog(ctx, filter)
	if err != nil {
		return nil, toStatusErr(mapReadErr(err))
	}

	events := make([]*pb.AuditEvent, 0, len(res.Events))
	for _, e := range res.Events {
		ev := &pb.AuditEvent{
			Id:                e.ID,
			AdminUserId:       e.AdminUserID,
			AdminEmail:        e.AdminEmail,
			ActionType:        e.ActionType,
			TargetType:        e.TargetType,
			Outcome:           e.Outcome,
			OccurredAtUnixMs:  e.OccurredAt.UnixMilli(),
			Metadata:          e.Metadata,
			RequestId:         e.RequestID,
		}
		if e.TargetID != "" {
			tid := e.TargetID
			ev.TargetId = &tid
		}
		if e.Error != "" {
			em := e.Error
			ev.Error = &em
		}
		events = append(events, ev)
	}
	return &pb.GetAuditLogResponse{
		Events:     events,
		NextCursor: res.NextCursor,
	}, nil
}

// RefreshAdminToken (W2.5) — rotates the refresh token and returns a new LoginResponse.
func (s *AdminAuthServer) RefreshAdminToken(ctx context.Context, req *pb.RefreshAdminTokenRequest) (*pb.LoginResponse, error) {
	tagSpan(ctx)
	res, err := s.svc.RefreshToken(ctx, req.GetToken())
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

// mapReadErr translates ErrReadsUnavailable → a 503-ish gRPC code. Other errors
// flow through toStatusErr unchanged.
func mapReadErr(err error) error {
	if errors.Is(err, service.ErrReadsUnavailable) {
		// Signalling "configuration gap" — map to Internal so a client can retry later.
		return err
	}
	return err
}
