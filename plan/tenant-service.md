# Tenant Service Plan — Docker Provisioner + Per-Tenant Isolation

## Owner Agent: `tenant-expert`

## Stack: Go, gRPC, PostgreSQL, Docker SDK (dockerode via Go Docker client)

---

## Objective

Upgrade tenant-service to: (1) respond to `onboarding.approved` Kafka events by provisioning per-tenant Docker containers, (2) generate Traefik dynamic config per tenant, (3) implement all CRUD/management APIs, (4) handle tenant enable/disable lifecycle.

---

## Current State

- Basic tenant provisioning + circuit breaker
- Missing Docker provisioner, Traefik integration, full API

---

## Target APIs

### REST API (port 8085)

```
POST   /tenants                    — provision tenant (from onboarding approval)
GET    /tenants                    — list all with usage stats (paginated)
GET    /tenants/:id                — full profile
PATCH  /tenants/:id/access         — enable/disable (with reason)
PUT    /tenants/:id/plan           — update quotas/feature flags
GET    /tenants/:id/usage          — current resource usage
DELETE /tenants/:id                — soft delete + deprovision containers
```

### gRPC Service (port 50055)

Proto: `proto/tenant.proto` — update with full CRUD + provisioning RPCs

---

## Deliverables

### 1. Docker Provisioner (`internal/docker/provisioner.go`)

```go
type TenantProvisioner struct {
    client      *client.Client
    networkName string
    imageTag    string
}

// Called when onboarding.approved event received
func (p *TenantProvisioner) Provision(ctx context.Context, tenant *Tenant) error {
    // 1. Create tenant DB schema
    // 2. Start per-tenant containers
    // 3. Apply Traefik labels
    // 4. Write Traefik dynamic config
    // 5. Emit tenant.provisioned Kafka event
}

// Containers spawned per tenant:
// - course-service-{tenantId}
// - assignment-grading-service-{tenantId}
// - content-management-service-{tenantId}
// Each with:
//   TENANT_ID env var
//   DB_SCHEMA = "tenant_{tenantId}"
//   Traefik labels for {slug}.slate.local routing
//   Connected to slate-network
```

### 2. Traefik Config Generator (`internal/traefik/generator.go`)

Writes `config/traefik/dynamic/tenant-{id}.yml` with:

- HTTP router rule: `Host("{slug}.slate.local")`
- Middleware: inject `X-Tenant-ID` header
- Service pointing to api-gateway (tenant JWT validated there)

### 3. PostgreSQL Schema Initialization

On tenant provisioning:

```sql
CREATE SCHEMA IF NOT EXISTS tenant_{tenantId};
-- Grant app user access
GRANT ALL ON SCHEMA tenant_{tenantId} TO appuser;
-- Run per-service migrations in that schema
```

### 4. Kafka Consumer

Topic: `onboarding.approved`
Handler:

```go
func handleOnboardingApproved(event OnboardingApprovedEvent) {
    // Extract tenant details
    // Call Provision()
    // Emit tenant.provisioned or tenant.provision_failed
}
```

Also consume:

- `tenant.disabled` → stop tenant containers, emit to course/assignment/content services
- `tenant.enabled` → restart containers

### 5. Kafka Events Produced

- `tenant.provisioned` — with tenantId, slug, subdomain, containerIds
- `tenant.provision_failed` — triggers Restate compensation
- `tenant.disabled` — consumed by all services to lock tenant data
- `tenant.enabled`
- `tenant.plan.updated`
- `tenant.deleted`

### 6. Tenant Model

```go
type Tenant struct {
    ID           uuid.UUID
    Slug         string        // URL-safe: "mit-university"
    Name         string
    AdminEmail   string
    Status       TenantStatus  // Provisioning | Active | Suspended | Deleted
    Plan         Plan          // storage_gb, max_users, max_courses, features
    Usage        Usage         // current storage_gb, user_count, course_count
    Subdomain    string        // "{slug}.slate.local"
    ContainerIDs []string      // running container IDs
    CreatedAt    time.Time
    UpdatedAt    time.Time
}
```

### 7. Tests

- Unit: provisioner logic (mock Docker client)
- Integration: full tenant lifecycle (provision → usage → disable → enable → delete)
- Kafka: onboarding.approved → tenant.provisioned flow
- Traefik: generated config is valid YAML with correct rules

---

## Files to Create/Modify

- [NEW] `services/tenant-service/internal/docker/provisioner.go`
- [NEW] `services/tenant-service/internal/traefik/generator.go`
- [NEW] `services/tenant-service/internal/kafka/consumer.go`
- [MODIFY] `services/tenant-service/internal/handlers/` — full CRUD
- [MODIFY] `services/tenant-service/migrations/` — tenant schema management
- [MODIFY] `proto/tenant.proto` — full gRPC definitions
- [MODIFY] `docker-compose.yml` — mount Docker socket into tenant-service
