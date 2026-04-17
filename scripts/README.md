# Slate Scripts

This directory contains utility scripts for managing the Slate platform.

## Quick Start

### 1. Install Dependencies

First, install all service dependencies:

```bash
./scripts/install-dependencies.sh
```

This script will:

- Check for required tools (Rust, Go, Node.js, Docker)
- Install dependencies for all Rust services (API Gateway, Content Management)
- Install dependencies for all Go services (User Auth, Assignment Grading)
- Install dependencies for all Node.js services (Course Service, Frontend)

### 2. Manage Services

Use the service management script to control Docker services:

```bash
# Start all services
./scripts/manage-services.sh start

# Start only infrastructure (for local development)
./scripts/manage-services.sh dev

# View service status
./scripts/manage-services.sh status

# View logs
./scripts/manage-services.sh logs

# Restart a specific service
./scripts/manage-services.sh restart api-gateway

# Rebuild a service after code changes
./scripts/manage-services.sh rebuild user-auth-service

# Stop all services
./scripts/manage-services.sh stop

# Clean everything (removes volumes)
./scripts/manage-services.sh clean
```

## Service Management Commands

### Available Commands

- `start` - Start all services or a specific service
- `stop` - Stop all services or a specific service
- `restart` - Restart all services or a specific service
- `rebuild` - Rebuild and restart services (use after code changes)
- `logs` - Show logs for services
- `status` - Show status of all running services
- `clean` - Stop and remove all containers, networks, and volumes
- `dev` - Start only infrastructure services (DBs, observability)

### Service Names

You can target specific services:

- `api-gateway`
- `user-auth-service`
- `course-service`
- `assignment-grading-service`
- `content-management-service`
- `postgres`
- `redis`
- `mongodb`
- `minio`
- `elasticsearch`
- `prometheus`
- `tempo`
- `loki`
- `grafana`
- `kafka`

### Examples

```bash
# Start only the API Gateway
./scripts/manage-services.sh start api-gateway

# Rebuild the User Auth Service after making changes
./scripts/manage-services.sh rebuild user-auth-service

# View logs for the Course Service
./scripts/manage-services.sh logs course-service

# Start infrastructure for local development
./scripts/manage-services.sh dev
```

## Development Workflow

### Local Development (Services Running Locally)

1. Start infrastructure only:

   ```bash
   ./scripts/manage-services.sh dev
   ```

2. Run services locally using their native tools:

   ```bash
   # API Gateway (Rust)
   cd services/api-gateway
   cargo run

   # User Auth Service (Go)
   cd services/user-auth-service
   go run cmd/server/main.go

   # Course Service (Node.js)
   cd services/course-service
   npm run start:dev
   ```

### Full Docker Development

1. Start all services:

   ```bash
   ./scripts/manage-services.sh start
   ```

2. Make code changes

3. Rebuild specific service:
   ```bash
   ./scripts/manage-services.sh rebuild <service-name>
   ```

## Service URLs

After starting services, access them at:

- **API Gateway**: http://localhost:8080
- **Grafana**: http://localhost:3000 (admin/admin)
- **Prometheus**: http://localhost:9090
- **MinIO Console**: http://localhost:9001 (minioadmin/minioadmin)
- **Tempo**: http://localhost:3200
- **Loki**: http://localhost:3100

## Database Ports

- **PostgreSQL (Auth)**: localhost:5432
- **PostgreSQL (CMS)**: localhost:5433
- **MongoDB**: localhost:27017
- **Redis**: localhost:6379
- **Elasticsearch**: localhost:9200
- **Kafka**: localhost:9092

## Testing Scripts

Additional testing scripts are available:

- `test_auth_endpoints.sh` - Test authentication endpoints
- `test_auth_integration.sh` - Test auth service integration
- `test_gateway_communication.sh` - Test gateway communication
- `test_service_routing.sh` - Test service routing
- `test_distributed_tracing.sh` - Test distributed tracing
- `verify_trace_propagation.sh` - Verify trace propagation
- `load_test_users.sh` - Run load tests

## Troubleshooting

### Services won't start

1. Check if ports are already in use:

   ```bash
   lsof -i :8080  # API Gateway
   lsof -i :5432  # PostgreSQL
   ```

2. Clean and restart:
   ```bash
   ./scripts/manage-services.sh clean
   ./scripts/manage-services.sh start
   ```

### Dependencies not installing

Make sure you have the required tools installed:

- Rust: https://rustup.rs/
- Go: https://go.dev/doc/install
- Node.js: https://nodejs.org/
- Docker: https://docs.docker.com/get-docker/

### Service build failures

1. Clean Docker build cache:

   ```bash
   docker builder prune -a
   ```

2. Rebuild without cache:
   ```bash
   ./scripts/manage-services.sh rebuild
   ```

## Notes

- The `dev` command starts only infrastructure services, allowing you to run application services locally for faster development iteration
- Use `rebuild` after making code changes to services
- Use `clean` to reset everything (warning: this removes all data)
- Logs can be followed in real-time with the `logs` command (Ctrl+C to exit)
