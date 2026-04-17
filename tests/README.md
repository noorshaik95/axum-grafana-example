# Slate LMS Test Suite

## Test Inventory

| Suite                                                               | Framework    | Tests      | What It Covers                                                                                                                                                                                          |
| ------------------------------------------------------------------- | ------------ | ---------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `tests/smoke_test.sh`                                               | Bash/curl    | ~20 checks | Service health, connectivity, auth flow, CORS, rate limiting                                                                                                                                            |
| `tests/api_integration_test.sh`                                     | Bash/curl    | ~15 checks | Registration, login, JWT auth, token refresh, course creation, error handling                                                                                                                           |
| `tests/e2e/student/auth.spec.ts`                                    | Playwright   | 4          | Redirect to login, login page rendering, authenticated dashboard, logout                                                                                                                                |
| `tests/e2e/student/dashboard.spec.ts`                               | Playwright   | 7          | Welcome heading, stat cards, My Courses section, Upcoming, grid layout, sidebar nav, quick actions                                                                                                      |
| `tests/e2e/student/courses.spec.ts`                                 | Playwright   | 6          | Heading, subtitle, search bar, filter tabs, course grid/empty state, search filtering                                                                                                                   |
| `tests/e2e/student/grades.spec.ts`                                  | Playwright   | 6          | Heading, subtitle, summary stats (GPA/Credits/Graded), stats grid, grades table, empty state                                                                                                            |
| `tests/e2e/admin/auth.spec.ts`                                      | Playwright   | 5          | Login page, valid/invalid credentials, dashboard stat cards, sidebar logout                                                                                                                             |
| `tests/e2e/admin/users.spec.ts`                                     | Playwright   | 7          | Heading, stat cards, table with columns, user rows, search, invite button, role badges                                                                                                                  |
| `tests/e2e/provider/courses.spec.ts`                                | Playwright   | 9          | Welcome heading, stat cards, My Courses, courses page, create button, grid/empty, create form, sidebar nav, sign out                                                                                    |
| `frontend/shared/lib/api/__tests__/client.test.ts`                  | Jest         | 11         | ApiClient: headers, auth, JSON parsing, 204, ApiError, HTTP methods                                                                                                                                     |
| `frontend/shared/lib/api/__tests__/auth.test.ts`                    | Jest         | 14         | Login, register, logout (token cleanup), getProfile                                                                                                                                                     |
| `frontend/shared/lib/api/__tests__/courses.test.ts`                 | Jest         | 10         | listCourses filters, getCourse, enrollInCourse                                                                                                                                                          |
| `frontend/shared/lib/api/__tests__/hooks.test.ts`                   | Jest         | 6          | useProfile, useCourses, useLogin (success + error)                                                                                                                                                      |
| `services/course-service/**/*.spec.ts` (8 suites)                   | Jest/NestJS  | 107        | CourseService (CRUD, templates, prerequisites, co-instructors, sections, cross-listing), EnrollmentService (self-enroll, instructor-add, roster, student enrollments), CourseController (gRPC handlers) |
| `services/assignment-grading-service/internal/service/*_test.go`    | Go/testify   | 50+        | CreateGrade, UpdateGrade, PublishGrade, GetGrade, GetStudentGradebook, GetCourseGradebook, GradeStatistics, ExportGrades, LatePolicyCalculator, SubmissionService                                       |
| `services/assignment-grading-service/internal/models/*_test.go`     | Go/testify   | ~15        | Assignment, Grade, Submission model validation                                                                                                                                                          |
| `services/assignment-grading-service/internal/repository/*_test.go` | Go/testify   | ~20        | Repository CRUD operations                                                                                                                                                                              |
| `services/assignment-grading-service/pkg/**/*_test.go`              | Go/testify   | ~15        | Rate limiter, file storage (sanitization, content types)                                                                                                                                                |
| `services/api-gateway/tests/`                                       | Rust/reqwest | 6          | Registration, login, list users, create user auth, JWT validation, gRPC error mapping                                                                                                                   |
| `services/user-auth-service/tests/integration/`                     | Go           | 3+         | Auth flow (register/login/validate), token refresh, inactive user blocking                                                                                                                              |

**Total: ~300+ automated tests across all layers.**

## Prerequisites

- All backend services running (via `docker-compose up` or individually)
- `curl` installed
- `jq` recommended for API integration tests (falls back to grep-based parsing)
- `grpcurl` optional for gRPC connectivity checks in smoke tests

## Running Tests

### Smoke Tests

Verifies basic connectivity and health of all platform services.

```bash
./tests/smoke_test.sh

# Override service URLs:
GATEWAY_URL=http://my-gateway:8080 ./tests/smoke_test.sh
```

### API Integration Tests

End-to-end API flow testing through the gateway.

```bash
./tests/api_integration_test.sh

# Against a custom API URL:
./tests/api_integration_test.sh http://staging-api:8080
```

### Playwright E2E Tests

Browser-based end-to-end tests for all 3 frontend apps.

**Prerequisites:**

- All 3 frontend dev servers running (student:3000, admin:3003, provider:3002)
- API gateway running on port 8080 (for auth token acquisition)
- Seeded admin user: `admin@slate.edu` / `Admin@123456`

**Setup (one time):**

```bash
cd tests/e2e
npm install
npx playwright install    # downloads browser binaries
```

**Usage:**

```bash
cd tests/e2e

# Run all tests across all 3 portals
npm test

# Run with headed browser (watch tests run)
npm run test:headed

# Run with interactive UI mode
npm run test:ui

# Run tests for a single portal
npm run test:student
npm run test:admin
npm run test:provider

# Debug a specific test
npx playwright test --debug student/auth.spec.ts
```

**Reports:** After a run, view the HTML report with `npm run report`.

### Frontend Shared API Client (Jest)

```bash
cd frontend/shared
npm test                    # all tests
npm test -- --coverage      # with coverage
```

### Course Service (NestJS/Jest)

```bash
cd services/course-service
npm test                    # all 107 tests
npm test -- --coverage      # with coverage report
```

### Assignment Grading Service (Go)

```bash
cd services/assignment-grading-service
go test ./...               # all tests
go test ./internal/service/ -v   # service layer with verbose output
```

### API Gateway (Rust)

```bash
cd services/api-gateway
cargo test                    # unit tests
cargo test -- --ignored       # integration tests (requires running services)
```

### User Auth Service (Go)

```bash
cd services/user-auth-service
go test ./...                                    # all unit tests
go test ./tests/integration/ -v                  # integration tests (requires postgres + redis)
```

## Service Port Reference

| Service                    | HTTP Port | gRPC Port |
| -------------------------- | --------- | --------- |
| API Gateway                | 8080      | -         |
| User Auth Service          | 8081      | 50051     |
| Course Service             | 3001      | 50052     |
| Content Management Service | 8082      | 50055     |
| Assignment Grading Service | 8083      | 50053     |
| Student Frontend           | 3000      | -         |
| Provider Frontend          | 3002      | -         |
| Admin Frontend             | 3003      | -         |

## Test Structure

```
tests/
  smoke_test.sh              # Service health + connectivity
  api_integration_test.sh    # End-to-end API flows
  e2e/
    playwright.config.ts     # 3 projects: student, admin, provider
    fixtures/auth.ts         # Auth fixture -- injects JWT via localStorage
    helpers/api.ts           # Direct API calls for test setup/teardown
    student/
      auth.spec.ts           # Login redirect, login page, authenticated dashboard, logout
      dashboard.spec.ts      # Welcome heading, stat cards, My Courses, sidebar nav, quick actions
      courses.spec.ts        # Course list, search, filter tabs, grid/empty state
      grades.spec.ts         # GPA/credits/graded stats, grades table, empty state
    admin/
      auth.spec.ts           # Login page rendering, valid/invalid creds, dashboard, sidebar logout
      users.spec.ts          # Users table, stat cards, search, invite button, role badges
    provider/
      courses.spec.ts        # Dashboard stats, My Courses, course management, create form, sidebar
frontend/shared/lib/api/__tests__/
  client.test.ts             # ApiClient HTTP methods, headers, error handling
  auth.test.ts               # Login, register, logout, getProfile
  courses.test.ts            # listCourses, getCourse, enrollInCourse
  hooks.test.ts              # useProfile, useCourses, useLogin hooks
services/course-service/src/
  course/course.service.spec.ts      # CourseService CRUD, templates, prereqs, sections, cross-listing
  course/course.controller.spec.ts   # CourseController gRPC handler mapping
  enrollment/enrollment.service.spec.ts  # EnrollmentService self-enroll, instructor-add, roster
services/assignment-grading-service/internal/
  service/grading_service_test.go    # CreateGrade, UpdateGrade, PublishGrade, GetGrade
  service/gradebook_service_test.go  # Gradebook, statistics, CSV export, letter grades
  service/assignment_service_test.go # Assignment CRUD
  service/submission_service_test.go # Submit, get, list submissions
  service/late_policy_calculator_test.go  # Late penalty calculations
```
