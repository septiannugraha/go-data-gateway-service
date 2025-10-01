package cache

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	"go.uber.org/zap"

	"go-data-gateway/internal/datasource"
)

// CachedDataSource wraps a DataSource with caching capabilities
type CachedDataSource struct {
	source datasource.DataSource
	cache  Cache
	logger *zap.Logger
	ttl    time.Duration
}

// NewCachedDataSource creates a new cached data source
func NewCachedDataSource(source datasource.DataSource, cache Cache, logger *zap.Logger) *CachedDataSource {
	return &CachedDataSource{
		source: source,
		cache:  cache,
		logger: logger,
		ttl:    5 * time.Minute, // default TTL
	}
}

// NewCachedDataSourceWithTTL creates a new cached data source with custom TTL
func NewCachedDataSourceWithTTL(source datasource.DataSource, cache Cache, logger *zap.Logger, ttl time.Duration) *CachedDataSource {
	return &CachedDataSource{
		source: source,
		cache:  cache,
		logger: logger,
		ttl:    ttl,
	}
}

// GetType returns the data source type
func (c *CachedDataSource) GetType() datasource.DataSourceType {
	return c.source.GetType()
}

// generateCacheKey creates a cache key from query
func (c *CachedDataSource) generateCacheKey(query string) string {
	h := sha256.New()
	h.Write([]byte(fmt.Sprintf("%s:%s", c.source.GetType(), query)))
	return hex.EncodeToString(h.Sum(nil))
}

// ExecuteQuery executes a query with caching
func (c *CachedDataSource) ExecuteQuery(ctx context.Context, query string, opts *datasource.QueryOptions) (*datasource.QueryResult, error) {
	// Generate cache key
	cacheKey := c.generateCacheKey(query)

	// Try to get from cache
	var cachedResult datasource.QueryResult
	err := c.cache.Get(ctx, cacheKey, &cachedResult)
	if err == nil {
		c.logger.Debug("Cache hit",
			zap.String("source", string(c.source.GetType())),
			zap.String("key", cacheKey))
		cachedResult.CacheHit = true
		return &cachedResult, nil
	}

	// Cache miss - execute query
	c.logger.Debug("Cache miss",
		zap.String("source", string(c.source.GetType())),
		zap.String("key", cacheKey))

	result, err := c.source.ExecuteQuery(ctx, query, opts)
	if err != nil {
		return nil, err
	}

	// Store in cache (fire and forget)
	go func() {
		cacheCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()

		ttl := c.ttl
		if opts != nil && opts.CacheTTL > 0 {
			ttl = opts.CacheTTL
		}

		if cacheErr := c.cache.SetWithTTL(cacheCtx, cacheKey, result, ttl); cacheErr != nil {
			c.logger.Warn("Failed to cache query result",
				zap.String("key", cacheKey),
				zap.Error(cacheErr))
		}
	}()

	return result, nil
}

// GetData retrieves data with filters and pagination
func (c *CachedDataSource) GetData(ctx context.Context, table string, opts *datasource.QueryOptions) (*datasource.QueryResult, error) {
	// For GetData, we can cache based on table+opts
	cacheKey := c.generateCacheKey(fmt.Sprintf("table:%s:opts:%v", table, opts))

	var cachedResult datasource.QueryResult
	err := c.cache.Get(ctx, cacheKey, &cachedResult)
	if err == nil {
		cachedResult.CacheHit = true
		return &cachedResult, nil
	}

	result, err := c.source.GetData(ctx, table, opts)
	if err != nil {
		return nil, err
	}

	// Cache the result
	go func() {
		cacheCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()

		ttl := c.ttl
		if opts != nil && opts.CacheTTL > 0 {
			ttl = opts.CacheTTL
		}

		c.cache.SetWithTTL(cacheCtx, cacheKey, result, ttl)
	}()

	return result, nil
}

// TestConnection tests the connection
func (c *CachedDataSource) TestConnection(ctx context.Context) error {
	return c.source.TestConnection(ctx)
}

// Close closes the data source
func (c *CachedDataSource) Close() error {
	return c.source.Close()
}

// GetStats returns cache statistics
func (c *CachedDataSource) GetStats() interface{} {
	stats := make(map[string]interface{})

	// Get source stats if available
	if s, ok := c.source.(interface{ GetStats() interface{} }); ok {
		stats["source"] = s.GetStats()
	}

	// Get cache stats
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if cacheStats, err := c.cache.Stats(ctx); err == nil {
		stats["cache"] = cacheStats
	}

	return stats
}

// InvalidateCache removes a specific query from cache
func (c *CachedDataSource) InvalidateCache(ctx context.Context, query string) error {
	cacheKey := c.generateCacheKey(query)
	return c.cache.Delete(ctx, cacheKey)
}

// ClearCache clears all cached entries for this data source
func (c *CachedDataSource) ClearCache(ctx context.Context) error {
	// Note: This clears the entire cache, not just for this source
	// In production, you might want to implement prefix-based clearing
	return c.cache.Clear(ctx)
}

// GetMetrics returns metrics for monitoring
func (c *CachedDataSource) GetMetrics() map[string]interface{} {
	metrics := make(map[string]interface{})

	// Get cache stats
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if stats, err := c.cache.Stats(ctx); err == nil {
		metrics["cache"] = stats
	}

	// Get source type
	metrics["source_type"] = string(c.source.GetType())
	metrics["cache_ttl_seconds"] = c.ttl.Seconds()

	return metrics
}
