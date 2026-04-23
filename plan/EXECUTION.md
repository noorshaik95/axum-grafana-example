# Slate Revamp — Execution Coordination

> Coordination layer over `docs/plan.md` (W1–W20) and `plan/00-overview.md`.
> Cross-references: `plan/API_INVENTORY.md` (service endpoint state), `plan/DESIGN_GAPS.md` (component gap audit), `plan/PROGRESS.md` (live status).

## Phase Order & Blocking Chain

```
WAVE 0  Foundations (serialised; 1 agent)
  0a  Design tokens wired into shared/tailwind.preset + all 3 globals.css
  0b  AppShell primitives (TopBar, NowBar, TabNav) stubs in shared/components/app-shell/
  0c  Gateway X-Request-ID middleware + W3C traceparent propagation libs
  0d  Seed bootstrap (scripts/seed-dev.sh hitting real APIs, idempotent)

WAVE 1  Shared component library (1 FE agent)
  Build the 18 new shared components (see §3)
  Delete per-portal components/ui/ dirs, repoint imports

WAVE 2  Existing-service API surface (parallel)       ║   WAVE 3  New-service scaffolds (parallel)
  W8  course-service                                  ║     W2  admin-auth-service
  W9  assignment-grading                              ║     W3  feature-flag-service
  W10 content-management                              ║     W4  incident-service
  W11 video-conferencing                              ║     W5  discussion-service
  W12 metrics                                         ║     W6  scheduling-service
  W14 user-auth                                       ║     W7  ai-service
  W15 email                                           ║     W13 onboarding (SSO + Canvas)
  W17 api-gateway                                     ║     W16 tenant-service (provision 8)

WAVE 4  Portal rewire (3 FE agents)
  W18 admin-fe   — 16 pages, delete sidebar, no mock-data
  W19 provider-fe — 14 pages, delete sidebar
  W20 student-fe  — 13 pages, delete sidebar, add VideoPlayer w/ resume

WAVE 5  Observability + seed + test lockdown
  Trace assertion harness (Playwright click → capture X-Request-ID → ≥3 Tempo spans)
  Eastfield full seed (23 users, 3 courses, OH, discussions, submissions)
  Playwright per portal + shared unit coverage ≥70%
  Grafana dashboards per portal, redirect middleware for legacy routes
```

### Hard blockers

- Wave 0c must land before any Wave 2/3 endpoint — every new endpoint must emit `X-Request-ID` from day one.
- Wave 1 must land before Wave 4 — no portal work may duplicate UI code.
- W16 tenant provisioner (Wave 3) must provision all 8 tenant services before Wave 5 seed; today only course/assignment/content are provisioned.

---

## Shared Component Inventory

Location root: `frontend/shared/components/`.

**Existing (do not duplicate):** button, card, progress, input, badge, avatar, spinner, empty-state, page-header, stat-card, sidebar-nav (deprecate after Wave 4), data-table, widget.

**New (Wave 1):**

| Component               | Path                          | Consumers                            | Purpose                                                           |
| ----------------------- | ----------------------------- | ------------------------------------ | ----------------------------------------------------------------- |
| `TopBar`                | `app-shell/top-bar.tsx`       | all                                  | Sticky blurred cream wordmark + ≤5 nav links + ⌘K + bell + avatar |
| `NowBar`                | `app-shell/now-bar.tsx`       | all                                  | forest-700 strip; `chips: NowBarChip[]`; auto-hides when empty    |
| `TabNav`                | `app-shell/tab-nav.tsx`       | all                                  | Horizontal tabs replacing sidebar; forest-500 active underline    |
| `AppShell`              | `app-shell/app-shell.tsx`     | all                                  | Composes TopBar + NowBar + TabNav + `<main>`                      |
| `CommandPalette`        | `command-palette/index.tsx`   | all                                  | ⌘K modal wrapping `/api/ai/cmd-palette`                           |
| `StatusPill`            | `status-pill.tsx`             | admin `/status`, `/ops`              | green/amber/red dot + label                                       |
| `PriorityBadge`         | `priority-badge.tsx`          | admin `/incidents`                   | P0–P4                                                             |
| `RiskBadge`             | `risk-badge.tsx`              | provider `/roster`                   | healthy/slipping/at_risk                                          |
| `ActionHero`            | `action-hero.tsx`             | student `/today`, provider `/teach`  | Serif heading + primary CTA                                       |
| `ListRow`               | `list-row.tsx`                | inbox, grading queue, roster         | Alternating warm-50/cream                                         |
| `AgendaItem`            | `agenda-item.tsx`             | student `/today`                     | icon + title + urgency pill + time                                |
| `RubricEditor`          | `rubric-editor.tsx`           | provider grading + assignment create | Controlled rubric rows                                            |
| `VideoPlayer`           | `video-player.tsx`            | student `/modules`, `/video`         | HLS + range + resume position                                     |
| `TraceWidget`           | `trace-widget.tsx`            | all (dev)                            | Shows current X-Request-ID + Tempo link                           |
| `ImpersonationBanner`   | `impersonation-banner.tsx`    | admin, tenant                        | Move from admin/components                                        |
| `FeatureFlagGate`       | `feature-flag-gate.tsx`       | all                                  | Gate children on flag lookup                                      |
| `NotificationBell`      | `notification-bell.tsx`       | all                                  | Polls inbox; badge count                                          |
| `EmptyStateIllustrated` | `empty-state-illustrated.tsx` | all                                  | Forest-palette SVG illustrations                                  |

---

## API Integration Matrix

Legend: ✓ real · ∆ stub/partial · ✗ missing (Wave 2/3 builds it).

### Student (13 pages)

| Page                      | Endpoints                                                                                  | Service                | State                              |
| ------------------------- | ------------------------------------------------------------------------------------------ | ---------------------- | ---------------------------------- |
| `/today`                  | `GET /api/ai/welcome`, `GET /api/assignments?status=open`, `GET /api/courses/next-lecture` | ai, assignment, course | ✗ ai, ∆ assignment, ✗ next-lecture |
| `/courses`                | `GET /api/courses/enrolled`                                                                | course                 | ✓                                  |
| `/courses/[id]`           | `GET /api/courses/:id/modules`, `.../next-up/:uid`                                         | course                 | ∆ (W8)                             |
| `/assignments`            | `GET /api/assignments?status=`                                                             | assignment             | ✓                                  |
| `/assignments/[id]`       | `POST /api/assignments/:id/submit`, `PATCH .../draft`                                      | assignment             | ∆ (W9)                             |
| `/grades`                 | `GET /api/grades`                                                                          | assignment             | ✓                                  |
| `/grades/[courseId]`      | `GET /api/ai/grade-projection?course=`                                                     | ai                     | ✗                                  |
| `/plan`                   | `POST /api/ai/study-plan`                                                                  | ai                     | ✗                                  |
| `/office-hours`           | `GET /api/scheduling/slots`, `POST .../bookings`                                           | scheduling             | ✗                                  |
| `/inbox`                  | `GET /api/discussions/inbox`                                                               | discussion             | ✗                                  |
| `/people`, `/people/[id]` | `GET /api/users?course=`                                                                   | user-auth              | ∆                                  |
| `/modules/[id]`           | `GET /api/content/:id`, `PUT .../position`                                                 | content                | ∆ (W10)                            |
| `/discussions`            | `GET /api/discussions/threads`, `POST .../posts`                                           | discussion             | ✗                                  |

### Provider (14 pages)

| Page                        | Endpoints                                                        | Service             | State     |
| --------------------------- | ---------------------------------------------------------------- | ------------------- | --------- |
| `/teach`                    | `GET /api/grading/queue/count`, `GET /api/metrics/roster-health` | assignment, metrics | ∆ / ✗     |
| `/courses`, `/courses/[id]` | `GET /api/courses/taught`                                        | course              | ✓         |
| `/grade/[assignmentId]`     | `GET /api/grading/queue/:id` (grouped)                           | assignment          | ✗ (W9.4)  |
| `/grade/[id]/[subId]`       | `GET /api/grading/submissions/:id`                               | assignment          | ∆         |
| `/grade/batch`              | `POST /api/grades/batch`                                         | assignment          | ✗ (W9.5)  |
| `/roster`, `/roster/[id]`   | `GET /api/metrics/roster-health?course=`                         | metrics             | ✗ (W12.1) |
| `/lecture/live`             | `POST /api/video/lectures/start`, `.../pulse`, `.../qa`          | video               | ✗ (W11)   |
| `/office-hours`             | `POST /api/scheduling/schedules`, `.../instructor-day`           | scheduling          | ✗         |
| `/discussion`               | `GET /api/discussions/needing-reply`                             | discussion          | ✗ (W5.4)  |
| `/analytics`                | `GET /api/courses/:id/analytics`                                 | course              | ✗ (W8.3)  |
| `/students`                 | `GET /api/users?course=&role=student`                            | user-auth           | ∆         |

### Admin (16 pages)

| Page                             | Endpoints                                                       | Service           | State     |
| -------------------------------- | --------------------------------------------------------------- | ----------------- | --------- |
| `/ops`                           | `GET /api/incidents?status=open`, `GET /api/metrics/platform`   | incident, metrics | ✗ / ∆     |
| `/schools`, `/schools/[id]`      | `GET /api/tenants`, `GET .../:id`, `GET /api/incidents?tenant=` | tenant, incident  | ✓ / ∆     |
| `/iam/users`, `/iam/users/[id]`  | `GET /api/admin/users`, `POST /api/admin/impersonate/:id`       | admin-auth        | ✗ (W2)    |
| `/iam/roles`, `/iam/audit`       | `GET /api/admin/roles`, `.../audit`                             | admin-auth        | ✗         |
| `/incidents`, `/incidents/[id]`  | `GET /api/incidents`, `POST /api/incidents`, `POST .../events`  | incident          | ✗ (W4)    |
| `/status`                        | `GET /api/status` (public)                                      | incident          | ✗         |
| `/flags`                         | `GET /api/flags`, `PUT /api/flags/:key`                         | feature-flag      | ✗ (W3)    |
| `/broadcast`                     | `POST /api/broadcast`                                           | email             | ✗ (W15)   |
| `/billing`                       | `GET /api/billing/network`                                      | metrics           | ∆         |
| `/onboarding`, `/onboard/school` | `POST /api/onboarding` (Restate)                                | onboarding        | ∆ (W13)   |
| `/data/import`                   | `POST /api/onboarding/canvas-import`                            | onboarding        | ✗ (W13.2) |
| `/impersonation`                 | `POST /api/admin/impersonate/:tenantId/:userId`                 | admin-auth        | ✗         |

**Rule:** after Wave 4 no page may import from `lib/mock-data.ts`. Those files get deleted.

---

## Observability Plan

### Current gaps

- Frontends don't inject traceparent; `X-Request-ID` not surfaced to users.
- Only gateway ↔ user-auth is wired for OTLP propagation; other services miss outbound-call propagation on gRPC and Kafka.
- No test harness — "eyeball in Grafana."

### Target

1. **Wave 0c** — `frontend/shared/lib/api/client.ts` fetch interceptor generates traceparent per request; parses `X-Request-ID` from response and stores in Zustand `traceStore`. `TraceWidget` in dev shows current ID + clickable Tempo URL.
2. **Gateway** — `services/api-gateway/src/middleware/request_id.rs`: generate if missing, always echo, tag span with `request_id`.
3. **Service libs** — Extend `libs/common-go/tracing` and `libs/common-rust/src/tracing` for (a) extract on incoming gRPC/HTTP, (b) inject on outbound gRPC via `otelgrpc`, (c) inject/extract on Kafka headers, (d) always tag spans with `tenant.slug` and `request_id`.
4. **Kafka** — All 16 topics propagate traceparent headers.
5. **Assertion harness (Wave 5)** — `scripts/assert_trace.sh`: Playwright click captures `X-Request-ID`, waits 5s for Tempo ingest, asserts ≥3 spans across ≥3 services.

Service owners for Wave 2/3 each ship one "traceparent propagation" PR with acceptance: Grafana trace shows their service's span connected to parent.

---

## Test-Data Strategy (Eastfield seed)

`scripts/seed-eastfield.sh` (idempotent, `--reset` flag). Creates via API gateway:

- **Users:** 1 tenant admin, 2 instructors (CS + Math), 20 students `student01..20@eastfield.edu` (password `Slate-test-1!`).
- **Courses:** CS 101 (Martinez, 15 enrolled), CS 201 (Martinez, 12), MATH 150 (Khan, 10).
- **Modules & lessons:** 4 modules × 3 lessons per course (video + reading + quiz); seed 30s test clip in MinIO.
- **Assignments:** 5 per course mixing states (due soon, graded, past-due-ungraded, draft); 2 with auto-tests; all with 4-row rubric.
- **Submissions:** 10 per `past_due_ungraded` producing 3 fingerprints → exercises pattern grouping (W9.4). 1 draft for `student01` in CS 101 PS4.
- **Grades:** seed to produce 5 at_risk, 5 slipping, 10 healthy in roster health (W12.1).
- **Discussions:** 3 threads per course; `@prof.martinez` mentions `student02` once (populates inbox). 1 thread 24h+ untouched → needing-reply.
- **Office hours:** Martinez Tue/Thu 2–4pm 15-min online; Khan Mon/Wed 10–11am 15-min in-person rm 204. Pre-book student03 → Martinez Tue 2:15 with context "Q3 from PS4".
- **Study plan:** pre-generated for student01 via `POST /api/ai/study-plan` so `/plan` has data without cold Claude call.

Target runtime: <90 s.

---

## Risk / Blocker Register

| #   | Risk                                                                                 | Mitigation                                                                               | Owner                          |
| --- | ------------------------------------------------------------------------------------ | ---------------------------------------------------------------------------------------- | ------------------------------ |
| R1  | Three portals each have own `components/ui/` dirs — violates "reuse not duplicate"   | Wave 4 deletes; add eslint `no-restricted-imports`                                       | shared-ui-expert               |
| R2  | Provisioner today boots only 3 of 8 tenant services                                  | W16 is Wave 3 critical path; seed runs after                                             | tenant-expert                  |
| R3  | Proto contract churn (W8 modules, W9 rubric, W12 roster health) breaks FE            | Freeze protos end of Wave 2 week 1                                                       | all service experts            |
| R4  | ai-service Claude cost blow-up from study-plan regeneration                          | Ship W7.5 budget tracker in first PR                                                     | ai-expert                      |
| R5  | Per-tenant DB migrations must run at tenant create, not platform boot                | Each service ships `migrations/` + `RunMigrations(schema)` gRPC called by tenant-service | tenant-expert + service owners |
| R6  | `PLATFORM_PUBLIC_KEY` must mount into user-auth and ai containers                    | Add to `ProvisionRequest`, mount as secret volume                                        | tenant-expert                  |
| R7  | Legacy route changes break bookmarks                                                 | Redirect middleware in each portal (`middleware.ts`)                                     | FE agents                      |
| R8  | `data-table.tsx`, `widget.tsx` in shared/ but not exported from index                | First Wave 1 PR exports them                                                             | shared-ui-expert               |
| R9  | W13 onboarding SSO depends on W14.1                                                  | W14 starts day 1 Wave 2, W13 day 3                                                       | sequencing                     |
| R10 | Tempo ingest lag (~3s) breaks naive assertion                                        | `sleep 5` + retry loop in harness                                                        | observability-expert           |
| R11 | Design canvases in `design/` HTML may not render 1:1 in React (backdrop-blur Safari) | Parity checklist alongside shell PR; deviations logged                                   | foundation-expert              |

---

## Agent Team

Team name: **slate-revamp** (`~/.claude/teams/slate-revamp/`).

| Agent                  | Type                   | Responsibility                                                                                                                                              |
| ---------------------- | ---------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `team-lead`            | Claude-orchestrator    | Owns TaskList, wave gating, blocker escalation                                                                                                              |
| `po-analyst`           | persistent team member | Facilitator/PO: unblocks agents by chasing cross-service questions (e.g. "AI service — what's the /welcome-message response shape?"); maintains PROGRESS.md |
| `foundation-expert`    | 1-shot Agent           | Wave 0 all four items                                                                                                                                       |
| `shared-ui-expert`     | 1-shot Agent           | Wave 1 component build-out                                                                                                                                  |
| Per-workstream experts | 1-shot Agent each      | W2–W20 as decomposed below                                                                                                                                  |
| `qa-expert`            | 1-shot Agent           | Wave 5 seed + Playwright + unit coverage                                                                                                                    |
| `observability-expert` | 1-shot Agent           | Wave 5 trace harness + dashboards                                                                                                                           |

Communication contract:

- Blockers → `SendMessage(to: 'po-analyst')`. PO either answers, or delegates to the relevant expert agent.
- Progress → `TaskUpdate` on owner's task. PO syncs `plan/PROGRESS.md` from TaskList snapshot.
- Task handoff between waves → only when all blocking tasks in previous wave complete.

---

## Critical Files To Touch First

- `frontend/shared/tailwind.preset.ts` — add forest/cream/warm tokens
- `frontend/shared/styles/globals.css` — CSS variables per DESIGN.md §7
- `frontend/{admin,provider,student}/app/globals.css` — drop per-portal overrides
- `frontend/shared/components/app-shell/{top-bar,now-bar,tab-nav,app-shell}.tsx` — new
- `frontend/shared/lib/api/client.ts` — traceparent + X-Request-ID interceptor
- `services/api-gateway/src/middleware/request_id.rs` — new
- `libs/common-go/tracing/tracing.go` — outbound propagation
- `libs/common-rust/src/tracing/` — outbound propagation
- `scripts/seed-dev.sh` — bootstrap API-backed seed
- `docker-compose.yml` — new compose entries for admin-auth, flag, incident
