package store

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
)

var ErrCacheMiss = errors.New("cache: key not found or expired")

// CacheProvider defines pluggable caching operations.
type CacheProvider interface {
	Get(ctx context.Context, key string, dest any) error
	Set(ctx context.Context, key string, value any, ttl time.Duration) error
	Delete(ctx context.Context, key string) error
}

type inMemoryItem struct {
	Value     []byte
	ExpiresAt time.Time
}

// InMemoryCache is a thread-safe in-memory CacheProvider.
type InMemoryCache struct {
	mu         sync.RWMutex
	items      map[string]inMemoryItem
	cleanup    chan struct{}
	wg         sync.WaitGroup
	tickerChan chan struct{}
}

// NewInMemoryCache creates a new InMemoryCache.
func NewInMemoryCache() *InMemoryCache {
	return &InMemoryCache{
		items:   make(map[string]inMemoryItem),
		cleanup: make(chan struct{}),
	}
}

// Start launches a periodic cleanup worker to purge expired keys.
func (c *InMemoryCache) Start(interval time.Duration) {
	c.wg.Add(1)
	go func() {
		defer c.wg.Done()
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				c.PurgeExpired()
			case <-c.cleanup:
				return
			}
		}
	}()
	log.Info().Msg("InMemoryCache eviction worker started")
}

// Stop stops the background eviction worker.
func (c *InMemoryCache) Stop() {
	close(c.cleanup)
	c.wg.Wait()
	log.Info().Msg("InMemoryCache eviction worker stopped")
}

// Get retrieves a key, unmarshaling it into the destination object.
func (c *InMemoryCache) Get(ctx context.Context, key string, dest any) error {
	c.mu.RLock()
	item, exists := c.items[key]
	c.mu.RUnlock()

	if !exists {
		return ErrCacheMiss
	}

	if !item.ExpiresAt.IsZero() && time.Now().After(item.ExpiresAt) {
		// Key expired: perform lazy delete
		go c.Delete(context.Background(), key)
		return ErrCacheMiss
	}

	return json.Unmarshal(item.Value, dest)
}

// Set stores a key-value pair with a TTL duration.
func (c *InMemoryCache) Set(ctx context.Context, key string, value any, ttl time.Duration) error {
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}

	var expiresAt time.Time
	if ttl > 0 {
		expiresAt = time.Now().Add(ttl)
	}

	c.mu.Lock()
	c.items[key] = inMemoryItem{
		Value:     data,
		ExpiresAt: expiresAt,
	}
	c.mu.Unlock()
	return nil
}

// Delete deletes a key.
func (c *InMemoryCache) Delete(ctx context.Context, key string) error {
	c.mu.Lock()
	delete(c.items, key)
	c.mu.Unlock()
	return nil
}

// PurgeExpired removes all expired items from the cache.
func (c *InMemoryCache) PurgeExpired() {
	now := time.Now()
	c.mu.Lock()
	for k, item := range c.items {
		if !item.ExpiresAt.IsZero() && now.After(item.ExpiresAt) {
			delete(c.items, k)
		}
	}
	c.mu.Unlock()
}
