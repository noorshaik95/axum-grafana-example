# Gateway ↔ Proto Audit (Task #29, 2026-04-18)

Owner: `gateway-proto-audit-expert`

Scope: every `/api/*` route declared in `config/gateway-config.yaml route_overrides`. For each, verify the declared `grpc_method` exists under the right `(service.Name / FullMethodName)` in the upstream service's `.proto` and is actually registered on the gRPC server. Categorize:

- **FIX-YAML** — proto has the method; yaml references the wrong name/package. Fixable inline.
- **NEEDS-SERVICE** — proto doesn't declare the method. Service owner must add it. Dispatch.
- **NEEDS-HTTP-PROXY** — service is REST-only. Blocked on CARRYOVER §1a (HTTP reverse-proxy). Short-term workaround: ask service owner for a thin gRPC wrapper.

---

## 1. metrics-service — CRITICAL (live /ops broken)

`metrics-service` registers ONLY `grpc.health.v1.Health` on its gRPC port (services/metrics-service/cmd/server/main.go:158–168). The actual metrics surface is a chi HTTP router on `:8080`. Every `metrics.MetricsService/*` gRPC route the gateway declares therefore returns **"proto: not found"** at runtime.

REST endpoints that already exist on metrics-service (handlers.go/platform_handler.go/roster_handler.go/export_handler.go):

- `GET /metrics/platform` (handlers.go:41)
- `GET /metrics/tenants/{id}` (handlers.go:42)
- `GET /metrics/students/{id}` (handlers.go:44)
- `GET /metrics/grades/distribution/{courseId}` (handlers.go:46)
- `GET /metrics/grades/comparison/{studentId}/{courseId}` (handlers.go:47)
- `GET /metrics/platform/extended` (platform_handler.go:24)
- `GET /roster/{courseId}/health` (roster_handler.go:28)
- `POST /api/metrics/export`, `GET /api/metrics/export/{jobId}` (export_handler.go:24–25)

| yaml route                                                  | method                                        | proto status                                   | category                            | action                                                                       |
| ----------------------------------------------------------- | --------------------------------------------- | ---------------------------------------------- | ----------------------------------- | ---------------------------------------------------------------------------- |
| `GET /api/metrics/platform`                                 | `metrics.MetricsService/GetPlatformMetrics`   | method DEFINED in proto, server NOT REGISTERED | **NEEDS-SERVICE (wrapper)**         | ask metrics-expert for a gRPC wrapper around `handlers.GetPlatformMetrics`   |
| `GET /api/metrics/tenants/:id`                              | `metrics.MetricsService/GetTenantMetrics`     | defined, unregistered                          | **NEEDS-SERVICE (wrapper)**         | same                                                                         |
| `GET /api/metrics/students/:id/progress`                    | `metrics.MetricsService/GetStudentProgress`   | defined, unregistered                          | **NEEDS-SERVICE (wrapper)**         | same                                                                         |
| `GET /api/metrics/grades/distribution/:course_id`           | `metrics.MetricsService/GetGradeDistribution` | defined, unregistered                          | **NEEDS-SERVICE (wrapper)**         | same                                                                         |
| `GET /api/metrics/grades/comparison/:student_id/:course_id` | `metrics.MetricsService/GetStudentComparison` | defined, unregistered                          | **NEEDS-SERVICE (wrapper)**         | same                                                                         |
| `GET /api/metrics/system`                                   | `metrics.MetricsService/GetSystemMetrics`     | defined, unregistered                          | **NEEDS-SERVICE (wrapper)**         | same                                                                         |
| `GET /api/metrics/alerts`                                   | `metrics.MetricsService/GetActiveAlerts`      | defined, unregistered                          | **NEEDS-SERVICE (wrapper)**         | same                                                                         |
| `GET /api/metrics/roster-health`                            | `metrics.MetricsService/GetRosterHealth`      | **NOT IN PROTO**                               | **NEEDS-SERVICE (proto + wrapper)** | add rpc + wrap `handlers.RosterHandler.GetRosterHealth`                      |
| `GET /api/metrics/grade-dist`                               | `metrics.MetricsService/GetGradeDistribution` | defined, unregistered                          | **NEEDS-SERVICE (wrapper)**         | alias of `/api/metrics/grades/distribution/:course_id` — resolved together   |
| `GET /api/metrics/export`                                   | `metrics.MetricsService/ExportMetrics`        | **NOT IN PROTO**                               | **NEEDS-SERVICE (proto + wrapper)** | add rpc + wrap `handlers.ExportHandler.CreateExport/GetExport`               |
| `GET /api/health/metrics`                                   | `metrics.MetricsService/HealthCheck`          | no such rpc; grpc.health.v1 is what's running  | **FIX-YAML**                        | change to `grpc.health.v1.Health/Check` (same pattern as scheduling-service) |

**Short-term recommendation (PO ratify):** metrics-expert adds a slim gRPC server that wraps the existing REST handlers (share the same `service.*` structs — 100 LOC, 15-20 minutes). Long-term: CARRYOVER §1a HTTP-proxy.

---

## 2. assignment-grading-service

proto: services/assignment-grading-service/api/proto/assignment.proto

| yaml route                                     | method                                            | proto status                    | category          | action                                                                                                                                                                                                                            |
| ---------------------------------------------- | ------------------------------------------------- | ------------------------------- | ----------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `GET /api/grading/queue/:assignment_id`        | `assignment.GradingService/GetGradingQueue`       | **NOT IN PROTO**                | **NEEDS-SERVICE** | dispatch to assignment-grading-expert                                                                                                                                                                                             |
| `POST /api/grades/batch`                       | `assignment.GradingService/BatchPublishGrades`    | **NOT IN PROTO**                | **NEEDS-SERVICE** | dispatch                                                                                                                                                                                                                          |
| `PATCH /api/assignments/:id/submissions/draft` | `assignment.SubmissionService/SaveDraft`          | **NOT IN PROTO**                | **NEEDS-SERVICE** | dispatch                                                                                                                                                                                                                          |
| `GET /api/assignments/:id/submissions/draft`   | `assignment.SubmissionService/GetDraft`           | **NOT IN PROTO**                | **NEEDS-SERVICE** | dispatch                                                                                                                                                                                                                          |
| `GET /api/grading/queue/count`                 | `metrics.MetricsService/GetGradingQueueCount` (!) | **NOT IN PROTO, wrong service** | **NEEDS-SERVICE** | yaml incorrectly points at metrics package; belongs on `assignment.GradingService`. Dispatch to assignment-grading-expert. **Also fix yaml to reference `assignment.GradingService/GetGradingQueueCount` once the method lands.** |
| `GET /api/health/assignments`                  | `assignment.AssignmentService/HealthCheck`        | defined ✓                       | OK                | —                                                                                                                                                                                                                                 |
| All CRUD + publish routes                      | various                                           | defined ✓                       | OK                | —                                                                                                                                                                                                                                 |

User-reported "Route not found" on bare `GET /api/grading/queue`: the yaml has no such route — the two candidates are `/api/grading/queue/:assignment_id` (per-assignment) and `/api/grading/queue/count` (aggregate). Bare `/api/grading/queue` is a FE bug (provider `/teach` should hit `/api/grading/queue/count`; confirmed at frontend/provider/app/(dashboard)/teach/page.tsx:43). Flag to provider-fe-expert.

---

## 3. course-service

proto: proto/course.proto

| yaml route                                                                                    | method                                    | proto status                                                                             | category          | action                    |
| --------------------------------------------------------------------------------------------- | ----------------------------------------- | ---------------------------------------------------------------------------------------- | ----------------- | ------------------------- |
| `GET /api/courses/:id/modules`                                                                | `course.CourseService/ListModules`        | **NOT IN PROTO** (content.proto has ListModules for content modules — different service) | **NEEDS-SERVICE** | dispatch to course-expert |
| `GET /api/courses/:id/next-up/:user_id`                                                       | `course.CourseService/GetNextUp`          | **NOT IN PROTO**                                                                         | **NEEDS-SERVICE** | dispatch                  |
| `GET /api/courses/:id/analytics`                                                              | `course.CourseService/GetCourseAnalytics` | **NOT IN PROTO**                                                                         | **NEEDS-SERVICE** | dispatch                  |
| `GET /api/courses/next-lecture`                                                               | `course.CourseService/GetNextLecture`     | **NOT IN PROTO**                                                                         | **NEEDS-SERVICE** | dispatch                  |
| `GET /api/health/courses`                                                                     | `course.CourseService/HealthCheck`        | defined ✓                                                                                | OK                | —                         |
| All CRUD, publish, enrollment, templates, prerequisites, co-teaching, sections, cross-listing | various                                   | defined ✓                                                                                | OK                | —                         |

---

## 4. ai-service

proto: proto/ai.proto (package `slate.ai.v1`, NOT `ai`)

The gateway yaml uses `ai.AiService/...` everywhere. The generated gRPC registers `slate.ai.v1.AiService/...` (services/ai-service/api/proto/ai.pb.go:1422; services/ai-service/cmd/server/main.go:100 `aipb.RegisterAiServiceServer`). Every ai route fails proto lookup.

| yaml route                     | method                            | proto status                                                                   | category     | action                                  |
| ------------------------------ | --------------------------------- | ------------------------------------------------------------------------------ | ------------ | --------------------------------------- |
| `GET /api/ai/welcome`          | `ai.AiService/GetWelcomeMessage`  | exists as `slate.ai.v1.AiService/GetWelcomeMessage`                            | **FIX-YAML** | rewrite                                 |
| `GET /api/ai/welcome-message`  | `ai.AiService/GetWelcomeMessage`  | same                                                                           | **FIX-YAML** | rewrite                                 |
| `POST /api/ai/cmd-palette`     | `ai.AiService/GetCommandPalette`  | proto has `GetCommandPaletteResults`, package `slate.ai.v1`                    | **FIX-YAML** | rewrite (both package AND method name)  |
| `GET /api/ai/grade-projection` | `ai.AiService/GetGradeProjection` | exists as `slate.ai.v1`                                                        | **FIX-YAML** | rewrite                                 |
| `POST /api/ai/study-plan`      | `ai.AiService/GenerateStudyPlan`  | exists                                                                         | **FIX-YAML** | rewrite                                 |
| `POST /api/ai/draft-feedback`  | `ai.AiService/GetDraftFeedback`   | exists                                                                         | **FIX-YAML** | rewrite                                 |
| `GET /api/health/ai`           | `ai.AiService/HealthCheck`        | NOT in proto; ai-service registers no health RPC beyond default grpc.health.v1 | **FIX-YAML** | change to `grpc.health.v1.Health/Check` |

Also: `GET /api/ai/grade-projection` and `GET /api/ai/welcome` are declared as GET in the yaml — the proto is pure gRPC (no transcoding annotations), so HTTP verb is a pure gateway convention. Leave as-is.

---

## 5. admin-auth-service

proto: proto/admin_auth.proto

| yaml route                                                                                          | method                                    | proto status                                    | category     | action                                  |
| --------------------------------------------------------------------------------------------------- | ----------------------------------------- | ----------------------------------------------- | ------------ | --------------------------------------- |
| all AdminLogin/Logout/Refresh/Validate/ListAdminUsers/ListAdminRoles/GetAuditLog/Impersonate routes | various                                   | defined ✓                                       | OK           | —                                       |
| `GET /api/health/admin-auth`                                                                        | `admin_auth.AdminAuthService/HealthCheck` | **NOT IN PROTO**; only grpc.health.v1 available | **FIX-YAML** | change to `grpc.health.v1.Health/Check` |

---

## 6. onboarding-service — CLOSED §1b (`22b4d70`)

proto: proto/onboarding.proto

**Ruled by po-analyst 2026-04-18:** onboarding is not a service-wrapper candidate. Two incompatible models collide here:

1. `proto/onboarding.proto` is the **legacy** CSV-bulk-import flow (CreateJob / UploadCSV / ProcessBatch / ListTasks / SyncIntegration). Not what admin-fe /onboarding invokes.
2. The **actual W13 onboarding-service** is Restate/Rust Virtual Objects (`OnboardingWorkflow` + `CanvasMigration`) with HTTP endpoints on :9080, not gRPC. Restate's programming model is HTTP-native; a gRPC shim would fight the framework.
3. The yaml referenced a **third** set (`StartOnboarding`, `GetOnboardingStatus`, `CanvasImport`) that existed in neither world — removed in `2ebfa8c`.

**Resolution (§1b, CLOSED `22b4d70` + cleanup `2ebfa8c`):** 14 Restate routes added to gateway-config.yaml using `http_proxy_target: http://restate-server:9080` (Restate ingress) + `http_proxy_path_template` for `:id` substitution (matcher key is `:id`, matching the rest of the yaml's convention; CanvasMigration nested under `/api/onboarding/:id/canvas/*`). Target is the Restate ingress port, NOT `onboarding-service:9080` (SDK endpoint is not publicly routable). Stale `onboarding.OnboardingService/{StartOnboarding,GetOnboardingStatus,CanvasImport,Health}` gRPC passthroughs + the `/api/health/onboarding` public-route entry removed in `2ebfa8c`.

| yaml route                                      | method                                                                 | status   |
| ----------------------------------------------- | ---------------------------------------------------------------------- | -------- |
| `POST /api/onboarding/:id/start`                | HTTP proxy → `http://restate-server:9080/OnboardingWorkflow/:id/start` | **DONE** |
| `GET /api/onboarding/:id/status`                | HTTP proxy → `/OnboardingWorkflow/:id/status`                          | **DONE** |
| `POST /api/onboarding/:id/approve`              | HTTP proxy → `/OnboardingWorkflow/:id/approve`                         | **DONE** |
| `POST /api/onboarding/:id/reject`               | HTTP proxy → `/OnboardingWorkflow/:id/reject`                          | **DONE** |
| _(+ 10 more CanvasMigration + workflow routes)_ | see gateway-config.yaml §Onboarding                                    | **DONE** |

---

## 7. incident-service

proto: proto/incident.proto

| yaml route                                                                                           | method                                 | proto status     | category     | action                                  |
| ---------------------------------------------------------------------------------------------------- | -------------------------------------- | ---------------- | ------------ | --------------------------------------- |
| all ListIncidents/CreateIncident/GetIncident/UpdateIncident/PostIncidentEvent/GetPublicStatus routes | various                                | defined ✓        | OK           | —                                       |
| `GET /api/health/incidents`                                                                          | `incident.IncidentService/HealthCheck` | **NOT IN PROTO** | **FIX-YAML** | change to `grpc.health.v1.Health/Check` |

---

## 8. feature-flag-service

proto: proto/feature_flag.proto

| yaml route               | method                         | proto status                                                         | category          | action                                               |
| ------------------------ | ------------------------------ | -------------------------------------------------------------------- | ----------------- | ---------------------------------------------------- |
| `GET /api/flags`         | `flag.FlagService/ListFlags`   | defined ✓                                                            | OK                | —                                                    |
| `GET /api/flags/:key`    | `flag.FlagService/GetFlag`     | **NOT IN PROTO** (proto has `EvaluateFlags` but not single-flag Get) | **NEEDS-SERVICE** | dispatch to flag-expert to add `GetFlag(key) → Flag` |
| `PUT /api/flags/:key`    | `flag.FlagService/UpdateFlag`  | defined ✓                                                            | OK                | —                                                    |
| `DELETE /api/flags/:key` | `flag.FlagService/DeleteFlag`  | defined ✓                                                            | OK                | —                                                    |
| `GET /api/health/flags`  | `flag.FlagService/HealthCheck` | NOT in proto                                                         | **FIX-YAML**      | `grpc.health.v1.Health/Check`                        |

---

## 9. video-conferencing-service

proto: proto/video_conferencing.proto

All session/lecture/QA/office-hours routes map to declared RPCs. No mismatches.

| yaml route              | method                                                    | proto status | category     | action                        |
| ----------------------- | --------------------------------------------------------- | ------------ | ------------ | ----------------------------- |
| `GET /api/health/video` | `video_conferencing.VideoConferencingService/HealthCheck` | NOT in proto | **FIX-YAML** | `grpc.health.v1.Health/Check` |

---

## 10. email-service

proto: proto/email.proto

| yaml route                                                                      | method                               | proto status | category     | action                        |
| ------------------------------------------------------------------------------- | ------------------------------------ | ------------ | ------------ | ----------------------------- |
| all SendMessage/GetInbox/MarkRead/GetThread/GetUnreadCount/SendBroadcast routes | various                              | defined ✓    | OK           | —                             |
| `GET /api/health/messages`                                                      | `email.MessagingService/HealthCheck` | NOT in proto | **FIX-YAML** | `grpc.health.v1.Health/Check` |

---

## 11. tenant-service

proto: proto/tenant.proto

| yaml route                                                                                                               | method                             | proto status | category     | action                        |
| ------------------------------------------------------------------------------------------------------------------------ | ---------------------------------- | ------------ | ------------ | ----------------------------- |
| all CreateTenant/GetTenant/UpdateTenant/DeleteTenant/ListTenants/GetProvisioningStatus/GetStorageQuota/CreateTenantAdmin | various                            | defined ✓    | OK           | —                             |
| `GET /api/health/tenants`                                                                                                | `tenant.TenantService/HealthCheck` | NOT in proto | **FIX-YAML** | `grpc.health.v1.Health/Check` |

---

## 12. user-auth-service

proto: proto/user.proto + proto/auth.proto

All routes match. `auth.AuthService/HealthCheck` (used for `/api/health/auth`) is defined in proto/auth.proto ✓.

---

## 13. content-management-service

proto: proto/content.proto

All routes (ContentService/UploadService/StreamingService/ProgressService/SearchService/DownloadService) match declared RPCs. `content.ContentService/HealthCheck` is declared ✓.

---

## 14. discussion-service

proto: proto/discussion.proto

All routes match. `discussion.DiscussionService/HealthCheck` is declared ✓.

---

## 15. scheduling-service

proto: services/scheduling-service/api/proto/scheduling.proto

All routes match. yaml correctly uses `grpc.health.v1.Health/Check` for `/api/health/scheduling` already (gateway-cleanup #28 landed this pattern).

---

## Summary — counts (post-ratification 2026-04-18)

- **FIX-YAML** (applied inline): 16 routes
  - 7 ai-service (pkg `ai.AiService` → `slate.ai.v1.AiService`; `GetCommandPalette` → `GetCommandPaletteResults`; ai health → `grpc.health.v1.Health/Check`)
  - 8 `*.HealthCheck` → `grpc.health.v1.Health/Check` (admin-auth, incidents, flags, video, messages, metrics, tenants) + ai
  - 1 onboarding `HealthCheck` → `Health`
- **NEEDS-SERVICE** (ratified, dispatched): 4 service experts
  - P1 metrics-expert — gRPC wrapper around existing REST + new `GetRosterHealth` / `ExportMetrics` RPCs. **Isomorphic response shapes required** (REST-first per CONTRACTS.md).
  - P2 assignment-grading-expert — `GetGradingQueue`, `BatchPublishGrades`, `GetGradingQueueCount` on `assignment.GradingService`; `SaveDraft`, `GetDraft` on `assignment.SubmissionService`. `GetGradingQueueCount` stays on grading, NOT metrics (yaml update once RPC lands).
  - P3 course-expert — `ListModules`, `GetNextUp`, `GetCourseAnalytics`, `GetNextLecture` on `course.CourseService` (course-module collection, distinct from content CMS).
  - P4 flag-expert — `GetFlag(key)` RPC (6th cleanly-additive member of the set).
- **BLOCKED-ON-§1a**: onboarding (3 routes). Service is Restate/HTTP on :9080 — wrong model for gRPC wrapper. Defer entirely to gateway-proxy-expert.
- **P6 health-service registration (batch dispatch)**: ai-service, video-conferencing-service, onboarding-service — add `healthpb.RegisterHealthServer(grpcSrv, health.NewServer())` (5-10 LOC each). Pattern reference: services/metrics-service/cmd/server/main.go:165.

## FIX-YAML edits applied in this pass

See diff on `config/gateway-config.yaml` (commit alongside this audit). Smoke-verified live via `docker run --network slate_slate-network curlimages/curl`:

- 6 health routes now return `HTTP 200 {"status":"SERVING"}` (metrics, admin-auth, incidents, flags, messages, tenants).
- /api/metrics/platform, /api/ai/welcome, /api/grading/queue/count now return `HTTP 401` (auth-gated — no longer "proto: not found").

## Dispatch list (ratified 2026-04-18)

1. **metrics-expert (P1, CRITICAL)** — gRPC wrapper + 2 new RPCs (`GetRosterHealth`, `ExportMetrics`). Isomorphic JSON shapes. Est 30 min.
2. **assignment-grading-expert (P2)** — 5 new RPCs (`GetGradingQueue`, `BatchPublishGrades`, `GetGradingQueueCount`, `SaveDraft`, `GetDraft`). Same wrapper pattern — delegate to existing REST handlers.
3. **course-expert (P3)** — 4 new RPCs on `course.CourseService`. Course-module disambiguation (see §3).
4. **flag-expert (P4)** — `GetFlag(key) → Flag`.
5. **ai-expert + video-expert + incident-expert (P6 batch)** — register `grpc.health.v1.Health`. 5-10 LOC each.

## Provider FE bug flagged

- `frontend/provider/app/(dashboard)/teach/page.tsx:43` comment describes calling `/api/grading/queue/count`. User reports "Route not found" on bare `/api/grading/queue` — that path was never in yaml. If code actually hits the bare path, fix FE-side. For provider-fe-expert.
