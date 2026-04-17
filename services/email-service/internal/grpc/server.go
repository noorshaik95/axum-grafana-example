package grpc

import (
	"context"
	"fmt"

	"slate/services/email-service/internal/repository"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	pb "slate/services/email-service/api/proto"
)

type EmailServer struct {
	pb.UnimplementedMessagingServiceServer
	repo *repository.MessageRepository
}

func NewEmailServer(repo *repository.MessageRepository) *EmailServer {
	return &EmailServer{repo: repo}
}

func (s *EmailServer) SendMessage(ctx context.Context, req *pb.SendMessageRequest) (*pb.SendMessageResponse, error) {
	if req.TenantId == "" || req.FromUserId == "" || req.Body == "" || len(req.RecipientIds) == 0 {
		return nil, status.Error(codes.InvalidArgument, "tenant_id, from_user_id, body, and recipient_ids are required")
	}

	msg, err := s.repo.SendMessage(ctx, req.TenantId, req.FromUserId, req.Subject, req.Body, req.RecipientIds)
	if err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("send message: %v", err))
	}

	return &pb.SendMessageResponse{
		MessageId: msg.ID,
		ThreadId:  msg.ThreadID,
		CreatedAt: timestamppb.New(msg.CreatedAt),
	}, nil
}

func (s *EmailServer) GetInbox(ctx context.Context, req *pb.GetInboxRequest) (*pb.GetInboxResponse, error) {
	if req.TenantId == "" || req.UserId == "" {
		return nil, status.Error(codes.InvalidArgument, "tenant_id and user_id are required")
	}

	result, err := s.repo.GetInbox(ctx, req.TenantId, req.UserId, req.Cursor, int(req.Limit))
	if err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("get inbox: %v", err))
	}

	var pbMessages []*pb.InboxMessage
	for _, m := range result.Messages {
		pbMsg := &pb.InboxMessage{
			Id:         m.ID,
			ThreadId:   m.ThreadID,
			FromUserId: m.FromUserID,
			Body:       m.Body,
			IsRead:     m.IsRead,
			CreatedAt:  timestamppb.New(m.CreatedAt),
		}
		if m.Subject != nil {
			pbMsg.Subject = *m.Subject
		}
		pbMessages = append(pbMessages, pbMsg)
	}

	return &pb.GetInboxResponse{
		Messages:   pbMessages,
		NextCursor: result.NextCursor,
		HasMore:    result.HasMore,
	}, nil
}

func (s *EmailServer) MarkRead(ctx context.Context, req *pb.MarkReadRequest) (*pb.MarkReadResponse, error) {
	if req.MessageId == "" || req.UserId == "" {
		return nil, status.Error(codes.InvalidArgument, "message_id and user_id are required")
	}

	if err := s.repo.MarkRead(ctx, req.MessageId, req.UserId); err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("mark read: %v", err))
	}

	return &pb.MarkReadResponse{Success: true}, nil
}

func (s *EmailServer) GetThread(ctx context.Context, req *pb.GetThreadRequest) (*pb.GetThreadResponse, error) {
	if req.ThreadId == "" {
		return nil, status.Error(codes.InvalidArgument, "thread_id is required")
	}

	messages, err := s.repo.GetThread(ctx, req.ThreadId)
	if err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("get thread: %v", err))
	}

	var pbMessages []*pb.ThreadMessage
	for _, m := range messages {
		pbMsg := &pb.ThreadMessage{
			Id:         m.ID,
			FromUserId: m.FromUserID,
			Body:       m.Body,
			CreatedAt:  timestamppb.New(m.CreatedAt),
		}
		if m.Subject != nil {
			pbMsg.Subject = *m.Subject
		}
		if m.ParentID != nil {
			pbMsg.ParentId = *m.ParentID
		}
		pbMessages = append(pbMessages, pbMsg)
	}

	return &pb.GetThreadResponse{Messages: pbMessages}, nil
}

func (s *EmailServer) GetUnreadCount(ctx context.Context, req *pb.GetUnreadCountRequest) (*pb.GetUnreadCountResponse, error) {
	if req.TenantId == "" || req.UserId == "" {
		return nil, status.Error(codes.InvalidArgument, "tenant_id and user_id are required")
	}

	count, err := s.repo.GetUnreadCount(ctx, req.TenantId, req.UserId)
	if err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("get unread count: %v", err))
	}

	return &pb.GetUnreadCountResponse{Count: int32(count)}, nil
}
