#!/usr/bin/env bash
set -euo pipefail

# Slate LMS — Preflight checks
# Run before `make up` to catch common issues early.

PASS="\033[32m✓\033[0m"
WARN="\033[33m⚠\033[0m"
FAIL="\033[31m✗\033[0m"

errors=0

pass()  { echo -e " $PASS  $1"; }
warn()  { echo -e " $WARN  $1"; }
fail()  { echo -e " $FAIL  $1"; errors=$((errors + 1)); }

echo ""
echo "Slate LMS — Preflight Checks"
echo "============================="
echo ""

# ── Docker daemon ────────────────────────────────────────────
echo "Docker:"
if docker info > /dev/null 2>&1; then
  pass "Docker daemon is running"
else
  fail "Docker daemon is not running — start Docker Desktop or dockerd"
fi

# ── Docker Compose v2 ───────────────────────────────────────
if docker compose version > /dev/null 2>&1; then
  compose_ver=$(docker compose version --short 2>/dev/null || echo "unknown")
  pass "Docker Compose v2 found ($compose_ver)"
else
  fail "docker compose (v2) not found — install Docker Compose plugin"
fi

echo ""

# ── Port availability ────────────────────────────────────────
echo "Ports:"
PORTS=(
  "8080:api-gateway"
  "8081:user-auth-service"
  "8082:content-management-service"
  "8083:assignment-grading-service"
  "3000:student-frontend"
  "3001:course-service"
  "3002:provider-frontend"
  "3003:admin-frontend"
  "5432:postgres (internal)"
  "5433:assignment-grading-db"
  "5434:postgres-cms"
  "5435:postgres (user-auth)"
  "27017:mongodb"
  "6379:redis"
  "9200:elasticsearch"
  "3500:grafana"
)

for entry in "${PORTS[@]}"; do
  port="${entry%%:*}"
  label="${entry#*:}"
  if lsof -i :"$port" -sTCP:LISTEN > /dev/null 2>&1; then
    fail "Port $port in use ($label)"
  else
    pass "Port $port free ($label)"
  fi
done

echo ""

# ── .env file ────────────────────────────────────────────────
echo "Environment:"
if [ -f .env ]; then
  if [ -s .env ]; then
    pass ".env exists and is not empty"
  else
    fail ".env exists but is empty — copy from .env.example"
  fi

  # Check JWT_SECRET
  if grep -q "JWT_SECRET" .env 2>/dev/null; then
    jwt_val=$(grep "JWT_SECRET" .env | head -1 | cut -d'=' -f2-)
    if [ "$jwt_val" = "your-super-secret-jwt-key-change-in-production" ]; then
      warn "JWT_SECRET is still the default placeholder — change it for production"
    else
      pass "JWT_SECRET has been customized"
    fi
  fi
else
  fail ".env file not found — run: cp .env.example .env"
fi

echo ""

# ── Optional tool versions (warn only) ──────────────────────
echo "Optional tools:"

if command -v node > /dev/null 2>&1; then
  node_ver=$(node -v | sed 's/v//')
  node_major=$(echo "$node_ver" | cut -d. -f1)
  if [ "$node_major" -ge 20 ] 2>/dev/null; then
    pass "Node.js $node_ver (>= 20)"
  else
    warn "Node.js $node_ver found — version 20+ recommended"
  fi
else
  warn "Node.js not found — needed for frontend local dev"
fi

if command -v go > /dev/null 2>&1; then
  go_ver=$(go version | sed -E 's/.*go([0-9]+\.[0-9]+).*/\1/')
  go_minor=$(echo "$go_ver" | cut -d. -f2)
  if [ "$go_minor" -ge 24 ] 2>/dev/null; then
    pass "Go $go_ver (>= 1.24)"
  else
    warn "Go $go_ver found — version 1.24+ recommended"
  fi
else
  warn "Go not found — needed for Go service local dev"
fi

if command -v cargo > /dev/null 2>&1; then
  cargo_ver=$(cargo --version | sed -E 's/.*([0-9]+\.[0-9]+\.[0-9]+).*/\1/')
  pass "Cargo $cargo_ver"
else
  warn "Cargo not found — needed for Rust service local dev"
fi

echo ""

# ── Summary ──────────────────────────────────────────────────
if [ "$errors" -gt 0 ]; then
  echo -e "\033[31m$errors critical issue(s) found. Fix them before running make up.\033[0m"
  echo ""
  exit 1
else
  echo -e "\033[32mAll preflight checks passed. Ready to launch!\033[0m"
  echo ""
  exit 0
fi
