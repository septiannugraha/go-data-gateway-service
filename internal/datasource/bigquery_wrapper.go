package datasource

import (
	"context"
	"fmt"
	"time"

	"go-data-gateway/internal/clients"
	"go-data-gateway/internal/config"
	"go.uber.org/zap"
)

// BigQueryWrapper wraps the BigQueryClient to implement DataSource interface
type BigQueryWrapper struct {
	client *clients.BigQueryClient
	logger *zap.Logger
}

// NewBigQueryWrapper creates a new BigQuery wrapper that implements DataSource
func NewBigQueryWrapper(cfg config.BigQueryConfig, logger *zap.Logger) (*BigQueryWrapper, error) {
	client, err := clients.NewBigQueryClient(cfg, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create BigQuery client: %w", err)
	}

	return &BigQueryWrapper{
		client: client,
		logger: logger,
	}, nil
}

// ExecuteQuery executes a SQL query (implements DataSource interface)
func (w *BigQueryWrapper) ExecuteQuery(ctx context.Context, query string, opts *QueryOptions) (*QueryResult, error) {
	start := time.Now()

	// Call the underlying BigQuery client
	results, err := w.client.ExecuteQuery(ctx, query)
	if err != nil {
		return nil, err
	}

	// Convert results to proper format
	var data []map[string]interface{}

	// Check if results is already []map[string]interface{}
	if resultData, ok := results.([]map[string]interface{}); ok {
		data = resultData
	} else {
		// Try to extract data from a map structure
		if resultMap, ok := results.(map[string]interface{}); ok {
			if mapData, ok := resultMap["data"].([]map[string]interface{}); ok {
				data = mapData
			} else {
				return nil, fmt.Errorf("unexpected result structure from BigQuery")
			}
		} else {
			return nil, fmt.Errorf("unexpected result type from BigQuery: %T", results)
		}
	}

	return &QueryResult{
		Data:      data,
		Count:     len(data),
		Source:    DataSourceBigQuery,
		QueryTime: time.Since(start),
		CacheHit:  false,
	}, nil
}

// GetData retrieves data with filters and pagination
func (w *BigQueryWrapper) GetData(ctx context.Context, table string, opts *QueryOptions) (*QueryResult, error) {
	// Build query with LIMIT for cost safety
	query := fmt.Sprintf("SELECT * FROM `%s`", table)

	if opts != nil {
		if opts.Limit > 0 {
			query += fmt.Sprintf(" LIMIT %d", opts.Limit)
		} else {
			// Default limit for safety
			query += " LIMIT 100"
		}

		if opts.Offset > 0 {
			query += fmt.Sprintf(" OFFSET %d", opts.Offset)
		}
	} else {
		// Default limit for safety
		query += " LIMIT 100"
	}

	return w.ExecuteQuery(ctx, query, opts)
}

// TestConnection tests the BigQuery connection
func (w *BigQueryWrapper) TestConnection(ctx context.Context) error {
	return w.client.TestConnection(ctx)
}

// GetType returns the data source type
func (w *BigQueryWrapper) GetType() DataSourceType {
	return DataSourceBigQuery
}

// Close closes the BigQuery client
func (w *BigQueryWrapper) Close() error {
	return w.client.Close()
}
