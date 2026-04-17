# Slate LMS — Dev Testing Guide

## Quick Start

```bash
# 1. Start all services
docker compose up -d

# 2. Seed mock data (Eastfield University)
chmod +x scripts/seed-dev.sh
./scripts/seed-dev.sh
```

---

## Frontend URLs

| Interface         | URL                   |
| ----------------- | --------------------- |
| Student portal    | http://localhost:3000 |
| Instructor portal | http://localhost:3002 |
| Admin panel       | http://localhost:3003 |
| API gateway       | http://localhost:8080 |

---

## Credentials

### System Admin

| Email             | Password       | Access              |
| ----------------- | -------------- | ------------------- |
| `admin@slate.edu` | `Admin@123456` | Full platform admin |

### Instructors

| Email                             | Password    | Courses         |
| --------------------------------- | ----------- | --------------- |
| `dr.sarah.chen@eastfield.edu`     | `Test@1234` | CS-101, CS-201  |
| `prof.james.wilson@eastfield.edu` | `Test@1234` | WEB-101, DB-101 |

### Students

| Email                         | Password    | Enrolled In             |
| ----------------------------- | ----------- | ----------------------- |
| `alice.johnson@eastfield.edu` | `Test@1234` | CS-101, DB-101          |
| `bob.smith@eastfield.edu`     | `Test@1234` | CS-101, CS-201          |
| `carol.white@eastfield.edu`   | `Test@1234` | CS-101, CS-201, WEB-101 |
| `david.brown@eastfield.edu`   | `Test@1234` | CS-201, WEB-101, DB-101 |
| `emma.davis@eastfield.edu`    | `Test@1234` | WEB-101, DB-101         |

---

## Seeded Courses

| Code    | Title                            | Instructor   | Students           |
| ------- | -------------------------------- | ------------ | ------------------ |
| CS-101  | Introduction to Computer Science | Sarah Chen   | Alice, Bob, Carol  |
| CS-201  | Data Structures and Algorithms   | Sarah Chen   | Bob, Carol, David  |
| WEB-101 | Web Development Fundamentals     | James Wilson | Carol, David, Emma |
| DB-101  | Database Systems                 | James Wilson | Alice, David, Emma |

### Assignments per Course

**CS-101**

- Lab 1: Hello World (20 pts)
- Lab 2: Control Flow (30 pts)
- Project: Number Guessing Game (50 pts) — due 2026-09-30

**CS-201**

- Lab 1: Linked List (40 pts)
- Lab 2: Binary Search Tree (40 pts)
- Project: Pathfinding (80 pts) — due 2026-09-30

**WEB-101**

- Lab 1: HTML Portfolio (25 pts)
- Lab 2: CSS Styling (25 pts)
- Project: Interactive Page (50 pts) — due 2026-09-30

**DB-101**

- Lab 1: SQL Basics (30 pts)
- Lab 2: Joins and Aggregates (30 pts)
- Project: Library System (60 pts) — due 2026-09-30

---

## Test Scenarios

### 1 — Student login and browse courses

```bash
# Login as Alice
curl -s -X POST http://localhost:8080/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"alice.johnson@eastfield.edu","password":"Test@1234"}' | jq .

# List courses (use token from above response)
TOKEN="<access_token>"
curl -s http://localhost:8080/api/courses \
  -H "Authorization: Bearer $TOKEN" | jq .
```

### 2 — Submit an assignment (as student)

```bash
# List assignments for a course
curl -s "http://localhost:8080/api/assignments?course_id=<COURSE_ID>" \
  -H "Authorization: Bearer $TOKEN" | jq .

# Submit an assignment
curl -s -X POST "http://localhost:8080/api/assignments/<ASSIGNMENT_ID>/submissions" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "assignment_id": "<ASSIGNMENT_ID>",
    "student_id": "<YOUR_USER_ID>",
    "file_name": "solution.py",
    "content_type": "text/plain",
    "file_content": "cHJpbnQoJ0hlbGxvIFdvcmxkJykK"
  }' | jq .
```

### 3 — Instructor grades a submission

```bash
# Login as Sarah
TOKEN=$(curl -s -X POST http://localhost:8080/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"dr.sarah.chen@eastfield.edu","password":"Test@1234"}' | jq -r '.access_token')

# Get course gradebook
curl -s "http://localhost:8080/api/courses/<COURSE_ID>/gradebook" \
  -H "Authorization: Bearer $TOKEN" | jq .

# Publish a grade
curl -s -X POST "http://localhost:8080/api/grades/<GRADE_ID>/publish" \
  -H "Authorization: Bearer $TOKEN" | jq .
```

### 4 — Admin manages tenants

```bash
# Login as admin
TOKEN=$(curl -s -X POST http://localhost:8080/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"admin@slate.edu","password":"Admin@123456"}' | jq -r '.access_token')

# List tenants
curl -s http://localhost:8080/api/tenants \
  -H "Authorization: Bearer $TOKEN" | jq .

# Platform metrics
curl -s http://localhost:8080/api/metrics/platform \
  -H "Authorization: Bearer $TOKEN" | jq .
```

### 5 — Token refresh

```bash
curl -s -X POST http://localhost:8080/api/auth/refresh \
  -H "Content-Type: application/json" \
  -d '{"refresh_token":"<REFRESH_TOKEN>"}' | jq .
```

---

## Service Health Checks

```bash
curl -s http://localhost:8080/api/health/auth        | jq .
curl -s http://localhost:8080/api/health/courses     | jq .
curl -s http://localhost:8080/api/health/assignments | jq .
curl -s http://localhost:8080/api/health/content     | jq .
```

---

## Re-seeding

The seed script is idempotent — re-running it skips users that already exist and re-assigns roles safely.

```bash
./scripts/seed-dev.sh
```

To wipe and start fresh:

```bash
docker compose down -v   # removes all volumes
docker compose up -d
./scripts/seed-dev.sh
```

---

## MinIO (Content/Materials)

For uploading course materials manually:

| Setting     | Value                 |
| ----------- | --------------------- |
| Console URL | http://localhost:9001 |
| Access key  | `minioadmin`          |
| Secret key  | `minioadmin`          |
| Bucket      | `content-storage`     |

Use the content upload API after logging in as an instructor:

```bash
# Initiate upload
curl -s -X POST http://localhost:8080/api/content/upload/initiate \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"file_name":"lecture1.pdf","content_type":"application/pdf","file_size":102400}' | jq .
```
