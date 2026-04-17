package docker

import (
	"context"
	"fmt"
	"strings"

	"slate/services/tenant-service/internal/models"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	"github.com/rs/zerolog/log"
)

// ServiceImage maps service names to their Docker image references.
var ServiceImages = map[string]string{
	"course-service":              "slate/course-service:latest",
	"assignment-grading-service":  "slate/assignment-grading-service:latest",
	"content-management-service":  "slate/content-management-service:latest",
}

// TenantProvisioner manages per-tenant Docker container lifecycle.
type TenantProvisioner struct {
	dockerClient *client.Client
	networkName  string
}

// NewProvisioner creates a TenantProvisioner connected to the local Docker daemon.
func NewProvisioner(networkName string) (*TenantProvisioner, error) {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, fmt.Errorf("failed to create docker client: %w", err)
	}

	return &TenantProvisioner{
		dockerClient: cli,
		networkName:  networkName,
	}, nil
}

// Provision creates and starts Docker containers for a new tenant.
// It returns the list of container IDs that were started.
func (p *TenantProvisioner) Provision(ctx context.Context, tenant *models.TenantV2) ([]string, error) {
	var containerIDs []string

	for svcName, image := range ServiceImages {
		containerName := fmt.Sprintf("%s-%s", svcName, tenant.ID)
		schema := fmt.Sprintf("tenant_%s", strings.ReplaceAll(tenant.ID, "-", "_"))

		cfg := &container.Config{
			Image: image,
			Env: []string{
				fmt.Sprintf("TENANT_ID=%s", tenant.ID),
				fmt.Sprintf("DB_SCHEMA=%s", schema),
			},
			Labels: map[string]string{
				"tenant_id":   tenant.ID,
				"tenant_slug": tenant.Slug,
				"service":     svcName,
				"managed_by":  "tenant-service",
				// Traefik labels for per-tenant routing
				"traefik.enable": "true",
				fmt.Sprintf("traefik.http.routers.%s.rule", containerName):                           fmt.Sprintf("Host(`%s.slate.local`) && PathPrefix(`/%s`)", tenant.Slug, svcName),
				fmt.Sprintf("traefik.http.routers.%s.middlewares", containerName):                     fmt.Sprintf("inject-tenant-%s", tenant.ID),
				fmt.Sprintf("traefik.http.middlewares.inject-tenant-%s.headers.customrequestheaders.X-Tenant-ID", tenant.ID): tenant.ID,
			},
		}

		hostCfg := &container.HostConfig{
			RestartPolicy: container.RestartPolicy{Name: "unless-stopped"},
		}

		resp, err := p.dockerClient.ContainerCreate(ctx, cfg, hostCfg, nil, nil, containerName)
		if err != nil {
			// Roll back containers already created in this batch
			for _, id := range containerIDs {
				_ = p.dockerClient.ContainerRemove(ctx, id, container.RemoveOptions{Force: true})
			}
			return nil, fmt.Errorf("failed to create container %s: %w", containerName, err)
		}

		// Connect to the shared network
		if err := p.dockerClient.NetworkConnect(ctx, p.networkName, resp.ID, &network.EndpointSettings{}); err != nil {
			log.Warn().Err(err).Str("container", containerName).Msg("failed to connect container to network, continuing")
		}

		if err := p.dockerClient.ContainerStart(ctx, resp.ID, container.StartOptions{}); err != nil {
			for _, id := range containerIDs {
				_ = p.dockerClient.ContainerRemove(ctx, id, container.RemoveOptions{Force: true})
			}
			_ = p.dockerClient.ContainerRemove(ctx, resp.ID, container.RemoveOptions{Force: true})
			return nil, fmt.Errorf("failed to start container %s: %w", containerName, err)
		}

		containerIDs = append(containerIDs, resp.ID)
		log.Info().Str("container", containerName).Str("id", resp.ID).Msg("tenant container started")
	}

	return containerIDs, nil
}

// Deprovision stops and removes all containers belonging to the given tenant.
// Volumes are preserved for data retention.
func (p *TenantProvisioner) Deprovision(ctx context.Context, tenantID string) error {
	return p.stopContainers(ctx, tenantID, true)
}

// StopContainers stops (but does not remove) all containers for a tenant.
func (p *TenantProvisioner) StopContainers(ctx context.Context, tenantID string) error {
	return p.stopContainers(ctx, tenantID, false)
}

// StartContainers restarts all stopped containers for a tenant.
func (p *TenantProvisioner) StartContainers(ctx context.Context, tenantID string) error {
	f := filters.NewArgs()
	f.Add("label", fmt.Sprintf("tenant_id=%s", tenantID))
	f.Add("label", "managed_by=tenant-service")

	containers, err := p.dockerClient.ContainerList(ctx, container.ListOptions{Filters: f, All: true})
	if err != nil {
		return fmt.Errorf("failed to list tenant containers: %w", err)
	}

	for _, c := range containers {
		if err := p.dockerClient.ContainerStart(ctx, c.ID, container.StartOptions{}); err != nil {
			log.Error().Err(err).Str("container", c.ID).Msg("failed to start container")
		}
	}
	return nil
}

func (p *TenantProvisioner) stopContainers(ctx context.Context, tenantID string, remove bool) error {
	f := filters.NewArgs()
	f.Add("label", fmt.Sprintf("tenant_id=%s", tenantID))
	f.Add("label", "managed_by=tenant-service")

	containers, err := p.dockerClient.ContainerList(ctx, container.ListOptions{Filters: f, All: true})
	if err != nil {
		return fmt.Errorf("failed to list tenant containers: %w", err)
	}

	for _, c := range containers {
		if err := p.dockerClient.ContainerStop(ctx, c.ID, container.StopOptions{}); err != nil {
			log.Error().Err(err).Str("container", c.ID).Msg("failed to stop container")
		}
		if remove {
			if err := p.dockerClient.ContainerRemove(ctx, c.ID, container.RemoveOptions{}); err != nil {
				log.Error().Err(err).Str("container", c.ID).Msg("failed to remove container")
			}
		}
	}

	return nil
}

// Close releases the Docker client resources.
func (p *TenantProvisioner) Close() error {
	return p.dockerClient.Close()
}
