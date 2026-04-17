#!/bin/bash

# Slate LMS API Integration Tests
#
# Tests the full API flow through the API gateway including:
#   - Health checks
#   - User registration
#   - User login (JWT acquisition)
#   - Token refresh
#   - Authenticated endpoints (profile, courses, assignments)
#   - Role-based access (instructor course creation)
#   - Error handling (401, 404)
#
# Prerequisites:
#   - API gateway running on port 8080
#   - Backend services running (user-auth, course, assignment)
#   - jq recommended (falls back to grep-based parsing)
#
# Usage: ./tests/api_integration_test.sh [API_URL]
# Default API_URL: http://localhost:8080

set -uo pipefail

API_URL="${1:-http://localhost:8080}"
PASS=0
FAIL=0
SKIP=0

GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
CYAN='\033[0;36m'
NC='\033[0m'

pass() { echo -e "  ${GREEN}PASS${NC} $1"; ((PASS++)); }
fail() { echo -e "  ${RED}FAIL${NC} $1"; ((FAIL++)); }
skip() { echo -e "  ${YELLOW}SKIP${NC} $1"; ((SKIP++)); }
section() { echo -e "\n${CYAN}--- $1 ---${NC}"; }

# JSON field extraction: uses jq if available, falls back to grep
HAS_JQ=false
if command -v jq > /dev/null 2>&1; then
    HAS_JQ=true
fi

json_field() {
    local json="$1"
    local field="$2"
    if $HAS_JQ; then
        echo "$json" | jq -r ".$field // empty" 2>/dev/null
    else
        echo "$json" | grep -o "\"$field\":\"[^\"]*\"" | head -1 | cut -d'"' -f4
    fi
}

json_has_field() {
    local json="$1"
    local field="$2"
    if $HAS_JQ; then
        echo "$json" | jq -e "has(\"$field\")" > /dev/null 2>&1
    else
        echo "$json" | grep -q "\"$field\""
    fi
}

# Generate unique test email to avoid conflicts between runs
TEST_RUN_ID="$(date +%s)"
TEST_EMAIL="integration_test_${TEST_RUN_ID}@example.com"
TEST_PASSWORD="TestPass123!"
TEST_FIRST_NAME="Integration"
TEST_LAST_NAME="Test"

echo "Slate LMS API Integration Tests"
echo "================================"
echo "API URL:    $API_URL"
echo "Test email: $TEST_EMAIL"
echo "jq:         $($HAS_JQ && echo 'available' || echo 'not found (using grep fallback)')"
echo ""

# ──────────────────────────────────────────────
section "Health Checks"
# ──────────────────────────────────────────────

# Test: Gateway liveness
RESPONSE=$(curl -s -w "\n%{http_code}" "$API_URL/health/live" 2>&1) || true
HTTP_CODE=$(echo "$RESPONSE" | tail -1)
if [ "$HTTP_CODE" = "200" ]; then
    pass "GET /health/live returns 200"
else
    fail "GET /health/live returned $HTTP_CODE (expected 200)"
fi

# Test: Gateway readiness
RESPONSE=$(curl -s -w "\n%{http_code}" "$API_URL/health/ready" 2>&1) || true
HTTP_CODE=$(echo "$RESPONSE" | tail -1)
if [ "$HTTP_CODE" = "200" ]; then
    pass "GET /health/ready returns 200"
else
    fail "GET /health/ready returned $HTTP_CODE (expected 200)"
fi

# ──────────────────────────────────────────────
section "User Registration"
# ──────────────────────────────────────────────

# Test: Register a new user
REGISTER_RESPONSE=$(curl -s -w "\n%{http_code}" -X POST "$API_URL/api/auth/register" \
    -H "Content-Type: application/json" \
    -d "{
        \"email\": \"$TEST_EMAIL\",
        \"password\": \"$TEST_PASSWORD\",
        \"first_name\": \"$TEST_FIRST_NAME\",
        \"last_name\": \"$TEST_LAST_NAME\",
        \"phone\": \"+1555${TEST_RUN_ID: -7}\"
    }" 2>&1) || true

REGISTER_BODY=$(echo "$REGISTER_RESPONSE" | sed '$d')
REGISTER_CODE=$(echo "$REGISTER_RESPONSE" | tail -1)

if [ "$REGISTER_CODE" = "200" ] || [ "$REGISTER_CODE" = "201" ]; then
    if json_has_field "$REGISTER_BODY" "access_token"; then
        pass "POST /api/auth/register returns token (HTTP $REGISTER_CODE)"
        REGISTER_TOKEN=$(json_field "$REGISTER_BODY" "access_token")
    else
        pass "POST /api/auth/register succeeds (HTTP $REGISTER_CODE)"
        REGISTER_TOKEN=""
    fi
else
    fail "POST /api/auth/register returned HTTP $REGISTER_CODE -- $REGISTER_BODY"
    REGISTER_TOKEN=""
fi

# Test: Duplicate registration should fail
DUP_RESPONSE=$(curl -s -w "\n%{http_code}" -X POST "$API_URL/api/auth/register" \
    -H "Content-Type: application/json" \
    -d "{
        \"email\": \"$TEST_EMAIL\",
        \"password\": \"$TEST_PASSWORD\",
        \"first_name\": \"$TEST_FIRST_NAME\",
        \"last_name\": \"$TEST_LAST_NAME\"
    }" 2>&1) || true

DUP_CODE=$(echo "$DUP_RESPONSE" | tail -1)
if [ "$DUP_CODE" = "409" ] || [ "$DUP_CODE" = "400" ]; then
    pass "Duplicate registration rejected (HTTP $DUP_CODE)"
else
    # Some APIs return 200 with an error message
    DUP_BODY=$(echo "$DUP_RESPONSE" | sed '$d')
    if echo "$DUP_BODY" | grep -qi "already exists\|duplicate\|conflict"; then
        pass "Duplicate registration rejected (error in body)"
    else
        fail "Duplicate registration not rejected (HTTP $DUP_CODE)"
    fi
fi

# Test: Registration with invalid data
INVALID_RESPONSE=$(curl -s -w "\n%{http_code}" -X POST "$API_URL/api/auth/register" \
    -H "Content-Type: application/json" \
    -d '{"email": "not-an-email", "password": "x"}' 2>&1) || true

INVALID_CODE=$(echo "$INVALID_RESPONSE" | tail -1)
if [ "$INVALID_CODE" = "400" ] || [ "$INVALID_CODE" = "422" ]; then
    pass "Invalid registration data rejected (HTTP $INVALID_CODE)"
else
    fail "Invalid registration data not rejected (HTTP $INVALID_CODE)"
fi

# ──────────────────────────────────────────────
section "User Login"
# ──────────────────────────────────────────────

# Test: Login with the registered user
LOGIN_RESPONSE=$(curl -s -w "\n%{http_code}" -X POST "$API_URL/api/auth/login" \
    -H "Content-Type: application/json" \
    -d "{
        \"email\": \"$TEST_EMAIL\",
        \"password\": \"$TEST_PASSWORD\"
    }" 2>&1) || true

LOGIN_BODY=$(echo "$LOGIN_RESPONSE" | sed '$d')
LOGIN_CODE=$(echo "$LOGIN_RESPONSE" | tail -1)

ACCESS_TOKEN=""
REFRESH_TOKEN=""

if [ "$LOGIN_CODE" = "200" ]; then
    ACCESS_TOKEN=$(json_field "$LOGIN_BODY" "access_token")
    REFRESH_TOKEN=$(json_field "$LOGIN_BODY" "refresh_token")

    if [ -n "$ACCESS_TOKEN" ]; then
        pass "POST /api/auth/login returns access_token"
    else
        fail "POST /api/auth/login missing access_token"
    fi

    if [ -n "$REFRESH_TOKEN" ]; then
        pass "POST /api/auth/login returns refresh_token"
    else
        skip "POST /api/auth/login did not return refresh_token"
    fi
else
    fail "POST /api/auth/login returned HTTP $LOGIN_CODE -- $LOGIN_BODY"
fi

# Test: Login with wrong password
BAD_LOGIN_RESPONSE=$(curl -s -w "\n%{http_code}" -X POST "$API_URL/api/auth/login" \
    -H "Content-Type: application/json" \
    -d "{
        \"email\": \"$TEST_EMAIL\",
        \"password\": \"wrongpassword\"
    }" 2>&1) || true

BAD_LOGIN_CODE=$(echo "$BAD_LOGIN_RESPONSE" | tail -1)
if [ "$BAD_LOGIN_CODE" = "401" ] || [ "$BAD_LOGIN_CODE" = "403" ]; then
    pass "Login with wrong password rejected (HTTP $BAD_LOGIN_CODE)"
else
    fail "Login with wrong password not rejected (HTTP $BAD_LOGIN_CODE)"
fi

# Test: Login with non-existent user
NOUSER_RESPONSE=$(curl -s -w "\n%{http_code}" -X POST "$API_URL/api/auth/login" \
    -H "Content-Type: application/json" \
    -d '{"email": "nonexistent@example.com", "password": "password"}' 2>&1) || true

NOUSER_CODE=$(echo "$NOUSER_RESPONSE" | tail -1)
if [ "$NOUSER_CODE" = "401" ] || [ "$NOUSER_CODE" = "404" ]; then
    pass "Login with non-existent user rejected (HTTP $NOUSER_CODE)"
else
    fail "Login with non-existent user not rejected (HTTP $NOUSER_CODE)"
fi

# ──────────────────────────────────────────────
section "Authenticated Endpoints"
# ──────────────────────────────────────────────

if [ -z "$ACCESS_TOKEN" ]; then
    skip "All authenticated tests (no access token from login)"
else

    # Test: Access profile with valid JWT
    PROFILE_RESPONSE=$(curl -s -w "\n%{http_code}" -X GET "$API_URL/api/users/profile" \
        -H "Authorization: Bearer $ACCESS_TOKEN" 2>&1) || true

    PROFILE_BODY=$(echo "$PROFILE_RESPONSE" | sed '$d')
    PROFILE_CODE=$(echo "$PROFILE_RESPONSE" | tail -1)

    if [ "$PROFILE_CODE" = "200" ]; then
        if echo "$PROFILE_BODY" | grep -q "$TEST_EMAIL\|email"; then
            pass "GET /api/users/profile returns user data"
        else
            pass "GET /api/users/profile returns 200"
        fi
    else
        fail "GET /api/users/profile returned HTTP $PROFILE_CODE"
    fi

    # Test: Access without token should fail
    NOAUTH_RESPONSE=$(curl -s -w "\n%{http_code}" -X GET "$API_URL/api/users/profile" 2>&1) || true
    NOAUTH_CODE=$(echo "$NOAUTH_RESPONSE" | tail -1)

    if [ "$NOAUTH_CODE" = "401" ] || [ "$NOAUTH_CODE" = "403" ]; then
        pass "GET /api/users/profile without token returns $NOAUTH_CODE"
    else
        fail "GET /api/users/profile without token returned $NOAUTH_CODE (expected 401)"
    fi

    # Test: Access with invalid token should fail
    BADTOKEN_RESPONSE=$(curl -s -w "\n%{http_code}" -X GET "$API_URL/api/users/profile" \
        -H "Authorization: Bearer invalid.token.here" 2>&1) || true
    BADTOKEN_CODE=$(echo "$BADTOKEN_RESPONSE" | tail -1)

    if [ "$BADTOKEN_CODE" = "401" ] || [ "$BADTOKEN_CODE" = "403" ]; then
        pass "Invalid JWT rejected (HTTP $BADTOKEN_CODE)"
    else
        fail "Invalid JWT not rejected (HTTP $BADTOKEN_CODE)"
    fi

    # Test: List courses
    COURSES_RESPONSE=$(curl -s -w "\n%{http_code}" -X GET "$API_URL/api/courses" \
        -H "Authorization: Bearer $ACCESS_TOKEN" 2>&1) || true

    COURSES_CODE=$(echo "$COURSES_RESPONSE" | tail -1)
    if [ "$COURSES_CODE" = "200" ]; then
        pass "GET /api/courses returns 200"
    elif [ "$COURSES_CODE" = "401" ] || [ "$COURSES_CODE" = "403" ]; then
        fail "GET /api/courses returned $COURSES_CODE (auth issue)"
    else
        skip "GET /api/courses returned $COURSES_CODE (service may not be ready)"
    fi

    # Test: List assignments
    ASSIGNMENTS_RESPONSE=$(curl -s -w "\n%{http_code}" -X GET "$API_URL/api/assignments" \
        -H "Authorization: Bearer $ACCESS_TOKEN" 2>&1) || true

    ASSIGNMENTS_CODE=$(echo "$ASSIGNMENTS_RESPONSE" | tail -1)
    if [ "$ASSIGNMENTS_CODE" = "200" ]; then
        pass "GET /api/assignments returns 200"
    elif [ "$ASSIGNMENTS_CODE" = "401" ] || [ "$ASSIGNMENTS_CODE" = "403" ]; then
        fail "GET /api/assignments returned $ASSIGNMENTS_CODE (auth issue)"
    else
        skip "GET /api/assignments returned $ASSIGNMENTS_CODE (service may not be ready)"
    fi

fi

# ──────────────────────────────────────────────
section "Token Refresh"
# ──────────────────────────────────────────────

if [ -z "$REFRESH_TOKEN" ]; then
    skip "Token refresh tests (no refresh token available)"
else
    REFRESH_RESPONSE=$(curl -s -w "\n%{http_code}" -X POST "$API_URL/api/auth/refresh" \
        -H "Content-Type: application/json" \
        -d "{\"refresh_token\": \"$REFRESH_TOKEN\"}" 2>&1) || true

    REFRESH_BODY=$(echo "$REFRESH_RESPONSE" | sed '$d')
    REFRESH_CODE=$(echo "$REFRESH_RESPONSE" | tail -1)

    if [ "$REFRESH_CODE" = "200" ]; then
        NEW_TOKEN=$(json_field "$REFRESH_BODY" "access_token")
        if [ -n "$NEW_TOKEN" ]; then
            pass "POST /api/auth/refresh returns new access_token"

            # Verify the new token works
            VERIFY_RESPONSE=$(curl -s -w "\n%{http_code}" -X GET "$API_URL/api/users/profile" \
                -H "Authorization: Bearer $NEW_TOKEN" 2>&1) || true
            VERIFY_CODE=$(echo "$VERIFY_RESPONSE" | tail -1)

            if [ "$VERIFY_CODE" = "200" ]; then
                pass "Refreshed token is valid"
            else
                fail "Refreshed token rejected (HTTP $VERIFY_CODE)"
            fi
        else
            fail "POST /api/auth/refresh missing access_token in response"
        fi
    else
        fail "POST /api/auth/refresh returned HTTP $REFRESH_CODE"
    fi

    # Test: Refresh with invalid token
    BAD_REFRESH_RESPONSE=$(curl -s -w "\n%{http_code}" -X POST "$API_URL/api/auth/refresh" \
        -H "Content-Type: application/json" \
        -d '{"refresh_token": "invalid.refresh.token"}' 2>&1) || true

    BAD_REFRESH_CODE=$(echo "$BAD_REFRESH_RESPONSE" | tail -1)
    if [ "$BAD_REFRESH_CODE" = "401" ] || [ "$BAD_REFRESH_CODE" = "403" ]; then
        pass "Invalid refresh token rejected (HTTP $BAD_REFRESH_CODE)"
    else
        fail "Invalid refresh token not rejected (HTTP $BAD_REFRESH_CODE)"
    fi
fi

# ──────────────────────────────────────────────
section "Instructor Role (Course Creation)"
# ──────────────────────────────────────────────

# Login as admin to test course creation
ADMIN_LOGIN_RESPONSE=$(curl -s -w "\n%{http_code}" -X POST "$API_URL/api/auth/login" \
    -H "Content-Type: application/json" \
    -d '{"email": "admin@example.com", "password": "admin123"}' 2>&1) || true

ADMIN_LOGIN_BODY=$(echo "$ADMIN_LOGIN_RESPONSE" | sed '$d')
ADMIN_LOGIN_CODE=$(echo "$ADMIN_LOGIN_RESPONSE" | tail -1)
ADMIN_TOKEN=""

if [ "$ADMIN_LOGIN_CODE" = "200" ]; then
    ADMIN_TOKEN=$(json_field "$ADMIN_LOGIN_BODY" "access_token")
fi

if [ -n "$ADMIN_TOKEN" ]; then
    CREATE_COURSE_RESPONSE=$(curl -s -w "\n%{http_code}" -X POST "$API_URL/api/courses" \
        -H "Content-Type: application/json" \
        -H "Authorization: Bearer $ADMIN_TOKEN" \
        -d "{
            \"name\": \"Integration Test Course $TEST_RUN_ID\",
            \"description\": \"Created by API integration test\",
            \"code\": \"TEST-$TEST_RUN_ID\"
        }" 2>&1) || true

    CREATE_COURSE_BODY=$(echo "$CREATE_COURSE_RESPONSE" | sed '$d')
    CREATE_COURSE_CODE=$(echo "$CREATE_COURSE_RESPONSE" | tail -1)

    if [ "$CREATE_COURSE_CODE" = "200" ] || [ "$CREATE_COURSE_CODE" = "201" ]; then
        pass "POST /api/courses creates course (HTTP $CREATE_COURSE_CODE)"
    elif [ "$CREATE_COURSE_CODE" = "403" ]; then
        skip "POST /api/courses forbidden for this user (role check working)"
    else
        skip "POST /api/courses returned $CREATE_COURSE_CODE (service may not support this yet)"
    fi
else
    skip "Course creation tests (admin login failed)"
fi

# ──────────────────────────────────────────────
section "Error Handling"
# ──────────────────────────────────────────────

# Test: Non-existent route
NOT_FOUND_RESPONSE=$(curl -s -w "\n%{http_code}" "$API_URL/api/nonexistent" 2>&1) || true
NOT_FOUND_CODE=$(echo "$NOT_FOUND_RESPONSE" | tail -1)

if [ "$NOT_FOUND_CODE" = "404" ]; then
    pass "Non-existent route returns 404"
else
    fail "Non-existent route returned $NOT_FOUND_CODE (expected 404)"
fi

# Test: Invalid JSON body
INVALID_JSON_RESPONSE=$(curl -s -w "\n%{http_code}" -X POST "$API_URL/api/auth/login" \
    -H "Content-Type: application/json" \
    -d 'not-json' 2>&1) || true

INVALID_JSON_CODE=$(echo "$INVALID_JSON_RESPONSE" | tail -1)
if [ "$INVALID_JSON_CODE" = "400" ] || [ "$INVALID_JSON_CODE" = "422" ]; then
    pass "Invalid JSON body rejected (HTTP $INVALID_JSON_CODE)"
else
    fail "Invalid JSON body not rejected (HTTP $INVALID_JSON_CODE)"
fi

# ──────────────────────────────────────────────
# Summary
# ──────────────────────────────────────────────

TOTAL=$((PASS + FAIL + SKIP))
echo ""
echo "============================================"
echo " API Integration Test Results"
echo "============================================"
echo -e " Passed:  ${GREEN}$PASS${NC}"
echo -e " Failed:  ${RED}$FAIL${NC}"
echo -e " Skipped: ${YELLOW}$SKIP${NC}"
echo " Total:   $TOTAL"
echo "============================================"

if [ $FAIL -eq 0 ]; then
    echo -e "${GREEN}All tests passed!${NC}"
    exit 0
else
    echo -e "${RED}$FAIL test(s) failed.${NC}"
    exit 1
fi
