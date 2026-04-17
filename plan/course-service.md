# Course Service Plan

## Owner Agent: `course-expert`

## Stack: NestJS (TypeScript), MongoDB, Kafka (kafkajs)

---

## Objective

Complete the course-service with full CRUD, content management, module/lesson ordering, tenant scoping, and Kafka event production/consumption.

---

## Target APIs

### REST (port 3001)

```
POST   /courses                                    — create course
GET    /courses                                    — paginated list (tenant, status, instructor filters)
GET    /courses/:id                                — course detail
PUT    /courses/:id                                — update course details
PATCH  /courses/:id/lock                           — lock course
PATCH  /courses/:id/unlock                         — unlock course
DELETE /courses/:id                                — soft delete
POST   /courses/:id/modules                        — add module
GET    /courses/:id/modules                        — list modules (ordered)
PUT    /courses/:id/modules/:moduleId              — update/reorder module
DELETE /courses/:id/modules/:moduleId              — delete module
POST   /courses/:id/modules/:moduleId/lessons      — add lesson
PUT    /courses/:id/modules/:moduleId/lessons/:lid — update/reorder lesson
DELETE /courses/:id/modules/:moduleId/lessons/:lid — delete lesson
PATCH  /students/:studentId/courses/:courseId/progress — update completion %
GET    /students/:studentId/courses                — enrolled courses
GET    /courses/:id/students                       — students enrolled in course
```

### gRPC (port 50060 — new)

For service-to-service: GetCourse, ListCourses, GetStudentProgress, etc.

---

## Data Models (MongoDB)

### Course

```typescript
{
  _id: ObjectId,
  tenantId: string,          // REQUIRED — all queries scoped here
  instructorId: string,
  title: string,
  description: string,
  coverImageUrl: string,
  category: string,
  tags: string[],
  status: 'draft' | 'active' | 'locked' | 'archived',
  modules: Module[],
  enrolledStudentIds: string[],
  settings: {
    allowSelfEnrollment: boolean,
    visibleToStudents: boolean,
  },
  createdAt: Date,
  updatedAt: Date,
}

Module {
  id: string,
  title: string,
  description: string,
  order: number,
  lessons: Lesson[],
  isVisible: boolean,
}

Lesson {
  id: string,
  title: string,
  order: number,
  contentType: 'video' | 'pdf' | 'link' | 'text',
  contentUrl: string,
  isVisible: boolean,
  visibleAfter: Date | null,
}
```

### StudentProgress

```typescript
{
  _id: ObjectId,
  tenantId: string,
  studentId: string,
  courseId: string,
  completedLessonIds: string[],
  completionPercentage: number,
  lastActiveAt: Date,
  totalTimeMinutes: number,
  enrolledAt: Date,
}
```

---

## Kafka Events

### Produced

- `course.created` `{ courseId, tenantId, instructorId, title }`
- `course.updated` `{ courseId, tenantId, changes: string[] }`
- `course.locked` `{ courseId, tenantId }`
- `course.deleted` `{ courseId, tenantId }` — triggers cascade in other services
- `module.created` `{ courseId, moduleId, tenantId }`
- `lesson.completed` `{ studentId, courseId, lessonId, tenantId, completionPct }`
- `user.enrolled` `{ studentId, courseId, tenantId }` — triggers grade record creation

### Consumed

- `tenant.disabled` → set all courses for tenant to `locked` status
- `tenant.enabled` → restore course statuses

---

## Key Implementation Details

### Tenant Scoping

Every controller method extracts `tenantId` from JWT claims:

```typescript
@UseGuards(JwtAuthGuard)
@Get()
async listCourses(@TenantId() tenantId: string, @Query() query: ListCoursesDto) {
  return this.courseService.findAll({ tenantId, ...query });
}
```

### Module Reordering (drag-and-drop support)

PATCH `/courses/:id/modules/reorder` — accepts array of `{ moduleId, order }` and bulk updates.

### File Upload Integration

When a lesson has `contentType: 'video'|'pdf'`, the upload goes through `content-management-service`. Course service stores only the contentUrl returned.

### At-Risk Student Detection

`GET /courses/:id/students?filter=at-risk` — students with `completionPct < 30%` OR `lastActiveAt > 14 days ago`.

---

## Tests

- Unit: CourseService methods (mock MongoDB)
- Integration: full CRUD cycle per endpoint
- Kafka: verify events emitted on create/lock/delete
- Tenant isolation: verify queries cannot cross tenant boundaries

---

## Files to Create/Modify

- [MODIFY] `services/course-service/src/course/course.service.ts` — full implementation
- [MODIFY] `services/course-service/src/course/course.controller.ts` — all endpoints
- [NEW] `services/course-service/src/module/` — module CRUD module
- [NEW] `services/course-service/src/progress/` — student progress module
- [NEW] `services/course-service/src/kafka/` — producer + consumer
- [MODIFY] `services/course-service/src/course/schemas/` — updated schemas
