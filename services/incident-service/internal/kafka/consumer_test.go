package kafka

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"slate/services/incident-service/internal/models"
	"slate/services/incident-service/internal/repository"
)

type fakeRepo struct {
	hasOpen           bool
	hasOpenErr        error
	createCalls       []repository.CreateIncidentParams
	createReturnErr   error
	createdID         string
	addEventCalls     int
	listIncidentsResp []*models.Incident
}

func (f *fakeRepo) HasOpenIncidentFor(_ context.Context, _ string, _ *string) (bool, error) {
	return f.hasOpen, f.hasOpenErr
}

func (f *fakeRepo) CreateIncident(_ context.Context, p repository.CreateIncidentParams) (*models.Incident, error) {
	f.createCalls = append(f.createCalls, p)
	if f.createReturnErr != nil {
		return nil, f.createReturnErr
	}
	svc := ""
	if p.Service != nil {
		svc = *p.Service
	}
	_ = svc
	return &models.Incident{
		ID:       f.createdID,
		Title:    p.Title,
		Priority: p.Priority,
		Status:   models.StatusOpen,
		Service:  p.Service,
		TenantID: p.TenantID,
	}, nil
}

func (f *fakeRepo) AddEvent(_ context.Context, _ string, _ *string, _, _ string) (*models.IncidentEvent, error) {
	f.addEventCalls++
	return &models.IncidentEvent{ID: "ev-" + f.createdID}, nil
}

func (f *fakeRepo) ListIncidents(_ context.Context, _ repository.ListFilter) ([]*models.Incident, error) {
	return f.listIncidentsResp, nil
}

type fakePublisher struct {
	openedCalls []*models.Incident
}

func (p *fakePublisher) PublishIncidentOpened(_ context.Context, inc *models.Incident) {
	p.openedCalls = append(p.openedCalls, inc)
}

type fakeCache struct {
	setCalls int
	last     []*models.Incident
}

func (c *fakeCache) SetActive(_ context.Context, incs []*models.Incident) error {
	c.setCalls++
	c.last = incs
	return nil
}

func TestHandleThresholdBreached_AutoOpenP1(t *testing.T) {
	repo := &fakeRepo{hasOpen: false, createdID: "auto-1"}
	pub := &fakePublisher{}
	cc := &fakeCache{}
	c := NewConsumerForTest(repo, pub, cc)

	tenantID := "tenant-xyz"
	payload, _ := json.Marshal(ThresholdBreachedEvent{
		TenantID:   &tenantID,
		Service:    "course-service",
		MetricName: "p99_latency_ms",
		Threshold:  1000,
		Observed:   2500,
	})

	if err := c.HandleThresholdBreached(context.Background(), payload); err != nil {
		t.Fatalf("HandleThresholdBreached: %v", err)
	}

	if len(repo.createCalls) != 1 {
		t.Fatalf("expected 1 CreateIncident call, got %d", len(repo.createCalls))
	}
	got := repo.createCalls[0]
	if got.Priority != models.PriorityP1 {
		t.Errorf("expected P1, got %q", got.Priority)
	}
	if got.Service == nil || *got.Service != "course-service" {
		t.Errorf("expected service=course-service, got %v", got.Service)
	}
	if got.TenantID == nil || *got.TenantID != "tenant-xyz" {
		t.Errorf("expected tenant_id forwarded, got %v", got.TenantID)
	}
	if repo.addEventCalls != 1 {
		t.Errorf("expected 1 AddEvent (auto_open), got %d", repo.addEventCalls)
	}
	if len(pub.openedCalls) != 1 {
		t.Errorf("expected 1 PublishIncidentOpened, got %d", len(pub.openedCalls))
	}
	if cc.setCalls != 1 {
		t.Errorf("expected 1 cache SetActive, got %d", cc.setCalls)
	}
}

func TestHandleThresholdBreached_SuppressesDuplicates(t *testing.T) {
	repo := &fakeRepo{hasOpen: true}
	pub := &fakePublisher{}
	cc := &fakeCache{}
	c := NewConsumerForTest(repo, pub, cc)

	payload, _ := json.Marshal(ThresholdBreachedEvent{Service: "course-service"})
	if err := c.HandleThresholdBreached(context.Background(), payload); err != nil {
		t.Fatalf("HandleThresholdBreached: %v", err)
	}

	if len(repo.createCalls) != 0 {
		t.Errorf("expected no CreateIncident when an open incident exists, got %d calls", len(repo.createCalls))
	}
	if len(pub.openedCalls) != 0 {
		t.Errorf("expected no publish when suppressed")
	}
	if cc.setCalls != 0 {
		t.Errorf("expected no cache write when suppressed")
	}
}

func TestHandleThresholdBreached_RequiresService(t *testing.T) {
	repo := &fakeRepo{}
	c := NewConsumerForTest(repo, &fakePublisher{}, nil)
	payload, _ := json.Marshal(ThresholdBreachedEvent{Service: ""})
	err := c.HandleThresholdBreached(context.Background(), payload)
	if err == nil {
		t.Error("expected error when service missing")
	}
}

func TestHandleThresholdBreached_MalformedJSON(t *testing.T) {
	c := NewConsumerForTest(&fakeRepo{}, &fakePublisher{}, nil)
	err := c.HandleThresholdBreached(context.Background(), []byte("not-json"))
	if err == nil {
		t.Error("expected error for malformed json")
	}
}

func TestHandleThresholdBreached_RepoCheckError(t *testing.T) {
	repo := &fakeRepo{hasOpenErr: errors.New("db down")}
	c := NewConsumerForTest(repo, &fakePublisher{}, nil)
	payload, _ := json.Marshal(ThresholdBreachedEvent{Service: "x"})
	if err := c.HandleThresholdBreached(context.Background(), payload); err == nil {
		t.Error("expected error when HasOpenIncidentFor fails")
	}
}
