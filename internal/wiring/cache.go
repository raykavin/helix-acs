package wiring

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/raykavin/helix-acs/internal/config"
	l "github.com/raykavin/helix-acs/internal/logger"
	"github.com/raykavin/helix-acs/internal/task"
)

// ConnectCache reads the cache DB config and connects to Redis with retries.
func ConnectCache(cfg config.ConfigProvider, log l.Logger) (*redis.Client, error) {
	cc, err := cfg.GetDatabase(CacheDBName)
	if err != nil {
		return nil, fmt.Errorf("unable to find configuration for cache database %q", CacheDBName)
	}
	uri := cc.GetURI()
	client, err := ConnectRedis(uri, log)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}
	log.WithField("uri", uri).Debug("Connected to cache database")
	return client, nil
}

// DisconnectCache gracefully closes the Redis connection.
func DisconnectCache(client *redis.Client, log l.Logger) {
	if err := client.Close(); err != nil {
		log.WithError(err).Error("error closing Redis connection")
	}
}

// ConnectRedis parses the Redis URL and verifies connectivity with retries.
func ConnectRedis(uri string, log l.Logger) (*redis.Client, error) {
	opts, err := redis.ParseURL(uri)
	if err != nil {
		return nil, fmt.Errorf("redis: invalid URI %q: %w", uri, err)
	}

	client := redis.NewClient(opts)

	var lastErr error
	for attempt := 1; attempt <= DBMaxRetries; attempt++ {
		ctx, cancel := context.WithTimeout(context.Background(), DBAttemptTimeout)
		err := client.Ping(ctx).Err()
		cancel()
		if err == nil {
			return client, nil
		}
		lastErr = err
		log.WithError(err).WithField("attempt", attempt).Warn("Redis ping failed, retrying")
		time.Sleep(DBRetryInterval)
	}

	_ = client.Close()
	return nil, fmt.Errorf("redis: failed after %d attempts: %w", DBMaxRetries, lastErr)
}

// NewTaskQueue creates the Redis-backed task queue.
func NewTaskQueue(redisClient *redis.Client, ttl time.Duration, maxAttempts int) *task.RedisQueue {
	return task.NewRedisQueue(redisClient, ttl, maxAttempts)
}
