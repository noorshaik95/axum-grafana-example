# Slate LMS — Architecture & Infrastructure Design

> **Purpose**: Comprehensive reference for engineers and AI agents working on this codebase.
> Covers auth separation, multi-tenant routing, impersonation, config persistence, IaaS
> abstraction, the local → GCP deployment path, Redis caching strategy, and all new
> services introduced in the April 2026 UI overhaul.
>
> **Last updated**: April 2026  
> **Status**: Design document. Sections marked 🔨 are not yet implemented.

---

## Table of Contents

1. [System Overview](#1-system-overview)
2. [Auth Separation: Platform Admin vs Tenant Users](#2-auth-separation)
3. [Service Topology](#3-service-topology)
4. [Request Routing](#4-request-routing)
5. [Tenant Provisioning Flow](#5-tenant-provisioning-flow)
6. [Impersonation Flow](#6-impersonation-flow)
7. [Config Persistence](#7-config-persistence)
8. [IaaS Abstraction Layer](#8-iaas-abstraction-layer)
9. [GCP Setup](#9-gcp-setup)
10. [Local → GCP Migration Path](#10-local--gcp-migration-path)
11. [Redis Caching Strategy](#11-redis-caching-strategy)
12. [New Services (April 2026)](#12-new-services-april-2026)
13. [Existing Service Updates (April 2026)](#13-existing-service-updates-april-2026)
14. [Frontend Route Map](#14-frontend-route-map)
15. [Open Questions & Future Decisions](#15-open-questions--future-decisions)

---

## 1. System Overview

Slate LMS is a **multi-tenant LMS platform** where each university (tenant) runs in
isolated containers on its own Docker network. A single platform admin layer manages
all tenants. There is no shared "monolith" — the platform admin and tenant users are
completely separated at the auth, routing, and container level.

```
Internet
    │
    ▼
 Traefik  (reverse proxy, subdomain router)
    │
    ├── admin.slate.local  ──────► Platform API Gateway ──► Platform Services
    │                                                         (tenant-service, onboarding,
    │                                                          metrics, email, admin-auth,
    │                                                          incident-service, feature-flag-service)
    │
    ├── eastfield.slate.local ──► Platform API Gateway ──► Eastfield containers
    │                              (+ X-Tenant-ID header)    (user-auth, course,
    │                                                          assignment, content, video,
    │                                                          discussion, scheduling, ai)
    │
    └── riverside.slate.local ──► Platform API Gateway ──► Riverside containers
```

Key principles:

- **Platform services** run once, always on. They manage tenants, not content.
- **Tenant services** are provisioned per university. One set of containers per tenant.
- **Platform API gateway** routes traffic to either platform services or the correct
  tenant's containers based on the subdomain (`X-Tenant-ID` injected by Traefik).
- **Auth is fully separated**: platform admins authenticate against a dedicated
  admin-auth-service; university users authenticate against their own tenant's
  user-auth-service instance.
- **Redis** is a shared platform infrastructure component used by both platform and
  tenant services via namespaced keys (`tenant:{slug}:{key}`).

---

## 2. Auth Separation

### Current state (wrong)

One shared `user-auth-service` handles authentication for both platform admins and
tenant users. This couples tenant data to the platform and breaks isolation.

### Target design

```
admin.slate.local     →  admin-auth-service   (platform-level, always running)
eastfield.slate.local →  user-auth-service    (tenant-level, provisioned per university)
riverside.slate.local →  user-auth-service    (different instance, different DB schema)
```

**admin-auth-service** (new, platform-level):

- Authenticates only platform admins (superadmins, billing staff, support).
- Stores users in a dedicated `platform_admin` PostgreSQL schema — never mixed with
  tenant data.
- Issues JWT tokens with `aud: platform` claim.
- Responsible for generating **impersonation tokens** (see §6).
- Lives in `services/admin-auth-service/` (to be created).
- Port: `50060` (gRPC), `8090` (HTTP).

**user-auth-service** (existing, tenant-level):

- One instance per tenant, provisioned by `tenant-service`.
- Container name: `user-auth-{tenant-slug}` (e.g. `user-auth-eastfield`).
- Each instance gets its own `TENANT_ID` and `DB_SCHEMA` env vars.
- Issues JWT tokens with `aud: tenant` and `tenant_id: {id}` claims.
- Validates impersonation tokens using the platform public key (injected at
  provisioning time via `PLATFORM_PUBLIC_KEY` env var).
- Supports SSO: SAML (Shibboleth), OIDC (Okta, Azure AD), Google Workspace.

### JWT claim structure

**Platform admin token:**

```json
{
  "sub": "admin-user-uuid",
  "aud": "platform",
  "roles": ["superadmin"],
  "iat": 1714000000,
  "exp": 1714003600
}
```

**Tenant user token:**

```json
{
  "sub": "user-uuid",
  "aud": "tenant",
  "tenant_id": "eastfield-uuid",
  "tenant_slug": "eastfield",
  "roles": ["student"],
  "iat": 1714000000,
  "exp": 1714003600
}
```

**Impersonated session token (issued by tenant's user-auth-service after validating
the impersonation token):**

```json
{
  "sub": "student-uuid",
  "aud": "tenant",
  "tenant_id": "eastfield-uuid",
  "tenant_slug": "eastfield",
  "roles": ["student"],
  "impersonated_by": "admin-user-uuid",
  "impersonation_id": "session-uuid",
  "iat": 1714000000,
  "exp": 1714001800
}
```

---

## 3. Service Topology

### Platform services (always running, on `slate-network`)

| Service                   | Language     | gRPC Port | HTTP Port | Purpose                                       |
| ------------------------- | ------------ | --------- | --------- | --------------------------------------------- |
| `api-gateway`             | Rust/Axum    | —         | 8080      | Routes all traffic                            |
| `admin-auth-service` 🔨   | Go           | 50060     | 8090      | Platform admin authentication + impersonation |
| `tenant-service`          | Go           | 50057     | 8087      | Provisions tenant containers                  |
| `onboarding-service`      | Rust/Restate | —         | 8084      | Durable onboarding workflows + LMS migration  |
| `email-service`           | Go           | 50058     | 8088      | Platform email, broadcast messages            |
| `metrics-service`         | Go           | 50059     | 8089      | Platform-wide analytics + reports             |
| `incident-service` 🔨     | Go           | 50061     | 8091      | Incident tickets P1–P4, triage, timeline      |
| `feature-flag-service` 🔨 | Go           | 50062     | 8092      | Per-tenant/per-role feature flags             |

### Platform infrastructure (always running, on `slate-network`)

| Component     | Purpose                                                            |
| ------------- | ------------------------------------------------------------------ |
| PostgreSQL    | Primary relational store (schema-per-tenant)                       |
| **Redis**     | Token cache, feature flags, analytics aggregates, resume positions |
| MongoDB       | Content metadata (course materials, attachments)                   |
| Kafka         | Async events between services                                      |
| MinIO         | File storage (uploads, video content)                              |
| Elasticsearch | Full-text search (courses, users, discussions)                     |
| Tempo         | Distributed tracing                                                |
| Prometheus    | Metrics collection                                                 |
| Grafana       | Dashboards                                                         |
| Loki          | Log aggregation                                                    |
| Traefik       | Reverse proxy + subdomain routing                                  |
| Restate       | Durable workflow orchestration                                     |

### Tenant services (provisioned per university, on `tenant-{slug}-network`)

| Service                | Language | gRPC Port (internal) | Purpose                                                   |
| ---------------------- | -------- | -------------------- | --------------------------------------------------------- |
| `user-auth-{slug}`     | Go       | 50051                | University user auth + roles + SSO                        |
| `course-{slug}`        | NestJS   | 50052                | Courses, modules, lessons, syllabus, analytics            |
| `assignment-{slug}`    | Go       | 50053                | Assignments, submissions, grading queue, batch grading    |
| `content-{slug}`       | Rust     | 50054                | File uploads, video streaming, resume positions           |
| `video-{slug}`         | Rust     | 50055                | Live lectures, office hours video rooms, Zoom integration |
| `discussion-{slug}` 🔨 | Go       | 50056                | Discussion threads, mentions, inbox messages              |
| `scheduling-{slug}` 🔨 | Go       | 50063                | Office hours booking, calendar slots                      |
| `ai-{slug}` 🔨         | Go       | 50064                | Study plans, grade projection, NL search, AI feedback     |

All tenant containers:

- Named `{service}-{slug}` (predictable, used by api-gateway for routing).
- Connected to `tenant-{slug}-network` (isolated from other tenants).
- Api-gateway is dynamically connected to each tenant's network when provisioned.
- Receive env vars: `TENANT_ID`, `TENANT_SLUG`, `DB_SCHEMA=tenant_{slug}`.
- Access shared Redis via `REDIS_URL` (keys namespaced by `tenant:{slug}:`).

---

## 4. Request Routing

### How Traefik routes tenant traffic

When a tenant is provisioned, `tenant-service` writes a YAML file to
`config/traefik/dynamic/tenant-{slug}.yml`:

```yaml
http:
  routers:
    eastfield:
      rule: 'Host(`eastfield.slate.local`)'
      entryPoints: [web]
      service: api-gateway
      middlewares: [inject-eastfield-tenant-id]

  middlewares:
    inject-eastfield-tenant-id:
      headers:
        customRequestHeaders:
          X-Tenant-ID: 'eastfield-uuid'
          X-Tenant-Slug: 'eastfield'

  services:
    api-gateway:
      loadBalancer:
        servers:
          - url: 'http://api-gateway:8080'
```

Traefik watches the `dynamic/` directory and hot-reloads on file change.

### How api-gateway routes to the correct tenant containers 🔨

The api-gateway receives `X-Tenant-ID` and `X-Tenant-Slug` headers from Traefik.
It uses the slug to resolve the correct container endpoints:

```
Request to eastfield.slate.local/api/auth/login
  → Traefik injects X-Tenant-Slug: eastfield
  → Api-gateway reads header
  → Resolves endpoint: user-auth-eastfield:50051
  → Forwards gRPC call
```

Implementation approach: **per-tenant config files** loaded at startup and
hot-reloaded on change. The `tenant-service` writes
`config/tenants/{slug}.yaml` alongside the Traefik config:

```yaml
# config/tenants/eastfield.yaml
tenant_id: 'eastfield-uuid'
slug: 'eastfield'
services:
  user-auth: 'http://user-auth-eastfield:50051'
  course: 'http://course-eastfield:50052'
  assignment: 'http://assignment-eastfield:50053'
  content: 'http://content-eastfield:50054'
  video: 'http://video-eastfield:50055'
  discussion: 'http://discussion-eastfield:50056'
  scheduling: 'http://scheduling-eastfield:50063'
  ai: 'http://ai-eastfield:50064'
```

Api-gateway watches `config/tenants/` and updates its routing table.

### Platform admin routing

Requests to `admin.slate.local` never have `X-Tenant-Slug`. The api-gateway
routes them to platform services (tenant-service, metrics-service, incident-service,
etc.) using the static `gateway-config.yaml`.

---

## 5. Tenant Provisioning Flow

### Trigger: onboarding workflow (Restate)

```
Admin submits onboarding form
  → POST /api/onboarding  (admin.slate.local)
  → onboarding-service creates Restate workflow
  → Workflow runs: validate → SSO config → 72h approval gate → approve
  → Kafka event: onboarding.approved { tenant_id, slug, admin_email, plan }
  → tenant-service Kafka consumer receives event
  → Begins provisioning
```

### Provisioning steps (tenant-service)

```
1. Create tenant record in DB (status: provisioning)
2. Create Docker network: tenant-{slug}-network
3. Pull images (or use pre-built local images in dev)
4. Start 8 containers on tenant-{slug}-network:
     user-auth-{slug}, course-{slug}, assignment-{slug},
     content-{slug}, video-{slug}, discussion-{slug},
     scheduling-{slug}, ai-{slug}
   Each container gets:
     TENANT_ID, TENANT_SLUG, DB_SCHEMA=tenant_{slug},
     PLATFORM_PUBLIC_KEY (for impersonation token validation),
     REDIS_URL (shared Redis, keys namespaced by tenant),
     KAFKA_BROKERS, MINIO_ENDPOINT, DB_DSN
5. Connect api-gateway container to tenant-{slug}-network
6. Write config/traefik/dynamic/tenant-{slug}.yml
7. Write config/tenants/{slug}.yaml  (api-gateway tenant routing)
8. Run DB schema migration for this tenant:
     CREATE SCHEMA IF NOT EXISTS tenant_{slug};
     (each tenant service handles its own schema on startup)
9. Update tenant record: status=active, container_ids=[...], subdomain=...
10. Emit Kafka event: tenant.provisioned
11. email-service sends welcome email to admin_email
```

### Deprovisioning steps

```
1. Update tenant status: deleted
2. Stop and remove 8 containers
3. Disconnect api-gateway from tenant network
4. Remove Docker network
5. Delete config/traefik/dynamic/tenant-{slug}.yml
6. Delete config/tenants/{slug}.yaml
7. Flush Redis keys: DEL tenant:{slug}:*
8. (Optional) Drop tenant DB schemas after data retention period
9. Emit Kafka event: tenant.deprovisioned
```

---

## 6. Impersonation Flow

Platform admins can impersonate any tenant user for support/debugging without
knowing their password. The impersonation session is scoped to the tenant's
subdomain and has a short TTL.

```
1. Admin logs into admin.slate.local (admin-auth-service)
   → Gets platform JWT (aud: platform)

2. Admin navigates to Schools → Stanford → Users → Dr. R. Khan → Impersonate

3. POST admin.slate.local/api/tenants/{tenant_id}/impersonate/{user_id}
   Authorization: Bearer <platform-admin-jwt>

   admin-auth-service generates impersonation token:
   {
     "type": "impersonation",
     "tenant_id": "eastfield-uuid",
     "tenant_slug": "eastfield",
     "user_id": "student-uuid",
     "impersonator_id": "admin-uuid",
     "impersonation_id": "unique-session-uuid",
     "exp": now + 5 minutes   ← short-lived
   }
   Signed with platform RSA private key (never leaves platform services).

4. Response:
   { "redirect_url": "https://eastfield.slate.local/auth/impersonate?token=<token>" }

5. Browser redirects to eastfield.slate.local/auth/impersonate?token=...

6. Eastfield's user-auth-service receives token:
   - Verifies signature using PLATFORM_PUBLIC_KEY env var
   - Checks exp, tenant_id, type
   - Logs impersonation event to audit table (immutable append-only)
   - Issues a normal tenant JWT with impersonated_by claim
   - Sets TTL to 30 minutes (not renewable)

7. Admin operates as the user within eastfield.slate.local
   - All actions are audit-logged with impersonation_id
   - UI shows a banner: "Impersonating alice.johnson@eastfield.edu"
   - Admin can end session, which revokes the impersonated JWT

8. End session: DELETE eastfield.slate.local/api/auth/impersonate/{impersonation_id}
   → user-auth-service adds impersonation_id to Redis revocation set (TTL: 2h)
   → Subsequent requests with that token are rejected
```

**Security properties:**

- Impersonation token is valid for 5 minutes (one-time use, exchanged for session).
- Session token is valid for 30 minutes, non-renewable.
- Impersonation events are immutably logged with timestamp, admin ID, user ID.
- Platform private key never touches tenant services.
- Tenant user-auth-service only receives the public key.

---

## 7. Config Persistence

Tenant configurations exist in three forms. All must persist across restarts.

### 7.1 Traefik dynamic configs

**Local**: Written to `config/traefik/dynamic/tenant-{slug}.yml`, which is a bind
mount from the host. Files survive container restarts because they're on the host
filesystem.

**GCP**: Written to a GCS bucket (`gs://slate-traefik-configs/dynamic/`). Traefik
uses the `file` provider pointed at a local directory; a sidecar process syncs the
GCS bucket to that directory on boot and watches for changes.

### 7.2 Api-gateway tenant routing configs

**Local**: Written to `config/tenants/{slug}.yaml` (host bind mount).

**GCP**: Same approach as Traefik configs — GCS-backed, synced to local directory.
Api-gateway watches the directory with `inotify`/`notify` and reloads routing table.

### 7.3 Tenant metadata in database

All tenant records (id, slug, name, status, plan, container IDs, subdomain) stored
in the `tenants_v2` table in PostgreSQL.

**Local**: Postgres Docker container with named volume (`postgres-data`).

**GCP**: Cloud SQL (PostgreSQL). The `tenants_v2` table is the source of truth.
On platform restart, tenant-service reads all `active` tenants, verifies their
containers are running, and reconciles any drift.

### Reconciliation on startup 🔨

On startup, `tenant-service` should:

1. Query all tenants with `status = active`.
2. Check if each tenant's containers are running.
3. If a container is missing, re-provision it.
4. Re-write Traefik/api-gateway configs if the files are missing.

---

## 8. IaaS Abstraction Layer

The current `DockerProvisioner` in `tenant-service` is tightly coupled to the
Docker SDK. To support GCP, the provisioner must be behind an interface.

### Interface definition (Go)

```go
// services/tenant-service/internal/provisioner/provisioner.go

type ProvisionRequest struct {
    TenantID          string
    Slug              string
    Plan              models.Plan
    PlatformPublicKey string
    DBConnectionDSN   string
    RedisURL          string
    KafkaBrokers      []string
    MinIOEndpoint     string
}

type ProvisionResult struct {
    ContainerIDs []string
    NetworkID    string
    Endpoints    map[string]string  // service → endpoint URL
}

type Provisioner interface {
    Provision(ctx context.Context, req ProvisionRequest) (*ProvisionResult, error)
    Deprovision(ctx context.Context, tenantID string) error
    Start(ctx context.Context, tenantID string) error
    Stop(ctx context.Context, tenantID string) error
    GetStatus(ctx context.Context, tenantID string) (string, error)
}
```

### Implementations

**`DockerProvisioner`** (existing, local dev):

- Uses Docker SDK to create containers and networks on the local daemon.
- Configured via `DOCKER_SOCKET=/var/run/docker.sock`.

**`GCEProvisioner`** 🔨 (GCP Compute Engine):

- Uses Docker SDK pointed at a remote GCE instance via TCP.
- Configured via `GCE_DOCKER_HOST=tcp://{instance-ip}:2376`.

**`CloudRunProvisioner`** 🔨 (GCP Cloud Run — future):

- Uses Google Cloud Run Admin API to deploy one Cloud Run service per tenant service.
- Services named `user-auth-{slug}`, `course-{slug}`, etc.

### Selecting the implementation

Configured via `PROVISIONER_BACKEND` env var:

- `docker` → DockerProvisioner (default, local dev)
- `gce` → GCEProvisioner (GCP Compute Engine)
- `cloud-run` → CloudRunProvisioner (GCP Cloud Run, future)

---

## 9. GCP Setup

### 9.1 Without a domain: using sslip.io

sslip.io is a free wildcard DNS service that maps any IP-encoded hostname to that IP.
It is the current hosting provider for the nip.io service (nip.io now runs on sslip.io
infrastructure), so both hostnames work, but **sslip.io is preferred**.

```
admin.34.100.200.50.sslip.io       →  34.100.200.50
eastfield.34.100.200.50.sslip.io   →  34.100.200.50
riverside.34.100.200.50.sslip.io   →  34.100.200.50
```

**How to use:**

1. Provision a GCE instance, note its external IP (e.g., `34.100.200.50`).
2. Set Traefik to accept `*.{IP}.sslip.io` routes.
3. Update `DOMAIN_SUFFIX` env var in tenant-service from `slate.local` to
   `34.100.200.50.sslip.io`.
4. No code changes needed.

When a real domain is acquired (e.g., `slate.io`):

- Change `DOMAIN_SUFFIX` to `slate.io`.
- Set up a wildcard DNS record: `*.slate.io → GCE IP`.
- No code changes needed.

### 9.2 GCP services used

| GCP Service                | Purpose                                               |
| -------------------------- | ----------------------------------------------------- |
| **Compute Engine (GCE)**   | Run platform services + tenant containers via Docker  |
| **Cloud SQL (PostgreSQL)** | Managed DB replacing local postgres container         |
| **Cloud Storage (GCS)**    | Config persistence (Traefik/tenant YAML files)        |
| **Secret Manager**         | JWT secrets, OAuth credentials, platform private key  |
| **VPC Networks**           | Network isolation per tenant                          |
| **Cloud NAT**              | Outbound internet for containers without external IPs |
| **Artifact Registry**      | Docker image registry for all service images          |
| **Cloud Logging**          | Log aggregation (replaces Loki)                       |
| **Cloud Monitoring**       | Metrics (replaces Prometheus+Grafana)                 |

### 9.3 GCE instance setup

```
Machine type: e2-standard-4 (4 vCPU, 16GB RAM) — enough for dev
OS:           Container-Optimized OS or Debian 12 with Docker
Disk:         100GB SSD persistent disk
Network:      External IP (static), firewall rules: 80/443 open
```

### 9.4 Firewall rules

```
Ingress: allow TCP 80  from 0.0.0.0/0   (HTTP, Traefik)
Ingress: allow TCP 443 from 0.0.0.0/0   (HTTPS, future)
Ingress: allow TCP 22  from your-ip/32  (SSH)
All other ingress: denied
```

---

## 10. Local → GCP Migration Path

### Phase 1: Local dev (current state)

- Everything in one `docker-compose.yml`.
- tenant-service uses `DockerProvisioner` pointed at local Docker socket.
- DNS via dnsmasq (macOS) or `/etc/hosts` (Linux).

### Phase 2: GCP dev (next step after local works)

1. Create GCE instance, set static external IP.
2. Build Docker images, push to Artifact Registry.
3. Create `docker-compose.platform.yml` (platform services only).
4. SSH to GCE, run `docker compose -f docker-compose.platform.yml up -d`.
5. Set `DOMAIN_SUFFIX={GCE-IP}.sslip.io` in tenant-service env.
6. Set `PROVISIONER_BACKEND=docker`.
7. Verify: create a tenant, check subdomain routes correctly.

### Phase 3: Managed GCP services

- Migrate PostgreSQL to Cloud SQL.
- Migrate config storage to GCS.
- Migrate secrets to Secret Manager.

### Phase 4: Cloud Run provisioning (future)

- Implement `CloudRunProvisioner`.
- Tenant containers become Cloud Run services (auto-scaling).

---

## 11. Redis Caching Strategy

Redis is a **shared platform component** (`redis:6379`). All keys are namespaced to
prevent collision between platform and tenant data.

### Key namespacing

```
platform:auth:{token_hash}          → validated admin token (TTL: 60s)
platform:flags:{tenant_id}          → feature flags for a tenant (TTL: 5m)
platform:metrics:platform           → platform-wide stats (TTL: 5m)
platform:incidents:active           → open incidents list (TTL: 30s)

tenant:{slug}:session:{token_hash}  → validated tenant token (TTL: 60s)
tenant:{slug}:revoked:{impersonation_id} → revoked impersonation (TTL: 2h)
tenant:{slug}:grades:{user_id}:{course_id} → grade projection (TTL: 10m)
tenant:{slug}:study_plan:{user_id}  → AI study plan (TTL: 24h)
tenant:{slug}:analytics:{course_id} → course analytics aggregate (TTL: 5m)
tenant:{slug}:roster:{course_id}    → roster health signals (TTL: 2m)
tenant:{slug}:video_pos:{user_id}:{content_id} → resume position (TTL: 30m, write-through)
tenant:{slug}:cmd_palette:{user_id}:{query_hash} → NL search results (TTL: 30s)
tenant:{slug}:flags:{user_id}       → per-user feature flag overrides (TTL: 5m)
tenant:{slug}:live:{session_id}     → live lecture state (TTL: 4h, auto-extended)
tenant:{slug}:oh_slots:{instructor_id} → available office hours slots (TTL: 5m)
```

### Caching decisions per service

| Service              | What it caches                      | TTL       | Pattern                               |
| -------------------- | ----------------------------------- | --------- | ------------------------------------- |
| api-gateway          | Token validation results            | 60s       | Read-through, invalidate on logout    |
| feature-flag-service | Flag rules per tenant               | 5m        | Write-through on flag change          |
| ai-service           | Grade projections, study plans      | 10m / 24h | Lazy-load, background refresh         |
| course-service       | Analytics aggregates, roster health | 2–5m      | Write-through on grade change         |
| content-service      | Video resume positions              | 30m       | Write-through on position update      |
| video-service        | Live lecture state, Q&A queue       | 4h        | In-memory primary, Redis for failover |
| metrics-service      | Platform stats                      | 5m        | Background job writes periodically    |
| scheduling-service   | Available OH slots                  | 5m        | Invalidate on booking                 |
| incident-service     | Active incidents list               | 30s       | Write-through                         |

### Rules

- Never cache sensitive data (passwords, private keys, full JWT tokens — only hashes).
- Always set a TTL; no indefinite keys.
- Tenant deprovisioning must flush `tenant:{slug}:*` keys.
- Cache misses fall through to the source of truth (PostgreSQL / service).

---

## 12. New Services (April 2026)

These services do not yet exist and must be created.

### 12.1 `ai-service` (Go) 🔨

**Purpose**: AI-powered features using Claude API (claude-sonnet-4-6 or claude-opus-4-7).

**Features:**

- **NL command palette** (`⌘K`): Interpret free-text queries, return ranked actions/routes.
- **Grade projection** ("Path to A"): Given current grades and remaining assignments,
  compute what scores are needed to hit a target grade.
- **Study plan generation**: Parse syllabus/assignment due dates, build a 16-week
  personalized study schedule.
- **AI draft feedback**: Instructors can request AI feedback on a submission before
  publishing grades (opt-in, per feature flag).
- **Welcome/status messages**: Generate personalized "Today" view hero text for students.

**gRPC port**: 50064  
**Key endpoints** (gRPC):

```
GenerateCmdPaletteResults(query, user_context) → results[]
GetGradeProjection(user_id, course_id) → projection
GenerateStudyPlan(user_id, tenant_id) → plan
GetDraftFeedback(submission_id) → feedback
GetWelcomeMessage(user_id, today_context) → message
```

**Caching**: Grade projections cached 10m; study plans cached 24h (regenerate on
new assignment or submission). Command palette results cached 30s per query hash.

**Rate limiting**: Each tenant gets a monthly token budget. Track usage in Redis
(`tenant:{slug}:ai:tokens_used:{month}`). Reject requests when budget exhausted.

---

### 12.2 `discussion-service` (Go) 🔨

**Purpose**: Discussion threads, mentions, inbox messages, announcements.

**Features:**

- **Discussion threads** per course (replace old forum pattern).
- **Mentions** (`@username`) with notifications.
- **Inbox** for students: all threads where they're mentioned or replied to,
  professor feedback, study group messages.
- **Threads for instructors**: view all threads needing reply, sorted by age.
- **Announcements**: instructor broadcasts to a course; optional email copy.

**gRPC port**: 50056  
**DB schema**: `tenant_{slug}` — tables: `threads`, `posts`, `mentions`, `inbox_items`

**Kafka events consumed**: `assignment.graded` → creates inbox notification.

**Kafka events produced**: `discussion.mention` → triggers email-service notification.

---

### 12.3 `scheduling-service` (Go) 🔨

**Purpose**: Office hours booking between students and instructors.

**Features:**

- Instructors define recurring time slots (day of week, time, format: online/in-person).
- Students book 15-min slots; pre-fill questions from current assignment context.
- Auto-attach open assignment questions to the booking invite.
- Instructors see daily OH schedule with booked students and their pre-read questions.
- Calendar integration (ical export) — no external calendar API dependency initially.

**gRPC port**: 50063  
**DB schema**: `tenant_{slug}` — tables: `oh_schedules`, `oh_slots`, `oh_bookings`

**Caching**: Available slots cached 5m per instructor, invalidated on booking.

---

### 12.4 `incident-service` (Go, platform-level) 🔨

**Purpose**: Platform-wide incident triage and management for Slate ops team.

**Features:**

- Incident CRUD with priority levels P0–P4.
- Timeline entries (comments, status changes) per incident.
- Notify tenant admin when their tenant is involved.
- Service status page (public-facing, derived from open incidents).
- Auto-open P1 incident when a monitored metric breaches threshold
  (Kafka consumer: `metrics.threshold_breached`).

**gRPC port**: 50061  
**Lives on**: `slate-network` (platform service)  
**DB schema**: `platform` — tables: `incidents`, `incident_events`, `incident_tenants`

**Kafka events consumed**: `metrics.threshold_breached`  
**Kafka events produced**: `incident.opened`, `incident.resolved`

---

### 12.5 `feature-flag-service` (Go, platform-level) 🔨

**Purpose**: Per-tenant, per-role, and percentage-rollout feature flags for controlled
feature releases.

**Features:**

- Create/update/delete flag rules from admin panel.
- Rollout targets: all tenants, specific tenants, % of instructors, specific schools.
- Flags evaluated at request time by consuming services; cached in Redis (5m TTL).
- Admin can override flags per individual user for piloting.

**gRPC port**: 50062  
**Lives on**: `slate-network` (platform service)  
**DB schema**: `platform` — tables: `flags`, `flag_rules`

**Key flags (from design)**:

- `new_grading_queue` — new pattern-grouped grading UI for instructors
- `ai_draft_feedback` — AI feedback on submissions (opt-in)
- `study_plan_v2` — updated study plan generation
- `live_class_pulse` — live lecture Q&A and engagement pulse
- `mobile_push` — push notifications

---

### 12.6 `admin-auth-service` (Go, platform-level) 🔨

See §2 for full design. Key additions vs user-auth-service:

- `POST /api/admin/auth/impersonate/{tenant_id}/{user_id}` — generate impersonation token.
- Fixed `platform_admins` schema (no tenant scoping).
- Audit log all admin actions via Kafka event `audit.admin_action`.

---

## 13. Existing Service Updates (April 2026)

### 13.1 `course-service` (NestJS)

**New:**

- Module/lesson structure: a course has weeks → each week has lecture, reading, quiz.
- Video resume position: store `last_position_seconds` per (user, content_id).
- Live lecture mode: create lecture session, track attendance, expose real-time pulse
  (% of enrolled students present) via Redis sorted set.
- Syllabus editor: CRUD on course weeks with drag-to-reorder.
- Course analytics: engagement %, completion %, median grade, at-risk count.
  Aggregated and cached in Redis (5m TTL).

**Updated Kafka events produced:**

- `course.lecture_started`, `course.lecture_ended`
- `course.module_completed` (triggers study plan update)

---

### 13.2 `assignment-grading-service` (Go)

**New:**

- **Grading queue** with pattern grouping: cluster submissions by similarity
  (text diff / test result patterns), present to instructor grouped.
- **Batch grading**: apply a shared rubric + feedback template to a group of submissions.
- **Rubric rows**: each assignment has configurable rubric criteria (base case, core logic, edge cases, style).
- **Auto-test integration**: accept `tests.py` file at assignment creation; run tests
  against submissions; surface pass/fail per submission.
- **Draft submissions**: student can save draft mid-assignment; draft stored in DB,
  not counted as submission.
- **Starter code/attachments**: store per assignment in MinIO.
- **Grade projection feed**: emit Kafka event `grade.updated` consumed by ai-service
  to invalidate grade projection cache.

---

### 13.3 `content-management-service` (Rust)

**New:**

- **Video resume position**: `PUT /api/content/{id}/position` writes current playback
  position for the authenticated user. Read at content load; cached in Redis (30m,
  write-through).
- **Streaming range requests**: ensure HTTP range requests work for seeking within
  video content.
- **Signed URL TTL by content type**: short TTL (15m) for video chunks, longer (1h)
  for static files.

---

### 13.4 `video-conferencing-service` (Rust)

**New:**

- **Live lecture mode**: instructor starts a lecture session; students join by course
  URL (`/lecture/live`). Track attendance in Redis. Surface real-time Q&A queue.
- **Q&A queue**: students submit questions during live lecture; upvote system;
  instructor sees sorted list.
- **External video integration**: first-class Zoom OAuth integration. When a lecture
  has a Zoom link, the frontend opens Zoom directly; the service tracks join/leave
  events via Zoom webhooks.
- **Office hours rooms**: instructors define video rooms for OH slots; scheduling-service
  links bookings to room URLs.
- **Pulse metric**: percentage of enrolled students with an active session, updated
  every 30s, stored in Redis.

---

### 13.5 `metrics-service` (Go)

**New:**

- **Roster health**: per-course query returning students sorted by risk signals
  (missed assignments, grade trend, days since last activity).
- **Grade distributions**: histogram per assignment, course median.
- **Export reports**: generate CSV/XLSX of gradebook, roster, or platform stats
  (async job, stored in MinIO, signed URL returned).
- **Platform-wide stats**: MAU, signups, uptime (derived from service health checks),
  active incidents count. Cached in Redis (5m TTL).
- **Kafka consumer**: `grade.updated` → update grade distribution aggregates.

---

### 13.6 `onboarding-service` (Rust/Restate)

**New:**

- **SSO config step**: after 72h approval, guide admin through SAML/OIDC/Google
  Workspace setup as a Restate workflow step with human-in-the-loop pause.
- **LMS migration (Canvas import)**: durable multi-step workflow:
  1. OAuth grant to Canvas account.
  2. Import courses (paginated, resumable).
  3. Reconcile rosters (match by email).
  4. Verify & go live (admin sign-off step).
     Migration is non-destructive — Canvas stays intact until admin clicks "cut over."

---

### 13.7 `user-auth-service` (Go)

**New:**

- **SSO providers**: SAML 2.0 (Shibboleth), OIDC (Okta, Azure AD), Google Workspace.
  Provider configured per tenant via `TENANT_SSO_CONFIG` env at provisioning.
- **MFA reset**: admin endpoint to reset MFA for a user (audit logged).
- **Audit log table**: append-only `audit_events` table in `tenant_{slug}` schema.
  Every auth event (login, logout, MFA change, impersonation) appended.
- **SSO step in onboarding**: expose a `TestSSOConnection` gRPC endpoint used during
  onboarding workflow.

---

### 13.8 `email-service` (Go)

**New:**

- **Broadcast messages**: `POST /api/broadcast` (platform admin) sends an in-app
  banner + optional email to all users in selected tenants.
- **Tenant admin notifications**: when an incident is opened involving their tenant,
  the tenant primary admin is notified via email.
- **Discussion mention notifications**: Kafka consumer `discussion.mention` →
  sends email notification.

---

## 14. Frontend Route Map

### Student frontend (`app.{domain}` or `{slug}.{domain}`)

| Route               | Description                               | Status   |
| ------------------- | ----------------------------------------- | -------- |
| `/today`            | Action-first daily view (hero + agenda)   | improved |
| `/courses`          | Course list                               | improved |
| `/courses/:id`      | Course detail with next-up module         | improved |
| `/assignments`      | Work queue (open, done, late)             | improved |
| `/assignments/:id`  | Submit + draft saving                     | improved |
| `/grades`           | Grade overview with GPA projection        | improved |
| `/grades/:courseId` | Grade breakdown + "path to A"             | improved |
| `/modules/:id`      | Video + reading + quiz with resume        | improved |
| `/inbox`            | Mentions, professor feedback, study group | **new**  |
| `/people`           | Classmates, professors, groups            | **new**  |
| `/people/:id`       | Person detail + shared courses            | **new**  |
| `/office-hours`     | Book slots with professors                | **new**  |
| `/plan`             | AI-generated 16-week study plan           | **new**  |

### Instructor frontend (`teach.{domain}`)

| Route                                | Description                                | Status   |
| ------------------------------------ | ------------------------------------------ | -------- |
| `/teach`                             | Teaching Today (queue + schedule)          | improved |
| `/courses/:id`                       | Course manage (modules, roster, gradebook) | improved |
| `/courses/:id/edit`                  | Syllabus editor                            | improved |
| `/assignments/new`                   | Create assignment with rubric + auto-tests | improved |
| `/gradebook`                         | Full gradebook with filter                 | improved |
| `/grade/:assignmentId`               | Grading queue with pattern groups          | **new**  |
| `/grade/:assignmentId/:submissionId` | Grade individual submission                | **new**  |
| `/grade/batch`                       | Batch grade a pattern                      | **new**  |
| `/roster`                            | Roster health sorted by risk               | **new**  |
| `/roster/:studentId`                 | Student detail + intervention suggestions  | **new**  |
| `/lecture/live`                      | Live lecture with pulse + Q&A              | **new**  |
| `/office-hours`                      | Host OH view (booked + walk-in)            | **new**  |
| `/discussion`                        | Threads needing reply                      | **new**  |
| `/analytics/:courseId`               | Course analytics                           | improved |

### Admin frontend (`admin.{domain}`)

| Route             | Description                                  | Status   |
| ----------------- | -------------------------------------------- | -------- |
| `/ops`            | Ops HQ: schools, MAU, uptime, open incidents | improved |
| `/schools`        | Tenant list with tier, health, renewal       | improved |
| `/schools/:id`    | Tenant detail + active incidents             | improved |
| `/users`          | Cross-tenant user search                     | improved |
| `/users/:id`      | User detail + impersonate                    | **new**  |
| `/incidents`      | Incident triage P0–P4                        | **new**  |
| `/incidents/:id`  | Incident room + timeline                     | **new**  |
| `/status`         | Service status page                          | **new**  |
| `/onboard/school` | Onboard school wizard (4 steps)              | **new**  |
| `/data/import`    | Canvas/LMS migration                         | **new**  |
| `/flags`          | Feature flags with rollout %                 | **new**  |
| `/broadcast`      | Broadcast message to tenants                 | **new**  |
| `/audit`          | Audit log (actor + action + target)          | improved |
| `/roles`          | Platform roles & permissions                 | improved |
| `/billing`        | Network-level billing: MRR, ARR, churn       | improved |
| `/settings`       | Platform settings                            | existing |

---

## 15. Open Questions & Future Decisions

### 15.1 Tenant-level API gateway

Currently a single platform api-gateway connects to each tenant's network.
Alternative: one api-gateway per tenant (6th provisioned container).
**Decision pending.** Single gateway is simpler for now.

### 15.2 Shared vs per-tenant databases

Current: shared PostgreSQL with `search_path = tenant_{slug}` per schema.
Alternative: separate database per tenant.
**Decision pending.** Schema-per-tenant is the current plan.

### 15.3 Domain acquisition

When a real domain is acquired:

1. Set up wildcard DNS: `*.slate.io → load balancer IP`.
2. Update `DOMAIN_SUFFIX=slate.io` env var in tenant-service.
3. Obtain wildcard TLS cert via Let's Encrypt or GCP Certificate Manager.
4. No code changes required.

### 15.4 Mobile app

The design includes a mobile tab bar component. Mobile app is not in scope for the
current build but the API is designed to support it.

### 15.5 Seed script final form

Once provisioning flow works end-to-end, `scripts/seed-dev.sh` should:

1. Start platform services via docker compose.
2. Run `./scripts/setup-local-dns.sh`.
3. Call `POST admin.slate.local/api/tenants` to provision Eastfield University.
4. Poll until `status: active`.
5. Seed users, courses, assignments, enrollments via `eastfield.slate.local`.

### 15.6 Discussion service vs inbox in email-service

Inbox/mentions could live in email-service or in a separate discussion-service.
**Decision**: separate discussion-service because the feature set is rich enough
(threads, upvotes, mentions, per-course filtering) and email-service should stay
focused on outbound notifications only.

### 15.7 AI service: per-tenant vs platform

AI service is provisioned per tenant (tenant container) to allow per-tenant token
budgets and data isolation. The Claude API key is shared (injected via env at
provisioning time) but usage is metered per tenant.

### 15.8 Zoom integration vs native video

The design shows both native live lecture (via video-conferencing-service) and Zoom
integration. Short-term: Zoom OAuth integration (open Zoom in browser, track events
via webhook). Long-term: native WebRTC via video-conferencing-service.
These are not mutually exclusive — instructor chooses per session.
