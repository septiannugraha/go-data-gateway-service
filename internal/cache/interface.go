package cache

import (
	"context"
	"time"
)

// Cache defines the interface for cache implementations
type Cache interface {
	// Get retrieves data from cache
	// Returns: data, hit/miss, error
	Get(ctx context.Context, key string) (interface{}, bool, error)

	// Set stores data in cache with TTL
	Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error

	// Delete removes a key from cache
	Delete(ctx context.Context, key string) error

	// Invalidate removes all keys matching a pattern
	Invalidate(ctx context.Context, pattern string) error

	// GenerateKey creates a cache key from query and source
	GenerateKey(source, query string) string

	// Stats returns cache statistics
	Stats(ctx context.Context) (map[string]interface{}, error)

	// Close closes any connections
	Close() error
}

// NoOpCache is a cache that does nothing (for when Redis is not available)
type NoOpCache struct{}

func (n *NoOpCache) Get(ctx context.Context, key string) (interface{}, bool, error) {
	return nil, false, nil
}

func (n *NoOpCache) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	return nil
}

func (n *NoOpCache) Delete(ctx context.Context, key string) error {
	return nil
}

func (n *NoOpCache) Invalidate(ctx context.Context, pattern string) error {
	return nil
}

func (n *NoOpCache) GenerateKey(source, query string) string {
	return ""
}

func (n *NoOpCache) Stats(ctx context.Context) (map[string]interface{}, error) {
	return map[string]interface{}{
		"connected": false,
		"type":      "noop",
	}, nil
}

func (n *NoOpCache) Close() error {
	return nil
}