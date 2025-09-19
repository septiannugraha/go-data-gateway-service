package datasource

import (
	"context"
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"strings"
	"time"

	"github.com/apache/arrow/go/v13/arrow"
	"github.com/apache/arrow/go/v13/arrow/array"
	"github.com/apache/arrow/go/v13/arrow/flight/flightsql"
	"github.com/apache/arrow/go/v13/arrow/memory"
	"github.com/patrickmn/go-cache"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
)

// DremioArrowClient implements DataSource using Arrow Flight SQL
type DremioArrowClient struct {
	client   *flightsql.Client
	conn     *grpc.ClientConn
	config   *DremioConfig
	logger   *zap.Logger
	cache    *cache.Cache
	memAlloc memory.Allocator
}

// DremioConfig holds Dremio connection configuration
type DremioConfig struct {
	Host          string
	Port          int  // Arrow Flight port (31010)
	Username      string
	Password      string
	Token         string
	UseTLS        bool
	Project       string // Optional: default project/space in Dremio
}

// NewDremioArrowClient creates a new Arrow Flight SQL client for Dremio
func NewDremioArrowClient(cfg *DremioConfig, logger *zap.Logger) (*DremioArrowClient, error) {
	// Build connection address
	addr := fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)

	// Configure TLS/non-TLS connection
	var dialOpts []grpc.DialOption
	if cfg.UseTLS {
		tlsConfig := &tls.Config{InsecureSkipVerify: true} // Use proper cert validation in production
		dialOpts = append(dialOpts, grpc.WithTransportCredentials(credentials.NewTLS(tlsConfig)))
	} else {
		dialOpts = append(dialOpts, grpc.WithTransportCredentials(insecure.NewCredentials()))
	}

	// First create the gRPC connection
	conn, err := grpc.Dial(addr, dialOpts...)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to Dremio: %w", err)
	}

	// Create Arrow Flight SQL client with the connection
	flightClient, err := flightsql.NewClient(addr, nil, nil, dialOpts...)
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to create flight SQL client: %w", err)
	}

	client := &DremioArrowClient{
		client:   flightClient,
		conn:     conn,
		config:   cfg,
		logger:   logger,
		cache:    cache.New(5*time.Minute, 10*time.Minute),
		memAlloc: memory.NewGoAllocator(),
	}

	// Authenticate if credentials are provided
	if cfg.Username != "" && cfg.Password != "" {
		if err := client.authenticate(); err != nil {
			return nil, fmt.Errorf("authentication failed: %w", err)
		}
	}

	logger.Info("Dremio Arrow Flight SQL client initialized", zap.String("host", cfg.Host))
	return client, nil
}

// authenticate performs authentication with Dremio
func (d *DremioArrowClient) authenticate() error {
	ctx := context.Background()

	// Create authentication context with username/password
	ctx = d.getAuthContext(ctx)

	// Test authentication with a simple query
	_, err := d.client.Execute(ctx, "SELECT 1", nil)
	if err != nil {
		return fmt.Errorf("authentication handshake failed: %w", err)
	}

	d.logger.Info("Authentication successful")
	return nil
}

// basicAuth creates a basic auth string
func basicAuth(username, password string) string {
	auth := username + ":" + password
	return base64.StdEncoding.EncodeToString([]byte(auth))
}

// ExecuteQuery executes a SQL query using Arrow Flight SQL
func (d *DremioArrowClient) ExecuteQuery(ctx context.Context, query string, opts *QueryOptions) (*QueryResult, error) {
	// Validate query is read-only
	if !isReadOnlySQL(query) {
		return nil, fmt.Errorf("only SELECT queries are allowed")
	}

	// Check cache
	cacheKey := fmt.Sprintf("arrow:%s:%v", query, opts)
	if cached, found := d.cache.Get(cacheKey); found {
		d.logger.Debug("Cache hit", zap.String("query", query))
		result := cached.(*QueryResult)
		result.CacheHit = true
		return result, nil
	}

	start := time.Now()
	d.logger.Info("Executing Arrow Flight SQL query", zap.String("sql", query))

	// Add authentication to context
	ctx = d.getAuthContext(ctx)

	// Execute query
	info, err := d.client.Execute(ctx, query, nil)
	if err != nil {
		return nil, fmt.Errorf("query execution failed: %w", err)
	}

	// Get the endpoint containing the result
	if len(info.Endpoint) == 0 {
		return nil, fmt.Errorf("no endpoints returned")
	}

	// Fetch results from the first endpoint
	endpoint := info.Endpoint[0]
	reader, err := d.client.DoGet(ctx, endpoint.GetTicket())
	if err != nil {
		return nil, fmt.Errorf("failed to get results: %w", err)
	}
	defer reader.Release()

	// Convert Arrow records to map format
	var results []map[string]interface{}
	for reader.Next() {
		record := reader.Record()
		results = append(results, d.recordToMaps(record)...)
		record.Release()
	}

	if reader.Err() != nil {
		return nil, fmt.Errorf("error reading results: %w", reader.Err())
	}

	queryTime := time.Since(start)
	d.logger.Info("Query completed",
		zap.Duration("duration", queryTime),
		zap.Int("rows", len(results)))

	result := &QueryResult{
		Data:      results,
		Count:     len(results),
		Source:    DataSourceDremio,
		QueryTime: queryTime,
	}

	// Cache the results
	if opts != nil && opts.CacheTTL > 0 {
		d.cache.Set(cacheKey, result, opts.CacheTTL)
	} else {
		d.cache.Set(cacheKey, result, cache.DefaultExpiration)
	}

	return result, nil
}

// recordToMaps converts Arrow record to slice of maps
func (d *DremioArrowClient) recordToMaps(record arrow.Record) []map[string]interface{} {
	var results []map[string]interface{}
	numRows := int(record.NumRows())
	schema := record.Schema()

	for row := 0; row < numRows; row++ {
		rowMap := make(map[string]interface{})
		for col := 0; col < int(record.NumCols()); col++ {
			field := schema.Field(col)
			column := record.Column(col)
			rowMap[field.Name] = d.getValueAt(column, row)
		}
		results = append(results, rowMap)
	}

	return results
}

// getValueAt extracts value from Arrow column at specific row
func (d *DremioArrowClient) getValueAt(column arrow.Array, row int) interface{} {
	if column.IsNull(row) {
		return nil
	}

	switch col := column.(type) {
	case *array.Int64:
		return col.Value(row)
	case *array.Float64:
		return col.Value(row)
	case *array.String:
		return col.Value(row)
	case *array.Boolean:
		return col.Value(row)
	case *array.Date32:
		// Convert days since epoch to time
		days := col.Value(row)
		return time.Unix(int64(days)*86400, 0)
	case *array.Timestamp:
		return col.Value(row).ToTime(col.DataType().(*arrow.TimestampType).Unit)
	default:
		// Return string representation for other types
		return col.ValueStr(row)
	}
}

// GetData retrieves data from a specific table
func (d *DremioArrowClient) GetData(ctx context.Context, table string, opts *QueryOptions) (*QueryResult, error) {
	// Build query with optional project/space prefix
	if d.config.Project != "" && !strings.Contains(table, ".") {
		table = d.config.Project + "." + table
	}

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
	}

	return d.ExecuteQuery(ctx, query, opts)
}

// TestConnection tests the connection to Dremio
func (d *DremioArrowClient) TestConnection(ctx context.Context) error {
	_, err := d.ExecuteQuery(ctx, "SELECT 1", nil)
	return err
}

// GetType returns the data source type
func (d *DremioArrowClient) GetType() DataSourceType {
	return DataSourceDremio
}

// getAuthContext adds authentication headers to context
func (d *DremioArrowClient) getAuthContext(ctx context.Context) context.Context {
	if d.config.Username != "" && d.config.Password != "" {
		return metadata.AppendToOutgoingContext(ctx,
			"authorization", "Basic "+basicAuth(d.config.Username, d.config.Password),
		)
	} else if d.config.Token != "" {
		return metadata.AppendToOutgoingContext(ctx,
			"authorization", "Bearer "+d.config.Token,
		)
	}
	return ctx
}

// Close closes the Arrow Flight SQL client and connection
func (d *DremioArrowClient) Close() error {
	var err error
	if d.client != nil {
		err = d.client.Close()
	}
	if d.conn != nil {
		if connErr := d.conn.Close(); connErr != nil && err == nil {
			err = connErr
		}
	}
	return err
}

// isReadOnlySQL validates that a SQL query is read-only
func isReadOnlySQL(sql string) bool {
	sql = strings.ToUpper(strings.TrimSpace(sql))
	forbidden := []string{"INSERT", "UPDATE", "DELETE", "DROP", "CREATE", "ALTER", "TRUNCATE", "MERGE"}

	for _, keyword := range forbidden {
		if strings.Contains(sql, keyword) {
			return false
		}
	}

	return strings.HasPrefix(sql, "SELECT") || strings.HasPrefix(sql, "WITH")
}