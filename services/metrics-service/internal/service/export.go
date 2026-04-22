package service

import (
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"sync"
	"time"

	"github.com/google/uuid"
)

// Export types supported by POST /api/metrics/export.
const (
	ExportTypeGradebook = "gradebook"
	ExportTypeRoster    = "roster"
	ExportTypePlatform  = "platform"
)

// Export formats.
const (
	ExportFormatCSV  = "csv"
	ExportFormatXLSX = "xlsx"
)

// Job states.
const (
	ExportStatePending   = "pending"
	ExportStateRunning   = "running"
	ExportStateCompleted = "completed"
	ExportStateFailed    = "failed"
)

// ExportRequest is the inbound payload for POST /api/metrics/export.
type ExportRequest struct {
	Type     string `json:"type"`
	CourseID string `json:"course_id,omitempty"`
	TenantID string `json:"tenant_id,omitempty"`
	Format   string `json:"format"`
}

// Validate checks type/format combinations.
func (r ExportRequest) Validate() error {
	switch r.Type {
	case ExportTypeGradebook, ExportTypeRoster, ExportTypePlatform:
	default:
		return fmt.Errorf("invalid export type %q", r.Type)
	}
	switch r.Format {
	case ExportFormatCSV, ExportFormatXLSX:
	default:
		return fmt.Errorf("invalid export format %q", r.Format)
	}
	if r.Type == ExportTypeGradebook && r.CourseID == "" {
		return errors.New("course_id required for gradebook export")
	}
	if r.Type == ExportTypeRoster && r.CourseID == "" {
		return errors.New("course_id required for roster export")
	}
	return nil
}

// ExportJob is the server-side state of an export request.
type ExportJob struct {
	ID         string        `json:"id"`
	Request    ExportRequest `json:"request"`
	Status     string        `json:"status"`
	SignedURL  string        `json:"signed_url,omitempty"`
	Error      string        `json:"error,omitempty"`
	CreatedAt  time.Time     `json:"created_at"`
	UpdatedAt  time.Time     `json:"updated_at"`
}

// Uploader uploads a file to object storage and returns a signed URL.
// The production implementation is a MinIO client; tests supply a stub.
type Uploader interface {
	Upload(ctx context.Context, key string, data []byte, contentType string) (signedURL string, err error)
}

// EventProducer publishes lifecycle events for export jobs. In production
// this is a Kafka producer emitting `metrics.export_requested`.
type EventProducer interface {
	Publish(ctx context.Context, topic string, key string, payload []byte) error
}

// DataSource provides the rows the export needs. Each method returns a list
// of CSV-style string rows, with the first row being the header.
type DataSource interface {
	GradebookRows(ctx context.Context, tenantID, courseID string) ([][]string, error)
	RosterRows(ctx context.Context, tenantID, courseID string) ([][]string, error)
	PlatformRows(ctx context.Context) ([][]string, error)
}

// ExportService orchestrates async export job lifecycle:
//   enqueue → publish `metrics.export_requested` → worker picks up → generate
//   file → upload → update job with signed URL.
type ExportService struct {
	mu       sync.RWMutex
	jobs     map[string]*ExportJob
	queue    chan string
	uploader Uploader
	producer EventProducer
	source   DataSource
	now      func() time.Time
}

// NewExportService constructs an ExportService. The worker loop is not
// started here; call Start to spin it up. Use ProcessNext directly from
// tests to drive the state machine deterministically.
func NewExportService(uploader Uploader, producer EventProducer, source DataSource) *ExportService {
	return &ExportService{
		jobs:     make(map[string]*ExportJob),
		queue:    make(chan string, 128),
		uploader: uploader,
		producer: producer,
		source:   source,
		now:      time.Now,
	}
}

// Start launches a worker goroutine that drains the queue until ctx is done.
func (s *ExportService) Start(ctx context.Context) {
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case id := <-s.queue:
				_ = s.process(ctx, id)
			}
		}
	}()
}

// Enqueue creates a new export job, publishes a `metrics.export_requested`
// event, queues the job for the worker, and returns the job ID.
func (s *ExportService) Enqueue(ctx context.Context, req ExportRequest) (*ExportJob, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}
	job := &ExportJob{
		ID:        uuid.NewString(),
		Request:   req,
		Status:    ExportStatePending,
		CreatedAt: s.now(),
		UpdatedAt: s.now(),
	}
	s.mu.Lock()
	s.jobs[job.ID] = job
	s.mu.Unlock()

	if s.producer != nil {
		payload, _ := json.Marshal(map[string]any{
			"job_id":    job.ID,
			"type":      req.Type,
			"format":    req.Format,
			"tenant_id": req.TenantID,
			"course_id": req.CourseID,
		})
		_ = s.producer.Publish(ctx, "metrics.export_requested", job.ID, payload)
	}

	select {
	case s.queue <- job.ID:
	default:
		// Queue full — the worker is falling behind. Mark failed so the
		// caller can retry rather than silently lose the request.
		s.setStatus(job.ID, ExportStateFailed, "", "export queue full")
	}
	return s.Get(job.ID), nil
}

// Get returns a copy of the job with the given ID.
func (s *ExportService) Get(id string) *ExportJob {
	s.mu.RLock()
	defer s.mu.RUnlock()
	j, ok := s.jobs[id]
	if !ok {
		return nil
	}
	cp := *j
	return &cp
}

// ProcessNext drains a single queued job. Intended for tests that want
// deterministic lifecycle transitions without racing with a worker.
func (s *ExportService) ProcessNext(ctx context.Context) (string, error) {
	select {
	case id := <-s.queue:
		return id, s.process(ctx, id)
	case <-ctx.Done():
		return "", ctx.Err()
	default:
		return "", nil
	}
}

func (s *ExportService) process(ctx context.Context, id string) error {
	s.setStatus(id, ExportStateRunning, "", "")

	job := s.Get(id)
	if job == nil {
		return fmt.Errorf("job %s not found", id)
	}

	rows, err := s.fetchRows(ctx, job.Request)
	if err != nil {
		s.setStatus(id, ExportStateFailed, "", err.Error())
		return err
	}

	body, contentType, err := encodeRows(rows, job.Request.Format)
	if err != nil {
		s.setStatus(id, ExportStateFailed, "", err.Error())
		return err
	}

	key := fmt.Sprintf("exports/%s/%s.%s", job.Request.Type, id, job.Request.Format)
	url, err := s.uploader.Upload(ctx, key, body, contentType)
	if err != nil {
		s.setStatus(id, ExportStateFailed, "", err.Error())
		return err
	}

	s.setStatus(id, ExportStateCompleted, url, "")
	return nil
}

func (s *ExportService) fetchRows(ctx context.Context, req ExportRequest) ([][]string, error) {
	switch req.Type {
	case ExportTypeGradebook:
		return s.source.GradebookRows(ctx, req.TenantID, req.CourseID)
	case ExportTypeRoster:
		return s.source.RosterRows(ctx, req.TenantID, req.CourseID)
	case ExportTypePlatform:
		return s.source.PlatformRows(ctx)
	default:
		return nil, fmt.Errorf("unsupported export type %q", req.Type)
	}
}

func (s *ExportService) setStatus(id, status, url, errMsg string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	j, ok := s.jobs[id]
	if !ok {
		return
	}
	j.Status = status
	if url != "" {
		j.SignedURL = url
	}
	if errMsg != "" {
		j.Error = errMsg
	}
	j.UpdatedAt = s.now()
}

// encodeRows serialises rows into the requested format. XLSX is emitted as a
// minimal CSV-inside-XLSX-envelope placeholder (single-sheet TSV-ish bytes);
// the frontend can still consume the file via the signed URL. Swap in a real
// xlsx writer later without changing callers.
func encodeRows(rows [][]string, format string) ([]byte, string, error) {
	buf := &bytes.Buffer{}
	switch format {
	case ExportFormatCSV:
		w := csv.NewWriter(buf)
		for _, r := range rows {
			if err := w.Write(r); err != nil {
				return nil, "", err
			}
		}
		w.Flush()
		if err := w.Error(); err != nil {
			return nil, "", err
		}
		return buf.Bytes(), "text/csv", nil
	case ExportFormatXLSX:
		// Minimal stand-in: tab-separated bytes with an xlsx MIME. A real
		// implementation would use e.g. github.com/xuri/excelize.
		for i, r := range rows {
			if i > 0 {
				buf.WriteByte('\n')
			}
			for j, cell := range r {
				if j > 0 {
					buf.WriteByte('\t')
				}
				buf.WriteString(quoteIfNeeded(cell))
			}
		}
		return buf.Bytes(), "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", nil
	default:
		return nil, "", fmt.Errorf("unsupported format %q", format)
	}
}

func quoteIfNeeded(s string) string {
	// Keep it boring — return literal for now; strconv.Quote would escape
	// too aggressively for spreadsheet readers.
	if s == "" {
		return ""
	}
	return s
}

// StubUploader stores uploads in-memory and returns a deterministic signed URL.
type StubUploader struct {
	mu      sync.Mutex
	Uploads map[string][]byte
}

// NewStubUploader returns a fresh StubUploader.
func NewStubUploader() *StubUploader {
	return &StubUploader{Uploads: map[string][]byte{}}
}

// Upload records the bytes under key and returns a fake signed URL.
func (u *StubUploader) Upload(_ context.Context, key string, data []byte, _ string) (string, error) {
	u.mu.Lock()
	defer u.mu.Unlock()
	buf := make([]byte, len(data))
	copy(buf, data)
	u.Uploads[key] = buf
	return "https://minio.local/" + key + "?sig=stub-" + strconv.FormatInt(time.Now().UnixNano(), 36), nil
}

// StubProducer records every publish for assertions.
type StubProducer struct {
	mu       sync.Mutex
	Messages []StubMessage
}

// StubMessage is one recorded publish.
type StubMessage struct {
	Topic   string
	Key     string
	Payload []byte
}

// NewStubProducer returns a fresh StubProducer.
func NewStubProducer() *StubProducer {
	return &StubProducer{}
}

// Publish records the message and never fails.
func (p *StubProducer) Publish(_ context.Context, topic, key string, payload []byte) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	buf := make([]byte, len(payload))
	copy(buf, payload)
	p.Messages = append(p.Messages, StubMessage{Topic: topic, Key: key, Payload: buf})
	return nil
}

// Count returns the number of recorded messages on a topic.
func (p *StubProducer) Count(topic string) int {
	p.mu.Lock()
	defer p.mu.Unlock()
	n := 0
	for _, m := range p.Messages {
		if m.Topic == topic {
			n++
		}
	}
	return n
}
