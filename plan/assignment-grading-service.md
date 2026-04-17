# Assignment Grading Service Plan

## Owner Agent: `grading-expert`

## Stack: Go, gRPC, PostgreSQL, Kafka, MinIO (file storage)

---

## Objective

Complete the assignment-grading-service with full assignment CRUD, submission handling, manual + auto grading, grading rules engine, grade estimation, and Kafka integration.

---

## Target APIs

### REST (port 8083)

```
POST   /assignments                         — create assignment with rubric
GET    /assignments                         — list (course, tenant, instructor filters)
GET    /assignments/:id                     — assignment detail
PUT    /assignments/:id                     — update (including deadline extension)
DELETE /assignments/:id                     — soft delete
POST   /assignments/:id/submissions         — student submits file(s)
GET    /assignments/:id/submissions         — list all submissions
GET    /assignments/:id/submissions/zip     — download all as ZIP
GET    /submissions/:id                     — single submission detail
POST   /submissions/:id/grade              — grade with feedback
POST   /assignments/:id/auto-grade          — trigger percentile auto-grading
GET    /courses/:id/grades                  — grade summary per course
GET    /courses/:id/grades/distribution     — percentile distribution
GET    /students/:id/grades                 — all grades for student
GET    /students/:id/grade-estimate         — computed final grade estimate
GET    /grading-rules                       — list rules for tenant
POST   /grading-rules                       — create rule
PUT    /grading-rules/:id                   — update rule
DELETE /grading-rules/:id                   — delete rule
```

### gRPC (port 50053)

Proto: `proto/assignment_grading.proto` — GetGrade, ListGrades, GetGradingRules, ComputeEstimate

---

## Data Models (PostgreSQL, schema per tenant)

### assignments

```sql
CREATE TABLE assignments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    course_id UUID NOT NULL,
    instructor_id UUID NOT NULL,
    title VARCHAR(500) NOT NULL,
    description TEXT,
    rubric JSONB,                    -- [{criterion, maxPoints, description}]
    assignment_type VARCHAR(50),      -- 'exam' | 'homework' | 'quiz' | 'project'
    due_date TIMESTAMPTZ,
    max_file_size_mb INT DEFAULT 50,
    allowed_file_types TEXT[],
    is_deleted BOOLEAN DEFAULT false,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);
```

### submissions

```sql
CREATE TABLE submissions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    assignment_id UUID NOT NULL,
    student_id UUID NOT NULL,
    file_urls TEXT[],                -- MinIO object keys
    status VARCHAR(50) DEFAULT 'submitted',  -- submitted | graded | late
    submitted_at TIMESTAMPTZ DEFAULT NOW(),
    is_late BOOLEAN DEFAULT false
);
```

### grades

```sql
CREATE TABLE grades (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    submission_id UUID NOT NULL UNIQUE,
    assignment_id UUID NOT NULL,
    student_id UUID NOT NULL,
    course_id UUID NOT NULL,
    score DECIMAL(5,2),
    max_score DECIMAL(5,2),
    percentage DECIMAL(5,2),
    letter_grade VARCHAR(5),
    rubric_scores JSONB,             -- per-criterion scores
    feedback TEXT,
    graded_by UUID,                  -- instructor or 'system' for auto-grade
    graded_at TIMESTAMPTZ,
    override_justification TEXT,
    percentile DECIMAL(5,2)          -- position in cohort
);
```

### grading_rules

```sql
CREATE TABLE grading_rules (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    course_id UUID,                   -- null = tenant-wide default
    assignment_type VARCHAR(50),
    weight DECIMAL(5,2),              -- e.g., 0.40 for 40%
    late_penalty_per_day DECIMAL(5,2),
    max_late_penalty DECIMAL(5,2),
    grade_scale JSONB,                -- [{"grade":"A","min":90},{"grade":"B","min":80}...]
    created_at TIMESTAMPTZ DEFAULT NOW()
);
```

---

## Auto-Grading Algorithm

```go
func AutoGrade(ctx context.Context, assignmentID uuid.UUID) error {
    grades := repo.GetAllGrades(assignmentID)
    scores := extractScores(grades)

    for _, g := range grades {
        percentile := computePercentile(g.Score, scores)
        g.Percentile = percentile
        // Auto-assign letter grade based on percentile thresholds
        // (configurable per grading rule)
        repo.UpdateGrade(g)
    }

    emitKafka("assignment.auto_graded", assignmentID)
}
```

## Grade Estimation

```go
func EstimateFinalGrade(studentID, courseID uuid.UUID) GradeEstimate {
    rules := repo.GetGradingRules(courseID)
    grades := repo.GetStudentGrades(studentID, courseID)

    weightedSum := 0.0
    totalWeight := 0.0

    for assignmentType, weight := range rules.Weights {
        typeGrades := filterByType(grades, assignmentType)
        if len(typeGrades) > 0 {
            avg := average(typeGrades)
            weightedSum += avg * weight
            totalWeight += weight
        }
    }

    estimated := weightedSum / totalWeight
    return GradeEstimate{
        Percentage: estimated,
        LetterGrade: letterGradeFromScale(estimated, rules.GradeScale),
        Confidence: totalWeight,  // higher = more assignments graded
    }
}
```

---

## Kafka Events

### Produced

- `assignment.created`
- `assignment.deadline.updated`
- `submission.uploaded` `{ studentId, assignmentId, courseId, tenantId }`
- `submission.graded` `{ studentId, assignmentId, score, tenantId }`
- `grade.finalized`

### Consumed

- `course.deleted` → soft-delete all assignments for course
- `user.enrolled` → create empty grade record (for progress tracking)

---

## File Handling (MinIO)

- Student uploads go to MinIO via presigned URL
- MinIO bucket: `assignments-{tenantId}`
- Object key: `{courseId}/{assignmentId}/{studentId}/{filename}`
- ZIP download: stream multiple objects from MinIO into zip

---

## Tests

- Unit: grading algorithm, estimation logic, percentile calculation
- Integration: full submission → grade → estimate cycle
- Kafka: verify events on all state transitions
- Authorization: verify student can't view other students' submissions

---

## Files to Create/Modify

- [MODIFY] `services/assignment-grading-service/` — complete implementation
- [NEW] `services/assignment-grading-service/internal/grading/` — engine
- [NEW] `services/assignment-grading-service/internal/estimation/` — grade estimator
- [NEW] `services/assignment-grading-service/internal/kafka/`
- [MODIFY] `proto/assignment_grading.proto`
- [MODIFY] `services/assignment-grading-service/migrations/`
