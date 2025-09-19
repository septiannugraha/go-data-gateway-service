package datasource

import (
	"context"
	"fmt"
	"time"

	"go.uber.org/zap"
	"go-data-gateway/internal/clients"
	"go-data-gateway/internal/config"
)

// DremioRESTWrapper wraps the original DremioClient to implement DataSource interface
type DremioRESTWrapper struct {
	client *clients.DremioClient
	logger *zap.Logger
}

// NewDremioRESTClient creates a new Dremio REST client that implements DataSource
func NewDremioRESTClient(host string, port int, username, password string, logger *zap.Logger) (DataSource, error) {
	// Create config for the original client
	cfg := config.DremioConfig{
		Host:     host,
		Port:     port,
		Username: username,
		Password: password,
	}

	// Create the original client
	dremioClient, err := clients.NewDremioClient(cfg, logger)
	if err != nil {
		return nil, err
	}

	return &DremioRESTWrapper{
		client: dremioClient,
		logger: logger,
	}, nil
}

// ExecuteQuery executes a SQL query
func (d *DremioRESTWrapper) ExecuteQuery(ctx context.Context, query string, opts *QueryOptions) (*QueryResult, error) {
	// Call the original client's ExecuteQuery with context
	result, err := d.client.ExecuteQuery(ctx, query)
	if err != nil {
		return nil, err
	}

	// Type assert the result to access fields
	resultMap, ok := result.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("unexpected result type from ExecuteQuery")
	}

	// Extract data from the result
	data, ok := resultMap["data"].([]map[string]interface{})
	if !ok {
		// Try to handle empty result
		data = []map[string]interface{}{}
	}

	// Convert to our QueryResult format
	return &QueryResult{
		Data:      data,
		Count:     len(data),
		Source:    DataSourceDremio,
		QueryTime: time.Second, // This is approximate - we don't have exact timing
		CacheHit:  false,
	}, nil
}

// GetData retrieves data from a specific table
func (d *DremioRESTWrapper) GetData(ctx context.Context, table string, opts *QueryOptions) (*QueryResult, error) {
	query := fmt.Sprintf("SELECT * FROM %s", table)

	if opts != nil {
		if opts.OrderBy != "" {
			query += fmt.Sprintf(" ORDER BY %s %s", opts.OrderBy, opts.OrderDir)
		}
		if opts.Limit > 0 {
			query += fmt.Sprintf(" LIMIT %d", opts.Limit)
			if opts.Offset > 0 {
				query += fmt.Sprintf(" OFFSET %d", opts.Offset)
			}
		}
	} else {
		query += " LIMIT 100"
	}

	return d.ExecuteQuery(ctx, query, opts)
}

// TestConnection tests the connection to Dremio
func (d *DremioRESTWrapper) TestConnection(ctx context.Context) error {
	_, err := d.ExecuteQuery(ctx, "SELECT 1", nil)
	return err
}

// GetType returns the data source type
func (d *DremioRESTWrapper) GetType() DataSourceType {
	return DataSourceDremio
}

// Close closes the Dremio client
func (d *DremioRESTWrapper) Close() error {
	// HTTP client doesn't need explicit closing
	return nil
}