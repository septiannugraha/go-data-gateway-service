package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"

	"go-data-gateway/internal/config"
)

// RedisCache implements the Cache interface using Redis
type RedisCache struct {
	client *redis.Client
	logger *zap.Logger
	ttl    time.Duration
}

// NewRedisCacheFromConfig creates a new Redis cache from config
func NewRedisCacheFromConfig(cfg config.RedisConfig, logger *zap.Logger) (*RedisCache, error) {
	return NewRedisCache(cfg.Host, cfg.Port, cfg.Password, cfg.DB, 5*time.Minute, logger)
}

// NewRedisCache creates a new Redis cache instance
func NewRedisCache(host string, port int, password string, db int, ttl time.Duration, logger *zap.Logger) (*RedisCache, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%d", host, port),
		Password: password,
		DB:       db,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	logger.Info("Redis cache initialized",
		zap.String("host", host),
		zap.Int("port", port))

	return &RedisCache{
		client: client,
		logger: logger,
		ttl:    ttl,
	}, nil
}

// Get retrieves a value from the cache
func (c *RedisCache) Get(ctx context.Context, key string, value interface{}) error {
	val, err := c.client.Get(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return fmt.Errorf("key not found: %s", key)
		}
		return err
	}

	return json.Unmarshal([]byte(val), value)
}

// Set stores a value in the cache
func (c *RedisCache) Set(ctx context.Context, key string, value interface{}) error {
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}

	return c.client.Set(ctx, key, data, c.ttl).Err()
}

// SetWithTTL stores a value in the cache with a custom TTL
func (c *RedisCache) SetWithTTL(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}

	return c.client.Set(ctx, key, data, ttl).Err()
}

// Delete removes a key from the cache
func (c *RedisCache) Delete(ctx context.Context, key string) error {
	return c.client.Del(ctx, key).Err()
}

// Exists checks if a key exists in the cache
func (c *RedisCache) Exists(ctx context.Context, key string) (bool, error) {
	val, err := c.client.Exists(ctx, key).Result()
	if err != nil {
		return false, err
	}
	return val > 0, nil
}

// Clear removes all keys from the cache
func (c *RedisCache) Clear(ctx context.Context) error {
	return c.client.FlushDB(ctx).Err()
}

// Close closes the Redis connection
func (c *RedisCache) Close() error {
	return c.client.Close()
}

// Stats returns cache statistics
func (c *RedisCache) Stats(ctx context.Context) (map[string]interface{}, error) {
	info, err := c.client.Info(ctx, "stats").Result()
	if err != nil {
		return nil, err
	}

	dbSize, _ := c.client.DBSize(ctx).Result()

	return map[string]interface{}{
		"connected": true,
		"db_size":   dbSize,
		"info":      info,
	}, nil
}