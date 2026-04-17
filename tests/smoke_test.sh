#!/bin/bash

# Smoke tests for the Slate LMS platform
# Verifies basic connectivity and health of all services
#
# Port allocations:
#   api-gateway:                8080
#   user-auth-service:          8081 (HTTP), 50051 (gRPC)
#   course-service:             3001 (HTTP), 50052 (gRPC)
#   content-management-service: 8082 (HTTP), 50055 (gRPC)
#   assignment-grading-service: 8083 (HTTP), 50053 (gRPC)
#   student frontend:           3000
#   course-service HTTP:        3001
#   provider frontend:          3002
#   admin frontend:             3003
#
# Usage: ./tests/smoke_test.sh

set -euo pipefail

GATEWAY_URL="${GATEWAY_URL:-http://localhost:8080}"
USER_AUTH_URL="${USER_AUTH_URL:-http://localhost:8081}"
COURSE_URL="${COURSE_URL:-http://localhost:3001}"
ASSIGNMENT_URL="${ASSIGNMENT_URL:-http://localhost:8083}"
CONTENT_URL="${CONTENT_URL:-http://localhost:8082}"
STUDENT_FRONTEND_URL="${STUDENT_FRONTEND_URL:-http://localhost:3000}"
PROVIDER_FRONTEND_URL="${PROVIDER_FRONTEND_URL:-http://localhost:3002}"
ADMIN_FRONTEND_URL="${ADMIN_FRONTEND_URL:-http://localhost:3003}"

echo "Running smoke tests..."
echo "Gateway URL:          $GATEWAY_URL"
echo "User Auth URL:        $USER_AUTH_URL"
echo "Course Service URL:   $COURSE_URL"
echo "Assignment Svc URL:   $ASSIGNMENT_URL"
echo "Content Mgmt URL:     $CONTENT_URL"
echo ""

# Colors for output
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m'

TESTS_PASSED=0
TESTS_FAILED=0
TESTS_SKIPPED=0

pass() {
    echo -e "${GREEN}  PASS${NC} $1"
    ((TESTS_PASSED++))
}

fail() {
    echo -e "${RED}  FAIL${NC} $1"
    ((TESTS_FAILED++))
}

skip() {
    echo -e "${YELLOW}  SKIP${NC} $1"
    ((TESTS_SKIPPED++))
}

# Helper: check if a URL responds with HTTP 200
check_url() {
    local url="$1"
    local timeout="${2:-5}"
    curl -sf --max-time "$timeout" "$url" > /dev/null 2>&1
}

echo "=== API Gateway ==="

if check_url "$GATEWAY_URL/health/live"; then
    pass "Gateway liveness probe"
else
    fail "Gateway liveness probe"
fi

if check_url "$GATEWAY_URL/health/ready"; then
    pass "Gateway readiness probe"
else
    fail "Gateway readiness probe"
fi

if curl -sf --max-time 5 "$GATEWAY_URL/metrics" 2>/dev/null | grep -q 'api_gateway'; then
    pass "Gateway metrics endpoint"
else
    fail "Gateway metrics endpoint"
fi

echo ""
echo "=== User Auth Service (port 8081) ==="

if check_url "$USER_AUTH_URL/health"; then
    pass "User Auth Service health"
else
    fail "User Auth Service health"
fi

# gRPC connectivity (optional, requires grpcurl)
if command -v grpcurl > /dev/null 2>&1; then
    if grpcurl -plaintext localhost:50051 list > /dev/null 2>&1; then
        pass "User Auth gRPC reflection (port 50051)"
    else
        fail "User Auth gRPC reflection (port 50051)"
    fi
else
    skip "User Auth gRPC reflection (grpcurl not installed)"
fi

echo ""
echo "=== Course Service (port 3001) ==="

if check_url "$COURSE_URL/health"; then
    pass "Course Service health"
else
    fail "Course Service health"
fi

echo ""
echo "=== Assignment Grading Service (port 8083) ==="

if check_url "$ASSIGNMENT_URL/health"; then
    pass "Assignment Grading Service health"
else
    fail "Assignment Grading Service health"
fi

echo ""
echo "=== Content Management Service (port 8082) ==="

if check_url "$CONTENT_URL/health"; then
    pass "Content Management Service health"
else
    fail "Content Management Service health"
fi

echo ""
echo "=== Auth Flow (via Gateway) ==="

# Register a smoke test user
REGISTER_RESPONSE=$(curl -s -X POST "$GATEWAY_URL/api/auth/register" \
    -H "Content-Type: application/json" \
    -d '{
        "email": "smoketest@example.com",
        "password": "testpass123",
        "first_name": "Smoke",
        "last_name": "Test",
        "phone": "+1234567890"
    }' 2>&1) || true

if echo "$REGISTER_RESPONSE" | grep -q "access_token\|email"; then
    pass "User registration"
elif echo "$REGISTER_RESPONSE" | grep -q "already exists\|duplicate"; then
    skip "User registration (user already exists)"
else
    fail "User registration -- Response: $REGISTER_RESPONSE"
fi

# Login
LOGIN_RESPONSE=$(curl -s -X POST "$GATEWAY_URL/api/auth/login" \
    -H "Content-Type: application/json" \
    -d '{
        "email": "admin@example.com",
        "password": "admin123"
    }' 2>&1) || true

ACCESS_TOKEN=""
if echo "$LOGIN_RESPONSE" | grep -q "access_token"; then
    pass "User login"
    ACCESS_TOKEN=$(echo "$LOGIN_RESPONSE" | grep -o '"access_token":"[^"]*"' | cut -d'"' -f4)
else
    fail "User login -- Response: $LOGIN_RESPONSE"
fi

# Authenticated request
if [ -n "$ACCESS_TOKEN" ]; then
    LIST_RESPONSE=$(curl -s -X GET "$GATEWAY_URL/api/users/profile" \
        -H "Authorization: Bearer $ACCESS_TOKEN" 2>&1) || true

    if echo "$LIST_RESPONSE" | grep -q "email\|user"; then
        pass "Authenticated request (GET /api/users/profile)"
    else
        fail "Authenticated request -- Response: $LIST_RESPONSE"
    fi
else
    skip "Authenticated request (no access token)"
fi

echo ""
echo "=== Security ==="

# CORS
CORS_RESPONSE=$(curl -s -I -X OPTIONS "$GATEWAY_URL/api/users" \
    -H "Origin: http://example.com" \
    -H "Access-Control-Request-Method: GET" 2>&1) || true

if echo "$CORS_RESPONSE" | grep -qi "access-control"; then
    pass "CORS headers present"
else
    skip "CORS headers (not enabled or not applicable)"
fi

# Rate limiting
RATE_LIMIT_COUNT=0
for i in {1..5}; do
    if check_url "$GATEWAY_URL/health/live"; then
        ((RATE_LIMIT_COUNT++))
    fi
done

if [ $RATE_LIMIT_COUNT -ge 3 ]; then
    pass "Rate limiting allows normal traffic ($RATE_LIMIT_COUNT/5 succeeded)"
else
    fail "Rate limiting too aggressive (only $RATE_LIMIT_COUNT/5 succeeded)"
fi

# Path traversal
SECURITY_CODE=$(curl -s -w "%{http_code}" -o /dev/null \
    "$GATEWAY_URL/api/users/../../../etc/passwd" 2>&1) || true

if [ "$SECURITY_CODE" = "400" ] || [ "$SECURITY_CODE" = "404" ]; then
    pass "Path traversal protection (HTTP $SECURITY_CODE)"
else
    fail "Path traversal protection (HTTP $SECURITY_CODE, expected 400 or 404)"
fi

echo ""
echo "=== Frontend Services ==="

if check_url "$STUDENT_FRONTEND_URL" 3; then
    pass "Student frontend (port 3000)"
else
    fail "Student frontend (port 3000)"
fi

if check_url "$PROVIDER_FRONTEND_URL" 3; then
    pass "Provider frontend (port 3002)"
else
    fail "Provider frontend (port 3002)"
fi

if check_url "$ADMIN_FRONTEND_URL" 3; then
    pass "Admin frontend (port 3003)"
else
    fail "Admin frontend (port 3003)"
fi

# Summary
echo ""
echo "============================================"
echo " Smoke Test Results"
echo "============================================"
echo -e " Passed:  ${GREEN}$TESTS_PASSED${NC}"
echo -e " Failed:  ${RED}$TESTS_FAILED${NC}"
echo -e " Skipped: ${YELLOW}$TESTS_SKIPPED${NC}"
echo "============================================"

if [ $TESTS_FAILED -eq 0 ]; then
    echo -e "${GREEN}All tests passed!${NC}"
    exit 0
else
    echo -e "${RED}Some tests failed.${NC}"
    exit 1
fi
