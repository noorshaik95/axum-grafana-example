#!/bin/bash
# Seed development data for Eastfield University (default dev tenant)
# Creates mock instructors, students, courses, and assignments via the API gateway.
#
# Usage: ./scripts/seed-dev.sh [--api-url http://localhost:8080]
# Requirements: curl, jq

set -euo pipefail

# ── Config ────────────────────────────────────────────────────────────────────
API_URL="${API_URL:-http://localhost:8080}"
COMPOSE_FILE="docker-compose.yml"

# Override via flag
while [[ $# -gt 0 ]]; do
  case "$1" in
    --api-url) API_URL="$2"; shift 2 ;;
    *) echo "Unknown flag: $1"; exit 1 ;;
  esac
done

# ── Colors ────────────────────────────────────────────────────────────────────
RED='\033[0;31m'; GREEN='\033[0;32m'; YELLOW='\033[1;33m'
BLUE='\033[0;34m'; BOLD='\033[1m'; NC='\033[0m'

log()     { echo -e "${BLUE}▸${NC} $*"; }
success() { echo -e "${GREEN}✓${NC} $*"; }
warn()    { echo -e "${YELLOW}⚠${NC} $*"; }
fail()    { echo -e "${RED}✗${NC} $*"; }
header()  { echo -e "\n${BOLD}$*${NC}"; }

# ── Helpers ───────────────────────────────────────────────────────────────────
require_cmd() {
  command -v "$1" &>/dev/null || { fail "Required: $1"; exit 1; }
}

wait_for_api() {
  log "Waiting for API gateway at $API_URL..."
  local retries=30
  while ! curl -sf "$API_URL/health" &>/dev/null; do
    retries=$((retries - 1))
    [[ $retries -eq 0 ]] && { fail "API gateway not reachable after 30s"; exit 1; }
    sleep 1
  done
  success "API gateway is ready"
}

# POST /api/auth/register → returns access_token and user.id
register() {
  local email="$1" password="$2" first="$3" last="$4"
  curl -sf -X POST "$API_URL/api/auth/register" \
    -H "Content-Type: application/json" \
    -d "{\"email\":\"$email\",\"password\":\"$password\",\"first_name\":\"$first\",\"last_name\":\"$last\"}"
}

# POST /api/auth/login → returns access_token
login() {
  local email="$1" password="$2"
  curl -sf -X POST "$API_URL/api/auth/login" \
    -H "Content-Type: application/json" \
    -d "{\"email\":\"$email\",\"password\":\"$password\"}"
}

# Assign role via SQL (runs inside postgres container)
assign_role() {
  local email="$1" role="$2"
  docker compose -f "$COMPOSE_FILE" exec -T postgres psql -U postgres -d userauth -q -c \
    "INSERT INTO user_roles (user_id, role_id, assigned_at)
     SELECT u.id, r.id, NOW()
     FROM users u JOIN roles r ON r.name = '$role'
     WHERE u.email = '$email'
     ON CONFLICT (user_id, role_id) DO NOTHING;" 2>/dev/null
}

# POST /api/courses → returns course id
create_course() {
  local token="$1" title="$2" code="$3" dept="$4" instructor_id="$5" term="$6" description="$7"
  curl -sf -X POST "$API_URL/api/courses" \
    -H "Authorization: Bearer $token" \
    -H "Content-Type: application/json" \
    -d "{
      \"title\": \"$title\",
      \"description\": \"$description\",
      \"term\": \"$term\",
      \"instructor_id\": \"$instructor_id\",
      \"metadata\": {
        \"course_code\": \"$code\",
        \"department\": \"$dept\",
        \"credits\": 3,
        \"max_students\": 30
      }
    }"
}

# POST /api/assignments → returns assignment id
create_assignment() {
  local token="$1" course_id="$2" title="$3" description="$4" points="$5" due="$6"
  curl -sf -X POST "$API_URL/api/assignments" \
    -H "Authorization: Bearer $token" \
    -H "Content-Type: application/json" \
    -d "{
      \"course_id\": \"$course_id\",
      \"title\": \"$title\",
      \"description\": \"$description\",
      \"max_points\": $points,
      \"due_date\": \"$due\",
      \"late_policy\": {\"penalty_percent_per_day\": 10, \"max_late_days\": 3}
    }"
}

# POST /api/courses/:id/enroll
enroll() {
  local token="$1" course_id="$2" student_id="$3"
  curl -sf -X POST "$API_URL/api/courses/$course_id/enroll" \
    -H "Authorization: Bearer $token" \
    -H "Content-Type: application/json" \
    -d "{\"course_id\": \"$course_id\", \"student_id\": \"$student_id\"}" || true
}

# POST /api/courses/:id/publish
publish_course() {
  local token="$1" course_id="$2"
  curl -sf -X POST "$API_URL/api/courses/$course_id/publish" \
    -H "Authorization: Bearer $token" \
    -H "Content-Type: application/json" \
    -d "{}" || true
}

# ── Preflight ─────────────────────────────────────────────────────────────────
require_cmd curl
require_cmd jq
require_cmd docker
wait_for_api

echo ""
echo -e "${BOLD}╔══════════════════════════════════════════╗${NC}"
echo -e "${BOLD}║   Slate LMS — Dev Seed (Eastfield Univ)  ║${NC}"
echo -e "${BOLD}╚══════════════════════════════════════════╝${NC}"

PASSWORD="Test@1234"
DUE_DATE="2026-09-30T23:59:59Z"

# ── Users ─────────────────────────────────────────────────────────────────────
header "1/4  Creating users"

declare -A USER_ID TOKEN

create_user() {
  local key="$1" email="$2" first="$3" last="$4" role="$5"
  log "Registering $first $last ($role)..."
  local resp
  resp=$(register "$email" "$PASSWORD" "$first" "$last" 2>/dev/null) \
    || { warn "  Already exists or registration failed — attempting login"; resp=$(login "$email" "$PASSWORD"); }

  USER_ID[$key]=$(echo "$resp" | jq -r '.user.id // empty')
  TOKEN[$key]=$(echo "$resp" | jq -r '.access_token // empty')

  if [[ -z "${USER_ID[$key]}" ]]; then
    warn "  Could not get user ID for $email (may already exist). Fetching via login..."
    resp=$(login "$email" "$PASSWORD")
    USER_ID[$key]=$(echo "$resp" | jq -r '.user.id // empty')
    TOKEN[$key]=$(echo "$resp" | jq -r '.access_token // empty')
  fi

  assign_role "$email" "$role" \
    && success "  $first $last → $email  [role: $role]" \
    || warn "  Role assignment skipped (postgres unreachable or already assigned)"
}

create_user "sarah"  "dr.sarah.chen@eastfield.edu"       "Sarah"  "Chen"    "instructor"
create_user "james"  "prof.james.wilson@eastfield.edu"   "James"  "Wilson"  "instructor"
create_user "alice"  "alice.johnson@eastfield.edu"       "Alice"  "Johnson" "student"
create_user "bob"    "bob.smith@eastfield.edu"           "Bob"    "Smith"   "student"
create_user "carol"  "carol.white@eastfield.edu"         "Carol"  "White"   "student"
create_user "david"  "david.brown@eastfield.edu"         "David"  "Brown"   "student"
create_user "emma"   "emma.davis@eastfield.edu"          "Emma"   "Davis"   "student"

# ── Courses ───────────────────────────────────────────────────────────────────
header "2/4  Creating courses"

declare -A COURSE_ID

create_and_store_course() {
  local key="$1" token_key="$2" title="$3" code="$4" dept="$5" instr_key="$6" term="$7" desc="$8"
  log "Creating: $title ($code)..."
  local resp
  resp=$(create_course "${TOKEN[$token_key]}" "$title" "$code" "$dept" "${USER_ID[$instr_key]}" "$term" "$desc" 2>/dev/null) || { warn "  Failed to create $title (service may be unavailable)"; return; }
  COURSE_ID[$key]=$(echo "$resp" | jq -r '.id // .course.id // empty')
  if [[ -n "${COURSE_ID[$key]}" ]]; then
    publish_course "${TOKEN[$token_key]}" "${COURSE_ID[$key]}"
    success "  $title → ID: ${COURSE_ID[$key]}"
  else
    warn "  Could not parse course ID from response"
  fi
}

create_and_store_course "cs101" "sarah" \
  "Introduction to Computer Science" "CS-101" "Computer Science" "sarah" "Fall 2026" \
  "Foundational programming concepts using Python. Covers variables, control flow, functions, and basic data structures."

create_and_store_course "cs201" "sarah" \
  "Data Structures and Algorithms" "CS-201" "Computer Science" "sarah" "Fall 2026" \
  "Core data structures (arrays, linked lists, trees, graphs) and algorithm analysis. Prerequisite: CS-101."

create_and_store_course "web101" "james" \
  "Web Development Fundamentals" "WEB-101" "Software Engineering" "james" "Fall 2026" \
  "Hands-on introduction to HTML, CSS, and JavaScript. Students build and deploy a personal portfolio site."

create_and_store_course "db101" "james" \
  "Database Systems" "DB-101" "Computer Science" "james" "Fall 2026" \
  "Relational databases, SQL, normalization, indexing, and transactions. Includes a group project with PostgreSQL."

# ── Assignments ───────────────────────────────────────────────────────────────
header "3/4  Creating assignments"

create_asgn() {
  local course_key="$1" token_key="$2" title="$3" desc="$4" pts="$5"
  [[ -z "${COURSE_ID[$course_key]:-}" ]] && { warn "  Skipping — course ${course_key} not created"; return; }
  log "Assignment: $title..."
  local resp
  resp=$(create_assignment "${TOKEN[$token_key]}" "${COURSE_ID[$course_key]}" "$title" "$desc" "$pts" "$DUE_DATE" 2>/dev/null) \
    || { warn "  Failed to create assignment"; return; }
  local id
  id=$(echo "$resp" | jq -r '.id // .assignment.id // empty')
  [[ -n "$id" ]] && success "  $title → ID: $id" || warn "  Could not parse assignment ID"
}

# CS-101
create_asgn "cs101" "sarah" "Lab 1: Hello World"          "Write your first Python program." 20
create_asgn "cs101" "sarah" "Lab 2: Control Flow"         "Implement loops and conditionals to solve basic problems." 30
create_asgn "cs101" "sarah" "Project: Number Guessing Game" "Build an interactive number guessing game in Python." 50

# CS-201
create_asgn "cs201" "sarah" "Lab 1: Linked List"          "Implement a singly linked list with insert, delete, search." 40
create_asgn "cs201" "sarah" "Lab 2: Binary Search Tree"   "Build a BST with in-order, pre-order, post-order traversal." 40
create_asgn "cs201" "sarah" "Project: Pathfinding"        "Implement BFS and DFS on a graph." 80

# WEB-101
create_asgn "web101" "james" "Lab 1: HTML Portfolio"      "Build a personal portfolio page with semantic HTML." 25
create_asgn "web101" "james" "Lab 2: CSS Styling"         "Style your portfolio using Flexbox and CSS Grid." 25
create_asgn "web101" "james" "Project: Interactive Page"  "Add JavaScript interactivity to your portfolio." 50

# DB-101
create_asgn "db101" "james" "Lab 1: SQL Basics"           "Write SELECT, INSERT, UPDATE, DELETE queries." 30
create_asgn "db101" "james" "Lab 2: Joins and Aggregates" "Practice INNER/LEFT JOIN, GROUP BY, HAVING." 30
create_asgn "db101" "james" "Project: Library System"     "Design and implement a library management database." 60

# ── Enrollments ───────────────────────────────────────────────────────────────
header "4/4  Enrolling students"

enroll_student() {
  local student_key="$1" course_key="$2"
  local student_id="${USER_ID[$student_key]:-}" course_id="${COURSE_ID[$course_key]:-}"
  [[ -z "$student_id" || -z "$course_id" ]] && { warn "  Skipping enroll $student_key → $course_key (missing IDs)"; return; }
  log "Enrolling ${student_key} in ${course_key}..."
  enroll "${TOKEN[$student_key]}" "$course_id" "$student_id" \
    && success "  ${student_key} enrolled in ${course_key}" \
    || warn "  Enroll failed (may already be enrolled)"
}

# CS-101: Alice, Bob, Carol
enroll_student "alice" "cs101"
enroll_student "bob"   "cs101"
enroll_student "carol" "cs101"

# CS-201: Bob, Carol, David
enroll_student "bob"   "cs201"
enroll_student "carol" "cs201"
enroll_student "david" "cs201"

# WEB-101: Carol, David, Emma
enroll_student "carol" "web101"
enroll_student "david" "web101"
enroll_student "emma"  "web101"

# DB-101: Alice, David, Emma
enroll_student "alice" "db101"
enroll_student "david" "db101"
enroll_student "emma"  "db101"

# ── Summary ───────────────────────────────────────────────────────────────────
echo ""
echo -e "${BOLD}╔══════════════════════════════════════════╗${NC}"
echo -e "${BOLD}║              Seed Complete               ║${NC}"
echo -e "${BOLD}╚══════════════════════════════════════════╝${NC}"
echo ""
echo -e "${BOLD}System Admin${NC}"
echo "  admin@slate.edu              Admin@123456"
echo ""
echo -e "${BOLD}Instructors${NC}"
echo "  dr.sarah.chen@eastfield.edu  Test@1234   (CS-101, CS-201)"
echo "  prof.james.wilson@eastfield.edu  Test@1234   (WEB-101, DB-101)"
echo ""
echo -e "${BOLD}Students${NC}"
echo "  alice.johnson@eastfield.edu  Test@1234   (CS-101, DB-101)"
echo "  bob.smith@eastfield.edu      Test@1234   (CS-101, CS-201)"
echo "  carol.white@eastfield.edu    Test@1234   (CS-101, CS-201, WEB-101)"
echo "  david.brown@eastfield.edu    Test@1234   (CS-201, WEB-101, DB-101)"
echo "  emma.davis@eastfield.edu     Test@1234   (WEB-101, DB-101)"
echo ""
echo -e "API: ${BLUE}$API_URL${NC}   Admin UI: ${BLUE}http://localhost:3003${NC}"
echo ""
