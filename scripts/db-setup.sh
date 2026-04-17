#!/bin/bash
set -e

# =============================================================================
# Slate LMS — Local Development Database Setup
# =============================================================================
# Creates databases, runs all migrations, and seeds default data.
#
# Usage:
#   ./scripts/db-setup.sh            # Run migrations on existing DBs
#   ./scripts/db-setup.sh --reset    # Drop and recreate DBs, then run migrations
#
# Environment variables (with defaults matching docker-compose.yml):
#   USERAUTH_DB_HOST      (default: localhost)
#   USERAUTH_DB_PORT      (default: 5435)
#   USERAUTH_DB_USER      (default: postgres)
#   USERAUTH_DB_PASSWORD  (default: postgres)
#   USERAUTH_DB_NAME      (default: userauth)
#
#   GRADING_DB_HOST       (default: localhost)
#   GRADING_DB_PORT       (default: 5433)
#   GRADING_DB_USER       (default: postgres)
#   GRADING_DB_PASSWORD   (default: postgres)
#   GRADING_DB_NAME       (default: assignment_grading)
#
#   CMS_DB_HOST           (default: localhost)
#   CMS_DB_PORT           (default: 5434)
#   CMS_DB_USER           (default: cms)
#   CMS_DB_PASSWORD       (default: cms_password)
#   CMS_DB_NAME           (default: cms)
#
# Note: Run `chmod +x scripts/db-setup.sh` after checkout.
# =============================================================================

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

# --- Color output helpers ---
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

ok()   { echo -e "  ${GREEN}✓${NC} $1"; }
fail() { echo -e "  ${RED}✗${NC} $1"; }
info() { echo -e "  ${CYAN}→${NC} $1"; }
header() { echo -e "\n${YELLOW}━━━ $1 ━━━${NC}"; }

# --- Configuration ---
USERAUTH_DB_HOST="${USERAUTH_DB_HOST:-localhost}"
USERAUTH_DB_PORT="${USERAUTH_DB_PORT:-5435}"
USERAUTH_DB_USER="${USERAUTH_DB_USER:-postgres}"
USERAUTH_DB_PASSWORD="${USERAUTH_DB_PASSWORD:-postgres}"
USERAUTH_DB_NAME="${USERAUTH_DB_NAME:-userauth}"

GRADING_DB_HOST="${GRADING_DB_HOST:-localhost}"
GRADING_DB_PORT="${GRADING_DB_PORT:-5433}"
GRADING_DB_USER="${GRADING_DB_USER:-postgres}"
GRADING_DB_PASSWORD="${GRADING_DB_PASSWORD:-postgres}"
GRADING_DB_NAME="${GRADING_DB_NAME:-assignment_grading}"

CMS_DB_HOST="${CMS_DB_HOST:-localhost}"
CMS_DB_PORT="${CMS_DB_PORT:-5434}"
CMS_DB_USER="${CMS_DB_USER:-cms}"
CMS_DB_PASSWORD="${CMS_DB_PASSWORD:-cms_password}"
CMS_DB_NAME="${CMS_DB_NAME:-cms}"

RESET=false
if [[ "$1" == "--reset" ]]; then
    RESET=true
fi

# --- Helper: build psql connection string ---
psql_cmd() {
    local host="$1" port="$2" user="$3" password="$4" dbname="$5"
    PGPASSWORD="$password" psql -h "$host" -p "$port" -U "$user" -d "$dbname" -v ON_ERROR_STOP=1 --quiet
}

# --- Helper: run psql against the maintenance DB (postgres) ---
psql_admin() {
    local host="$1" port="$2" user="$3" password="$4"
    PGPASSWORD="$password" psql -h "$host" -p "$port" -U "$user" -d postgres -v ON_ERROR_STOP=1 --quiet
}

# --- Helper: check if a database exists ---
db_exists() {
    local host="$1" port="$2" user="$3" password="$4" dbname="$5"
    PGPASSWORD="$password" psql -h "$host" -p "$port" -U "$user" -d postgres -tAc \
        "SELECT 1 FROM pg_database WHERE datname='$dbname'" 2>/dev/null | grep -q 1
}

# --- Helper: drop and recreate a database ---
reset_db() {
    local host="$1" port="$2" user="$3" password="$4" dbname="$5"
    info "Dropping database '$dbname'..."
    PGPASSWORD="$password" psql -h "$host" -p "$port" -U "$user" -d postgres -v ON_ERROR_STOP=1 --quiet <<SQL
SELECT pg_terminate_backend(pid) FROM pg_stat_activity WHERE datname = '$dbname' AND pid <> pg_backend_pid();
DROP DATABASE IF EXISTS $dbname;
SQL
    ok "Dropped '$dbname'"
    create_db "$host" "$port" "$user" "$password" "$dbname"
}

# --- Helper: create a database if it doesn't exist ---
create_db() {
    local host="$1" port="$2" user="$3" password="$4" dbname="$5"
    if db_exists "$host" "$port" "$user" "$password" "$dbname"; then
        ok "Database '$dbname' already exists"
    else
        info "Creating database '$dbname'..."
        PGPASSWORD="$password" psql -h "$host" -p "$port" -U "$user" -d postgres -v ON_ERROR_STOP=1 --quiet \
            -c "CREATE DATABASE $dbname;"
        ok "Created database '$dbname'"
    fi
}

# --- Helper: run migration files in order ---
# Filters to only .sql files that are NOT .down.sql (up migrations only)
run_migrations() {
    local host="$1" port="$2" user="$3" password="$4" dbname="$5" migrations_dir="$6"
    local count=0

    if [[ ! -d "$migrations_dir" ]]; then
        fail "Migrations directory not found: $migrations_dir"
        return 1
    fi

    # Sort files and filter out .down.sql rollback files
    for migration in $(find "$migrations_dir" -maxdepth 1 -name '*.sql' ! -name '*.down.sql' | sort); do
        local filename
        filename="$(basename "$migration")"
        info "Running $filename..."
        if PGPASSWORD="$password" psql -h "$host" -p "$port" -U "$user" -d "$dbname" \
            -v ON_ERROR_STOP=1 --quiet -f "$migration" 2>&1; then
            ok "$filename"
        else
            fail "$filename"
            return 1
        fi
        count=$((count + 1))
    done

    ok "Ran $count migration(s)"
}

# =============================================================================
# Main
# =============================================================================

echo -e "${CYAN}Slate LMS — Database Setup${NC}"
if $RESET; then
    echo -e "${RED}Mode: RESET (databases will be dropped and recreated)${NC}"
else
    echo -e "${GREEN}Mode: MIGRATE (apply migrations to existing databases)${NC}"
fi

# --- 1. userauth ---
header "userauth (port $USERAUTH_DB_PORT)"

if $RESET; then
    reset_db "$USERAUTH_DB_HOST" "$USERAUTH_DB_PORT" "$USERAUTH_DB_USER" "$USERAUTH_DB_PASSWORD" "$USERAUTH_DB_NAME"
else
    create_db "$USERAUTH_DB_HOST" "$USERAUTH_DB_PORT" "$USERAUTH_DB_USER" "$USERAUTH_DB_PASSWORD" "$USERAUTH_DB_NAME"
fi

run_migrations "$USERAUTH_DB_HOST" "$USERAUTH_DB_PORT" "$USERAUTH_DB_USER" "$USERAUTH_DB_PASSWORD" "$USERAUTH_DB_NAME" \
    "$PROJECT_ROOT/services/user-auth-service/migrations"

# --- 2. assignment_grading ---
header "assignment_grading (port $GRADING_DB_PORT)"

if $RESET; then
    reset_db "$GRADING_DB_HOST" "$GRADING_DB_PORT" "$GRADING_DB_USER" "$GRADING_DB_PASSWORD" "$GRADING_DB_NAME"
else
    create_db "$GRADING_DB_HOST" "$GRADING_DB_PORT" "$GRADING_DB_USER" "$GRADING_DB_PASSWORD" "$GRADING_DB_NAME"
fi

run_migrations "$GRADING_DB_HOST" "$GRADING_DB_PORT" "$GRADING_DB_USER" "$GRADING_DB_PASSWORD" "$GRADING_DB_NAME" \
    "$PROJECT_ROOT/services/assignment-grading-service/migrations"

# --- 3. cms ---
header "cms (port $CMS_DB_PORT)"

if $RESET; then
    reset_db "$CMS_DB_HOST" "$CMS_DB_PORT" "$CMS_DB_USER" "$CMS_DB_PASSWORD" "$CMS_DB_NAME"
else
    create_db "$CMS_DB_HOST" "$CMS_DB_PORT" "$CMS_DB_USER" "$CMS_DB_PASSWORD" "$CMS_DB_NAME"
fi

run_migrations "$CMS_DB_HOST" "$CMS_DB_PORT" "$CMS_DB_USER" "$CMS_DB_PASSWORD" "$CMS_DB_NAME" \
    "$PROJECT_ROOT/services/content-management-service/migrations"

# --- Done ---
header "Setup Complete"
ok "All databases initialized and migrations applied."
echo ""
echo -e "  Default admin login:  ${CYAN}admin@slate.edu${NC} / ${CYAN}Admin@123456${NC}"
echo ""
