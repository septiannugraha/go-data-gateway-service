package v1

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"go-data-gateway/internal/datasource"
	"go.uber.org/zap"
)

// StreamRequest represents a streaming query request
type StreamRequest struct {
	Query      string                   `json:"query,omitempty"`
	DataSource string                   `json:"data_source"`
	Table      string                   `json:"table,omitempty"`
	ChunkSize  int                      `json:"chunk_size,omitempty"`
	Format     string                   `json:"format,omitempty"` // json, ndjson, csv
	Options    *datasource.QueryOptions `json:"options,omitempty"`
}

// StreamHandler handles streaming responses for large datasets
type StreamHandler struct {
	dataSources map[string]datasource.DataSource
	logger      *zap.Logger
}

// NewStreamHandler creates a new stream handler
func NewStreamHandler(dataSources map[string]datasource.DataSource, logger *zap.Logger) *StreamHandler {
	return &StreamHandler{
		dataSources: dataSources,
		logger:      logger,
	}
}

// Stream handles streaming query execution
func (h *StreamHandler) Stream(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Parse request
	var req StreamRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Error("Failed to parse stream request", zap.Error(err))
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate and set defaults
	if req.ChunkSize <= 0 {
		req.ChunkSize = 1000
	}
	if req.ChunkSize > 10000 {
		req.ChunkSize = 10000
	}
	if req.Format == "" {
		req.Format = "ndjson"
	}

	// Get data source
	dataSource, exists := h.dataSources[req.DataSource]
	if !exists {
		http.Error(w, fmt.Sprintf("Unknown data source: %s", req.DataSource), http.StatusBadRequest)
		return
	}

	// Set appropriate headers based on format
	switch req.Format {
	case "json":
		w.Header().Set("Content-Type", "application/json")
	case "ndjson":
		w.Header().Set("Content-Type", "application/x-ndjson")
	case "csv":
		w.Header().Set("Content-Type", "text/csv")
	default:
		http.Error(w, "Unsupported format", http.StatusBadRequest)
		return
	}

	// Set streaming headers
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Transfer-Encoding", "chunked")
	w.Header().Set("X-Content-Type-Options", "nosniff")

	// Create flusher for streaming
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming not supported", http.StatusInternalServerError)
		return
	}

	// Stream data based on format
	switch req.Format {
	case "json":
		h.streamJSON(ctx, w, flusher, dataSource, req)
	case "ndjson":
		h.streamNDJSON(ctx, w, flusher, dataSource, req)
	case "csv":
		h.streamCSV(ctx, w, flusher, dataSource, req)
	}
}

// streamJSON streams data in JSON array format
func (h *StreamHandler) streamJSON(ctx context.Context, w io.Writer, flusher http.Flusher,
	dataSource datasource.DataSource, req StreamRequest) {

	// Write opening bracket
	w.Write([]byte("[\n"))
	flusher.Flush()

	offset := 0
	firstChunk := true
	totalRows := 0

	for {
		// Check context
		if ctx.Err() != nil {
			break
		}

		// Prepare query options with pagination
		opts := &datasource.QueryOptions{
			Limit:  req.ChunkSize,
			Offset: offset,
		}
		if req.Options != nil {
			opts.OrderBy = req.Options.OrderBy
			opts.OrderDir = req.Options.OrderDir
		}

		// Execute query for this chunk
		var result *datasource.QueryResult
		var err error

		if req.Query != "" {
			result, err = dataSource.ExecuteQuery(ctx, req.Query, opts)
		} else if req.Table != "" {
			result, err = dataSource.GetData(ctx, req.Table, opts)
		} else {
			break
		}

		if err != nil {
			h.logger.Error("Stream query failed", zap.Error(err))
			break
		}

		// Write results
		for i, row := range result.Data {
			if !firstChunk || i > 0 {
				w.Write([]byte(",\n"))
			}
			jsonData, _ := json.Marshal(row)
			w.Write([]byte("  "))
			w.Write(jsonData)
			firstChunk = false
			totalRows++
		}

		flusher.Flush()

		// Check if we got less than chunk size (end of data)
		if len(result.Data) < req.ChunkSize {
			break
		}

		offset += req.ChunkSize
	}

	// Write closing bracket
	w.Write([]byte("\n]"))
	flusher.Flush()

	h.logger.Info("JSON streaming completed",
		zap.Int("total_rows", totalRows),
		zap.String("data_source", req.DataSource))
}

// streamNDJSON streams data in newline-delimited JSON format
func (h *StreamHandler) streamNDJSON(ctx context.Context, w io.Writer, flusher http.Flusher,
	dataSource datasource.DataSource, req StreamRequest) {

	offset := 0
	totalRows := 0
	startTime := time.Now()

	for {
		// Check context
		if ctx.Err() != nil {
			break
		}

		// Prepare query options with pagination
		opts := &datasource.QueryOptions{
			Limit:  req.ChunkSize,
			Offset: offset,
		}
		if req.Options != nil {
			opts.OrderBy = req.Options.OrderBy
			opts.OrderDir = req.Options.OrderDir
		}

		// Execute query for this chunk
		var result *datasource.QueryResult
		var err error

		if req.Query != "" {
			result, err = dataSource.ExecuteQuery(ctx, req.Query, opts)
		} else if req.Table != "" {
			result, err = dataSource.GetData(ctx, req.Table, opts)
		} else {
			break
		}

		if err != nil {
			// Write error as NDJSON
			errorObj := map[string]string{
				"error": err.Error(),
				"type":  "error",
			}
			jsonData, _ := json.Marshal(errorObj)
			w.Write(jsonData)
			w.Write([]byte("\n"))
			flusher.Flush()
			break
		}

		// Write results
		for _, row := range result.Data {
			jsonData, _ := json.Marshal(row)
			w.Write(jsonData)
			w.Write([]byte("\n"))
			totalRows++

			// Flush every 100 rows for responsiveness
			if totalRows%100 == 0 {
				flusher.Flush()
			}
		}

		// Final flush for this chunk
		flusher.Flush()

		// Log progress
		h.logger.Debug("Streamed chunk",
			zap.Int("chunk_rows", len(result.Data)),
			zap.Int("total_rows", totalRows),
			zap.Duration("elapsed", time.Since(startTime)))

		// Check if we got less than chunk size (end of data)
		if len(result.Data) < req.ChunkSize {
			break
		}

		offset += req.ChunkSize
	}

	// Write summary as final NDJSON line
	summary := map[string]interface{}{
		"type":       "summary",
		"total_rows": totalRows,
		"duration":   time.Since(startTime).Milliseconds(),
		"timestamp":  time.Now(),
	}
	jsonData, _ := json.Marshal(summary)
	w.Write(jsonData)
	w.Write([]byte("\n"))
	flusher.Flush()

	h.logger.Info("NDJSON streaming completed",
		zap.Int("total_rows", totalRows),
		zap.Duration("duration", time.Since(startTime)),
		zap.String("data_source", req.DataSource))
}

// streamCSV streams data in CSV format
func (h *StreamHandler) streamCSV(ctx context.Context, w io.Writer, flusher http.Flusher,
	dataSource datasource.DataSource, req StreamRequest) {

	offset := 0
	totalRows := 0
	headerWritten := false

	for {
		// Check context
		if ctx.Err() != nil {
			break
		}

		// Prepare query options with pagination
		opts := &datasource.QueryOptions{
			Limit:  req.ChunkSize,
			Offset: offset,
		}
		if req.Options != nil {
			opts.OrderBy = req.Options.OrderBy
			opts.OrderDir = req.Options.OrderDir
		}

		// Execute query for this chunk
		var result *datasource.QueryResult
		var err error

		if req.Query != "" {
			result, err = dataSource.ExecuteQuery(ctx, req.Query, opts)
		} else if req.Table != "" {
			result, err = dataSource.GetData(ctx, req.Table, opts)
		} else {
			break
		}

		if err != nil {
			h.logger.Error("Stream query failed", zap.Error(err))
			break
		}

		// Write CSV
		if len(result.Data) > 0 {
			// Write header on first chunk
			if !headerWritten {
				headers := make([]string, 0)
				for key := range result.Data[0] {
					headers = append(headers, key)
				}
				h.writeCSVRow(w, headers)
				headerWritten = true
			}

			// Write data rows
			for _, row := range result.Data {
				values := make([]string, 0)
				for key := range result.Data[0] { // Use same key order as header
					value := ""
					if v, ok := row[key]; ok {
						value = fmt.Sprintf("%v", v)
					}
					values = append(values, value)
				}
				h.writeCSVRow(w, values)
				totalRows++
			}

			flusher.Flush()
		}

		// Check if we got less than chunk size (end of data)
		if len(result.Data) < req.ChunkSize {
			break
		}

		offset += req.ChunkSize
	}

	h.logger.Info("CSV streaming completed",
		zap.Int("total_rows", totalRows),
		zap.String("data_source", req.DataSource))
}

// writeCSVRow writes a CSV row
func (h *StreamHandler) writeCSVRow(w io.Writer, values []string) {
	for i, value := range values {
		if i > 0 {
			w.Write([]byte(","))
		}
		// Simple CSV escaping (should use encoding/csv for production)
		if needsQuoting(value) {
			w.Write([]byte(strconv.Quote(value)))
		} else {
			w.Write([]byte(value))
		}
	}
	w.Write([]byte("\n"))
}

// needsQuoting checks if a CSV value needs quoting
func needsQuoting(s string) bool {
	for _, c := range s {
		if c == ',' || c == '"' || c == '\n' || c == '\r' {
			return true
		}
	}
	return false
}

// StreamSSE handles Server-Sent Events streaming
func (h *StreamHandler) StreamSSE(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Set SSE headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	// Parse request
	var req StreamRequest
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

	// Set defaults
	if req.ChunkSize <= 0 {
		req.ChunkSize = 100
	}

	// Get data source
	dataSource, exists := h.dataSources[req.DataSource]
	if !exists {
		h.sendSSEError(w, fmt.Sprintf("Unknown data source: %s", req.DataSource))
		return
	}

	// Send initial event
	h.sendSSEEvent(w, "start", map[string]interface{}{
		"data_source": req.DataSource,
		"chunk_size":  req.ChunkSize,
		"timestamp":   time.Now(),
	})
	flusher.Flush()

	offset := 0
	totalRows := 0
	startTime := time.Now()

	for {
		// Check context
		if ctx.Err() != nil {
			h.sendSSEEvent(w, "abort", map[string]string{"reason": "Context cancelled"})
			flusher.Flush()
			break
		}

		// Prepare query options
		opts := &datasource.QueryOptions{
			Limit:  req.ChunkSize,
			Offset: offset,
		}

		// Execute query
		var result *datasource.QueryResult
		var err error

		if req.Query != "" {
			result, err = dataSource.ExecuteQuery(ctx, req.Query, opts)
		} else if req.Table != "" {
			result, err = dataSource.GetData(ctx, req.Table, opts)
		} else {
			break
		}

		if err != nil {
			h.sendSSEEvent(w, "error", map[string]string{"error": err.Error()})
			flusher.Flush()
			break
		}

		// Send data chunk
		if len(result.Data) > 0 {
			h.sendSSEEvent(w, "data", map[string]interface{}{
				"rows":       result.Data,
				"chunk_size": len(result.Data),
				"offset":     offset,
				"cache_hit":  result.CacheHit,
			})
			flusher.Flush()
			totalRows += len(result.Data)
		}

		// Send progress update
		h.sendSSEEvent(w, "progress", map[string]interface{}{
			"rows_processed": totalRows,
			"elapsed_ms":     time.Since(startTime).Milliseconds(),
		})
		flusher.Flush()

		// Check if done
		if len(result.Data) < req.ChunkSize {
			break
		}

		offset += req.ChunkSize
	}

	// Send completion event
	h.sendSSEEvent(w, "complete", map[string]interface{}{
		"total_rows": totalRows,
		"duration":   time.Since(startTime).Milliseconds(),
		"timestamp":  time.Now(),
	})
	flusher.Flush()

	h.logger.Info("SSE streaming completed",
		zap.Int("total_rows", totalRows),
		zap.Duration("duration", time.Since(startTime)))
}

// sendSSEEvent sends an SSE event
func (h *StreamHandler) sendSSEEvent(w io.Writer, event string, data interface{}) {
	jsonData, _ := json.Marshal(data)
	fmt.Fprintf(w, "event: %s\n", event)
	fmt.Fprintf(w, "data: %s\n\n", jsonData)
}

// sendSSEError sends an SSE error event
func (h *StreamHandler) sendSSEError(w io.Writer, message string) {
	h.sendSSEEvent(w, "error", map[string]string{"error": message})
}