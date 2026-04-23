// Package broadcast centralizes the W15 platform broadcast fan-out so both the
// HTTP handler and the gRPC endpoint share a single implementation.
package broadcast

import (
	"context"
	"fmt"
	"time"

	"slate/services/email-service/internal/email"
	"slate/services/email-service/internal/kafka"
	"slate/services/email-service/internal/tenants"

	"github.com/google/uuid"
)

// Targets specifies the broadcast audience. Exactly one of AllTenants or TenantIDs must be set.
type Targets struct {
	AllTenants bool
	TenantIDs  []string
}

// Request is the parsed broadcast payload regardless of transport.
type Request struct {
	AuthorID string
	Message  string
	Channels []string
	Targets  Targets
}

// Result summarizes fan-out counts.
type Result struct {
	BroadcastID string
	TenantIDs   []string
	EmailsSent  int
	InAppSent   int
	SentAt      time.Time
}

// ValidationError marks a caller input error so callers can distinguish
// 400 vs. 500 without string-matching.
type ValidationError struct{ Msg string }

func (e *ValidationError) Error() string { return e.Msg }

// Service runs the fan-out logic.
type Service struct {
	Producer *kafka.Producer
	Sender   email.Sender
	Tenants  tenants.Resolver
}

// New constructs a Service. Any dependency may be nil — the corresponding
// channel becomes a no-op for dev.
func New(p *kafka.Producer, s email.Sender, t tenants.Resolver) *Service {
	return &Service{Producer: p, Sender: s, Tenants: t}
}

// Dispatch validates the request and fans out across the requested channels.
func (s *Service) Dispatch(ctx context.Context, req Request) (*Result, error) {
	if req.Message == "" {
		return nil, &ValidationError{Msg: "message is required"}
	}
	if !req.Targets.AllTenants && len(req.Targets.TenantIDs) == 0 {
		return nil, &ValidationError{Msg: "targets.all_tenants or targets.tenant_ids required"}
	}
	if req.Targets.AllTenants && len(req.Targets.TenantIDs) > 0 {
		return nil, &ValidationError{Msg: "targets.all_tenants and targets.tenant_ids are mutually exclusive"}
	}
	if len(req.Channels) == 0 {
		return nil, &ValidationError{Msg: "at least one channel required"}
	}

	tenantIDs := req.Targets.TenantIDs
	if req.Targets.AllTenants {
		if s.Tenants == nil {
			return nil, fmt.Errorf("tenant resolver not configured")
		}
		all, err := s.Tenants.AllTenantIDs(ctx)
		if err != nil {
			return nil, fmt.Errorf("resolve tenants: %w", err)
		}
		tenantIDs = all
	}

	res := &Result{
		BroadcastID: uuid.New().String(),
		TenantIDs:   tenantIDs,
		SentAt:      time.Now().UTC(),
	}

	for _, ch := range req.Channels {
		switch ch {
		case "in_app":
			if s.Producer == nil {
				continue
			}
			var publishTenants []string
			if !req.Targets.AllTenants {
				publishTenants = tenantIDs
			}
			if err := s.Producer.PublishBroadcastInApp(ctx, res.BroadcastID, req.AuthorID, req.Message, publishTenants); err != nil {
				return nil, fmt.Errorf("publish in_app: %w", err)
			}
			if req.Targets.AllTenants {
				res.InAppSent = 1
			} else {
				res.InAppSent = len(tenantIDs)
			}
		case "email_admins":
			if s.Tenants == nil || s.Sender == nil {
				continue
			}
			for _, tid := range tenantIDs {
				admin, err := s.Tenants.PrimaryAdmin(ctx, tid)
				if err != nil {
					return nil, fmt.Errorf("resolve admin for tenant %s: %w", tid, err)
				}
				if admin == nil {
					continue
				}
				payload := email.Payload{
					To:       admin.Email,
					Subject:  "Platform broadcast",
					Body:     req.Message,
					Template: "platform_broadcast",
					TenantID: tid,
				}
				if err := s.Sender.Send(ctx, payload); err != nil {
					return nil, fmt.Errorf("send email for tenant %s: %w", tid, err)
				}
				res.EmailsSent++
			}
		default:
			return nil, &ValidationError{Msg: "unknown channel: " + ch}
		}
	}

	return res, nil
}
