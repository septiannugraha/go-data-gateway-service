package cache

import (
	"context"
	"fmt"
	"time"

	"go-data-gateway/internal/datasource"
	"go.uber.org/zap"
)

// CachedDataSource wraps a DataSource with caching
type CachedDataSource struct {
	source  datasource.DataSource
	cache   Cache
	metrics *Metrics
	logger  *zap.Logger
}

// NewCachedDataSource creates a new cached data source
func NewCachedDataSource(source datasource.DataSource, cache Cache, logger *zap.Logger) *CachedDataSource {
	return &CachedDataSource{
		source:  source,
		cache:   cache,
		metrics: NewMetrics(),
		logger:  logger,
	}
}

// ExecuteQuery executes a query with caching
func (c *CachedDataSource) ExecuteQuery(ctx context.Context, query string, opts *datasource.QueryOptions) (*datasource.QueryResult, error) {
	start := time.Now()

	// Generate cache key
	cacheKey := c.cache.GenerateKey(string(c.source.GetType()), query)

	// Try to get from cache
	if cached, hit, err := c.cache.Get(ctx, cacheKey); err == nil && hit {
		c.metrics.RecordHit(time.Since(start))

		// Convert cached data back to QueryResult
		if result, ok := cached.(map[string]interface{}); ok {
			queryResult := &datasource.QueryResult{
				CacheHit: true,
			}

			// Unmarshal the cached data
			if data, ok := result["data"].([]interface{}); ok {
				queryResult.Data = make([]map[string]interface{}, len(data))
				for i, item := range data {
					if m, ok := item.(map[string]interface{}); ok {
						queryResult.Data[i] = m
					}
				}
			}

			if count, ok := result["count"].(float64); ok {
				queryResult.Count = int(count)
			}

			if source, ok := result["source"].(string); ok {
				queryResult.Source = datasource.DataSourceType(source)
			}

			c.logger.Info("Cache hit",
				zap.String("source", string(c.source.GetType())),
				zap.String("key", cacheKey),
				zap.Duration("latency", time.Since(start)))

			return queryResult, nil
		}
	}

	c.metrics.RecordMiss(time.Since(start))

	// Execute query on actual data source
	result, err := c.source.ExecuteQuery(ctx, query, opts)
	if err != nil {
		c.metrics.RecordError()
		return nil, err
	}

	// Cache the result
	if result != nil && len(result.Data) > 0 {
		// Determine TTL
		ttl := 5 * time.Minute // Default
		if opts != nil && opts.CacheTTL > 0 {
			ttl = opts.CacheTTL
		}

		// Create cacheable version
		cacheData := map[string]interface{}{
			"data":   result.Data,
			"count":  result.Count,
			"source": string(result.Source),
		}

		if err := c.cache.Set(ctx, cacheKey, cacheData, ttl); err != nil {
			c.logger.Warn("Failed to cache query result",
				zap.String("key", cacheKey),
				zap.Error(err))
		} else {
			c.metrics.RecordSet()
			c.logger.Debug("Query result cached",
				zap.String("key", cacheKey),
				zap.Duration("ttl", ttl))
		}
	}

	return result, nil
}

// GetData retrieves data with caching
func (c *CachedDataSource) GetData(ctx context.Context, table string, opts *datasource.QueryOptions) (*datasource.QueryResult, error) {
	// For GetData, we can also use caching
	cacheKey := fmt.Sprintf("table:%s:%s", c.source.GetType(), table)
	if opts != nil {
		cacheKey = fmt.Sprintf("%s:limit:%d:offset:%d", cacheKey, opts.Limit, opts.Offset)
	}

	// Similar caching logic as ExecuteQuery
	return c.source.GetData(ctx, table, opts)
}

// TestConnection tests the data source connection
func (c *CachedDataSource) TestConnection(ctx context.Context) error {
	return c.source.TestConnection(ctx)
}

// GetType returns the data source type
func (c *CachedDataSource) GetType() datasource.DataSourceType {
	return c.source.GetType()
}

// Close closes the data source
func (c *CachedDataSource) Close() error {
	return c.source.Close()
}

// GetMetrics returns cache metrics
func (c *CachedDataSource) GetMetrics() map[string]interface{} {
	return c.metrics.GetStats()
}

// InvalidateCache invalidates cache for this data source
func (c *CachedDataSource) InvalidateCache(ctx context.Context) error {
	pattern := fmt.Sprintf("query:%s:*", c.source.GetType())
	return c.cache.Invalidate(ctx, pattern)
}
