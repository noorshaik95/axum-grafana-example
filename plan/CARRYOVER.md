# Slate Revamp — Carryover Items (post-Wave-5)

> Items not completed in the 2026-04-18/19/20 revamp session.
> Next session should pick these up in the order shown. Each references the original ask + the agent who'd own it.

## 1. ~~Gateway-expert config cleanup bundle~~ ✅ COMPLETED 2026-04-20 via gateway-cleanup-expert (Task #28)

All 9 items landed. Build green (130+ tests). Live-stack smoke matrix: 14 endpoints, zero 404 for ratified paths. flag_middleware role-sort + lowercase applied; all new FE route overrides added; deprecated admin-auth routes pruned; ListAdminRoles + GetAuditLog routes added; course REST passthroughs + metrics endpoints added; X-Request-ID echo restored (was a stale-image issue — rebuild fixed).

**New gap surfaced during cleanup — must be addressed before SSO/impersonation can work end-to-end:**

### 1a. HTTP-passthrough gap in api-gateway (ARCHITECTURAL)

**Problem**: user-auth-service exposes SSO + impersonation + MFA-reset as plain HTTP handlers on SERVER_PORT (not gRPC). The api-gateway only supports gRPC transcoding — no generic HTTP reverse-proxy capability. Therefore these routes 404 through the gateway:

- `GET /api/auth/sso/initiate`
- `GET /api/auth/sso/callback`
- `POST /api/auth/impersonate`
- `DELETE /api/auth/impersonate/:impersonation_id`
- `POST /api/admin/auth/mfa/reset`
- `POST /api/admin/auth/test-sso`

**Two options:**

- **(A)** Add a generic HTTP reverse-proxy middleware to api-gateway. New feature work — preserves the "UI → gateway → service" traceability the revamp is built around. Estimated ~3-5 hours.
- **(B)** Portals bypass gateway for these specific routes and hit user-auth-service directly (via a distinct env var `NEXT_PUBLIC_USER_AUTH_URL`). Faster, but breaks the unified-gateway principle + breaks traceparent propagation for those paths.

**Recommended owner:** `gateway-proxy-expert` (new spawn). Go with option (A). Acceptance: all 6 endpoints reachable via gateway with traceparent propagation + X-Request-ID echo, admin-auth-expert's 14 W2.5 tests still pass.

### 1b. Service-side Unimplemented surfacing

Gateway now correctly routes several endpoints, but the upstream gRPC methods may not yet exist and will surface as `Unimplemented`. Track and resolve:

- `assignment-grading-service`: `GetGradingQueue`, `BatchGrade`, `GetDraft`, `SaveDraft` (W9 shipped these but proto names may differ)
- `course-service`: `GetNextLecture`, `GetModules`, `GetNextUp`, `GetAnalytics`
- `metrics-service`: `GetGradeDistribution`, `ExportReport`
- `ai-service`: verify all 5 routes (cmd-palette, welcome, grade-projection, study-plan, draft-feedback) match gateway's method names

**Recommended owner:** `service-proto-audit-expert` (new spawn). Runs smoke-curls against gateway and flags any `Unimplemented` back to the relevant service expert.

## 2. #25 observability — live-stack verification

Observability-expert landed all deliverables (assert_trace.sh, 3 Grafana dashboards, Kafka 16-topic harness, doc refresh). Syntactically verified but not yet run against a live seeded stack. Once infra is live:

```
./scripts/seed-eastfield.sh
./scripts/assert_trace.sh admin /ops
./scripts/assert_trace.sh provider /grade/<id>
./scripts/assert_trace.sh student /today
./scripts/kafka_trace_test.sh
```

Expect ≥3 spans / ≥3 services per click path, and all 16 Kafka topics to propagate traceparent. If anything fails, the specific service's span is missing — escalate to that service's owning expert, don't patch for them.

## 3. Auth-tenant W14.x follow-up — impersonation validate end-to-end

Auth-tenant-expert flagged a non-blocking follow-up: end-to-end integration test from admin-auth issue → user-auth validate → tenant session JWT. Test infra exists (InMemoryRevocationStore, RSA keys) but the full E2E assertion isn't wired. Low priority — only matters when admin impersonation is used in prod.

## 4. #18 tenant-service deferrals

flag-expert flagged two rulings during W16:

- **(B)** Rename Kafka topic `tenant.provision_failed` → `tenant.provisioning_failed`. PO ruled: defer — no consumer wired yet.
- **(D)** RunMigrations gRPC vs self-migration. PO ruled: defer — self-migration is working.

These may surface when a new tenant service is added later. Address on first incident, not pre-emptively.

## 5. Non-priority FE pages (Wave 4 out-of-scope)

Admin: /onboarding, /data/import, /billing, /impersonation, /system/\* (health, kafka, alerts)
Provider: /lecture/live, /discussion, /analytics, /messages, /students
Student: /calculator, /files, /discussions, /people/:id, /profile

student-fe-expert stubbed the 3 student non-priority pages as "placeholder with redirect CTA." Other portals' non-priority pages compile but still use older patterns or stubs. A dedicated follow-up wave should wire these against their real APIs (most endpoints already exist).

## 6. Admin-auth W2.5 (#23) — in flight at session end

admin-auth-expert received proto-shape ratification from PO for ListAdminRoles / GetAuditLog / RefreshAdminToken. Coding should land next session. Acceptance: 3 new RPCs + migrations + tests + build-green.

## 7. Decorative component rescue

Per shared-ui-expert's Task #4 do-over, the following portal-local decorative components were deleted from shared: `bento-grid`, `particles`, `floating-dock`, `mode-toggle`. If any portal page still imports one (student `/discussions`, admin `/admin-service/analytics`, etc.), either remove the import or reintroduce a portal-local copy. `grep -rn "bento-grid\|particles\|floating-dock\|mode-toggle" frontend/{admin,provider,student}/app` after next session's build to catch any regressions.

---

## Session durable state (for next session)

- **Team:** `slate-revamp` at `~/.claude/teams/slate-revamp/config.json`
- **PO/analyst role:** respawn with the same prompt as this session; memory lives in `plan/PROGRESS.md` + `plan/CONTRACTS.md`
- **Plan docs:** `plan/EXECUTION.md`, `plan/CONTRACTS.md`, `plan/PROGRESS.md`, `plan/DESIGN_GAPS.md`, `plan/CARRYOVER.md` (this file)
- **Build-green baseline:** absolute. 3-portal tsc exit 0 last verified 2026-04-20. Any fresh work must preserve this.
- **T5 drift fix:** `@types/react@18.3.27` pinned at root via both explicit devDep + overrides block. Do not change.
