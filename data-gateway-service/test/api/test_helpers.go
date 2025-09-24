package api_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// TestClient provides helper methods for API testing
type TestClient struct {
	BaseURL string
	APIKey  string
	Client  *http.Client
}

// NewTestClient creates a new test client
func NewTestClient(baseURL, apiKey string) *TestClient {
	return &TestClient{
		BaseURL: baseURL,
		APIKey:  apiKey,
		Client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// DoRequest performs an HTTP request with common headers
func (tc *TestClient) DoRequest(method, endpoint string, body interface{}) (*http.Response, error) {
	var bodyReader io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		bodyReader = bytes.NewReader(jsonBody)
	}

	req, err := http.NewRequest(method, fmt.Sprintf("%s%s", tc.BaseURL, endpoint), bodyReader)
	if err != nil {
		return nil, err
	}

	// Add common headers
	if tc.APIKey != "" {
		req.Header.Set("X-API-Key", tc.APIKey)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	return tc.Client.Do(req)
}

// Get performs a GET request
func (tc *TestClient) Get(endpoint string) (*http.Response, error) {
	return tc.DoRequest("GET", endpoint, nil)
}

// Post performs a POST request
func (tc *TestClient) Post(endpoint string, body interface{}) (*http.Response, error) {
	return tc.DoRequest("POST", endpoint, body)
}

// Put performs a PUT request
func (tc *TestClient) Put(endpoint string, body interface{}) (*http.Response, error) {
	return tc.DoRequest("PUT", endpoint, body)
}

// Delete performs a DELETE request
func (tc *TestClient) Delete(endpoint string) (*http.Response, error) {
	return tc.DoRequest("DELETE", endpoint, nil)
}

// AssertSuccessResponse checks if response is successful
func AssertSuccessResponse(t *testing.T, resp *http.Response) map[string]interface{} {
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	err := json.NewDecoder(resp.Body).Decode(&result)
	require.NoError(t, err)

	require.True(t, result["success"].(bool), "Response should be successful")
	return result
}

// AssertErrorResponse checks if response contains expected error
func AssertErrorResponse(t *testing.T, resp *http.Response, expectedStatus int, expectedError string) {
	require.Equal(t, expectedStatus, resp.StatusCode)

	var result map[string]interface{}
	err := json.NewDecoder(resp.Body).Decode(&result)
	require.NoError(t, err)

	require.False(t, result["success"].(bool), "Response should not be successful")
	require.Contains(t, result["error"].(string), expectedError)
}

// TestDataGenerator generates test data for various endpoints
type TestDataGenerator struct{}

// NewTestDataGenerator creates a new test data generator
func NewTestDataGenerator() *TestDataGenerator {
	return &TestDataGenerator{}
}

// GenerateQueryRequest generates a test query request
func (g *TestDataGenerator) GenerateQueryRequest(source string) map[string]string {
	queries := map[string]string{
		"DATAWAREHOUSE": "SELECT * FROM nessie_iceberg.tender_data LIMIT 10",
		"BIGQUERY":      "SELECT * FROM `project.dataset.table` LIMIT 10",
	}

	query := queries[source]
	if query == "" {
		query = "SELECT 1"
	}

	return map[string]string{
		"sql":    query,
		"source": source,
	}
}

// GenerateBatchRequest generates a test batch request
func (g *TestDataGenerator) GenerateBatchRequest(numQueries int) map[string]interface{} {
	queries := make([]map[string]string, numQueries)
	for i := 0; i < numQueries; i++ {
		source := "DATAWAREHOUSE"
		if i%2 == 0 {
			source = "BIGQUERY"
		}
		queries[i] = map[string]string{
			"id":          fmt.Sprintf("query-%d", i+1),
			"query":       fmt.Sprintf("SELECT %d as num", i+1),
			"data_source": source,
		}
	}

	return map[string]interface{}{
		"queries": queries,
		"options": map[string]interface{}{
			"max_concurrency": 5,
			"timeout":         30,
		},
	}
}

// GenerateStreamRequest generates a test stream request
func (g *TestDataGenerator) GenerateStreamRequest(source, format string) map[string]interface{} {
	return map[string]interface{}{
		"query":       "SELECT * FROM large_table",
		"data_source": source,
		"chunk_size":  1000,
		"format":      format,
		"options": map[string]interface{}{
			"timeout": 60,
		},
	}
}

// GenerateTenderSearchRequest generates a tender search request
func (g *TestDataGenerator) GenerateTenderSearchRequest() map[string]interface{} {
	return map[string]interface{}{
		"keyword":        "construction",
		"status":         "active",
		"tahun_anggaran": 2025,
		"provinsi":       "DKI Jakarta",
		"nilai_pagu_min": 1000000000,
		"nilai_pagu_max": 5000000000,
		"limit":          50,
		"offset":         0,
	}
}

// GenerateRUPSearchRequest generates a RUP search request
func (g *TestDataGenerator) GenerateRUPSearchRequest() map[string]interface{} {
	return map[string]interface{}{
		"keyword":        "procurement",
		"tahun_anggaran": 2025,
		"kd_klpd":        "K001",
		"nama_klpd":      "Ministry",
		"pagu_min":       500000000,
		"pagu_max":       2000000000,
		"limit":          25,
		"offset":         0,
	}
}

// BenchmarkHelper provides utilities for benchmark testing
type BenchmarkHelper struct {
	Client *TestClient
}

// NewBenchmarkHelper creates a new benchmark helper
func NewBenchmarkHelper(client *TestClient) *BenchmarkHelper {
	return &BenchmarkHelper{
		Client: client,
	}
}

// MeasureEndpointLatency measures average latency for an endpoint
func (h *BenchmarkHelper) MeasureEndpointLatency(t *testing.T, method, endpoint string, body interface{}, iterations int) time.Duration {
	var totalTime time.Duration

	for i := 0; i < iterations; i++ {
		start := time.Now()

		resp, err := h.Client.DoRequest(method, endpoint, body)
		require.NoError(t, err)
		resp.Body.Close()

		totalTime += time.Since(start)
	}

	return totalTime / time.Duration(iterations)
}

// RunConcurrentRequests runs concurrent requests and measures performance
func (h *BenchmarkHelper) RunConcurrentRequests(t *testing.T, method, endpoint string, body interface{}, concurrency int) []time.Duration {
	results := make(chan time.Duration, concurrency)

	for i := 0; i < concurrency; i++ {
		go func() {
			start := time.Now()
			resp, err := h.Client.DoRequest(method, endpoint, body)
			if err == nil {
				resp.Body.Close()
			}
			results <- time.Since(start)
		}()
	}

	// Collect results
	durations := make([]time.Duration, concurrency)
	for i := 0; i < concurrency; i++ {
		durations[i] = <-results
	}

	return durations
}

// MockResponseWriter for testing streaming responses
type MockResponseWriter struct {
	Headers    http.Header
	Body       *bytes.Buffer
	StatusCode int
}

func NewMockResponseWriter() *MockResponseWriter {
	return &MockResponseWriter{
		Headers:    make(http.Header),
		Body:       new(bytes.Buffer),
		StatusCode: http.StatusOK,
	}
}

func (m *MockResponseWriter) Header() http.Header {
	return m.Headers
}

func (m *MockResponseWriter) Write(b []byte) (int, error) {
	return m.Body.Write(b)
}

func (m *MockResponseWriter) WriteHeader(statusCode int) {
	m.StatusCode = statusCode
}

// LoadTestData loads test fixtures
func LoadTestData(filename string) ([]byte, error) {
	// In a real implementation, this would load from fixtures directory
	// For now, return mock data
	mockData := map[string]interface{}{
		"tenders": []map[string]interface{}{
			{
				"tender_id":     "TENDER-001",
				"nama_paket":    "Construction Project A",
				"nilai_pagu":    5000000000,
				"status_tender": "active",
			},
			{
				"tender_id":     "TENDER-002",
				"nama_paket":    "IT Infrastructure B",
				"nilai_pagu":    2000000000,
				"status_tender": "completed",
			},
		},
		"rup": []map[string]interface{}{
			{
				"kd_kro":         1001,
				"nama_kro":       "Procurement Plan 2025",
				"pagu_kro":       10000000000,
				"tahun_anggaran": 2025,
			},
		},
	}

	return json.Marshal(mockData)
}

// ValidateJSONSchema validates response against expected schema
func ValidateJSONSchema(t *testing.T, data interface{}, requiredFields []string) {
	dataMap, ok := data.(map[string]interface{})
	require.True(t, ok, "Data should be a map")

	for _, field := range requiredFields {
		_, exists := dataMap[field]
		require.True(t, exists, fmt.Sprintf("Field '%s' should exist", field))
	}
}

// RetryRequest retries a request with exponential backoff
func RetryRequest(client *TestClient, method, endpoint string, body interface{}, maxRetries int) (*http.Response, error) {
	var lastErr error
	backoff := 100 * time.Millisecond

	for i := 0; i < maxRetries; i++ {
		resp, err := client.DoRequest(method, endpoint, body)
		if err == nil && resp.StatusCode < 500 {
			return resp, nil
		}

		lastErr = err
		if resp != nil {
			resp.Body.Close()
		}

		time.Sleep(backoff)
		backoff *= 2
	}

	return nil, fmt.Errorf("failed after %d retries: %v", maxRetries, lastErr)
}