package docker

import (
	"context"
	"testing"

	"slate/services/tenant-service/internal/models"
)

// fakeReconcileRepo captures reconciler state changes.
type fakeReconcileRepo struct {
	active              []*models.TenantV2
	statusDetail        []string // status|detail entries in order
	afterProvisionIDs   []string
}

func (f *fakeReconcileRepo) ListActiveTenants(context.Context) ([]*models.TenantV2, error) {
	return f.active, nil
}
func (f *fakeReconcileRepo) UpdateTenantStatusDetail(_ context.Context, id, status, detail string) error {
	f.statusDetail = append(f.statusDetail, id+":"+status+":"+detail)
	return nil
}
func (f *fakeReconcileRepo) UpdateTenantAfterProvision(_ context.Context, t *models.TenantV2) error {
	f.afterProvisionIDs = append(f.afterProvisionIDs, t.ID)
	return nil
}

type fakeEndpointsWriter struct {
	slugs []string
}

func (f *fakeEndpointsWriter) WriteTenantEndpoints(slug string) error {
	f.slugs = append(f.slugs, slug)
	return nil
}

func TestReconcile_SkipsFullyProvisionedTenants(t *testing.T) {
	api := newFakeDockerAPI()
	p := NewWithDockerAPI(api, Env{})
	// Provision tenant so all 8 containers exist.
	if _, err := p.Provision(context.Background(), &models.TenantV2{ID: "t-1", Slug: "eastfield"}); err != nil {
		t.Fatal(err)
	}

	repo := &fakeReconcileRepo{active: []*models.TenantV2{{ID: "t-1", Slug: "eastfield"}}}
	eps := &fakeEndpointsWriter{}
	if err := Reconcile(context.Background(), p, repo, eps); err != nil {
		t.Fatal(err)
	}

	if len(repo.afterProvisionIDs) != 0 {
		t.Fatalf("expected no re-provisioning for fully-healthy tenant, got %v", repo.afterProvisionIDs)
	}
	if len(eps.slugs) != 1 || eps.slugs[0] != "eastfield" {
		t.Fatalf("expected endpoints yaml rewritten once, got %v", eps.slugs)
	}
}

func TestReconcile_ReprovisionsMissingServices(t *testing.T) {
	api := newFakeDockerAPI()
	p := NewWithDockerAPI(api, Env{})
	// Provision, then manually remove one container to simulate drift.
	if _, err := p.Provision(context.Background(), &models.TenantV2{ID: "t-1", Slug: "eastfield"}); err != nil {
		t.Fatal(err)
	}
	// Delete the ai container.
	for id, c := range api.containers {
		if c.Labels["service"] == "ai" {
			delete(api.containers, id)
			break
		}
	}

	repo := &fakeReconcileRepo{active: []*models.TenantV2{{ID: "t-1", Slug: "eastfield"}}}
	if err := Reconcile(context.Background(), p, repo, &fakeEndpointsWriter{}); err != nil {
		t.Fatal(err)
	}

	if len(repo.afterProvisionIDs) != 1 || repo.afterProvisionIDs[0] != "t-1" {
		t.Fatalf("expected re-provisioning of t-1, got %v", repo.afterProvisionIDs)
	}
	// Reconciler re-Provisions the full 8-service set; the tenant network
	// already exists so ensureNetwork returns the same ID.
	if len(api.networks) != 1 {
		t.Fatalf("expected exactly 1 network after reconcile, got %d", len(api.networks))
	}
}
