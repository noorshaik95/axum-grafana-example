package handlers

import (
	"context"

	"slate/services/metrics-service/internal/models"
	"slate/services/metrics-service/internal/repository"
	"slate/services/metrics-service/internal/service"
)

// repoPlatformSource adapts the repository to service.PlatformSource. It
// lives here so the repository package does not depend on the service
// package (which would create an import cycle).
type repoPlatformSource struct {
	repo *repository.Repository
}

func newRepoPlatformSource(r *repository.Repository) *repoPlatformSource {
	return &repoPlatformSource{repo: r}
}

func (s *repoPlatformSource) GetPlatformMetrics(ctx context.Context) (*models.PlatformMetrics, error) {
	return s.repo.GetPlatformMetrics(ctx)
}

func (s *repoPlatformSource) GetTenantMAU(ctx context.Context) ([]service.TenantMAU, error) {
	rows, err := s.repo.GetTenantMAU(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]service.TenantMAU, len(rows))
	for i, r := range rows {
		out[i] = service.TenantMAU{TenantID: r.TenantID, MAU: r.MAU}
	}
	return out, nil
}

func (s *repoPlatformSource) GetSignupsLast30d(ctx context.Context) (int, error) {
	return s.repo.GetSignupsLast30d(ctx)
}

func (s *repoPlatformSource) GetUptimePct(ctx context.Context) (float64, error) {
	return s.repo.GetUptimePct(ctx)
}

// repoDataSource adapts the repository to service.DataSource.
type repoDataSource struct {
	repo *repository.Repository
}

func newRepoDataSource(r *repository.Repository) *repoDataSource {
	return &repoDataSource{repo: r}
}

func (s *repoDataSource) GradebookRows(ctx context.Context, tenantID, courseID string) ([][]string, error) {
	return s.repo.GradebookRows(ctx, tenantID, courseID)
}
func (s *repoDataSource) RosterRows(ctx context.Context, tenantID, courseID string) ([][]string, error) {
	return s.repo.RosterRows(ctx, tenantID, courseID)
}
func (s *repoDataSource) PlatformRows(ctx context.Context) ([][]string, error) {
	return s.repo.PlatformRows(ctx)
}
