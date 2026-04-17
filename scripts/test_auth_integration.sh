#!/bin/bash

# Auth Integration Test Script
# Tests the complete authentication flow including token validation across all services

set -e

GATEWAY_URL="http://localhost:8080"
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Test counters
TESTS_PASSED=0
TESTS_FAILED=0
TOTAL_TESTS=0

# Generate unique identifiers for this test run
TEST_RUN_ID=$(date +%s)
UNIQUE_EMAIL="testuser${TEST_RUN_ID}@example.com"

echo "=========================================="
echo "Auth Integration Test Suite"
echo "=========================================="
echo "Test Run ID: $TEST_RUN_ID"
echo "Gateway URL: $GATEWAY_URL"
echo ""

# Function to print test header
print_test_header() {
    local test_name=$1
    echo ""
    echo -e "${BLUE}=========================================="
    echo "Test: $test_name"
    echo -e "==========================================${NC}"
}

# Function to record test result
record_test() {
    local test_name=$1
    local passed=$2
    
    TOTAL_TESTS=$((TOTAL_TESTS + 1))
    
    if [ "$passed" = "true" ]; then
        echo -e "${GREEN}✓ PASS: $test_name${NC}"
        TESTS_PASSED=$((TESTS_PASSED + 1))
    else
        echo -e "${RED}✗ FAIL: $test_name${NC}"
        TESTS_FAILED=$((TESTS_FAILED + 1))
    fi
}

# Function to check HTTP status code
check_status() {
    local expected=$1
    local actual=$2
    local test_name=$3
    
    if [ "$actual" = "$expected" ]; then
        record_test "$test_name (HTTP $expected)" "true"
        return 0
    else
        record_test "$test_name (expected $expected, got $actual)" "false"
        return 1
    fi
}

# Function to check if response contains a string
check_response_contains() {
    local response=$1
    local expected_string=$2
    local test_name=$3
    
    if echo "$response" | grep -q "$expected_string"; then
        record_test "$test_name" "true"
        return 0
    else
        record_test "$test_name (missing: $expected_string)" "false"
        echo "  Response: $response"
        return 1
    fi
}

# Wait for services to be ready
echo -e "${YELLOW}Waiting for services to be ready...${NC}"
sleep 3

# Check if gateway is accessible
if ! curl -s -f "${GATEWAY_URL}/health" > /dev/null 2>&1; then
    echo -e "${RED}✗ API Gateway is not accessible at ${GATEWAY_URL}${NC}"
    echo "Please ensure all services are running with: docker-compose up"
    exit 1
fi
echo -e "${GREEN}✓ API Gateway is accessible${NC}"

# ==========================================
# Test 1: User Registration
# ==========================================
print_test_header "1. User Registration"

echo "Registering user: $UNIQUE_EMAIL"
REGISTER_RESPONSE=$(curl -s -w "\nHTTP_STATUS:%{http_code}" -X POST "${GATEWAY_URL}/api/auth/register" \
    -H "Content-Type: application/json" \
    -d "{
        \"email\": \"$UNIQUE_EMAIL\",
        \"password\": \"TestPass123!\",
        \"first_name\": \"Test\",
        \"last_name\": \"User\",
        \"phone\": \"+1234567890\"
    }")

HTTP_STATUS=$(echo "$REGISTER_RESPONSE" | grep "HTTP_STATUS" | cut -d: -f2)
BODY=$(echo "$REGISTER_RESPONSE" | sed '/HTTP_STATUS/d')

check_status "200" "$HTTP_STATUS" "User registration"

if echo "$BODY" | grep -q "access_token"; then
    ACCESS_TOKEN=$(echo "$BODY" | python3 -c "import sys, json; print(json.load(sys.stdin)['access_token'])" 2>/dev/null)
    USER_ID=$(echo "$BODY" | python3 -c "import sys, json; print(json.load(sys.stdin)['user']['id'])" 2>/dev/null)
    record_test "Registration returns access_token" "true"
    echo "  User ID: $USER_ID"
    echo "  Token: ${ACCESS_TOKEN:0:30}..."
else
    record_test "Registration returns access_token" "false"
    echo "  Response: $BODY"
    exit 1
fi

# ==========================================
# Test 2: User Login
# ==========================================
print_test_header "2. User Login"

echo "Logging in with: $UNIQUE_EMAIL"
LOGIN_RESPONSE=$(curl -s -w "\nHTTP_STATUS:%{http_code}" -X POST "${GATEWAY_URL}/api/auth/login" \
    -H "Content-Type: application/json" \
    -d "{
        \"email\": \"$UNIQUE_EMAIL\",
        \"password\": \"TestPass123!\"
    }")

HTTP_STATUS=$(echo "$LOGIN_RESPONSE" | grep "HTTP_STATUS" | cut -d: -f2)
BODY=$(echo "$LOGIN_RESPONSE" | sed '/HTTP_STATUS/d')

check_status "200" "$HTTP_STATUS" "User login"

if echo "$BODY" | grep -q "access_token"; then
    LOGIN_TOKEN=$(echo "$BODY" | python3 -c "import sys, json; print(json.load(sys.stdin)['access_token'])" 2>/dev/null)
    record_test "Login returns access_token" "true"
    echo "  Token: ${LOGIN_TOKEN:0:30}..."
    # Use the login token for subsequent tests
    ACCESS_TOKEN=$LOGIN_TOKEN
else
    record_test "Login returns access_token" "false"
    echo "  Response: $BODY"
fi

# ==========================================
# Test 3: Protected Endpoint with Valid Token
# ==========================================
print_test_header "3. Protected Endpoint with Valid Token"

echo "Testing GET /api/users/${USER_ID} with valid token"
USER_RESPONSE=$(curl -s -w "\nHTTP_STATUS:%{http_code}" \
    -H "Authorization: Bearer $ACCESS_TOKEN" \
    "${GATEWAY_URL}/api/users/${USER_ID}")

HTTP_STATUS=$(echo "$USER_RESPONSE" | grep "HTTP_STATUS" | cut -d: -f2)
BODY=$(echo "$USER_RESPONSE" | sed '/HTTP_STATUS/d')

check_status "200" "$HTTP_STATUS" "Protected endpoint with valid token"
check_response_contains "$BODY" "$UNIQUE_EMAIL" "Response contains user email"

# ==========================================
# Test 4: Protected Endpoint with Invalid Token
# ==========================================
print_test_header "4. Protected Endpoint with Invalid Token"

INVALID_TOKEN="eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIiwiaWF0IjoxNTE2MjM5MDIyfQ.SflKxwRJSMeKKF2QT4fwpMeJf36POk6yJV_adQssw5c"

echo "Testing GET /api/users/${USER_ID} with invalid token"
INVALID_RESPONSE=$(curl -s -w "\nHTTP_STATUS:%{http_code}" \
    -H "Authorization: Bearer $INVALID_TOKEN" \
    "${GATEWAY_URL}/api/users/${USER_ID}")

HTTP_STATUS=$(echo "$INVALID_RESPONSE" | grep "HTTP_STATUS" | cut -d: -f2)
BODY=$(echo "$INVALID_RESPONSE" | sed '/HTTP_STATUS/d')

check_status "401" "$HTTP_STATUS" "Protected endpoint with invalid token"

# ==========================================
# Test 5: Protected Endpoint with No Token
# ==========================================
print_test_header "5. Protected Endpoint with No Token"

echo "Testing GET /api/users/${USER_ID} without token"
NO_TOKEN_RESPONSE=$(curl -s -w "\nHTTP_STATUS:%{http_code}" \
    "${GATEWAY_URL}/api/users/${USER_ID}")

HTTP_STATUS=$(echo "$NO_TOKEN_RESPONSE" | grep "HTTP_STATUS" | cut -d: -f2)

check_status "401" "$HTTP_STATUS" "Protected endpoint without token"

# ==========================================
# Test 6: Token Validation Across Services
# ==========================================
print_test_header "6. Token Validation Across All Services"

# Test 6.1: User Service Protected Endpoint
echo ""
echo -e "${YELLOW}6.1 Testing User Service${NC}"
echo "Testing GET /api/users (list users)"
USERS_LIST_RESPONSE=$(curl -s -w "\nHTTP_STATUS:%{http_code}" \
    -H "Authorization: Bearer $ACCESS_TOKEN" \
    "${GATEWAY_URL}/api/users")

HTTP_STATUS=$(echo "$USERS_LIST_RESPONSE" | grep "HTTP_STATUS" | cut -d: -f2)
check_status "200" "$HTTP_STATUS" "User service with valid token"

# Test 6.2: Course Service Protected Endpoint
echo ""
echo -e "${YELLOW}6.2 Testing Course Service${NC}"
echo "Creating a course..."
COURSE_RESPONSE=$(curl -s -w "\nHTTP_STATUS:%{http_code}" -X POST "${GATEWAY_URL}/api/courses" \
    -H "Authorization: Bearer $ACCESS_TOKEN" \
    -H "Content-Type: application/json" \
    -d "{
        \"title\": \"Test Course ${TEST_RUN_ID}\",
        \"description\": \"Integration test course\",
        \"term\": \"Fall 2024\",
        \"syllabus\": \"Test syllabus\",
        \"instructor_id\": \"$USER_ID\",
        \"metadata\": {
            \"max_students\": 30,
            \"department\": \"CS\",
            \"course_code\": \"CS${TEST_RUN_ID}\",
            \"credits\": 3
        }
    }")

HTTP_STATUS=$(echo "$COURSE_RESPONSE" | grep "HTTP_STATUS" | cut -d: -f2)
BODY=$(echo "$COURSE_RESPONSE" | sed '/HTTP_STATUS/d')

check_status "201" "$HTTP_STATUS" "Course service with valid token"

if echo "$BODY" | grep -q "id"; then
    COURSE_ID=$(echo "$BODY" | python3 -c "import sys, json; print(json.load(sys.stdin)['id'])" 2>/dev/null)
    echo "  Course ID: $COURSE_ID"
else
    echo "  Warning: Could not extract course ID"
    COURSE_ID="test-course-id"
fi

# Test 6.3: Assignment Service Protected Endpoint
echo ""
echo -e "${YELLOW}6.3 Testing Assignment Service${NC}"
echo "Creating an assignment..."
ASSIGNMENT_RESPONSE=$(curl -s -w "\nHTTP_STATUS:%{http_code}" -X POST "${GATEWAY_URL}/api/assignments" \
    -H "Authorization: Bearer $ACCESS_TOKEN" \
    -H "Content-Type: application/json" \
    -d "{
        \"course_id\": \"$COURSE_ID\",
        \"title\": \"Test Assignment ${TEST_RUN_ID}\",
        \"description\": \"Integration test assignment\",
        \"max_points\": 100,
        \"due_date\": \"2024-12-31T23:59:59Z\",
        \"late_policy\": {
            \"penalty_percent_per_day\": 10,
            \"max_late_days\": 3
        }
    }")

HTTP_STATUS=$(echo "$ASSIGNMENT_RESPONSE" | grep "HTTP_STATUS" | cut -d: -f2)
check_status "201" "$HTTP_STATUS" "Assignment service with valid token"

# ==========================================
# Test 7: Error Handling and Logging Verification
# ==========================================
print_test_header "7. Error Handling and Logging"

echo ""
echo -e "${YELLOW}7.1 Checking API Gateway Logs${NC}"
echo "Looking for token validation logs in api-gateway..."

# Check if docker-compose is running
if docker-compose ps api-gateway 2>/dev/null | grep -q "Up"; then
    GATEWAY_LOGS=$(docker-compose logs --tail=50 api-gateway 2>/dev/null | grep -i "token\|auth\|validate" || echo "")
    
    if [ -n "$GATEWAY_LOGS" ]; then
        record_test "API Gateway logs contain auth activity" "true"
        echo "  Sample log entries:"
        echo "$GATEWAY_LOGS" | head -5 | sed 's/^/    /'
    else
        record_test "API Gateway logs contain auth activity" "false"
        echo "  No auth-related logs found"
    fi
else
    echo -e "${YELLOW}  Skipping: Docker Compose not available${NC}"
fi

echo ""
echo -e "${YELLOW}7.2 Checking User Auth Service Logs${NC}"
echo "Looking for validation attempts in user-auth-service..."

if docker-compose ps user-auth-service 2>/dev/null | grep -q "Up"; then
    AUTH_SERVICE_LOGS=$(docker-compose logs --tail=50 user-auth-service 2>/dev/null | grep -i "validate\|token" || echo "")
    
    if [ -n "$AUTH_SERVICE_LOGS" ]; then
        record_test "User auth service logs validation attempts" "true"
        echo "  Sample log entries:"
        echo "$AUTH_SERVICE_LOGS" | head -5 | sed 's/^/    /'
    else
        record_test "User auth service logs validation attempts" "false"
        echo "  No validation logs found"
    fi
else
    echo -e "${YELLOW}  Skipping: Docker Compose not available${NC}"
fi

echo ""
echo -e "${YELLOW}7.3 Verifying Error Messages${NC}"
echo "Testing that error messages don't leak sensitive information..."

# Test with malformed token
MALFORMED_RESPONSE=$(curl -s -w "\nHTTP_STATUS:%{http_code}" \
    -H "Authorization: Bearer malformed-token-12345" \
    "${GATEWAY_URL}/api/users/${USER_ID}")

BODY=$(echo "$MALFORMED_RESPONSE" | sed '/HTTP_STATUS/d')

# Check that error message is generic and doesn't expose internals
if echo "$BODY" | grep -qiE "unauthorized|invalid.*token|authentication.*required"; then
    if ! echo "$BODY" | grep -qiE "secret|key|signature|internal|stack|error.*trace"; then
        record_test "Error messages are appropriate and secure" "true"
        echo "  Error message: $(echo "$BODY" | head -1)"
    else
        record_test "Error messages don't leak sensitive info" "false"
        echo "  Warning: Error message may contain sensitive information"
    fi
else
    record_test "Error messages are present" "false"
fi

# ==========================================
# Test Summary
# ==========================================
echo ""
echo "=========================================="
echo "Test Summary"
echo "=========================================="
echo ""
echo "Total Tests: $TOTAL_TESTS"
echo -e "${GREEN}Passed: $TESTS_PASSED${NC}"
echo -e "${RED}Failed: $TESTS_FAILED${NC}"
echo ""

if [ $TESTS_FAILED -eq 0 ]; then
    echo -e "${GREEN}=========================================="
    echo "✓ ALL TESTS PASSED!"
    echo -e "==========================================${NC}"
    echo ""
    echo "Authentication integration is working correctly:"
    echo "  ✓ User registration and login functional"
    echo "  ✓ JWT tokens generated and validated"
    echo "  ✓ Protected endpoints enforce authentication"
    echo "  ✓ Token validation works across all services"
    echo "  ✓ Error handling is appropriate"
    echo ""
    exit 0
else
    echo -e "${RED}=========================================="
    echo "✗ SOME TESTS FAILED"
    echo -e "==========================================${NC}"
    echo ""
    echo "Please review the failed tests above and check:"
    echo "  - All services are running (docker-compose ps)"
    echo "  - Service logs for errors (docker-compose logs)"
    echo "  - Network connectivity between services"
    echo ""
    exit 1
fi
