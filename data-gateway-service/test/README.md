# API Testing Documentation

## Overview

This directory contains comprehensive tests for the Go Data Gateway API, covering all 19 endpoints with unit tests, integration tests, and performance benchmarks.

## Test Structure

```
test/
├── api/
│   ├── api_test.go           # Main test suite for all endpoints
│   ├── test_helpers.go       # Helper functions and utilities
│   └── fixtures/
│       └── test_data.json    # Test data fixtures
├── scripts/
│   └── test_api.sh          # Shell scripts for API testing
└── README.md                 # This file
```

## Running Tests

### Quick Start

```bash
# Run all tests
make test

# Run only API tests
make test-api

# Run with coverage
make test-coverage

# Run integration tests (requires services)
make test-integration

# Run performance tests
make test-performance
```

### Test Commands

#### Unit Tests (Fast, No External Dependencies)
```bash
go test -v -short ./test/api/...
```

#### Integration Tests (Requires Running Services)
```bash
# Start services first
docker-compose up -d

# Run integration tests
RUN_INTEGRATION_TESTS=true go test -v ./test/api/...
```

#### Performance Tests
```bash
RUN_PERF_TESTS=true go test -v -run TestPerformance ./test/api/...
```

#### Specific Test Suites
```bash
# Test only infrastructure endpoints
go test -v -run TestHealth ./test/api/...
go test -v -run TestReady ./test/api/...

# Test only query endpoints
go test -v -run TestQuery ./test/api/...
go test -v -run TestBatch ./test/api/...

# Test only tender endpoints
go test -v -run TestTender ./test/api/...

# Test authentication
go test -v -run TestAuth ./test/api/...
```

## Test Coverage

### Coverage Goals

| Component | Target | Current |
|-----------|--------|---------|
| Overall | 80% | - |
| Handlers | 90% | - |
| Middleware | 85% | - |
| Critical Paths | 100% | - |

### Generate Coverage Report

```bash
# Generate coverage report
make test-coverage

# View HTML coverage report
go tool cover -html=coverage.out

# Get coverage percentage
go test -cover ./test/api/...
```

## Test Categories

### 1. Infrastructure Tests
Tests for health checks, readiness, metrics, and cache stats:
- `TestHealthEndpoint`
- `TestReadyEndpoint`
- `TestMetricsEndpoint`
- `TestCacheStatsEndpoint`

### 2. Authentication Tests
Tests for API key and Bearer token authentication:
- `TestAuthenticationRequired`
- `TestAuthenticationWithAPIKey`
- `TestAuthenticationWithBearer`
- `TestInvalidAuthentication`

### 3. Query Endpoint Tests
Tests for single query, batch, and streaming:
- `TestQueryEndpoint`
- `TestBatchEndpoint`
- `TestStreamEndpoint`
- `TestStreamSSEEndpoint`

### 4. Tender Endpoint Tests
Tests for tender CRUD operations:
- `TestTenderListEndpoint`
- `TestTenderGetByIDEndpoint`
- `TestTenderSearchEndpoint`

### 5. RUP Endpoint Tests
Tests for RUP data operations:
- `TestRUPListEndpoint`
- `TestRUPGetByIDEndpoint`
- `TestRUPSearchEndpoint`

### 6. BigQuery Tests
Tests for BigQuery-specific operations:
- `TestBigQueryEstimateCost`

### 7. Error Handling Tests
Tests for various error scenarios:
- `TestErrorResponses`
- `TestInvalidJSON`
- `TestMissingRequiredFields`
- `TestServiceUnavailable`

### 8. Performance Tests
Benchmarks and load tests:
- `TestQueryEndpointPerformance`
- `TestBatchConcurrency`
- `TestStreamingPerformance`

## Environment Setup

### Required Environment Variables

```bash
# For integration tests
export DREMIO_HOST=localhost
export DREMIO_PORT=32010
export DREMIO_USERNAME=test_user
export DREMIO_PASSWORD=test_password

export BIGQUERY_PROJECT_ID=test-project
export BIGQUERY_DATASET_ID=test_dataset
export GOOGLE_APPLICATION_CREDENTIALS=/path/to/credentials.json

export REDIS_HOST=localhost
export REDIS_PORT=6379

# For performance tests
export RUN_PERF_TESTS=true
export PERF_TEST_DURATION=60s
export PERF_TEST_RPS=100
```

### Docker Compose for Testing

```yaml
version: '3.8'
services:
  redis:
    image: redis:7-alpine
    ports:
      - "6379:6379"

  dremio:
    image: dremio/dremio-oss:latest
    ports:
      - "9047:9047"
      - "31010:31010"
      - "32010:32010"
```

## Test Data Fixtures

Test data is stored in `fixtures/test_data.json` and includes:
- Sample tender data
- Sample RUP data
- Valid and invalid queries
- Batch request templates
- Expected error responses
- Performance targets

### Loading Test Data

```go
// In tests
data, err := LoadTestData("tenders.json")
require.NoError(t, err)
```

## Writing New Tests

### Test Template

```go
func (suite *APITestSuite) TestNewEndpoint() {
    // Arrange
    requestBody := map[string]interface{}{
        "field": "value",
    }

    // Act
    resp, err := suite.client.Post("/api/v1/endpoint", requestBody)
    require.NoError(suite.T(), err)
    defer resp.Body.Close()

    // Assert
    assert.Equal(suite.T(), http.StatusOK, resp.StatusCode)
    result := AssertSuccessResponse(suite.T(), resp)
    assert.NotNil(suite.T(), result["data"])
}
```

### Table-Driven Tests

```go
func (suite *APITestSuite) TestEndpointVariations() {
    tests := []struct {
        name           string
        input          map[string]interface{}
        expectedStatus int
        expectedError  string
    }{
        {
            name:           "Valid request",
            input:          map[string]interface{}{"field": "value"},
            expectedStatus: http.StatusOK,
        },
        {
            name:           "Missing field",
            input:          map[string]interface{}{},
            expectedStatus: http.StatusBadRequest,
            expectedError:  "field is required",
        },
    }

    for _, tt := range tests {
        suite.T().Run(tt.name, func(t *testing.T) {
            resp, err := suite.client.Post("/api/v1/endpoint", tt.input)
            require.NoError(t, err)
            defer resp.Body.Close()

            assert.Equal(t, tt.expectedStatus, resp.StatusCode)
            if tt.expectedError != "" {
                AssertErrorResponse(t, resp, tt.expectedStatus, tt.expectedError)
            }
        })
    }
}
```

## Performance Testing

### Load Testing with hey

```bash
# Install hey
go install github.com/rakyll/hey@latest

# Basic load test
hey -n 1000 -c 10 -H "X-API-Key: test-key" \
    http://localhost:8080/api/v1/tender

# POST request load test
hey -n 1000 -c 10 -m POST \
    -H "X-API-Key: test-key" \
    -H "Content-Type: application/json" \
    -d '{"sql":"SELECT 1","source":"DATAWAREHOUSE"}' \
    http://localhost:8080/api/v1/query
```

### Performance Benchmarks

```go
func BenchmarkQueryEndpoint(b *testing.B) {
    client := NewTestClient("http://localhost:8080", "test-key")
    body := map[string]string{
        "sql": "SELECT 1",
        "source": "DATAWAREHOUSE",
    }

    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        resp, _ := client.Post("/api/v1/query", body)
        resp.Body.Close()
    }
}
```

Run benchmarks:
```bash
go test -bench=. -benchmem ./test/api/...
```

## Continuous Integration

### GitHub Actions Workflow

```yaml
name: API Tests
on: [push, pull_request]

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v2

      - name: Set up Go
        uses: actions/setup-go@v2
        with:
          go-version: 1.25

      - name: Start services
        run: docker-compose up -d

      - name: Run tests
        run: make test-api

      - name: Upload coverage
        uses: codecov/codecov-action@v2
        with:
          file: ./coverage.out
```

## Debugging Tests

### Verbose Output

```bash
# Run with verbose output
go test -v ./test/api/...

# Run with detailed logging
LOG_LEVEL=debug go test -v ./test/api/...
```

### Debug Single Test

```bash
# Run single test with debugging
go test -v -run TestQueryEndpoint ./test/api/...

# With Delve debugger
dlv test ./test/api -- -test.run TestQueryEndpoint
```

### Test Timeout

```bash
# Set custom timeout for long-running tests
go test -timeout 30m ./test/api/...
```

## Common Issues and Solutions

### Issue: Tests fail with "connection refused"
**Solution**: Ensure all required services are running
```bash
docker-compose up -d
# Wait for services to be ready
sleep 10
```

### Issue: Authentication tests fail
**Solution**: Check API key configuration
```bash
export API_KEYS="test-api-key-123,demo-key-456"
```

### Issue: BigQuery tests fail
**Solution**: Ensure credentials are set
```bash
export GOOGLE_APPLICATION_CREDENTIALS=/path/to/credentials.json
```

### Issue: Slow tests
**Solution**: Run only unit tests for faster feedback
```bash
go test -short ./test/api/...
```

## Test Maintenance

### Weekly Tasks
- [ ] Review and update test coverage
- [ ] Check for flaky tests
- [ ] Update test data fixtures

### Monthly Tasks
- [ ] Performance regression testing
- [ ] Security testing review
- [ ] Update test documentation

### Before Release
- [ ] Full test suite execution
- [ ] Performance benchmarks
- [ ] Load testing
- [ ] Security scan
- [ ] Coverage report review

## Contributing

When adding new endpoints or modifying existing ones:

1. **Update OpenAPI Spec**: Modify `api/swagger.yaml`
2. **Add Tests**: Create tests in `api_test.go`
3. **Update Fixtures**: Add test data to `fixtures/`
4. **Update Documentation**: Update this README
5. **Run Full Suite**: Ensure all tests pass

## Contact

For questions or issues related to testing:
- Create an issue in the repository
- Contact the development team

## License

See LICENSE file in the root directory.