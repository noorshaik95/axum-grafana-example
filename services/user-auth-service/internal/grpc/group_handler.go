package grpc

import (
	"context"
	"strings"

	pb "slate/services/user-auth-service/api/proto"
	"slate/services/user-auth-service/internal/models"

	"slate/libs/common-go/tracing"

	"go.opentelemetry.io/otel/attribute"
	otelcodes "go.opentelemetry.io/otel/codes"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// modelGroupToProto converts a models.UserGroup to a proto Group message.
func modelGroupToProto(g *models.UserGroup) *pb.Group {
	return &pb.Group{
		Id:             g.ID,
		Name:           g.Name,
		Description:    g.Description,
		OrganizationId: g.OrganizationID,
		IsActive:       g.IsActive,
		CreatedAt:      timestamppb.New(g.CreatedAt),
		UpdatedAt:      timestamppb.New(g.UpdatedAt),
		CreatedBy:      g.CreatedBy,
	}
}

// modelMemberToProto converts a models.GroupMember to a proto GroupMember message.
func modelMemberToProto(m *models.GroupMember) *pb.GroupMember {
	return &pb.GroupMember{
		GroupId:  m.GroupID,
		UserId:   m.UserID,
		Role:     m.Role,
		JoinedAt: timestamppb.New(m.JoinedAt),
	}
}

// CreateGroup creates a new user group.
func (s *UserServiceServer) CreateGroup(ctx context.Context, req *pb.CreateGroupRequest) (*pb.GroupResponse, error) {
	ctx, span := tracing.StartSpan(ctx, "create_group_handler",
		attribute.String("group_name", req.GetName()))
	defer span.End()

	if strings.TrimSpace(req.GetName()) == "" {
		return nil, status.Error(codes.InvalidArgument, "group name is required")
	}

	group := models.NewUserGroup(
		req.GetName(),
		req.GetDescription(),
		req.GetOrganizationId(),
		req.GetCreatedBy(),
	)

	if err := s.groupRepo.CreateGroup(ctx, group); err != nil {
		span.SetStatus(otelcodes.Error, err.Error())
		return nil, status.Errorf(codes.Internal, "failed to create group: %v", err)
	}

	return &pb.GroupResponse{Group: modelGroupToProto(group)}, nil
}

// GetGroup retrieves a group by ID.
func (s *UserServiceServer) GetGroup(ctx context.Context, req *pb.GetGroupRequest) (*pb.GroupResponse, error) {
	ctx, span := tracing.StartSpan(ctx, "get_group_handler",
		attribute.String("group_id", req.GetGroupId()))
	defer span.End()

	if req.GetGroupId() == "" {
		return nil, status.Error(codes.InvalidArgument, "group_id is required")
	}

	group, err := s.groupRepo.GetGroupByID(ctx, req.GetGroupId())
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return nil, status.Error(codes.NotFound, "group not found")
		}
		span.SetStatus(otelcodes.Error, err.Error())
		return nil, status.Errorf(codes.Internal, "failed to get group: %v", err)
	}

	return &pb.GroupResponse{Group: modelGroupToProto(group)}, nil
}

// UpdateGroup updates an existing group.
func (s *UserServiceServer) UpdateGroup(ctx context.Context, req *pb.UpdateGroupRequest) (*pb.GroupResponse, error) {
	ctx, span := tracing.StartSpan(ctx, "update_group_handler",
		attribute.String("group_id", req.GetGroupId()))
	defer span.End()

	if req.GetGroupId() == "" {
		return nil, status.Error(codes.InvalidArgument, "group_id is required")
	}

	group, err := s.groupRepo.GetGroupByID(ctx, req.GetGroupId())
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return nil, status.Error(codes.NotFound, "group not found")
		}
		span.SetStatus(otelcodes.Error, err.Error())
		return nil, status.Errorf(codes.Internal, "failed to get group: %v", err)
	}

	if req.Name != nil {
		group.Name = *req.Name
	}
	if req.Description != nil {
		group.Description = *req.Description
	}
	if req.IsActive != nil {
		group.IsActive = *req.IsActive
	}

	if err := s.groupRepo.UpdateGroup(ctx, group); err != nil {
		span.SetStatus(otelcodes.Error, err.Error())
		return nil, status.Errorf(codes.Internal, "failed to update group: %v", err)
	}

	return &pb.GroupResponse{Group: modelGroupToProto(group)}, nil
}

// DeleteGroup soft-deletes a group.
func (s *UserServiceServer) DeleteGroup(ctx context.Context, req *pb.DeleteGroupRequest) (*emptypb.Empty, error) {
	ctx, span := tracing.StartSpan(ctx, "delete_group_handler",
		attribute.String("group_id", req.GetGroupId()))
	defer span.End()

	if req.GetGroupId() == "" {
		return nil, status.Error(codes.InvalidArgument, "group_id is required")
	}

	if err := s.groupRepo.DeleteGroup(ctx, req.GetGroupId()); err != nil {
		if strings.Contains(err.Error(), "not found") {
			return nil, status.Error(codes.NotFound, "group not found")
		}
		span.SetStatus(otelcodes.Error, err.Error())
		return nil, status.Errorf(codes.Internal, "failed to delete group: %v", err)
	}

	return &emptypb.Empty{}, nil
}

// ListGroups lists groups with pagination.
func (s *UserServiceServer) ListGroups(ctx context.Context, req *pb.ListGroupsRequest) (*pb.ListGroupsResponse, error) {
	ctx, span := tracing.StartSpan(ctx, "list_groups_handler",
		attribute.String("organization_id", req.GetOrganizationId()))
	defer span.End()

	page := int(req.GetPage())
	if page < 1 {
		page = 1
	}
	pageSize := int(req.GetPageSize())
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	groups, total, err := s.groupRepo.ListGroups(ctx, req.GetOrganizationId(), page, pageSize)
	if err != nil {
		span.SetStatus(otelcodes.Error, err.Error())
		return nil, status.Errorf(codes.Internal, "failed to list groups: %v", err)
	}

	pbGroups := make([]*pb.Group, len(groups))
	for i, g := range groups {
		pbGroups[i] = modelGroupToProto(g)
	}

	return &pb.ListGroupsResponse{
		Groups: pbGroups,
		Total:  int32(total),
	}, nil
}

// AddGroupMember adds a user to a group.
func (s *UserServiceServer) AddGroupMember(ctx context.Context, req *pb.AddGroupMemberRequest) (*emptypb.Empty, error) {
	ctx, span := tracing.StartSpan(ctx, "add_group_member_handler",
		attribute.String("group_id", req.GetGroupId()),
		attribute.String("user_id", req.GetUserId()))
	defer span.End()

	if req.GetGroupId() == "" || req.GetUserId() == "" {
		return nil, status.Error(codes.InvalidArgument, "group_id and user_id are required")
	}

	role := req.GetRole()
	if role == "" {
		role = "member"
	}

	member := models.NewGroupMember(req.GetGroupId(), req.GetUserId(), role)
	if err := s.groupRepo.AddMember(ctx, member); err != nil {
		span.SetStatus(otelcodes.Error, err.Error())
		return nil, status.Errorf(codes.Internal, "failed to add group member: %v", err)
	}

	return &emptypb.Empty{}, nil
}

// RemoveGroupMember removes a user from a group.
func (s *UserServiceServer) RemoveGroupMember(ctx context.Context, req *pb.RemoveGroupMemberRequest) (*emptypb.Empty, error) {
	ctx, span := tracing.StartSpan(ctx, "remove_group_member_handler",
		attribute.String("group_id", req.GetGroupId()),
		attribute.String("user_id", req.GetUserId()))
	defer span.End()

	if req.GetGroupId() == "" || req.GetUserId() == "" {
		return nil, status.Error(codes.InvalidArgument, "group_id and user_id are required")
	}

	if err := s.groupRepo.RemoveMember(ctx, req.GetGroupId(), req.GetUserId()); err != nil {
		if strings.Contains(err.Error(), "not found") {
			return nil, status.Error(codes.NotFound, "member not found in group")
		}
		span.SetStatus(otelcodes.Error, err.Error())
		return nil, status.Errorf(codes.Internal, "failed to remove group member: %v", err)
	}

	return &emptypb.Empty{}, nil
}

// GetGroupMembers retrieves all members of a group.
func (s *UserServiceServer) GetGroupMembers(ctx context.Context, req *pb.GetGroupMembersRequest) (*pb.GetGroupMembersResponse, error) {
	ctx, span := tracing.StartSpan(ctx, "get_group_members_handler",
		attribute.String("group_id", req.GetGroupId()))
	defer span.End()

	if req.GetGroupId() == "" {
		return nil, status.Error(codes.InvalidArgument, "group_id is required")
	}

	members, err := s.groupRepo.GetGroupMembers(ctx, req.GetGroupId())
	if err != nil {
		span.SetStatus(otelcodes.Error, err.Error())
		return nil, status.Errorf(codes.Internal, "failed to get group members: %v", err)
	}

	pbMembers := make([]*pb.GroupMember, len(members))
	for i, m := range members {
		pbMembers[i] = modelMemberToProto(m)
	}

	return &pb.GetGroupMembersResponse{Members: pbMembers}, nil
}

// GetUserGroups retrieves all groups a user belongs to.
func (s *UserServiceServer) GetUserGroups(ctx context.Context, req *pb.GetUserGroupsRequest) (*pb.GetUserGroupsResponse, error) {
	ctx, span := tracing.StartSpan(ctx, "get_user_groups_handler",
		attribute.String("user_id", req.GetUserId()))
	defer span.End()

	if req.GetUserId() == "" {
		return nil, status.Error(codes.InvalidArgument, "user_id is required")
	}

	groups, err := s.groupRepo.GetUserGroups(ctx, req.GetUserId())
	if err != nil {
		span.SetStatus(otelcodes.Error, err.Error())
		return nil, status.Errorf(codes.Internal, "failed to get user groups: %v", err)
	}

	pbGroups := make([]*pb.Group, len(groups))
	for i, g := range groups {
		pbGroups[i] = modelGroupToProto(g)
	}

	return &pb.GetUserGroupsResponse{Groups: pbGroups}, nil
}
