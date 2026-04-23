# Service Contracts (provisional)

Managed by po-analyst. Agents must not deviate without ratification.

---

## build.green-baseline (non-negotiable, applies to every task)

Defined: 2026-04-18 by po-analyst (team-lead directive)
Consumers: **every agent**, every task, every wave
Status: **mandatory** — build failure blocks task completion

The build passed after Wave 0. That is the baseline. Every task from Wave 1 onward must merge with a fully green build.

### Rules

1. **Typecheck + lint + unit tests must pass** on task completion. The command the repo uses at root (e.g. `pnpm typecheck && pnpm lint && pnpm test`, or the Go/Rust equivalents for services) must exit 0.

2. **"Pre-existing error" is not an acceptable excuse.** The build was green at Wave 0 handoff. Any red error is something this revamp effort introduced — or something your current change exposed. Either way, it's now your task's responsibility to fix before closing.

3. **No scope-downgrade escape hatches.** Rejected patterns:
   - "I'll land the token fix and follow up on the imports" → no. Land both.
   - "This 40-file refactor is too risky" → no. Codemod + typecheck is not a refactor.
   - Re-export shims preserving a forbidden path → no.
   - Blanket `// @ts-ignore` or `eslint-disable` lines without a documented justification → no.
   - Carving out packages from CI → no.
   - "Move X to shared to be safe" when X isn't in the inventory → no; delete or leave, don't dump tech debt into shared.

4. **If a genuine pre-existing blocker appears** (flaky test red since before Wave 0, external service gone down, etc.), SendMessage po-analyst **before** closing the task with a specific diagnosis. PO routes to team-lead if needed. Silent downgrades of the bar are not accepted.

5. **Completion reports must state explicitly:** "typecheck green, lint green, tests green" (or the service-specific equivalent). A report without this statement is treated as incomplete.

6. **If your change breaks a teammate's in-flight work**, that's a coordination problem — SendMessage them through po-analyst, don't merge over it.

---

## runtime-up gate (non-negotiable, applies to every task touching services or infra)

Defined: 2026-04-22 by team-lead (post-Tier-1 E2E failure analysis)
Consumers: **every agent** touching docker-compose.yml, Dockerfiles, service code, gateway config
Status: **mandatory** — complements build.green-baseline; build green is necessary but not sufficient

Three gates must all pass before a task is closed:

1. **Gate 1 — Local build/test green**
   `cargo build && cargo test` (Rust) / `go build ./... && go test ./...` (Go) / `pnpm typecheck && pnpm lint` (FE) exits 0.

2. **Gate 2 — Docker Compose build**
   `docker compose build <service>` exits 0 with no layer errors. If the change affects multiple services, build all of them.

3. **Gate 3 — Runtime healthy**
   `docker compose up -d --no-deps <service>` followed by `docker compose ps <service>` must show `healthy` (not `starting` or `restarting`) within 60s.
   Then smoke-test the critical path: at minimum one curl to a route the service owns must return the expected 2xx (or expected 404/401 for auth-gated routes).

### Anti-patterns that caused the Tier-1 cascade (2026-04-22):

- Unit tests passed but bcrypt hash was wrong in migration SQL → Gate 3 required.
- circuit_breaker.rs fix landed but old gateway binary still running → Gate 3 required.
- YAML route changed but gateway container not restarted → Gate 3 required.
- Service added to compose but missing env var (TENANT_DB_DSN) → Gate 3 catches on boot failure.

### Gate 3 timeout policy

If the service fails to become healthy within 60s, run `docker compose logs <service> --tail 50` and diagnose before declaring completion. Do not close the task with a restartng container.

### Gate 3.1 — Commit on HEAD (mandatory, every task)

Defined: 2026-04-23 by po-analyst (three uncommitted-work incidents in one session)
Applies to: **every agent** on every task — FE, backend, infra, config

Before reporting a task complete, run:

```
git log --oneline | head -3
```

Confirm your commit SHA appears at the top. Paste the SHA in your completion report.

A working-tree change that is not committed to HEAD is **not complete**. The next `docker compose build`
from HEAD will not include it, and the next agent to touch the same files will not see it. Three incidents
in April 2026 slipped through because agents verified the running container (which had the fix) but never
committed — leaving the repo in a broken state for everyone downstream.

**Required completion report line:**

> Committed: `<sha>` — verified on HEAD via `git log --oneline | head -1`

---

## trace.propagation (observability acceptance gate)

Defined: 2026-04-18 by po-analyst (ratified by foundation-expert's Wave 0c delivery)
Consumers: **every** Wave 2/3 service expert
Status: **mandatory** — non-negotiable

Every service (existing or new) must:

1. **Extract on incoming requests** — pull `traceparent` and `X-Request-ID` from inbound HTTP/gRPC metadata. If `X-Request-ID` missing, generate one and echo on the response.
2. **Tag its span** — attach `request_id` and `tenant.slug` attributes to the active span.
3. **Propagate outbound** — inject `traceparent` + `X-Request-ID` on outbound gRPC (via `otelgrpc`), outbound HTTP, and Kafka message headers.

### Go services — use `libs/common-go/tracing`

- `TagSpanWithCorrelation(ctx, spanAttrs)` — attach request_id + tenant.slug on current span
- `InjectTraceparent(ctx, outboundMetadata)` — for outbound gRPC/HTTP
- `ExtractTraceparent(incomingMetadata) → ctx` — for inbound
- `KafkaHeadersFromContext(ctx) → []kafka.Header` — producer
- `ContextFromKafkaHeaders(headers) → ctx` — consumer

### Rust services — use `libs/common-rust/src/observability/propagation`

- `inject_traceparent(cx, &mut req_headers)` — outbound HTTP/gRPC
- `read_correlation_from_incoming(&req_headers) → Correlation` — inbound
- `tag_span_with_correlation(&span, &correlation)` — span attributes
- `kafka_headers_for(cx) → Vec<Header>` — producer
- `context_from_kafka_headers(&headers) → Context` — consumer

### Acceptance (per service PR)

- Grafana/Tempo trace shows the service's span **connected to the parent** from the calling service.
- `X-Request-ID` flows end-to-end from browser → gateway → service → downstream service without breaks.
- Kafka consumer spans link back to producer span via traceparent headers.

---

## tenant.provision-status

Defined: 2026-04-18 by po-analyst (from team-lead ask after foundation-expert's seed-dev.sh report)
Consumers: seed-dev.sh (foundation-expert, done), Eastfield seed (qa-expert, Wave 5), admin onboarding UI (admin-fe Wave 4)
Status: **provisional — to be ratified by tenant-expert on W16**

### Problem

`scripts/seed-dev.sh` currently **soft-passes** tenant provisioning because the tenant-service stub returns 200 immediately without a lifecycle state. Seed can't tell "provisioning in progress" from "container stuck on first boot" from "done and healthy," so it moves on prematurely. Same gap will hit the Eastfield seed in Wave 5.

### Required change

tenant-service must expose a `status` field on provision responses and on the tenant detail endpoint.

### Response shape (provisioning endpoint + GET /api/tenants/:id)

Add field `status` to the existing response:

```json
{
  "id": "uuid",
  "slug": "eastfield",
  "name": "Eastfield University",
  "status": "provisioning" | "active" | "failed",
  "status_detail": "string — human-readable; e.g. 'waiting on course-service health check'",
  "created_at": "iso8601",
  "updated_at": "iso8601"
}
```

### Lifecycle semantics

- `provisioning` — containers starting, migrations running, Traefik config being written. Transient.
- `active` — all 8 tenant containers report healthy + Traefik routes resolving + initial admin user exists. Terminal success.
- `failed` — any step errored; `status_detail` carries the reason. Terminal failure; requires manual retry via a new provision call.

### Kafka event (optional, nice-to-have)

If tenant-expert already emits `tenant.provisioned` on success, extend with a `tenant.provisioning_failed` topic carrying `{tenant_id, slug, failed_step, error}` so email-service can alert the platform admin.

### Seed assertion (post-ratification)

`scripts/seed-dev.sh` will poll `GET /api/tenants/:id` every 2s for up to 120s, fail hard if status stays `provisioning` past timeout or ever transitions to `failed`.

### Open questions for tenant-expert

1. Is `provisioning → active` atomic when the last container is healthy, or do we need a 2-step (`containers_up → routes_verified → active`)?
2. Does the existing DB column exist or do we need a migration? (If migration: flag on W16.3 task so tenant-expert ships it alongside network/config changes.)

---

## feature-flag.FlagService (W3)

Defined: 2026-04-18 by po-analyst (from gateway-expert proposal, 2026-04-18)
Consumers: api-gateway (W17.3 flag middleware), admin-fe `/flags` (W18.4)
Status: **provisional — ratify with flag-expert on W3**

gRPC `:50062`, HTTP exposed at `/api/flags/*`.

### Proto `flag.FlagService`

```proto
rpc EvaluateFlags(EvaluateFlagsRequest) returns (EvaluateFlagsResponse);
// Gateway-hot-path. Called once per authenticated request (via Redis cache).
message EvaluateFlagsRequest {
  string tenant_id   = 1;
  string tenant_slug = 2;   // ADDED vs gateway proposal — plan W3.2 EvalContext requires slug
  string user_id     = 3;
  repeated string roles = 4;
}
message EvaluateFlagsResponse {
  map<string, bool> flags = 1;
  int64 evaluated_at_unix_ms = 2;
}

rpc ListFlags(ListFlagsRequest) returns (ListFlagsResponse);
// Admin-only. For /api/flags GET.
message ListFlagsRequest  { string tenant_id = 1; }
message ListFlagsResponse { repeated Flag flags = 1; }
message Flag {
  string key                = 1;
  string description        = 2;
  bool   state              = 3;
  repeated TargetRule targets = 4;
  int64  updated_at_unix_ms = 5;
}
message TargetRule {
  // Rule types per plan W3.2: "all" | "tenant" | "role" | "percentage" | "user"
  string rule_type = 1;
  // JSON-encoded value matching rule_type, e.g. {"roles":["instructor"]} or {"percent":50}
  string rule_value_json = 2;
}

rpc UpdateFlag(UpdateFlagRequest) returns (Flag);
// Admin-only. For PUT /api/flags/:key. Invalidates Redis cache platform:flags:{tenant_id}* on success.
message UpdateFlagRequest {
  string key = 1;
  bool   state = 2;
  repeated TargetRule targets = 3;
}
```

### Gateway cache behaviour (W17.3)

- Redis key: `platform:flags:{tenant_id}:{sha1(user_id,roles)}`
- TTL: 5 minutes
- Miss → call `EvaluateFlags` gRPC → cache result
- Service-down → return empty map, request proceeds (fail-open)
- On `UpdateFlag` → evaluator must publish Redis `DEL platform:flags:{tenant_id}:*` to purge that tenant's cached evaluations

### Deltas vs docs/plan.md W3

- plan W3.1 mentioned a method `GetFlags(tenant_id, user_id) → []Flag` (ambiguous between evaluate and list). Gateway proposal splits this into **EvaluateFlags** (hot-path bool map) and **ListFlags** (admin full Flag records). **Accepted** — split is correct.
- Added `tenant_slug` to EvaluateFlagsRequest to match plan W3.2 `EvalContext{TenantID, TenantSlug, UserID, Roles}`. flag-expert please preserve.

### Ratification hooks

flag-expert: if you need to deviate on method names or add fields, SendMessage po-analyst with a diff before implementation.

### Ratified additions (2026-04-18, flag-expert build accepted by team-lead)

- **`rpc GetFlag(GetFlagRequest) returns (Flag)`** — additive, ratified 2026-04-18 on flag-expert-2 P4 completion (Task #33 ✅, part of proto-audit §4 dispatch).
  - `GetFlagRequest { string key = 1; string tenant_id = 2; }`
  - Returns ratified `Flag` message verbatim (same shape as `ListFlagsResponse.flags[]` entry).
  - Semantics: `codes.NotFound` when key doesn't exist (cleaner than disabled-default); `codes.InvalidArgument` on empty key; tenant-scoped cache (`platform:flags:{tenant_id}:*` scan).
  - Gateway route: `GET /api/flags/:key → flag.FlagService/GetFlag` (already in `config/gateway-config.yaml` from earlier proto-audit FIX-YAML pass — resolves on next gateway reflection refresh).
  - Status: **ratified**. flag-expert-2 build green (20 tests, incl. 3 new GetFlag-specific: `_ok` / `_notFound` / `_emptyKey`). Handler at `services/feature-flag-service/internal/grpc/server.go:69`.

- **`rpc DeleteFlag(DeleteFlagRequest) returns (DeleteFlagResponse)`** — additive, ratified. Needed by admin-fe `/flags` for flag retirement flow.
  - `DeleteFlagRequest { string key = 1; string tenant_id = 2; }`
  - `DeleteFlagResponse { bool deleted = 1; }` — `true` when a row was removed, `false` when the key didn't exist (idempotent). Gateway maps `true` → HTTP 200, `false` → HTTP 404.
  - Semantics: hard delete (no tombstone).
  - Implementation MUST `DEL platform:flags:{tenant_id}:*` on successful delete so stale evaluations do not leak (same invariant as UpdateFlag).
  - Gateway-expert: wire `DELETE /api/flags/:key` against this method when admin-fe `/flags` lands.
  - **Amendment note (2026-04-18, PO):** original ratification text said `returns (google.protobuf.Empty)` in prose; flag-expert shipped `DeleteFlagResponse{bool deleted}` in the proto. PO amended ratification to match HEAD — the `deleted` bool is a useful idempotency signal (200 vs 404 mapping). No consumer wired yet, so contract-matches-code with zero downstream impact.
- **Status:** ratified. flag-expert build reported green (`go build`, `go test` 16 tests exit 0). Service marked complete.

---

## incident.IncidentService (W4)

Defined: 2026-04-18 by po-analyst (from gateway-expert proposal, 2026-04-18)
Consumers: api-gateway (/api/incidents/\* + public /api/status), admin-fe `/ops` `/incidents` `/incidents/[id]` `/status` (W18.2, W18.3)
Status: **provisional — ratify with incident-expert on W4**

gRPC `:50061`, HTTP at `/api/incidents/*` (auth) and `/api/status` (public).

### Proto `incident.IncidentService`

```proto
rpc ListIncidents(ListIncidentsRequest) returns (ListIncidentsResponse);
message ListIncidentsRequest {
  optional string tenant_id = 1;
  optional StatusFilter status_filter = 2;  // Open | Watch | Resolved
  int32 limit = 3;
}
enum StatusFilter { OPEN = 0; WATCH = 1; RESOLVED = 2; }

rpc GetIncident(GetIncidentRequest) returns (Incident);
rpc CreateIncident(CreateIncidentRequest) returns (Incident);
rpc UpdateIncident(UpdateIncidentRequest) returns (Incident);  // ADDED — present in plan W4.2, missing from proposal
rpc PostIncidentEvent(PostIncidentEventRequest) returns (IncidentEvent);
// ^ Equivalent to plan W4.2 `AddEvent`. Naming: PostIncidentEvent accepted (HTTP verb affinity).

rpc GetPublicStatus(google.protobuf.Empty) returns (PublicStatusResponse);
// Unauthed. Powers /api/status public page. Replaces plan W4.2 `GetServiceStatus` (renamed for clarity that it's public).
message PublicStatusResponse {
  OverallStatus overall = 1;  // Operational | Degraded | Outage
  repeated ComponentStatus components = 2;
  int32 incidents_last_7d = 3;
}

message Incident {
  string id = 1;
  optional string tenant_id = 2;
  Priority priority = 3;  // P0..P4
  IncidentStatus status = 4;
  string title = 5;
  string impact = 6;
  int64 opened_at_unix_ms = 7;
  optional int64 resolved_at_unix_ms = 8;
  repeated IncidentEvent events = 9;
}
```

### Deltas vs docs/plan.md W4

- plan W4.2 named the event method `AddEvent`; gateway uses `PostIncidentEvent`. **Accepted** (clearer intent, consistent with HTTP POST).
- plan W4.2 named the status method `GetServiceStatus`; gateway uses `GetPublicStatus`. **Accepted** (clarifies unauth path).
- `UpdateIncident` **restored** in the ratified contract — needed for `/incidents/[id]` status transitions (Open → Watch → Resolved, priority changes).

### Ratification hooks

incident-expert: Kafka consumer for `metrics.threshold_breached` (plan W4.3) is unchanged. Redis cache `platform:incidents:active` TTL 30s per plan W4.4 also unchanged.

### Ratified additions (2026-04-18, incident-expert build accepted)

All additive, optional, no rename/remove of ratified fields. Admin-fe + gateway may depend on these.

- **`Incident.service` (optional string)** — drives `GetPublicStatus` per-service aggregation without a separate services table.
- **`Incident.created_by` (optional)** — surfaces the already-persisted `created_by` column.
- **`ComponentStatus.highest_priority` (optional)** — lets admin `/status` page render the worst active priority per service.

**Status:** ratified. incident-expert build green (go build/test/vet + protoc all exit 0; 27 tests across 4 packages). Service marked complete.

---

## admin_auth.AdminAuthService (W2)

Defined: 2026-04-18 by po-analyst (from gateway-expert proposal, 2026-04-18)
Consumers: api-gateway (/api/admin/auth/_, /api/admin/impersonate/_), admin-fe `/iam/users` `/users/[id]` (W18.5)
Status: **provisional — ratify with admin-auth-expert on W2**

gRPC `:50060`, HTTP at `/api/admin/auth/*` and `/api/admin/*`.

### Proto `admin_auth.AdminAuthService`

```proto
rpc AdminLogin(LoginRequest) returns (LoginResponse);
// POST /api/admin/auth/login. Issues JWT with aud=platform.

rpc AdminLogout(LogoutRequest) returns (LogoutResponse);
// POST /api/admin/auth/logout. Revokes token in Redis.

rpc ValidateAdminToken(ValidateTokenRequest) returns (ValidateTokenResponse);
// Same shape as user-auth's ValidateToken but with aud=platform check.

rpc ListAdminUsers(ListAdminUsersRequest) returns (ListAdminUsersResponse);
// GET /api/admin/users. NEW vs plan W2.1 scope — added to support admin-fe /iam/users (W18 requirement).

rpc Impersonate(ImpersonateRequest) returns (ImpersonateResponse);
// POST /api/admin/impersonate/:tenant_id/:user_id. Equivalent to plan W2.2 GenerateImpersonationToken.
message ImpersonateRequest {
  string admin_user_id = 1;    // from caller's platform JWT
  string tenant_id     = 2;
  string tenant_slug   = 3;
  string target_user_id = 4;
}
message ImpersonateResponse {
  string redirect_url = 1;          // http://{slug}.slate.local/auth/impersonate?token={token}
  string impersonation_id = 2;      // revocation key; stored in Redis tenant:{slug}:revoked:{id}
  int64  expires_at_unix_ms = 3;    // 5 min TTL per plan W2.2
}
```

### Deltas vs docs/plan.md W2

- Method renamed: plan W2.2 `GenerateImpersonationToken` → gateway's `Impersonate`. **Accepted**.
- `ListAdminUsers` is **new** — not in plan W2.1 scope but required for admin-fe W18 `/iam/users`. admin-auth-expert: include in first PR.
- Plan W2.1 also lists `Register` (create platform admin). Gateway proposal omitted — may be manual DB/CLI for first pass. admin-auth-expert: flag if `Register` is needed before admin-fe `/iam/users` create flow.

### Ratification hooks

- Impersonation token signing: RSA using platform private key, `type=impersonation` claim, `exp=5m` — per plan W2.2. user-auth-service (W14.4) validates using platform public key.
- All admin actions append to `platform_audit` table + Kafka `audit.admin_action` per plan W2.3.

### Ratified build (2026-04-18, admin-auth-expert)

- **Status:** ratified. `go build`, `go test -count=1` (11 tests: 5 jwt + 5 service + 1 revoker), `go vet` all exit 0. Integration tests present: `TestAdminLoginValidateLogoutFlow`, `TestImpersonateRoundTripAndVerify`, `TestImpersonateRejectsRolesWithoutPermission`.
- All 7 RPCs shipped per contract: AdminLogin, AdminLogout, ValidateAdminToken, ListAdminUsers, Register, Impersonate, GenerateImpersonationToken. Zero deviations.
- RS256 impersonation token signing + audit table + Kafka `audit.admin_action` all in.
- **Downstream TODOs (non-blocking; flagged for later waves):**
  - **Redis-backed revocation** — currently in-memory. Plan W2 did not require it; fine for first pass. Revisit before production hardening; consumer: user-auth validate path when it needs to honour revocation across admin-auth pods.
  - **`PLATFORM_PRIVATE_KEY_PATH` docker secret mount** — required before `/api/admin/impersonate` runs in prod. **Action: W16 tenant-expert (#18) must include a `secret` volume mount for `PLATFORM_PRIVATE_KEY_PATH` in the generated compose/Traefik config per-tenant**, and user-auth (W14, #14) must mount the corresponding `PLATFORM_PUBLIC_KEY` (already in R6 risk register).

---

## ai.AiService (W7) — ratified

Defined: 2026-04-18 by po-analyst (from ai-expert completion proof, accepted by team-lead + PO HEAD-verify).
Consumers: api-gateway (`/api/ai/*`), student `/today` `/plan` `/grades/[courseId]` `⌘K`, provider grading `RubricEditor` (feedback).
Status: **ratified**. ai-expert `#8` build green (go build/test/vet exit 0, 23 tests); prompt caching + R4 budget tracker both shipped first commit.

gRPC tenant service on `:50064`, proto `/slate.ai.v1.AiService`.

### RPCs (all 5 shipped)

```proto
rpc GetWelcomeMessage        (WelcomeMessageRequest)  returns (WelcomeMessageResponse);
rpc GetGradeProjection       (GradeProjectionRequest) returns (GradeProjectionResponse);
rpc GenerateStudyPlan        (StudyPlanRequest)       returns (StudyPlanResponse);
rpc GetCommandPaletteResults (CmdPaletteRequest)      returns (CmdPaletteResponse);
rpc GetDraftFeedback         (DraftFeedbackRequest)   returns (DraftFeedbackResponse);
```

HTTP mapping (gateway REST→gRPC):

- `GET  /api/ai/welcome` → GetWelcomeMessage
- `GET  /api/ai/grade-projection` → GetGradeProjection
- `POST /api/ai/study-plan` → GenerateStudyPlan
- `POST /api/ai/cmd-palette` → GetCommandPaletteResults
- `POST /api/ai/feedback` → GetDraftFeedback (provider grading)

### Ratified additive

- **GetDraftFeedback / `POST /api/ai/feedback`** (W7.6). Not in EXECUTION.md §API-Matrix original 4 (welcome/study-plan/grade-projection/cmd-palette); present in `docs/plan.md` W7. Added to the matrix here for FE agents.
- Wire field **`DraftFeedbackRequest.flag_enabled` (bool)** — REST layer must flip on feature-flag lookup (`ai_draft_feedback`) **before** forwarding. gateway-expert: wire this.

### R4 budget tracker (cost-containment, ratified)

- Single-chokepoint in `internal/claude/claude.go::Invoke()`: `budget.Check(ctx, tenantSlug)` BEFORE SDK call, `budget.Record(ctx, tenantSlug, tokens)` AFTER response.
- Redis key: `tenant:{slug}:ai:tokens_used:{YYYY-MM}` with end-of-month TTL.
- Hard cap env: `TENANT_AI_TOKEN_BUDGET` (default 1_000_000).
- Over-limit: `ErrBudgetExceeded` → gRPC `ResourceExhausted`.

### Prompt caching (ratified, mandatory on every Claude call)

- `CacheControlEphemeralParam{Type:"ephemeral"}` on the system block of every `Invoke()`.
- System prompts as package-level const strings (`welcomeSystemPrompt`, `studyPlanSystemPrompt`, `cmdPaletteSystemPrompt`, etc.) guarantee stable prefix for cache hit.
- Invariants pinned by `TestPromptCachingStructure` + `TestCmdPalette_StablePromptForCaching`.

### Span attribute schema (`ai.*`) — ratified namespace

Consumers: billing/cost-ledger joiner (W12 metrics or future rollup), Grafana/Tempo cost dashboards.

| Attribute               | Type   | Source                                                                            |
| ----------------------- | ------ | --------------------------------------------------------------------------------- |
| `ai.provider`           | string | constant `"anthropic"`                                                            |
| `ai.purpose`            | string | one of `welcome` / `grade_projection` / `study_plan` / `cmd_palette` / `feedback` |
| `ai.tenant_slug`        | string | request `tenant_slug` — required for cost attribution                             |
| `ai.model_name`         | string | `resp.Model`                                                                      |
| `ai.input_tokens`       | int64  | `resp.Usage.InputTokens`                                                          |
| `ai.output_tokens`      | int64  | `resp.Usage.OutputTokens`                                                         |
| `ai.cache_read_tokens`  | int64  | `resp.Usage.CacheReadInputTokens`                                                 |
| `ai.cache_write_tokens` | int64  | `resp.Usage.CacheCreationInputTokens`                                             |

Correlation key = W3C `trace_id`; no dedicated `request_id` span attribute on ai-service (trivial-add follow-up if a downstream consumer asks).

### Kafka

- Consumer-only: subscribes to `grade.updated` + `assignment.created` for cache invalidation (W7.2, W7.3).
- No producer on this service.

### Gateway-config deviations (logged 2026-04-18, routed to gateway-expert #7 cleanup queue)

Three known gaps between `config/gateway-config.yaml` and ai-service proto on HEAD.

1. **CommandPalette route mismatch** — yaml:1086 has `ai.AiService/GetCommandPalette`; proto ships `GetCommandPaletteResults`. Fix yaml (spec name wins).
2. **`/api/ai/feedback` unrouted** — server ships `GetDraftFeedback` gated behind `ai_draft_feedback` flag. Add route + gateway-layer flag short-circuit.
3. **Proto package prefix** — proto is `package slate.ai.v1;`; yaml uses `ai.AiService/`. Likely OK if dispatch strips vendored prefixes; runtime `curl` confirm required. If broken, change proto to `package ai` + regen (ai-expert consented).

---

## user.UserService — ResolveUsername extension (**RATIFIED**)

Defined: 2026-04-18 by po-analyst (from social-expert blocker, 2026-04-18)
**Promoted provisional → ratified 2026-04-18** on auth-tenant-expert's #14 W14 acceptance (PO HEAD-verified).
Consumers: discussion-service (W5 #16, @mention resolution — can flip stubbed `UserResolver` to real gRPC client), potentially email-service (W15 #15, @mention in messages).
Status: **✅ ratified**. HEAD: `internal/grpc/user_handler.go:67` + `internal/models/user.go` + `internal/service/username_resolver.go` + tests. Shape confirmed batched, unknown-omit, case-insensitive-unique.

### Problem

`proto/user.proto` has no `username` field and no `ResolveUsername` RPC — only `GetUser(id)` with email/first_name/last_name. discussion-service CreatePost parses `@handle` mentions and needs a batched username→user_id lookup. social-expert currently has a stubbed `UserResolver` interface returning empty IDs; mentions aren't materialized until this lands.

### Proposed additions to `user.UserService`

```proto
rpc ResolveUsername(ResolveUsernameRequest) returns (ResolveUsernameResponse);
// Batched. Called by discussion-service CreatePost and email-service broadcast.
// Unknown usernames omitted from response (not treated as error).

message ResolveUsernameRequest {
  repeated string usernames = 1;
  string tenant_id = 2;  // lookup scoped to tenant
}
message ResolveUsernameResponse {
  map<string, string> user_ids_by_username = 1;  // username → user_id
}
```

### Required `User` field addition

```proto
message User {
  // ... existing fields ...
  string username = N;  // required; unique per tenant; case-insensitive match
}
```

### Lifecycle semantics

- `username` is set at registration; immutable thereafter for MVP (can relax later).
- Collision within a tenant → registration fails with `ALREADY_EXISTS`.
- Case-insensitive storage (lowercase normalised); case-insensitive lookup.
- Convention for seed (`EXECUTION.md` mentions `@prof.martinez`, `student02`): username = email local-part with `.` preserved, lowercase. Allow override at registration.

### Migration

- Add `users.username VARCHAR(64) NOT NULL` with UNIQUE (tenant_id, LOWER(username)) index.
- Backfill: populate from email local-part for existing rows.

### Interim behaviour (until auth-tenant-expert ratifies + ships)

- discussion-service's `UserResolver` returns empty user-id map — mentions parsed but not materialized; notification fan-out to the inbox is a no-op until the resolver is wired for real.
- This is NOT an acceptable end-state for Wave 5 seed (Eastfield seed includes a `@prof.martinez` mention populating student02's inbox — seed asserts inbox non-empty). W14 must land ResolveUsername before Wave 5 runs.

### Ratification hooks

auth-tenant-expert (W14 #14): please confirm on your completion report whether ResolveUsername + `username` field are included in your shipped build. If you've already flipped #14 without them, flag it and spawn a W14.x follow-up. If in-scope but different shape, SendMessage po-analyst with a diff.

---

## scheduling.SchedulingService (W6) — ratified

Defined: 2026-04-18 by po-analyst (from scheduling-expert #17 completion proof, PO HEAD-verified).
Consumers: api-gateway (`/api/scheduling/*`), student `/office-hours`, provider `/office-hours` + `/teach` instructor-day view.
Status: **✅ ratified**. scheduling-expert build green: `go build`/`go test -count=1 ./... (17 tests)`/`go vet`/`protoc` all exit 0.

gRPC tenant service on `:50063`.

### RPCs (all 6 per docs/plan.md W6.1)

```proto
rpc CreateSchedule      (CreateScheduleRequest)      returns (Schedule);
rpc ListSchedules       (ListSchedulesRequest)       returns (ListSchedulesResponse);
rpc ListAvailableSlots  (ListAvailableSlotsRequest)  returns (ListSlotsResponse);
rpc BookSlot            (BookSlotRequest)            returns (Booking);
rpc CancelBooking       (CancelBookingRequest)       returns (google.protobuf.Empty);
rpc GetInstructorDay    (GetInstructorDayRequest)    returns (GetInstructorDayResponse);
```

HTTP mapping (gateway config at `config/gateway-config.yaml:1040-1066`):

- `GET    /api/scheduling/slots` → ListAvailableSlots
- `POST   /api/scheduling/bookings` → BookSlot
- `DELETE /api/scheduling/bookings/:id` → CancelBooking
- `POST   /api/scheduling/schedules` → CreateSchedule
- `GET    /api/scheduling/schedules` → ListSchedules
- `GET    /api/scheduling/instructor-day` → GetInstructorDay

scheduling-expert realigned gateway yaml themselves (prior yaml had stale method names `ListSlots`/`ListBookings`/`CreateBooking`/`HealthCheck` — removed/renamed to match proto). HealthCheck uses standard `grpc.health.v1.Health/Check`.

### Ratified additive

- **`ListSlotsResponse.from_cache bool`** — observability signal so callers can distinguish fresh vs. cached results. Defaults `false`; zero-value-backward-compat if ever removed.
- **`BookSlotRequest.pre_context string` (per plan W6.4)** — server persists as `oh_bookings.questions` column (legacy name). Response echoes as `questions`. Note the caller-side vs. server-side name divergence; kept because the column already existed pre-W6 and renaming the column would churn the repo.
- **`Schedule.is_active bool` exposed unconditionally** — needed so callers passing `active_only=false` can see state. Non-breaking.
- **DB-level partial UNIQUE index `oh_bookings_unique_confirmed_slot ON (instructor_id, slot_date, slot_start_time) WHERE status='confirmed'`** — beyond W6.2 migration spec. Rationale: atomic double-book rejection, PG error 23505 maps to `codes.FailedPrecondition`. Clean engineering; ratified.

### Trace propagation

- Inbound extract via `libs/common-go/grpc.TracingUnaryInterceptor` + custom `CorrelationUnaryInterceptor` that generates `x-request-id` when absent and echoes on response.
- Span tag `request.id` + `tenant.slug` via `commontracing.TagSpanWithCorrelation` on every handler.
- Tenant.slug fallback from `config.TenantConfig.Slug` when metadata lacks `x-tenant-slug` — correct given per-tenant container invariant.
- Outbound: service is read-only toward Postgres + Redis; no outbound gRPC/Kafka in W6 scope. Scaffolding ready for future AI/grading integrations.

### Kafka

- Consumer: none in W6.
- Producer: none in W6.

---

## discussion.DiscussionService (W5) — ratified

Defined: 2026-04-18 by po-analyst (from social-expert #16 completion proof, PO HEAD-verified earlier this session + contract detail now provided).
Consumers: api-gateway (`/api/discussions/*`, `/api/health/discussions`), student `/inbox` `/discussions`, provider `/discussion`.
Status: **✅ ratified**. social-expert build green: `go build`/`go test -count=1 ./... (9 funcs / 19 cases)`/`go vet`/`protoc` all exit 0.

gRPC tenant service on `:50056`, proto `/discussion.DiscussionService/...`.

### RPCs (all 8 shipped + gateway-aligned)

```proto
rpc CreateThread          (CreateThreadRequest)          returns (Thread);
rpc ListThreads           (ListThreadsRequest)           returns (ListThreadsResponse);
rpc GetThread             (GetThreadRequest)             returns (ThreadWithPosts);
rpc CreatePost            (CreatePostRequest)            returns (Post);
rpc GetInbox              (GetInboxRequest)              returns (GetInboxResponse);
rpc MarkInboxRead         (MarkInboxReadRequest)         returns (MarkInboxReadResponse);
rpc GetThreadsNeedingReply(GetThreadsNeedingReplyRequest) returns (GetThreadsNeedingReplyResponse);
rpc HealthCheck           (HealthCheckRequest)           returns (HealthCheckResponse);
```

HTTP mapping per `config/gateway-config.yaml:954-992`:

- `POST   /api/discussions/threads` → CreateThread
- `GET    /api/discussions/threads` → ListThreads
- `GET    /api/discussions/threads/:id` → GetThread
- `POST   /api/discussions/threads/:id/posts` → CreatePost
- `GET    /api/discussions/inbox` → GetInbox
- `PATCH  /api/discussions/inbox/:id/read` → MarkInboxRead
- `GET    /api/discussions/needing-reply` → GetThreadsNeedingReply
- `GET    /api/health/discussions` → HealthCheck

### Ratified deviations (all accepted)

- **Method renames (plan W5.1 vs gateway):** `MarkRead` → `MarkInboxRead`, `ListThreadsNeedingReply` → `GetThreadsNeedingReply`. Gateway shape wins (integration contract); proto comments cite both names.
- **`HealthCheck` RPC added.** Not in docs/plan.md W5.1; gateway expected it. Returns `{status: "ok"}`.
- **Additive proto fields:** `CreateThreadRequest.initial_post` (convenience — co-create first post atomically), `Post.mentioned_user_ids` (UI renders mention chips without a second call), `GetInboxRequest.unseen_only` (filter), `Thread.tenant_id` (write-path only, omitted from returned proto).
- **Additive tables:**
  - `instructor_courses (tenant_id, instructor_id, course_id)` — local projection populated by course-service Kafka roster stream (wire-up follow-up). Avoids synchronous course-service call in `GetThreadsNeedingReply`.
  - `posts.author_role VARCHAR(32)` + partial index on `author_role='instructor'` — query-performance support for the 24h needing-reply boundary.

### Trace propagation

- Inbound extract via `commontracing.ExtractTraceparent` + `TracingUnaryInterceptor("discussion-service")`. Synthesizes `x-request-id` when missing (`req-<hex-unix-nano>`). Falls back to `TENANT_SLUG` env for `x-tenant-slug` (tenant-pod-local invariant).
- Span tag `request_id` + `tenant.slug` via `commontracing.TagSpanWithCorrelation` + domain tags `tenant.id` / `course.id` / `thread.id` / `instructor.id`.
- Outbound Kafka `discussion.mention` → `commontracing.KafkaHeadersFromContext(ctx)` attaches traceparent + x-request-id + x-tenant-slug. Verified by `TestKafkaPublishContextCarriesCorrelation`.
- Outbound gRPC: none currently (UserResolver is stubbed; flips to real when user-auth `ResolveUsername` wires in — now ratified, flip path open).

### Kafka

- Producer: `discussion.mention` (on CreatePost when mentions present). Headers propagate.
- Consumer: none in W5.

### UserResolver integration note

Current production binding is `NoopResolver{}` — returns empty map → mentions silently no-op at runtime. Tests use `StaticResolver`. **With `user.UserService.ResolveUsername` now ratified + HEAD-live (auth-tenant-expert #14 ✅), social-expert can flip `NoopResolver` → real gRPC client in a follow-up micro-commit.** Wave 5 seed's `@prof.martinez` → student02 inbox assertion requires this flip.

---

## email.MessagingService (W15) — ratified

Defined: 2026-04-18 by po-analyst (from email-expert #15 completion proof, PO HEAD-verified earlier this session + contract detail now provided).
Consumers: api-gateway (`/api/messages/*`, `/api/broadcast`, `/api/inbox`), admin `/broadcast`, student `/inbox`.
Status: **✅ ratified**. email-expert build green: `go build`/`go test -count=1 ./... (60 PASS)`/`go vet`/`protoc` all exit 0.

gRPC tenant service. Proto `email.MessagingService`.

### RPCs (existing + additive)

Existing (unchanged): `SendMessage`, `GetInbox`, `MarkRead`, `GetThread`, `GetUnreadCount`, `HealthCheck`.

**Additive ratified**:

```proto
rpc SendBroadcast(SendBroadcastRequest) returns (SendBroadcastResponse);
message SendBroadcastRequest {
  BroadcastTargets targets = 1;   // all_tenants bool XOR tenant_ids[]
  string message = 2;
  repeated string channels = 3;   // "email" | "in_app"
  string author_id = 4;
}
message BroadcastTargets {
  bool all_tenants = 1;
  repeated string tenant_ids = 2;
}
message SendBroadcastResponse {
  string broadcast_id = 1;
  repeated string tenant_ids = 2;
  int32 emails_sent = 3;
  int32 in_app_sent = 4;
  int64 sent_at = 5;
}
```

HTTP: `POST /api/broadcast` → `email.MessagingService/SendBroadcast` (gateway line 948). Closes pre-existing route→server gap.

### Kafka topics

**Produced** (all with traceparent + x-request-id + x-tenant-slug headers via `KafkaHeadersFromContext`):

- `broadcast.in_app` — ratified additive. One record per tenant in tenant-targeted fan-out; one empty-tenant record for all-tenants fan-out. Payload: `{broadcast_id, author_id, message, tenant_id, timestamp}`.
- `message.sent`, `message.read` — existing; now produce with trace headers.

**Consumed** (context restored via `ContextFromKafkaHeaders` on every handler):

- Existing: `assignment.graded`, `announcement.posted`, `room.created`.
- **New (W15.2)**:
  - `discussion.mention` — payload shape consumed: `{tenant_id, thread_id, thread_title, mentioned_by, mentioned_user, mentioned_email, excerpt}`. **Producer alignment required with social-expert (W5).** social-expert's discussion-service emits on CreatePost-with-mentions; schema flagged here for producer/consumer matching.
  - `incident.opened` — payload shape: `{incident_id, tenant_id, severity, title, summary}`. Already aligned with `incident.IncidentService` ratified contract; empty `tenant_id` records skipped (platform-wide uses broadcast path).
  - `grade.updated` — payload shape consumed: `{tenant_id, student_id, student_email, assignment_title, score, max_score}`, gated by env `EMAIL_GRADE_NOTIFICATIONS=true`. **Producer alignment required with grading-expert (W9).**

### Trace propagation

- gRPC inbound: `commongrpc.TracingUnaryInterceptor("email-service")` extracts traceparent + applies `TagSpanWithCorrelation`.
- Kafka inbound: every consumer handler calls `tracing.ContextFromKafkaHeaders(msg.Headers)` — verified by `TestConsumer_TraceHeadersExtracted`.
- Kafka outbound: `tracing.KafkaHeadersFromContext(ctx)` attached on every `WriteMessages` call.
- HTTP outbound (email relay): `email.HTTPSender.Send` injects via `otel.GetTextMapPropagator().Inject(ctx, propagation.HeaderCarrier(req.Header))` — verified by `TestHTTPSenderInjectsTraceparent`.

### Open follow-ups (non-blocking)

- **HTTP chi middleware for X-Request-ID / tenant.slug extraction** — not added in this PR. Chi router currently relies on the service-level tracing init but has no explicit middleware for the correlation headers. If a cross-service chi-middleware is added (shared pattern), email-service should opt in. Flagged for session cleanup.
- **TenantResolver production binding** — currently `StaticResolver` (clean interface seam, not a silent stub). Flip to a tenant-service gRPC client when W16 tenant surface allows. Same pattern as social-expert's `NoopResolver` flip.

---

## metrics.MetricsService (W12) — ratified

Defined: 2026-04-18 by po-analyst (from metrics-expert #12 completion proof, PO HEAD-verified earlier).
Consumers: api-gateway (`/api/metrics/*`, `/api/roster/*`), admin `/ops` + `/schools/[id]` dashboards, provider `/roster` + `/analytics`, incident-service (`metrics.threshold_breached` Kafka upstream — already ratified on incident side).
Status: **✅ ratified**. metrics-expert build green: `go build`/`go test ./... (48 PASS)`/`go vet` all exit 0. REST-only service, no proto.

HTTP service on `:50057` (chi router). No `.proto` file — gRPC server runs health + reflection only.

### Endpoint surface

W12 new routes:

- `GET  /roster/:courseId/health` — RosterHealth per W12.1 shape
- `POST /roster/:courseId/invalidate` — additive, manual cache-bust (ratified — see note)
- `POST /roster/nudge` — pre-existing
- `GET  /metrics/grades/histogram/:assignmentId` — letter-bucket histogram (ratified coarse A/B/C/D/F)
- `POST /api/metrics/export` — async export job enqueue, emits `metrics.export_requested`
- `GET  /api/metrics/export/:jobId` — additive, job polling (ratified)
- `GET  /metrics/platform/extended` — extended PlatformMetrics (sibling path, ratified)

Pre-existing (unchanged): `/metrics/platform`, `/metrics/tenants/:id`, `/metrics/courses/:id`, `/metrics/students/:id`, `/metrics/students/:id/time-on-task`, `/metrics/grades/distribution/:courseId`, `/metrics/grades/comparison/:studentId/:courseId`.

### Ratified additive / scope decisions

1. **Letter-grade coarseness: A/B/C/D/F (coarse).** Chart-friendly default. If admin-fe `/analytics` needs fine-grain (A+/A/A-) later, add an `?granularity=fine` query param as a backward-compatible extension.
2. **`POST /roster/:courseId/invalidate` additive route** — ratified. Useful for admin manual cache-bust during debugging; cheap to keep.
3. **`GET /api/metrics/export/:jobId` additive polling endpoint** — ratified. Async job flow needs the status poll; the spec's omission was a gap.
4. **`GET /metrics/platform/extended` sibling vs overwrite** — ratified as sibling. Overwriting `/metrics/platform` would break consumers of the lighter base shape; sibling preserves both. docs/plan.md §W12.4 phrasing ambiguous, sibling is lower-risk.
5. **`incident.StubClient`** returning 0 active incidents — acceptable per task protocol. Flip to real `incident.IncidentService` gRPC client when admin-fe `/ops` cares about live counts. Follow-up; not a deviation.

### Trace propagation

- Inbound HTTP: `handlers.TracingMiddleware` (chi middleware) — `ExtractTraceparent`, `WithRequestID` (generates `req-<hex>` if missing), `WithTenantSlug`, echoes `X-Request-ID` on response.
- Span tag: `TagSpanWithCorrelation` on every request.
- gRPC inbound (health + reflection only): `TracingUnaryInterceptor("metrics-service")` + `otelgrpc` stats handler.
- Kafka producer: `KafkaHeadersFromContext(ctx)` on every emit (incl. `metrics.export_requested`).
- Kafka consumer: `ContextFromKafkaHeaders` restores context on `grade.updated` / `submission.graded` handlers (links to producer span).

### Kafka

- Producer: `metrics.export_requested` (on `POST /api/metrics/export`), `metrics.threshold_breached` (already contract-aligned with `incident.IncidentService` consumer).
- Consumer: `grade.updated` — invalidates roster + platform caches. Producer side (grading-expert) schema alignment in progress.

### Gateway-config gaps (logged 2026-04-18, routed to gateway-expert #7 cleanup queue)

Gateway yaml currently routes only the pre-existing REST paths; the 4 new W12 endpoints are unrouted. Adding to the cleanup bundle:

- `POST /api/metrics/export` (→ metrics-service HTTP)
- `GET  /api/metrics/export/:jobId` (→ metrics-service HTTP)
- `GET  /api/metrics/grades/histogram/:assignmentId` (→ metrics-service HTTP)
- `GET  /api/metrics/platform/extended` (→ metrics-service HTTP)

### Ratified gRPC wrapper additions (2026-04-18, P0 metrics-expert #30 ✅)

Task #30 shipped a gRPC Server wrapping the existing REST + repo layer. Live admin `/ops` incident unblocked. Contract condition (isomorphic REST JSON ↔ proto shapes) enforced by test pins:

- `TestDistToProto_IsomorphicWithRESTShape` — grade-distribution
- `TestJobToProto_ShapeParity` — export job
- `TestGetRosterHealth_gRPC_FetcherIntegration` — roster-health incl. sort-by-risk ordering

**10 RPCs registered on `metrics.MetricsService`:**

1. `GetPlatformMetrics`
2. `GetTenantMetrics`
3. `GetStudentProgress`
4. `GetGradeDistribution`
5. `GetStudentComparison`
6. `GetSystemMetrics` — stub reusing PlatformMetrics counts (full system surface lives in Prometheus; out of P0 scope)
7. `GetActiveAlerts` — empty-list stub (alerts pipeline not yet seeded; admin-fe already handles empty)
8. **`GetRosterHealth`** — additive ratified
9. **`CreateExport` + `GetExport`** — additive ratified (replaces CARRYOVER placeholder for async job status poll)
10. **`GetPlatformMetricsExtended`** — additive ratified

**`GetGradingQueueCount` explicitly NOT on metrics** — belongs on `assignment.GradingService` per proto-audit P2 dispatch. grading-expert owns it in Task #31.

**Go-side cosmetic note (non-breaking):** proto field `signups_last_30d` → Go identifier `SignupsLast_30D` (protoc-gen-go leading-digit rule). JSON tag preserved as `signups_last_30d`; REST wire compat unchanged; only affects Go code references.

**Wrapper pattern is the template** for other REST-first services (grading #31, course #32 in flight) — thin delegate around existing service + repo layer, zero behavior duplication, isomorphic shapes pinned by tests.

Pattern: same REST-passthrough approach as the course-service REST routes (option (a) already routed to gateway-expert earlier this session).

---

## assignment.GradingService + assignment.SubmissionService (P2 / W9) — ratified

Defined: 2026-04-22 (grading-expert re-close of Task #31 with REST-parity patch).
Status: **✅ ratified additively**. assignment-grading-service build green: `go build/test/vet` exit 0 + 83 tests pass; `docker compose build assignment-grading-service` exit 0.

5 RPCs exposed on gRPC `:50055`, wired via gateway `config/gateway-config.yaml` lines 646-664 + 1284:

| RPC                    | Service                        | Delegates to                                                | REST parity (JSON shape)                                 |
| ---------------------- | ------------------------------ | ----------------------------------------------------------- | -------------------------------------------------------- |
| `GetGradingQueue`      | `assignment.GradingService`    | `GradingQueueService.GetGradingQueue`                       | `{assignment_id, patterns[], total_pending}`             |
| `GetGradingQueueCount` | `assignment.GradingService`    | `GradingQueueService.GetGradingQueue` (sums `Count`)        | `{assignment_id, pending, patterns}`                     |
| `BatchPublishGrades`   | `assignment.GradingService`    | `BatchGradingService.Apply` (pattern-mode) / legacy publish | `{pattern_id, assignment_id, graded_count, grade_ids[]}` |
| `SaveDraft`            | `assignment.SubmissionService` | `SubmissionService.UpsertDraft`                             | `Submission`                                             |
| `GetDraft`             | `assignment.SubmissionService` | `SubmissionService.GetDraft`                                | `Submission`                                             |

### Additive proto fields — ratified (back-compat preserved)

- **`GetGradingQueueResponse`**: added `assignment_id`, `patterns[] (PatternGroup)`, `total_pending`; legacy `groups[]` kept as alias populated identically to `patterns`.
- **`GetGradingQueueCountResponse`**: added `assignment_id`, `pending`, `patterns` (int32 count); legacy `count` kept as alias of `pending`.
- **`BatchPublishGradesRequest`**: added `pattern_id`, `assignment_id`, `submission_ids[]`, `rubric_scores[] (RubricScoreInput)`, `feedback_template`, `graded_by`, `instructor_id`; legacy `grade_ids[]` preserved for publish-only callers. **Dispatch rule:** if `pattern_id != "" || rubric_scores not empty` → `BatchGradingService.Apply` (pattern-mode, emits `grade.updated` per student); else if `grade_ids` present → loop `PublishGrade(id)` (legacy-mode back-compat).
- **`BatchPublishGradesResponse`**: added `pattern_id`, `assignment_id`, `graded_count`, `grade_ids[]`; legacy `published_count` (alias of `graded_count`) + `failed_ids[]` preserved.
- **New message `RubricScoreInput { row_id, points }`** — mirrors `service.RubricScoreInput` in Go code.

### Kafka contract

`grade.updated` emission path is **unchanged** — pattern-mode gRPC BatchPublishGrades delegates to the same `batch_grading_service.go::Apply` used by REST `POST /grades/batch`. One execution + one Kafka contract shared between gateway-transcoded gRPC callers and direct-REST callers. No new event shapes or topics.

### Trace propagation

All 5 RPCs inherit `otelgrpc.NewServerHandler()` + `tracing.TracingUnaryInterceptor` + `tracing.LoggingUnaryInterceptor` at `cmd/server/main.go:228-234`. Outbound `grade.updated` events inherit traceparent/x-request-id/x-tenant-slug injection from `pkg/kafka/producer.go::PublishEvent` (W9.5 wiring, unchanged).

### Precedent

Same "additive + legacy aliased" pattern as metrics-service P0 ratification above. Legacy callers (any that existed against the pre-patch proto signatures) continue to work; new callers get REST-parity shapes.

---

## content.ContentManagementService (W10) — ratified

Defined: 2026-04-18 (session close-out). Consumers: student `/modules/[id]` VideoPlayer, metrics-service (`content.position_updated` Kafka consumer).
Status: **✅ ratified**. content-expert build green: `cargo build --release`/`cargo clippy`/11 W10-specific tests exit 0.

REST service on axum :8082 (per-tenant). No `.proto` changes.

### Additive endpoints + schema

- **`PUT /content/:id/position`** (W10.1) — body `{position_seconds: int}` → response `{position_seconds: int}`. Sync Redis SETEX + async DB upsert.
- **`GET /content/:id`** — adds optional `resume_position_seconds` field to `ContentMetadataResponse` (serde skip_serializing_if None).
- **DB migration `20250418000001_create_video_positions.sql`** — new `video_positions(user_id UUID, content_id UUID, position_seconds INT, updated_at TIMESTAMPTZ)`. Disjoint from pre-existing `progress_tracking.last_position_seconds` (separate entity: content_objects MinIO uploads vs transcoded-lesson resources).
- **Redis key `tenant:{slug}:video_pos:{user_id}:{content_id}` SETEX TTL 1800s** — Redis-first hydration with DB fallback in `GET /content/:id`.
- **TTL by content-type** (W10.3): `fn get_url_ttl(content_type) -> Duration`. `video/*` → 900s, else → 3600s. Replaces 3 hard-coded 3600s call-sites in content_handlers.rs.
- **Range support** (W10.2): S3/MinIO presigned GET URLs honour `Range: bytes=…` natively; verified by `tests/range_request_test.rs` (ignored until MinIO test env). HTTP 206 + `Accept-Ranges: bytes` + `Content-Range: bytes 100-199/…` emitted by MinIO, not our code path.

### Kafka produced (W10)

- **`content.position_updated`** — emitted on `PUT /content/:id/position`. Payload `{content_id, tenant_id, tenant_slug, user_id, position_seconds, occurred_at_unix_ms}`. Headers: `traceparent` + `x-tenant-slug` via `common_rust::observability::propagation::kafka_headers_for`. **Consumer: metrics-service** (engagement-pct input).

### Trace propagation

- **New W10 addition**: `tower_http::trace::TraceLayer::new_for_http()` wrap on axum REST router in main.rs. Previously untraced; now participates in Tempo trace tree.
- gRPC side pre-existing.
- Outbound Kafka: `build_trace_headers()` helper wires `current_traceparent()` + `x-tenant-slug` on every `WriteMessages` call. Tests pin the header presence.

### Gateway wiring

- `PUT /api/content/:id/position` not yet proxied through gateway yaml; same pattern as course-service REST passthroughs. **Routed to gateway-expert cleanup queue** (see `plan/CARRYOVER.md §1`).
- Coexists with pre-existing `content.StreamingService/UpdatePlaybackPosition` gRPC (different entity — resources table vs content_objects). Two surfaces intentional until frontend consolidation happens.

### Pre-existing baseline exception (documented per standing-expectations addendum)

- 3 sqlx-tokio test failures in `src/upload/handler.rs` (`test_generate_storage_key`, `test_generate_chunk_key`, `test_generate_permanent_key`) — all `#[test]` functions that construct sqlx Pool outside Tokio runtime, panic with "this functionality requires a Tokio context" from sqlx-core-0.7.4.
- **Git-blame evidence**: all 3 tests introduced in commit `c71000b` (NoorUllah Shaik, **2025-11-15**), ~5 months before Wave 0 baseline (2026-04-18).
- **Why accepted**: (a) content-expert disclosed them in completion report, (b) traced to pre-revamp commit with git blame, (c) W10 shipped 11 new passing tests and zero regressions to these 3. This is a narrow exception per the addendum below.
- **Follow-up task** (not W10's ownership): upload-area remediation to swap `#[test]` → `#[tokio::test]` on these 3 functions. Logged in `plan/CARRYOVER.md`.

---

## email.MessagingService — Kafka consumed schema alignment (ratified 2026-04-18, session close)

### `discussion.mention` consumer shape — aligned

social-expert's discussion-service producer (W5 #16) adjusted to email-expert's W15 consumer schema. **Canonical shape:**

```
{
  type:            "discussion.mention",
  tenant_id:       string,
  thread_id:       string,
  thread_title:    string,            // NEW (producer side): taken from the thread row
  mentioned_by:    string,            // renamed from author_id; user_id value
  mentioned_user:  string,            // renamed from mentioned_user_id; user_id value
  mentioned_email: string,            // LEFT EMPTY by producer — see resolution rule below
  excerpt:         string,            // NEW: post content truncated to 280 chars on word boundary
  post_id:         string,            // additive (for metrics-service)
  course_id:       string,            // additive (for metrics-service)
  timestamp:       ISO-8601,
}
```

**`mentioned_email` resolution rule**: producer leaves empty to avoid per-mention extra hop to user-auth during write tx. **Consumer MUST fall back** to `user.UserService/GetUser(mentioned_user)` when empty. email-service already dials user-auth for broadcast delivery, so this adds one cached lookup per event rather than N-per-post on the producer side. Clean architectural split: producer stays fast, consumer owns enrichment.

Kafka key: `mentioned_user_id` (partitioning by recipient). Headers: traceparent + x-request-id + x-tenant-slug via `KafkaHeadersFromContext`.

### `discussion.mention` producer test pinning

`TestMentionParsingAndInboxAndKafka` in discussion-service now asserts `MentionedBy`, `ThreadTitle`, `Excerpt` on every emitted event. Schema drift will surface as a test failure on producer rebuilds.

---

## discussion.DiscussionService (W5) — NoopResolver → GRPCResolver flip (ratified 2026-04-18, session close)

social-expert flipped the production binding from `NoopResolver{}` to `GRPCResolver` against `user.UserService/ResolveUsername` in a micro-commit. HEAD: `services/discussion-service/internal/userauth/{grpc_client.go, grpc_client_test.go, resolver.go}` present.

**Behavior:**

- Production dial: `cmd/server/main.go` creates `grpc.NewClient(cfg.UserAuth.Addr)` with `otelgrpc.NewClientHandler` for outbound traceparent propagation. Wires `userauth.NewGRPCResolver` into `grpcserver.Options.UserResolver`.
- **Dial-fallback intentional**: if `USER_AUTH_GRPC_ADDR` is unreachable at startup, service falls back to `NoopResolver{}` (logs warn) so unit tests + isolated dev bringup + chicken/egg compose ordering don't crash. Per-RPC runtime errors (user-auth up but call fails) still surface as gRPC Internal to caller.
- Re-keys server's lowercase-normalized response back to caller's original casing for display parity.

**Tests added:** `TestGRPCResolverReKeysByOriginalCasing`, `TestGRPCResolverEmptyInputShortCircuits`, `TestGRPCResolverSurfacesServerErrors`. All exit 0 on HEAD.

**Wave 5 seed unblock:** `@prof.martinez` → student02 inbox assertion now resolves end-to-end (no stub). Eastfield seed CTA lights up without any further seed-script changes.

---

## Wave 5 observability (ratified 2026-04-18) — "deliverables + syntactically verified" close

observability-expert shipped all Wave 5 deliverables; live-stack matrix (9 CTA × 16 topic traceparent verification) deferred to first seeded-env spinup.

### Deliverables accepted

- `scripts/assert_trace.sh` — exists, structurally correct, consumes Playwright-side `X-Request-ID` captures from `tests/e2e/test-results/**/x-request-id.txt` (qa-expert wired the handoff).
- Grafana dashboards per portal.
- 16-topic Kafka traceparent verification spec + assertion logic.

### Live-run verification — deferred

**Post-deploy commands** (run once first tenant spins up via seed-eastfield.sh):

```bash
# 1. Bring up the stack
docker compose up -d

# 2. Run Eastfield seed (provisions tenant, 23 users, seed data)
scripts/seed-eastfield.sh

# 3. Run Playwright suite with trace-ID capture
cd tests/e2e && npx playwright test

# 4. Execute trace-assertion harness over captured X-Request-IDs
scripts/assert_trace.sh
# Expected: ≥3 spans across ≥3 services per captured request ID
```

**Why deferred:** live stack up-and-running is not a Wave 5 code gate; it's a first-deploy readiness check. Captured in `plan/CARRYOVER.md §3` so no one re-discovers these commands.

---

## `grade.updated` Kafka event — canonical schema (ratified 2026-04-18)

Producer: `services/assignment-grading-service` (`pkg/kafka/producer.go::PublishEvent` → Kafka topic `grade.updated`).

### Canonical payload (Event.Data)

```json
{
  "grade_id": "string",
  "assignment_id": "string",
  "assignment_title": "string", // from assignment.Title (already in scope at emission; zero extra DB hops)
  "student_id": "string", // legacy primary — metrics + email consumer read
  "user_id": "string", // alias of student_id — ai-service reads this name
  "student_email": "string", // **always "" from producer** — consumer resolves lazily
  "tenant_id": "string", // legacy; metrics falls back here when tenant_slug empty
  "tenant_slug": "string", // ai + metrics primary tenant key
  "course_id": "string", // ai + metrics required
  "score": "number",
  "max_score": "number", // from assignment.MaxPoints
  "adjusted_score": "number", // after late-policy penalty
  "pattern_id": "string" // batch-grading identifier; additive, ignorable by non-batch consumers
}
```

Outer `Event` envelope unchanged: `{type: "grade.updated", aggregate_id: grade_id, timestamp: rfc3339, data: <above>}`. Kafka headers per trace.propagation: `traceparent` + `x-request-id` + `x-tenant-slug`.

### Consumer parity — verified on HEAD

| Consumer                                                                    | File                                                  | Fields read                                                                          | Test                                                           |
| --------------------------------------------------------------------------- | ----------------------------------------------------- | ------------------------------------------------------------------------------------ | -------------------------------------------------------------- |
| ai-service grade cache invalidator                                          | `services/ai-service/internal/kafka/consumer.go`      | `tenant_slug`, `user_id`, `course_id`                                                | `TestGradeUpdatedEvent_ConsumerShapeParity/ai-service...`      |
| metrics-service roster-health                                               | `services/metrics-service/internal/kafka/consumer.go` | `tenant_slug`, `tenant_id`, `course_id`                                              | `TestGradeUpdatedEvent_ConsumerShapeParity/metrics-service...` |
| email-service outbound-email gate (opt-in `EMAIL_GRADE_NOTIFICATIONS=true`) | `services/email-service/internal/kafka/consumer.go`   | `tenant_id`, `student_id`, `student_email`, `assignment_title`, `score`, `max_score` | `TestGradeUpdatedEvent_ConsumerShapeParity/email-service...`   |

All three subtests pass. Parity test marshals one emitted event and unmarshals into all three mirrored consumer struct literals — any drift fails the test immediately.

### `student_email` coupling — ratified option (b)

Producer emits `student_email = ""` always. Consumer-side resolution (email-service) calls user-auth `GetUser(student_id)` lazily, **only when the opt-in email gate fires**.

**Rationale:** batch grading processes 10–20 submissions per call; populating email producer-side would serialize N user-auth gRPC hops in the HTTP handler before Kafka emit. Gate is opt-in + default-off, so producer shouldn't pay N hops for a feature most tenants don't enable. email-service already has `tenants.Resolver` for `incident.opened`; adding `users.Resolver` is symmetric. Consumer gate already falls through on empty email.

**Follow-up for email-expert (non-blocking, post-Wave-5):** swap `event.StudentEmail == ""` fallthrough at `consumer.go:322` for `users.Resolver.GetEmail(event.StudentID)`. Until then, grade emails silently no-op on the gate.

---

## Standing-expectation addendum — FE Docker-build gate (2026-04-18)

**Addition to FE completion checklist.** Tasks #20 / #21 / #22 (and any future Wave-4-style FE work) are not complete until the portal's Docker build exits 0 on HEAD:

```bash
cd /Users/noorullahshaik/Code/slate/frontend
docker build -f admin/Dockerfile    -t slate-admin-test    .
docker build -f provider/Dockerfile -t slate-provider-test .
docker build -f student/Dockerfile  -t slate-student-test  .
```

Same strict bar as backend agents' `go build` / `cargo build`. Docker build = production shape; if it's red, the deploy is broken, regardless of whether `tsc` / `lint` / `npm run build` all pass.

**Completion report format** (add alongside tsc/lint/npm-build lines):

```
- docker build -f <portal>/Dockerfile -t slate-<portal>-test frontend/ → exit 0 (both stages green)
```

**Precedent (2026-04-18, post-close hygiene):** user reported CI Docker builds failing. PO verified admin + provider build exit 0 locally; student not locally tested. Direct resends dispatched to admin-fe-expert, provider-fe-expert, student-fe-expert with 30-min response window. Going-forward rule added here so no future FE task closes without Docker parity.

**Not a loophole:** additive to existing FE checklist (tsc + lint + npm-build all still required). Bar-raising, not softening.

---

## Standing-expectation addendum — Backend Docker-build gate (2026-04-22)

**Addition to ALL service completion checklists** (Go + Rust + FE). Local `go build / go test / go vet` or `cargo build --release / cargo test / cargo clippy` exit 0 is **necessary but not sufficient**. Every service task closes only when:

```bash
docker compose build <service-name>    # exits 0
```

**Why (precedent 2026-04-22):** user-auth-service (#14) landed with `go build/test/vet` all green locally, but Dockerfile used single-file build (`go build -o server cmd/server/main.go`) which broke when W14 added sibling `cmd/server/w14_wiring.go` — `docker compose build` failed with `undefined: wireW14Handlers`. Fix landed in commit `8e3260a` (package path `./cmd/server`). Workspace resolution masks Cargo.toml dep gaps the same way.

**Root cause of gap:** agents tested `cargo build --release` / `go build ./...` against the workspace; Docker builds in a fresh container without workspace resolution, exposing latent Cargo.toml / Dockerfile misconfigurations. Local green ≠ production green.

**Completion report format** (add alongside existing build/test/vet lines):

```
- docker compose build <service> → exit 0
```

**Applies to:** every service Dockerfile in the repo — Go services, Rust services, frontend portals. No exemptions.

**Not a loophole:** additive to existing build-green-baseline (go build/test/vet or cargo build/test/clippy + tsc/lint/npm-build still required). Bar-raising, not softening.

**PO enforcement:** PO must run `docker compose build` end-to-end before final Wave close. Agent claims of "Docker build verified" require the exit-0 line quoted above; missing the line = revert acceptance.

---

## Standing-expectation addendum — narrow pre-existing-baseline exception (2026-04-18)

**Default rule unchanged:** every task merges with `go test/cargo test/npm test` exit 0. "Pre-existing error" is NOT an acceptable excuse.

**Narrow exception** (single pattern, bounded scope):

> Pre-Wave-0 test failures may be excluded from strict build-green **only** if the agent:
> (a) discloses them in their completion report explicitly,
> (b) traces them to a pre-revamp commit with git-blame evidence (commit SHA + date predating Wave 0 = 2026-04-18), and
> (c) ships no new code that makes them worse.

**Precedent applied:** content-expert W10 #10 (3 sqlx-tokio test failures from commit `c71000b`, 2025-11-15). Accepted ✅ under this exception with full disclosure, git-blame, and 11 new W10 tests green + 0 regressions.

**This is an exception, not a loophole.** It does not authorize:

- Claiming "pre-existing" on a test failure introduced during Wave 0 or later.
- Skipping the disclosure + git-blame step.
- Shipping new code adjacent to the failing test (which would fail condition (c)).

Future use requires SendMessage po-analyst with the disclosure + SHA **before** task completion. Silent exclusions will be reverted.

---
