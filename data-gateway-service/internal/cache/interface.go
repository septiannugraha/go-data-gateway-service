package cache

import (
	"context"
	"fmt"
	"time"
)

// Cache defines the interface for cache implementations
type Cache interface {
	Get(ctx context.Context, key string, value interface{}) error
	Set(ctx context.Context, key string, value interface{}) error
	SetWithTTL(ctx context.Context, key string, value interface{}, ttl time.Duration) error
	Delete(ctx context.Context, key string) error
	Exists(ctx context.Context, key string) (bool, error)
	Clear(ctx context.Context) error
	Close() error
	Stats(ctx context.Context) (map[string]interface{}, error)
}

// NoOpCache is a cache that doesn't cache anything
type NoOpCache struct{}

func NewNoOpCache() *NoOpCache {
	return &NoOpCache{}
}

func (n *NoOpCache) Get(ctx context.Context, key string, value interface{}) error {
	return ErrKeyNotFound
}

func (n *NoOpCache) Set(ctx context.Context, key string, value interface{}) error {
	return nil
}

func (n *NoOpCache) SetWithTTL(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	return nil
}

func (n *NoOpCache) Delete(ctx context.Context, key string) error {
	return nil
}

func (n *NoOpCache) Exists(ctx context.Context, key string) (bool, error) {
	return false, nil
}

func (n *NoOpCache) Clear(ctx context.Context) error {
	return nil
}

func (n *NoOpCache) Close() error {
	return nil
}

func (n *NoOpCache) Stats(ctx context.Context) (map[string]interface{}, error) {
	return map[string]interface{}{
		"type": "noop",
	}, nil
}

// Errors
var (
	ErrKeyNotFound = fmt.Errorf("key not found")
)