package v1

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"go-data-gateway/internal/datasource"
	"go.uber.org/zap"
)

// BatchRequest represents a batch query request
type BatchRequest struct {
	Queries []BatchQuery `json:"queries"`
	Options BatchOptions `json:"options,omitempty"`
}

// BatchQuery represents a single query in a batch
type BatchQuery struct {
	ID          string                    `json:"id"`
	Query       string                    `json:"query,omitempty"`
	DataSource  string                    `json:"data_source"`
	Table       string                    `json:"table,omitempty"`
	Options     *datasource.QueryOptions  `json:"options,omitempty"`
}

// BatchOptions controls batch execution behavior
type BatchOptions struct {
	MaxConcurrency int           `json:"max_concurrency,omitempty"`
	Timeout        time.Duration `json:"timeout,omitempty"`
	StopOnError    bool          `json:"stop_on_error,omitempty"`
}

// BatchResponse represents the response for batch queries
type BatchResponse struct {
	Results   []BatchResult `json:"results"`
	Summary   BatchSummary  `json:"summary"`
	Timestamp time.Time     `json:"timestamp"`
}

// BatchResult represents the result of a single query in batch
type BatchResult struct {
	ID        string                     `json:"id"`
	Status    string                     `json:"status"` // success, error, skipped
	Data      []map[string]interface{}   `json:"data,omitempty"`
	Error     string                     `json:"error,omitempty"`
	QueryTime time.Duration              `json:"query_time_ms"`
	RowCount  int                        `json:"row_count"`
	CacheHit  bool                       `json:"cache_hit"`
}

// BatchSummary provides aggregate metrics for the batch
type BatchSummary struct {
	TotalQueries     int           `json:"total_queries"`
	SuccessfulQueries int          `json:"successful_queries"`
	FailedQueries    int           `json:"failed_queries"`
	SkippedQueries   int           `json:"skipped_queries"`
	TotalTime        time.Duration `json:"total_time_ms"`
	CacheHits        int           `json:"cache_hits"`
}

// BatchHandler handles batch query requests
type BatchHandler struct {
	dataSources map[string]datasource.DataSource
	logger      *zap.Logger
}

// NewBatchHandler creates a new batch handler
func NewBatchHandler(dataSources map[string]datasource.DataSource, logger *zap.Logger) *BatchHandler {
	return &BatchHandler{
		dataSources: dataSources,
		logger:      logger,
	}
}

// Execute handles batch query execution
func (h *BatchHandler) Execute(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	startTime := time.Now()

	// Parse request
	var req BatchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Error("Failed to parse batch request", zap.Error(err))
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate request
	if len(req.Queries) == 0 {
		http.Error(w, "No queries provided", http.StatusBadRequest)
		return
	}

	if len(req.Queries) > 100 {
		http.Error(w, "Batch size exceeds maximum of 100 queries", http.StatusBadRequest)
		return
	}

	// Set defaults
	if req.Options.MaxConcurrency <= 0 {
		req.Options.MaxConcurrency = 5
	}
	if req.Options.MaxConcurrency > 20 {
		req.Options.MaxConcurrency = 20
	}
	if req.Options.Timeout <= 0 {
		req.Options.Timeout = 30 * time.Second
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(ctx, req.Options.Timeout)
	defer cancel()

	// Execute queries
	results := h.executeBatch(ctx, req)

	// Prepare response
	response := h.buildResponse(results, startTime)

	// Log batch summary
	h.logger.Info("Batch query completed",
		zap.Int("total_queries", response.Summary.TotalQueries),
		zap.Int("successful", response.Summary.SuccessfulQueries),
		zap.Int("failed", response.Summary.FailedQueries),
		zap.Duration("duration", response.Summary.TotalTime))

	// Send response
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// executeBatch executes queries with concurrency control
func (h *BatchHandler) executeBatch(ctx context.Context, req BatchRequest) []BatchResult {
	results := make([]BatchResult, len(req.Queries))
	semaphore := make(chan struct{}, req.Options.MaxConcurrency)
	var wg sync.WaitGroup
	var stopFlag int32

	for i, query := range req.Queries {
		// Check if we should stop on error
		if req.Options.StopOnError && stopFlag > 0 {
			results[i] = BatchResult{
				ID:     query.ID,
				Status: "skipped",
				Error:  "Skipped due to previous error",
			}
			continue
		}

		wg.Add(1)
		go func(idx int, q BatchQuery) {
			defer wg.Done()

			// Acquire semaphore
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			// Check context
			if ctx.Err() != nil {
				results[idx] = BatchResult{
					ID:     q.ID,
					Status: "error",
					Error:  "Context cancelled",
				}
				return
			}

			// Execute query
			result := h.executeQuery(ctx, q)
			results[idx] = result

			// Set stop flag if needed
			if req.Options.StopOnError && result.Status == "error" {
				stopFlag = 1
			}
		}(i, query)
	}

	wg.Wait()
	return results
}

// executeQuery executes a single query
func (h *BatchHandler) executeQuery(ctx context.Context, query BatchQuery) BatchResult {
	startTime := time.Now()
	result := BatchResult{
		ID: query.ID,
	}

	// Get data source
	dataSource, exists := h.dataSources[query.DataSource]
	if !exists {
		result.Status = "error"
		result.Error = fmt.Sprintf("Unknown data source: %s", query.DataSource)
		return result
	}

	// Execute query
	var queryResult *datasource.QueryResult
	var err error

	if query.Query != "" {
		// Direct SQL query
		queryResult, err = dataSource.ExecuteQuery(ctx, query.Query, query.Options)
	} else if query.Table != "" {
		// Table query
		queryResult, err = dataSource.GetData(ctx, query.Table, query.Options)
	} else {
		result.Status = "error"
		result.Error = "Either query or table must be specified"
		return result
	}

	// Handle result
	if err != nil {
		result.Status = "error"
		result.Error = err.Error()
		h.logger.Warn("Batch query failed",
			zap.String("id", query.ID),
			zap.Error(err))
	} else {
		result.Status = "success"
		result.Data = queryResult.Data
		result.RowCount = queryResult.Count
		result.CacheHit = queryResult.CacheHit
		h.logger.Debug("Batch query succeeded",
			zap.String("id", query.ID),
			zap.Int("rows", queryResult.Count),
			zap.Bool("cache_hit", queryResult.CacheHit))
	}

	result.QueryTime = time.Since(startTime)
	return result
}

// buildResponse builds the batch response with summary
func (h *BatchHandler) buildResponse(results []BatchResult, startTime time.Time) BatchResponse {
	response := BatchResponse{
		Results:   results,
		Timestamp: time.Now(),
		Summary: BatchSummary{
			TotalQueries: len(results),
			TotalTime:    time.Since(startTime),
		},
	}

	// Calculate summary statistics
	for _, result := range results {
		switch result.Status {
		case "success":
			response.Summary.SuccessfulQueries++
			if result.CacheHit {
				response.Summary.CacheHits++
			}
		case "error":
			response.Summary.FailedQueries++
		case "skipped":
			response.Summary.SkippedQueries++
		}
	}

	return response
}

// Stream handles streaming batch results
func (h *BatchHandler) Stream(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Set up SSE headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	// Parse request
	var req BatchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.sendSSEError(w, "Failed to parse request")
		return
	}

	// Create flusher
	flusher, ok := w.(http.Flusher)
	if !ok {
		h.sendSSEError(w, "Streaming not supported")
		return
	}

	// Send initial message
	h.sendSSEMessage(w, "start", map[string]interface{}{
		"total_queries": len(req.Queries),
		"timestamp":     time.Now(),
	})
	flusher.Flush()

	// Process queries one by one
	for i, query := range req.Queries {
		if ctx.Err() != nil {
			break
		}

		// Execute query
		result := h.executeQuery(ctx, query)

		// Send result
		h.sendSSEMessage(w, "result", map[string]interface{}{
			"index":  i,
			"result": result,
		})
		flusher.Flush()
	}

	// Send completion message
	h.sendSSEMessage(w, "complete", map[string]interface{}{
		"timestamp": time.Now(),
	})
	flusher.Flush()
}

// sendSSEMessage sends a Server-Sent Event message
func (h *BatchHandler) sendSSEMessage(w http.ResponseWriter, event string, data interface{}) {
	jsonData, _ := json.Marshal(data)
	fmt.Fprintf(w, "event: %s\n", event)
	fmt.Fprintf(w, "data: %s\n\n", jsonData)
}

// sendSSEError sends an SSE error message
func (h *BatchHandler) sendSSEError(w http.ResponseWriter, message string) {
	h.sendSSEMessage(w, "error", map[string]string{"error": message})
}