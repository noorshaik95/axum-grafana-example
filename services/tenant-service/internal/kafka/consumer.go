package kafka

import (
	"context"
	"encoding/json"
	"fmt"

	"slate/services/tenant-service/internal/docker"
	"slate/services/tenant-service/internal/models"
	"slate/services/tenant-service/internal/traefik"

	"github.com/rs/zerolog/log"
	"github.com/segmentio/kafka-go"
)

// OnboardingApprovedEvent is the payload from the onboarding service.
type OnboardingApprovedEvent struct {
	OnboardingID string `json:"onboardingId"`
	TenantName   string `json:"tenantName"`
	Slug         string `json:"slug"`
	AdminEmail   string `json:"adminEmail"`
	Plan         *models.Plan `json:"plan,omitempty"`
}

// TenantProvisionedEvent is emitted after successful provisioning.
type TenantProvisionedEvent struct {
	TenantID     string   `json:"tenantId"`
	Slug         string   `json:"slug"`
	Subdomain    string   `json:"subdomain"`
	ContainerIDs []string `json:"containerIds"`
}

// TenantProvisionFailedEvent is emitted on provisioning failure.
type TenantProvisionFailedEvent struct {
	OnboardingID string `json:"onboardingId"`
	Error        string `json:"error"`
}

// TenantLifecycleEvent is a generic tenant event.
type TenantLifecycleEvent struct {
	TenantID string `json:"tenantId"`
}

// TenantRepo is the subset of repository methods needed by the consumer.
type TenantRepo interface {
	CreateTenant(ctx context.Context, tenant *models.TenantV2) error
	GetBySlug(ctx context.Context, slug string) (*models.TenantV2, error)
	UpdateTenantStatus(ctx context.Context, id, status string) error
	UpdateTenantAfterProvision(ctx context.Context, tenant *models.TenantV2) error
}

// Consumer handles Kafka message consumption for tenant lifecycle events.
type Consumer struct {
	brokers      []string
	groupID      string
	provisioner  *docker.TenantProvisioner
	traefikGen   *traefik.Generator
	repo         TenantRepo
	producer     *Producer
}

// NewConsumer creates a new Kafka consumer for tenant events.
func NewConsumer(
	brokers []string,
	groupID string,
	provisioner *docker.TenantProvisioner,
	traefikGen *traefik.Generator,
	repo TenantRepo,
	producer *Producer,
) *Consumer {
	return &Consumer{
		brokers:     brokers,
		groupID:     groupID,
		provisioner: provisioner,
		traefikGen:  traefikGen,
		repo:        repo,
		producer:    producer,
	}
}

// Run starts consuming from all relevant topics. Blocks until ctx is cancelled.
func (c *Consumer) Run(ctx context.Context) error {
	errCh := make(chan error, 2)

	go func() { errCh <- c.consumeOnboardingApproved(ctx) }()
	go func() { errCh <- c.consumeTenantDisabled(ctx) }()

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (c *Consumer) consumeOnboardingApproved(ctx context.Context) error {
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers: c.brokers,
		Topic:   "onboarding.approved",
		GroupID: c.groupID,
	})
	defer reader.Close()

	for {
		msg, err := reader.ReadMessage(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return nil
			}
			log.Error().Err(err).Msg("error reading onboarding.approved message")
			continue
		}

		var event OnboardingApprovedEvent
		if err := json.Unmarshal(msg.Value, &event); err != nil {
			log.Error().Err(err).Msg("failed to unmarshal onboarding.approved event")
			continue
		}

		if err := c.handleOnboardingApproved(ctx, &event); err != nil {
			log.Error().Err(err).Str("onboardingId", event.OnboardingID).Msg("failed to handle onboarding.approved")
		}
	}
}

func (c *Consumer) consumeTenantDisabled(ctx context.Context) error {
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers: c.brokers,
		Topic:   "tenant.disabled",
		GroupID: c.groupID,
	})
	defer reader.Close()

	for {
		msg, err := reader.ReadMessage(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return nil
			}
			log.Error().Err(err).Msg("error reading tenant.disabled message")
			continue
		}

		var event TenantLifecycleEvent
		if err := json.Unmarshal(msg.Value, &event); err != nil {
			log.Error().Err(err).Msg("failed to unmarshal tenant.disabled event")
			continue
		}

		if err := c.handleTenantDisabled(ctx, event.TenantID); err != nil {
			log.Error().Err(err).Str("tenantId", event.TenantID).Msg("failed to handle tenant.disabled")
		}
	}
}

func (c *Consumer) handleOnboardingApproved(ctx context.Context, event *OnboardingApprovedEvent) error {
	// Idempotency: check if tenant with this slug already exists
	existing, _ := c.repo.GetBySlug(ctx, event.Slug)
	if existing != nil {
		log.Info().Str("slug", event.Slug).Msg("tenant already provisioned, skipping")
		return nil
	}

	subdomain := fmt.Sprintf("%s.slate.local", event.Slug)
	plan := event.Plan
	if plan == nil {
		plan = &models.Plan{
			StorageGB:  10,
			MaxUsers:   50,
			MaxCourses: 25,
			Features:   []string{"api_access", "basic_analytics"},
		}
	}

	tenant := &models.TenantV2{
		Slug:       event.Slug,
		Name:       event.TenantName,
		AdminEmail: event.AdminEmail,
		Status:     models.StatusProvisioning,
		Plan:       *plan,
		Subdomain:  subdomain,
	}

	if err := c.repo.CreateTenant(ctx, tenant); err != nil {
		c.producer.ProduceProvisionFailed(ctx, event.OnboardingID, err.Error())
		return fmt.Errorf("failed to create tenant record: %w", err)
	}

	// Write Traefik dynamic config
	if err := c.traefikGen.WriteConfig(tenant.ID, tenant.Slug); err != nil {
		log.Error().Err(err).Msg("failed to write traefik config")
	}

	// Provision Docker containers
	containerIDs, err := c.provisioner.Provision(ctx, tenant)
	if err != nil {
		c.repo.UpdateTenantStatus(ctx, tenant.ID, "failed")
		c.producer.ProduceProvisionFailed(ctx, event.OnboardingID, err.Error())
		return fmt.Errorf("docker provisioning failed: %w", err)
	}

	tenant.ContainerIDs = containerIDs
	tenant.Status = models.StatusActive
	if err := c.repo.UpdateTenantAfterProvision(ctx, tenant); err != nil {
		return fmt.Errorf("failed to update tenant after provision: %w", err)
	}

	c.producer.ProduceTenantProvisioned(ctx, &TenantProvisionedEvent{
		TenantID:     tenant.ID,
		Slug:         tenant.Slug,
		Subdomain:    tenant.Subdomain,
		ContainerIDs: containerIDs,
	})

	log.Info().Str("tenantId", tenant.ID).Str("slug", tenant.Slug).Msg("tenant provisioned successfully")
	return nil
}

func (c *Consumer) handleTenantDisabled(ctx context.Context, tenantID string) error {
	if err := c.provisioner.StopContainers(ctx, tenantID); err != nil {
		return fmt.Errorf("failed to stop containers: %w", err)
	}

	if err := c.repo.UpdateTenantStatus(ctx, tenantID, models.StatusSuspended); err != nil {
		return fmt.Errorf("failed to update tenant status: %w", err)
	}

	log.Info().Str("tenantId", tenantID).Msg("tenant containers stopped (disabled)")
	return nil
}
