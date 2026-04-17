# Slate LMS - Project Map

## Overview

Slate is a Learning Management System (LMS) built with a microservices architecture. The backend uses gRPC for inter-service communication, with an API Gateway translating HTTP REST to gRPC. The frontend consists of three Next.js applications (student, provider/instructor, admin).

## Service Inventory

| Service                      | Language/Framework      | Docker Service Name          | HTTP Port | gRPC Port | Metrics Port      | Database                                        |
| ---------------------------- | ----------------------- | ---------------------------- | --------- | --------- | ----------------- | ----------------------------------------------- |
| API Gateway                  | Rust / Axum + Tonic     | `api-gateway`                | 8080      | -         | -                 | -                                               |
| User Auth Service            | Go / gRPC               | `user-auth-service`          | 8081      | 50051     | 9090 (host: 9091) | PostgreSQL (`postgres`, port 5435)              |
| Course Service               | TypeScript / NestJS     | `course-service`             | 3001      | 50052     | 9090 (host: 9094) | MongoDB (`mongodb`, port 27017)                 |
| Content Management Service   | Rust / Tonic            | `content-management-service` | 8082      | 50054     | 9092 (host: 9096) | PostgreSQL (`postgres-cms`, port 5434)          |
| Assignment Grading Service   | Go / gRPC               | `assignment-grading-service` | 8083      | 50053     | 9090 (host: 9093) | PostgreSQL (`assignment-grading-db`, port 5433) |
| Student Frontend             | TypeScript / Next.js 14 | (local dev only)             | 3000      | -         | -                 | -                                               |
| Provider/Instructor Frontend | TypeScript / Next.js 14 | (local dev only)             | 3002      | -         | -                 | -                                               |
| Admin Frontend               | TypeScript / Next.js 14 | (local dev only)             | 3003      | -         | -                 | -                                               |

## Port Allocation Map

### Host Ports (exposed by docker-compose)

| Port  | Service                            | Protocol   |
| ----- | ---------------------------------- | ---------- |
| 3000  | Grafana / Student Frontend (local) | HTTP       |
| 3001  | Course Service HTTP                | HTTP       |
| 3002  | Provider Frontend (local dev)      | HTTP       |
| 3003  | Admin Frontend (local dev)         | HTTP       |
| 3100  | Loki                               | HTTP       |
| 3200  | Tempo                              | HTTP       |
| 4317  | Tempo OTLP                         | gRPC       |
| 4318  | Tempo OTLP                         | HTTP       |
| 5433  | Assignment Grading PostgreSQL      | PostgreSQL |
| 5434  | Content Management PostgreSQL      | PostgreSQL |
| 5435  | User Auth PostgreSQL               | PostgreSQL |
| 6379  | Redis                              | Redis      |
| 8080  | API Gateway                        | HTTP       |
| 8081  | User Auth Service HTTP             | HTTP       |
| 8082  | Content Management Service HTTP    | HTTP       |
| 8083  | Assignment Grading Service HTTP    | HTTP       |
| 9000  | MinIO S3 API                       | HTTP       |
| 9001  | MinIO Console                      | HTTP       |
| 9090  | Prometheus                         | HTTP       |
| 9091  | User Auth Service Metrics          | HTTP       |
| 9093  | Assignment Grading Service Metrics | HTTP       |
| 9094  | Course Service Metrics             | HTTP       |
| 9096  | Content Management Service Metrics | HTTP       |
| 9097  | Kafka (host-mapped from 9092)      | Kafka      |
| 9200  | ElasticSearch                      | HTTP       |
| 14268 | Tempo Jaeger Ingest                | HTTP       |
| 27017 | MongoDB                            | MongoDB    |
| 50051 | User Auth Service                  | gRPC       |
| 50052 | Course Service                     | gRPC       |
| 50053 | Assignment Grading Service         | gRPC       |
| 50054 | Content Management Service         | gRPC       |

## Tech Stack

### Backend

- **API Gateway**: Rust, Axum, Tonic (gRPC), tower-http
- **User Auth Service**: Go, gRPC, PostgreSQL, Redis, JWT, bcrypt, OpenTelemetry
- **Course Service**: TypeScript, NestJS, gRPC (@grpc/grpc-js), MongoDB (Mongoose), OpenTelemetry
- **Content Management Service**: Rust, Tonic (gRPC), SQLx (PostgreSQL), MinIO (S3), ElasticSearch, Redis (job queue), FFmpeg (transcoding)
- **Assignment Grading Service**: Go, gRPC, PostgreSQL, Kafka (event publishing), OpenTelemetry

### Frontend

- **All apps**: Next.js 14, React 18, TypeScript, Tailwind CSS, Radix UI primitives
- **State management**: Zustand (admin), TanStack React Query
- **Shared package**: `frontend/shared` (types, utilities)

### Infrastructure

- **Databases**: PostgreSQL 15, MongoDB 7, Redis 7
- **Object Storage**: MinIO (S3-compatible)
- **Search**: ElasticSearch 8.11
- **Message Broker**: Kafka (Confluent 7.5) + Zookeeper
- **Observability**: Prometheus, Grafana, Tempo (traces), Loki (logs), Promtail

## Database Schemas

### PostgreSQL - User Auth (`postgres` container, db: `userauth`)

Tables (from migrations):

- `users` - id, email, password_hash, first_name, last_name, phone, is_active, timezone, avatar_url, bio, auth_method
- `roles` - id, name, description, permissions (text[])
- `user_roles` - user_id, role_id (junction table)
- `oauth_providers` - id, user_id, provider, provider_user_id, access_token, refresh_token, token_expiry, provider_type
- `mfa_configs` - (from migration 002) MFA/2FA configuration
- `user_groups` - (from migration 002) group management
- `group_members` - (from migration 002) group membership
- `parent_child_links` - (from migration 002) parent-child account relationships
- `saml_configs` - SAML provider configuration (enhanced in migration 005)
- `saml_metadata_cache` - cached SAML IdP metadata

Default roles: `admin`, `user`, `manager`

### PostgreSQL - Assignment Grading (`assignment-grading-db` container, db: `assignment_grading`)

Tables:

- `assignments` - id (UUID), course_id, title, description, max_points, due_date, late_penalty_percent, max_late_days
- `submissions` - id (UUID), assignment_id, student_id, file_path, submitted_at, status, is_late, days_late (unique per student+assignment)
- `grades` - id (UUID), submission_id, student_id, assignment_id, score, adjusted_score, feedback, status (draft/published), graded_by

### PostgreSQL - Content Management (`postgres-cms` container, db: `cms`)

Tables:

- `modules` - id (UUID), course_id, name, description, display_order, created_by
- `lessons` - id (UUID), module_id (FK), name, description, display_order
- `resources` - id (UUID), lesson_id (FK), name, description, content_type, file_size, storage_key, manifest_url, duration_seconds, published, downloadable, copyright_setting, display_order
- `upload_sessions` - id (UUID), user_id, filename, content_type, total_size, chunk_size, total_chunks, uploaded_chunks, storage_key, status, lesson_id
- `progress_tracking` - id (UUID), student_id, resource_id (FK), completed, completed_at
- `transcoding_jobs` - id (UUID), resource_id (FK), status, input/output keys, error_message
- `download_tracking` - id (UUID), student_id, resource_id (FK), downloaded_at

### MongoDB - Course Service (db: `courses`)

Collections managed by NestJS/Mongoose (schema defined in code, not migration files):

- Courses, Enrollments, CourseTemplates, Sections, CrossListings, Prerequisites

## Proto / gRPC Services

All proto files are in `/proto/`:

| Proto File           | Package      | Services                                                                                                                                            |
| -------------------- | ------------ | --------------------------------------------------------------------------------------------------------------------------------------------------- |
| `user.proto`         | `user`       | `UserService` - Auth (Login, Register, RefreshToken, ValidateToken, Logout), User CRUD, Profile, Roles/RBAC, OAuth, SAML, MFA, Groups, Parent-Child |
| `auth.proto`         | `auth`       | `AuthService` - ValidateToken, HealthCheck (thin wrapper used by API Gateway)                                                                       |
| `course.proto`       | `course`     | `CourseService` - Course CRUD, Publishing, Enrollment, Templates, Prerequisites, Co-teaching, Sections, Cross-listing, HealthCheck                  |
| `content.proto`      | `content`    | `ContentService`, `UploadService`, `StreamingService`, `ProgressService`, `SearchService`, `DownloadService`                                        |
| `assignment.proto`   | `assignment` | `AssignmentService`, `SubmissionService`, `GradingService`, `GradebookService`                                                                      |
| `service_auth.proto` | `gateway`    | `ServiceAuthPolicy` - Dynamic auth policy discovery (each backend implements this)                                                                  |

## API Endpoint Map

The API Gateway (port 8080) translates HTTP REST to gRPC. Routes are defined in `config/gateway-config.yaml`.

### Authentication (public - no auth required)

- `POST /api/auth/login` -> `user.UserService/Login`
- `POST /api/auth/register` -> `user.UserService/Register`
- `POST /api/auth/refresh` -> `user.UserService/RefreshToken`
- `POST /api/auth/validate` -> `user.UserService/ValidateToken`
- `POST /api/auth/logout` -> `user.UserService/Logout`

### OAuth / SAML (public)

- `POST /auth/oauth/authorize` -> `user.UserService/GetOAuthAuthorizationURL`
- `GET  /auth/oauth/callback` -> `user.UserService/HandleOAuthCallback`
- `POST /auth/saml/login` -> `user.UserService/GetSAMLAuthRequest`
- `POST /auth/saml/acs` -> `user.UserService/HandleSAMLAssertion`
- `GET  /auth/saml/metadata` -> `user.UserService/GetSAMLMetadata`

### MFA

- `POST /api/mfa/setup` -> `user.UserService/SetupMFA`
- `POST /api/mfa/verify` -> `user.UserService/VerifyMFA`
- `POST /api/mfa/disable` -> `user.UserService/DisableMFA`
- `GET  /api/mfa/status` -> `user.UserService/GetMFAStatus`
- `POST /api/mfa/validate` -> `user.UserService/ValidateMFACode`

### Users

- `GET /api/users/:user_id` -> `user.UserService/GetUser`
- Other CRUD operations auto-discovered

### Groups

- `POST   /api/groups/:id/members` -> `user.UserService/AddGroupMember`
- `DELETE /api/groups/:id/members/:user_id` -> `user.UserService/RemoveGroupMember`
- `GET    /api/groups/:id/members` -> `user.UserService/GetGroupMembers`
- `GET    /api/users/:id/groups` -> `user.UserService/GetUserGroups`

### Courses

- `POST   /api/courses` -> `course.CourseService/CreateCourse`
- `GET    /api/courses/:id` -> `course.CourseService/GetCourse`
- `PUT    /api/courses/:id` -> `course.CourseService/UpdateCourse`
- `DELETE /api/courses/:id` -> `course.CourseService/DeleteCourse`
- `GET    /api/courses` -> `course.CourseService/ListCourses`
- `POST   /api/courses/:id/publish` -> `course.CourseService/PublishCourse`
- `POST   /api/courses/:id/enroll` -> `course.CourseService/SelfEnroll`
- `POST   /api/courses/:id/students` -> `course.CourseService/InstructorAddStudent`
- `GET    /api/courses/:id/roster` -> `course.CourseService/GetCourseRoster`
- `GET    /api/students/:id/enrollments` -> `course.CourseService/GetStudentEnrollments`

### Assignments

- `POST   /api/assignments` -> `assignment.AssignmentService/CreateAssignment`
- `GET    /api/assignments/:id` -> `assignment.AssignmentService/GetAssignment`
- `PUT    /api/assignments/:id` -> `assignment.AssignmentService/UpdateAssignment`
- `DELETE /api/assignments/:id` -> `assignment.AssignmentService/DeleteAssignment`
- `GET    /api/assignments` -> `assignment.AssignmentService/ListAssignments`
- `POST   /api/assignments/:assignment_id/submissions` -> `assignment.SubmissionService/SubmitAssignment`
- `POST   /api/grades/:id/publish` -> `assignment.GradingService/PublishGrade`
- `GET    /api/students/:student_id/gradebook` -> `assignment.GradebookService/GetStudentGradebook`
- `GET    /api/courses/:course_id/gradebook` -> `assignment.GradebookService/GetCourseGradebook`
- `GET    /api/assignments/:assignment_id/statistics` -> `assignment.GradebookService/GetGradeStatistics`
- `GET    /api/courses/:course_id/gradebook/export` -> `assignment.GradebookService/ExportGrades`

### Content Management

- Module/Lesson/Resource CRUD auto-discovered
- `GET  /api/content/structure` -> `content.ContentService/GetContentStructure`
- `POST /api/content/reorder` -> `content.ContentService/ReorderContent`
- `POST /api/content/publish` -> `content.ContentService/PublishContent`
- `POST /api/content/upload/initiate` -> `content.UploadService/InitiateUpload`
- `POST /api/content/upload/chunk` -> `content.UploadService/UploadChunk`
- `POST /api/content/upload/complete` -> `content.UploadService/CompleteUpload`
- `GET  /api/content/videos/:id/manifest` -> `content.StreamingService/GetVideoManifest`
- `POST /api/content/videos/:id/position` -> `content.StreamingService/UpdatePlaybackPosition`
- `POST /api/content/progress/complete` -> `content.ProgressService/MarkComplete`
- `GET  /api/content/progress` -> `content.ProgressService/GetProgress`
- `GET  /api/content/search` -> `content.SearchService/SearchContent`
- `POST /api/content/download` -> `content.DownloadService/GenerateDownloadUrl`

### Health Checks (public)

- `GET /api/health/auth` -> `auth.AuthService/HealthCheck`
- `GET /api/health/courses` -> `course.CourseService/HealthCheck`
- `GET /api/health/content` -> `content.ContentService/HealthCheck`
- `GET /api/health/assignments` -> `assignment.AssignmentService/HealthCheck`

## How to Run

### Full stack (Docker)

```bash
docker-compose up --build
```

This starts all backend services, databases, and observability stack. Frontend apps run locally.

### Individual Frontend Apps (local dev)

```bash
# Student portal (port 3000)
cd frontend/student && npm install && npm run dev

# Provider/Instructor portal (port 3002)
cd frontend/provider && npm install && npm run dev

# Admin dashboard (port 3003)
cd frontend/admin && npm install && npm run dev
```

### Individual Backend Services (local dev)

```bash
# API Gateway (Rust)
cd services/api-gateway && cargo run

# User Auth Service (Go)
cd services/user-auth-service && go run cmd/server/main.go

# Course Service (NestJS)
cd services/course-service && npm install && npm run start:dev

# Content Management Service (Rust)
cd services/content-management-service && cargo run

# Assignment Grading Service (Go)
cd services/assignment-grading-service && go run cmd/server/main.go
```

## Key Environment Variables

### API Gateway

- `GATEWAY_SERVER_HOST` / `GATEWAY_SERVER_PORT` (default: 0.0.0.0:8080)
- `GATEWAY_AUTH_SERVICE_ENDPOINT` - gRPC endpoint for user-auth-service
- `GATEWAY_OBSERVABILITY_TEMPO_ENDPOINT` - OTLP collector endpoint
- `AUTH_TYPE` - normal, oauth, saml
- `MAX_REQUEST_BODY_SIZE` / `MAX_UPLOAD_BODY_SIZE` - DoS prevention limits

### User Auth Service

- `SERVER_HOST` / `SERVER_PORT` / `GRPC_HOST` / `GRPC_PORT`
- `DB_HOST` / `DB_PORT` / `DB_USER` / `DB_PASSWORD` / `DB_NAME` / `DB_SSLMODE`
- `JWT_SECRET` / `JWT_ACCESS_TOKEN_DURATION` / `JWT_REFRESH_TOKEN_DURATION`
- `REDIS_HOST` / `REDIS_PORT`
- `AUTH_TYPE` - normal, oauth, saml
- `OAUTH_GOOGLE_*` - OAuth provider config
- `SAML_*` - SAML provider config
- `RATE_LIMIT_ENABLED` / `RATE_LIMIT_LOGIN_MAX` / `RATE_LIMIT_LOGIN_WINDOW`
- `OTEL_EXPORTER_OTLP_ENDPOINT`

### Course Service

- `PORT` / `GRPC_HOST` / `GRPC_PORT`
- `MONGO_URI` / `MONGO_DB_NAME`
- `OTEL_EXPORTER_OTLP_ENDPOINT`
- `PROTO_PATH` / `PROTO_DIR`

### Content Management Service

- `SERVER__HOST` / `SERVER__PORT` / `SERVER__GRPC_PORT` / `SERVER__METRICS_PORT`
- `DATABASE__URL` / `DATABASE__MAX_CONNECTIONS`
- `S3__ENDPOINT` / `S3__ACCESS_KEY` / `S3__SECRET_KEY` / `S3__BUCKET` / `S3__REGION`
- `ELASTICSEARCH__URL` / `ELASTICSEARCH__INDEX`
- `REDIS__URL` / `REDIS__QUEUE_NAME`
- `OBSERVABILITY__OTLP_ENDPOINT` / `OBSERVABILITY__SERVICE_NAME`

### Assignment Grading Service

- `SERVER_HOST` / `SERVER_PORT` / `GRPC_HOST` / `GRPC_PORT`
- `DB_HOST` / `DB_PORT` / `DB_USER` / `DB_PASSWORD` / `DB_NAME`
- `KAFKA_ENABLED` / `KAFKA_BROKERS` / `KAFKA_TOPIC`
- `STORAGE_TYPE` / `STORAGE_LOCAL_PATH` / `STORAGE_MAX_SIZE`
- `OTEL_EXPORTER_OTLP_ENDPOINT`

## CORS Configuration

CORS is configured in `config/gateway-config.yaml` under the `cors:` section:

- Enabled for development with specific origins: `http://localhost:3000` (student), `http://localhost:3002` (provider), `http://localhost:3003` (admin)
- Also set via `CORS_ALLOWED_ORIGINS` env var in docker-compose.yml for the api-gateway service
- Allowed headers: `content-type`, `authorization`, `x-request-id`, `x-trace-id`
- The code at `services/api-gateway/src/app/server.rs` checks env var `CORS_ALLOWED_ORIGINS` first, falling back to config file values
- Dev mode (env `DEV_MODE=true` or `ENVIRONMENT=development`) allows all origins

## Architecture Notes

### Auth Flow

1. API Gateway receives HTTP request
2. Gateway checks `config/gateway-config.yaml` public_routes for auth exemption
3. For protected routes, gateway calls `auth.AuthService/ValidateToken` on user-auth-service via gRPC
4. User-auth-service has an `AuthServiceServer` (auth_handler.go) that wraps `UserServiceServer` to serve the `auth.AuthService` proto
5. Gateway also queries each backend's `ServiceAuthPolicy/GetAuthPolicy` for dynamic role-based auth policies
6. Auth context (user_id, roles) is passed downstream

### Gateway Route Discovery

- The gateway supports auto-discovery of gRPC methods following naming conventions
- Explicit route overrides are defined in `config/gateway-config.yaml` under `route_overrides`
- Each backend service implements `gateway.ServiceAuthPolicy/GetAuthPolicy` for dynamic auth

### Content Pipeline

- Chunked upload -> MinIO S3 storage -> Transcoding workers (FFmpeg) -> HLS/DASH streaming
- Search indexing via ElasticSearch
- Progress tracking per student per resource
- Download URL generation with presigned S3 URLs

## Known Issues (Fixed)

### Port 50052 conflict (FIXED)

- content-management-service and course-service both used gRPC port 50052
- Fixed: content-management-service now uses gRPC port 50054
- Updated in: docker-compose.yml, config/gateway-config.yaml, services/content-management-service/src/config/mod.rs

### Admin frontend port 3001 conflict (FIXED)

- Admin Next.js app and course-service HTTP both used port 3001
- Fixed: Admin frontend now uses port 3003
- Updated in: frontend/admin/package.json

### Auth middleware service name

- Investigated: The gateway calls `auth.AuthService/ValidateToken` (from auth.proto)
- The user-auth-service correctly registers both `user.UserService` and `auth.AuthService` servers
- `auth_handler.go` wraps `UserServiceServer` to implement `auth.AuthService/ValidateToken`
- No fix needed - the architecture is correct
