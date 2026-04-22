package service

import (
	"context"
	"testing"

	"slate/services/metrics-service/internal/cache"
	"slate/services/metrics-service/internal/incident"
	"slate/services/metrics-service/internal/models"
)

type fakePlatformSource struct {
	base       *models.PlatformMetrics
	tenantMAU  []TenantMAU
	signups    int
	uptime     float64
	baseCalls  int
}

func (f *fakePlatformSource) GetPlatformMetrics(context.Context) (*models.PlatformMetrics, error) {
	f.baseCalls++
	return f.base, nil
}
func (f *fakePlatformSource) GetTenantMAU(context.Context) ([]TenantMAU, error) {
	return f.tenantMAU, nil
}
func (f *fakePlatformSource) GetSignupsLast30d(context.Context) (int, error) { return f.signups, nil }
func (f *fakePlatformSource) GetUptimePct(context.Context) (float64, error)  { return f.uptime, nil }

type stubIncidents struct{ n int }

func (s stubIncidents) ActiveCount(context.Context) (int, error) { return s.n, nil }

func TestPlatformService_Get_PopulatesAndCaches(t *testing.T) {
	src := &fakePlatformSource{
		base:      &models.PlatformMetrics{DAU: 100, MAU: 500, ActiveTenants: 3, TotalCourses: 40, TotalStudents: 400},
		tenantMAU: []TenantMAU{{TenantID: "t1", MAU: 200}, {TenantID: "t2", MAU: 300}},
		signups:   42,
		uptime:    99.95,
	}
	svc := NewPlatformService(src, stubIncidents{n: 2}, cache.NewMemory())

	got, err := svc.Get(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if got.DAU != 100 {
		t.Fatalf("expected DAU 100, got %d", got.DAU)
	}
	if len(got.TenantMAU) != 2 {
		t.Fatalf("expected 2 tenants, got %d", len(got.TenantMAU))
	}
	if got.SignupsLast30d != 42 || got.UptimePct != 99.95 || got.ActiveIncidents != 2 {
		t.Fatalf("unexpected values: %+v", got)
	}

	// Second Get hits cache.
	_, _ = svc.Get(context.Background())
	if src.baseCalls != 1 {
		t.Fatalf("expected 1 underlying fetch, got %d", src.baseCalls)
	}
}

func TestPlatformService_Invalidate(t *testing.T) {
	src := &fakePlatformSource{base: &models.PlatformMetrics{}}
	svc := NewPlatformService(src, nil, cache.NewMemory())

	_, _ = svc.Get(context.Background())
	svc.Invalidate()
	_, _ = svc.Get(context.Background())
	if src.baseCalls != 2 {
		t.Fatalf("expected 2 fetches after invalidation, got %d", src.baseCalls)
	}
}

func TestPlatformService_NilIncidentDefaultsToStub(t *testing.T) {
	src := &fakePlatformSource{base: &models.PlatformMetrics{}}
	svc := NewPlatformService(src, nil, nil)

	got, err := svc.Get(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if got.ActiveIncidents != 0 {
		t.Fatalf("expected 0 incidents from stub, got %d", got.ActiveIncidents)
	}
}

// compile-time check that incident.StubClient satisfies the interface.
var _ incident.Client = incident.NewStub()
