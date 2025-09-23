#!/bin/bash

# Performance Benchmarking Script for Go Data Gateway
# This script runs comprehensive performance tests and generates reports

set -e

echo "🚀 Go Data Gateway Performance Benchmark"
echo "========================================"
echo ""

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Check if required tools are installed
check_dependencies() {
    echo "Checking dependencies..."

    if ! command -v go &> /dev/null; then
        echo -e "${RED}Go is not installed${NC}"
        exit 1
    fi

    if ! command -v hey &> /dev/null; then
        echo -e "${YELLOW}Installing 'hey' for load testing...${NC}"
        go install github.com/rakyll/hey@latest
    fi

    if ! command -v ab &> /dev/null; then
        echo -e "${YELLOW}Apache Bench (ab) not found. Some tests will be skipped.${NC}"
    fi

    echo -e "${GREEN}✓ Dependencies checked${NC}"
    echo ""
}

# Run Go benchmarks
run_go_benchmarks() {
    echo "Running Go benchmarks..."
    echo "------------------------"

    cd ../benchmark

    # Run all benchmarks with memory profiling
    go test -bench=. -benchmem -benchtime=10s -cpu=1,2,4 > ../scripts/benchmark_results.txt 2>&1

    # Run specific benchmarks with different configurations
    echo "Running detailed benchmarks..."
    go test -bench=BenchmarkSimpleQuery -benchmem -count=5 >> ../scripts/benchmark_results.txt 2>&1
    go test -bench=BenchmarkBatchQuery -benchmem -count=5 >> ../scripts/benchmark_results.txt 2>&1
    go test -bench=BenchmarkStreaming -benchmem -count=5 >> ../scripts/benchmark_results.txt 2>&1
    go test -bench=BenchmarkConcurrentRequests -benchmem -count=5 >> ../scripts/benchmark_results.txt 2>&1

    echo -e "${GREEN}✓ Go benchmarks completed${NC}"
    echo ""
}

# Run load test with hey
run_load_test() {
    echo "Running load tests with 'hey'..."
    echo "--------------------------------"

    # Start the server in background if not running
    SERVER_URL="http://localhost:8080"
    API_KEY="test-key-123"

    # Check if server is running
    if ! curl -s "${SERVER_URL}/health" > /dev/null; then
        echo -e "${YELLOW}Server not running. Please start it first.${NC}"
        return
    fi

    # Test simple query endpoint
    echo "Testing /api/v1/query endpoint..."
    hey -n 1000 -c 10 -m POST \
        -H "Content-Type: application/json" \
        -H "X-API-Key: ${API_KEY}" \
        -d '{"query":"SELECT * FROM test_table LIMIT 100","data_source":"dremio"}' \
        "${SERVER_URL}/api/v1/query" > hey_query_results.txt 2>&1

    # Test batch endpoint
    echo "Testing /api/v1/batch endpoint..."
    hey -n 500 -c 5 -m POST \
        -H "Content-Type: application/json" \
        -H "X-API-Key: ${API_KEY}" \
        -d '{"queries":[{"id":"q1","query":"SELECT 1","data_source":"dremio"},{"id":"q2","query":"SELECT 2","data_source":"dremio"}]}' \
        "${SERVER_URL}/api/v1/batch" > hey_batch_results.txt 2>&1

    # Test streaming endpoint
    echo "Testing /api/v1/stream endpoint..."
    hey -n 100 -c 2 -m POST \
        -H "Content-Type: application/json" \
        -H "X-API-Key: ${API_KEY}" \
        -d '{"table":"test_table","data_source":"dremio","chunk_size":100,"format":"ndjson"}' \
        "${SERVER_URL}/api/v1/stream" > hey_stream_results.txt 2>&1

    echo -e "${GREEN}✓ Load tests completed${NC}"
    echo ""
}

# Run Apache Bench tests
run_ab_test() {
    if ! command -v ab &> /dev/null; then
        return
    fi

    echo "Running Apache Bench tests..."
    echo "-----------------------------"

    SERVER_URL="http://localhost:8080"

    # Test health endpoint (no auth)
    ab -n 10000 -c 100 "${SERVER_URL}/health" > ab_health_results.txt 2>&1

    echo -e "${GREEN}✓ Apache Bench tests completed${NC}"
    echo ""
}

# Generate summary report
generate_report() {
    echo "Generating Performance Report..."
    echo "================================"
    echo ""

    REPORT_FILE="performance_report.md"

    cat > "$REPORT_FILE" << EOF
# Go Data Gateway Performance Report
Generated: $(date)

## Configuration
- Go Version: $(go version)
- CPU: $(nproc) cores
- Memory: $(free -h | grep Mem | awk '{print $2}')

## Go Benchmark Results

\`\`\`
$(tail -n 20 benchmark_results.txt)
\`\`\`

## Load Test Results (hey)

### Query Endpoint
\`\`\`
$(grep -A 10 "Summary" hey_query_results.txt 2>/dev/null || echo "No results")
\`\`\`

### Batch Endpoint
\`\`\`
$(grep -A 10 "Summary" hey_batch_results.txt 2>/dev/null || echo "No results")
\`\`\`

### Streaming Endpoint
\`\`\`
$(grep -A 10 "Summary" hey_stream_results.txt 2>/dev/null || echo "No results")
\`\`\`

## Performance Optimizations Applied

1. **Connection Pooling**: Implemented for Dremio Arrow Flight
   - Max connections: 10
   - Min connections: 2
   - Health check interval: 1 minute

2. **Caching**: Redis with 5-minute TTL
   - Cache hit ratio target: >60%

3. **Batch Processing**: Concurrent query execution
   - Max concurrency: 20
   - Chunk size: 1000 rows

4. **Streaming**: Multiple formats supported
   - NDJSON for real-time processing
   - CSV for data export
   - SSE for web clients

## Recommendations

1. Monitor cache hit ratio in production
2. Adjust connection pool size based on load
3. Use batch endpoints for multiple queries
4. Stream large datasets to reduce memory usage

EOF

    echo -e "${GREEN}✓ Report generated: $REPORT_FILE${NC}"
    echo ""
}

# Clean up old results
cleanup() {
    echo "Cleaning up old results..."
    rm -f *_results.txt 2>/dev/null || true
    echo ""
}

# Main execution
main() {
    check_dependencies
    cleanup

    echo "Select benchmark type:"
    echo "1. Full benchmark suite"
    echo "2. Go benchmarks only"
    echo "3. Load tests only"
    echo "4. Quick test"
    read -p "Enter choice (1-4): " choice

    case $choice in
        1)
            run_go_benchmarks
            run_load_test
            run_ab_test
            generate_report
            ;;
        2)
            run_go_benchmarks
            ;;
        3)
            run_load_test
            run_ab_test
            ;;
        4)
            echo "Running quick benchmark..."
            cd ../benchmark
            go test -bench=BenchmarkSimpleQuery -benchtime=3s
            ;;
        *)
            echo "Invalid choice"
            exit 1
            ;;
    esac

    echo ""
    echo -e "${GREEN}🎉 Benchmark complete!${NC}"

    # Show summary
    if [ -f "performance_report.md" ]; then
        echo ""
        echo "Summary:"
        echo "--------"
        grep -A 3 "Summary" hey_query_results.txt 2>/dev/null || true
    fi
}

# Run main function
main