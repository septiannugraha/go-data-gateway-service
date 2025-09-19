package clients

import (
	"context"
	"fmt"
	"strings"
	"time"

	"cloud.google.com/go/bigquery"
	"github.com/patrickmn/go-cache"
	"go.uber.org/zap"
	"google.golang.org/api/iterator"

	"go-data-gateway/internal/config"
)

// BigQueryClient handles connections to Google BigQuery
type BigQueryClient struct {
	client *bigquery.Client
	config config.BigQueryConfig
	cache  *cache.Cache
	logger *zap.Logger
}

// NewBigQueryClient creates a new BigQuery client
func NewBigQueryClient(cfg config.BigQueryConfig, logger *zap.Logger) (*BigQueryClient, error) {
	ctx := context.Background()

	// Create BigQuery client
	client, err := bigquery.NewClient(ctx, cfg.ProjectID)
	if err != nil {
		return nil, fmt.Errorf("failed to create BigQuery client: %w", err)
	}

	return &BigQueryClient{
		client: client,
		config: cfg,
		cache:  cache.New(5*time.Minute, 10*time.Minute),
		logger: logger,
	}, nil
}

// Query executes a SQL query against BigQuery
func (c *BigQueryClient) Query(ctx context.Context, sqlQuery string) ([]map[string]interface{}, error) {
	// Check cache first
	cacheKey := fmt.Sprintf("bigquery:%s", sqlQuery)
	if cached, found := c.cache.Get(cacheKey); found {
		c.logger.Debug("Cache hit", zap.String("query", sqlQuery))
		return cached.([]map[string]interface{}), nil
	}

	c.logger.Info("Executing BigQuery",
		zap.String("sql", sqlQuery),
		zap.String("project", c.config.ProjectID))

	start := time.Now()

	// Create query
	q := c.client.Query(sqlQuery)
	q.DefaultDatasetID = c.config.DatasetID

	// Run query
	it, err := q.Read(ctx)
	if err != nil {
		c.logger.Error("Query execution failed", zap.Error(err))
		return nil, fmt.Errorf("query execution failed: %w", err)
	}

	// Collect results
	var results []map[string]interface{}

	for {
		var row map[string]bigquery.Value
		err := it.Next(&row)
		if err == iterator.Done {
			break
		}
		if err != nil {
			c.logger.Error("Error reading row", zap.Error(err))
			return nil, fmt.Errorf("error reading row: %w", err)
		}

		// Convert BigQuery values to standard map
		result := make(map[string]interface{})
		for k, v := range row {
			result[k] = convertBigQueryValue(v)
		}
		results = append(results, result)
	}

	// Log performance metrics
	c.logger.Info("BigQuery completed",
		zap.Duration("duration", time.Since(start)),
		zap.Int("rows", len(results)),
		zap.Uint64("total_rows", it.TotalRows))

	// Cache results
	c.cache.Set(cacheKey, results, cache.DefaultExpiration)

	return results, nil
}

// ExecuteQuery provides a simpler interface for executing queries
func (c *BigQueryClient) ExecuteQuery(ctx context.Context, query string) (interface{}, error) {
	// Validate query is read-only
	if !isReadOnlySQL(query) {
		return nil, fmt.Errorf("only SELECT queries are allowed")
	}

	results, err := c.Query(ctx, query)
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"data":   results,
		"count":  len(results),
		"source": "bigquery",
	}, nil
}

// QueryWithParams executes a parameterized query
func (c *BigQueryClient) QueryWithParams(ctx context.Context, sqlQuery string, params map[string]interface{}) ([]map[string]interface{}, error) {
	q := c.client.Query(sqlQuery)
	q.DefaultDatasetID = c.config.DatasetID

	// Add parameters
	for key, value := range params {
		q.Parameters = append(q.Parameters, bigquery.QueryParameter{
			Name:  key,
			Value: value,
		})
	}

	it, err := q.Read(ctx)
	if err != nil {
		return nil, err
	}

	var results []map[string]interface{}
	for {
		var row map[string]bigquery.Value
		err := it.Next(&row)
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}

		result := make(map[string]interface{})
		for k, v := range row {
			result[k] = convertBigQueryValue(v)
		}
		results = append(results, result)
	}

	return results, nil
}

// TestConnection verifies the BigQuery connection
func (c *BigQueryClient) TestConnection(ctx context.Context) error {
	query := c.client.Query("SELECT 1 as test")
	_, err := query.Read(ctx)
	return err
}

// Close closes the BigQuery client
func (c *BigQueryClient) Close() error {
	return c.client.Close()
}

// convertBigQueryValue converts BigQuery values to standard Go types
func convertBigQueryValue(v bigquery.Value) interface{} {
	switch val := v.(type) {
	case []bigquery.Value:
		// Handle arrays
		result := make([]interface{}, len(val))
		for i, item := range val {
			result[i] = convertBigQueryValue(item)
		}
		return result
	case map[string]bigquery.Value:
		// Handle structs
		result := make(map[string]interface{})
		for k, item := range val {
			result[k] = convertBigQueryValue(item)
		}
		return result
	default:
		// Return primitive types as-is
		return val
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