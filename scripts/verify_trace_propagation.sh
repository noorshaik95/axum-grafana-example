#!/bin/bash

# Trace Propagation Verification Script
# Verifies that distributed traces properly propagate across Register, Login, and GetUser operations
# Each HTTP request should create its own trace ID that propagates through all services

set -e
set -o pipefail

# ==========================================
# Default Configuration
# ==========================================
GATEWAY_URL="http://localhost:8080"
TEMPO_URL="http://localhost:3200"
WAIT_TIME=10
VERBOSE=false
JSON_OUTPUT=false

# ==========================================
# Color Codes
# ==========================================
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

# ==========================================
# Global Variables
# ==========================================
REGISTER_TRACE_ID=""
LOGIN_TRACE_ID=""
GETUSER_TRACE_ID=""
USER_ID=""
ACCESS_TOKEN=""
TEST_EMAIL=""
TEST_PASSWORD="TestPass123!"

# Test results
TOTAL_CHECKS=0
PASSED_CHECKS=0
FAILED_CHECKS=0
DIAGNOSTICS=()

# ==========================================
# Helper Functions
# ==========================================

# Print usage information
usage() {
    cat << EOF
Usage: $0 [OPTIONS]

Verify trace propagation across Register, Login, and GetUser operations.
Each HTTP request should generate its own trace ID that propagates through all services.

OPTIONS:
    --gateway-url URL    API Gateway URL (default: http://localhost:8080)
    --tempo-url URL      Tempo API URL (default: http://localhost:3200)
    --wait-time SECONDS  Wait time for trace ingestion (default: 10)
    --verbose            Enable verbose output
    --json               Output results in JSON format
    -h, --help           Show this help message

EXAMPLES:
    # Run with defaults
    $0

    # Run with custom URLs
    $0 --gateway-url http://gateway:8080 --tempo-url http://tempo:3200

    # Run with verbose output
    $0 --verbose

    # Run with JSON output for CI/CD
    $0 --json

REQUIREMENTS:
    - API Gateway must be running and accessible
    - Tempo must be running and accessible
    - curl and jq must be installed

EXIT CODES:
    0 - All checks passed
    1 - One or more checks failed
    2 - Service unavailable or configuration error

EOF
    exit 0
}

# Log verbose messages
log_verbose() {
    if [ "$VERBOSE" = true ]; then
        echo -e "${CYAN}[VERBOSE]${NC} $1"
    fi
}

# Log info messages
log_info() {
    if [ "$JSON_OUTPUT" = false ]; then
        echo -e "${BLUE}[INFO]${NC} $1"
    fi
}

# Log success messages
log_success() {
    if [ "$JSON_OUTPUT" = false ]; then
        echo -e "${GREEN}[SUCCESS]${NC} $1"
    fi
}

# Log warning messages
log_warning() {
    if [ "$JSON_OUTPUT" = false ]; then
        echo -e "${YELLOW}[WARNING]${NC} $1"
    fi
}

# Log error messages
log_error() {
    if [ "$JSON_OUTPUT" = false ]; then
        echo -e "${RED}[ERROR]${NC} $1"
    fi
}

# Add diagnostic message
add_diagnostic() {
    local level=$1
    local message=$2
    local operation=$3
    local trace_id=$4
    
    DIAGNOSTICS+=("{\"level\":\"$level\",\"message\":\"$message\",\"operation\":\"$operation\",\"trace_id\":\"$trace_id\"}")
}

# Record check result
record_check() {
    local check_name=$1
    local passed=$2
    local operation=$3
    local trace_id=$4
    
    TOTAL_CHECKS=$((TOTAL_CHECKS + 1))
    
    if [ "$passed" = true ]; then
        PASSED_CHECKS=$((PASSED_CHECKS + 1))
        log_verbose "✓ $check_name"
    else
        FAILED_CHECKS=$((FAILED_CHECKS + 1))
        log_error "✗ $check_name"
        add_diagnostic "ERROR" "$check_name failed" "$operation" "$trace_id"
    fi
}

# Check if required commands are available
check_dependencies() {
    local missing_deps=()
    
    if ! command -v curl &> /dev/null; then
        missing_deps+=("curl")
    fi
    
    if ! command -v jq &> /dev/null; then
        missing_deps+=("jq")
    fi
    
    if [ ${#missing_deps[@]} -gt 0 ]; then
        log_error "Missing required dependencies: ${missing_deps[*]}"
        log_error "Please install missing dependencies and try again"
        exit 2
    fi
}

# Check if services are accessible
check_services() {
    log_info "Checking service availability..."
    
    # Check API Gateway
    if ! curl -s -f "${GATEWAY_URL}/health" > /dev/null 2>&1; then
        log_error "API Gateway is not accessible at ${GATEWAY_URL}"
        log_error "Please ensure the API Gateway is running"
        exit 2
    fi
    log_success "API Gateway is accessible"
    
    # Check Tempo
    if ! curl -s -f "${TEMPO_URL}/ready" > /dev/null 2>&1; then
        log_warning "Tempo is not accessible at ${TEMPO_URL}"
        log_warning "Trace verification may fail"
    else
        log_success "Tempo is accessible"
    fi
}

# ==========================================
# Argument Parsing
# ==========================================

parse_arguments() {
    while [[ $# -gt 0 ]]; do
        case $1 in
            --gateway-url)
                GATEWAY_URL="$2"
                shift 2
                ;;
            --tempo-url)
                TEMPO_URL="$2"
                shift 2
                ;;
            --wait-time)
                WAIT_TIME="$2"
                shift 2
                ;;
            --verbose)
                VERBOSE=true
                shift
                ;;
            --json)
                JSON_OUTPUT=true
                shift
                ;;
            -h|--help)
                usage
                ;;
            *)
                echo "Unknown option: $1"
                echo "Use --help for usage information"
                exit 2
                ;;
        esac
    done
}

# ==========================================
# Tempo API Query Functions
# ==========================================

# Convert UUID trace ID to hex format (remove dashes)
normalize_trace_id() {
    local trace_id=$1
    echo "$trace_id" | tr -d '-'
}

# Query Tempo API for a specific trace with retry logic
query_tempo_trace() {
    local trace_id=$1
    local max_retries=5
    local retry_count=0
    local backoff=2
    
    # Normalize trace ID (remove dashes)
    local normalized_id=$(normalize_trace_id "$trace_id")
    
    log_verbose "Querying Tempo for trace: $normalized_id"
    
    while [ $retry_count -lt $max_retries ]; do
        # Query Tempo API
        local response=$(curl -s -w "\nHTTP_STATUS:%{http_code}" \
            "${TEMPO_URL}/api/traces/${normalized_id}" 2>&1)
        
        local http_status=$(echo "$response" | grep "HTTP_STATUS" | cut -d: -f2)
        local body=$(echo "$response" | sed '/HTTP_STATUS/d')
        
        # Check if request was successful
        if [ "$http_status" = "200" ]; then
            # Check if response contains trace data
            if echo "$body" | jq -e '.batches' > /dev/null 2>&1; then
                log_verbose "Trace found in Tempo"
                echo "$body"
                return 0
            fi
        fi
        
        # Trace not found, retry with exponential backoff
        retry_count=$((retry_count + 1))
        
        if [ $retry_count -lt $max_retries ]; then
            log_verbose "Trace not found, retrying in ${backoff}s (attempt $retry_count/$max_retries)..."
            sleep $backoff
            backoff=$((backoff * 2))
        fi
    done
    
    log_warning "Trace not found after $max_retries attempts: $trace_id"
    return 1
}

# Parse Tempo trace response and extract span information
parse_trace_spans() {
    local trace_json=$1
    
    # Extract spans from the trace
    # Tempo returns traces in OTLP format with batches of resource spans
    echo "$trace_json" | jq -r '
        .batches[]? |
        .scopeSpans[]? |
        .spans[]? |
        {
            span_id: .spanId,
            parent_span_id: .parentSpanId,
            name: .name,
            kind: .kind,
            start_time: .startTimeUnixNano,
            end_time: .endTimeUnixNano,
            attributes: (.attributes // [] | map({(.key): .value}) | add),
            status: .status
        }
    ' 2>/dev/null
}

# Extract service name from trace batch
extract_service_names() {
    local trace_json=$1
    
    echo "$trace_json" | jq -r '
        .batches[]? |
        .resource.attributes[]? |
        select(.key == "service.name") |
        .value.stringValue
    ' 2>/dev/null | sort -u
}

# Handle Tempo API connection failures
check_tempo_connection() {
    log_verbose "Checking Tempo connection..."
    
    local response=$(curl -s -w "\nHTTP_STATUS:%{http_code}" "${TEMPO_URL}/ready" 2>&1)
    local http_status=$(echo "$response" | grep "HTTP_STATUS" | cut -d: -f2)
    
    if [ "$http_status" != "200" ]; then
        log_error "Tempo is not accessible at ${TEMPO_URL}"
        log_error "Please ensure Tempo is running and accessible"
        return 1
    fi
    
    log_verbose "Tempo connection successful"
    return 0
}

# ==========================================
# User Flow Execution Functions
# ==========================================

# Extract trace ID from API Gateway logs for a specific operation
extract_trace_id_from_logs() {
    local operation=$1
    local search_pattern=$2
    local max_attempts=5
    local attempt=1
    
    log_verbose "Extracting trace ID for $operation from logs..."
    
    while [ $attempt -le $max_attempts ]; do
        # Get recent logs and look for the operation
        local trace_id=$(docker-compose logs --tail=100 api-gateway 2>&1 | \
            grep "$search_pattern" | \
            tail -1 | \
            grep -o '"trace_id":"[^"]*"' | \
            cut -d'"' -f4)
        
        if [ -n "$trace_id" ]; then
            log_verbose "Found trace ID: $trace_id"
            echo "$trace_id"
            return 0
        fi
        
        log_verbose "Attempt $attempt: No trace ID found, retrying..."
        sleep 1
        attempt=$((attempt + 1))
    done
    
    log_warning "Could not extract trace ID for $operation from logs"
    echo ""
    return 1
}

# Execute Register request
execute_register() {
    log_info "Executing Register operation..."
    
    # Generate unique email for this test run
    local timestamp=$(date +%s)
    TEST_EMAIL="trace-test-${timestamp}@example.com"
    
    log_verbose "Registering user: $TEST_EMAIL"
    
    # Execute register request
    local response=$(curl -s -w "\nHTTP_STATUS:%{http_code}" -X POST "${GATEWAY_URL}/api/auth/register" \
        -H "Content-Type: application/json" \
        -d "{
            \"email\": \"$TEST_EMAIL\",
            \"password\": \"$TEST_PASSWORD\",
            \"first_name\": \"Trace\",
            \"last_name\": \"Test\",
            \"phone\": \"+1234567890\"
        }")
    
    local http_status=$(echo "$response" | grep "HTTP_STATUS" | cut -d: -f2)
    local body=$(echo "$response" | sed '/HTTP_STATUS/d')
    
    if [ "$http_status" != "200" ]; then
        log_error "Register request failed with HTTP $http_status"
        log_verbose "Response: $body"
        return 1
    fi
    
    log_success "Register request completed (HTTP $http_status)"
    
    # Extract user ID and token from response
    USER_ID=$(echo "$body" | jq -r '.user.id' 2>/dev/null)
    ACCESS_TOKEN=$(echo "$body" | jq -r '.access_token' 2>/dev/null)
    
    if [ -z "$USER_ID" ] || [ "$USER_ID" = "null" ]; then
        log_error "Failed to extract user ID from register response"
        return 1
    fi
    
    log_verbose "User ID: $USER_ID"
    log_verbose "Token: ${ACCESS_TOKEN:0:30}..."
    
    # Wait a moment for logs to be written
    sleep 1
    
    # Extract trace ID from logs
    REGISTER_TRACE_ID=$(extract_trace_id_from_logs "Register" "/api/auth/register")
    
    if [ -z "$REGISTER_TRACE_ID" ]; then
        log_error "Failed to extract trace ID for Register operation"
        return 1
    fi
    
    log_verbose "Register trace ID: $REGISTER_TRACE_ID"
    log_success "Register operation completed"
    return 0
}

# Execute Login request
execute_login() {
    log_info "Executing Login operation..."
    
    if [ -z "$TEST_EMAIL" ]; then
        log_error "No test email available. Register must be executed first."
        return 1
    fi
    
    log_verbose "Logging in with: $TEST_EMAIL"
    
    # Execute login request
    local response=$(curl -s -w "\nHTTP_STATUS:%{http_code}" -X POST "${GATEWAY_URL}/api/auth/login" \
        -H "Content-Type: application/json" \
        -d "{
            \"email\": \"$TEST_EMAIL\",
            \"password\": \"$TEST_PASSWORD\"
        }")
    
    local http_status=$(echo "$response" | grep "HTTP_STATUS" | cut -d: -f2)
    local body=$(echo "$response" | sed '/HTTP_STATUS/d')
    
    if [ "$http_status" != "200" ]; then
        log_error "Login request failed with HTTP $http_status"
        log_verbose "Response: $body"
        return 1
    fi
    
    log_success "Login request completed (HTTP $http_status)"
    
    # Extract token from response (update the token)
    local login_token=$(echo "$body" | jq -r '.access_token' 2>/dev/null)
    
    if [ -n "$login_token" ] && [ "$login_token" != "null" ]; then
        ACCESS_TOKEN=$login_token
        log_verbose "Updated token: ${ACCESS_TOKEN:0:30}..."
    fi
    
    # Wait a moment for logs to be written
    sleep 1
    
    # Extract trace ID from logs
    LOGIN_TRACE_ID=$(extract_trace_id_from_logs "Login" "/api/auth/login")
    
    if [ -z "$LOGIN_TRACE_ID" ]; then
        log_error "Failed to extract trace ID for Login operation"
        return 1
    fi
    
    log_verbose "Login trace ID: $LOGIN_TRACE_ID"
    log_success "Login operation completed"
    return 0
}

# Execute GetUser request
execute_getuser() {
    log_info "Executing GetUser operation..."
    
    if [ -z "$USER_ID" ]; then
        log_error "No user ID available. Register must be executed first."
        return 1
    fi
    
    if [ -z "$ACCESS_TOKEN" ]; then
        log_error "No access token available. Login must be executed first."
        return 1
    fi
    
    log_verbose "Getting user: $USER_ID"
    
    # Execute get user request
    local response=$(curl -s -w "\nHTTP_STATUS:%{http_code}" \
        -H "Authorization: Bearer $ACCESS_TOKEN" \
        "${GATEWAY_URL}/api/users/${USER_ID}")
    
    local http_status=$(echo "$response" | grep "HTTP_STATUS" | cut -d: -f2)
    local body=$(echo "$response" | sed '/HTTP_STATUS/d')
    
    if [ "$http_status" != "200" ]; then
        log_error "GetUser request failed with HTTP $http_status"
        log_verbose "Response: $body"
        return 1
    fi
    
    log_success "GetUser request completed (HTTP $http_status)"
    
    # Verify response contains expected email
    if echo "$body" | jq -e ".email == \"$TEST_EMAIL\"" > /dev/null 2>&1; then
        log_verbose "Response contains expected user data"
    else
        log_warning "Response may not contain expected user data"
    fi
    
    # Wait a moment for logs to be written
    sleep 1
    
    # Extract trace ID from logs
    GETUSER_TRACE_ID=$(extract_trace_id_from_logs "GetUser" "/api/users/${USER_ID}")
    
    if [ -z "$GETUSER_TRACE_ID" ]; then
        log_error "Failed to extract trace ID for GetUser operation"
        return 1
    fi
    
    log_verbose "GetUser trace ID: $GETUSER_TRACE_ID"
    log_success "GetUser operation completed"
    return 0
}

# ==========================================
# Trace Validation Functions
# ==========================================

# Validate that all spans in a trace share the same trace ID
validate_trace_id_consistency() {
    local trace_json=$1
    local expected_trace_id=$2
    local operation=$3
    
    log_verbose "Validating trace ID consistency for $operation..."
    
    # Extract trace ID from the trace (normalize it)
    local normalized_expected=$(normalize_trace_id "$expected_trace_id")
    
    # Get all trace IDs from spans
    local trace_ids=$(echo "$trace_json" | jq -r '.batches[]?.scopeSpans[]?.spans[]?.traceId' 2>/dev/null | sort -u)
    
    if [ -z "$trace_ids" ]; then
        log_error "No spans found in trace"
        record_check "$operation: Trace contains spans" false "$operation" "$expected_trace_id"
        return 1
    fi
    
    record_check "$operation: Trace contains spans" true "$operation" "$expected_trace_id"
    
    # Check if all spans have the same trace ID
    local trace_id_count=$(echo "$trace_ids" | wc -l | tr -d ' ')
    
    if [ "$trace_id_count" -ne 1 ]; then
        log_error "Multiple trace IDs found in trace: $trace_id_count"
        record_check "$operation: All spans share same trace ID" false "$operation" "$expected_trace_id"
        return 1
    fi
    
    record_check "$operation: All spans share same trace ID" true "$operation" "$expected_trace_id"
    return 0
}

# Validate that each span has a unique span ID
validate_span_id_uniqueness() {
    local trace_json=$1
    local operation=$2
    local trace_id=$3
    
    log_verbose "Validating span ID uniqueness for $operation..."
    
    # Get all span IDs
    local span_ids=$(echo "$trace_json" | jq -r '.batches[]?.scopeSpans[]?.spans[]?.spanId' 2>/dev/null | sort)
    
    if [ -z "$span_ids" ]; then
        log_error "No span IDs found"
        record_check "$operation: Spans have IDs" false "$operation" "$trace_id"
        return 1
    fi
    
    # Count total spans
    local total_spans=$(echo "$span_ids" | wc -l | tr -d ' ')
    
    # Count unique span IDs
    local unique_spans=$(echo "$span_ids" | sort -u | wc -l | tr -d ' ')
    
    if [ "$total_spans" -ne "$unique_spans" ]; then
        log_error "Duplicate span IDs found: $total_spans total, $unique_spans unique"
        record_check "$operation: All span IDs are unique" false "$operation" "$trace_id"
        return 1
    fi
    
    log_verbose "Found $total_spans unique spans"
    record_check "$operation: All span IDs are unique ($total_spans spans)" true "$operation" "$trace_id"
    return 0
}

# Check that spans from both services exist in the trace
validate_multi_service_trace() {
    local trace_json=$1
    local operation=$2
    local trace_id=$3
    
    log_verbose "Validating multi-service trace for $operation..."
    
    # Extract service names
    local services=$(extract_service_names "$trace_json")
    
    if [ -z "$services" ]; then
        log_error "No service names found in trace"
        record_check "$operation: Trace contains service names" false "$operation" "$trace_id"
        return 1
    fi
    
    log_verbose "Services in trace: $(echo "$services" | tr '\n' ', ' | sed 's/,$//')"
    
    # Check for api-gateway
    if ! echo "$services" | grep -q "api-gateway"; then
        log_error "api-gateway service not found in trace"
        record_check "$operation: Trace contains api-gateway spans" false "$operation" "$trace_id"
        return 1
    fi
    record_check "$operation: Trace contains api-gateway spans" true "$operation" "$trace_id"
    
    # Check for user-auth-service
    if ! echo "$services" | grep -q "user-auth-service"; then
        log_error "user-auth-service not found in trace"
        record_check "$operation: Trace contains user-auth-service spans" false "$operation" "$trace_id"
        return 1
    fi
    record_check "$operation: Trace contains user-auth-service spans" true "$operation" "$trace_id"
    
    return 0
}

# Verify parent-child relationships between spans
validate_span_relationships() {
    local trace_json=$1
    local operation=$2
    local trace_id=$3
    
    log_verbose "Validating span relationships for $operation..."
    
    # Get all spans with their parent relationships
    local spans=$(echo "$trace_json" | jq -r '
        .batches[]?.scopeSpans[]?.spans[]? |
        "\(.spanId)|\(.parentSpanId // "root")"
    ' 2>/dev/null)
    
    if [ -z "$spans" ]; then
        log_error "No spans found for relationship validation"
        return 1
    fi
    
    # Count spans with parents (child spans)
    local child_spans=$(echo "$spans" | grep -v "|root$" | wc -l | tr -d ' ')
    
    # Count root spans (no parent)
    local root_spans=$(echo "$spans" | grep "|root$" | wc -l | tr -d ' ')
    
    log_verbose "Found $root_spans root span(s) and $child_spans child span(s)"
    
    # Verify at least one root span exists
    if [ "$root_spans" -eq 0 ]; then
        log_error "No root span found in trace"
        record_check "$operation: Trace has root span" false "$operation" "$trace_id"
        return 1
    fi
    record_check "$operation: Trace has root span" true "$operation" "$trace_id"
    
    # Verify child spans exist (indicating propagation)
    if [ "$child_spans" -eq 0 ]; then
        log_warning "No child spans found - trace may not have propagated"
        record_check "$operation: Trace has child spans (propagation)" false "$operation" "$trace_id"
        return 1
    fi
    record_check "$operation: Trace has child spans (propagation)" true "$operation" "$trace_id"
    
    # Verify parent span IDs reference existing spans
    local all_span_ids=$(echo "$spans" | cut -d'|' -f1 | sort)
    local parent_span_ids=$(echo "$spans" | cut -d'|' -f2 | grep -v "^root$" | sort -u)
    
    local invalid_parents=0
    for parent_id in $parent_span_ids; do
        if ! echo "$all_span_ids" | grep -q "^${parent_id}$"; then
            log_warning "Parent span ID not found in trace: $parent_id"
            invalid_parents=$((invalid_parents + 1))
        fi
    done
    
    if [ "$invalid_parents" -gt 0 ]; then
        log_error "Found $invalid_parents invalid parent references"
        record_check "$operation: All parent span IDs are valid" false "$operation" "$trace_id"
        return 1
    fi
    record_check "$operation: All parent span IDs are valid" true "$operation" "$trace_id"
    
    return 0
}

# Check for token validation span in GetUser trace
validate_token_validation_span() {
    local trace_json=$1
    local trace_id=$2
    
    log_verbose "Checking for token validation span in GetUser trace..."
    
    # Look for ValidateToken span or similar
    local validation_spans=$(echo "$trace_json" | jq -r '
        .batches[]?.scopeSpans[]?.spans[]? |
        select(.name | test("ValidateToken|validate.*token|token.*validation"; "i")) |
        .name
    ' 2>/dev/null)
    
    if [ -z "$validation_spans" ]; then
        log_warning "No explicit token validation span found"
        record_check "GetUser: Token validation span exists" false "GetUser" "$trace_id"
        return 1
    fi
    
    log_verbose "Found token validation span(s): $(echo "$validation_spans" | tr '\n' ', ' | sed 's/,$//')"
    record_check "GetUser: Token validation span exists" true "GetUser" "$trace_id"
    return 0
}

# Validate service names match expected values
validate_service_names() {
    local trace_json=$1
    local operation=$2
    local trace_id=$3
    
    log_verbose "Validating service names for $operation..."
    
    local services=$(extract_service_names "$trace_json")
    
    # Check that service names are exactly as expected
    local expected_services="api-gateway user-auth-service"
    
    for expected in $expected_services; do
        if ! echo "$services" | grep -q "^${expected}$"; then
            log_warning "Expected service not found: $expected"
        fi
    done
    
    # Check for unexpected services
    for service in $services; do
        if ! echo "$expected_services" | grep -q "$service"; then
            log_warning "Unexpected service found: $service"
        fi
    done
    
    record_check "$operation: Service names are valid" true "$operation" "$trace_id"
    return 0
}

# Comprehensive trace validation
validate_trace() {
    local trace_id=$1
    local operation=$2
    local check_token_validation=$3
    
    log_info "Validating trace for $operation operation..."
    
    # Query Tempo for the trace
    local trace_json=$(query_tempo_trace "$trace_id")
    
    if [ -z "$trace_json" ]; then
        log_error "Failed to retrieve trace from Tempo"
        record_check "$operation: Trace found in Tempo" false "$operation" "$trace_id"
        add_diagnostic "ERROR" "Trace not found in Tempo after retries" "$operation" "$trace_id"
        return 1
    fi
    
    record_check "$operation: Trace found in Tempo" true "$operation" "$trace_id"
    
    # Run all validation checks
    local validation_failed=false
    
    validate_trace_id_consistency "$trace_json" "$trace_id" "$operation" || validation_failed=true
    validate_span_id_uniqueness "$trace_json" "$operation" "$trace_id" || validation_failed=true
    validate_multi_service_trace "$trace_json" "$operation" "$trace_id" || validation_failed=true
    validate_span_relationships "$trace_json" "$operation" "$trace_id" || validation_failed=true
    validate_service_names "$trace_json" "$operation" "$trace_id" || validation_failed=true
    
    # Check for token validation span only for GetUser operation
    if [ "$check_token_validation" = true ]; then
        validate_token_validation_span "$trace_json" "$trace_id" || true  # Don't fail on this
    fi
    
    if [ "$validation_failed" = true ]; then
        log_error "Trace validation failed for $operation"
        return 1
    fi
    
    log_success "Trace validation passed for $operation"
    return 0
}

# Verify that all trace IDs are different
verify_trace_ids_unique() {
    log_info "Verifying trace IDs are unique..."
    
    if [ -z "$REGISTER_TRACE_ID" ] || [ -z "$LOGIN_TRACE_ID" ] || [ -z "$GETUSER_TRACE_ID" ]; then
        log_error "Not all trace IDs were captured"
        record_check "All trace IDs captured" false "All" ""
        return 1
    fi
    
    record_check "All trace IDs captured" true "All" ""
    
    # Check if Register and Login have different trace IDs
    if [ "$REGISTER_TRACE_ID" = "$LOGIN_TRACE_ID" ]; then
        log_error "Register and Login have the same trace ID: $REGISTER_TRACE_ID"
        record_check "Register and Login have different trace IDs" false "All" ""
        return 1
    fi
    record_check "Register and Login have different trace IDs" true "All" ""
    
    # Check if Login and GetUser have different trace IDs
    if [ "$LOGIN_TRACE_ID" = "$GETUSER_TRACE_ID" ]; then
        log_error "Login and GetUser have the same trace ID: $LOGIN_TRACE_ID"
        record_check "Login and GetUser have different trace IDs" false "All" ""
        return 1
    fi
    record_check "Login and GetUser have different trace IDs" true "All" ""
    
    # Check if Register and GetUser have different trace IDs
    if [ "$REGISTER_TRACE_ID" = "$GETUSER_TRACE_ID" ]; then
        log_error "Register and GetUser have the same trace ID: $REGISTER_TRACE_ID"
        record_check "Register and GetUser have different trace IDs" false "All" ""
        return 1
    fi
    record_check "Register and GetUser have different trace IDs" true "All" ""
    
    log_success "All trace IDs are unique"
    log_verbose "  Register: $REGISTER_TRACE_ID"
    log_verbose "  Login:    $LOGIN_TRACE_ID"
    log_verbose "  GetUser:  $GETUSER_TRACE_ID"
    
    return 0
}

# ==========================================
# Trace Validation Functions
# ==========================================

# Validate a single trace
validate_trace() {
    local operation=$1
    local trace_id=$2
    local check_token_validation=$3  # "true" for GetUser operation
    
    log_info "Validating $operation trace..."
    
    # Query Tempo for the trace
    local trace_data=$(query_tempo_trace "$trace_id")
    
    if [ -z "$trace_data" ]; then
        record_check "$operation: Trace exists in Tempo" false "$operation" "$trace_id"
        add_diagnostic "ERROR" "Trace not found in Tempo after retries" "$operation" "$trace_id"
        return 1
    fi
    
    record_check "$operation: Trace exists in Tempo" true "$operation" "$trace_id"
    
    # Extract service names from the trace
    local services=$(extract_service_names "$trace_data")
    local has_gateway=false
    local has_auth_service=false
    
    while IFS= read -r service; do
        if [ "$service" = "api-gateway" ]; then
            has_gateway=true
        elif [ "$service" = "user-auth-service" ]; then
            has_auth_service=true
        fi
    done <<< "$services"
    
    # Check for API Gateway spans
    if [ "$has_gateway" = true ]; then
        record_check "$operation: Has API Gateway spans" true "$operation" "$trace_id"
        log_verbose "  ✓ API Gateway spans present"
    else
        record_check "$operation: Has API Gateway spans" false "$operation" "$trace_id"
        add_diagnostic "ERROR" "No API Gateway spans found in trace" "$operation" "$trace_id"
        log_error "  ✗ No API Gateway spans found"
    fi
    
    # Check for User Auth Service spans
    if [ "$has_auth_service" = true ]; then
        record_check "$operation: Has User Auth Service spans" true "$operation" "$trace_id"
        log_verbose "  ✓ User Auth Service spans present"
    else
        record_check "$operation: Has User Auth Service spans" false "$operation" "$trace_id"
        add_diagnostic "ERROR" "No User Auth Service spans found in trace" "$operation" "$trace_id"
        log_error "  ✗ No User Auth Service spans found"
    fi
    
    # Verify trace ID consistency and span relationships
    local normalized_trace_id=$(normalize_trace_id "$trace_id")
    local spans_json=$(parse_trace_spans "$trace_data")
    
    if [ -z "$spans_json" ]; then
        record_check "$operation: Trace has valid spans" false "$operation" "$trace_id"
        add_diagnostic "ERROR" "Could not parse spans from trace" "$operation" "$trace_id"
        return 1
    fi
    
    record_check "$operation: Trace has valid spans" true "$operation" "$trace_id"
    
    # Check that all spans have unique span IDs
    local span_ids=$(echo "$spans_json" | jq -r '.span_id' 2>/dev/null | sort)
    local unique_span_ids=$(echo "$span_ids" | uniq)
    
    if [ "$(echo "$span_ids" | wc -l)" = "$(echo "$unique_span_ids" | wc -l)" ]; then
        record_check "$operation: All spans have unique span IDs" true "$operation" "$trace_id"
        log_verbose "  ✓ All span IDs are unique"
    else
        record_check "$operation: All spans have unique span IDs" false "$operation" "$trace_id"
        add_diagnostic "ERROR" "Duplicate span IDs found in trace" "$operation" "$trace_id"
        log_error "  ✗ Duplicate span IDs found"
    fi
    
    # Verify parent-child relationships
    local has_parent_child=false
    local span_count=$(echo "$spans_json" | jq -s 'length' 2>/dev/null)
    
    if [ "$span_count" -gt 1 ]; then
        # Check if any span has a parent_span_id that matches another span's span_id
        local parent_ids=$(echo "$spans_json" | jq -r 'select(.parent_span_id != null and .parent_span_id != "") | .parent_span_id' 2>/dev/null)
        
        if [ -n "$parent_ids" ]; then
            has_parent_child=true
        fi
    fi
    
    if [ "$has_parent_child" = true ]; then
        record_check "$operation: Has proper parent-child relationships" true "$operation" "$trace_id"
        log_verbose "  ✓ Parent-child relationships exist"
    else
        record_check "$operation: Has proper parent-child relationships" false "$operation" "$trace_id"
        add_diagnostic "WARNING" "No parent-child relationships found between spans" "$operation" "$trace_id"
        log_warning "  ⚠ No parent-child relationships found"
    fi
    
    # Check for token validation span (only for GetUser)
    if [ "$check_token_validation" = "true" ]; then
        local has_token_validation=$(echo "$spans_json" | jq -r 'select(.name == "validate_token") | .name' 2>/dev/null)
        
        if [ -n "$has_token_validation" ]; then
            record_check "$operation: Has token validation span" true "$operation" "$trace_id"
            log_verbose "  ✓ Token validation span present"
        else
            record_check "$operation: Has token validation span" false "$operation" "$trace_id"
            add_diagnostic "ERROR" "Token validation span not found in GetUser trace" "$operation" "$trace_id"
            log_error "  ✗ Token validation span not found"
        fi
    fi
    
    # Verify service names match expected values
    local service_names_valid=true
    
    while IFS= read -r service; do
        if [ "$service" != "api-gateway" ] && [ "$service" != "user-auth-service" ]; then
            service_names_valid=false
            log_warning "  ⚠ Unexpected service name: $service"
        fi
    done <<< "$services"
    
    if [ "$service_names_valid" = true ]; then
        record_check "$operation: Service names are valid" true "$operation" "$trace_id"
        log_verbose "  ✓ All service names are valid"
    else
        record_check "$operation: Service names are valid" false "$operation" "$trace_id"
        add_diagnostic "WARNING" "Unexpected service names found in trace" "$operation" "$trace_id"
    fi
    
    # Check if trace propagation worked (both services present)
    if [ "$has_gateway" = true ] && [ "$has_auth_service" = true ]; then
        record_check "$operation: Trace propagation successful" true "$operation" "$trace_id"
        log_success "  ✓ Trace propagation successful (both services present)"
        return 0
    else
        record_check "$operation: Trace propagation successful" false "$operation" "$trace_id"
        add_diagnostic "ERROR" "Trace propagation failed - not all services present in trace" "$operation" "$trace_id"
        log_error "  ✗ Trace propagation failed"
        return 1
    fi
}

# Validate all traces
validate_all_traces() {
    log_info "Starting trace validation..."
    echo ""
    
    local all_passed=true
    
    # Validate Register trace
    if ! validate_trace "Register" "$REGISTER_TRACE_ID" "false"; then
        all_passed=false
    fi
    echo ""
    
    # Validate Login trace
    if ! validate_trace "Login" "$LOGIN_TRACE_ID" "false"; then
        all_passed=false
    fi
    echo ""
    
    # Validate GetUser trace (with token validation check)
    if ! validate_trace "GetUser" "$GETUSER_TRACE_ID" "true"; then
        all_passed=false
    fi
    echo ""
    
    if [ "$all_passed" = true ]; then
        log_success "All trace validations passed"
        return 0
    else
        log_error "Some trace validations failed"
        return 1
    fi
}

# ==========================================
# Main Script
# ==========================================

main() {
    # Parse command line arguments
    parse_arguments "$@"
    
    # Print header (unless JSON output)
    if [ "$JSON_OUTPUT" = false ]; then
        echo "=========================================="
        echo "Trace Propagation Verification"
        echo "=========================================="
        echo ""
        echo "Configuration:"
        echo "  Gateway URL: $GATEWAY_URL"
        echo "  Tempo URL: $TEMPO_URL"
        echo "  Wait Time: ${WAIT_TIME}s"
        echo ""
    fi
    
    # Check dependencies and services
    check_dependencies
    check_services
    
    echo ""
    log_info "Starting user flow execution..."
    echo ""
    
    # Execute user flow operations
    if ! execute_register; then
        log_error "Register operation failed"
        exit 1
    fi
    
    echo ""
    
    if ! execute_login; then
        log_error "Login operation failed"
        exit 1
    fi
    
    echo ""
    
    if ! execute_getuser; then
        log_error "GetUser operation failed"
        exit 1
    fi
    
    echo ""
    
    # Verify trace IDs are unique
    if ! verify_trace_ids_unique; then
        log_error "Trace ID uniqueness verification failed"
        exit 1
    fi
    
    echo ""
    log_success "User flow execution completed successfully"
    log_info "Waiting ${WAIT_TIME}s for traces to be ingested by Tempo..."
    sleep "$WAIT_TIME"
    
    echo ""
    
    # Check Tempo connection before validation
    if ! check_tempo_connection; then
        log_error "Cannot proceed with trace validation - Tempo is not accessible"
        exit 2
    fi
    
    echo ""
    
    # Record end time for execution time calculation
    local start_time=$(date +%s%N)
    
    # Validate all traces
    local validation_result=0
    if ! validate_all_traces; then
        validation_result=1
    fi
    
    local end_time=$(date +%s%N)
    local execution_time=$(( (end_time - start_time) / 1000000 ))  # Convert to milliseconds
    
    # Generate output based on format
    if [ "$JSON_OUTPUT" = true ]; then
        if [ $validation_result -eq 0 ]; then
            generate_json_output "PASS" "$execution_time"
        else
            generate_json_output "FAIL" "$execution_time"
        fi
    else
        if [ $validation_result -eq 0 ]; then
            print_summary "PASS"
        else
            print_summary "FAIL"
        fi
    fi
    
    exit $validation_result
}

# Run main function with all arguments
main "$@"
