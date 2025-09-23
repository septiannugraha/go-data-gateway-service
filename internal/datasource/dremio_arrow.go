package datasource

import (
	"context"
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"strings"
	"time"

	"github.com/apache/arrow-go/v18/arrow"
	"github.com/apache/arrow-go/v18/arrow/array"
	"github.com/apache/arrow-go/v18/arrow/flight"
	pb "github.com/apache/arrow-go/v18/arrow/flight/gen/flight"
	"github.com/apache/arrow-go/v18/arrow/memory"
	"github.com/patrickmn/go-cache"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
)

// DremioArrowClient implements DataSource using Arrow Flight SQL
type DremioArrowClient struct {
	client   flight.Client
	pool     *ArrowConnectionPool // Optional connection pool
	config   *DremioConfig
	logger   *zap.Logger
	cache    *cache.Cache
	memAlloc memory.Allocator
	ctx      context.Context
	usePool  bool
}

// DremioConfig holds Dremio connection configuration
type DremioConfig struct {
	Host     string
	Port     int // Arrow Flight port (31010)
	Username string
	Password string
	Token    string
	UseTLS   bool
	Project  string // Optional: default project/space in Dremio
}

// NewDremioArrowClientWithPool creates a new Arrow Flight SQL client with connection pooling
func NewDremioArrowClientWithPool(cfg *DremioConfig, poolConfig *PoolConfig, logger *zap.Logger) (*DremioArrowClient, error) {
	// Create connection pool
	pool, err := NewArrowConnectionPool(cfg, poolConfig, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create connection pool: %w", err)
	}

	client := &DremioArrowClient{
		pool:     pool,
		config:   cfg,
		logger:   logger,
		cache:    cache.New(5*time.Minute, 10*time.Minute),
		memAlloc: memory.NewGoAllocator(),
		ctx:      context.Background(),
		usePool:  true,
	}

	logger.Info("Dremio Arrow Flight client initialized with connection pool",
		zap.String("host", cfg.Host),
		zap.Int("port", cfg.Port),
		zap.Int("max_connections", poolConfig.MaxConnections))

	return client, nil
}

// NewDremioArrowClient creates a new Arrow Flight SQL client for Dremio (single connection)
func NewDremioArrowClient(cfg *DremioConfig, logger *zap.Logger) (*DremioArrowClient, error) {
	ctx := context.Background()

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

	// Create raw Flight client (like colleague's implementation)
	flightClient, err := flight.NewClientWithMiddleware(addr, nil, nil, dialOpts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create flight client: %w", err)
	}

	client := &DremioArrowClient{
		client:   flightClient,
		config:   cfg,
		logger:   logger,
		cache:    cache.New(5*time.Minute, 10*time.Minute),
		memAlloc: memory.NewGoAllocator(),
		ctx:      ctx,
	}

	// Set up authentication context if credentials provided
	if cfg.Username != "" && cfg.Password != "" {
		auth := base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%s:%s", cfg.Username, cfg.Password)))
		client.ctx = metadata.AppendToOutgoingContext(client.ctx, "authorization", "Basic "+auth)
		logger.Info("Authentication context set up", zap.String("user", cfg.Username))
	}

	logger.Info("Dremio Arrow Flight client initialized", zap.String("host", cfg.Host), zap.Int("port", cfg.Port))
	return client, nil
}

// basicAuth creates a basic auth string
func basicAuth(username, password string) string {
	auth := username + ":" + password
	return base64.StdEncoding.EncodeToString([]byte(auth))
}

// ExecuteQuery executes a SQL query using Arrow Flight
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
	d.logger.Info("Executing Arrow Flight query", zap.String("sql", query))

	// Create flight descriptor for SQL query (raw Flight protocol)
	desc := &pb.FlightDescriptor{
		Type: pb.FlightDescriptor_CMD,
		Cmd:  []byte(query),
	}

	var results []map[string]interface{}

	// Use connection pool if available
	if d.usePool && d.pool != nil {
		err := d.pool.WithConnection(ctx, func(client flight.Client) error {
			// Get flight info for the query
			info, err := client.GetFlightInfo(ctx, desc)
			if err != nil {
				return fmt.Errorf("failed to get flight info: %w", err)
			}

			// Check if we have endpoints
			if len(info.GetEndpoint()) == 0 {
				return fmt.Errorf("no endpoints returned")
			}

			// Fetch results from the first endpoint
			endpoint := info.GetEndpoint()[0]
			stream, err := client.DoGet(ctx, endpoint.GetTicket())
			if err != nil {
				return fmt.Errorf("failed to get data stream: %w", err)
			}

			// Create record reader from stream
			reader, err := flight.NewRecordReader(stream)
			if err != nil {
				return fmt.Errorf("failed to create record reader: %w", err)
			}
			defer reader.Release()

			// Convert Arrow records to map format
			for reader.Next() {
				record := reader.Record()
				if record != nil {
					results = append(results, d.recordToMaps(record)...)
					record.Release()
				}
			}

			if reader.Err() != nil {
				return fmt.Errorf("error reading results: %w", reader.Err())
			}

			return nil
		})

		if err != nil {
			return nil, err
		}
	} else {
		// Use single connection (original code)
		info, err := d.client.GetFlightInfo(d.ctx, desc)
		if err != nil {
			return nil, fmt.Errorf("failed to get flight info: %w", err)
		}

		// Check if we have endpoints
		if len(info.GetEndpoint()) == 0 {
			return nil, fmt.Errorf("no endpoints returned")
		}

		// Fetch results from the first endpoint
		endpoint := info.GetEndpoint()[0]
		stream, err := d.client.DoGet(d.ctx, endpoint.GetTicket())
		if err != nil {
			return nil, fmt.Errorf("failed to get data stream: %w", err)
		}

		// Create record reader from stream
		reader, err := flight.NewRecordReader(stream)
		if err != nil {
			return nil, fmt.Errorf("failed to create record reader: %w", err)
		}
		defer reader.Release()

		// Convert Arrow records to map format
		for reader.Next() {
			record := reader.Record()
			if record != nil {
				results = append(results, d.recordToMaps(record)...)
				record.Release()
			}
		}

		if reader.Err() != nil {
			return nil, fmt.Errorf("error reading results: %w", reader.Err())
		}
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

// recordToMaps converts Arrow Record to slice of maps
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

	// Sanitize inputs to prevent SQL injection
	sanitizer := NewSQLSanitizer()
	query, err := sanitizer.BuildSafeTableQuery(table, opts)
	if err != nil {
		return nil, fmt.Errorf("query validation failed: %w", err)
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

// Close closes the Arrow Flight client or connection pool
func (d *DremioArrowClient) Close() error {
	if d.usePool && d.pool != nil {
		return d.pool.Close()
	}
	if d.client != nil {
		return d.client.Close()
	}
	return nil
}

// GetPoolMetrics returns connection pool metrics (if using pool)
func (d *DremioArrowClient) GetPoolMetrics() map[string]interface{} {
	if d.usePool && d.pool != nil {
		return d.pool.GetMetrics()
	}
	return map[string]interface{}{
		"pool_enabled": false,
	}
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
