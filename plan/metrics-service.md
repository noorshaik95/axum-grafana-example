# Metrics Service Plan

## Owner Agent: `metrics-expert`

## Stack: Go, gRPC, PostgreSQL (TimescaleDB extension), Kafka

---

## Objective

Build the metrics/analytics service that aggregates platform-wide and per-tenant stats, student progress metrics, grade distributions, and course engagement data. Powers admin platform dashboards and student grade comparison graphs.

---

## Current State

- Stub placeholder with keep-alive loop
- No actual implementation

---

## Target APIs

### REST (port 8087)

```
GET /metrics/platform                         — platform-wide stats (admin)
GET /metrics/tenants/:id                      — per-tenant usage & activity
GET /metrics/courses/:id                      — course engagement stats
GET /metrics/students/:id                     — individual student progress
GET /metrics/students/:id/time-on-task        — time spent per course/module
GET /metrics/grades/distribution/:courseId    — grade percentile distribution
GET /metrics/grades/comparison/:studentId/:courseId — student vs class avg
```

### gRPC (port 50057)

Proto: `proto/metrics.proto` — GetPlatformMetrics, GetStudentProgress, GetGradeDistribution

---

## Data Models (PostgreSQL + TimescaleDB)

### event_log (hypertable — time-series)

```sql
CREATE TABLE event_log (
    time TIMESTAMPTZ NOT NULL,
    tenant_id UUID NOT NULL,
    event_type VARCHAR(100) NOT NULL,
    user_id UUID,
    course_id UUID,
    assignment_id UUID,
    metadata JSONB,
    INDEX (tenant_id, event_type, time DESC)
);
SELECT create_hypertable('event_log', 'time');
```

### student_progress_snapshots

```sql
CREATE TABLE student_progress_snapshots (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    student_id UUID NOT NULL,
    course_id UUID NOT NULL,
    completion_pct DECIMAL(5,2),
    time_on_task_minutes INT,
    lessons_completed INT,
    last_activity_at TIMESTAMPTZ,
    snapshot_at TIMESTAMPTZ DEFAULT NOW()
);
```

### grade_stats

```sql
CREATE TABLE grade_stats (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    course_id UUID NOT NULL,
    assignment_id UUID,             -- null = course-level aggregate
    mean_score DECIMAL(5,2),
    median_score DECIMAL(5,2),
    p25 DECIMAL(5,2),
    p75 DECIMAL(5,2),
    std_dev DECIMAL(5,2),
    student_count INT,
    computed_at TIMESTAMPTZ DEFAULT NOW()
);
```

---

## Kafka Consumers

### `lesson.completed` → update student progress

```go
func handleLessonCompleted(event LessonCompletedEvent) {
    // Increment lessons_completed, recalculate completion_pct
    // Update time_on_task_minutes
    // Upsert student_progress_snapshots
    // Update event_log
}
```

### `submission.uploaded` → track assignment engagement

```go
func handleSubmissionUploaded(event SubmissionEvent) {
    // Log to event_log
    // Update per-course submission rate stats
}
```

### `submission.graded` → update grade distribution

```go
func handleSubmissionGraded(event GradeEvent) {
    // Recalculate grade_stats for assignment + course
    // Update percentile for all students in assignment
}
```

### `room.ended` → log attendance

```go
func handleRoomEnded(event RoomEndedEvent) {
    // Log attendance per participant to event_log
}
```

### `message.sent` → communication activity

```go
func handleMessageSent(event MessageSentEvent) {
    // Increment communication metrics in event_log
}
```

---

## Platform Stats (Admin Dashboard)

```go
type PlatformMetrics struct {
    ActiveTenants    int
    DAU              int     // distinct users in last 24h
    MAU              int     // distinct users in last 30d
    TotalStorageGB   float64
    TotalCourses     int
    TotalStudents    int
    ErrorRate7d      float64 // from Prometheus
    KafkaLag         int64   // consumer group lag
}
```

## Grade Distribution (for student comparison graph)

```go
type GradeDistribution struct {
    AssignmentID string
    Buckets      []DistributionBucket  // [0-10, 10-20, ..., 90-100]
    Mean         float64
    Median       float64
    StudentScore float64   // requesting student's score
    Percentile   float64   // student's percentile in class
}
```

---

## Tests

- Unit: percentile calculation, distribution bucketing
- Integration: Kafka event → metric update → API read
- Platform stats: verify counts across multiple tenants

---

## Files to Create/Modify

- [REWRITE] `services/metrics-service/` — full Go implementation
- [NEW] `services/metrics-service/cmd/server/main.go`
- [NEW] `services/metrics-service/internal/analytics/`
- [NEW] `services/metrics-service/internal/kafka/`
- [NEW] `services/metrics-service/migrations/` (TimescaleDB)
- [NEW] `proto/metrics.proto`
- [MODIFY] `services/metrics-service/Dockerfile` — remove placeholder
