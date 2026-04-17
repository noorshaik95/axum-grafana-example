# Student Frontend Plan

## Owner Agent: `student-expert`

## Stack: Next.js 14 (App Router), TypeScript, TanStack Query, Tailwind CSS, shadcn/ui, Playwright

---

## Objective

Build the complete student learner interface: course access, assignment submission, grade tracking, live class attendance, announcements, and messaging.

---

## Route Structure

```
app/
├── (auth)/
│   ├── login/page.tsx
│   └── register/page.tsx
├── (dashboard)/
│   ├── layout.tsx                          ← Student shell with sidebar
│   ├── page.tsx                            ← Dashboard overview
│   ├── courses/
│   │   ├── page.tsx                        ← Enrolled courses grid
│   │   └── [id]/
│   │       ├── page.tsx                    ← Course home (syllabus, modules)
│   │       ├── modules/[moduleId]/page.tsx ← Module view with lessons
│   │       └── assignments/
│   │           ├── page.tsx                ← Course assignments list
│   │           └── [assignmentId]/page.tsx ← Submit + view grade
│   ├── assignments/
│   │   └── page.tsx                        ← All assignments across courses
│   ├── grades/
│   │   ├── page.tsx                        ← Grades overview (all courses)
│   │   └── [courseId]/page.tsx             ← Per-course grade breakdown
│   ├── video/
│   │   ├── page.tsx                        ← Upcoming classes + recordings
│   │   └── [sessionId]/page.tsx            ← Join live class
│   ├── announcements/
│   │   └── page.tsx                        ← All announcements
│   ├── messages/
│   │   ├── page.tsx                        ← Inbox
│   │   ├── sent/page.tsx
│   │   └── [threadId]/page.tsx
│   └── profile/page.tsx                    ← Profile + settings
```

---

## Key Components

### 1. Dashboard (`components/dashboard/`)

```tsx
// DashboardPage shows:
// - EnrolledCoursesWidget: course cards with completion bar
// - UpcomingDeadlinesWidget: assignments due in next 7 days
// - RecentGradesWidget: last 5 graded assignments
// - TodayScheduleWidget: live classes today
// - QuickStatsWidget: GPA estimate, completion rate, streak
```

### 2. Course Module Viewer (`components/courses/ModuleViewer.tsx`)

```tsx
// Left sidebar: module/lesson tree with completion checkmarks
// Main area: lesson content renderer
//   - PDF → react-pdf viewer
//   - Video → HTML5 video player
//   - Link → preview card + external link button
// "Mark as Complete" button at bottom of each lesson
// Next lesson navigation
```

### 3. Assignment Submission (`components/assignments/SubmissionUploader.tsx`)

```tsx
// Status: not_started | in_progress | submitted | graded
// File drop zone (drag & drop, multiple files)
// Submission history (previous attempts)
// After grading: score display + rubric breakdown + feedback
// Due date countdown timer
```

### 4. Grade Tracker (`components/grades/`)

**GradesOverview** — summary cards per course:

- Current grade + estimated final grade
- Progress bar to next letter grade cutoff
- Assignment type breakdown (homework 40%, exams 40%, etc.)

**GradeComparisonChart** — student vs class:

```tsx
// Recharts bar chart:
// - Your score: highlighted bar
// - Class distribution: background histogram
// - Class average: dashed line
// - Your percentile: "You're in the top 30%" badge
```

### 5. Student Progress Widget (`components/progress/`)

- Completion bar per course
- Activity calendar (GitHub-style heatmap)
- Streak counter (consecutive days with activity)
- Time-on-task breakdown (chart: hours per course)

### 6. Video Join Interface (`components/video/JoinSession.tsx`)

```tsx
// Pre-join check: camera/mic permissions
// Session details: title, instructor, start time, duration
// Join button (disabled until 10 min before start — shows countdown)
// After joining: embed WebRTC interface or external link
// Post-session: recording available notification
```

---

## API Integration

```typescript
// lib/api/student.ts
export const studentApi = {
  // Courses
  getEnrolledCourses: () => get('/students/me/courses'),
  getCourseModules: (courseId: string) => get(`/courses/${courseId}/modules`),
  markLessonComplete: (courseId: string, lessonId: string) =>
    patch(`/students/me/courses/${courseId}/progress`, { lessonId }),

  // Assignments
  getAssignments: (params) => get('/assignments', { params }),
  submitAssignment: (assignmentId: string, files: File[]) => {
    const form = new FormData()
    files.forEach((f) => form.append('files', f))
    return post(`/assignments/${assignmentId}/submissions`, form)
  },

  // Grades
  getGrades: (courseId?: string) => get('/students/me/grades', { params: { courseId } }),
  getGradeEstimate: (courseId: string) => get(`/students/me/grade-estimate/${courseId}`),
  getGradeDistribution: (courseId: string, assignmentId?: string) =>
    get(`/metrics/grades/distribution/${courseId}`, { params: { assignmentId } }),

  // Progress
  getProgress: () => get('/metrics/students/me'),
  getTimeOnTask: () => get('/metrics/students/me/time-on-task'),
}
```

---

## Playwright Tests (`tests/e2e/student/`)

```typescript
// tests/e2e/student/courses.spec.ts
test('View enrolled course and mark lesson complete', async ({ page }) => {
  await loginAsStudent(page);
  await page.click('text=Introduction to Python');
  await page.click('text=Week 1: Variables');
  await page.click('button:has-text("Mark as Complete")');
  await expect(page.getByTestId('lesson-completed-Week-1-Variables')).toBeVisible();
});

// tests/e2e/student/assignments.spec.ts
test('Submit assignment file', async ({ page }) => {
  await loginAsStudent(page);
  await page.goto('/assignments');
  await page.click('text=Python Homework 1');
  await page.setInputFiles('input[type="file"]', 'tests/fixtures/homework.pdf');
  await page.click('button:has-text("Submit")');
  await expect(page.getByText('Submitted')).toBeVisible();
});

// tests/e2e/student/grades.spec.ts
test('View grade and class comparison', async ({ page }) => { ... });

// tests/e2e/student/video.spec.ts
test('Join live class when enabled', async ({ page }) => { ... });
```

---

## Files to Create/Modify

- [REWRITE] `frontend/student/app/` — complete app directory restructure
- [NEW] `frontend/student/components/assignments/SubmissionUploader.tsx`
- [NEW] `frontend/student/components/grades/GradeComparisonChart.tsx`
- [NEW] `frontend/student/components/courses/ModuleViewer.tsx`
- [NEW] `frontend/student/components/progress/ActivityCalendar.tsx`
- [NEW] `frontend/student/components/video/JoinSession.tsx`
- [NEW] `frontend/student/lib/api/`
- [NEW] `tests/e2e/student/`
