// Reconciler is the startup routine that walks every active tenant in the
// database, checks the live Docker state, and re-creates anything that is
// missing. This makes a cold-start of tenant-service safe after a host
// reboot or after an orphaned container cleanup.
package docker

import (
	"context"
	"fmt"

	"github.com/rs/zerolog/log"

	"slate/services/tenant-service/internal/models"
)

// ReconcileRepo is the repository surface the reconciler needs — narrower than
// the full TenantCRUDRepository so tests can stub easily.
type ReconcileRepo interface {
	ListActiveTenants(ctx context.Context) ([]*models.TenantV2, error)
	UpdateTenantStatusDetail(ctx context.Context, id, status, detail string) error
	UpdateTenantAfterProvision(ctx context.Context, tenant *models.TenantV2) error
}

// ReconcileEndpointsWriter writes the config/tenants/{slug}.yaml endpoints
// file for a given tenant — matches traefik.Generator.
type ReconcileEndpointsWriter interface {
	WriteTenantEndpoints(slug string) error
}

// Reconcile walks every active/provisioning tenant and re-provisions any whose
// live container set is missing one or more of the expected 8 services. It
// also re-writes the per-tenant endpoints yaml when it's missing. Errors on
// individual tenants are logged but do not abort the pass so a single bad
// tenant can't stop the rest of the platform from coming up.
func Reconcile(ctx context.Context, prov Provisioner, repo ReconcileRepo, eps ReconcileEndpointsWriter) error {
	tenants, err := repo.ListActiveTenants(ctx)
	if err != nil {
		return fmt.Errorf("list active tenants: %w", err)
	}
	expected := make(map[string]bool, len(ServiceSpecs))
	for _, s := range ServiceSpecs {
		expected[s.Name] = true
	}

	for _, tenant := range tenants {
		if eps != nil {
			if err := eps.WriteTenantEndpoints(tenant.Slug); err != nil {
				log.Warn().Err(err).Str("slug", tenant.Slug).Msg("reconciler: failed to rewrite tenant endpoints yaml")
			}
		}

		live, err := prov.ListTenantContainers(ctx, tenant.ID)
		if err != nil {
			log.Warn().Err(err).Str("tenantId", tenant.ID).Msg("reconciler: list containers failed")
			continue
		}
		have := make(map[string]bool, len(live))
		for _, c := range live {
			have[c.Service] = true
		}

		missing := make([]string, 0)
		for svc := range expected {
			if !have[svc] {
				missing = append(missing, svc)
			}
		}
		if len(missing) == 0 {
			continue
		}

		log.Info().Str("tenantId", tenant.ID).Strs("missing", missing).Msg("reconciler: re-provisioning tenant")
		_ = repo.UpdateTenantStatusDetail(ctx, tenant.ID, models.StatusProvisioning,
			fmt.Sprintf("reconciling missing services: %v", missing))

		containerIDs, err := prov.Provision(ctx, tenant)
		if err != nil {
			log.Error().Err(err).Str("tenantId", tenant.ID).Msg("reconciler: re-provisioning failed")
			_ = repo.UpdateTenantStatusDetail(ctx, tenant.ID, models.StatusProvisionFailed,
				"reconcile failed: "+err.Error())
			continue
		}
		tenant.ContainerIDs = containerIDs
		tenant.Status = models.StatusActive
		if err := repo.UpdateTenantAfterProvision(ctx, tenant); err != nil {
			log.Error().Err(err).Str("tenantId", tenant.ID).Msg("reconciler: db update failed")
			continue
		}
		_ = repo.UpdateTenantStatusDetail(ctx, tenant.ID, models.StatusActive, "")
	}
	return nil
}
