# LMS Platform — Full Rewrite Execution Plan

## Execution Strategy

This document is the master index for the full platform rewrite. Each service has its own plan file in this directory. Work proceeds in phases; agents work in parallel within each phase.

---

## Phases

### Phase 1 — Infrastructure Foundation (Week 1)

- [ ] Traefik reverse proxy + subdomain routing (`plan/infra-traefik.md`)
- [ ] Per-tenant Docker provisioner in tenant-service (`plan/tenant-service.md`)
- [ ] Restate server setup + docker-compose integration (`plan/onboarding-service.md`)
- [ ] Shared proto schema updates (Kafka event schemas, new service protos)

### Phase 2 — Backend Services (Weeks 2–3)

Run all service plans in parallel. Each agent owns one service.

- [ ] `plan/course-service.md`
- [ ] `plan/assignment-grading-service.md`
- [ ] `plan/video-conferencing-service.md`
- [ ] `plan/content-management-service.md`
- [ ] `plan/onboarding-service.md` (Restate)
- [ ] `plan/tenant-service.md` (Docker provisioner)
- [ ] `plan/email-service.md`
- [ ] `plan/metrics-service.md`

### Phase 3 — Frontend Applications (Week 4)

Run all three in parallel after backend APIs are stable.

- [ ] `plan/admin-frontend.md`
- [ ] `plan/provider-frontend.md`
- [ ] `plan/student-frontend.md`

### Phase 4 — Integration & Testing (Week 5)

- [ ] End-to-end Playwright tests for all 3 frontends
- [ ] Kafka integration tests (event flow verification)
- [ ] Tenant provisioning E2E test (onboarding → containers → subdomain)
- [ ] Load testing baseline

---

## Agent Team Assignments

| Agent Name          | Domain                | Plan File                            |
| ------------------- | --------------------- | ------------------------------------ |
| `admin-expert`      | Admin Frontend        | `plan/admin-frontend.md`             |
| `provider-expert`   | Provider Frontend     | `plan/provider-frontend.md`          |
| `student-expert`    | Student Frontend      | `plan/student-frontend.md`           |
| `course-expert`     | Course Service        | `plan/course-service.md`             |
| `grading-expert`    | Assignment Grading    | `plan/assignment-grading-service.md` |
| `video-expert`      | Video Conferencing    | `plan/video-conferencing-service.md` |
| `content-expert`    | Content Management    | `plan/content-management-service.md` |
| `onboarding-expert` | Onboarding + Restate  | `plan/onboarding-service.md`         |
| `tenant-expert`     | Tenant + Docker       | `plan/tenant-service.md`             |
| `email-expert`      | Email Service         | `plan/email-service.md`              |
| `metrics-expert`    | Metrics Service       | `plan/metrics-service.md`            |
| `infra-expert`      | Traefik + DNS + Infra | `plan/infra-traefik.md`              |

---

## Key Architectural Decisions

1. **Restate.dev** replaces the naive onboarding REST CRUD — durable workflow orchestration in Rust
2. **Traefik** handles subdomain routing for per-tenant isolation (`{tenant}.slate.local`)
3. **Docker API** (via dockerode in tenant-service) provisions/deprovisions containers per tenant
4. **Schema-per-tenant** in PostgreSQL via `search_path = tenant_{tenantId}`
5. **Shared infra** (Kafka, Redis, monitoring) — not duplicated per tenant
6. **API Gateway** routes tenant-scoped requests based on JWT claims + Host header

---

## Repository Layout

```
plan/               ← This directory — execution plans
services/           ← Backend microservices
  onboarding-service/   ← Restate workflow (Rust)
  tenant-service/       ← Docker provisioner (Go)
  course-service/       ← NestJS
  assignment-grading-service/ ← Go
  content-management-service/ ← Rust
  video-conferencing-service/ ← Rust
  email-service/        ← Go
  metrics-service/      ← Go
frontend/
  admin/            ← Next.js 14, port 3003
  provider/         ← Next.js 14, port 3002
  student/          ← Next.js 14, port 3000
  shared/           ← Shared types, API client, hooks
config/
  traefik/          ← Traefik static + dynamic configs
proto/              ← All protobuf definitions
```
