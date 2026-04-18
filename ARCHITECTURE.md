# Slate LMS — Architecture

## 1. Overview

Slate is a **multi-tenant Learning Management System** built on a microservices architecture. Each university (tenant) gets an isolated environment: dedicated Docker containers, a separate PostgreSQL schema, and its own Traefik subdomain (`{slug}.slate.local`). Three Next.js 14 portals serve different roles — students, instructors (providers), and platform administrators.

---

## 2. System Topology

```
Internet / Dev Browser
        │
        ▼
┌─────────────────────────────────────────────────────────────────────┐
│  Traefik Reverse Proxy (port 80/443)                                │
│  Subdomain routing: {slug}.slate.local → tenant containers          │
│  Path-based routing: api.slate.local   → api-gateway                │
└──────────────────────────────┬──────────────────────────────────────┘
                               │
          ┌────────────────────┼────────────────────────┐
          ▼                    ▼                         ▼
   Student Portal       Provider Portal          Admin Portal
   (Next.js 14)         (Next.js 14)             (Next.js 14)
   port 3001            port 3002                port 3003
          │                    │                         │
          └────────────────────┴─────────────────────────┘
                               │  (all REST via /api/v1/*)
                               ▼
┌─────────────────────────────────────────────────────────────────────┐
│  API Gateway  (Rust / Axum)                                         │
│  port 8080                                                          │
│  ├─ JWT auth middleware (validates against user-auth-service)       │
│  ├─ Rate limiting (Redis-backed)                                    │
│  ├─ Circuit breaker per upstream                                    │
│  └─ Dynamic route config (refreshable at runtime)                  │
└──────────────────────────────┬──────────────────────────────────────┘
                               │  gRPC + HTTP proxy
     ┌─────────────────────────┼─────────────────────────────────────┐
     │                         │                                      │
     ▼                         ▼                                      ▼
user-auth-service        course-service                  assignment-grading-service
(Go / gRPC :50051)      (NestJS / HTTP :3000)            (Go / HTTP+gRPC :8082/:50053)
     │                         │                                      │
     ▼                         ▼                                      ▼
tenant-service           content-management-service        metrics-service
(Go / gRPC :50052)      (Rust / Axum :8083)               (Go / HTTP+gRPC :8085/:50056)
     │                         │                                      │
     ▼                         ▼                                      ▼
onboarding-service       video-conferencing-service         email-service
(Rust / Restate :9090)  (Rust / Axum :8084)               (Go / HTTP+gRPC :8086/:50057)
     │
     ▼
 (all talk to shared infrastructure below)

┌──────────────────────────────────────────────────────────────────────┐
│  Shared Infrastructure                                               │
│  ├─ PostgreSQL 16   (schema-per-tenant, port 5432)                  │
│  ├─ MongoDB         (course-service, port 27017)                    │
│  ├─ Redis           (cache + rate limiting, port 6379)              │
│  ├─ Kafka + Zookeeper  (async events, port 9092)                    │
│  ├─ MinIO           (object storage, port 9000/9001)                │
│  └─ Restate Server  (durable workflows, port 8080/9070/9071)        │
└──────────────────────────────────────────────────────────────────────┘

┌──────────────────────────────────────────────────────────────────────┐
│  Observability Stack                                                 │
│  ├─ Prometheus    (metrics scraping, port 9090)                     │
│  ├─ Loki          (log aggregation, port 3100)                      │
│  ├─ Tempo         (distributed tracing, port 4317)                  │
│  └─ Grafana       (dashboards, port 3000)                           │
└──────────────────────────────────────────────────────────────────────┘
```

### Two-Network Architecture

| Network | Purpose |
|---|---|
| `slate-network` | Shared backbone — all services, databases, observability |
| `tenant-dev-network` | Per-tenant containers provisioned by `tenant-service` |

---

## 3. Frontend Portals

| Portal | Port | Path prefix | Primary Users |
|---|---|---|---|
| Student | 3001 | `/` | Enrolled students |
| Provider (Instructor) | 3002 | `/` | Course instructors |
| Admin | 3003 | `/` | Platform administrators |

**Design system (shared):** Forest-green color palette, Instrument Serif display font, Inter UI font, JetBrains Mono for code. Action-first layout: every page leads with "what should I do next?"

**Key student routes:** `/today` (landing), `/courses`, `/courses/[id]`, `/grades`, `/plan` (study plan), `/office-hours`, `/discuss/[id]`, `/inbox`, `/profile`, `/settings`

**Key instructor routes:** `/teach` (dashboard), `/grade/[assignmentId]` (grading queue), `/roster/[courseId]` (health), `/office-hours` (schedule), `/courses`, `/inbox`

**Key admin routes:** `/` (ops HQ), `/schools`, `/schools/[id]`, `/incidents`, `/flags`, `/broadcast`, `/services`

---

## 4. Backend Services

| Service | Lang | HTTP Port | gRPC Port | Responsibility | Key Endpoints |
|---|---|---|---|---|---|
| api-gateway | Rust/Axum | 8080 | — | Auth, routing, rate-limit, circuit-break | `/*` (proxy) |
| user-auth-service | Go | 8081 | 50051 | Auth (JWT), users, RBAC, MFA, OAuth | `/api/auth/*`, `/api/users/*` |
| tenant-service | Go | 8082 | 50052 | Tenant CRUD, Docker provisioning, Traefik config | `/tenants/*`, `/admin/schools/*`, `/admin/incidents/*`, `/admin/flags/*` |
| course-service | NestJS | 3000 | — | Courses, modules, lessons, enrollment, progress, office-hours, study plan | `/courses/*`, `/office-hours/*`, `/study-plan/*` |
| assignment-grading-service | Go | 8083 | 50053 | Assignments, submissions, grading, gradebook, grading queue | `/assignments/*`, `/submissions/*`, `/grading/*` |
| content-management-service | Rust/Axum | 8084 | — | File uploads, MinIO signed URLs, visibility | `/content/*` |
| video-conferencing-service | Rust/Axum | 8085 | — | Room management, invitations, sessions | `/rooms/*` |
| email-service | Go | 8086 | 50057 | In-platform messaging, discussion threads, admin broadcast | `/messages/*`, `/discussions/*`, `/admin/broadcast/*` |
| metrics-service | Go | 8087 | 50056 | Analytics, grade distributions, roster health | `/metrics/*`, `/roster/*` |
| onboarding-service | Rust/Restate | 9090 | — | Durable tenant onboarding workflow, 72h approval gate | `/onboarding/*` |

---

## 5. Data Layer

### PostgreSQL (schema-per-tenant)
- Each tenant gets a dedicated schema (e.g., `tenant_acme`, `tenant_stanford`) in the shared PostgreSQL instance.
- The `public` schema holds platform-level data (tenants table, etc.).
- Migrations run per-schema on tenant provisioning.

### MongoDB
- Used exclusively by `course-service` for course, module, lesson, enrollment, and progress documents.

### Kafka Event Bus
Key topics:

| Topic | Producer | Consumers |
|---|---|---|
| `tenant.provisioned` | tenant-service | metrics-service, email-service |
| `tenant.disabled` | tenant-service | video-conferencing-service |
| `course.published` | course-service | metrics-service |
| `assignment.submitted` | assignment-grading-service | metrics-service, email-service |
| `grade.created` | assignment-grading-service | metrics-service |
| `message.sent` | email-service | — |
| `enrollment.created` | course-service | metrics-service |

### Redis
- JWT token blacklist (logout)
- Rate limiting counters (API gateway + tenant-service)
- Session cache

### MinIO
- All user-uploaded content (assignment attachments, course materials, profile pictures)
- `content-management-service` generates presigned S3-compatible URLs

---

## 6. Observability

| Tool | Purpose | Port |
|---|---|---|
| Prometheus | Metrics scraping (all services expose `/metrics`) | 9090 |
| Loki | Structured log aggregation (JSON logs via zerolog/pino) | 3100 |
| Tempo | Distributed tracing (OpenTelemetry, OTLP) | 4317 |
| Grafana | Dashboards connecting all three data sources | 3000 |

All services use **OpenTelemetry** with OTLP export to Tempo. Trace context propagates via `traceparent` / `grpc-trace-bin` headers across service boundaries.

---

## 7. Auth Flow

```
1. Client calls POST /api/v1/auth/register  →  user-auth-service creates user + role
2. Client calls POST /api/v1/auth/login     →  returns { accessToken, refreshToken }
3. All subsequent requests include:         →  Authorization: Bearer <accessToken>
4. api-gateway auth middleware:             →  validates JWT locally (HMAC-SHA256)
                                               attaches X-User-ID, X-Tenant-ID, X-Role headers
5. Upstream services trust headers          →  no re-validation needed
6. Roles:  student | instructor | admin     →  each frontend reads role from JWT claims
                                               and routes to the appropriate portal
```

**Token lifecycle:** Access token = 15 min. Refresh token = 7 days. Refresh via `POST /api/v1/auth/refresh`.

---

## 8. Tenant Isolation

```
Admin creates tenant  →  tenant-service.CreateTenant()
        │
        ├── Writes PostgreSQL schema: CREATE SCHEMA tenant_{slug}
        ├── Writes Traefik config:    /config/traefik/dynamic/{tenantId}.yaml
        │                              routes {slug}.slate.local → tenant containers
        ├── Provisions Docker containers (async via goroutine):
        │   ├── tenant-specific Next.js frontend instances
        │   └── (optional) tenant-isolated DB replica
        └── Publishes Kafka event:    tenant.provisioned
```

Each tenant's containers join `tenant-dev-network` and are reachable at `{slug}.slate.local` via Traefik's dynamic configuration. The shared `slate-network` carries cross-service gRPC/HTTP traffic.

---

## 9. Onboarding Flow (Restate Durable Workflows)

The `onboarding-service` (Rust + Restate SDK) implements a 4-step saga:

```
Step 1: ReceiveApplication    — validate university data, persist draft
        │
        ▼
Step 2: AdminReview (72h gate) — durable timer waits for human approval
        │  (auto-reject if no action in 72h)
        ▼
Step 3: ProvisionTenant       — call tenant-service to create + provision
        │  (compensate: deprovision on failure)
        ▼
Step 4: SendWelcome           — email credentials + subdomain to admin
```

Restate provides exactly-once execution, automatic retries, and durable state so the 72h approval gate survives server restarts.

---

## 10. New Features (v1 Design)

These endpoints were added as part of the v1 design overhaul:

### Student Portal
| Route | Description | Backend |
|---|---|---|
| `/today` | Action-first daily dashboard | course-service, assignment-grading-service, metrics-service |
| `/plan` | AI-generated 16-week study plan | `GET/PUT /study-plan`, `POST /study-plan/regenerate` (course-service) |
| `/office-hours` | Book 15-min slots with professors | `GET/POST /office-hours`, `POST /office-hours/book` (course-service) |
| `/discuss/[id]` | Course Q&A discussion threads | `GET/POST /discussions/*` (email-service) |
| `/grades` | Grade comparison with peer percentile | metrics-service |

### Instructor Portal
| Route | Description | Backend |
|---|---|---|
| `/teach` | Grading-first instructor dashboard | assignment-grading-service, metrics-service |
| `/grade/[assignmentId]` | Queue-style grading with pattern grouping | `GET /grading/queue/*`, `POST /grading/submit/*`, `POST /grading/batch` (assignment-grading-service) |
| `/roster/[courseId]` | Student health / risk-sorted roster | `GET /roster/:courseId/health`, `GET /roster/:courseId/students/:id`, `POST /roster/nudge` (metrics-service) |
| `/office-hours` | Instructor OH slot creation | `GET /office-hours/hosted`, `POST /office-hours/slots`, `DELETE /office-hours/slots/:id` (course-service) |

### Admin Portal
| Route | Description | Backend |
|---|---|---|
| `/schools` | Tenant relationship pages with health + renewal | `GET /admin/schools`, `GET /admin/schools/:id` (tenant-service) |
| `/incidents` | Incident triage with timeline | `GET/POST /admin/incidents/*` (tenant-service) |
| `/flags` | Feature flag management | `GET /admin/flags`, `PATCH /admin/flags/:id` (tenant-service) |
| `/broadcast` | Platform-wide messaging | `POST /admin/broadcast`, `GET /admin/broadcast/history` (email-service) |
