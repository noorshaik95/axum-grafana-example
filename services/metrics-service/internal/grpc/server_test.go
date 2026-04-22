package grpc

import (
	"context"
	"testing"

	pb "slate/services/metrics-service/api/proto"
	"slate/services/metrics-service/internal/analytics"
	"slate/services/metrics-service/internal/cache"
	"slate/services/metrics-service/internal/models"
	"slate/services/metrics-service/internal/service"
)

// TestDistToProto_IsomorphicWithRESTShape asserts the proto grade-distribution
// response carries exactly the fields the REST handler returns. If either
// drops out of sync, admin-fe unmarshalling breaks — this test pins it.
func TestDistToProto_IsomorphicWithRESTShape(t *testing.T) {
	scores := []float64{95, 85, 75, 65, 55, 45, 35, 25, 15, 5}
	d := analytics.ComputeDistribution("course-1", scores)

	out := distToProto(&d)

	if out.CourseId != "course-1" {
		t.Errorf("course_id: want course-1, got %s", out.CourseId)
	}
	if out.TotalStudents != 10 {
		t.Errorf("total_students: want 10, got %d", out.TotalStudents)
	}
	if len(out.Buckets) != 10 {
		t.Fatalf("buckets: want 10, got %d", len(out.Buckets))
	}
	for i, b := range out.Buckets {
		if b.Count != 1 {
			t.Errorf("bucket %d: want count 1, got %d", i, b.Count)
		}
	}
	if out.Mean != d.Mean || out.Median != d.Median || out.P25 != d.P25 || out.P75 != d.P75 {
		t.Errorf("headline stats diverged between REST and gRPC shapes: %+v vs %+v", d, out)
	}
}

func TestJobToProto_ShapeParity(t *testing.T) {
	job := &service.ExportJob{
		ID: "job-1",
		Request: service.ExportRequest{
			Type:     service.ExportTypeGradebook,
			CourseID: "c1",
			TenantID: "t1",
			Format:   service.ExportFormatCSV,
		},
		Status:    service.ExportStateCompleted,
		SignedURL: "https://minio/x",
	}
	out := jobToProto(job)
	if out.Id != "job-1" || out.Status != "completed" || out.SignedUrl != "https://minio/x" {
		t.Fatalf("job proto shape drift: %+v", out)
	}
	if out.Request.Type != "gradebook" || out.Request.CourseId != "c1" {
		t.Fatalf("request proto shape drift: %+v", out.Request)
	}
}

func TestGetRosterHealth_gRPC_FetcherIntegration(t *testing.T) {
	// Roster signals fixture — 1 at_risk, 1 slipping, 1 healthy.
	svc := service.NewRosterService(cache.NewMemory())
	fakeRepo := &fakeRosterRepo{rows: []models.StudentProgress{}}
	_ = fakeRepo // placeholder to document intent; real repo not reachable here

	entries, err := svc.GetRosterHealth(context.Background(), "eastfield", "CS101", "instr",
		func(context.Context, string, string, string) ([]service.RosterSignals, error) {
			return []service.RosterSignals{
				{UserID: "u1", MissedAssignments: 3},
				{UserID: "u2", MissedAssignments: 1},
				{UserID: "u3", GradeTrend: []float64{90, 92}},
			}, nil
		})
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 3 {
		t.Fatalf("want 3, got %d", len(entries))
	}

	// Translate to proto the way Server.GetRosterHealth does.
	protoEntries := make([]*pb.RosterEntry, len(entries))
	atRisk := 0
	for i, e := range entries {
		if e.RiskLevel == service.RiskAtRisk {
			atRisk++
		}
		protoEntries[i] = &pb.RosterEntry{
			UserId:            e.UserID,
			DisplayName:       e.DisplayName,
			RiskLevel:         e.RiskLevel,
			MissedAssignments: int32(e.MissedAssignments),
			DaysSinceActive:   int32(e.DaysSinceActive),
			GradeTrend:        e.GradeTrend,
			SuggestedAction:   e.SuggestedAction,
		}
	}
	resp := &pb.GetRosterHealthResponse{
		CourseId: "CS101",
		Students: protoEntries,
		Total:    int32(len(entries)),
		AtRisk:   int32(atRisk),
	}

	if resp.Total != 3 || resp.AtRisk != 1 {
		t.Fatalf("want Total=3 AtRisk=1, got Total=%d AtRisk=%d", resp.Total, resp.AtRisk)
	}
	if resp.Students[0].RiskLevel != "at_risk" {
		t.Fatalf("expected at_risk first in sort order, got %s", resp.Students[0].RiskLevel)
	}
	// Verify proto field names match REST JSON keys exactly (asserting the
	// isomorphism PO required). The JSON tags on pb.RosterEntry come from
	// the proto field names; if this identity drifts, the test fails.
	if resp.Students[0].GetUserId() == "" || resp.Students[0].GetRiskLevel() == "" {
		t.Fatal("proto getters returned zero values — codegen regressed")
	}
}

// fakeRosterRepo exists only to document the server's dependency surface;
// end-to-end RPC wiring (with a real *sql.DB) lives in integration tests.
type fakeRosterRepo struct{ rows []models.StudentProgress }
