#!/bin/bash
set -e

# =============================================================================
# Slate LMS — Database Health Check
# =============================================================================
# Verifies that all databases are accessible and contain the expected tables.
#
# Usage:
#   ./scripts/db-check.sh
#
# Uses the same environment variables as db-setup.sh for connection details.
# Note: Run `chmod +x scripts/db-check.sh` after checkout.
# =============================================================================

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# --- Color output helpers ---
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
CYAN='\033[0;36m'
NC='\033[0m'

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

ERRORS=0

# --- Helper: check database connectivity ---
check_connection() {
    local host="$1" port="$2" user="$3" password="$4" dbname="$5"
    if PGPASSWORD="$password" psql -h "$host" -p "$port" -U "$user" -d "$dbname" -c "SELECT 1;" >/dev/null 2>&1; then
        ok "Connected to '$dbname' on $host:$port"
        return 0
    else
        fail "Cannot connect to '$dbname' on $host:$port"
        ERRORS=$((ERRORS + 1))
        return 1
    fi
}

# --- Helper: check if a table exists ---
check_table() {
    local host="$1" port="$2" user="$3" password="$4" dbname="$5" table="$6"
    local exists
    exists=$(PGPASSWORD="$password" psql -h "$host" -p "$port" -U "$user" -d "$dbname" -tAc \
        "SELECT EXISTS (SELECT FROM information_schema.tables WHERE table_schema='public' AND table_name='$table');" 2>/dev/null)
    if [[ "$exists" == "t" ]]; then
        ok "Table '$table' exists"
    else
        fail "Table '$table' missing"
        ERRORS=$((ERRORS + 1))
    fi
}

# --- Helper: count rows in a table ---
count_rows() {
    local host="$1" port="$2" user="$3" password="$4" dbname="$5" table="$6"
    local count
    count=$(PGPASSWORD="$password" psql -h "$host" -p "$port" -U "$user" -d "$dbname" -tAc \
        "SELECT COUNT(*) FROM $table;" 2>/dev/null)
    info "$table: $count row(s)"
}

# =============================================================================
# Main
# =============================================================================

echo -e "${CYAN}Slate LMS — Database Health Check${NC}"

# --- 1. userauth ---
header "userauth (port $USERAUTH_DB_PORT)"

if check_connection "$USERAUTH_DB_HOST" "$USERAUTH_DB_PORT" "$USERAUTH_DB_USER" "$USERAUTH_DB_PASSWORD" "$USERAUTH_DB_NAME"; then
    USERAUTH_TABLES=(
        users roles user_roles
        oauth_providers user_mfa
        saml_configs saml_sessions saml_metadata_cache
        user_groups group_members
        parent_child_accounts
    )
    for table in "${USERAUTH_TABLES[@]}"; do
        check_table "$USERAUTH_DB_HOST" "$USERAUTH_DB_PORT" "$USERAUTH_DB_USER" "$USERAUTH_DB_PASSWORD" "$USERAUTH_DB_NAME" "$table"
    done

    echo ""
    info "Seed data check:"
    count_rows "$USERAUTH_DB_HOST" "$USERAUTH_DB_PORT" "$USERAUTH_DB_USER" "$USERAUTH_DB_PASSWORD" "$USERAUTH_DB_NAME" "roles"
    count_rows "$USERAUTH_DB_HOST" "$USERAUTH_DB_PORT" "$USERAUTH_DB_USER" "$USERAUTH_DB_PASSWORD" "$USERAUTH_DB_NAME" "users"
fi

# --- 2. assignment_grading ---
header "assignment_grading (port $GRADING_DB_PORT)"

if check_connection "$GRADING_DB_HOST" "$GRADING_DB_PORT" "$GRADING_DB_USER" "$GRADING_DB_PASSWORD" "$GRADING_DB_NAME"; then
    GRADING_TABLES=(assignments submissions grades)
    for table in "${GRADING_TABLES[@]}"; do
        check_table "$GRADING_DB_HOST" "$GRADING_DB_PORT" "$GRADING_DB_USER" "$GRADING_DB_PASSWORD" "$GRADING_DB_NAME" "$table"
    done
fi

# --- 3. cms ---
header "cms (port $CMS_DB_PORT)"

if check_connection "$CMS_DB_HOST" "$CMS_DB_PORT" "$CMS_DB_USER" "$CMS_DB_PASSWORD" "$CMS_DB_NAME"; then
    CMS_TABLES=(
        modules lessons resources
        upload_sessions progress_tracking
        transcoding_jobs download_tracking
    )
    for table in "${CMS_TABLES[@]}"; do
        check_table "$CMS_DB_HOST" "$CMS_DB_PORT" "$CMS_DB_USER" "$CMS_DB_PASSWORD" "$CMS_DB_NAME" "$table"
    done
fi

# --- Summary ---
header "Summary"

if [[ $ERRORS -eq 0 ]]; then
    ok "All checks passed"
    exit 0
else
    fail "$ERRORS check(s) failed"
    exit 1
fi
