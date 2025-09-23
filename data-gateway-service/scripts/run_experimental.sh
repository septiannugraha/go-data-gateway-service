#!/bin/bash

# Go 1.25 Experimental Features Runner
# This script enables experimental Go 1.25 features for testing

echo "🚀 Go 1.25 Experimental Features Test"
echo "======================================"
echo ""

# Colors for output
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Check Go version
echo "Checking Go version..."
go version

echo ""
echo "Select experimental feature to test:"
echo "1. JSON v2 - Faster JSON unmarshalling"
echo "2. Green Tea GC - New garbage collector (10-40% overhead reduction)"
echo "3. Container Awareness - Auto-adjust GOMAXPROCS"
echo "4. All features combined"
echo "5. Run benchmarks with features"
read -p "Enter choice (1-5): " choice

case $choice in
    1)
        echo -e "${BLUE}Testing JSON v2...${NC}"
        echo "Enabling GOEXPERIMENT=jsonv2"

        # Create a test file for JSON v2
        cat > test_jsonv2.go << 'EOF'
//go:build goexperiment.jsonv2

package main

import (
    "encoding/json/v2"
    "fmt"
    "time"
)

type TestData struct {
    ID        int       `json:"id"`
    Name      string    `json:"name"`
    Value     float64   `json:"value"`
    Timestamp time.Time `json:"timestamp"`
    Tags      []string  `json:"tags"`
}

func main() {
    // Test JSON v2 performance
    data := TestData{
        ID:        1,
        Name:      "Test",
        Value:     123.45,
        Timestamp: time.Now(),
        Tags:      []string{"go1.25", "experimental", "jsonv2"},
    }

    // Marshal
    start := time.Now()
    jsonData, _ := json.Marshal(data)
    fmt.Printf("Marshal time: %v\n", time.Since(start))

    // Unmarshal
    var decoded TestData
    start = time.Now()
    json.Unmarshal(jsonData, &decoded)
    fmt.Printf("Unmarshal time: %v\n", time.Since(start))

    fmt.Printf("Decoded: %+v\n", decoded)
}
EOF
        GOEXPERIMENT=jsonv2 go run test_jsonv2.go
        ;;

    2)
        echo -e "${GREEN}Testing Green Tea GC...${NC}"
        echo "Running server with GOEXPERIMENT=greenteagc"
        echo ""

        # Run with new GC
        GOEXPERIMENT=greenteagc \
        GODEBUG=gctrace=1 \
        ./server-chi &

        SERVER_PID=$!
        echo "Server running with PID: $SERVER_PID"
        echo "Monitoring GC for 10 seconds..."
        sleep 10

        # Send some requests to trigger GC
        for i in {1..100}; do
            curl -s http://localhost:8080/health > /dev/null 2>&1
        done

        kill $SERVER_PID 2>/dev/null
        echo "GC test completed"
        ;;

    3)
        echo -e "${YELLOW}Testing Container Awareness...${NC}"
        echo "Current GOMAXPROCS: $(nproc)"
        echo ""

        # Test with container limits
        echo "Testing CPU limit detection..."

        # Create a test program
        cat > test_container.go << 'EOF'
package main

import (
    "fmt"
    "runtime"
    "time"
)

func main() {
    fmt.Printf("Initial GOMAXPROCS: %d\n", runtime.GOMAXPROCS(0))
    fmt.Printf("NumCPU: %d\n", runtime.NumCPU())

    // Monitor for 5 seconds
    for i := 0; i < 5; i++ {
        time.Sleep(1 * time.Second)
        fmt.Printf("GOMAXPROCS after %d sec: %d\n", i+1, runtime.GOMAXPROCS(0))
    }
}
EOF

        # Run normally
        echo "Without container awareness:"
        go run test_container.go

        echo ""
        echo "With container awareness (simulated):"
        # This would automatically adjust in a real container
        GOMAXPROCS=2 go run test_container.go
        ;;

    4)
        echo -e "${GREEN}Running with ALL experimental features...${NC}"

        GOEXPERIMENT=jsonv2,greenteagc \
        GODEBUG=gctrace=1 \
        GOGC=100 \
        ./server-chi &

        SERVER_PID=$!
        echo "Server running with all features, PID: $SERVER_PID"
        echo "Press any key to stop..."
        read -n 1
        kill $SERVER_PID 2>/dev/null
        ;;

    5)
        echo -e "${BLUE}Running benchmarks with experimental features...${NC}"

        echo "Baseline benchmark (no experiments):"
        cd ../benchmark
        go test -bench=BenchmarkSimpleQuery -benchtime=10s -benchmem | head -20

        echo ""
        echo "With JSON v2:"
        GOEXPERIMENT=jsonv2 go test -bench=BenchmarkSimpleQuery -benchtime=10s -benchmem | head -20

        echo ""
        echo "With Green Tea GC:"
        GOEXPERIMENT=greenteagc go test -bench=BenchmarkSimpleQuery -benchtime=10s -benchmem | head -20

        echo ""
        echo "With all features:"
        GOEXPERIMENT=jsonv2,greenteagc go test -bench=BenchmarkSimpleQuery -benchtime=10s -benchmem | head -20
        ;;

    *)
        echo "Invalid choice"
        exit 1
        ;;
esac

echo ""
echo -e "${GREEN}✅ Experimental features test completed!${NC}"

# Cleanup
rm -f test_jsonv2.go test_container.go 2>/dev/null