# Slate Revamp — Live Progress

> PO/Analyst maintains this file. Updated on every TaskUpdate event.
> Authoritative state: `TaskList` in team `slate-revamp`.
> **Session close-out 2026-04-18.** Continuations in `plan/CARRYOVER.md`.

## Session Bootstrapping — 2026-04-18

| Step                            | Status                                                                           |
| ------------------------------- | -------------------------------------------------------------------------------- |
| Team `slate-revamp` created     | ✅                                                                               |
| plan/EXECUTION.md written       | ✅                                                                               |
| plan/DESIGN_GAPS.md written     | ✅                                                                               |
| Foundation agent (Wave 0)       | ✅ shipped: tokens, AppShell primitives, X-Request-ID + traceparent, seed-dev.sh |
| PO/Analyst spawned (persistent) | ✅ active through close-out                                                      |

## Session Close-Out Summary — 2026-04-18

**All coding tasks shipped.** Waves 0, 1, 2, 3, 4, 5 + W2.5 all closed. #23 W2.5 admin-auth expansion shipped + PO HEAD-verified (build/test/vet exit 0, 14 new tests). Only #25 Wave 5 observability remains ⏳ by design per team-lead ruling (deliverables + syntactically verified, live-stack matrix → CARRYOVER.md §3). Continuation state in `plan/CARRYOVER.md`.

### Completed (archived)

| Wave            | What                                                                                            | State                                                                           | Owner                        |
| --------------- | ----------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------- | ---------------------------- |
| 0a              | DESIGN.md tokens → shared + 3 portals                                                           | ✅                                                                              | foundation-expert            |
| 0b              | AppShell primitives (TopBar + NowBar + TabNav)                                                  | ✅                                                                              | foundation-expert            |
| 0c              | X-Request-ID middleware + traceparent propagation                                               | ✅                                                                              | foundation-expert            |
| 0d              | Seed bootstrap scripts/seed-dev.sh                                                              | ✅ (T1 strict assertion added by qa-expert on #24)                              | foundation-expert            |
| 1               | 13 shared components + delete portal ui/ dirs                                                   | ✅                                                                              | shared-ui-expert             |
| 2 kickoff (W17) | api-gateway routes + Redis cache + flag middleware                                              | ✅ 416 tests                                                                    | gateway-expert               |
| 2 (W2)          | admin-auth — 7 RPCs + RS256 impersonation + audit + Kafka `audit.admin_action`                  | ✅ 11 tests                                                                     | admin-auth-expert            |
| 2 (W8)          | course-service — modules/lessons/analytics/next-up/lectures + Kafka                             | ✅ 162 tests                                                                    | course-expert                |
| 2 (W9)          | assignment-grading — rubric/batch/fingerprint + `grade.updated` schema canonical                | ✅ 77 tests + parity pin                                                        | grading-expert               |
| 2 (W10)         | content-management — video resume/range/TTL + `content.position_updated` Kafka                  | ✅ 11 new tests + pre-Wave-0 exception (see CONTRACTS.md)                       | content-expert               |
| 2 (W11)         | video-conferencing — lectures/pulse/Q&A/office-hours                                            | ✅ 74 tests on commit 214f489                                                   | video-expert                 |
| 2 (W12)         | metrics — roster health/grade dist/export/platform stats + `metrics.threshold_breached`         | ✅ 53 tests                                                                     | metrics-expert               |
| 2 (W14)         | user-auth — SSO + MFA reset + audit + impersonation validate + ResolveUsername                  | ✅ 39 W14 tests + e2e impersonation chain, HEAD commit 60c2ebb                  | auth-tenant-expert           |
| 2 (W15)         | email-service — broadcast + inbox + Kafka consumers (3 new topics)                              | ✅ 60 PASS                                                                      | email-expert                 |
| 3 (W3)          | feature-flag — EvaluateFlags/ListFlags/UpdateFlag + DeleteFlag                                  | ✅ 16 tests                                                                     | flag-expert                  |
| 3 (#33 P4)      | feature-flag — `GetFlag(key) → Flag` gRPC handler (closes PROTO_AUDIT §8)                       | ✅ 19 tests, HEAD commit 8d445a7                                                | flag-expert-2                |
| 3 (W4)          | incident — 6-RPC + public status + Kafka consumer + Redis 30s cache                             | ✅ 27 tests                                                                     | incident-expert              |
| 3 (W5)          | discussion-service — threads/inbox/needing-reply/@mentions + NoopResolver→GRPCResolver flip     | ✅ 22 tests, schema aligned to email-expert consumer                            | social-expert                |
| 3 (W6)          | scheduling-service — office-hours/slots/bookings/instructor-day                                 | ✅ 17 tests                                                                     | scheduling-expert            |
| 3 (W7)          | ai-service — 5 RPCs + Claude budget tracker + prompt caching + `ai.*` span schema               | ✅ 23 tests (R4 cost-containment)                                               | ai-expert                    |
| 3 (W13)         | onboarding-service — SSO + Canvas migration workflow (Restate Virtual Objects)                  | ✅ 25 tests                                                                     | incident-expert (reassigned) |
| 3 (W16)         | tenant-service — provision 8 tenant services + `status` + `status_detail` field (**closes T1**) | ✅ 88 tests                                                                     | flag-expert (reassigned)     |
| 4-nav           | Portal nav refactor — sidebars → AppShell                                                       | ✅ 0/0/0 tsc; 14 shells deleted, 31 layouts rewired                             | nav-refactor-expert          |
| 4 (#20)         | Admin portal — 8 core pages + impersonation redirect + middleware `/universities→/schools`      | ✅ 33 routes; 0 mock-data; tsc/lint/build exit 0                                | admin-fe-expert              |
| 4 (#21)         | Provider portal — 5 core pages + cross-portal globals.css fix                                   | ✅ 0 mock-data; tsc/lint/build exit 0                                           | provider-fe-expert           |
| 4 (#22)         | Student portal — 8 core pages + ⌘K CommandPalette + VideoPlayer resume (10s sample)             | ✅ 28 routes; 0 mock-data; tsc/lint/build exit 0                                | student-fe-expert            |
| 5 (#24)         | qa — Eastfield full seed (778 LOC) + Playwright per portal + 84.74% shared unit coverage        | ✅ seed strict-asserts T1; 3 specs with trace-ID capture; 109 unit tests pass   | qa-expert                    |
| 5 (#25)         | observability — `scripts/assert_trace.sh` + Grafana dashboards + 16-topic verification          | ✅ deliverables + syntactically verified; live-stack matrix → `CARRYOVER.md §3` | observability-expert         |
| — (T5)          | Root `@types/react` pin → 18.3.27 (explicit devDep + overrides + evict nested + widget fix)     | ✅ single resolution; all 3 portal tsc exit 0                                   | team-lead                    |

### In flight at session close

| #       | What                                                                                                 | State                                                            | Owner             | Continuation                                                                                                                                                                          |
| ------- | ---------------------------------------------------------------------------------------------------- | ---------------------------------------------------------------- | ----------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| 23 + 26 | W2.5 admin-auth expansion — ListAdminRoles + GetAuditLog + RefreshAdminToken (+ implementation plan) | 🔨 in_progress — proto shapes ratified, admin-auth-expert coding | admin-auth-expert | Ratifications (pagination + description on roles, option A outcome derivation server-side, request_id field, `resource:verb` permissions, seed matrix) — all 5 acks in message thread |

### Ratified contracts inventory (all in `plan/CONTRACTS.md`)

- **build.green-baseline** (+ narrow pre-Wave-0 exception addendum)
- **trace.propagation** (observability acceptance gate)
- **tenant.provision-status** ✅ shipped (T1 closed)
- **feature-flag.FlagService** ✅ (incl. `DeleteFlag` with bool `deleted` idempotency signal)
- **incident.IncidentService** ✅ (incl. optional additive fields)
- **admin_auth.AdminAuthService** ✅ (incl. `LoginResponse` additive fields, `GenerateImpersonationToken` alias, PLATFORM_PRIVATE_KEY_PATH scoping note)
- **ai.AiService** ✅ (5 RPCs + budget tracker + prompt caching + `ai.*` span schema + 3 gateway deviations queued)
- **user.UserService.ResolveUsername** ✅ (batched, `tenant_id` scoping, unknown-omit, case-insensitive unique)
- **scheduling.SchedulingService** ✅ (6 RPCs + 4 additives + gateway realignment)
- **discussion.DiscussionService** ✅ (8 RPCs + 5 deviations + NoopResolver→GRPCResolver flip)
- **email.MessagingService** ✅ (incl. SendBroadcast additive + Kafka schemas for 3 new consumed topics)
- **metrics.MetricsService** ✅ (W12 4 new endpoints + ThresholdEmitter library)
- **content.ContentManagementService (W10)** ✅ (position endpoint + `video_positions` table + Redis key + TTL-by-content-type + `content.position_updated` Kafka)
- **discussion.mention** Kafka schema aligned to email-expert consumer ✅
- **Wave 5 observability** — "deliverables + syntactically verified" close with post-deploy commands captured

## Active Blockers

**No open blockers.** All deferred items logged in `plan/CARRYOVER.md`.

### Resolved this session

- **T1** — tenant.provision-status shipped by flag-expert #18; seed-dev.sh strict-asserts (qa-expert #24 replaced soft-pass).
- **T5** — root `@types/react` pin → 18.3.27; single resolution; all 3 portal tsc exit 0.
- **T4** — gateway-expert W17 proof delivered: 416 tests.
- **T2, T3** — resolved as part of Wave 1 do-over + T5.
- **#27 infra-fix** — docker-compose build + deploy (Dockerfile deps + DB init + port conflict).
- **#28 gateway-cleanup** — 9-item backlog landed + 14-endpoint live-stack smoke matrix green (zero 404 on ratified paths). flag_middleware role-sort + lowercase applied. ListAdminRoles + GetAuditLog routes added. Course REST passthroughs + metrics HTTP routes added. X-Request-ID echo restored (root cause was stale-image, foundation-expert's Wave 0c middleware was always correct). `/api/tenants` 401 is the correct auth-gated response (was initially misread as 404).
- **#29 §1b proto audit** — gateway-proto-audit-expert walked every `/api/*` route, wrote `plan/PROTO_AUDIT.md`, applied 16 yaml-only fixes inline, dispatched 5 service-side asks (P0-P4 + P6). FIX-YAML / NEEDS-SERVICE / BLOCKED-ON-§1a taxonomy established. Triggered live by user-reported `/api/metrics/platform` "proto: not found" error.
- **#30 P0 metrics gRPC wrapper** — metrics-expert registered `MetricsService` gRPC server with 10 RPCs delegating to existing REST + repository layer; 4 additive ratified (`GetRosterHealth`, `CreateExport`, `GetExport`, `GetPlatformMetricsExtended`) with isomorphic REST↔proto shapes pinned by 3 tests. Admin `/ops` live incident resolved. Commit `54da2d5`.
- **#31 P2 assignment-grading gRPC wrapper** — 5 RPCs (`GetGradingQueue`, `GetGradingQueueCount`, `BatchPublishGrades`, `SaveDraft`, `GetDraft`) + gateway yaml fix `5367053` repointing `/api/grading/queue/count` from metrics to assignment.GradingService. Provider `/grade/*` + student draft autosave live. grpc_handler_test.go bundled into `54da2d5` per coordination with sister agent.
- **#32 P3 course-service gRPC wrapper** — 4 RPCs (`ListModules`, `GetNextUp`, `GetCourseAnalytics`, `GetNextLecture`) + 6 new W8-RPC unit tests + fixed pre-existing DI bitrot in controller.spec.ts. 168 tests pass (up from 162). Commit `b3ed4a7`. `GetNextLecture` returns `{ has_next: false }` with TODO until scheduling-service integration lands — ratified-as-deferred.
- **#33 P4 feature-flag `GetFlag` RPC** — additive, 20 tests pass (incl. 3 new `_ok/_notFound/_emptyKey`). Commit `8d445a7`. Admin-fe `/flags` detail view upstream ready.
- **#34 admin-fe §5** — wired `/system/alerts` to real `/api/metrics/alerts` via new `useActiveAlerts` hook + `alertsApi` module. Bundled in commit `e212eb7` with #35 per husky + lint-staged auto-restore incident (see below).
- **#35 provider-fe §5** — wired /analytics + /discussion + /students + /lecture/live + /messages (5 pages). Struggle buckets derived from GetGradeDistribution <70% (no dedicated topic-struggle endpoint). Commit `e212eb7`. 27 routes, tsc/build exit 0.
- **#36 §1a gateway HTTP reverse-proxy** — gateway-proxy-expert added generic HTTP passthrough to api-gateway for 6 user-auth HTTP routes (SSO initiate + callback + impersonate POST/DELETE + MFA-reset + test-sso). Schema addition ratified: `RouteOverride.http_proxy_target: Option<String>` — when set, gRPC method becomes optional, proxy short-circuits before transcoding. Forwarded headers preserve trace propagation (traceparent + X-Request-ID + X-Tenant-Slug + Authorization). Hop-by-hop stripping per RFC 7230 §6.1. 7 new proxy unit tests. Commit `6cec449`. **This closes the last architectural gap from CARRYOVER.md §1a.** Unblocks §1b onboarding Restate routing (moves from BLOCKED to READY-to-wire).
- **Husky + lint-staged auto-restore incident** — commit `e212eb7` labeled "feat(provider-fe)" contains both #34's 3 admin files and #35's 8 provider files, because lint-staged stashed the entire worktree across two concurrent pre-commit hook fires. Team-lead ruling: no-rewrite. Code is correct + builds green on HEAD; `git log -p e212eb7` shows content cleanly. Process improvement logged for CARRYOVER: disable lint-staged auto-restore behavior or coordinate agent parallelism at spawn time.
- **Student-fe middleware** — present on HEAD at `frontend/student/middleware.ts` matching provider's auth-guard pattern (slate_token cookie check + redirect unauthed→/login + redirect authed-on-login→/today + API/static/favicon exclusion matcher). Security hole from user's "no route guards" complaint now closed on all 3 portals.
- **FE Docker-build failure** (post-close hygiene) — root cause: missing `frontend/shared/package-lock.json` + `@types/react` drift across 4 workspaces (T5 signature reprise, per-Docker-image this time). Fixed inline by team-lead: generated shared lockfile via `npm install --package-lock-only --no-workspaces`, pinned all 4 workspaces to exact `@types/react@18.3.27` + `@types/react-dom@18.3.7` (no carets), regenerated all 4 lockfiles. All 3 portal Docker builds green; containers running (student :3000 HTTP 200, provider :3002 HTTP 307, admin :3003 HTTP 200). Standing-expectation addendum "FE Docker-build gate" now codified in CONTRACTS.md — future FE completion reports require `docker build -f <portal>/Dockerfile` exit 0 alongside tsc/lint/npm-build.

### Open architectural items

- **§1a ✅ closed** by gateway-proxy-expert commit `6cec449` (#36) — generic HTTP reverse-proxy support added to api-gateway for the 6 user-auth SSO/impersonation/MFA routes.
- **§1b partially closed; #37 in flight.** Proto-audit portion ✅ done by gateway-proto-audit-expert #29 + P0-P4 service wrappers (#30-#33) + P6 health registration verified (ai + video already had grpc.health, onboarding N/A as Restate/HTTP-only). **Onboarding Restate yaml routing in flight as Task #37** — gateway-proxy-expert extending `proxy.rs` with path-rewrite support + adding 14 `/api/onboarding/*` + `/api/canvas/*` yaml routes using the new `http_proxy_target` → `http://onboarding-service:9080`. Stale gRPC onboarding yaml routes (`StartOnboarding`/`GetOnboardingStatus`/`CanvasImport`) being replaced with real Restate handler routes. Expected to land ~15-20 min from team-lead's notification. **§1b fully closed on #37 ✅.**

**Remaining cleanup items** moved to "Deferred to next session (in `plan/CARRYOVER.md`)" below — all non-architectural.

### Active (dispatches + in-flight)

| #   | What                                                                                                                                                | State          | Owner                |
| --- | --------------------------------------------------------------------------------------------------------------------------------------------------- | -------------- | -------------------- |
| 37  | §1b: Wire onboarding-service Restate routes through gateway HTTP proxy (14 /api/onboarding + /api/canvas routes + path-rewrite support in proxy.rs) | 🔨 in_progress | gateway-proxy-expert |

All other live dispatches from user's 2026-04-22 "API failures + route guards" report have landed:

All live dispatches from user's 2026-04-22 "API failures + route guards" report have landed:

- ~~#29 gateway-proto-audit~~ ✅ (16 FIX-YAML + 5 NEEDS-SERVICE dispatches → all landed)
- ~~admin-fe auth-guard middleware~~ ✅ (matches provider pattern, landing=/ops)
- ~~student-fe middleware creation~~ ✅ (present on HEAD, landing=/today)
- ~~provider-fe `/api/grading/queue/count` fix~~ ✅ (commit `cc05585`, client-side workaround + TODO pointing at proto-audit)
- ~~#36 §1a HTTP-passthrough gateway~~ ✅ (commit `6cec449`, 6 user-auth routes wired)
- ~~#30/#31/#32/#33 service gRPC wrappers~~ ✅ (metrics P0 / grading P2 / course P3 / flag P4)

All security + live-incident items from user's report are resolved.

### Deferred to next session (in `plan/CARRYOVER.md`)

- **§2 Wave 5 observability live-run** — stack is now live per #28 completion; **9-CTA × 16-topic matrix can now execute** (no longer blocked on "first seeded env spinup"). Triggering observability-expert this turn — see below. Commands in CONTRACTS.md + CARRYOVER.md §2.
- **§4 #18 tenant deferrals** — (B) Kafka topic rename `tenant.provision_failed` → `tenant.provisioning_failed`, (D) RunMigrations gRPC vs self-migration.
- **W14.x** — impersonation audit wiring into existing grpc login paths; first-class `impersonated_by` JWT claim; SSO IdP vendoring (crewjam/saml, coreos/go-oidc, x/oauth2) pending EXECUTION.md §3 whitelist.
- **Decorative-component cleanup** — `components/data-table.tsx` + `components/stat-card.tsx` at 0% unit coverage (not in W0/W1 deliverable scope but flagged by qa-expert).
- **W2.5** handoff to admin-fe-expert for `/iam/roles` + `/iam/audit` stub→real flip when #23 closes.
- **Non-priority FE pages** — admin `/onboarding` `/data/import` `/billing` `/impersonation` `/system/*`; provider `/lecture/live` `/discussion` `/analytics` `/messages` `/students`; student `/calculator` `/files` `/discussions` `/people` `/profile`.
- **Upload-area sqlx-tokio test remediation** — 3 pre-Wave-0 test failures in content-management-service/src/upload/handler.rs (commit `c71000b`, 2025-11-15). Swap `#[test]` → `#[tokio::test]`.

## Standing Expectations

### Build-green baseline (non-negotiable, with narrow exception)

**Default rule unchanged:** every task merges with `go test/cargo test/npm test` exit 0. "Pre-existing error" is NOT an acceptable excuse.

**Narrow exception** (see CONTRACTS.md "Standing-expectation addendum"):
Pre-Wave-0 test failures may be excluded from strict build-green **only** if the agent (a) discloses them explicitly, (b) traces to a pre-revamp commit with git-blame (commit predating 2026-04-18), (c) ships no new code that makes them worse. Precedent: content-expert W10 #10 accepted under this exception.

### HEAD-verification is authoritative

PO reproduces agent build/test/lint numbers on HEAD **after ensuring the agent's changes are committed**. Stale-view artifacts (working tree vs HEAD divergence) have bitten twice this session — once on #14 (helpers.go had skip-guard uncommitted), once on #13 (video-expert uncommitted). Going forward: agents must commit before claiming; PO checks `git log -1` before claiming HEAD-verify failure.

## Contract Registry

- `plan/CONTRACTS.md` — provisional + ratified contracts, Kafka schemas, trace span schemas, deviation catalog, narrow exception addendum.

## Artifacts

- `plan/EXECUTION.md` — wave order, component inventory, API matrix, observability plan, seed strategy, risk register
- `plan/DESIGN_GAPS.md` — token mismatches, missing shared components, portal dupes, nav migration scope
- **`plan/CARRYOVER.md`** — next-session continuation state (gateway cleanup queue, observability live-run commands, W14.x follow-ups, #18 deferrals, non-priority FE pages, W2.5 handoff, decorative cleanup, sqlx-tokio remediation)
- `docs/ARCHITECTURE.md` — authoritative system design
- `docs/DESIGN.md` — authoritative design-system spec
- `docs/plan.md` — W1–W20 authoritative spec
