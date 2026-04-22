package service

import (
	"context"
	"encoding/csv"
	"errors"
	"strings"
	"testing"
)

type fakeSource struct {
	gradebook [][]string
	roster    [][]string
	platform  [][]string
	err       error
}

func (f *fakeSource) GradebookRows(context.Context, string, string) ([][]string, error) {
	return f.gradebook, f.err
}
func (f *fakeSource) RosterRows(context.Context, string, string) ([][]string, error) {
	return f.roster, f.err
}
func (f *fakeSource) PlatformRows(context.Context) ([][]string, error) {
	return f.platform, f.err
}

func TestExport_Validate(t *testing.T) {
	good := ExportRequest{Type: ExportTypeGradebook, Format: ExportFormatCSV, CourseID: "c1"}
	if err := good.Validate(); err != nil {
		t.Fatalf("unexpected: %v", err)
	}

	cases := []ExportRequest{
		{Type: "bogus", Format: ExportFormatCSV, CourseID: "c1"},
		{Type: ExportTypeGradebook, Format: "pdf", CourseID: "c1"},
		{Type: ExportTypeGradebook, Format: ExportFormatCSV}, // missing course_id
		{Type: ExportTypeRoster, Format: ExportFormatCSV},    // missing course_id
	}
	for i, c := range cases {
		if err := c.Validate(); err == nil {
			t.Errorf("case %d: expected validation error", i)
		}
	}
}

func TestExport_Lifecycle_Gradebook(t *testing.T) {
	src := &fakeSource{
		gradebook: [][]string{
			{"student_id", "name", "grade"},
			{"u1", "Alice", "95"},
			{"u2", "Bob", "72"},
		},
	}
	up := NewStubUploader()
	prod := NewStubProducer()
	svc := NewExportService(up, prod, src)

	ctx := context.Background()
	job, err := svc.Enqueue(ctx, ExportRequest{
		Type:     ExportTypeGradebook,
		CourseID: "c1",
		TenantID: "t1",
		Format:   ExportFormatCSV,
	})
	if err != nil {
		t.Fatal(err)
	}
	if job.Status != ExportStatePending {
		t.Fatalf("expected pending, got %s", job.Status)
	}
	if prod.Count("metrics.export_requested") != 1 {
		t.Fatalf("expected 1 event, got %d", prod.Count("metrics.export_requested"))
	}

	id, err := svc.ProcessNext(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if id != job.ID {
		t.Fatalf("expected %s, got %s", job.ID, id)
	}

	done := svc.Get(job.ID)
	if done.Status != ExportStateCompleted {
		t.Fatalf("expected completed, got %s (err=%s)", done.Status, done.Error)
	}
	if done.SignedURL == "" {
		t.Fatal("expected signed URL")
	}

	// Verify uploaded CSV content
	var uploaded []byte
	for _, b := range up.Uploads {
		uploaded = b
	}
	r := csv.NewReader(strings.NewReader(string(uploaded)))
	rows, _ := r.ReadAll()
	if len(rows) != 3 || rows[1][1] != "Alice" {
		t.Fatalf("unexpected CSV: %v", rows)
	}
}

func TestExport_Lifecycle_RosterXLSX(t *testing.T) {
	src := &fakeSource{
		roster: [][]string{
			{"user_id", "risk"},
			{"u1", "at_risk"},
		},
	}
	svc := NewExportService(NewStubUploader(), NewStubProducer(), src)

	job, err := svc.Enqueue(context.Background(), ExportRequest{
		Type:     ExportTypeRoster,
		CourseID: "c1",
		Format:   ExportFormatXLSX,
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.ProcessNext(context.Background()); err != nil {
		t.Fatal(err)
	}
	if svc.Get(job.ID).Status != ExportStateCompleted {
		t.Fatalf("expected completed, got %s", svc.Get(job.ID).Status)
	}
}

func TestExport_Lifecycle_Platform(t *testing.T) {
	src := &fakeSource{platform: [][]string{{"metric", "value"}, {"dau", "100"}}}
	svc := NewExportService(NewStubUploader(), NewStubProducer(), src)

	job, _ := svc.Enqueue(context.Background(), ExportRequest{
		Type:   ExportTypePlatform,
		Format: ExportFormatCSV,
	})
	if _, err := svc.ProcessNext(context.Background()); err != nil {
		t.Fatal(err)
	}
	if svc.Get(job.ID).Status != ExportStateCompleted {
		t.Fatalf("expected completed, got %s", svc.Get(job.ID).Status)
	}
}

func TestExport_FetchError(t *testing.T) {
	src := &fakeSource{err: errors.New("db down")}
	svc := NewExportService(NewStubUploader(), NewStubProducer(), src)

	job, _ := svc.Enqueue(context.Background(), ExportRequest{
		Type:     ExportTypeGradebook,
		CourseID: "c1",
		Format:   ExportFormatCSV,
	})
	_, err := svc.ProcessNext(context.Background())
	if err == nil {
		t.Fatal("expected error")
	}
	got := svc.Get(job.ID)
	if got.Status != ExportStateFailed {
		t.Fatalf("expected failed, got %s", got.Status)
	}
	if got.Error == "" {
		t.Fatal("expected error message")
	}
}

type brokenUploader struct{}

func (brokenUploader) Upload(context.Context, string, []byte, string) (string, error) {
	return "", errors.New("minio unavailable")
}

func TestExport_UploadError(t *testing.T) {
	src := &fakeSource{gradebook: [][]string{{"h"}}}
	svc := NewExportService(brokenUploader{}, NewStubProducer(), src)
	job, _ := svc.Enqueue(context.Background(), ExportRequest{
		Type: ExportTypeGradebook, CourseID: "c1", Format: ExportFormatCSV,
	})
	_, _ = svc.ProcessNext(context.Background())
	if svc.Get(job.ID).Status != ExportStateFailed {
		t.Fatalf("expected failed, got %s", svc.Get(job.ID).Status)
	}
}

func TestEncodeRows_CSV(t *testing.T) {
	rows := [][]string{{"a", "b"}, {"1", "2,hello"}}
	body, ct, err := encodeRows(rows, ExportFormatCSV)
	if err != nil {
		t.Fatal(err)
	}
	if ct != "text/csv" {
		t.Fatalf("expected text/csv, got %s", ct)
	}
	if !strings.Contains(string(body), "\"2,hello\"") {
		t.Fatalf("expected quoted CSV, got %s", body)
	}
}

func TestEncodeRows_UnsupportedFormat(t *testing.T) {
	if _, _, err := encodeRows(nil, "pdf"); err == nil {
		t.Fatal("expected error")
	}
}
