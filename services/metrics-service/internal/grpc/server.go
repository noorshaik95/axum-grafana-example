// Package grpc hosts the thin gRPC wrapper over the existing REST handlers.
// The wrapper exists so api-gateway can reach metrics RPCs declared in
// config/gateway-config.yaml; it is *consumption-layer only* per PO
// ratification (CONTRACTS.md metrics.MetricsService W12). Every response is
// isomorphic to the equivalent REST JSON payload — adding or renaming fields
// here means the REST JSON must change too.
package grpc

import (
	"context"
	"database/sql"
	"sort"
	"time"

	pb "slate/services/metrics-service/api/proto"
	"slate/services/metrics-service/internal/analytics"
	"slate/services/metrics-service/internal/incident"
	"slate/services/metrics-service/internal/models"
	"slate/services/metrics-service/internal/repository"
	"slate/services/metrics-service/internal/service"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// Server satisfies pb.MetricsServiceServer by delegating to the same repo +
// service layer the REST handlers already use.
type Server struct {
	pb.UnimplementedMetricsServiceServer

	repo        *repository.Repository
	roster      *service.RosterService
	export      *service.ExportService
	platform    *service.PlatformService
	incidents   incident.Client
}

// NewServer wires the server to existing dependencies.
func NewServer(
	repo *repository.Repository,
	roster *service.RosterService,
	export *service.ExportService,
	platform *service.PlatformService,
) *Server {
	return &Server{
		repo:      repo,
		roster:    roster,
		export:    export,
		platform:  platform,
		incidents: incident.NewStub(),
	}
}

// GetPlatformMetrics mirrors GET /metrics/platform.
func (s *Server) GetPlatformMetrics(ctx context.Context, _ *pb.PlatformMetricsRequest) (*pb.PlatformMetricsResponse, error) {
	m, err := s.repo.GetPlatformMetrics(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "platform metrics: %v", err)
	}
	return &pb.PlatformMetricsResponse{
		ActiveTenants: int32(m.ActiveTenants),
		Dau:           int32(m.DAU),
		Mau:           int32(m.MAU),
		TotalCourses:  int32(m.TotalCourses),
		TotalStudents: int32(m.TotalStudents),
	}, nil
}

// GetTenantMetrics mirrors GET /metrics/tenants/{id}. Only the fields the
// REST handler already returns are populated; legacy SystemMetrics fields
// (tier, storage quota, etc.) stay zero until a tenant-metadata source is
// wired. Keeping those zero preserves the REST JSON shape.
func (s *Server) GetTenantMetrics(ctx context.Context, req *pb.GetTenantMetricsRequest) (*pb.TenantMetricsResponse, error) {
	if req.GetTenantId() == "" {
		return nil, status.Error(codes.InvalidArgument, "tenant_id required")
	}
	m, err := s.repo.GetTenantMetrics(ctx, req.GetTenantId())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "tenant metrics: %v", err)
	}
	return &pb.TenantMetricsResponse{
		TenantId:        m.TenantID,
		UserCount:       int32(m.ActiveUsers),
		CourseCount:     int32(m.TotalCourses),
		ActiveUsersCount: int64(m.ActiveUsers),
		ApiRequests:     int64(m.EventCount),
		CollectedAt:     timestamppb.New(time.Now()),
	}, nil
}

// GetStudentProgress mirrors GET /metrics/students/{id}.
func (s *Server) GetStudentProgress(ctx context.Context, req *pb.StudentProgressRequest) (*pb.StudentProgressResponse, error) {
	tenantID := req.GetTenantId()
	if tenantID == "" {
		resolved, err := s.repo.GetTenantIDForStudent(ctx, req.GetStudentId())
		if err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "tenant not found for student")
		}
		tenantID = resolved
	}
	rows, err := s.repo.GetStudentProgress(ctx, tenantID, req.GetStudentId())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "student progress: %v", err)
	}
	resp := &pb.StudentProgressResponse{Entries: make([]*pb.StudentProgressEntry, 0, len(rows))}
	for _, r := range rows {
		entry := &pb.StudentProgressEntry{
			Id:                r.ID,
			TenantId:          r.TenantID,
			StudentId:         r.StudentID,
			CourseId:          r.CourseID,
			CompletionPct:     r.CompletionPct,
			TimeOnTaskMinutes: int32(r.TimeOnTaskMinutes),
			LessonsCompleted:  int32(r.LessonsCompleted),
			UpdatedAt:         timestamppb.New(r.UpdatedAt),
		}
		if !r.LastActivityAt.IsZero() {
			entry.LastActivityAt = timestamppb.New(r.LastActivityAt)
		}
		resp.Entries = append(resp.Entries, entry)
	}
	return resp, nil
}

// GetGradeDistribution mirrors GET /metrics/grades/distribution/{courseId}.
func (s *Server) GetGradeDistribution(ctx context.Context, req *pb.GradeDistributionRequest) (*pb.GradeDistributionResponse, error) {
	courseID := req.GetCourseId()
	if courseID == "" {
		return nil, status.Error(codes.InvalidArgument, "course_id required")
	}
	tenantID := req.GetTenantId()
	if tenantID == "" {
		if resolved, err := s.repo.GetTenantIDForCourse(ctx, courseID); err == nil {
			tenantID = resolved
		}
	}
	scores, err := s.repo.GetScoresForCourse(ctx, tenantID, courseID)
	if err != nil && err != sql.ErrNoRows {
		return nil, status.Errorf(codes.Internal, "scores: %v", err)
	}
	dist := analytics.ComputeDistribution(courseID, scores)
	return distToProto(&dist), nil
}

// GetStudentComparison mirrors GET /metrics/grades/comparison/{studentId}/{courseId}.
func (s *Server) GetStudentComparison(ctx context.Context, req *pb.StudentComparisonRequest) (*pb.StudentComparisonResponse, error) {
	courseID := req.GetCourseId()
	studentID := req.GetStudentId()
	if courseID == "" || studentID == "" {
		return nil, status.Error(codes.InvalidArgument, "course_id + student_id required")
	}
	tenantID := req.GetTenantId()
	if tenantID == "" {
		if resolved, err := s.repo.GetTenantIDForCourse(ctx, courseID); err == nil {
			tenantID = resolved
		}
	}
	scores, err := s.repo.GetScoresForCourse(ctx, tenantID, courseID)
	if err != nil && err != sql.ErrNoRows {
		return nil, status.Errorf(codes.Internal, "scores: %v", err)
	}
	dist := analytics.ComputeDistribution(courseID, scores)
	studentScore, err := s.repo.GetStudentScoreForCourse(ctx, tenantID, studentID, courseID)
	if err != nil && err != sql.ErrNoRows {
		return nil, status.Errorf(codes.Internal, "student score: %v", err)
	}
	sort.Float64s(scores)
	return &pb.StudentComparisonResponse{
		Distribution:      distToProto(&dist),
		StudentScore:      studentScore,
		StudentPercentile: analytics.ComputeStudentPercentile(scores, studentScore),
		LetterGrade:       analytics.LetterGrade(studentScore),
	}, nil
}

// GetRosterHealth mirrors GET /roster/{courseId}/health.
func (s *Server) GetRosterHealth(ctx context.Context, req *pb.GetRosterHealthRequest) (*pb.GetRosterHealthResponse, error) {
	courseID := req.GetCourseId()
	if courseID == "" {
		return nil, status.Error(codes.InvalidArgument, "course_id required")
	}
	tenantSlug := req.GetTenantSlug()
	if tenantSlug == "" {
		tenantSlug = "default"
	}
	tenantID, err := s.repo.GetTenantIDForCourse(ctx, courseID)
	if err != nil {
		tenantID = tenantSlug
	}

	fetch := func(ctx context.Context, _, cid, _ string) ([]service.RosterSignals, error) {
		rows, err := s.repo.GetRosterSignals(ctx, tenantID, cid)
		if err != nil {
			return nil, err
		}
		out := make([]service.RosterSignals, len(rows))
		for i, row := range rows {
			out[i] = service.RosterSignals{
				UserID:            row.UserID,
				DisplayName:       row.DisplayName,
				MissedAssignments: row.MissedAssignments,
				DaysSinceActive:   row.DaysSinceActive,
				GradeTrend:        row.GradeTrend,
			}
		}
		return out, nil
	}
	entries, err := s.roster.GetRosterHealth(ctx, tenantSlug, courseID, req.GetInstructorId(), fetch)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "roster: %v", err)
	}

	atRisk := 0
	protoEntries := make([]*pb.RosterEntry, 0, len(entries))
	for _, e := range entries {
		if e.RiskLevel == service.RiskAtRisk {
			atRisk++
		}
		protoEntries = append(protoEntries, &pb.RosterEntry{
			UserId:            e.UserID,
			DisplayName:       e.DisplayName,
			RiskLevel:         e.RiskLevel,
			MissedAssignments: int32(e.MissedAssignments),
			DaysSinceActive:   int32(e.DaysSinceActive),
			GradeTrend:        e.GradeTrend,
			SuggestedAction:   e.SuggestedAction,
		})
	}
	return &pb.GetRosterHealthResponse{
		CourseId: courseID,
		Students: protoEntries,
		Total:    int32(len(entries)),
		AtRisk:   int32(atRisk),
	}, nil
}

// CreateExport mirrors POST /api/metrics/export.
func (s *Server) CreateExport(ctx context.Context, req *pb.CreateExportRequest) (*pb.ExportJob, error) {
	job, err := s.export.Enqueue(ctx, service.ExportRequest{
		Type:     req.GetType(),
		CourseID: req.GetCourseId(),
		TenantID: req.GetTenantId(),
		Format:   req.GetFormat(),
	})
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "%v", err)
	}
	return jobToProto(job), nil
}

// GetExport mirrors GET /api/metrics/export/{jobId}.
func (s *Server) GetExport(_ context.Context, req *pb.GetExportRequest) (*pb.ExportJob, error) {
	job := s.export.Get(req.GetJobId())
	if job == nil {
		return nil, status.Error(codes.NotFound, "job not found")
	}
	return jobToProto(job), nil
}

// GetPlatformMetricsExtended mirrors GET /metrics/platform/extended.
func (s *Server) GetPlatformMetricsExtended(ctx context.Context, _ *pb.PlatformMetricsRequest) (*pb.PlatformMetricsExtendedResponse, error) {
	ext, err := s.platform.Get(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "extended platform metrics: %v", err)
	}
	resp := &pb.PlatformMetricsExtendedResponse{
		ActiveTenants:   int32(ext.PlatformMetrics.ActiveTenants),
		Dau:             int32(ext.PlatformMetrics.DAU),
		Mau:             int32(ext.PlatformMetrics.MAU),
		TotalCourses:    int32(ext.PlatformMetrics.TotalCourses),
		TotalStudents:   int32(ext.PlatformMetrics.TotalStudents),
		SignupsLast_30D: int32(ext.SignupsLast30d),
		UptimePct:       ext.UptimePct,
		ActiveIncidents: int32(ext.ActiveIncidents),
		GeneratedAt:     timestamppb.New(ext.GeneratedAt),
	}
	resp.TenantMau = make([]*pb.TenantMAU, len(ext.TenantMAU))
	for i, t := range ext.TenantMAU {
		resp.TenantMau[i] = &pb.TenantMAU{TenantId: t.TenantID, Mau: int32(t.MAU)}
	}
	return resp, nil
}

// GetSystemMetrics reuses the base PlatformMetrics for counts and fills the
// remaining fields with zero so the REST consumers see a stable shape. Real
// system-level wiring (storage, response-time histograms) lives in
// Prometheus; surfacing it through this RPC is follow-up work.
func (s *Server) GetSystemMetrics(ctx context.Context, _ *pb.GetSystemMetricsRequest) (*pb.SystemMetricsResponse, error) {
	m, err := s.repo.GetPlatformMetrics(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "system metrics: %v", err)
	}
	return &pb.SystemMetricsResponse{
		ActiveTenantsCount: int32(m.ActiveTenants),
		TotalTenantsCount:  int32(m.ActiveTenants),
		TotalUsers:         int64(m.MAU),
		TotalCourses:       int64(m.TotalCourses),
		CollectedAt:        timestamppb.New(time.Now()),
	}, nil
}

// GetActiveAlerts is a compatibility stub — the alerts pipeline is not yet
// seeded. Returning an empty list is the shape admin-fe already handles.
func (s *Server) GetActiveAlerts(context.Context, *pb.GetActiveAlertsRequest) (*pb.AlertsResponse, error) {
	return &pb.AlertsResponse{Alerts: nil, TotalCount: 0}, nil
}

// --- helpers ---

func distToProto(d *models.GradeDistribution) *pb.GradeDistributionResponse {
	buckets := make([]*pb.DistributionBucket, len(d.Buckets))
	for i, b := range d.Buckets {
		buckets[i] = &pb.DistributionBucket{Min: b.Min, Max: b.Max, Count: int32(b.Count)}
	}
	return &pb.GradeDistributionResponse{
		CourseId:      d.CourseID,
		AssignmentId:  d.AssignmentID,
		Buckets:       buckets,
		Mean:          d.Mean,
		Median:        d.Median,
		P25:           d.P25,
		P75:           d.P75,
		TotalStudents: int32(d.TotalStudents),
	}
}

func jobToProto(j *service.ExportJob) *pb.ExportJob {
	if j == nil {
		return nil
	}
	return &pb.ExportJob{
		Id: j.ID,
		Request: &pb.CreateExportRequest{
			Type:     j.Request.Type,
			CourseId: j.Request.CourseID,
			TenantId: j.Request.TenantID,
			Format:   j.Request.Format,
		},
		Status:    j.Status,
		SignedUrl: j.SignedURL,
		Error:     j.Error,
		CreatedAt: timestamppb.New(j.CreatedAt),
		UpdatedAt: timestamppb.New(j.UpdatedAt),
	}
}
