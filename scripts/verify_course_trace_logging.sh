#!/bin/bash

# Course Service Trace Logging Verification Script
# Verifies that trace IDs from OpenTelemetry appear in Pino logs and match Tempo traces

set -e
set -o pipefail

# ==========================================
# Default Configuration
# ==========================================
GATEWAY_URL="http://localhost:8080"
TEMPO_URL="http://localhost:3200"
LOKI_URL="http://localhost:3100"
WAIT_TIME=10
VERBOSE=false

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
COURSE_ID=""
ACCESS_TOKEN=""
TRACE_ID=""
TOTAL_CHECKS=0
PASSED_CHECKS=0
FAILED_CHECKS=0

# ==========================================
# Helper Functions
# ==========================================

# Print usage information
usage() {
    cat << EOF
Usage: $0 [OPTIONS]

Verify that trace IDs from OpenTelemetry appear in course-service Pino logs
and match the corresponding traces in Tempo.

OPTIONS:
    --gateway-url URL    API Gateway URL (default: http://localhost:8080)
    --tempo-url URL      Tempo API URL (default: http://localhost:3200)
    --loki-url URL       Loki API URL (default: http://localhost:3100)
    --wait-time SECONDS  Wait time for log/trace ingestion (default: 10)
    --verbose            Enable verbose output
    -h, --help           Show this help message

EXAMPLES:
    # Run with defaults
    $0

    # Run with custom URLs
    $0 --gateway-url http://gateway:8080 --loki-url http://loki:3100

    # Run with verbose output
    $0 --verbose

REQUIREMENTS:
    - API Gateway must be running and accessible
    - Course Service must be running with trace logging enabled
    - Tempo must be running and accessible
    - Loki must be running and accessible
    - curl and jq must be installed
    - User must be registered and authenticated

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
    echo -e "${BLUE}[INFO]${NC} $1"
}

# Log success messages
log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

# Log warning messages
log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

# Log error messages
log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Record check result
record_check() {
    local check_name=$1
    local passed=$2
    
    TOTAL_CHECKS=$((TOTAL_CHECKS + 1))
    
    if [ "$passed" = true ]; then
        PASSED_CHECKS=$((PASSED_CHECKS + 1))
        log_success "✓ $check_name"
    else
        FAILED_CHECKS=$((FAILED_CHECKS + 1))
        log_error "✗ $check_name"
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
    
    if ! command -v docker &> /dev/null; then
        missing_deps+=("docker")
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
        exit 2
    fi
    log_verbose "API Gateway is accessible"
    
    # Check Tempo
    if ! curl -s -f "${TEMPO_URL}/ready" > /dev/null 2>&1; then
        log_warning "Tempo is not accessible at ${TEMPO_URL}"
    else
        log_verbose "Tempo is accessible"
    fi
    
    # Check Loki
    if ! curl -s -f "${LOKI_URL}/ready" > /dev/null 2>&1; then
        log_warning "Loki is not accessible at ${LOKI_URL}"
    else
        log_verbose "Loki is accessible"
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
            --loki-url)
                LOKI_URL="$2"
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
# Authentication
# ==========================================

# Get authentication token
get_auth_token() {
    log_info "Authenticating..."
    
    # Try to register a new user
    local timestamp=$(date +%s)
    local test_email="trace-log-test-${timestamp}@example.com"
    local test_password="TestPass123!"
    
    log_verbose "Registering user: $test_email"
    
    local response=$(curl -s -w "\nHTTP_STATUS:%{http_code}" -X POST "${GATEWAY_URL}/api/auth/register" \
        -H "Content-Type: application/json" \
        -d "{
            \"email\": \"$test_email\",
            \"password\": \"$test_password\",
            \"first_name\": \"Trace\",
            \"last_name\": \"Test\",
            \"phone\": \"+1234567890\"
        }")
    
    local http_status=$(echo "$response" | grep "HTTP_STATUS" | cut -d: -f2)
    local body=$(echo "$response" | sed '/HTTP_STATUS/d')
    
    if [ "$http_status" != "200" ]; then
        log_error "Authentication failed with HTTP $http_status"
        log_verbose "Response: $body"
        exit 2
    fi
    
    ACCESS_TOKEN=$(echo "$body" | jq -r '.access_token' 2>/dev/null)
    
    if [ -z "$ACCESS_TOKEN" ] || [ "$ACCESS_TOKEN" = "null" ]; then
        log_error "Failed to extract access token"
        exit 2
    fi
    
    log_success "Authentication successful"
    log_verbose "Token: ${ACCESS_TOKEN:0:30}..."
}

# ==========================================
# Course Operations
# ==========================================

# Create a course
create_course() {
    log_info "Creating a test course..."
    
    local timestamp=$(date +%s)
    local course_code="TRACE-TEST-${timestamp}"
    
    log_verbose "Course code: $course_code"
    
    local response=$(curl -s -w "\nHTTP_STATUS:%{http_code}" -X POST "${GATEWAY_URL}/api/courses" \
        -H "Authorization: Bearer $ACCESS_TOKEN" \
        -H "Content-Type: application/json" \
        -d "{
            \"title\": \"Trace Logging Test Course\",
            \"code\": \"$course_code\",
            \"description\": \"Test course for trace logging verification\",
            \"credits\": 3
        }")
    
    local http_status=$(echo "$response" | grep "HTTP_STATUS" | cut -d: -f2)
    local body=$(echo "$response" | sed '/HTTP_STATUS/d')
    
    if [ "$http_status" != "200" ] && [ "$http_status" != "201" ]; then
        log_error "Course creation failed with HTTP $http_status"
        log_verbose "Response: $body"
        return 1
    fi
    
    COURSE_ID=$(echo "$body" | jq -r '.id // ._id // .course_id' 2>/dev/null)
    
    if [ -z "$COURSE_ID" ] || [ "$COURSE_ID" = "null" ]; then
        log_error "Failed to extract course ID from response"
        log_verbose "Response: $body"
        return 1
    fi
    
    log_success "Course created successfully"
    log_verbose "Course ID: $COURSE_ID"
    record_check "Course creation request successful" true
    return 0
}

# ==========================================
# Trace and Log Verification
# ==========================================

# Normalize trace ID (remove dashes)
normalize_trace_id() {
    local trace_id=$1
    echo "$trace_id" | tr -d '-'
}

# Query Tempo for traces from course-service
query_tempo_for_course_traces() {
    log_info "Querying Tempo for course-service traces..."
    
    local max_retries=5
    local retry_count=0
    local backoff=2
    
    while [ $retry_count -lt $max_retries ]; do
        # Search for recent traces from course-service
        local response=$(curl -s "${TEMPO_URL}/api/search?tags=service.name%3Dcourse-service&limit=10")
        
        if [ -n "$response" ]; then
            local trace_count=$(echo "$response" | jq '.traces | length' 2>/dev/null)
            
            if [ "$trace_count" -gt 0 ]; then
                # Get the most recent trace ID
                TRACE_ID=$(echo "$response" | jq -r '.traces[0].traceID' 2>/dev/null)
                
                if [ -n "$TRACE_ID" ] && [ "$TRACE_ID" != "null" ]; then
                    log_success "Found trace in Tempo"
                    log_verbose "Trace ID: $TRACE_ID"
                    record_check "Trace found in Tempo" true
                    return 0
                fi
            fi
        fi
        
        retry_count=$((retry_count + 1))
        
        if [ $retry_count -lt $max_retries ]; then
            log_verbose "No traces found, retrying in ${backoff}s (attempt $retry_count/$max_retries)..."
            sleep $backoff
            backoff=$((backoff * 2))
        fi
    done
    
    log_error "No traces found in Tempo after $max_retries attempts"
    record_check "Trace found in Tempo" false
    return 1
}

# Query Loki for logs with the trace ID
query_loki_for_trace_logs() {
    local trace_id=$1
    
    log_info "Querying Loki for logs with trace_id=$trace_id..."
    
    # Normalize trace ID
    local normalized_id=$(normalize_trace_id "$trace_id")
    
    # Build LogQL query
    local logql_query="{service=\"course-service\"} | json | trace_id=\"$normalized_id\""
    
    log_verbose "LogQL query: $logql_query"
    
    # Query Loki
    local response=$(curl -s -G "${LOKI_URL}/loki/api/v1/query" \
        --data-urlencode "query=$logql_query" \
        --data-urlencode "limit=100")
    
    if [ -z "$response" ]; then
        log_error "Empty response from Loki"
        record_check "Loki query successful" false
        return 1
    fi
    
    record_check "Loki query successful" true
    
    # Parse response
    local result_count=$(echo "$response" | jq '.data.result | length' 2>/dev/null)
    
    if [ -z "$result_count" ] || [ "$result_count" = "null" ]; then
        log_error "Failed to parse Loki response"
        log_verbose "Response: $response"
        record_check "Loki response parseable" false
        return 1
    fi
    
    record_check "Loki response parseable" true
    
    if [ "$result_count" -eq 0 ]; then
        log_error "No log entries found with trace_id=$normalized_id"
        record_check "Logs found with trace_id" false
        return 1
    fi
    
    log_success "Found $result_count log stream(s) with trace_id=$normalized_id"
    record_check "Logs found with trace_id" true
    
    # Count total log entries across all streams
    local total_logs=0
    for i in $(seq 0 $((result_count - 1))); do
        local stream_logs=$(echo "$response" | jq ".data.result[$i].values | length" 2>/dev/null)
        total_logs=$((total_logs + stream_logs))
    done
    
    log_verbose "Total log entries: $total_logs"
    
    if [ "$total_logs" -eq 0 ]; then
        log_error "No log entries found in streams"
        record_check "Log entries exist" false
        return 1
    fi
    
    record_check "Log entries exist ($total_logs entries)" true
    
    # Verify trace_id format in logs
    local first_log=$(echo "$response" | jq -r '.data.result[0].values[0][1]' 2>/dev/null)
    
    if [ -z "$first_log" ] || [ "$first_log" = "null" ]; then
        log_warning "Could not extract log entry for validation"
        return 0
    fi
    
    local log_trace_id=$(echo "$first_log" | jq -r '.trace_id' 2>/dev/null)
    
    if [ -z "$log_trace_id" ] || [ "$log_trace_id" = "null" ]; then
        log_error "trace_id field not found in log entry"
        record_check "trace_id field present in logs" false
        return 1
    fi
    
    record_check "trace_id field present in logs" true
    
    # Verify trace_id matches
    if [ "$log_trace_id" = "$normalized_id" ]; then
        log_success "trace_id in logs matches Tempo trace ID"
        record_check "trace_id matches between logs and Tempo" true
    else
        log_error "trace_id mismatch - Logs: $log_trace_id, Tempo: $normalized_id"
        record_check "trace_id matches between logs and Tempo" false
        return 1
    fi
    
    # Check for span_id field
    local log_span_id=$(echo "$first_log" | jq -r '.span_id' 2>/dev/null)
    
    if [ -n "$log_span_id" ] && [ "$log_span_id" != "null" ]; then
        log_verbose "span_id field present: $log_span_id"
        record_check "span_id field present in logs" true
    else
        log_warning "span_id field not found in log entry"
        record_check "span_id field present in logs" false
    fi
    
    # Check for trace_flags field
    local log_trace_flags=$(echo "$first_log" | jq -r '.trace_flags' 2>/dev/null)
    
    if [ -n "$log_trace_flags" ] && [ "$log_trace_flags" != "null" ]; then
        log_verbose "trace_flags field present: $log_trace_flags"
        record_check "trace_flags field present in logs" true
    else
        log_warning "trace_flags field not found in log entry"
        record_check "trace_flags field present in logs" false
    fi
    
    return 0
}

# Verify all log entries for the trace share the same trace_id
verify_trace_id_consistency() {
    local trace_id=$1
    
    log_info "Verifying trace_id consistency across log entries..."
    
    local normalized_id=$(normalize_trace_id "$trace_id")
    local logql_query="{service=\"course-service\"} | json | trace_id=\"$normalized_id\""
    
    local response=$(curl -s -G "${LOKI_URL}/loki/api/v1/query" \
        --data-urlencode "query=$logql_query" \
        --data-urlencode "limit=1000")
    
    # Extract all trace_ids from log entries
    local trace_ids=$(echo "$response" | jq -r '.data.result[].values[][1]' 2>/dev/null | jq -r '.trace_id' 2>/dev/null | sort -u)
    
    if [ -z "$trace_ids" ]; then
        log_warning "Could not extract trace_ids for consistency check"
        return 0
    fi
    
    local unique_count=$(echo "$trace_ids" | wc -l | tr -d ' ')
    
    if [ "$unique_count" -eq 1 ]; then
        log_success "All log entries share the same trace_id"
        record_check "All log entries have consistent trace_id" true
    else
        log_error "Found $unique_count different trace_ids in logs"
        record_check "All log entries have consistent trace_id" false
        return 1
    fi
    
    return 0
}

# ==========================================
# Summary and Output
# ==========================================

print_summary() {
    echo ""
    echo "=========================================="
    echo "Verification Summary"
    echo "=========================================="
    echo ""
    echo "Total Checks: $TOTAL_CHECKS"
    echo -e "${GREEN}Passed: $PASSED_CHECKS${NC}"
    echo -e "${RED}Failed: $FAILED_CHECKS${NC}"
    echo ""
    
    if [ "$FAILED_CHECKS" -eq 0 ]; then
        echo -e "${GREEN}✓ All checks passed!${NC}"
        echo ""
        echo "Trace logging is working correctly:"
        echo "  - Trace IDs from OpenTelemetry appear in Pino logs"
        echo "  - Trace IDs in logs match traces in Tempo"
        echo "  - Log entries include trace_id, span_id, and trace_flags"
        echo ""
        return 0
    else
        echo -e "${RED}✗ Some checks failed${NC}"
        echo ""
        echo "Please review the errors above and check:"
        echo "  - OpenTelemetry is properly initialized in course-service"
        echo "  - Pino logger configuration includes trace context mixin"
        echo "  - Logs are being sent to Loki"
        echo "  - Traces are being sent to Tempo"
        echo ""
        return 1
    fi
}

# ==========================================
# Main Script
# ==========================================

main() {
    # Parse command line arguments
    parse_arguments "$@"
    
    # Print header
    echo "=========================================="
    echo "Course Service Trace Logging Verification"
    echo "=========================================="
    echo ""
    echo "Configuration:"
    echo "  Gateway URL: $GATEWAY_URL"
    echo "  Tempo URL: $TEMPO_URL"
    echo "  Loki URL: $LOKI_URL"
    echo "  Wait Time: ${WAIT_TIME}s"
    echo ""
    
    # Check dependencies and services
    check_dependencies
    check_services
    
    echo ""
    
    # Get authentication token
    get_auth_token
    
    echo ""
    
    # Create a course
    if ! create_course; then
        log_error "Course creation failed"
        exit 1
    fi
    
    echo ""
    log_info "Waiting ${WAIT_TIME}s for traces and logs to be ingested..."
    sleep "$WAIT_TIME"
    
    echo ""
    
    # Query Tempo for traces
    if ! query_tempo_for_course_traces; then
        log_error "Failed to find traces in Tempo"
        print_summary
        exit 1
    fi
    
    echo ""
    
    # Query Loki for logs with the trace ID
    if ! query_loki_for_trace_logs "$TRACE_ID"; then
        log_error "Failed to verify logs in Loki"
        print_summary
        exit 1
    fi
    
    echo ""
    
    # Verify trace ID consistency
    verify_trace_id_consistency "$TRACE_ID"
    
    # Print summary
    if print_summary; then
        exit 0
    else
        exit 1
    fi
}

# Run main function with all arguments
main "$@"
