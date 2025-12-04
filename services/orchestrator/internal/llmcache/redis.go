package llmcache

import (
	"context"
	"fmt"
	"sync/atomic"
	"time"

	"github.com/redis/go-redis/v9"
)

// =============================================================================
// Redis Cache Implementation
// =============================================================================

// RedisCache implements Cache using Redis as the backend.
type RedisCache struct {
	client *redis.Client
	prefix string

	// Stats (atomic for thread safety)
	hits    int64
	misses  int64
	puts    int64
	deletes int64
	saved   int64
}

// RedisCacheConfig configures the Redis cache.
type RedisCacheConfig struct {
	// Addr is the Redis server address (host:port).
	Addr string

	// Password is the Redis password (optional).
	Password string

	// DB is the Redis database number.
	DB int

	// Prefix is the key prefix for all cache entries.
	Prefix string

	// MaxRetries is the maximum number of retries.
	MaxRetries int

	// PoolSize is the connection pool size.
	PoolSize int
}

// NewRedisCache creates a new Redis-backed cache.
func NewRedisCache(config *RedisCacheConfig) (*RedisCache, error) {
	if config == nil {
		return nil, fmt.Errorf("config is required")
	}

	client := redis.NewClient(&redis.Options{
		Addr:       config.Addr,
		Password:   config.Password,
		DB:         config.DB,
		MaxRetries: config.MaxRetries,
		PoolSize:   config.PoolSize,
	})

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("redis ping failed: %w", err)
	}

	prefix := config.Prefix
	if prefix == "" {
		prefix = "llmcache"
	}

	return &RedisCache{
		client: client,
		prefix: prefix,
	}, nil
}

// NewRedisCacheFromClient creates a cache from an existing Redis client.
func NewRedisCacheFromClient(client *redis.Client, prefix string) *RedisCache {
	if prefix == "" {
		prefix = "llmcache"
	}
	return &RedisCache{
		client: client,
		prefix: prefix,
	}
}

// makeKey creates the Redis key for a cache key.
func (c *RedisCache) makeKey(key *Key) string {
	return fmt.Sprintf("%s:%s", c.prefix, key.String())
}

// Get retrieves a cached result.
func (c *RedisCache) Get(ctx context.Context, key *Key) (*Result, error) {
	redisKey := c.makeKey(key)

	data, err := c.client.Get(ctx, redisKey).Bytes()
	if err == redis.Nil {
		atomic.AddInt64(&c.misses, 1)
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("redis get failed: %w", err)
	}

	result, err := UnmarshalResult(data)
	if err != nil {
		// Corrupted data - delete and return miss
		_ = c.client.Del(ctx, redisKey)
		atomic.AddInt64(&c.misses, 1)
		return nil, nil
	}

	atomic.AddInt64(&c.hits, 1)
	atomic.AddInt64(&c.saved, int64(result.ApproxTokens))

	// Update hit count asynchronously
	result.HitCount++
	go func() {
		if data, err := MarshalResult(result); err == nil {
			// Get remaining TTL
			ttl, _ := c.client.TTL(context.Background(), redisKey).Result()
			if ttl > 0 {
				_ = c.client.Set(context.Background(), redisKey, data, ttl).Err()
			}
		}
	}()

	return result, nil
}

// Put stores a result in the cache.
func (c *RedisCache) Put(ctx context.Context, key *Key, result *Result, ttl time.Duration) error {
	data, err := MarshalResult(result)
	if err != nil {
		return fmt.Errorf("marshal failed: %w", err)
	}

	redisKey := c.makeKey(key)
	if err := c.client.Set(ctx, redisKey, data, ttl).Err(); err != nil {
		return fmt.Errorf("redis set failed: %w", err)
	}

	atomic.AddInt64(&c.puts, 1)
	return nil
}

// Delete removes a cached result.
func (c *RedisCache) Delete(ctx context.Context, key *Key) error {
	redisKey := c.makeKey(key)
	if err := c.client.Del(ctx, redisKey).Err(); err != nil {
		return fmt.Errorf("redis del failed: %w", err)
	}

	atomic.AddInt64(&c.deletes, 1)
	return nil
}

// Stats returns cache statistics.
func (c *RedisCache) Stats() *Stats {
	hits := atomic.LoadInt64(&c.hits)
	misses := atomic.LoadInt64(&c.misses)
	puts := atomic.LoadInt64(&c.puts)
	deletes := atomic.LoadInt64(&c.deletes)
	saved := atomic.LoadInt64(&c.saved)

	total := hits + misses
	var hitRate float64
	if total > 0 {
		hitRate = float64(hits) / float64(total)
	}

	return &Stats{
		Hits:       hits,
		Misses:     misses,
		Puts:       puts,
		Deletes:    deletes,
		HitRate:    hitRate,
		TotalSaved: saved,
	}
}

// Close closes the Redis connection.
func (c *RedisCache) Close() error {
	return c.client.Close()
}

// Ping checks if Redis is reachable.
func (c *RedisCache) Ping(ctx context.Context) error {
	return c.client.Ping(ctx).Err()
}

// FlushAll clears all cache entries with the configured prefix.
// Use with caution in production!
func (c *RedisCache) FlushAll(ctx context.Context) error {
	pattern := fmt.Sprintf("%s:*", c.prefix)

	var cursor uint64
	for {
		keys, nextCursor, err := c.client.Scan(ctx, cursor, pattern, 100).Result()
		if err != nil {
			return fmt.Errorf("scan failed: %w", err)
		}

		if len(keys) > 0 {
			if err := c.client.Del(ctx, keys...).Err(); err != nil {
				return fmt.Errorf("del failed: %w", err)
			}
		}

		cursor = nextCursor
		if cursor == 0 {
			break
		}
	}

	return nil
}

// Count returns the approximate number of cached entries.
func (c *RedisCache) Count(ctx context.Context) (int64, error) {
	pattern := fmt.Sprintf("%s:*", c.prefix)

	var count int64
	var cursor uint64
	for {
		keys, nextCursor, err := c.client.Scan(ctx, cursor, pattern, 100).Result()
		if err != nil {
			return 0, fmt.Errorf("scan failed: %w", err)
		}

		count += int64(len(keys))
		cursor = nextCursor
		if cursor == 0 {
			break
		}
	}

	return count, nil
}
