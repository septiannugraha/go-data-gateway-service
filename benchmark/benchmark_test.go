package benchmark

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"go.uber.org/zap"

	"go-data-gateway/internal/cache"
	"go-data-gateway/internal/datasource"
	v1 "go-data-gateway/internal/handlers/v1"
	custommw "go-data-gateway/internal/middleware/chi"
)

// BenchmarkConfig holds benchmark configuration
type BenchmarkConfig struct {
	UseCache       bool
	UsePool        bool
	DataSourceType string // "dremio" or "bigquery"
}

// setupTestServer creates a test server for benchmarking
func setupTestServer(b *testing.B, cfg BenchmarkConfig) *httptest.Server {
	// Create logger
	logger, _ := zap.NewDevelopment()

	// Create mock data sources
	dataSources := make(map[string]datasource.DataSource)

	// Add mock Dremio source
	dataSources["dremio"] = &MockDataSource{
		sourceType: datasource.DataSourceDremio,
		delay:      50 * time.Millisecond, // Simulate network delay
	}

	// Add mock BigQuery source
	dataSources["BIGQUERY"] = &MockDataSource{
		sourceType: datasource.DataSourceBigQuery,
		delay:      100 * time.Millisecond, // BigQuery typically slower
	}

	// Wrap with cache if enabled
	if cfg.UseCache {
		cacheService := &cache.NoOpCache{} // Use real Redis in production benchmarks
		for key, source := range dataSources {
			dataSources[key] = cache.NewCachedDataSource(source, cacheService, logger)
		}
	}

	// Create router
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(custommw.Logger(logger))
	r.Use(middleware.Recoverer)

	// Create handlers
	queryHandler := v1.NewQueryHandler(dataSources, logger)
	batchHandler := v1.NewBatchHandler(dataSources, logger)
	streamHandler := v1.NewStreamHandler(dataSources, logger)

	// Register routes
	r.Post("/api/v1/query", queryHandler.Execute)
	r.Post("/api/v1/batch", batchHandler.Execute)
	r.Post("/api/v1/stream", streamHandler.Stream)

	return httptest.NewServer(r)
}

// MockDataSource implements a mock data source for benchmarking
type MockDataSource struct {
	sourceType datasource.DataSourceType
	delay      time.Duration
}

func (m *MockDataSource) ExecuteQuery(ctx context.Context, query string, opts *datasource.QueryOptions) (*datasource.QueryResult, error) {
	// Simulate processing delay
	time.Sleep(m.delay)

	// Generate mock data
	data := make([]map[string]interface{}, 0)
	limit := 100
	if opts != nil && opts.Limit > 0 {
		limit = opts.Limit
	}

	for i := 0; i < limit; i++ {
		data = append(data, map[string]interface{}{
			"id":         fmt.Sprintf("ID%d", i+1),
			"name":       fmt.Sprintf("Item %d", i+1),
			"value":      float64(i * 100),
			"created_at": time.Now().Add(-time.Duration(i) * time.Hour),
		})
	}

	return &datasource.QueryResult{
		Data:      data,
		Count:     len(data),
		Source:    m.sourceType,
		QueryTime: m.delay,
	}, nil
}

func (m *MockDataSource) GetData(ctx context.Context, table string, opts *datasource.QueryOptions) (*datasource.QueryResult, error) {
	return m.ExecuteQuery(ctx, fmt.Sprintf("SELECT * FROM %s", table), opts)
}

func (m *MockDataSource) TestConnection(ctx context.Context) error {
	return nil
}

func (m *MockDataSource) GetType() datasource.DataSourceType {
	return m.sourceType
}

func (m *MockDataSource) Close() error {
	return nil
}

// BenchmarkSimpleQuery benchmarks single query execution
func BenchmarkSimpleQuery(b *testing.B) {
	configs := []struct {
		name string
		cfg  BenchmarkConfig
	}{
		{"NoCache", BenchmarkConfig{UseCache: false}},
		{"WithCache", BenchmarkConfig{UseCache: true}},
	}

	for _, tc := range configs {
		b.Run(tc.name, func(b *testing.B) {
			server := setupTestServer(b, tc.cfg)
			defer server.Close()

			reqBody := map[string]interface{}{
				"query":       "SELECT * FROM test_table LIMIT 100",
				"data_source": "dremio",
			}
			jsonBody, _ := json.Marshal(reqBody)

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				resp, err := http.Post(
					server.URL+"/api/v1/query",
					"application/json",
					bytes.NewBuffer(jsonBody),
				)
				if err != nil {
					b.Fatal(err)
				}
				resp.Body.Close()
			}
		})
	}
}

// BenchmarkBatchQuery benchmarks batch query execution
func BenchmarkBatchQuery(b *testing.B) {
	batchSizes := []int{1, 5, 10, 20}

	for _, size := range batchSizes {
		b.Run(fmt.Sprintf("BatchSize%d", size), func(b *testing.B) {
			server := setupTestServer(b, BenchmarkConfig{UseCache: true})
			defer server.Close()

			// Create batch request
			queries := make([]map[string]interface{}, size)
			for i := 0; i < size; i++ {
				queries[i] = map[string]interface{}{
					"id":          fmt.Sprintf("query-%d", i),
					"query":       fmt.Sprintf("SELECT * FROM table_%d", i),
					"data_source": "dremio",
				}
			}

			reqBody := map[string]interface{}{
				"queries": queries,
				"options": map[string]interface{}{
					"max_concurrency": 5,
				},
			}
			jsonBody, _ := json.Marshal(reqBody)

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				resp, err := http.Post(
					server.URL+"/api/v1/batch",
					"application/json",
					bytes.NewBuffer(jsonBody),
				)
				if err != nil {
					b.Fatal(err)
				}
				resp.Body.Close()
			}
		})
	}
}

// BenchmarkStreaming benchmarks streaming responses
func BenchmarkStreaming(b *testing.B) {
	chunkSizes := []int{100, 500, 1000}

	for _, chunkSize := range chunkSizes {
		b.Run(fmt.Sprintf("ChunkSize%d", chunkSize), func(b *testing.B) {
			server := setupTestServer(b, BenchmarkConfig{UseCache: false})
			defer server.Close()

			reqBody := map[string]interface{}{
				"table":       "large_table",
				"data_source": "dremio",
				"chunk_size":  chunkSize,
				"format":      "ndjson",
			}
			jsonBody, _ := json.Marshal(reqBody)

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				resp, err := http.Post(
					server.URL+"/api/v1/stream",
					"application/json",
					bytes.NewBuffer(jsonBody),
				)
				if err != nil {
					b.Fatal(err)
				}

				// Read and discard stream
				buf := make([]byte, 4096)
				for {
					_, err := resp.Body.Read(buf)
					if err != nil {
						break
					}
				}
				resp.Body.Close()
			}
		})
	}
}

// BenchmarkConcurrentRequests benchmarks concurrent request handling
func BenchmarkConcurrentRequests(b *testing.B) {
	concurrencyLevels := []int{1, 10, 50, 100}

	for _, level := range concurrencyLevels {
		b.Run(fmt.Sprintf("Concurrent%d", level), func(b *testing.B) {
			server := setupTestServer(b, BenchmarkConfig{UseCache: true})
			defer server.Close()

			reqBody := map[string]interface{}{
				"query":       "SELECT * FROM test_table",
				"data_source": "dremio",
			}
			jsonBody, _ := json.Marshal(reqBody)

			b.ResetTimer()
			b.RunParallel(func(pb *testing.PB) {
				for pb.Next() {
					resp, err := http.Post(
						server.URL+"/api/v1/query",
						"application/json",
						bytes.NewBuffer(jsonBody),
					)
					if err != nil {
						b.Fatal(err)
					}
					resp.Body.Close()
				}
			})
		})
	}
}

// BenchmarkMemoryAllocation benchmarks memory allocation patterns
func BenchmarkMemoryAllocation(b *testing.B) {
	b.Run("QueryAllocation", func(b *testing.B) {
		ctx := context.Background()
		mockDS := &MockDataSource{
			sourceType: datasource.DataSourceDremio,
			delay:      0, // No delay for allocation testing
		}

		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			result, _ := mockDS.ExecuteQuery(ctx, "SELECT * FROM test", &datasource.QueryOptions{
				Limit: 1000,
			})
			_ = result
		}
	})
}

// BenchmarkCacheHitRate measures cache effectiveness
func BenchmarkCacheHitRate(b *testing.B) {
	// This would use real Redis in production
	cacheService := &cache.NoOpCache{}
	logger, _ := zap.NewDevelopment()

	mockDS := &MockDataSource{
		sourceType: datasource.DataSourceDremio,
		delay:      50 * time.Millisecond,
	}

	cachedDS := cache.NewCachedDataSource(mockDS, cacheService, logger)
	ctx := context.Background()

	// Warm up cache
	query := "SELECT * FROM cached_table"
	opts := &datasource.QueryOptions{Limit: 100}
	cachedDS.ExecuteQuery(ctx, query, opts)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		result, _ := cachedDS.ExecuteQuery(ctx, query, opts)
		if !result.CacheHit && i > 0 {
			b.Error("Expected cache hit")
		}
	}
}

// TestMain sets up environment for benchmarks
func TestMain(m *testing.M) {
	// Set up any required environment variables
	os.Setenv("ENV", "test")
	os.Setenv("PORT", "8080")

	// Run benchmarks
	os.Exit(m.Run())
}

// Additional benchmark helpers

// measureLatency measures p50, p95, p99 latencies
func measureLatency(b *testing.B, endpoint string, reqBody []byte) {
	server := setupTestServer(b, BenchmarkConfig{UseCache: true})
	defer server.Close()

	latencies := make([]time.Duration, 0, 1000)

	for i := 0; i < 1000; i++ {
		start := time.Now()
		resp, err := http.Post(
			server.URL+endpoint,
			"application/json",
			bytes.NewBuffer(reqBody),
		)
		if err != nil {
			b.Fatal(err)
		}
		resp.Body.Close()
		latencies = append(latencies, time.Since(start))
	}

	// Calculate percentiles (simplified - use proper stats library in production)
	p50 := latencies[len(latencies)*50/100]
	p95 := latencies[len(latencies)*95/100]
	p99 := latencies[len(latencies)*99/100]

	b.Logf("Latencies - p50: %v, p95: %v, p99: %v", p50, p95, p99)
}