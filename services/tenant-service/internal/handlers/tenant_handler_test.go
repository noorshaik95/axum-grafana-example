package handlers

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"slate/services/tenant-service/internal/docker"
	"slate/services/tenant-service/internal/kafka"
	"slate/services/tenant-service/internal/models"
	"slate/services/tenant-service/internal/traefik"
)

// fakeProv is a docker.Provisioner stub controllable by tests.
type fakeProv struct {
	mu       sync.Mutex
	provErr  error
	calls    int
}

func (f *fakeProv) Provision(_ context.Context, _ *models.TenantV2) ([]string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls++
	if f.provErr != nil {
		return nil, f.provErr
	}
	return []string{"c1", "c2", "c3", "c4", "c5", "c6", "c7", "c8"}, nil
}
func (f *fakeProv) Deprovision(context.Context, string) error       { return nil }
func (f *fakeProv) StopContainers(context.Context, string) error    { return nil }
func (f *fakeProv) StartContainers(context.Context, string) error   { return nil }
func (f *fakeProv) ListTenantContainers(context.Context, string) ([]docker.ContainerInfo, error) {
	return nil, nil
}
func (f *fakeProv) Close() error { return nil }

// fakeRepo is a minimal TenantCRUDRepository capturing state updates.
type fakeRepo struct {
	mu       sync.Mutex
	tenants  map[string]*models.TenantV2
	detail   map[string]string // id → "status:detail"
}

func newFakeRepo() *fakeRepo {
	return &fakeRepo{tenants: map[string]*models.TenantV2{}, detail: map[string]string{}}
}
func (r *fakeRepo) CreateTenant(_ context.Context, t *models.TenantV2) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if t.ID == "" {
		t.ID = "t-" + t.Slug
	}
	r.tenants[t.ID] = t
	return nil
}
func (r *fakeRepo) GetTenantByID(_ context.Context, id string) (*models.TenantV2, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	t, ok := r.tenants[id]
	if !ok {
		return nil, errors.New("not found")
	}
	return t, nil
}
func (r *fakeRepo) GetBySlug(_ context.Context, slug string) (*models.TenantV2, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, t := range r.tenants {
		if t.Slug == slug {
			return t, nil
		}
	}
	return nil, errors.New("not found")
}
func (r *fakeRepo) ListTenants(context.Context, int, int, string) ([]*models.TenantV2, int, error) {
	return nil, 0, nil
}
func (r *fakeRepo) ListActiveTenants(context.Context) ([]*models.TenantV2, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]*models.TenantV2, 0, len(r.tenants))
	for _, t := range r.tenants {
		out = append(out, t)
	}
	return out, nil
}
func (r *fakeRepo) UpdateTenantStatus(_ context.Context, id, status string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if t, ok := r.tenants[id]; ok {
		t.Status = status
	}
	return nil
}
func (r *fakeRepo) UpdateTenantStatusDetail(_ context.Context, id, status, detail string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.detail[id] = status + ":" + detail
	if t, ok := r.tenants[id]; ok {
		t.Status = status
		t.StatusDetail = detail
	}
	return nil
}
func (r *fakeRepo) UpdateTenantPlan(context.Context, string, models.Plan) error { return nil }
func (r *fakeRepo) UpdateTenantAfterProvision(_ context.Context, t *models.TenantV2) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.tenants[t.ID] = t
	return nil
}
func (r *fakeRepo) SoftDeleteTenant(context.Context, string) error { return nil }

func newTestHandler(t *testing.T, provErr error) (*TenantHandler, *fakeProv, *fakeRepo) {
	t.Helper()
	prov := &fakeProv{provErr: provErr}
	repo := newFakeRepo()
	gen := traefik.NewGeneratorWithTenants(t.TempDir(), t.TempDir())
	h := NewTenantHandler(repo, prov, gen, kafka.NoopProducer{}, nil)
	return h, prov, repo
}

func TestProvisionAsync_RecordsActiveOnSuccess(t *testing.T) {
	h, prov, repo := newTestHandler(t, nil)

	tenant := &models.TenantV2{ID: "t-1", Slug: "eastfield", Name: "Eastfield",
		Status: models.StatusProvisioning}
	repo.tenants[tenant.ID] = tenant

	// provisionAsync is designed to run in a goroutine; call synchronously here.
	h.provisionAsync(tenant)

	if prov.calls != 1 {
		t.Fatalf("expected 1 provision call, got %d", prov.calls)
	}
	got := repo.tenants["t-1"]
	if got.Status != models.StatusActive {
		t.Fatalf("expected status=active, got %s", got.Status)
	}
	if got.StatusDetail != "" {
		t.Fatalf("expected empty status_detail on success, got %q", got.StatusDetail)
	}
	if len(got.ContainerIDs) != 8 {
		t.Fatalf("expected 8 container IDs recorded, got %d", len(got.ContainerIDs))
	}
}

func TestProvisionAsync_RecordsFailedWithDetail(t *testing.T) {
	h, _, repo := newTestHandler(t, errors.New("simulated start failure for discussion-eastfield"))

	tenant := &models.TenantV2{ID: "t-2", Slug: "eastfield", Name: "Eastfield",
		Status: models.StatusProvisioning}
	repo.tenants[tenant.ID] = tenant

	h.provisionAsync(tenant)

	got := repo.tenants["t-2"]
	if got.Status != models.StatusProvisionFailed {
		t.Fatalf("expected status=failed, got %s", got.Status)
	}
	if got.StatusDetail == "" {
		t.Fatal("expected non-empty status_detail on failure")
	}
	// Expect the detail to carry the underlying reason so seed-dev.sh /
	// admin-fe can show it to the operator.
	want := "provisioning failed: simulated start failure for discussion-eastfield"
	if got.StatusDetail != want {
		t.Fatalf("unexpected status_detail %q", got.StatusDetail)
	}
}

// Helper: ensure test timeout doesn't accidentally swallow goroutine leaks.
var _ = time.Second
