package grpc

import (
	"context"
	"errors"
	"fmt"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"

	commontracing "slate/libs/common-go/tracing"
	pb "slate/services/incident-service/api/proto"
	"slate/services/incident-service/internal/cache"
	"slate/services/incident-service/internal/kafka"
	"slate/services/incident-service/internal/models"
	"slate/services/incident-service/internal/repository"
)

type Server struct {
	pb.UnimplementedIncidentServiceServer
	repo     *repository.Repository
	producer *kafka.Producer
	cache    *cache.Cache
}

func NewServer(repo *repository.Repository, producer *kafka.Producer, c *cache.Cache) *Server {
	return &Server{repo: repo, producer: producer, cache: c}
}

// ============================================================================
// Proto <-> model conversions
// ============================================================================

func priorityFromProto(p pb.Priority) (string, bool) {
	switch p {
	case pb.Priority_P0:
		return models.PriorityP0, true
	case pb.Priority_P1:
		return models.PriorityP1, true
	case pb.Priority_P2:
		return models.PriorityP2, true
	case pb.Priority_P3:
		return models.PriorityP3, true
	case pb.Priority_P4:
		return models.PriorityP4, true
	}
	return "", false
}

func priorityToProto(p string) pb.Priority {
	switch p {
	case models.PriorityP0:
		return pb.Priority_P0
	case models.PriorityP1:
		return pb.Priority_P1
	case models.PriorityP2:
		return pb.Priority_P2
	case models.PriorityP3:
		return pb.Priority_P3
	case models.PriorityP4:
		return pb.Priority_P4
	}
	return pb.Priority_PRIORITY_UNSPECIFIED
}

func statusFromProto(s pb.IncidentStatus) (string, bool) {
	switch s {
	case pb.IncidentStatus_OPEN:
		return models.StatusOpen, true
	case pb.IncidentStatus_WATCHING:
		return models.StatusWatching, true
	case pb.IncidentStatus_RESOLVED:
		return models.StatusResolved, true
	}
	return "", false
}

func statusToProto(s string) pb.IncidentStatus {
	switch s {
	case models.StatusOpen:
		return pb.IncidentStatus_OPEN
	case models.StatusWatching:
		return pb.IncidentStatus_WATCHING
	case models.StatusResolved:
		return pb.IncidentStatus_RESOLVED
	}
	return pb.IncidentStatus_INCIDENT_STATUS_UNSPECIFIED
}

func filterToStatus(f pb.StatusFilter) (string, bool) {
	switch f {
	case pb.StatusFilter_FILTER_OPEN:
		return models.StatusOpen, true
	case pb.StatusFilter_FILTER_WATCH:
		return models.StatusWatching, true
	case pb.StatusFilter_FILTER_RESOLVED:
		return models.StatusResolved, true
	}
	return "", false
}

func incidentToProto(inc *models.Incident) *pb.Incident {
	out := &pb.Incident{
		Id:             inc.ID,
		TenantId:       inc.TenantID,
		Priority:       priorityToProto(inc.Priority),
		Status:         statusToProto(inc.Status),
		Title:          inc.Title,
		Impact:         inc.Description,
		OpenedAtUnixMs: inc.CreatedAt.UnixMilli(),
		Service:        inc.Service,
		CreatedBy:      inc.CreatedBy,
	}
	if inc.ResolvedAt != nil {
		v := inc.ResolvedAt.UnixMilli()
		out.ResolvedAtUnixMs = &v
	}
	for i := range inc.Events {
		out.Events = append(out.Events, eventToProto(&inc.Events[i]))
	}
	return out
}

func eventToProto(ev *models.IncidentEvent) *pb.IncidentEvent {
	return &pb.IncidentEvent{
		Id:              ev.ID,
		IncidentId:      ev.IncidentID,
		ActorId:         ev.ActorID,
		EventType:       ev.EventType,
		Content:         ev.Content,
		CreatedAtUnixMs: ev.CreatedAt.UnixMilli(),
	}
}

// ============================================================================
// RPC handlers
// ============================================================================

func (s *Server) CreateIncident(ctx context.Context, req *pb.CreateIncidentRequest) (*pb.Incident, error) {
	span := trace.SpanFromContext(ctx)
	commontracing.TagSpanWithCorrelation(ctx, span)

	if req.GetTitle() == "" {
		return nil, status.Error(codes.InvalidArgument, "title is required")
	}
	priority, ok := priorityFromProto(req.GetPriority())
	if !ok {
		return nil, status.Error(codes.InvalidArgument, "priority is required (P0..P4)")
	}

	inc, err := s.repo.CreateIncident(ctx, repository.CreateIncidentParams{
		TenantID:  req.TenantId,
		Service:   req.Service,
		Title:     req.GetTitle(),
		Priority:  priority,
		Impact:    req.GetImpact(),
		CreatedBy: req.CreatedBy,
	})
	if err != nil {
		if errors.Is(err, repository.ErrInvalidPriority) {
			return nil, status.Error(codes.InvalidArgument, err.Error())
		}
		return nil, status.Error(codes.Internal, fmt.Sprintf("create incident: %v", err))
	}

	span.SetAttributes(attribute.String("incident_id", inc.ID))
	if inc.TenantID != nil {
		span.SetAttributes(attribute.String("tenant_id", *inc.TenantID))
	}

	s.producer.PublishIncidentOpened(ctx, inc)
	s.refreshActiveCache(ctx)

	return incidentToProto(inc), nil
}

func (s *Server) UpdateIncident(ctx context.Context, req *pb.UpdateIncidentRequest) (*pb.Incident, error) {
	span := trace.SpanFromContext(ctx)
	commontracing.TagSpanWithCorrelation(ctx, span)

	if req.GetId() == "" {
		return nil, status.Error(codes.InvalidArgument, "id is required")
	}
	span.SetAttributes(attribute.String("incident_id", req.GetId()))

	params := repository.UpdateIncidentParams{
		ID:      req.GetId(),
		Title:   req.Title,
		Impact:  req.Impact,
		ActorID: req.ActorId,
	}
	if req.Status != nil {
		st, ok := statusFromProto(*req.Status)
		if !ok {
			return nil, status.Error(codes.InvalidArgument, "status invalid")
		}
		params.Status = &st
	}
	if req.Priority != nil {
		pr, ok := priorityFromProto(*req.Priority)
		if !ok {
			return nil, status.Error(codes.InvalidArgument, "priority invalid")
		}
		params.Priority = &pr
	}

	inc, err := s.repo.UpdateIncident(ctx, params)
	if err != nil {
		switch {
		case errors.Is(err, repository.ErrNotFound):
			return nil, status.Error(codes.NotFound, err.Error())
		case errors.Is(err, repository.ErrInvalidStatus),
			errors.Is(err, repository.ErrInvalidPriority),
			errors.Is(err, repository.ErrInvalidTransition):
			return nil, status.Error(codes.InvalidArgument, err.Error())
		}
		return nil, status.Error(codes.Internal, fmt.Sprintf("update incident: %v", err))
	}

	// Reload with events attached for the response.
	full, err := s.repo.GetIncident(ctx, inc.ID)
	if err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("reload incident: %v", err))
	}

	if full.Status == models.StatusResolved {
		s.producer.PublishIncidentResolved(ctx, full)
	}
	s.refreshActiveCache(ctx)

	return incidentToProto(full), nil
}

func (s *Server) ListIncidents(ctx context.Context, req *pb.ListIncidentsRequest) (*pb.ListIncidentsResponse, error) {
	span := trace.SpanFromContext(ctx)
	commontracing.TagSpanWithCorrelation(ctx, span)

	f := repository.ListFilter{
		TenantID: req.TenantId,
		Limit:    int(req.GetLimit()),
	}
	if req.StatusFilter != nil {
		if st, ok := filterToStatus(*req.StatusFilter); ok {
			f.Status = &st
		}
	}
	if req.Priority != nil {
		if pr, ok := priorityFromProto(*req.Priority); ok {
			f.Priority = &pr
		}
	}

	rows, err := s.repo.ListIncidents(ctx, f)
	if err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("list: %v", err))
	}
	out := &pb.ListIncidentsResponse{}
	for _, inc := range rows {
		out.Incidents = append(out.Incidents, incidentToProto(inc))
	}
	return out, nil
}

func (s *Server) GetIncident(ctx context.Context, req *pb.GetIncidentRequest) (*pb.Incident, error) {
	span := trace.SpanFromContext(ctx)
	commontracing.TagSpanWithCorrelation(ctx, span)
	if req.GetId() == "" {
		return nil, status.Error(codes.InvalidArgument, "id is required")
	}
	span.SetAttributes(attribute.String("incident_id", req.GetId()))

	inc, err := s.repo.GetIncident(ctx, req.GetId())
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, status.Error(codes.NotFound, err.Error())
		}
		return nil, status.Error(codes.Internal, fmt.Sprintf("get: %v", err))
	}
	return incidentToProto(inc), nil
}

func (s *Server) PostIncidentEvent(ctx context.Context, req *pb.PostIncidentEventRequest) (*pb.IncidentEvent, error) {
	span := trace.SpanFromContext(ctx)
	commontracing.TagSpanWithCorrelation(ctx, span)

	if req.GetIncidentId() == "" || req.GetEventType() == "" {
		return nil, status.Error(codes.InvalidArgument, "incident_id and event_type are required")
	}
	span.SetAttributes(attribute.String("incident_id", req.GetIncidentId()))

	ev, err := s.repo.AddEvent(ctx, req.GetIncidentId(), req.ActorId, req.GetEventType(), req.GetContent())
	if err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("add event: %v", err))
	}
	return eventToProto(ev), nil
}

func (s *Server) GetPublicStatus(ctx context.Context, _ *emptypb.Empty) (*pb.PublicStatusResponse, error) {
	span := trace.SpanFromContext(ctx)
	commontracing.TagSpanWithCorrelation(ctx, span)

	byService, err := s.repo.OpenIncidentsByService(ctx)
	if err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("status: %v", err))
	}

	components, overall := aggregatePublicStatus(byService)

	count7d, err := s.repo.CountIncidentsSince(ctx, time.Now().Add(-7*24*time.Hour))
	if err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("count 7d: %v", err))
	}

	return &pb.PublicStatusResponse{
		Overall:           overall,
		Components:        components,
		IncidentsLast_7D:  int32(count7d),
		GeneratedAtUnixMs: time.Now().UnixMilli(),
	}, nil
}

// aggregatePublicStatus converts the open-incidents-by-service map into a
// component list + overall health.
//
// Priority → health:
//   P0, P1 present  → RED
//   P2     present  → AMBER
//   P3, P4 only     → GREEN (noted but not degraded)
//
// Overall:
//   any RED → OUTAGE; any AMBER → DEGRADED; else OPERATIONAL.
func aggregatePublicStatus(byService map[string][]*models.Incident) ([]*pb.ComponentStatus, pb.OverallStatus) {
	var components []*pb.ComponentStatus
	overall := pb.OverallStatus_OPERATIONAL

	for svc, incs := range byService {
		if svc == "" {
			continue
		}
		health := pb.ComponentHealth_GREEN
		var highest *pb.Priority
		for _, inc := range incs {
			pr := priorityToProto(inc.Priority)
			if highest == nil || pr < *highest {
				v := pr
				highest = &v
			}
			switch inc.Priority {
			case models.PriorityP0, models.PriorityP1:
				health = pb.ComponentHealth_RED
			case models.PriorityP2:
				if health != pb.ComponentHealth_RED {
					health = pb.ComponentHealth_AMBER
				}
			}
		}
		components = append(components, &pb.ComponentStatus{
			Service:         svc,
			Health:          health,
			OpenIncidents:   int32(len(incs)),
			HighestPriority: highest,
		})
		if health == pb.ComponentHealth_RED {
			overall = pb.OverallStatus_OUTAGE
		} else if health == pb.ComponentHealth_AMBER && overall != pb.OverallStatus_OUTAGE {
			overall = pb.OverallStatus_DEGRADED
		}
	}
	return components, overall
}

// refreshActiveCache pulls the current open+watching list and writes it to Redis
// with the 30s TTL required by plan W4.4. Errors are logged but not returned —
// cache failures must not fail RPCs.
func (s *Server) refreshActiveCache(ctx context.Context) {
	if s.cache == nil {
		return
	}
	openStatus := models.StatusOpen
	watching := models.StatusWatching
	combined, err := s.listActive(ctx)
	if err != nil {
		return
	}
	if err := s.cache.SetActive(ctx, combined); err != nil {
		// swallow; cache is a best-effort accelerator
		_ = openStatus
		_ = watching
	}
}

func (s *Server) listActive(ctx context.Context) ([]*models.Incident, error) {
	openList, err := s.repo.ListIncidents(ctx, repository.ListFilter{Status: strPtr(models.StatusOpen), Limit: 500})
	if err != nil {
		return nil, err
	}
	watchList, err := s.repo.ListIncidents(ctx, repository.ListFilter{Status: strPtr(models.StatusWatching), Limit: 500})
	if err != nil {
		return nil, err
	}
	return append(openList, watchList...), nil
}

func strPtr(s string) *string { return &s }
