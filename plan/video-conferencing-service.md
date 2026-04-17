# Video Conferencing Service Plan

## Owner Agent: `video-expert`

## Stack: Rust, tonic (gRPC), axum (WebSocket), PostgreSQL (sqlx), Kafka

---

## Current State

- Service compiles and starts (just fixed in current session)
- Core session CRUD + WebRTC signaling implemented
- GCS recording disabled for dev (RECORDING_ENABLED=false)

---

## Remaining Work

### New REST Endpoints (via axum HTTP handlers)

```
POST   /rooms                          — create room (maps to gRPC ScheduleSession)
POST   /rooms/:id/invitations          — invite students (bulk)
GET    /rooms                          — list rooms (filter: instructor, course, upcoming/past)
GET    /rooms/:id                      — room detail + join token
DELETE /rooms/:id                      — cancel session
GET    /rooms/:id/join                 — generate join token (validates eligibility)
```

### gRPC additions (proto updates)

```protobuf
rpc InviteStudents(InviteStudentsRequest) returns (InviteStudentsResponse);
rpc GetRoomDetail(GetRoomRequest) returns (RoomDetail);
rpc GenerateJoinToken(JoinTokenRequest) returns (JoinTokenResponse);
```

### Kafka Integration

**Produced:**

- `room.created` `{ sessionId, instructorId, courseId, scheduledAt, tenantId }`
- `room.cancelled` `{ sessionId, tenantId }`
- `room.started` `{ sessionId, tenantId }` — emitted when first participant joins
- `room.ended` `{ sessionId, durationMinutes, participantCount, tenantId }`

**Consumed:**

- `course.deleted` → cancel all scheduled rooms for that course
- `tenant.disabled` → cancel all upcoming rooms for tenant

### Database Additions

```sql
ALTER TABLE video_sessions ADD COLUMN course_id UUID;
ALTER TABLE video_sessions ADD COLUMN tenant_id UUID NOT NULL DEFAULT gen_random_uuid();

CREATE TABLE session_invitations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    session_id UUID NOT NULL,
    student_id UUID NOT NULL,
    tenant_id UUID NOT NULL,
    invited_at TIMESTAMPTZ DEFAULT NOW(),
    status VARCHAR(50) DEFAULT 'pending'  -- pending | accepted | declined
);
```

### Recording Flow (disabled for dev, architecture documented)

When `RECORDING_ENABLED=true`:

1. Recording started → stored in `/tmp/recordings/`
2. `room.ended` → `RecordingProcessor` uploads to GCS
3. URL stored in DB, available via `GET /rooms/:id/recordings`

### Tenant Scoping

All queries must include `tenant_id` filter. Extract from JWT via `X-Tenant-ID` header (injected by Traefik).

### Kafka Consumer Setup

```rust
// src/kafka/consumer.rs
pub async fn run_consumer(brokers: &str) {
    let consumer = create_consumer(brokers, "video-conferencing-group");
    consumer.subscribe(&["course.deleted", "tenant.disabled"]).unwrap();

    loop {
        match consumer.recv().await {
            "course.deleted" => cancel_sessions_for_course(payload.course_id),
            "tenant.disabled" => cancel_sessions_for_tenant(payload.tenant_id),
        }
    }
}
```

### Tests

- Integration: full session lifecycle (create → invite → join → end)
- WebRTC: signaling message exchange test
- Kafka: verify room.created/ended events
- Tenant isolation: verify cross-tenant access denied

---

## Files to Create/Modify

- [MODIFY] `services/video-conferencing-service/src/grpc/service.rs` — add invite/token RPCs
- [NEW] `services/video-conferencing-service/src/kafka/` — consumer + producer
- [NEW] `services/video-conferencing-service/src/handlers/rooms.rs` — REST handlers
- [MODIFY] `services/video-conferencing-service/src/database/repository.rs` — tenant-scoped queries
- [MODIFY] `services/video-conferencing-service/migrations/` — add invitations table, tenant_id columns
- [MODIFY] `proto/video_conferencing.proto` — add new RPCs
