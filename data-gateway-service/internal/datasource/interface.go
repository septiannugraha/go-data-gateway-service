package datasource

import (
	"context"
	"time"
)

// DataSourceType represents the type of data source
type DataSourceType string

const (
	DataSourceDremio   DataSourceType = "DATAWAREHOUSE"
	DataSourceBigQuery DataSourceType = "BIGQUERY"
	DataSourceMySQL    DataSourceType = "MYSQL"
	DataSourcePostgres DataSourceType = "POSTGRES"
)

// QueryResult represents the result of a query
type QueryResult struct {
	Data      []map[string]interface{} `json:"data"`
	Count     int                      `json:"count"`
	Source    DataSourceType           `json:"source"`
	CacheHit  bool                     `json:"cache_hit,omitempty"`
	QueryTime time.Duration            `json:"query_time_ms,omitempty"`
	Metadata  map[string]interface{}   `json:"metadata,omitempty"`
}

// QueryOptions represents options for query execution
type QueryOptions struct {
	Limit      int
	Offset     int
	OrderBy    string
	OrderDir   string
	Filters    map[string]interface{}
	CacheTTL   time.Duration
	Timeout    time.Duration
	Parameters []interface{}
}

// DataSource defines the interface for all data sources
type DataSource interface {
	// ExecuteQuery executes a raw SQL query
	ExecuteQuery(ctx context.Context, query string, opts *QueryOptions) (*QueryResult, error)

	// GetData retrieves data with filters and pagination
	GetData(ctx context.Context, table string, opts *QueryOptions) (*QueryResult, error)

	// TestConnection verifies the data source connection
	TestConnection(ctx context.Context) error

	// GetType returns the data source type
	GetType() DataSourceType

	// Close closes any open connections
	Close() error
}

// Factory creates data sources based on type
type Factory interface {
	Create(sourceType DataSourceType) (DataSource, error)
}
