#!/bin/bash

# Service Routing Test Script
# Tests each service individually through the API Gateway

set -e

GATEWAY_URL="http://localhost:8080"
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo "=========================================="
echo "Service Routing Test Suite"
echo "=========================================="
echo ""

# Function to test if a service is healthy
test_service_health() {
    local service_name=$1
    local port=$2
    
    echo -e "${YELLOW}Testing ${service_name} health...${NC}"
    
    if curl -s -f "http://localhost:${port}/health" > /dev/null 2>&1; then
        echo -e "${GREEN}✓ ${service_name} is healthy${NC}"
        return 0
    else
        echo -e "${RED}✗ ${service_name} is not responding${NC}"
        return 1
    fi
}

# Function to test gateway health
test_gateway_health() {
    echo -e "${YELLOW}Testing API Gateway health...${NC}"
    
    if curl -s -f "${GATEWAY_URL}/health" > /dev/null 2>&1; then
        echo -e "${GREEN}✓ API Gateway is healthy${NC}"
        return 0
    else
        echo -e "${RED}✗ API Gateway is not responding${NC}"
        return 1
    fi
}

# Test 1: User Auth Service
test_user_auth_service() {
    echo ""
    echo "=========================================="
    echo "Test 1: User Auth Service"
    echo "=========================================="
    
    # Test registration
    echo -e "${YELLOW}Testing user registration...${NC}"
    REGISTER_RESPONSE=$(curl -s -X POST "${GATEWAY_URL}/api/auth/register" \
        -H "Content-Type: application/json" \
        -d '{
            "email": "test@example.com",
            "password": "Test123!@#",
            "first_name": "Test",
            "last_name": "User",
            "phone": "+14155552671"
        }')
    
    if echo "$REGISTER_RESPONSE" | grep -q "access_token"; then
        echo -e "${GREEN}✓ User registration successful${NC}"
        ACCESS_TOKEN=$(echo "$REGISTER_RESPONSE" | grep -o '"access_token":"[^"]*"' | cut -d'"' -f4)
        USER_ID=$(echo "$REGISTER_RESPONSE" | grep -o '"id":"[^"]*"' | cut -d'"' -f4)
        echo "  Access Token: ${ACCESS_TOKEN:0:20}..."
        echo "  User ID: $USER_ID"
    else
        echo -e "${RED}✗ User registration failed${NC}"
        echo "  Response: $REGISTER_RESPONSE"
        return 1
    fi
    
    # Test login
    echo -e "${YELLOW}Testing user login...${NC}"
    LOGIN_RESPONSE=$(curl -s -X POST "${GATEWAY_URL}/api/auth/login" \
        -H "Content-Type: application/json" \
        -d '{
            "email": "test@example.com",
            "password": "Test123!@#"
        }')
    
    if echo "$LOGIN_RESPONSE" | grep -q "access_token"; then
        echo -e "${GREEN}✓ User login successful${NC}"
    else
        echo -e "${RED}✗ User login failed${NC}"
        echo "  Response: $LOGIN_RESPONSE"
        return 1
    fi
    
    # Test token validation
    echo -e "${YELLOW}Testing token validation...${NC}"
    VALIDATE_RESPONSE=$(curl -s -X POST "${GATEWAY_URL}/api/auth/validate" \
        -H "Content-Type: application/json" \
        -d "{\"token\": \"$ACCESS_TOKEN\"}")
    
    if echo "$VALIDATE_RESPONSE" | grep -q "valid"; then
        echo -e "${GREEN}✓ Token validation successful${NC}"
    else
        echo -e "${RED}✗ Token validation failed${NC}"
        echo "  Response: $VALIDATE_RESPONSE"
        return 1
    fi
    
    # Test protected endpoint - Get User
    echo -e "${YELLOW}Testing protected endpoint (Get User)...${NC}"
    USER_RESPONSE=$(curl -s -X GET "${GATEWAY_URL}/api/users/${USER_ID}" \
        -H "Authorization: Bearer $ACCESS_TOKEN")
    
    if echo "$USER_RESPONSE" | grep -q "email"; then
        echo -e "${GREEN}✓ Protected endpoint access successful${NC}"
    else
        echo -e "${RED}✗ Protected endpoint access failed${NC}"
        echo "  Response: $USER_RESPONSE"
        return 1
    fi
    
    echo -e "${GREEN}✓ All User Auth Service tests passed${NC}"
    return 0
}

# Test 2: Course Service
test_course_service() {
    echo ""
    echo "=========================================="
    echo "Test 2: Course Service"
    echo "=========================================="
    
    # First, create a user and get token
    echo -e "${YELLOW}Creating test user for course tests...${NC}"
    REGISTER_RESPONSE=$(curl -s -X POST "${GATEWAY_URL}/api/auth/register" \
        -H "Content-Type: application/json" \
        -d '{
            "email": "instructor@example.com",
            "password": "Test123!@#",
            "first_name": "Instructor",
            "last_name": "User",
            "phone": "+14155552672"
        }')
    
    ACCESS_TOKEN=$(echo "$REGISTER_RESPONSE" | grep -o '"access_token":"[^"]*"' | cut -d'"' -f4)
    INSTRUCTOR_ID=$(echo "$REGISTER_RESPONSE" | grep -o '"id":"[^"]*"' | cut -d'"' -f4)
    
    # Test course creation
    echo -e "${YELLOW}Testing course creation...${NC}"
    COURSE_RESPONSE=$(curl -s -X POST "${GATEWAY_URL}/api/courses" \
        -H "Authorization: Bearer $ACCESS_TOKEN" \
        -H "Content-Type: application/json" \
        -d "{
            \"title\": \"Introduction to Computer Science\",
            \"description\": \"Learn the basics of programming\",
            \"term\": \"Fall 2024\",
            \"syllabus\": \"Week 1: Variables, Week 2: Functions\",
            \"instructor_id\": \"$INSTRUCTOR_ID\",
            \"metadata\": {
                \"max_students\": 30,
                \"department\": \"CS\",
                \"course_code\": \"CS101\",
                \"credits\": 3
            }
        }")
    
    if echo "$COURSE_RESPONSE" | grep -q "id"; then
        echo -e "${GREEN}✓ Course creation successful${NC}"
        COURSE_ID=$(echo "$COURSE_RESPONSE" | grep -o '"id":"[^"]*"' | head -1 | cut -d'"' -f4)
        echo "  Course ID: $COURSE_ID"
    else
        echo -e "${RED}✗ Course creation failed${NC}"
        echo "  Response: $COURSE_RESPONSE"
        return 1
    fi
    
    # Test get course
    echo -e "${YELLOW}Testing get course...${NC}"
    GET_COURSE_RESPONSE=$(curl -s -X GET "${GATEWAY_URL}/api/courses/${COURSE_ID}" \
        -H "Authorization: Bearer $ACCESS_TOKEN")
    
    if echo "$GET_COURSE_RESPONSE" | grep -q "Introduction to Computer Science"; then
        echo -e "${GREEN}✓ Get course successful${NC}"
    else
        echo -e "${RED}✗ Get course failed${NC}"
        echo "  Response: $GET_COURSE_RESPONSE"
        return 1
    fi
    
    # Test list courses
    echo -e "${YELLOW}Testing list courses...${NC}"
    LIST_COURSES_RESPONSE=$(curl -s -X GET "${GATEWAY_URL}/api/courses?page=1&page_size=10" \
        -H "Authorization: Bearer $ACCESS_TOKEN")
    
    if echo "$LIST_COURSES_RESPONSE" | grep -q "courses"; then
        echo -e "${GREEN}✓ List courses successful${NC}"
    else
        echo -e "${RED}✗ List courses failed${NC}"
        echo "  Response: $LIST_COURSES_RESPONSE"
        return 1
    fi
    
    echo -e "${GREEN}✓ All Course Service tests passed${NC}"
    return 0
}

# Test 3: Assignment Grading Service
test_assignment_service() {
    echo ""
    echo "=========================================="
    echo "Test 3: Assignment Grading Service"
    echo "=========================================="
    
    # Create user and course first
    echo -e "${YELLOW}Setting up test data...${NC}"
    REGISTER_RESPONSE=$(curl -s -X POST "${GATEWAY_URL}/api/auth/register" \
        -H "Content-Type: application/json" \
        -d '{
            "email": "teacher@example.com",
            "password": "Test123!@#",
            "first_name": "Teacher",
            "last_name": "User",
            "phone": "+14155552673"
        }')
    
    ACCESS_TOKEN=$(echo "$REGISTER_RESPONSE" | grep -o '"access_token":"[^"]*"' | cut -d'"' -f4)
    INSTRUCTOR_ID=$(echo "$REGISTER_RESPONSE" | grep -o '"id":"[^"]*"' | cut -d'"' -f4)
    
    # Create a course
    COURSE_RESPONSE=$(curl -s -X POST "${GATEWAY_URL}/api/courses" \
        -H "Authorization: Bearer $ACCESS_TOKEN" \
        -H "Content-Type: application/json" \
        -d "{
            \"title\": \"Math 101\",
            \"description\": \"Basic Mathematics\",
            \"term\": \"Fall 2024\",
            \"instructor_id\": \"$INSTRUCTOR_ID\"
        }")
    
    COURSE_ID=$(echo "$COURSE_RESPONSE" | grep -o '"id":"[^"]*"' | head -1 | cut -d'"' -f4)
    
    # Test assignment creation
    echo -e "${YELLOW}Testing assignment creation...${NC}"
    ASSIGNMENT_RESPONSE=$(curl -s -X POST "${GATEWAY_URL}/api/assignments" \
        -H "Authorization: Bearer $ACCESS_TOKEN" \
        -H "Content-Type: application/json" \
        -d "{
            \"course_id\": \"$COURSE_ID\",
            \"title\": \"Homework 1\",
            \"description\": \"Complete exercises 1-10\",
            \"max_points\": 100,
            \"due_date\": \"2024-12-31T23:59:59Z\",
            \"late_policy\": {
                \"penalty_percent_per_day\": 10,
                \"max_late_days\": 3
            }
        }")
    
    if echo "$ASSIGNMENT_RESPONSE" | grep -q "id"; then
        echo -e "${GREEN}✓ Assignment creation successful${NC}"
        ASSIGNMENT_ID=$(echo "$ASSIGNMENT_RESPONSE" | grep -o '"id":"[^"]*"' | head -1 | cut -d'"' -f4)
        echo "  Assignment ID: $ASSIGNMENT_ID"
    else
        echo -e "${RED}✗ Assignment creation failed${NC}"
        echo "  Response: $ASSIGNMENT_RESPONSE"
        return 1
    fi
    
    # Test get assignment
    echo -e "${YELLOW}Testing get assignment...${NC}"
    GET_ASSIGNMENT_RESPONSE=$(curl -s -X GET "${GATEWAY_URL}/api/assignments/${ASSIGNMENT_ID}" \
        -H "Authorization: Bearer $ACCESS_TOKEN")
    
    if echo "$GET_ASSIGNMENT_RESPONSE" | grep -q "Homework 1"; then
        echo -e "${GREEN}✓ Get assignment successful${NC}"
    else
        echo -e "${RED}✗ Get assignment failed${NC}"
        echo "  Response: $GET_ASSIGNMENT_RESPONSE"
        return 1
    fi
    
    # Test list assignments
    echo -e "${YELLOW}Testing list assignments...${NC}"
    LIST_ASSIGNMENTS_RESPONSE=$(curl -s -X GET "${GATEWAY_URL}/api/assignments?course_id=${COURSE_ID}&page=1&page_size=10" \
        -H "Authorization: Bearer $ACCESS_TOKEN")
    
    if echo "$LIST_ASSIGNMENTS_RESPONSE" | grep -q "assignments"; then
        echo -e "${GREEN}✓ List assignments successful${NC}"
    else
        echo -e "${RED}✗ List assignments failed${NC}"
        echo "  Response: $LIST_ASSIGNMENTS_RESPONSE"
        return 1
    fi
    
    echo -e "${GREEN}✓ All Assignment Service tests passed${NC}"
    return 0
}

# Main execution
main() {
    echo "Waiting for services to be ready..."
    sleep 5
    
    # Test gateway health
    if ! test_gateway_health; then
        echo -e "${RED}API Gateway is not healthy. Exiting.${NC}"
        exit 1
    fi
    
    # Test individual service health (only for services with HTTP endpoints)
    test_service_health "Course Service" 3001
    echo -e "${YELLOW}Note: User Auth and Assignment services use gRPC only (no HTTP health endpoint)${NC}"
    
    echo ""
    echo "=========================================="
    echo "Starting Routing Tests"
    echo "=========================================="
    
    # Run tests
    FAILED=0
    
    if ! test_user_auth_service; then
        FAILED=$((FAILED + 1))
    fi
    
    if ! test_course_service; then
        FAILED=$((FAILED + 1))
    fi
    
    if ! test_assignment_service; then
        FAILED=$((FAILED + 1))
    fi
    
    echo ""
    echo "=========================================="
    echo "Test Summary"
    echo "=========================================="
    
    if [ $FAILED -eq 0 ]; then
        echo -e "${GREEN}✓ All tests passed!${NC}"
        exit 0
    else
        echo -e "${RED}✗ $FAILED test suite(s) failed${NC}"
        exit 1
    fi
}

main
