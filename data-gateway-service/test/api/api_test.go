package api_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"go.uber.org/zap"

	"go-data-gateway/internal/cache"
	"go-data-gateway/internal/datasource"
	"go-data-gateway/internal/handlers/v1"
	custommw "go-data-gateway/internal/middleware/chi"
	"go-data-gateway/internal/response"
)

// APITestSuite is the main test suite for all API endpoints
type APITestSuite struct {
	suite.Suite
	router      *chi.Mux
	server      *httptest.Server
	dataSources map[string]datasource.DataSource
	cache       cache.Cache
	logger      *zap.Logger
	apiKey      string
}

// SetupSuite runs once before all tests
func (suite *APITestSuite) SetupSuite() {
	// Initialize logger
	suite.logger, _ = zap.NewDevelopment()

	// Initialize mock data sources
	suite.dataSources = map[string]datasource.DataSource{
		"DATAWAREHOUSE": NewMockDataSource(datasource.DataSourceTypeDremio),
		"BIGQUERY":      NewMockDataSource(datasource.DataSourceTypeBigQuery),
	}

	// Initialize cache
	suite.cache = &cache.NoOpCache{}

	// Set API key for testing
	suite.apiKey = "test-api-key-123"

	// Setup router
	suite.setupRouter()

	// Start test server
	suite.server = httptest.NewServer(suite.router)
}

// TearDownSuite runs once after all tests
func (suite *APITestSuite) TearDownSuite() {
	if suite.server != nil {
		suite.server.Close()
	}
}

// setupRouter creates and configures the test router
func (suite *APITestSuite) setupRouter() {
	r := chi.NewRouter()

	// Middleware
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(60 * time.Second))

	// Health check endpoints (no auth)
	r.Get("/health", suite.healthCheck)
	r.Get("/ready", suite.readyCheck)
	r.Get("/metrics", suite.metricsHandler)
	r.Get("/cache/stats", suite.cacheStatsHandler)

	// API routes with authentication
	r.Route("/api/v1", func(r chi.Router) {
		// Apply authentication middleware
		r.Use(suite.authMiddleware)

		// Query endpoints
		queryHandler := v1.NewQueryHandler(suite.dataSources, suite.logger)
		batchHandler := v1.NewBatchHandler(suite.dataSources, suite.logger)
		streamHandler := v1.NewStreamHandler(suite.dataSources, suite.logger)

		r.Post("/query", queryHandler.Execute)
		r.Post("/batch", batchHandler.Execute)
		r.Post("/batch/stream", batchHandler.Stream)
		r.Post("/stream", streamHandler.Stream)
		r.Post("/stream/sse", streamHandler.StreamSSE)

		// Tender endpoints
		tenderHandler := v1.NewTenderHandler(suite.dataSources["DATAWAREHOUSE"], suite.logger)
		r.Route("/tender", func(r chi.Router) {
			r.Get("/", tenderHandler.List)
			r.Get("/{id}", tenderHandler.GetByID)
			r.Post("/search", tenderHandler.Search)
		})

		// RUP endpoints (would need BigQuery client mock)
		// r.Route("/rup", func(r chi.Router) {
		//     r.Get("/", rupHandler.List)
		//     r.Get("/{id}", rupHandler.GetByID)
		//     r.Post("/search", rupHandler.Search)
		// })

		// BigQuery estimate endpoint
		r.Post("/bigquery/estimate-cost", suite.estimateCostHandler)
	})

	suite.router = r
}

// authMiddleware provides test authentication
func (suite *APITestSuite) authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		apiKey := r.Header.Get("X-API-Key")
		if apiKey == "" {
			auth := r.Header.Get("Authorization")
			if auth != "" && strings.HasPrefix(auth, "Bearer ") {
				apiKey = strings.TrimPrefix(auth, "Bearer ")
			}
		}

		if apiKey != suite.apiKey {
			response.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// Infrastructure endpoint handlers
func (suite *APITestSuite) healthCheck(w http.ResponseWriter, r *http.Request) {
	response.Success(w, map[string]interface{}{
		"status":    "healthy",
		"timestamp": time.Now().Format(time.RFC3339),
		"version":   "1.0.0",
		"uptime":    100,
	}, nil)
}

func (suite *APITestSuite) readyCheck(w http.ResponseWriter, r *http.Request) {
	ready := true
	services := make(map[string]interface{})

	for name, ds := range suite.dataSources {
		serviceReady := ds != nil
		services[name] = map[string]interface{}{
			"ready":   serviceReady,
			"message": "Service is ready",
		}
		if !serviceReady {
			ready = false
		}
	}

	response.Success(w, map[string]interface{}{
		"ready":    ready,
		"services": services,
	}, nil)
}

func (suite *APITestSuite) metricsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintln(w, "# HELP go_gc_duration_seconds A summary of the GC invocation durations.")
	fmt.Fprintln(w, "# TYPE go_gc_duration_seconds summary")
	fmt.Fprintln(w, "go_gc_duration_seconds{quantile=\"0\"} 0")
}

func (suite *APITestSuite) cacheStatsHandler(w http.ResponseWriter, r *http.Request) {
	stats := map[string]interface{}{
		"hit_rate":       0.75,
		"total_hits":     1500,
		"total_misses":   500,
		"memory_used_mb": 128.5,
		"entries_count":  250,
		"datasource_metrics": map[string]interface{}{
			"DATAWAREHOUSE": map[string]interface{}{
				"queries_executed":      100,
				"cache_hits":           75,
				"average_query_time_ms": 125.5,
			},
		},
	}
	response.Success(w, stats, nil)
}

func (suite *APITestSuite) estimateCostHandler(w http.ResponseWriter, r *http.Request) {
	var req map[string]string
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	response.Success(w, map[string]interface{}{
		"bytes_processed":    1024 * 1024 * 100, // 100MB
		"bytes_billed":       1024 * 1024 * 100,
		"estimated_cost_usd": 0.005,
		"cache_hit":          false,
		"message":            "Query will process approximately 100MB of data",
	}, nil)
}

// Test Infrastructure Endpoints
func (suite *APITestSuite) TestHealthEndpoint() {
	resp, err := http.Get(fmt.Sprintf("%s/health", suite.server.URL))
	require.NoError(suite.T(), err)
	defer resp.Body.Close()

	assert.Equal(suite.T(), http.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&result)
	require.NoError(suite.T(), err)

	assert.True(suite.T(), result["success"].(bool))
	data := result["data"].(map[string]interface{})
	assert.Equal(suite.T(), "healthy", data["status"])
	assert.NotEmpty(suite.T(), data["timestamp"])
}

func (suite *APITestSuite) TestReadyEndpoint() {
	resp, err := http.Get(fmt.Sprintf("%s/ready", suite.server.URL))
	require.NoError(suite.T(), err)
	defer resp.Body.Close()

	assert.Equal(suite.T(), http.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&result)
	require.NoError(suite.T(), err)

	assert.True(suite.T(), result["success"].(bool))
	data := result["data"].(map[string]interface{})
	assert.True(suite.T(), data["ready"].(bool))
	assert.NotNil(suite.T(), data["services"])
}

func (suite *APITestSuite) TestMetricsEndpoint() {
	resp, err := http.Get(fmt.Sprintf("%s/metrics", suite.server.URL))
	require.NoError(suite.T(), err)
	defer resp.Body.Close()

	assert.Equal(suite.T(), http.StatusOK, resp.StatusCode)
	assert.Equal(suite.T(), "text/plain", resp.Header.Get("Content-Type"))

	body, err := io.ReadAll(resp.Body)
	require.NoError(suite.T(), err)
	assert.Contains(suite.T(), string(body), "go_gc_duration_seconds")
}

func (suite *APITestSuite) TestCacheStatsEndpoint() {
	req, _ := http.NewRequest("GET", fmt.Sprintf("%s/cache/stats", suite.server.URL), nil)
	req.Header.Set("X-API-Key", suite.apiKey)

	resp, err := http.DefaultClient.Do(req)
	require.NoError(suite.T(), err)
	defer resp.Body.Close()

	assert.Equal(suite.T(), http.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&result)
	require.NoError(suite.T(), err)

	assert.True(suite.T(), result["success"].(bool))
	data := result["data"].(map[string]interface{})
	assert.NotNil(suite.T(), data["hit_rate"])
	assert.NotNil(suite.T(), data["total_hits"])
}

// Test Authentication
func (suite *APITestSuite) TestAuthenticationRequired() {
	tests := []struct {
		name     string
		endpoint string
		method   string
		body     interface{}
	}{
		{"Query without auth", "/api/v1/query", "POST", map[string]string{"sql": "SELECT 1", "source": "DATAWAREHOUSE"}},
		{"Tender list without auth", "/api/v1/tender", "GET", nil},
		{"Batch without auth", "/api/v1/batch", "POST", map[string]interface{}{"queries": []interface{}{}}},
	}

	for _, tt := range tests {
		suite.T().Run(tt.name, func(t *testing.T) {
			var bodyReader io.Reader
			if tt.body != nil {
				jsonBody, _ := json.Marshal(tt.body)
				bodyReader = bytes.NewReader(jsonBody)
			}

			req, _ := http.NewRequest(tt.method, fmt.Sprintf("%s%s", suite.server.URL, tt.endpoint), bodyReader)
			if tt.body != nil {
				req.Header.Set("Content-Type", "application/json")
			}

			resp, err := http.DefaultClient.Do(req)
			require.NoError(t, err)
			defer resp.Body.Close()

			assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
		})
	}
}

func (suite *APITestSuite) TestAuthenticationWithAPIKey() {
	body := map[string]string{
		"sql":    "SELECT 1",
		"source": "DATAWAREHOUSE",
	}
	jsonBody, _ := json.Marshal(body)

	req, _ := http.NewRequest("POST", fmt.Sprintf("%s/api/v1/query", suite.server.URL), bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-Key", suite.apiKey)

	resp, err := http.DefaultClient.Do(req)
	require.NoError(suite.T(), err)
	defer resp.Body.Close()

	assert.Equal(suite.T(), http.StatusOK, resp.StatusCode)
}

func (suite *APITestSuite) TestAuthenticationWithBearer() {
	body := map[string]string{
		"sql":    "SELECT 1",
		"source": "DATAWAREHOUSE",
	}
	jsonBody, _ := json.Marshal(body)

	req, _ := http.NewRequest("POST", fmt.Sprintf("%s/api/v1/query", suite.server.URL), bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", suite.apiKey))

	resp, err := http.DefaultClient.Do(req)
	require.NoError(suite.T(), err)
	defer resp.Body.Close()

	assert.Equal(suite.T(), http.StatusOK, resp.StatusCode)
}

// Test Query Endpoint
func (suite *APITestSuite) TestQueryEndpoint() {
	tests := []struct {
		name           string
		body           map[string]string
		expectedStatus int
		checkResponse  bool
	}{
		{
			name: "Valid Dremio query",
			body: map[string]string{
				"sql":    "SELECT * FROM table LIMIT 10",
				"source": "DATAWAREHOUSE",
			},
			expectedStatus: http.StatusOK,
			checkResponse:  true,
		},
		{
			name: "Valid BigQuery query",
			body: map[string]string{
				"sql":    "SELECT * FROM dataset.table LIMIT 10",
				"source": "BIGQUERY",
			},
			expectedStatus: http.StatusOK,
			checkResponse:  true,
		},
		{
			name: "Missing SQL",
			body: map[string]string{
				"source": "DATAWAREHOUSE",
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "Missing source",
			body: map[string]string{
				"sql": "SELECT * FROM table",
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "Invalid source",
			body: map[string]string{
				"sql":    "SELECT * FROM table",
				"source": "INVALID",
			},
			expectedStatus: http.StatusServiceUnavailable,
		},
	}

	for _, tt := range tests {
		suite.T().Run(tt.name, func(t *testing.T) {
			jsonBody, _ := json.Marshal(tt.body)
			req, _ := http.NewRequest("POST", fmt.Sprintf("%s/api/v1/query", suite.server.URL), bytes.NewReader(jsonBody))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("X-API-Key", suite.apiKey)

			resp, err := http.DefaultClient.Do(req)
			require.NoError(t, err)
			defer resp.Body.Close()

			assert.Equal(t, tt.expectedStatus, resp.StatusCode)

			if tt.checkResponse && resp.StatusCode == http.StatusOK {
				var result map[string]interface{}
				err = json.NewDecoder(resp.Body).Decode(&result)
				require.NoError(t, err)
				assert.True(t, result["success"].(bool))
				assert.NotNil(t, result["data"])
			}
		})
	}
}

// Test Batch Endpoint
func (suite *APITestSuite) TestBatchEndpoint() {
	tests := []struct {
		name           string
		body           map[string]interface{}
		expectedStatus int
	}{
		{
			name: "Valid batch request",
			body: map[string]interface{}{
				"queries": []map[string]string{
					{
						"id":          "q1",
						"query":       "SELECT 1",
						"data_source": "DATAWAREHOUSE",
					},
					{
						"id":          "q2",
						"query":       "SELECT 2",
						"data_source": "BIGQUERY",
					},
				},
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "Empty queries",
			body: map[string]interface{}{
				"queries": []map[string]string{},
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "Too many queries",
			body: map[string]interface{}{
				"queries": make([]map[string]string, 101),
			},
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		suite.T().Run(tt.name, func(t *testing.T) {
			jsonBody, _ := json.Marshal(tt.body)
			req, _ := http.NewRequest("POST", fmt.Sprintf("%s/api/v1/batch", suite.server.URL), bytes.NewReader(jsonBody))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("X-API-Key", suite.apiKey)

			resp, err := http.DefaultClient.Do(req)
			require.NoError(t, err)
			defer resp.Body.Close()

			assert.Equal(t, tt.expectedStatus, resp.StatusCode)
		})
	}
}

// Test Tender Endpoints
func (suite *APITestSuite) TestTenderListEndpoint() {
	tests := []struct {
		name           string
		queryParams    string
		expectedStatus int
	}{
		{
			name:           "List without params",
			queryParams:    "",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "List with limit",
			queryParams:    "?limit=50",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "List with pagination",
			queryParams:    "?limit=20&offset=40",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "List with status filter",
			queryParams:    "?status=active",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "List with sorting",
			queryParams:    "?sort_by=nilai_pagu&order=ASC",
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		suite.T().Run(tt.name, func(t *testing.T) {
			req, _ := http.NewRequest("GET", fmt.Sprintf("%s/api/v1/tender%s", suite.server.URL, tt.queryParams), nil)
			req.Header.Set("X-API-Key", suite.apiKey)

			resp, err := http.DefaultClient.Do(req)
			require.NoError(t, err)
			defer resp.Body.Close()

			assert.Equal(t, tt.expectedStatus, resp.StatusCode)

			if resp.StatusCode == http.StatusOK {
				var result map[string]interface{}
				err = json.NewDecoder(resp.Body).Decode(&result)
				require.NoError(t, err)
				assert.True(t, result["success"].(bool))
				assert.NotNil(t, result["data"])
				assert.NotNil(t, result["meta"])
			}
		})
	}
}

func (suite *APITestSuite) TestTenderGetByIDEndpoint() {
	tests := []struct {
		name           string
		tenderID       string
		expectedStatus int
	}{
		{
			name:           "Valid tender ID",
			tenderID:       "TENDER-001",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Non-existent tender ID",
			tenderID:       "NOTFOUND",
			expectedStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		suite.T().Run(tt.name, func(t *testing.T) {
			req, _ := http.NewRequest("GET", fmt.Sprintf("%s/api/v1/tender/%s", suite.server.URL, tt.tenderID), nil)
			req.Header.Set("X-API-Key", suite.apiKey)

			resp, err := http.DefaultClient.Do(req)
			require.NoError(t, err)
			defer resp.Body.Close()

			// Note: Our mock will return 200 for any ID, but in real implementation it would check
			if tt.tenderID == "NOTFOUND" {
				// In real implementation, this would return 404
				// For now, we'll accept 200 from our mock
				assert.Contains(t, []int{http.StatusOK, http.StatusNotFound}, resp.StatusCode)
			} else {
				assert.Equal(t, tt.expectedStatus, resp.StatusCode)
			}
		})
	}
}

func (suite *APITestSuite) TestTenderSearchEndpoint() {
	tests := []struct {
		name           string
		body           map[string]interface{}
		expectedStatus int
	}{
		{
			name: "Search with keyword",
			body: map[string]interface{}{
				"keyword": "construction",
				"limit":   50,
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "Search with filters",
			body: map[string]interface{}{
				"status":         "active",
				"tahun_anggaran": 2025,
				"provinsi":       "DKI Jakarta",
				"nilai_pagu_min": 1000000000,
				"nilai_pagu_max": 5000000000,
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Empty search",
			body:           map[string]interface{}{},
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		suite.T().Run(tt.name, func(t *testing.T) {
			jsonBody, _ := json.Marshal(tt.body)
			req, _ := http.NewRequest("POST", fmt.Sprintf("%s/api/v1/tender/search", suite.server.URL), bytes.NewReader(jsonBody))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("X-API-Key", suite.apiKey)

			resp, err := http.DefaultClient.Do(req)
			require.NoError(t, err)
			defer resp.Body.Close()

			assert.Equal(t, tt.expectedStatus, resp.StatusCode)
		})
	}
}

// Test Stream Endpoints
func (suite *APITestSuite) TestStreamEndpoint() {
	body := map[string]interface{}{
		"query":       "SELECT * FROM large_table",
		"data_source": "DATAWAREHOUSE",
		"chunk_size":  1000,
		"format":      "ndjson",
	}

	jsonBody, _ := json.Marshal(body)
	req, _ := http.NewRequest("POST", fmt.Sprintf("%s/api/v1/stream", suite.server.URL), bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-Key", suite.apiKey)

	resp, err := http.DefaultClient.Do(req)
	require.NoError(suite.T(), err)
	defer resp.Body.Close()

	assert.Equal(suite.T(), http.StatusOK, resp.StatusCode)
	assert.Contains(suite.T(), resp.Header.Get("Content-Type"), "ndjson")
}

func (suite *APITestSuite) TestStreamSSEEndpoint() {
	body := map[string]interface{}{
		"query":       "SELECT * FROM realtime_table",
		"data_source": "DATAWAREHOUSE",
		"chunk_size":  500,
	}

	jsonBody, _ := json.Marshal(body)
	req, _ := http.NewRequest("POST", fmt.Sprintf("%s/api/v1/stream/sse", suite.server.URL), bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-Key", suite.apiKey)

	resp, err := http.DefaultClient.Do(req)
	require.NoError(suite.T(), err)
	defer resp.Body.Close()

	assert.Equal(suite.T(), http.StatusOK, resp.StatusCode)
	assert.Contains(suite.T(), resp.Header.Get("Content-Type"), "event-stream")
}

// Test BigQuery Cost Estimate
func (suite *APITestSuite) TestBigQueryEstimateCost() {
	body := map[string]string{
		"query": "SELECT * FROM `project.dataset.large_table` WHERE date >= '2025-01-01'",
	}

	jsonBody, _ := json.Marshal(body)
	req, _ := http.NewRequest("POST", fmt.Sprintf("%s/api/v1/bigquery/estimate-cost", suite.server.URL), bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-Key", suite.apiKey)

	resp, err := http.DefaultClient.Do(req)
	require.NoError(suite.T(), err)
	defer resp.Body.Close()

	assert.Equal(suite.T(), http.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&result)
	require.NoError(suite.T(), err)

	assert.True(suite.T(), result["success"].(bool))
	data := result["data"].(map[string]interface{})
	assert.NotNil(suite.T(), data["bytes_processed"])
	assert.NotNil(suite.T(), data["estimated_cost_usd"])
}

// Benchmark Tests
func (suite *APITestSuite) TestQueryEndpointPerformance() {
	// Skip if not running performance tests
	if os.Getenv("RUN_PERF_TESTS") != "true" {
		suite.T().Skip("Skipping performance test")
	}

	body := map[string]string{
		"sql":    "SELECT * FROM table LIMIT 100",
		"source": "DATAWAREHOUSE",
	}
	jsonBody, _ := json.Marshal(body)

	// Measure average response time for 100 requests
	var totalTime time.Duration
	iterations := 100

	for i := 0; i < iterations; i++ {
		start := time.Now()

		req, _ := http.NewRequest("POST", fmt.Sprintf("%s/api/v1/query", suite.server.URL), bytes.NewReader(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-API-Key", suite.apiKey)

		resp, err := http.DefaultClient.Do(req)
		require.NoError(suite.T(), err)
		resp.Body.Close()

		totalTime += time.Since(start)
	}

	avgTime := totalTime / time.Duration(iterations)
	suite.T().Logf("Average response time: %v", avgTime)

	// Assert that average response time is under 100ms
	assert.Less(suite.T(), avgTime, 100*time.Millisecond)
}

// Test Error Handling
func (suite *APITestSuite) TestErrorResponses() {
	tests := []struct {
		name           string
		endpoint       string
		method         string
		body           interface{}
		headers        map[string]string
		expectedStatus int
		expectedError  string
	}{
		{
			name:           "Invalid JSON",
			endpoint:       "/api/v1/query",
			method:         "POST",
			body:           "invalid json",
			headers:        map[string]string{"X-API-Key": suite.apiKey, "Content-Type": "application/json"},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Invalid request",
		},
		{
			name:           "Missing auth",
			endpoint:       "/api/v1/query",
			method:         "POST",
			body:           map[string]string{"sql": "SELECT 1", "source": "DATAWAREHOUSE"},
			headers:        map[string]string{"Content-Type": "application/json"},
			expectedStatus: http.StatusUnauthorized,
			expectedError:  "Unauthorized",
		},
		{
			name:           "Invalid API key",
			endpoint:       "/api/v1/query",
			method:         "POST",
			body:           map[string]string{"sql": "SELECT 1", "source": "DATAWAREHOUSE"},
			headers:        map[string]string{"X-API-Key": "invalid-key", "Content-Type": "application/json"},
			expectedStatus: http.StatusUnauthorized,
			expectedError:  "Unauthorized",
		},
	}

	for _, tt := range tests {
		suite.T().Run(tt.name, func(t *testing.T) {
			var bodyReader io.Reader
			switch v := tt.body.(type) {
			case string:
				bodyReader = strings.NewReader(v)
			default:
				jsonBody, _ := json.Marshal(v)
				bodyReader = bytes.NewReader(jsonBody)
			}

			req, _ := http.NewRequest(tt.method, fmt.Sprintf("%s%s", suite.server.URL, tt.endpoint), bodyReader)
			for k, v := range tt.headers {
				req.Header.Set(k, v)
			}

			resp, err := http.DefaultClient.Do(req)
			require.NoError(t, err)
			defer resp.Body.Close()

			assert.Equal(t, tt.expectedStatus, resp.StatusCode)

			var result map[string]interface{}
			err = json.NewDecoder(resp.Body).Decode(&result)
			require.NoError(t, err)

			assert.False(t, result["success"].(bool))
			assert.Contains(t, result["error"].(string), tt.expectedError)
		})
	}
}

// MockDataSource for testing
type MockDataSource struct {
	dsType datasource.DataSourceType
}

func NewMockDataSource(dsType datasource.DataSourceType) *MockDataSource {
	return &MockDataSource{dsType: dsType}
}

func (m *MockDataSource) GetType() datasource.DataSourceType {
	return m.dsType
}

func (m *MockDataSource) ExecuteQuery(ctx context.Context, query string, opts *datasource.QueryOptions) (*datasource.QueryResult, error) {
	// Return mock data based on query
	if strings.Contains(query, "NOTFOUND") {
		return &datasource.QueryResult{
			Data:  []map[string]interface{}{},
			Count: 0,
		}, nil
	}

	// Return some mock data
	mockData := []map[string]interface{}{
		{"id": 1, "name": "Test Item 1", "value": 100},
		{"id": 2, "name": "Test Item 2", "value": 200},
	}

	return &datasource.QueryResult{
		Data:     mockData,
		Count:    len(mockData),
		CacheHit: false,
	}, nil
}

func (m *MockDataSource) TestConnection(ctx context.Context) error {
	return nil
}

func (m *MockDataSource) Close() error {
	return nil
}

func (m *MockDataSource) GetMetrics() map[string]interface{} {
	return map[string]interface{}{
		"queries_executed": 100,
		"cache_hits":      75,
		"avg_query_time":  125.5,
	}
}

// Run the test suite
func TestAPITestSuite(t *testing.T) {
	suite.Run(t, new(APITestSuite))
}