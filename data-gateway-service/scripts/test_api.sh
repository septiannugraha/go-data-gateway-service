#!/bin/bash

# =============================================================================
# Go Data Gateway API Testing Script
# =============================================================================
# This script provides comprehensive testing for all API endpoints
# Usage: ./test_api.sh [command] [options]
# =============================================================================

set -euo pipefail

# Configuration
BASE_URL="${BASE_URL:-http://localhost:8080}"
API_KEY="${API_KEY:-test-api-key-123}"
VERBOSE="${VERBOSE:-false}"
COLOR_OUTPUT="${COLOR_OUTPUT:-true}"

# Colors for output
if [ "$COLOR_OUTPUT" = "true" ]; then
    RED='\033[0;31m'
    GREEN='\033[0;32m'
    YELLOW='\033[1;33m'
    BLUE='\033[0;34m'
    NC='\033[0m' # No Color
else
    RED=''
    GREEN=''
    YELLOW=''
    BLUE=''
    NC=''
fi

# Logging functions
log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Helper function for curl with common headers
api_call() {
    local method="$1"
    local endpoint="$2"
    local data="${3:-}"

    local curl_args=(-X "$method" -H "X-API-Key: $API_KEY")

    if [ -n "$data" ]; then
        curl_args+=(-H "Content-Type: application/json" -d "$data")
    fi

    if [ "$VERBOSE" = "true" ]; then
        curl_args+=(-v)
    else
        curl_args+=(-s)
    fi

    curl "${curl_args[@]}" "$BASE_URL$endpoint"
}

# Pretty print JSON
pretty_json() {
    if command -v jq &> /dev/null; then
        jq '.'
    else
        cat
    fi
}

# =============================================================================
# Infrastructure Tests
# =============================================================================

test_health() {
    log_info "Testing health endpoint..."
    response=$(curl -s "$BASE_URL/health")

    if echo "$response" | grep -q "healthy"; then
        log_success "Health check passed"
        [ "$VERBOSE" = "true" ] && echo "$response" | pretty_json
    else
        log_error "Health check failed"
        echo "$response"
        return 1
    fi
}

test_ready() {
    log_info "Testing readiness endpoint..."
    response=$(curl -s "$BASE_URL/ready")

    if echo "$response" | grep -q "ready"; then
        log_success "Readiness check passed"
        [ "$VERBOSE" = "true" ] && echo "$response" | pretty_json
    else
        log_error "Readiness check failed"
        echo "$response"
        return 1
    fi
}

test_metrics() {
    log_info "Testing metrics endpoint..."
    response=$(curl -s "$BASE_URL/metrics")

    if echo "$response" | grep -q "go_gc_duration_seconds"; then
        log_success "Metrics endpoint working"
        [ "$VERBOSE" = "true" ] && echo "$response" | head -20
    else
        log_error "Metrics endpoint failed"
        return 1
    fi
}

test_cache_stats() {
    log_info "Testing cache stats endpoint..."
    response=$(api_call "GET" "/cache/stats")

    if echo "$response" | grep -q "hit_rate"; then
        log_success "Cache stats retrieved"
        [ "$VERBOSE" = "true" ] && echo "$response" | pretty_json
    else
        log_error "Cache stats failed"
        echo "$response"
        return 1
    fi
}

# =============================================================================
# Authentication Tests
# =============================================================================

test_auth_required() {
    log_info "Testing authentication requirement..."

    # Test without API key
    response=$(curl -s -o /dev/null -w "%{http_code}" "$BASE_URL/api/v1/tender")

    if [ "$response" = "401" ]; then
        log_success "Authentication correctly required"
    else
        log_error "Expected 401, got $response"
        return 1
    fi
}

test_auth_api_key() {
    log_info "Testing API key authentication..."

    response=$(curl -s -o /dev/null -w "%{http_code}" \
        -H "X-API-Key: $API_KEY" \
        "$BASE_URL/api/v1/tender")

    if [ "$response" = "200" ]; then
        log_success "API key authentication working"
    else
        log_error "API key authentication failed with status $response"
        return 1
    fi
}

test_auth_bearer() {
    log_info "Testing Bearer token authentication..."

    response=$(curl -s -o /dev/null -w "%{http_code}" \
        -H "Authorization: Bearer $API_KEY" \
        "$BASE_URL/api/v1/tender")

    if [ "$response" = "200" ]; then
        log_success "Bearer token authentication working"
    else
        log_error "Bearer token authentication failed with status $response"
        return 1
    fi
}

# =============================================================================
# Query Endpoints Tests
# =============================================================================

test_query_endpoint() {
    log_info "Testing query endpoint..."

    data='{
        "sql": "SELECT * FROM table LIMIT 10",
        "source": "DATAWAREHOUSE"
    }'

    response=$(api_call "POST" "/api/v1/query" "$data")

    if echo "$response" | grep -q "success"; then
        log_success "Query executed successfully"
        [ "$VERBOSE" = "true" ] && echo "$response" | pretty_json
    else
        log_error "Query execution failed"
        echo "$response"
        return 1
    fi
}

test_batch_endpoint() {
    log_info "Testing batch endpoint..."

    data='{
        "queries": [
            {
                "id": "q1",
                "query": "SELECT 1",
                "data_source": "DATAWAREHOUSE"
            },
            {
                "id": "q2",
                "query": "SELECT 2",
                "data_source": "BIGQUERY"
            }
        ],
        "options": {
            "max_concurrency": 2
        }
    }'

    response=$(api_call "POST" "/api/v1/batch" "$data")

    if echo "$response" | grep -q "results"; then
        log_success "Batch queries executed"
        [ "$VERBOSE" = "true" ] && echo "$response" | pretty_json
    else
        log_error "Batch execution failed"
        echo "$response"
        return 1
    fi
}

test_stream_endpoint() {
    log_info "Testing stream endpoint..."

    data='{
        "query": "SELECT * FROM large_table",
        "data_source": "DATAWAREHOUSE",
        "chunk_size": 1000,
        "format": "ndjson"
    }'

    # Test with timeout since it's streaming
    response=$(timeout 5 curl -s -X POST \
        -H "X-API-Key: $API_KEY" \
        -H "Content-Type: application/json" \
        -d "$data" \
        "$BASE_URL/api/v1/stream" || true)

    if [ -n "$response" ]; then
        log_success "Stream endpoint working"
        [ "$VERBOSE" = "true" ] && echo "$response" | head -5
    else
        log_warning "Stream endpoint returned empty (might be normal)"
    fi
}

# =============================================================================
# Tender Endpoints Tests
# =============================================================================

test_tender_list() {
    log_info "Testing tender list endpoint..."

    response=$(api_call "GET" "/api/v1/tender?limit=10")

    if echo "$response" | grep -q "success"; then
        log_success "Tender list retrieved"
        [ "$VERBOSE" = "true" ] && echo "$response" | pretty_json
    else
        log_error "Tender list failed"
        echo "$response"
        return 1
    fi
}

test_tender_get_by_id() {
    log_info "Testing tender get by ID..."

    response=$(api_call "GET" "/api/v1/tender/TENDER-001")

    if echo "$response" | grep -q "tender_id\|data"; then
        log_success "Tender retrieved by ID"
        [ "$VERBOSE" = "true" ] && echo "$response" | pretty_json
    else
        log_error "Tender get by ID failed"
        echo "$response"
        return 1
    fi
}

test_tender_search() {
    log_info "Testing tender search..."

    data='{
        "keyword": "construction",
        "status": "active",
        "limit": 10
    }'

    response=$(api_call "POST" "/api/v1/tender/search" "$data")

    if echo "$response" | grep -q "success\|data"; then
        log_success "Tender search completed"
        [ "$VERBOSE" = "true" ] && echo "$response" | pretty_json
    else
        log_error "Tender search failed"
        echo "$response"
        return 1
    fi
}

# =============================================================================
# RUP Endpoints Tests
# =============================================================================

test_rup_list() {
    log_info "Testing RUP list endpoint..."

    response=$(api_call "GET" "/api/v1/rup?limit=10")

    # RUP requires BigQuery, might not be available
    if echo "$response" | grep -q "success\|unavailable"; then
        log_warning "RUP endpoint returned (might need BigQuery setup)"
        [ "$VERBOSE" = "true" ] && echo "$response" | pretty_json
    else
        log_error "RUP list failed unexpectedly"
        echo "$response"
        return 1
    fi
}

# =============================================================================
# BigQuery Tests
# =============================================================================

test_bigquery_estimate() {
    log_info "Testing BigQuery cost estimate..."

    data='{
        "query": "SELECT * FROM `project.dataset.table` WHERE date >= '\''2025-01-01'\''"
    }'

    response=$(api_call "POST" "/api/v1/bigquery/estimate-cost" "$data")

    if echo "$response" | grep -q "bytes_processed\|estimated_cost"; then
        log_success "Cost estimate retrieved"
        [ "$VERBOSE" = "true" ] && echo "$response" | pretty_json
    else
        log_warning "Cost estimate might not be available"
        [ "$VERBOSE" = "true" ] && echo "$response"
    fi
}

# =============================================================================
# Performance Tests
# =============================================================================

test_performance_simple() {
    log_info "Running simple performance test..."

    if ! command -v time &> /dev/null; then
        log_warning "time command not available, skipping"
        return 0
    fi

    log_info "Testing query endpoint performance (10 requests)..."

    total_time=0
    for i in {1..10}; do
        start=$(date +%s%N)
        api_call "GET" "/api/v1/tender?limit=1" > /dev/null 2>&1
        end=$(date +%s%N)
        elapsed=$((($end - $start) / 1000000))
        total_time=$(($total_time + $elapsed))

        [ "$VERBOSE" = "true" ] && echo "Request $i: ${elapsed}ms"
    done

    avg_time=$(($total_time / 10))
    log_info "Average response time: ${avg_time}ms"

    if [ $avg_time -lt 200 ]; then
        log_success "Performance test passed (avg < 200ms)"
    else
        log_warning "Performance could be improved (avg: ${avg_time}ms)"
    fi
}

test_concurrent_requests() {
    log_info "Testing concurrent requests..."

    if ! command -v xargs &> /dev/null; then
        log_warning "xargs not available, skipping concurrent test"
        return 0
    fi

    # Run 5 concurrent requests
    seq 1 5 | xargs -P5 -I{} curl -s \
        -H "X-API-Key: $API_KEY" \
        "$BASE_URL/api/v1/tender?limit=1" > /dev/null

    if [ $? -eq 0 ]; then
        log_success "Concurrent requests handled successfully"
    else
        log_error "Concurrent requests failed"
        return 1
    fi
}

# =============================================================================
# Load Testing (requires hey)
# =============================================================================

test_load_with_hey() {
    log_info "Running load test with hey..."

    if ! command -v hey &> /dev/null; then
        log_warning "hey not installed. Install with: go install github.com/rakyll/hey@latest"
        return 0
    fi

    log_info "Running 100 requests with 10 concurrent workers..."

    hey -n 100 -c 10 -q 10 \
        -H "X-API-Key: $API_KEY" \
        "$BASE_URL/api/v1/tender?limit=1" > hey_report.txt 2>&1

    if [ $? -eq 0 ]; then
        log_success "Load test completed"

        # Extract summary from hey output
        if [ "$VERBOSE" = "true" ]; then
            grep -E "Summary|Requests/sec|Latencies" hey_report.txt
        fi

        rm hey_report.txt
    else
        log_error "Load test failed"
        cat hey_report.txt
        rm hey_report.txt
        return 1
    fi
}

# =============================================================================
# Test Suites
# =============================================================================

run_infrastructure_tests() {
    log_info "Running infrastructure tests..."
    test_health
    test_ready
    test_metrics
    test_cache_stats
}

run_auth_tests() {
    log_info "Running authentication tests..."
    test_auth_required
    test_auth_api_key
    test_auth_bearer
}

run_query_tests() {
    log_info "Running query endpoint tests..."
    test_query_endpoint
    test_batch_endpoint
    test_stream_endpoint
}

run_tender_tests() {
    log_info "Running tender endpoint tests..."
    test_tender_list
    test_tender_get_by_id
    test_tender_search
}

run_all_tests() {
    log_info "Running all API tests..."

    local failed=0

    # Infrastructure
    run_infrastructure_tests || ((failed++))

    # Authentication
    run_auth_tests || ((failed++))

    # Endpoints
    run_query_tests || ((failed++))
    run_tender_tests || ((failed++))

    # Optional tests
    test_rup_list || log_warning "RUP tests skipped"
    test_bigquery_estimate || log_warning "BigQuery tests skipped"

    if [ $failed -eq 0 ]; then
        log_success "All tests passed!"
        return 0
    else
        log_error "$failed test suites failed"
        return 1
    fi
}

run_performance_tests() {
    log_info "Running performance tests..."
    test_performance_simple
    test_concurrent_requests
    test_load_with_hey
}

# =============================================================================
# Utility Functions
# =============================================================================

check_service() {
    log_info "Checking if service is running at $BASE_URL..."

    if curl -s -o /dev/null -w "%{http_code}" "$BASE_URL/health" | grep -q "200"; then
        log_success "Service is running"
        return 0
    else
        log_error "Service is not responding at $BASE_URL"
        log_info "Please ensure the service is running:"
        log_info "  docker-compose up -d"
        log_info "  OR"
        log_info "  go run cmd/server/main.go"
        return 1
    fi
}

show_usage() {
    cat << EOF
Usage: $0 [command] [options]

Commands:
    all              Run all tests (default)
    infra            Run infrastructure tests (health, ready, metrics)
    auth             Run authentication tests
    query            Run query endpoint tests
    tender           Run tender endpoint tests
    performance      Run performance tests
    load             Run load tests (requires hey)
    check            Check if service is running
    help             Show this help message

Options:
    BASE_URL=<url>   Set the base URL (default: http://localhost:8080)
    API_KEY=<key>    Set the API key (default: test-api-key-123)
    VERBOSE=true     Enable verbose output
    COLOR_OUTPUT=false Disable colored output

Examples:
    $0                          # Run all tests
    $0 auth                     # Run only auth tests
    $0 performance              # Run performance tests
    BASE_URL=https://api.example.com $0 all  # Test against production
    VERBOSE=true $0 query      # Run query tests with verbose output

EOF
}

# =============================================================================
# Main
# =============================================================================

main() {
    local command="${1:-all}"

    case "$command" in
        all)
            check_service && run_all_tests
            ;;
        infra|infrastructure)
            check_service && run_infrastructure_tests
            ;;
        auth|authentication)
            check_service && run_auth_tests
            ;;
        query)
            check_service && run_query_tests
            ;;
        tender)
            check_service && run_tender_tests
            ;;
        performance|perf)
            check_service && run_performance_tests
            ;;
        load)
            check_service && test_load_with_hey
            ;;
        check)
            check_service
            ;;
        help|-h|--help)
            show_usage
            ;;
        *)
            log_error "Unknown command: $command"
            show_usage
            exit 1
            ;;
    esac
}

# Run main function
main "$@"