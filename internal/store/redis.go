package store

import (
	"context"
	"encoding/json"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog/log"
)

// RedisStore wraps a Redis client for caching and session management.
type RedisStore struct {
	Client *redis.Client
}

// NewRedisStore creates a Redis connection from a URL string.
// Returns nil,nil if the URL is empty (Redis is optional).
func NewRedisStore(url string) (*RedisStore, error) {
	if url == "" {
		return nil, nil
	}

	opts, err := redis.ParseURL(url)
	if err != nil {
		return nil, err
	}

	client := redis.NewClient(opts)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		log.Warn().Err(err).Msg("Redis ping failed, continuing without cache")
		return nil, err
	}

	log.Info().Str("addr", opts.Addr).Msg("Redis connected")
	return &RedisStore{Client: client}, nil
}

// Set stores a value with TTL.
func (r *RedisStore) Set(ctx context.Context, key string, value any, ttl time.Duration) error {
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return r.Client.Set(ctx, key, data, ttl).Err()
}

// Get retrieves a value by key into the provided destination.
func (r *RedisStore) Get(ctx context.Context, key string, dest any) error {
	data, err := r.Client.Get(ctx, key).Bytes()
	if err != nil {
		return err
	}
	return json.Unmarshal(data, dest)
}

// Delete removes a key.
func (r *RedisStore) Delete(ctx context.Context, key string) error {
	return r.Client.Del(ctx, key).Err()
}

// Close closes the Redis connection.
func (r *RedisStore) Close() error {
	return r.Client.Close()
}
