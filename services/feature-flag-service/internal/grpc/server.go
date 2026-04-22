package grpc

import (
	"context"
	"errors"
	"time"

	"go.opentelemetry.io/otel/trace"
	grpccodes "google.golang.org/grpc/codes"
	grpcstatus "google.golang.org/grpc/status"

	pb "slate/services/feature-flag-service/api/proto"
	"slate/libs/common-go/tracing"
	"slate/services/feature-flag-service/internal/cache"
	"slate/services/feature-flag-service/internal/repository"
	"slate/services/feature-flag-service/internal/service"
)

// Server implements pb.FlagServiceServer.
type Server struct {
	pb.UnimplementedFlagServiceServer
	store     repository.Store
	evaluator *service.Evaluator
	cache     cache.Cache
}

// NewServer wires together the persistence, evaluator and cache dependencies.
func NewServer(store repository.Store, evaluator *service.Evaluator, cache cache.Cache) *Server {
	return &Server{store: store, evaluator: evaluator, cache: cache}
}

// EvaluateFlags is the gateway-hot-path RPC. Cache hit → return cached map.
// Cache miss → load from store, evaluate, write-through, return.
func (s *Server) EvaluateFlags(ctx context.Context, req *pb.EvaluateFlagsRequest) (*pb.EvaluateFlagsResponse, error) {
	tracing.TagSpanWithCorrelation(ctx, trace.SpanFromContext(ctx))

	if req.GetTenantId() == "" {
		return nil, grpcstatus.Error(grpccodes.InvalidArgument, "tenant_id is required")
	}

	if flags, hit, err := s.cache.Get(ctx, req.GetTenantId(), req.GetUserId(), req.GetRoles()); err == nil && hit {
		return &pb.EvaluateFlagsResponse{
			Flags:             flags,
			EvaluatedAtUnixMs: time.Now().UnixMilli(),
		}, nil
	}

	flags, err := s.store.List(ctx)
	if err != nil {
		return nil, grpcstatus.Errorf(grpccodes.Internal, "list flags: %v", err)
	}

	result := s.evaluator.EvaluateAll(flags, service.EvalContext{
		TenantID:   req.GetTenantId(),
		TenantSlug: req.GetTenantSlug(),
		UserID:     req.GetUserId(),
		Roles:      req.GetRoles(),
	})

	_ = s.cache.Set(ctx, req.GetTenantId(), req.GetUserId(), req.GetRoles(), result)

	return &pb.EvaluateFlagsResponse{
		Flags:             result,
		EvaluatedAtUnixMs: time.Now().UnixMilli(),
	}, nil
}

// GetFlag returns a single flag by key.
func (s *Server) GetFlag(ctx context.Context, req *pb.GetFlagRequest) (*pb.Flag, error) {
	tracing.TagSpanWithCorrelation(ctx, trace.SpanFromContext(ctx))
	if req.GetKey() == "" {
		return nil, grpcstatus.Error(grpccodes.InvalidArgument, "key is required")
	}
	flag, err := s.store.GetByKey(ctx, req.GetKey())
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, grpcstatus.Error(grpccodes.NotFound, "flag not found")
		}
		return nil, grpcstatus.Errorf(grpccodes.Internal, "get flag: %v", err)
	}
	return toProtoFlag(*flag), nil
}

// ListFlags returns the full admin view of every flag and its rules.
func (s *Server) ListFlags(ctx context.Context, req *pb.ListFlagsRequest) (*pb.ListFlagsResponse, error) {
	tracing.TagSpanWithCorrelation(ctx, trace.SpanFromContext(ctx))

	flags, err := s.store.List(ctx)
	if err != nil {
		return nil, grpcstatus.Errorf(grpccodes.Internal, "list flags: %v", err)
	}
	resp := &pb.ListFlagsResponse{Flags: make([]*pb.Flag, 0, len(flags))}
	for _, f := range flags {
		resp.Flags = append(resp.Flags, toProtoFlag(f))
	}
	return resp, nil
}

// UpdateFlag upserts a flag, rewrites its rules atomically, then invalidates
// the cached evaluations for the caller's tenant.
func (s *Server) UpdateFlag(ctx context.Context, req *pb.UpdateFlagRequest) (*pb.Flag, error) {
	tracing.TagSpanWithCorrelation(ctx, trace.SpanFromContext(ctx))

	if req.GetKey() == "" {
		return nil, grpcstatus.Error(grpccodes.InvalidArgument, "key is required")
	}

	rules := make([]repository.Rule, 0, len(req.GetTargets()))
	for _, t := range req.GetTargets() {
		rules = append(rules, repository.Rule{
			RuleType:  t.GetRuleType(),
			RuleValue: normalizeJSON(t.GetRuleValueJson()),
		})
	}

	flag, err := s.store.Upsert(ctx, req.GetKey(), req.GetState(), rules)
	if err != nil {
		return nil, grpcstatus.Errorf(grpccodes.Internal, "upsert flag: %v", err)
	}

	// Tenant to invalidate comes from the x-tenant-slug / correlation. If
	// absent (platform-wide edit), clear everything.
	if err := s.cache.InvalidateTenant(ctx, ""); err != nil {
		// Cache flush failure shouldn't fail the write — log via span event and continue.
		trace.SpanFromContext(ctx).AddEvent("cache_invalidate_failed")
	}

	return toProtoFlag(*flag), nil
}

// DeleteFlag removes a flag and invalidates all cached evaluations.
func (s *Server) DeleteFlag(ctx context.Context, req *pb.DeleteFlagRequest) (*pb.DeleteFlagResponse, error) {
	tracing.TagSpanWithCorrelation(ctx, trace.SpanFromContext(ctx))

	if req.GetKey() == "" {
		return nil, grpcstatus.Error(grpccodes.InvalidArgument, "key is required")
	}

	if err := s.store.Delete(ctx, req.GetKey()); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, grpcstatus.Error(grpccodes.NotFound, "flag not found")
		}
		return nil, grpcstatus.Errorf(grpccodes.Internal, "delete flag: %v", err)
	}

	if err := s.cache.InvalidateTenant(ctx, ""); err != nil {
		trace.SpanFromContext(ctx).AddEvent("cache_invalidate_failed")
	}
	return &pb.DeleteFlagResponse{Deleted: true}, nil
}

func toProtoFlag(f repository.Flag) *pb.Flag {
	targets := make([]*pb.TargetRule, 0, len(f.Rules))
	for _, r := range f.Rules {
		targets = append(targets, &pb.TargetRule{
			RuleType:      r.RuleType,
			RuleValueJson: r.RuleValue,
		})
	}
	return &pb.Flag{
		Key:              f.Key,
		Description:      f.Description,
		State:            f.Enabled,
		Targets:          targets,
		UpdatedAtUnixMs:  f.UpdatedAt.UnixMilli(),
	}
}

// normalizeJSON returns a sensible default (`{}`) when the caller omits the
// rule_value_json field — the DB column is NOT NULL JSONB.
func normalizeJSON(s string) string {
	if s == "" {
		return "{}"
	}
	return s
}
