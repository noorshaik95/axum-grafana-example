# Contributing to Slate LMS

## Prerequisites

- Docker Desktop (with Compose v2)
- Node.js >= 20 (frontend local dev)
- Go >= 1.24 (Go service local dev)
- Rust/Cargo (Rust service local dev)

## Quick Start

```bash
cp .env.example .env      # configure environment
make up                   # build and start everything
# Student: http://localhost:3000  Provider: http://localhost:3002  Admin: http://localhost:3003
```

## Port Reference

| Service                    | HTTP | gRPC  | Metrics |
| -------------------------- | ---- | ----- | ------- |
| API Gateway                | 8080 | —     | —       |
| User Auth Service          | 8081 | 50051 | 9091    |
| Course Service             | 3001 | 50052 | 9094    |
| Content Management Service | 8082 | 50054 | 9096    |
| Assignment Grading Service | 8083 | 50053 | 9093    |
| Student Frontend           | 3000 | —     | —       |
| Provider Frontend          | 3002 | —     | —       |
| Admin Frontend             | 3003 | —     | —       |

| Infrastructure      | Port  |
| ------------------- | ----- |
| PostgreSQL (auth)   | 5435  |
| PostgreSQL (assign) | 5433  |
| PostgreSQL (CMS)    | 5434  |
| MongoDB             | 27017 |
| Redis               | 6379  |
| Kafka               | 9097  |
| Elasticsearch       | 9200  |
| MinIO API           | 9000  |
| MinIO Console       | 9001  |
| Grafana             | 3500  |
| Prometheus          | 9090  |
| Tempo               | 3200  |
| Loki                | 3100  |

## Dev Workflow

### Full stack (Docker)

```bash
make up       # build and start everything
make logs     # tail all logs
make down     # stop everything
make clean    # stop and remove volumes
```

### Infrastructure only + local frontends

```bash
make dev        # start databases, redis, kafka, observability
make frontend   # install deps and start all 3 Next.js dev servers
```

### Per-service local run

```bash
# Go services
cd services/user-auth-service && make run
cd services/assignment-grading-service && make run

# Rust services (requires cargo)
cd services/api-gateway && cargo run
cd services/content-management-service && cargo run

# NestJS
cd services/course-service && npm run start:dev

# Frontends
cd frontend/student && npm run dev
cd frontend/provider && npm run dev
cd frontend/admin && npm run dev
```

### Protobuf regeneration

```bash
make proto    # regenerates .pb.go files for Go services
```

## Testing

```bash
# Smoke tests (requires running stack)
make test

# Unit tests per service
cd services/user-auth-service && go test ./...
cd services/assignment-grading-service && go test ./...
cd services/api-gateway && cargo test
cd services/content-management-service && cargo test

# E2E tests (Playwright — requires running stack)
cd tests/e2e && npx playwright test

# Frontend lint + typecheck
cd frontend/student && npm run lint && npx tsc --noEmit
```

## Default Credentials

| Account       | Email           | Password     |
| ------------- | --------------- | ------------ |
| Admin         | admin@slate.edu | Admin@123456 |
| Grafana       | admin           | admin        |
| MinIO Console | minioadmin      | minioadmin   |

## Pre-commit Hooks

This project uses [husky](https://typicode.github.io/husky/) + [lint-staged](https://github.com/lint-staged/lint-staged) to run checks before each commit.

### What runs on commit

- **pre-commit**: `lint-staged` runs TypeScript type-checking on any staged `.ts`/`.tsx` files in the matching frontend project, and `prettier --check` on staged `.js`, `.ts`, `.tsx`, `.json`, and `.md` files.
- **commit-msg**: Validates that the commit message follows [Conventional Commits](https://www.conventionalcommits.org/) format. Must start with one of: `feat`, `fix`, `chore`, `refactor`, `docs`, `test`, `style`, `ci`.

### Setup

After cloning, run `npm install` at the repo root to install husky and enable the hooks automatically via the `prepare` script.

### Skipping hooks

If you need to bypass hooks temporarily (e.g., WIP commits):

```bash
git commit --no-verify -m "chore: WIP"
```

Use this sparingly — CI will still enforce these checks.

## Architecture

See [CLAUDE_README.md](./CLAUDE_README.md) for detailed architecture, service boundaries, and data flow.

## Troubleshooting

### Port conflicts

Run `make up` — it calls `scripts/preflight.sh` automatically and reports which ports are in use. To find what's using a port:

```bash
lsof -i :8080
```

### .env missing

```bash
cp .env.example .env
```

Edit `.env` and set `JWT_SECRET` to something other than the default placeholder.

### Docker Compose version

Slate requires Compose v2 (the `docker compose` plugin, not standalone `docker-compose`). Check with:

```bash
docker compose version
```

If it says "command not found", install the Docker Compose plugin or upgrade Docker Desktop.

### Service won't start

Check logs for the specific service:

```bash
make logs-api-gateway
make logs-user-auth-service
```

### Database issues

To reset all data and start fresh:

```bash
make clean    # removes all volumes
make up       # rebuild from scratch
```
