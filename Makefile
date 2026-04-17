.PHONY: dev up down logs ps proto test clean frontend preflight help

# Default target
.DEFAULT_GOAL := help

## dev: Start only infrastructure services (databases, redis, kafka, etc.)
dev:
	docker compose -f docker-compose.yml up -d \
		postgres assignment-grading-db postgres-cms \
		redis mongodb minio elasticsearch \
		zookeeper kafka \
		prometheus tempo loki grafana promtail

## preflight: Run preflight checks (Docker, ports, .env)
preflight:
	@bash scripts/preflight.sh

## up: Run preflight checks, then build and start all services
up:
	@bash scripts/preflight.sh || exit 1
	docker compose up --build -d

## down: Stop all services
down:
	docker compose down

## logs: Tail logs for all services
logs:
	docker compose logs -f

## logs-%: Tail logs for a specific service (e.g., make logs-api-gateway)
logs-%:
	docker compose logs -f $*

## ps: Show running containers
ps:
	docker compose ps

## proto: Regenerate protobuf files for all services that support it
proto:
	@echo "==> Regenerating protobuf files..."
	@if [ -f services/user-auth-service/Makefile ]; then \
		echo "  -> user-auth-service"; \
		$(MAKE) -C services/user-auth-service proto; \
	fi
	@if [ -f services/assignment-grading-service/Makefile ]; then \
		echo "  -> assignment-grading-service"; \
		$(MAKE) -C services/assignment-grading-service proto; \
	fi
	@echo "==> Done."

## test: Run smoke tests
test:
	@if [ -f tests/smoke_test.sh ]; then \
		bash tests/smoke_test.sh; \
	else \
		echo "tests/smoke_test.sh not found"; \
		exit 1; \
	fi

## clean: Stop all services and remove volumes
clean:
	docker compose down -v

## frontend: Install deps and start all frontend dev servers
frontend:
	@echo "==> Starting frontend dev servers..."
	@cd frontend/student  && npm install && npm run dev &
	@cd frontend/provider && npm install && npm run dev &
	@cd frontend/admin    && npm install && npm run dev &
	@echo "Student: http://localhost:3000"
	@echo "Provider: http://localhost:3002"
	@echo "Admin: http://localhost:3003"

## help: Show this help message
help:
	@echo "Slate LMS — Available targets:"
	@echo ""
	@grep -E '^## ' $(MAKEFILE_LIST) | sed 's/^## /  /'
	@echo ""
	@echo "Examples:"
	@echo "  make dev              Start infra only (for local frontend dev)"
	@echo "  make up               Build & start everything"
	@echo "  make logs-api-gateway Tail logs for api-gateway"
	@echo "  make clean            Stop everything and remove volumes"
