# Provider Frontend Plan

## Owner Agent: `provider-expert`

## Stack: Next.js 14 (App Router), TypeScript, TanStack Query, Tailwind CSS, shadcn/ui, Playwright

---

## Objective

Build the complete instructor/lecturer interface for managing courses, content, students, grading, video sessions, announcements, and messaging.

---

## Route Structure

```
app/
├── (auth)/
│   └── login/page.tsx
├── (dashboard)/
│   ├── layout.tsx                          ← Dashboard shell with sidebar
│   ├── page.tsx                            ← Dashboard overview
│   ├── courses/
│   │   ├── page.tsx                        ← Course list
│   │   ├── new/page.tsx                    ← Create course
│   │   └── [id]/
│   │       ├── page.tsx                    ← Course detail
│   │       ├── content/page.tsx            ← Content/module management
│   │       ├── assignments/
│   │       │   ├── page.tsx                ← Assignment list
│   │       │   ├── new/page.tsx            ← Create assignment
│   │       │   └── [assignmentId]/
│   │       │       ├── page.tsx            ← Assignment detail
│   │       │       └── submissions/page.tsx← Grade submissions
│   │       ├── students/page.tsx           ← Student progress table
│   │       └── settings/page.tsx           ← Course settings
│   ├── grading/
│   │   ├── page.tsx                        ← All pending grading
│   │   └── rules/page.tsx                  ← Grading rules CRUD
│   ├── video/
│   │   ├── page.tsx                        ← Upcoming + past sessions
│   │   └── new/page.tsx                    ← Schedule new session
│   ├── announcements/
│   │   ├── page.tsx                        ← Announcements list
│   │   └── new/page.tsx                    ← Create announcement
│   └── messages/
│       ├── page.tsx                        ← Inbox
│       ├── sent/page.tsx                   ← Sent folder
│       └── [threadId]/page.tsx             ← Thread view
```

---

## Key Components

### 1. Course Content Manager (`components/courses/content/ContentManager.tsx`)

Drag-and-drop module/lesson reorder using `@dnd-kit`:

```tsx
// DnD sortable list of modules
// Each module expands to show lessons
// Add/edit/delete inline
// Upload button per lesson → opens FileUploadModal
// Visibility toggle per lesson (date picker, group selector)
```

### 2. Grade Submission Interface (`components/grading/SubmissionGrader.tsx`)

```tsx
// Split view: submission file (PDF/image viewer) on left, grading form on right
// Rubric criteria checkboxes with point inputs
// Total score auto-calculated from rubric
// Feedback text area
// Grade override toggle with justification
```

### 3. Grade Distribution Chart (`components/grading/GradeDistributionChart.tsx`)

Using Recharts or Chart.js:

- Histogram of grade distribution across class
- Marker for each student's position
- Percentile lines (P25, P75)

### 4. Student Progress Table (`components/students/ProgressTable.tsx`)

```tsx
// DataTable columns: Student Name | Completion % | Grade | Last Active | Status
// Status: on-track | at-risk | inactive
// Sort/filter by all columns
// Click row → student detail modal
// "At Risk" filter toggle
```

### 5. Video Session Manager (`components/video/SessionManager.tsx`)

```tsx
// Upcoming sessions list with join button (enabled 10min before)
// Create session modal: title, date/time, duration, course link
// Bulk invite students (checkboxes from course roster)
// Past sessions with recording links
```

### 6. Rich Text Announcements (`components/announcements/AnnouncementEditor.tsx`)

Using TipTap editor:

- Bold, italic, lists, links, image embeds
- Schedule picker (deliver at future date/time)
- Course scope selector (one course or all)

### 7. Grading Rules CRUD (`components/grading/GradingRulesEditor.tsx`)

```tsx
// Table of assignment types with weight % inputs (must sum to 100%)
// Grade scale editor: A=90+, B=80+, C=70+, D=60+, F=<60
// Late penalty: % per day + max cap
// Per-course or tenant-wide toggle
```

---

## API Integration

```typescript
// lib/api/courses.ts
export const coursesApi = {
  list: (params) => get('/courses', { params }),
  create: (data: CreateCourseDto) => post('/courses', data),
  update: (id: string, data: UpdateCourseDto) => put(`/courses/${id}`, data),
  lock: (id: string) => patch(`/courses/${id}/lock`),
  unlock: (id: string) => patch(`/courses/${id}/unlock`),
  delete: (id: string) => del(`/courses/${id}`),
  getModules: (id: string) => get(`/courses/${id}/modules`),
  reorderModules: (id: string, order: ModuleOrder[]) =>
    patch(`/courses/${id}/modules/reorder`, order),
}

// lib/api/grading.ts
export const gradingApi = {
  getSubmissions: (assignmentId: string) => get(`/assignments/${assignmentId}/submissions`),
  grade: (submissionId: string, data: GradeDto) => post(`/submissions/${submissionId}/grade`, data),
  autoGrade: (assignmentId: string) => post(`/assignments/${assignmentId}/auto-grade`),
  getDistribution: (courseId: string, assignmentId?: string) =>
    get(`/metrics/grades/distribution/${courseId}`),
  getRules: () => get('/grading-rules'),
  createRule: (data: GradingRuleDto) => post('/grading-rules', data),
}
```

---

## Playwright Tests (`tests/e2e/provider/`)

```typescript
// tests/e2e/provider/courses.spec.ts
test('Create and publish course', async ({ page }) => {
  await loginAsProvider(page);
  await page.goto('/courses/new');
  await page.fill('[name="title"]', 'Introduction to Python');
  await page.fill('[name="description"]', 'Learn Python from scratch');
  await page.click('button:has-text("Create Course")');
  await expect(page).toHaveURL(/\/courses\/[a-z0-9-]+/);
});

// tests/e2e/provider/grading.spec.ts
test('Grade a submission', async ({ page }) => { ... });

// tests/e2e/provider/content.spec.ts
test('Upload course material', async ({ page }) => { ... });

// tests/e2e/provider/video.spec.ts
test('Schedule and manage video session', async ({ page }) => { ... });
```

---

## Files to Create/Modify

- [REWRITE] `frontend/provider/app/` — complete app directory
- [NEW] `frontend/provider/components/courses/`
- [NEW] `frontend/provider/components/grading/`
- [NEW] `frontend/provider/components/video/`
- [NEW] `frontend/provider/components/announcements/`
- [NEW] `frontend/provider/components/messages/`
- [NEW] `frontend/provider/components/students/`
- [NEW] `frontend/provider/lib/api/`
- [NEW] `tests/e2e/provider/`
