# Email Service Plan

## Owner Agent: `email-expert`

## Stack: Go, gRPC, PostgreSQL, Kafka

---

## Objective

Build a full in-platform messaging service (not SMTP email — this is inbox/messaging between users). Supports send, reply, inbox, sent folder, read/unread, archive, and Kafka-triggered notifications.

---

## Current State

- Stub placeholder with keep-alive loop
- No actual implementation

---

## Target APIs

### REST (port 8086)

```
POST   /messages                    — send message
GET    /messages/inbox              — paginated inbox for calling user
GET    /messages/sent               — sent messages
GET    /messages/:id                — message detail with thread
POST   /messages/:id/reply          — reply to message
PATCH  /messages/:id/read           — mark as read
PATCH  /messages/:id/unread         — mark as unread
DELETE /messages/:id                — soft delete (archive)
GET    /messages/unread-count       — count of unread messages
GET    /messages/threads/:threadId  — full conversation thread
POST   /messages/bulk               — send to multiple recipients or groups
```

### gRPC (port 50056)

Proto: `proto/email.proto` — SendMessage, GetInbox, MarkRead, GetThread

---

## Data Models (PostgreSQL)

### messages

```sql
CREATE TABLE messages (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    thread_id UUID NOT NULL,          -- groups replies together
    parent_id UUID,                   -- null = root message
    from_user_id UUID NOT NULL,
    subject VARCHAR(1000),
    body TEXT NOT NULL,
    is_deleted_by_sender BOOLEAN DEFAULT false,
    created_at TIMESTAMPTZ DEFAULT NOW()
);
```

### message_recipients

```sql
CREATE TABLE message_recipients (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    message_id UUID NOT NULL,
    recipient_user_id UUID NOT NULL,
    tenant_id UUID NOT NULL,
    is_read BOOLEAN DEFAULT false,
    is_archived BOOLEAN DEFAULT false,
    read_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT NOW()
);
```

### message_attachments

```sql
CREATE TABLE message_attachments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    message_id UUID NOT NULL,
    content_id UUID NOT NULL,         -- reference to content-management-service
    filename VARCHAR(500),
    file_size_bytes BIGINT
);
```

---

## Kafka Consumer Triggers

### `assignment.graded` → notify student

```go
func handleAssignmentGraded(event AssignmentGradedEvent) {
    SendSystemMessage(
        from: "system",
        to: event.StudentID,
        subject: fmt.Sprintf("Your assignment '%s' has been graded", event.AssignmentTitle),
        body: fmt.Sprintf("Score: %.1f/%.1f (%.1f%%)\n\nFeedback: %s", ...),
    )
}
```

### `announcement.posted` → distribute to enrolled students

```go
func handleAnnouncementPosted(event AnnouncementEvent) {
    for _, studentId := range event.EnrolledStudentIDs {
        SendSystemMessage(from: event.InstructorID, to: studentId, ...)
    }
}
```

### `room.created` → send invite notifications

```go
func handleRoomCreated(event RoomCreatedEvent) {
    for _, studentId := range event.InvitedStudentIDs {
        SendSystemMessage(...)
    }
}
```

---

## Kafka Events Produced

- `message.sent` `{ messageId, from, to[], subject, tenantId }`
- `message.read` `{ messageId, userId, tenantId }`

---

## Key Features

### Threading

All replies share the same `thread_id` (UUID of root message). Frontend renders as conversation view.

### Bulk Messaging

`POST /messages/bulk` accepts `{ recipientType: 'course' | 'group' | 'all', courseId, body }`. Service resolves recipients list from course-service via gRPC.

### Pagination

Inbox and sent use cursor-based pagination: `?cursor=<lastId>&limit=20&sort=newest`

---

## Tests

- Unit: message threading, bulk resolution
- Integration: send → inbox → reply → thread view
- Kafka: verify all triggered notifications
- Authorization: verify user can only read their own messages

---

## Files to Create/Modify

- [REWRITE] `services/email-service/` — full Go implementation
- [NEW] `services/email-service/cmd/server/main.go`
- [NEW] `services/email-service/internal/messaging/`
- [NEW] `services/email-service/internal/kafka/`
- [NEW] `services/email-service/migrations/`
- [NEW] `proto/email.proto`
- [MODIFY] `services/email-service/Dockerfile` — remove placeholder
