package cache

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"go-data-gateway/internal/config"
)

// RedisCache implements caching using Redis
type RedisCache struct {
	client *redis.Client
	logger *zap.Logger
	ttl    time.Duration
}

// NewRedisCache creates a new Redis cache instance
func NewRedisCache(cfg config.RedisConfig, logger *zap.Logger) (*RedisCache, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%d", cfg.Host, cfg.Port),
		Password: cfg.Password,
		DB:       cfg.DB,
	})

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		logger.Warn("Redis connection failed, caching disabled", zap.Error(err))
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	logger.Info("Redis cache initialized",
		zap.String("host", cfg.Host),
		zap.Int("port", cfg.Port))

	return &RedisCache{
		client: client,
		logger: logger,
		ttl:    5 * time.Minute, // Default TTL
	}, nil
}

// GenerateKey creates a cache key from query and source
func (r *RedisCache) GenerateKey(source, query string) string {
	data := fmt.Sprintf("%s:%s", source, query)
	hash := sha256.Sum256([]byte(data))
	return fmt.Sprintf("query:%s:%x", source, hash[:8])
}

// Get retrieves cached data
func (r *RedisCache) Get(ctx context.Context, key string) (interface{}, bool, error) {
	val, err := r.client.Get(ctx, key).Result()
	if err == redis.Nil {
		return nil, false, nil // Cache miss
	}
	if err != nil {
		r.logger.Warn("Redis get error", zap.String("key", key), zap.Error(err))
		return nil, false, err
	}

	var data interface{}
	if err := json.Unmarshal([]byte(val), &data); err != nil {
		r.logger.Warn("Failed to unmarshal cached data", zap.String("key", key), zap.Error(err))
		return nil, false, err
	}

	r.logger.Debug("Cache hit", zap.String("key", key))
	return data, true, nil
}

// Set stores data in cache
func (r *RedisCache) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	data, err := json.Marshal(value)
	if err != nil {
		r.logger.Warn("Failed to marshal data for cache", zap.String("key", key), zap.Error(err))
		return err
	}

	// Use provided TTL or default
	if ttl == 0 {
		ttl = r.ttl
	}

	if err := r.client.Set(ctx, key, data, ttl).Err(); err != nil {
		r.logger.Warn("Redis set error", zap.String("key", key), zap.Error(err))
		return err
	}

	r.logger.Debug("Data cached",
		zap.String("key", key),
		zap.Duration("ttl", ttl))
	return nil
}

// Delete removes a key from cache
func (r *RedisCache) Delete(ctx context.Context, key string) error {
	if err := r.client.Del(ctx, key).Err(); err != nil {
		r.logger.Warn("Redis delete error", zap.String("key", key), zap.Error(err))
		return err
	}
	return nil
}

// Invalidate removes all keys matching a pattern
func (r *RedisCache) Invalidate(ctx context.Context, pattern string) error {
	iter := r.client.Scan(ctx, 0, pattern, 0).Iterator()
	var keysToDelete []string

	for iter.Next(ctx) {
		keysToDelete = append(keysToDelete, iter.Val())
	}

	if err := iter.Err(); err != nil {
		r.logger.Warn("Redis scan error", zap.String("pattern", pattern), zap.Error(err))
		return err
	}

	if len(keysToDelete) > 0 {
		if err := r.client.Del(ctx, keysToDelete...).Err(); err != nil {
			r.logger.Warn("Redis delete error", zap.Error(err))
			return err
		}
		r.logger.Info("Cache invalidated",
			zap.String("pattern", pattern),
			zap.Int("keys_deleted", len(keysToDelete)))
	}

	return nil
}

// Stats returns cache statistics
func (r *RedisCache) Stats(ctx context.Context) (map[string]interface{}, error) {
	info, err := r.client.Info(ctx, "stats").Result()
	if err != nil {
		return nil, err
	}

	dbSize, _ := r.client.DBSize(ctx).Result()

	return map[string]interface{}{
		"connected": true,
		"db_size":   dbSize,
		"info":      info,
	}, nil
}

// Close closes the Redis connection
func (r *RedisCache) Close() error {
	return r.client.Close()
}

// SetTTL updates the default TTL
func (r *RedisCache) SetTTL(ttl time.Duration) {
	r.ttl = ttl
}