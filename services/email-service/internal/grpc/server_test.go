package grpc

import (
	"context"
	"testing"

	"slate/services/email-service/internal/broadcast"
	"slate/services/email-service/internal/email"
	"slate/services/email-service/internal/tenants"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	pb "slate/services/email-service/api/proto"
)

func TestSendBroadcast_UnimplementedWithoutService(t *testing.T) {
	s := NewEmailServer(nil)
	_, err := s.SendBroadcast(context.Background(), &pb.SendBroadcastRequest{Targets: &pb.BroadcastTargets{AllTenants: true}})
	if status.Code(err) != codes.Unimplemented {
		t.Errorf("code = %v, want Unimplemented", status.Code(err))
	}
}

func TestSendBroadcast_RequiresTargets(t *testing.T) {
	svc := broadcast.New(nil, &email.RecordingSender{}, tenants.NewStaticResolver(nil))
	s := NewEmailServerWithBroadcast(nil, svc)
	_, err := s.SendBroadcast(context.Background(), &pb.SendBroadcastRequest{Message: "hi", Channels: []string{"in_app"}})
	if status.Code(err) != codes.InvalidArgument {
		t.Errorf("code = %v, want InvalidArgument", status.Code(err))
	}
}

func TestSendBroadcast_ValidationBubbles(t *testing.T) {
	svc := broadcast.New(nil, &email.RecordingSender{}, tenants.NewStaticResolver(nil))
	s := NewEmailServerWithBroadcast(nil, svc)
	_, err := s.SendBroadcast(context.Background(), &pb.SendBroadcastRequest{
		Targets:  &pb.BroadcastTargets{TenantIds: []string{"t1"}},
		Channels: []string{"sms"}, // unknown
		Message:  "x",
	})
	if status.Code(err) != codes.InvalidArgument {
		t.Errorf("code = %v, want InvalidArgument", status.Code(err))
	}
}

func TestSendBroadcast_EmailFanOut(t *testing.T) {
	sender := &email.RecordingSender{}
	resolver := tenants.NewStaticResolver([]tenants.Admin{
		{TenantID: "t1", Email: "a@e.com"},
		{TenantID: "t2", Email: "b@e.com"},
	})
	svc := broadcast.New(nil, sender, resolver)
	s := NewEmailServerWithBroadcast(nil, svc)

	resp, err := s.SendBroadcast(context.Background(), &pb.SendBroadcastRequest{
		AuthorId: "admin-1",
		Message:  "restart",
		Channels: []string{"email_admins"},
		Targets:  &pb.BroadcastTargets{TenantIds: []string{"t1", "t2"}},
	})
	if err != nil {
		t.Fatalf("SendBroadcast: %v", err)
	}
	if resp.EmailsSent != 2 {
		t.Errorf("EmailsSent = %d, want 2", resp.EmailsSent)
	}
	if resp.BroadcastId == "" {
		t.Errorf("BroadcastId empty")
	}
	if len(sender.Sent) != 2 {
		t.Errorf("recorded = %d, want 2", len(sender.Sent))
	}
}
