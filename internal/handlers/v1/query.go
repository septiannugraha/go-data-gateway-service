package v1

import (
	"encoding/json"
	"net/http"
	"time"

	"go.uber.org/zap"

	"go-data-gateway/internal/datasource"
	"go-data-gateway/internal/response"
)

// QueryHandler handles query requests with multiple data sources
type QueryHandler struct {
	dataSources map[string]datasource.DataSource
	logger      *zap.Logger
}

// NewQueryHandler creates a new query handler
func NewQueryHandler(dataSources map[string]datasource.DataSource, logger *zap.Logger) *QueryHandler {
	return &QueryHandler{
		dataSources: dataSources,
		logger:      logger,
	}
}

// QueryRequest represents a query request
type QueryRequest struct {
	SQL    string                    `json:"sql" binding:"required"`
	Source datasource.DataSourceType `json:"source" binding:"required"`
}

// Execute handles query execution requests
func (h *QueryHandler) Execute(w http.ResponseWriter, r *http.Request) {
	var req QueryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	h.logger.Info("Executing query",
		zap.String("source", string(req.Source)),
		zap.String("sql", req.SQL))

	// Find the appropriate data source
	var source datasource.DataSource
	for _, ds := range h.dataSources {
		if ds.GetType() == req.Source {
			source = ds
			break
		}
	}

	if source == nil {
		response.Error(w, "Data source not available: "+string(req.Source), http.StatusServiceUnavailable)
		return
	}

	// Execute query with timeout
	opts := &datasource.QueryOptions{
		Timeout:  30 * time.Second,
		CacheTTL: 5 * time.Minute,
	}

	result, err := source.ExecuteQuery(r.Context(), req.SQL, opts)
	if err != nil {
		h.logger.Error("Query execution failed",
			zap.String("source", string(req.Source)),
			zap.Error(err))
		response.ErrorWithDetails(w, "Query execution failed", err.Error(), http.StatusInternalServerError)
		return
	}

	// Send successful response
	response.Success(w, result, nil)
}
