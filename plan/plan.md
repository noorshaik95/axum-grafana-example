# Slate LMS — Full Implementation Plan (April 2026)

> **Purpose**: Step-by-step implementation tasks organized as parallel workstreams
> for subagent execution. Each workstream is self-contained and can be assigned to a
> dedicated agent. Dependencies between workstreams are called out explicitly.
>
> **Reference**: `docs/ARCHITECTURE.md` for design decisions and contracts.
> **Test guide**: `TESTING.md` for credentials and curl examples.

---

## How to use this plan

- Each **workstream** maps to one agent/developer.
- Tasks within a workstream are sequential.
- Workstreams themselves can run in parallel unless a `DEPENDS ON` note is present.
- Each task includes: what to build, which files to touch, and what "done" looks like.

---

## Workstream Index

| # | Workstream | Owner | Status |
|---|---|---|---|
| W1 | Infrastructure: Redis + multi-tenant updates | infra agent | 🔨 |
| W2 | admin-auth-service (new platform service) | auth agent | 🔨 |
| W3 | feature-flag-service (new platform service) | flag agent | 🔨 |
| W4 | incident-service (new platform service) | ops agent | 🔨 |
| W5 | discussion-service (new tenant service) | social agent | 🔨 |
| W6 | scheduling-service (new tenant service) | scheduling agent | 🔨 |
| W7 | ai-service (new tenant service) | ai agent | 🔨 |
| W8 | course-service updates | course agent | 🔨 |
| W9 | assignment-grading-service updates | grading agent | 🔨 |
| W10 | content-management-service updates | content agent | 🔨 |
| W11 | video-conferencing-service updates | video agent | 🔨 |
| W12 | metrics-service updates | metrics agent | 🔨 |
| W13 | onboarding-service updates (SSO + LMS migration) | onboarding agent | 🔨 |
| W14 | user-auth-service updates (SSO + audit log) | auth-tenant agent | 🔨 |
| W15 | email-service updates (broadcast + notifications) | email agent | 🔨 |
| W16 | tenant-service updates (provision 8 services) | tenant agent | 🔨 |
| W17 | api-gateway updates (new routes + Redis token cache) | gateway agent | 🔨 |
| W18 | Admin frontend (Next.js) | admin-fe agent | 🔨 |
| W19 | Instructor/Provider frontend (Next.js) | provider-fe agent | 🔨 |
| W20 | Student frontend (Next.js) | student-fe agent | 🔨 |

---

## W1: Infrastructure — Redis + Multi-tenant Updates

**DEPENDS ON**: nothing  
**Agent type**: infra

### W1.1 — Add Redis to docker-compose.yml

**File**: `docker-compose.yml`

Add a Redis service on `slate-network`:
```yaml
redis:
  image: redis:7-alpine
  container_name: slate-redis
  restart: unless-stopped
  ports:
    - "6379:6379"
  volumes:
    - redis-data:/data
  networks:
    - slate-network
  command: redis-server --appendonly yes
```

Add `redis-data` to the volumes section.

**Done when**: `docker compose up redis` starts successfully; `redis-cli ping` returns PONG.

---

### W1.2 — Add Redis env vars to all services

**File**: `docker-compose.yml`

Add `REDIS_URL=redis://redis:6379` to all platform service environment sections:
- api-gateway
- tenant-service
- metrics-service
- email-service
- onboarding-service
- (new services: admin-auth-service, incident-service, feature-flag-service)

**Done when**: Services can connect to Redis on startup without error.

---

### W1.3 — Add Redis to .env.example

**File**: `.env.example`

```
REDIS_URL=redis://redis:6379
REDIS_MAX_CONNECTIONS=20
```

---

### W1.4 — Create Redis helper package (Go)

**File**: `libs/common-go/redis/client.go`

Create a shared Redis client factory:
```go
package redis

import (
    "github.com/redis/go-redis/v9"
)

func NewClient(url string) (*redis.Client, error) { ... }

// Namespace wraps a client with a key prefix: "tenant:{slug}:"
func NewNamespacedClient(client *redis.Client, prefix string) *NamespacedClient { ... }
```

Include: connection pooling, health check method, helper for `SET`/`GET`/`DEL`/`TTL`.

**Done when**: `go build ./libs/common-go/redis/...` passes.

---

### W1.5 — Update docker-compose.yml: tenant services list

**File**: `docker-compose.yml`

Remove any static tenant service entries (course-service, assignment-service, etc.)
that don't belong in the platform compose file. Only platform services should remain.
Refer to `docs/ARCHITECTURE.md §3` for the definitive list of platform vs tenant services.

**Done when**: `docker compose up -d` starts only platform services + infra. No
hardcoded `eastfield` or tenant-specific containers.

---

## W2: admin-auth-service (New)

**DEPENDS ON**: W1 (Redis available)  
**Agent type**: auth

### W2.1 — Scaffold service

**Directory**: `services/admin-auth-service/`

Fork from `services/user-auth-service/`. Changes:
- Remove `TENANT_ID`, `TENANT_SLUG`, `DB_SCHEMA` env vars.
- Fixed DB schema: `platform_admins` (hardcoded).
- Remove tenant-scoped handlers; keep: Login, Register, ValidateToken, Logout.
- Update `go.mod` module path to `github.com/slate/admin-auth-service`.

**Files to create**:
- `cmd/server/main.go`
- `internal/auth/strategies/` — keep only email+password, remove tenant OAuth
- `internal/repository/admin_repository.go`
- `internal/grpc/admin_handler.go`
- `migrations/001_init.sql` — `platform_admins`, `platform_roles`, `platform_audit`

---

### W2.2 — Add impersonation token endpoint

**File**: `services/admin-auth-service/internal/grpc/impersonation_handler.go`

```proto
// In admin_auth.proto
rpc GenerateImpersonationToken(ImpersonationRequest) returns (ImpersonationResponse);

message ImpersonationRequest {
  string admin_user_id = 1;
  string tenant_id = 2;
  string tenant_slug = 3;
  string target_user_id = 4;
}
message ImpersonationResponse {
  string token = 1;
  string redirect_url = 2;
}
```

Implementation:
- Validate admin has `superadmin` or `support` role.
- Generate short-lived (5m) RSA-signed JWT with `type=impersonation` claim.
- Return redirect URL: `http://{slug}.slate.local/auth/impersonate?token={token}`.
- Emit Kafka event: `audit.impersonation_started`.

**Done when**: Integration test proves the token verifies with the platform public key.

---

### W2.3 — Add audit log table + Kafka emission

**File**: `services/admin-auth-service/migrations/002_audit.sql`

```sql
CREATE TABLE IF NOT EXISTS platform_audit (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  actor_id VARCHAR(36) NOT NULL,
  action VARCHAR(100) NOT NULL,
  target_id VARCHAR(36),
  target_type VARCHAR(50),
  metadata JSONB,
  created_at TIMESTAMPTZ DEFAULT NOW()
);
```

On every admin action (login, impersonation, password reset), insert a row.

---

### W2.4 — Docker image + compose entry

**File**: `services/admin-auth-service/Dockerfile`  
**File**: `docker-compose.yml`

```yaml
admin-auth-service:
  build: ./services/admin-auth-service
  container_name: slate-admin-auth
  environment:
    - DB_DSN=postgres://postgres:postgres@postgres:5432/adminauth?sslmode=disable
    - JWT_SECRET=${ADMIN_JWT_SECRET}
    - PLATFORM_PRIVATE_KEY_PATH=/run/secrets/platform_private_key
    - REDIS_URL=redis://redis:6379
    - KAFKA_BROKERS=kafka:9092
    - GRPC_PORT=50060
    - HTTP_PORT=8090
  networks:
    - slate-network
  depends_on: [postgres, redis, kafka]
```

**Done when**: `POST /api/admin/auth/login` returns a JWT with `aud: platform`.

---

## W3: feature-flag-service (New)

**DEPENDS ON**: W1 (Redis)  
**Agent type**: platform

### W3.1 — Scaffold service

**Directory**: `services/feature-flag-service/`  
**Language**: Go

```
cmd/server/main.go
internal/
  grpc/
    flags_handler.go   — GetFlags(tenant_id, user_id) → []Flag
    admin_handler.go   — CreateFlag, UpdateFlag, DeleteFlag
  repository/
    flags_repository.go
  service/
    evaluator.go       — evaluate flag rules against tenant/user context
proto/
  feature_flag.proto
migrations/
  001_init.sql
Dockerfile
```

**`migrations/001_init.sql`**:
```sql
CREATE TABLE flags (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  key VARCHAR(100) UNIQUE NOT NULL,
  description TEXT,
  enabled BOOLEAN DEFAULT false,
  created_at TIMESTAMPTZ DEFAULT NOW(),
  updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE flag_rules (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  flag_id UUID REFERENCES flags(id) ON DELETE CASCADE,
  rule_type VARCHAR(50) NOT NULL,  -- 'all', 'tenant', 'role', 'percentage', 'user'
  rule_value JSONB NOT NULL,
  created_at TIMESTAMPTZ DEFAULT NOW()
);
```

---

### W3.2 — Flag evaluation logic

**File**: `services/feature-flag-service/internal/service/evaluator.go`

```go
type EvalContext struct {
    TenantID string
    TenantSlug string
    UserID string
    Roles []string
}

func (e *Evaluator) IsEnabled(flagKey string, ctx EvalContext) bool
```

Rule types:
- `all` → always enabled/disabled for everyone
- `tenant` → `rule_value: {"tenant_ids": ["id1","id2"]}`
- `role` → `rule_value: {"roles": ["instructor"]}`
- `percentage` → `rule_value: {"percent": 50}` — deterministic hash of user_id
- `user` → `rule_value: {"user_ids": ["uid1"]}`

---

### W3.3 — Redis caching

Cache evaluated flags per tenant: `platform:flags:{tenant_id}` (TTL: 5m, JSON array).
On `UpdateFlag` or `DeleteFlag`, invalidate the cache for affected tenants.

---

### W3.4 — Docker + compose entry

Same pattern as W2.4. Port: 50062 (gRPC), 8092 (HTTP).

**Done when**: `GetFlags` returns correct flags for a given tenant context.

---

## W4: incident-service (New)

**DEPENDS ON**: W1 (Redis)  
**Agent type**: platform

### W4.1 — Scaffold + migrations

**Directory**: `services/incident-service/`  
**Language**: Go

```sql
-- migrations/001_init.sql
CREATE TABLE incidents (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  title VARCHAR(255) NOT NULL,
  description TEXT,
  priority VARCHAR(5) NOT NULL CHECK (priority IN ('P0','P1','P2','P3','P4')),
  status VARCHAR(20) DEFAULT 'open' CHECK (status IN ('open','watching','resolved')),
  tenant_id UUID,
  created_by UUID,
  resolved_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ DEFAULT NOW(),
  updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE incident_events (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  incident_id UUID REFERENCES incidents(id) ON DELETE CASCADE,
  actor_id UUID,
  event_type VARCHAR(50) NOT NULL,  -- 'comment','status_change','escalation'
  content TEXT,
  created_at TIMESTAMPTZ DEFAULT NOW()
);
```

---

### W4.2 — gRPC handlers

```proto
service IncidentService {
  rpc CreateIncident(CreateIncidentRequest) returns (Incident);
  rpc UpdateIncident(UpdateIncidentRequest) returns (Incident);
  rpc ListIncidents(ListIncidentsRequest) returns (ListIncidentsResponse);
  rpc GetIncident(GetIncidentRequest) returns (Incident);
  rpc AddEvent(AddEventRequest) returns (IncidentEvent);
  rpc GetServiceStatus(Empty) returns (ServiceStatusResponse);
}
```

`GetServiceStatus` aggregates open incidents → returns per-service health (green/amber/red).

---

### W4.3 — Kafka consumer

Consume `metrics.threshold_breached` events → auto-open a P1 incident if one doesn't
already exist for that service+tenant combination.

---

### W4.4 — Redis cache for active incidents

Cache `platform:incidents:active` (TTL: 30s). Written on every status change.
Used by `/ops` and `/status` admin pages for fast reads.

---

### W4.5 — Docker + compose entry

Port: 50061 (gRPC), 8091 (HTTP).

---

## W5: discussion-service (New Tenant Service)

**DEPENDS ON**: W1, W16 (tenant provisioner updated)  
**Agent type**: social

### W5.1 — Scaffold + proto

**Directory**: `services/discussion-service/`  
**Language**: Go

```proto
service DiscussionService {
  rpc CreateThread(CreateThreadRequest) returns (Thread);
  rpc ListThreads(ListThreadsRequest) returns (ListThreadsResponse);
  rpc GetThread(GetThreadRequest) returns (ThreadWithPosts);
  rpc CreatePost(CreatePostRequest) returns (Post);
  rpc GetInbox(GetInboxRequest) returns (InboxResponse);
  rpc MarkRead(MarkReadRequest) returns (Empty);
  rpc ListThreadsNeedingReply(ListNeedingReplyRequest) returns (ListThreadsResponse);
}
```

---

### W5.2 — Database migrations

```sql
CREATE TABLE threads (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  course_id UUID NOT NULL,
  title VARCHAR(500),
  created_by UUID NOT NULL,
  created_at TIMESTAMPTZ DEFAULT NOW(),
  last_activity_at TIMESTAMPTZ DEFAULT NOW(),
  reply_count INT DEFAULT 0
);

CREATE TABLE posts (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  thread_id UUID REFERENCES threads(id) ON DELETE CASCADE,
  author_id UUID NOT NULL,
  content TEXT NOT NULL,
  parent_post_id UUID,  -- for nested replies
  created_at TIMESTAMPTZ DEFAULT NOW(),
  edited_at TIMESTAMPTZ
);

CREATE TABLE mentions (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  post_id UUID REFERENCES posts(id) ON DELETE CASCADE,
  mentioned_user_id UUID NOT NULL,
  seen_at TIMESTAMPTZ
);

CREATE TABLE inbox_items (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id UUID NOT NULL,
  type VARCHAR(50) NOT NULL,  -- 'mention','reply','feedback','announcement'
  reference_id UUID NOT NULL,  -- thread_id or post_id
  seen_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ DEFAULT NOW()
);
```

---

### W5.3 — Mention parsing + inbox population

On `CreatePost`:
1. Parse content for `@username` mentions.
2. Resolve usernames to user IDs via user-auth-service gRPC call.
3. Insert rows into `mentions` table.
4. Insert rows into `inbox_items` for each mentioned user.
5. Emit Kafka event `discussion.mention` → email-service sends notification.

---

### W5.4 — `ListThreadsNeedingReply` for instructors

Query threads in courses taught by the requesting instructor where:
- No instructor post in the last 24h.
- Thread has unread student posts.
Sorted by time since last activity (oldest first).

---

### W5.5 — Dockerfile + update provisioner

See W16.1 for provisioner update. This service must be included in the 8 tenant containers.

---

## W6: scheduling-service (New Tenant Service)

**DEPENDS ON**: W1, W16  
**Agent type**: scheduling

### W6.1 — Scaffold + proto

**Directory**: `services/scheduling-service/`  
**Language**: Go

```proto
service SchedulingService {
  rpc CreateSchedule(CreateScheduleRequest) returns (Schedule);
  rpc ListSchedules(ListSchedulesRequest) returns (ListSchedulesResponse);
  rpc ListAvailableSlots(ListSlotsRequest) returns (ListSlotsResponse);
  rpc BookSlot(BookSlotRequest) returns (Booking);
  rpc CancelBooking(CancelBookingRequest) returns (Empty);
  rpc GetInstructorDay(GetInstructorDayRequest) returns (InstructorDayResponse);
}
```

---

### W6.2 — Database migrations

```sql
CREATE TABLE oh_schedules (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  instructor_id UUID NOT NULL,
  day_of_week INT NOT NULL,  -- 0=Sunday ... 6=Saturday
  start_time TIME NOT NULL,
  end_time TIME NOT NULL,
  slot_duration_minutes INT DEFAULT 15,
  format VARCHAR(20) DEFAULT 'online',  -- 'online','in_person'
  location TEXT,
  is_active BOOLEAN DEFAULT true,
  created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE oh_bookings (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  schedule_id UUID REFERENCES oh_schedules(id),
  instructor_id UUID NOT NULL,
  student_id UUID NOT NULL,
  slot_date DATE NOT NULL,
  slot_start_time TIME NOT NULL,
  questions TEXT,
  status VARCHAR(20) DEFAULT 'confirmed',
  created_at TIMESTAMPTZ DEFAULT NOW()
);
```

---

### W6.3 — Slot availability algorithm

`ListAvailableSlots(instructor_id, date_from, date_to)`:
1. Fetch all active schedules for instructor.
2. Generate concrete slots for each day in range.
3. Subtract already-booked slots from `oh_bookings`.
4. Cache result in Redis `tenant:{slug}:oh_slots:{instructor_id}` (TTL: 5m).
5. Invalidate cache on `BookSlot` or `CancelBooking`.

---

### W6.4 — Pre-fill questions from assignment context

`BookSlot` accepts optional `pre_context` field (student's current assignment +
question draft from the frontend). Stored as `questions` on the booking.

On `GetInstructorDay`, return bookings with `questions` pre-populated so instructors
can read before the appointment.

---

## W7: ai-service (New Tenant Service)

**DEPENDS ON**: W1, W16  
**Agent type**: ai

### W7.1 — Scaffold + proto

**Directory**: `services/ai-service/`  
**Language**: Go

```proto
service AiService {
  rpc GetCommandPaletteResults(CmdPaletteRequest) returns (CmdPaletteResponse);
  rpc GetGradeProjection(GradeProjectionRequest) returns (GradeProjectionResponse);
  rpc GenerateStudyPlan(StudyPlanRequest) returns (StudyPlanResponse);
  rpc GetDraftFeedback(DraftFeedbackRequest) returns (DraftFeedbackResponse);
  rpc GetWelcomeMessage(WelcomeMessageRequest) returns (WelcomeMessageResponse);
}
```

**Dependencies**:
- `ANTHROPIC_API_KEY` env var (shared platform key, injected at provisioning).
- Uses `claude-sonnet-4-6` by default; `claude-opus-4-7` for complex study plans.

---

### W7.2 — Grade projection

**File**: `services/ai-service/internal/service/grade_projection.go`

Input: current grades, remaining assignments (max points + due date), target grade.

Algorithm (no AI needed for basic version):
1. Compute current weighted average.
2. For remaining assignments, solve for needed score to hit target.
3. Return: current grade, projected grade, "path to A" (required scores per assignment).

Cache in Redis `tenant:{slug}:grades:{user_id}:{course_id}` (TTL: 10m).
Invalidate on Kafka event `grade.updated`.

---

### W7.3 — Study plan generation (Claude API)

**File**: `services/ai-service/internal/service/study_plan.go`

Input: list of assignments (title, due_date, estimated_hours), current date.

Prompt to Claude:
```
You are helping a student at {university_name} plan their study schedule.
Given the following assignments due in the next 16 weeks, generate a
week-by-week study plan. Group related work, account for due dates,
and spread load evenly. Return JSON: [{week, tasks: [{day, duration_minutes, title}]}]

Assignments: {json}
```

Cache result in Redis `tenant:{slug}:study_plan:{user_id}` (TTL: 24h).
Regenerate when a new assignment is added (Kafka event `assignment.created`).

---

### W7.4 — NL command palette

**File**: `services/ai-service/internal/service/cmd_palette.go`

Input: free-text query, user context (role, courses enrolled).

Prompt to Claude:
```
You are a navigation assistant for an LMS. The user typed: "{query}".
The user is a {role} in courses: {courses}.
Return up to 5 action suggestions as JSON:
[{"label": "...", "route": "/...", "icon": "..."}]
Only suggest routes that exist: {route_list}
```

Cache in Redis `tenant:{slug}:cmd_palette:{user_id}:{query_hash}` (TTL: 30s).

---

### W7.5 — Token budget tracking

**File**: `services/ai-service/internal/service/budget.go`

Before each Claude API call:
1. `INCR tenant:{slug}:ai:tokens_used:{YYYY-MM}` in Redis.
2. If value exceeds `TENANT_AI_TOKEN_BUDGET` env var → return error (graceful degradation).

Reset: monthly via cron or TTL on the Redis key.

---

### W7.6 — AI draft feedback

**File**: `services/ai-service/internal/service/draft_feedback.go`

Input: submission content, assignment rubric rows.

Prompt to Claude: evaluate submission against each rubric row, return feedback per row.
Only enabled if `ai_draft_feedback` feature flag is on for the tenant.

---

## W8: course-service Updates

**DEPENDS ON**: W1  
**Agent type**: course

### W8.1 — Module/lesson data model

**Files**: `services/course-service/src/` (NestJS)

Add entities:
```typescript
// module.entity.ts
@Entity()
export class CourseModule {
  @PrimaryGeneratedColumn('uuid') id: string;
  @ManyToOne(() => Course) course: Course;
  @Column() week_number: number;
  @Column() title: string;
  @Column({ default: 'draft' }) status: string; // 'draft','live','scheduled'
  @Column({ nullable: true }) scheduled_at: Date;
  @Column({ default: 0 }) sort_order: number;
}

// lesson.entity.ts  
@Entity()
export class Lesson {
  @PrimaryGeneratedColumn('uuid') id: string;
  @ManyToOne(() => CourseModule) module: CourseModule;
  @Column() type: string;  // 'video','reading','quiz'
  @Column() title: string;
  @Column({ nullable: true }) content_id: string;  // ref to content-service
  @Column({ nullable: true }) duration_minutes: number;
  @Column({ default: 0 }) sort_order: number;
}

// student_progress.entity.ts
@Entity()
export class StudentProgress {
  @PrimaryGeneratedColumn('uuid') id: string;
  @Column() user_id: string;
  @ManyToOne(() => Lesson) lesson: Lesson;
  @Column({ default: false }) completed: boolean;
  @Column({ nullable: true }) completed_at: Date;
}
```

Add REST endpoints:
- `GET /courses/:id/modules` — list modules with lessons
- `POST /courses/:id/modules` — create module (instructor)
- `PATCH /courses/:id/modules/:moduleId` — update (reorder, status)
- `POST /courses/:id/modules/:moduleId/publish`

---

### W8.2 — Live lecture session

**File**: `services/course-service/src/lectures/lecture.service.ts`

```typescript
async startLecture(courseId: string, instructorId: string): Promise<LectureSession>
async endLecture(sessionId: string): Promise<void>
async getAttendance(sessionId: string): Promise<AttendanceStats>
```

Attendance tracked in Redis: `tenant:{slug}:live:{sessionId}` as a sorted set of
user IDs with join timestamp. TTL: 4h.

Emit Kafka event `course.lecture_started` → video-conferencing-service picks this up
to create the video room.

---

### W8.3 — Course analytics

**File**: `services/course-service/src/analytics/analytics.service.ts`

```typescript
async getCourseAnalytics(courseId: string): Promise<CourseAnalytics>
// Returns: engagement%, completion%, median_grade, at_risk_count
```

Compute from:
- `student_progress` table → engagement + completion.
- Kafka consumer `grade.updated` → update running median in Redis.

Cache in Redis `tenant:{slug}:analytics:{course_id}` (TTL: 5m).

---

### W8.4 — Next-up module for student

Add `GET /courses/:id/next-up/:userId` endpoint:
- Returns the first incomplete lesson for the student in this course.
- Used by the course detail page to show "Resume at 4:22 →".

---

## W9: assignment-grading-service Updates

**DEPENDS ON**: W1  
**Agent type**: grading

### W9.1 — Rubric rows

**File**: `services/assignment-grading-service/internal/models/rubric.go`

```go
type RubricRow struct {
    ID           string
    AssignmentID string
    Title        string
    MaxPoints    int
    SortOrder    int
}
```

DB migration: add `rubric_rows` table.

gRPC: add `CreateRubricRow`, `ListRubricRows`, `UpdateRubricRow` to AssignmentService proto.

---

### W9.2 — Draft submissions

**File**: `services/assignment-grading-service/internal/handlers/submission_handler.go`

Add `is_draft bool` to the Submission model.
Draft submissions do not count toward due date enforcement.
`PATCH /assignments/{id}/submissions/draft` — upsert draft content for the current user.
`GET /assignments/{id}/submissions/draft` — return current draft.

---

### W9.3 — Auto-test integration

**Files**:
- `services/assignment-grading-service/internal/service/autotester.go`
- Assignment creation: accept `tests_file` upload → store in MinIO.
- On submission: trigger auto-test job (run tests.py against submission in a
  sandboxed subprocess). Store pass/fail per test in `submission_test_results` table.
- Auto-test results surface in grading queue as part of submission metadata.

**Safety**: run tests in a subprocess with memory/CPU limits and 10s timeout.
Never run untrusted student code as the service user — use a restricted execution
context or container-in-container (out of scope for first pass; mock the runner).

---

### W9.4 — Grading queue with pattern grouping

**File**: `services/assignment-grading-service/internal/service/queue.go`

`GetGradingQueue(assignmentId, instructorId)` returns submissions grouped by pattern:
- Pattern grouping algorithm: cluster submissions by test result fingerprint
  (which tests pass/fail). Same fingerprint → same pattern.
- For assignments without auto-tests: group by instructor-defined rubric score
  distribution (assign patterns after grading starts).
- Return: `[{pattern_id, description, count, auto_score_suggestion, submissions[]}]`

---

### W9.5 — Batch grading

**File**: `services/assignment-grading-service/internal/handlers/grade_handler.go`

`POST /grades/batch`:
- Body: `{pattern_id, assignment_id, rubric_scores: [{row_id, points}], feedback_template}`
- Apply the same rubric scores + feedback to all submissions in the pattern.
- Emit `grade.updated` Kafka event for each student.

---

### W9.6 — Starter code / attachments

Store per-assignment files (starter code, instructions PDF) in MinIO.
Expose `GET /assignments/{id}/attachments` → returns signed URLs.

---

## W10: content-management-service Updates

**DEPENDS ON**: W1  
**Agent type**: content

### W10.1 — Video resume position

**File**: `services/content-management-service/src/video_position.rs`

Add endpoint `PUT /api/content/{content_id}/position`:
```json
{ "position_seconds": 270 }
```

Storage: write-through Redis (`tenant:{slug}:video_pos:{user_id}:{content_id}`, TTL: 30m)
and async write to DB table `video_positions (user_id, content_id, position_seconds, updated_at)`.

On `GET /api/content/{content_id}`, include `resume_position_seconds` in the response.

---

### W10.2 — HTTP range request support

**File**: `services/content-management-service/src/handlers/streaming.rs`

Ensure MinIO presigned URLs for video content support byte-range requests.
The frontend video player needs this for seeking.

Verify `Accept-Ranges: bytes` header is returned for video content types.

---

### W10.3 — Signed URL TTL by content type

**File**: `services/content-management-service/src/service/presigned.rs`

```rust
fn get_url_ttl(content_type: &str) -> Duration {
    if content_type.starts_with("video/") {
        Duration::from_secs(900)   // 15 min for video chunks
    } else {
        Duration::from_secs(3600)  // 1h for documents
    }
}
```

---

## W11: video-conferencing-service Updates

**DEPENDS ON**: W1  
**Agent type**: video

### W11.1 — Live lecture mode

**File**: `services/video-conferencing-service/src/grpc/service.rs`

Add to proto:
```proto
rpc StartLecture(StartLectureRequest) returns (LectureSession);
rpc EndLecture(EndLectureRequest) returns (Empty);
rpc JoinLecture(JoinLectureRequest) returns (JoinResponse);
rpc GetLecturePulse(GetLecturePulseRequest) returns (LecturePulse);
```

Attendance tracking: Redis sorted set `tenant:{slug}:live:{session_id}` with
user_id → join_timestamp. Pulse = `ZCARD` / enrolled_count.

Kafka consumer: `course.lecture_started` → initialize session.

---

### W11.2 — Q&A queue

**File**: `services/video-conferencing-service/src/service/qa.rs`

Redis list `tenant:{slug}:live:{session_id}:questions` — each item:
`{id, user_id, text, upvotes, submitted_at}`.

Endpoints:
- `SubmitQuestion(session_id, text)` → push to Redis list.
- `UpvoteQuestion(session_id, question_id)` → `INCR` upvote counter.
- `GetQuestions(session_id)` → return sorted by upvotes desc.

---

### W11.3 — Zoom OAuth integration

**File**: `services/video-conferencing-service/src/zoom/`

- `GET /api/video/zoom/auth` → redirect to Zoom OAuth.
- `GET /api/video/zoom/callback` → exchange code, store tokens in DB.
- `POST /api/video/meetings` → create Zoom meeting via Zoom API, return join URL.
- `POST /api/video/zoom/webhook` → receive Zoom meeting events (join, leave).

Store Zoom access/refresh tokens per instructor in DB (encrypted at rest).

---

### W11.4 — Office hours rooms

`POST /api/video/oh-rooms` → create a video room for an OH slot.
Room has `booking_id` reference from scheduling-service.
Returns join URL (either native WebRTC or Zoom depending on tenant config).

---

## W12: metrics-service Updates

**DEPENDS ON**: W1  
**Agent type**: metrics

### W12.1 — Roster health signals

**File**: `services/metrics-service/internal/service/roster_health.go`

`GetRosterHealth(courseId, instructorId)` returns students sorted by risk:
```go
type RosterEntry struct {
    UserID          string
    DisplayName     string
    RiskLevel       string  // "healthy", "slipping", "at_risk"
    MissedAssignments int
    DaysSinceActive int
    GradeTrend      []float64  // last 5 grades
    SuggestedAction string
}
```

Risk algorithm:
- `at_risk`: 2+ missed assignments OR grade trend down 3+ weeks OR 5+ days inactive.
- `slipping`: 1 missed assignment OR grade trend down 2 weeks.
- `healthy`: otherwise.

Cache in Redis `tenant:{slug}:roster:{course_id}` (TTL: 2m).
Kafka consumer `grade.updated` → invalidate cache.

---

### W12.2 — Grade distributions

**File**: `services/metrics-service/internal/service/grade_dist.go`

`GetGradeDistribution(assignmentId)` → histogram of scores bucketed by letter grade.
Compute from assignment grades table.

---

### W12.3 — Export reports

**File**: `services/metrics-service/internal/service/export.go`

`POST /api/metrics/export`:
- Body: `{type: "gradebook"|"roster"|"platform", course_id?, tenant_id?, format: "csv"|"xlsx"}`
- Run async: generate file, upload to MinIO, return signed URL.
- Use a Kafka event `metrics.export_requested` + background worker pattern.

---

### W12.4 — Platform-wide stats

Extend existing platform stats endpoint:
- Add: MAU per tenant, signups last 30d, uptime (derived from health checks).
- Cache `platform:metrics:platform` in Redis (TTL: 5m).

---

## W13: onboarding-service Updates

**DEPENDS ON**: W14 (user-auth SSO support)  
**Agent type**: onboarding

### W13.1 — SSO config step in Restate workflow

**File**: `services/onboarding-service/src/workflow.rs`

After the 72h approval gate, add a new Restate step: `configure_sso`.

This step:
1. Presents the admin with a form (frontend driven).
2. Waits for `CompleteSSO` signal (human-in-the-loop Restate timer, 7-day timeout).
3. On completion, stores SSO config in DB and passes `sso_provider` + `sso_config` to
   the provisioning step.

---

### W13.2 — Canvas LMS migration workflow

**File**: `services/onboarding-service/src/canvas_migration.rs`

New Restate Virtual Object: `CanvasMigration`:

Steps:
1. `ConnectCanvas(oauth_code)` → exchange code, store tokens.
2. `ImportCourses()` → paginated fetch from Canvas API, write to course-service.
   Resumable: store last imported page in Restate object state.
3. `ReconcileRosters()` → match Canvas users by email to Slate users.
   Unmatched users: create accounts via user-auth-service.
4. `VerifyAndGoLive(admin_sign_off)` → human-in-the-loop pause; admin reviews and
   confirms. On confirmation, emit `migration.completed` Kafka event.

Canvas API client:
- Base URL: `https://canvas.instructure.com/api/v1` or self-hosted Canvas URL.
- Use access token from OAuth step.
- Rate limit: respect `X-Rate-Limit-Remaining` header; back off if needed.

---

## W14: user-auth-service Updates

**DEPENDS ON**: nothing (standalone)  
**Agent type**: auth-tenant

### W14.1 — SSO providers

**File**: `services/user-auth-service/internal/auth/strategies/`

Add strategy implementations:
- `saml_strategy.go` — SAML 2.0 with Shibboleth IdP. Library: `github.com/crewjam/saml`.
- `oidc_strategy.go` — OIDC (Okta, Azure AD). Library: `github.com/coreos/go-oidc`.
- `google_strategy.go` — Google Workspace OAuth2.

Each strategy:
- Configured by `TENANT_SSO_CONFIG` env var (JSON blob injected at provisioning).
- On successful auth: upsert user record, assign default role from SSO attributes.

Add endpoints:
- `GET /api/auth/sso/initiate` → redirect to IdP.
- `GET /api/auth/sso/callback` → handle response, issue JWT.
- `POST /api/admin/auth/test-sso` → test SSO connection (used by onboarding).

---

### W14.2 — MFA reset endpoint

**File**: `services/user-auth-service/internal/grpc/user_handler.go`

```proto
rpc ResetMFA(ResetMFARequest) returns (Empty);
```

Only callable with a platform admin token (`aud: platform`).
Logs the reset to `audit_events` table.

---

### W14.3 — Audit log table

**File**: `services/user-auth-service/migrations/009_audit_log.sql`

```sql
CREATE TABLE IF NOT EXISTS audit_events (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  actor_id VARCHAR(36),
  actor_type VARCHAR(20),  -- 'user', 'admin', 'system'
  action VARCHAR(100) NOT NULL,
  target_id VARCHAR(36),
  target_type VARCHAR(50),
  metadata JSONB,
  ip_address INET,
  created_at TIMESTAMPTZ DEFAULT NOW()
);
CREATE INDEX idx_audit_events_actor ON audit_events(actor_id);
CREATE INDEX idx_audit_events_created ON audit_events(created_at);
```

Append to `audit_events` on: login, logout, failed login, password change, MFA change,
impersonation start/end, SSO login.

---

### W14.4 — Impersonation token validation

**File**: `services/user-auth-service/internal/auth/impersonation.go`

`POST /api/auth/impersonate` endpoint:
1. Receive short-lived impersonation token (from query param or body).
2. Verify RSA signature using `PLATFORM_PUBLIC_KEY` env var.
3. Check `type == "impersonation"`, `tenant_id` matches this tenant, `exp` not expired.
4. Check Redis revocation set: `tenant:{slug}:revoked:{impersonation_id}`.
5. Log to `audit_events`.
6. Issue 30-min tenant JWT with `impersonated_by` claim.
7. Return JWT.

`DELETE /api/auth/impersonate/{impersonation_id}`:
1. Add `impersonation_id` to Redis revocation set (TTL: 2h).
2. Return 204.

---

## W15: email-service Updates

**DEPENDS ON**: nothing  
**Agent type**: email

### W15.1 — Broadcast messages

**File**: `services/email-service/internal/handlers/broadcast_handler.go`

`POST /api/broadcast` (admin only):
```json
{
  "targets": {"all_tenants": true} | {"tenant_ids": ["id1"]},
  "message": "30-min rolling restart across the platform.",
  "channels": ["in_app", "email_admins"]
}
```

- `in_app`: publish to Kafka topic `broadcast.in_app` → frontends consume via WebSocket or SSE.
- `email_admins`: send email to primary admin of each targeted tenant.

---

### W15.2 — Kafka consumers for notifications

Add consumers:
- `discussion.mention` → email the mentioned user: "You were mentioned in [thread title]".
- `incident.opened` (where tenant_id is set) → email tenant primary admin.
- `grade.updated` → (if configured) email student with grade notification.

---

## W16: tenant-service Updates

**DEPENDS ON**: W5, W6, W7 (new services scaffolded with Dockerfiles)  
**Agent type**: tenant

### W16.1 — Update DockerProvisioner to provision 8 services

**File**: `services/tenant-service/internal/docker/provisioner.go`

Currently provisions: course, assignment, content.
Must provision all 8:
1. `user-auth-{slug}`
2. `course-{slug}`
3. `assignment-{slug}`
4. `content-{slug}`
5. `video-{slug}`
6. `discussion-{slug}`
7. `scheduling-{slug}`
8. `ai-{slug}`

Each container:
- Correct image name (build images for new services first).
- `TENANT_ID`, `TENANT_SLUG`, `DB_SCHEMA` env vars.
- `REDIS_URL`, `KAFKA_BROKERS`, `MINIO_ENDPOINT`, `DB_DSN`.
- `PLATFORM_PUBLIC_KEY` (for user-auth and ai).
- `ANTHROPIC_API_KEY` (for ai service).
- Connected to `tenant-{slug}-network`.

---

### W16.2 — Create per-tenant networks

**File**: `services/tenant-service/internal/docker/provisioner.go`

Currently connects containers to `slate-network`. Must:
1. Create `tenant-{slug}-network` Docker network before starting containers.
2. Connect all 8 tenant containers to `tenant-{slug}-network`.
3. Connect `api-gateway` container to `tenant-{slug}-network` (Docker network connect).

---

### W16.3 — Update config/tenants/{slug}.yaml format

**File**: `services/tenant-service/internal/traefik/config_writer.go`

Update written YAML to include all 8 service endpoints:
```yaml
services:
  user-auth:   "http://user-auth-{slug}:50051"
  course:      "http://course-{slug}:50052"
  assignment:  "http://assignment-{slug}:50053"
  content:     "http://content-{slug}:50054"
  video:       "http://video-{slug}:50055"
  discussion:  "http://discussion-{slug}:50056"
  scheduling:  "http://scheduling-{slug}:50063"
  ai:          "http://ai-{slug}:50064"
```

---

### W16.4 — Deprovisioning: flush Redis keys

**File**: `services/tenant-service/internal/docker/deprovisioner.go`

Add step: `SCAN + DEL tenant:{slug}:*` before stopping containers.

---

## W17: api-gateway Updates

**DEPENDS ON**: W1, W2, W3  
**Agent type**: gateway

### W17.1 — Redis token validation cache

**File**: `services/api-gateway/src/auth/service.rs`

After successful `validate_token` gRPC call, cache result in Redis:
- Key: `tenant:{slug}:session:{sha256(token)}` or `platform:auth:{sha256(token)}`
- TTL: 60s.
- On next request with same token: check Redis first; skip gRPC call if cache hit.

On logout: delete the cache key.

This reduces gRPC calls to user-auth-service by ~95% for active sessions.

---

### W17.2 — New route registrations

**File**: `services/api-gateway/src/app/routes.rs`  
**File**: `config/gateway-config.yaml`

Add routes for new services:
- `/api/discussions/*` → `discussion-{slug}:50056`
- `/api/scheduling/*` → `scheduling-{slug}:50063`
- `/api/ai/*` → `ai-{slug}:50064`
- `/api/incidents/*` → `incident-service:50061` (platform)
- `/api/flags/*` → `feature-flag-service:50062` (platform)
- `/api/broadcast` → `email-service:50058` (platform)
- `/api/admin/auth/*` → `admin-auth-service:50060` (platform)
- `/api/status` → `incident-service:50061` (public, no auth)

---

### W17.3 — Feature flag middleware

**File**: `services/api-gateway/src/auth/middleware.rs`

On each authenticated request:
1. Fetch flags for tenant from feature-flag-service (or Redis cache).
2. Attach as request extension: `FeatureFlags { flags: HashMap<String, bool> }`.
3. Route handlers can check flags without an extra service call.

---

### W17.4 — Tenant routing for new services

**File**: `services/api-gateway/src/handlers/gateway/routing.rs`

Extend the tenant routing table lookup to include the 3 new service keys:
`discussion`, `scheduling`, `ai`.

---

## W18: Admin Frontend Updates

**DEPENDS ON**: W2, W3, W4 (new platform services)  
**Agent type**: admin-fe  
**Directory**: `frontend/admin/`

### W18.1 — New routes scaffold

Add pages (Next.js App Router):

```
app/
  ops/page.tsx                    — Ops HQ (replace /admin/dashboard)
  incidents/page.tsx              — Incident list
  incidents/[id]/page.tsx         — Incident room
  status/page.tsx                 — Service status (public)
  schools/[id]/page.tsx           — Tenant detail (improve existing)
  users/[id]/page.tsx             — User detail + impersonate
  flags/page.tsx                  — Feature flags
  broadcast/page.tsx              — Broadcast message
  onboarding/school/page.tsx      — Onboard school wizard
  data/import/page.tsx            — LMS migration
  billing/page.tsx                — Billing network view
  audit/page.tsx                  — Audit log
```

---

### W18.2 — Ops HQ page (`/ops`)

Cards: Schools (count + amber count), MAU, Uptime 30d, Open Incidents.

Live activity feed (polling every 30s):
- Fetch from `/api/incidents?status=open&limit=5`.
- Fetch from `/api/metrics/platform`.

Incident count badge on sidebar item.

---

### W18.3 — Incident pages

`/incidents`: table with columns: Priority badge, Title, Tenant, Age, Status.
Filter tabs: Open / Watch / Resolved.

`/incidents/[id]`: Incident room layout:
- Impact summary + tenant badge.
- Timeline (chronological events + comments).
- Actions: Post update, Notify tenant, Escalate P0.
- Real-time updates via polling (30s).

---

### W18.4 — Feature flags page (`/flags`)

Table: Flag key, Description, Current state (toggle), Targets, Last updated.
"Edit" opens a slide-over with rule builder:
- Select rule type (all / tenant / role / percentage).
- Configure value.
- Save → call `PUT /api/flags/{key}`.

---

### W18.5 — Impersonation flow

On `/users/[id]` page:
- Show user info (name, role, last active, recent actions).
- "Impersonate" button → `POST /api/tenants/{tenantId}/impersonate/{userId}`.
- On response: redirect browser to `redirect_url`.

---

### W18.6 — API client updates

**File**: `frontend/admin/lib/api/client.ts`

Add methods:
```typescript
incidentsApi: { list, get, create, update, addEvent }
flagsApi: { list, update }
broadcastApi: { send }
statusApi: { get }
auditApi: { list }
```

---

## W19: Instructor/Provider Frontend Updates

**DEPENDS ON**: W8, W9, W11, W12, W5, W6  
**Agent type**: provider-fe  
**Directory**: `frontend/provider/` (or `frontend/instructor/`)

### W19.1 — New routes scaffold

```
app/
  teach/page.tsx                     — Teaching Today
  grade/[assignmentId]/page.tsx      — Grading queue
  grade/[assignmentId]/[subId]/page.tsx  — Grade submission
  grade/batch/page.tsx               — Batch grade
  roster/page.tsx                    — Roster health
  roster/[studentId]/page.tsx        — Student detail
  lecture/live/page.tsx              — Live lecture
  office-hours/page.tsx              — Host OH view
  discussion/page.tsx                — Threads needing reply
```

---

### W19.2 — Teaching Today page (`/teach`)

3 cards: Grading queue count, Upcoming (lecture, OH), Threads needing reply.
"At risk" callout: list students with `risk_level: at_risk`.

---

### W19.3 — Grading queue (`/grade/:assignmentId`)

Pattern groups displayed as cards:
- Auto-tests passed count / total, similarity description.
- Expand to see student list in pattern.
- "1-click full credit" for passing patterns.
- "Open queue" → `/grade/:assignmentId/:subId` for first in pattern.

Keyboard shortcuts: `j/k` to move between submissions within queue.

---

### W19.4 — Grade submission (`/grade/:id/:subId`)

Split pane: left = submission content (code highlighted), right = rubric.
Rubric: each row with point input and check/flag icon.
"Save & Next ↵" submits grade and advances to next submission.
Comment textarea with "AI draft suggestion" button (if feature flag on).

---

### W19.5 — Roster health (`/roster`)

Table sorted by risk level (at_risk → slipping → healthy).
Columns: Avatar, Name, Risk badge, Last active, Grade trend sparkline, Action button.
"Reach out" action opens pre-filled message modal.

---

### W19.6 — Live lecture (`/lecture/live`)

Live indicator (pulsing red dot) + attendance count.
Slide title display (manual input).
Q&A queue: sorted by upvotes, instructor can "dismiss" a question.
"End lecture" button.

---

## W20: Student Frontend Updates

**DEPENDS ON**: W7, W5, W6  
**Agent type**: student-fe  
**Directory**: `frontend/student/` (or `frontend/app/`)

### W20.1 — New routes scaffold

```
app/
  today/page.tsx              — Today view (action-first hero)
  inbox/page.tsx              — Inbox
  people/page.tsx             — People list
  people/[id]/page.tsx        — Person detail
  office-hours/page.tsx       — Book OH
  plan/page.tsx               — Study plan
```

---

### W20.2 — Today view (`/today`)

Hero card: AI welcome message + current streak.
Agenda: due-soon items, upcoming lecture (countdown), reading, study group.
"Resume →" button linking to most recent in-progress content.

Implementation:
- Fetch welcome message: `GET /api/ai/welcome-message`.
- Fetch due assignments: `GET /api/assignments?status=open&limit=3`.
- Fetch upcoming lecture: `GET /api/courses/next-lecture`.

---

### W20.3 — Inbox (`/inbox`)

Tabs: @You / All / Profs / Peers.
Each row: avatar, sender, preview, time.
Click → opens thread modal or navigates to full thread.

Fetch: `GET /api/discussions/inbox`.
Mark read on open: `PATCH /api/discussions/inbox/{id}/read`.

---

### W20.4 — Office hours booking (`/office-hours`)

List instructors with available slots (from scheduling-service).
"Book" opens a slot picker.
Pre-fill questions textarea: auto-populated from current assignment draft questions.

On submit: `POST /api/scheduling/bookings`.

---

### W20.5 — Study plan (`/plan`)

16-week calendar grid.
"Generate plan" → `POST /api/ai/study-plan` (long operation, show spinner).
Each week: list of tasks with checkboxes (student can mark done).
"Regenerate" button clears cache and requests fresh plan.

---

### W20.6 — Grade breakdown with "Path to A"

Enhance `/grades/:courseId` page:
- Add "Path to A" panel: fetch from `GET /api/ai/grade-projection`.
- Show projected grade + required scores per remaining assignment.
- Update in real-time as grades change (invalidated by `grade.updated` event).

---

### W20.7 — Video resume

On `/modules/:id`:
- On load: `GET /api/content/{contentId}` → read `resume_position_seconds`.
- Start video at that position if > 0.
- Every 10s of playback: `PUT /api/content/{contentId}/position` → update position.

---

## Cross-cutting Concerns

### API Gateway routes summary

When all workstreams complete, `config/gateway-config.yaml` must include routes for:

```yaml
# Tenant routes (use {slug} from X-Tenant-Slug header)
/api/auth/*           → user-auth-{slug}:50051
/api/courses/*        → course-{slug}:50052
/api/assignments/*    → assignment-{slug}:50053
/api/grades/*         → assignment-{slug}:50053
/api/content/*        → content-{slug}:50054
/api/video/*          → video-{slug}:50055
/api/discussions/*    → discussion-{slug}:50056
/api/scheduling/*     → scheduling-{slug}:50063
/api/ai/*             → ai-{slug}:50064

# Platform routes (no tenant)
/api/admin/auth/*     → admin-auth-service:50060
/api/tenants/*        → tenant-service:50057
/api/onboarding/*     → onboarding-service:8084
/api/incidents/*      → incident-service:50061
/api/flags/*          → feature-flag-service:50062
/api/metrics/*        → metrics-service:50059
/api/broadcast        → email-service:50058
/api/status           → incident-service:50061 (public)
```

---

### Kafka topics (full list)

| Topic | Producer | Consumers |
|---|---|---|
| `onboarding.approved` | onboarding-service | tenant-service |
| `tenant.provisioned` | tenant-service | email-service |
| `tenant.deprovisioned` | tenant-service | email-service |
| `grade.updated` | assignment-{slug} | metrics-service, ai-{slug} |
| `assignment.created` | assignment-{slug} | ai-{slug} |
| `course.lecture_started` | course-{slug} | video-{slug} |
| `course.lecture_ended` | course-{slug} | video-{slug} |
| `course.module_completed` | course-{slug} | ai-{slug} |
| `discussion.mention` | discussion-{slug} | email-service |
| `metrics.threshold_breached` | metrics-service | incident-service |
| `incident.opened` | incident-service | email-service |
| `incident.resolved` | incident-service | email-service |
| `audit.admin_action` | admin-auth-service | (audit store) |
| `audit.impersonation_started` | admin-auth-service | (audit store) |
| `migration.completed` | onboarding-service | tenant-service |
| `broadcast.in_app` | email-service | (frontend WebSocket/SSE) |

---

### Execution order recommendation

Start these workstreams first (blockers for others):
1. **W1** (Redis + docker-compose) — blocks all others.
2. **W16** (tenant provisioner) — start scaffolding in parallel with W5/W6/W7.
3. **W14** (user-auth SSO) — blocks W13 (onboarding SSO step).

Safe to run fully in parallel after W1:
- W2, W3, W4 (platform services)
- W5, W6, W7 (new tenant services)
- W8, W9, W10, W11, W12 (existing service updates)
- W15 (email-service)
- W17 (api-gateway)

Frontends (W18, W19, W20) should start only after their backend dependencies have
working endpoints (at minimum stubs returning correct shapes).
