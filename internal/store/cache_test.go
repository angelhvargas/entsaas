package store

import (
	"context"
	"sync"
	"testing"
	"time"
)

func TestInMemoryCache_BasicCRUD(t *testing.T) {
	cache := NewInMemoryCache()
	ctx := context.Background()

	// Set item
	err := cache.Set(ctx, "hello", "world", 10*time.Millisecond)
	if err != nil {
		t.Fatalf("failed to set key: %v", err)
	}

	// Get item
	var val string
	err = cache.Get(ctx, "hello", &val)
	if err != nil {
		t.Fatalf("failed to get key: %v", err)
	}
	if val != "world" {
		t.Errorf("got val = %q, want %q", val, "world")
	}

	// Delete item
	err = cache.Delete(ctx, "hello")
	if err != nil {
		t.Fatalf("failed to delete key: %v", err)
	}

	err = cache.Get(ctx, "hello", &val)
	if err != ErrCacheMiss {
		t.Errorf("got error = %v, want %v", err, ErrCacheMiss)
	}
}

func TestInMemoryCache_TTLExpiration(t *testing.T) {
	cache := NewInMemoryCache()
	ctx := context.Background()

	err := cache.Set(ctx, "short-lived", 42, 5*time.Millisecond)
	if err != nil {
		t.Fatalf("failed to set key: %v", err)
	}

	// Sleep to let item expire
	time.Sleep(15 * time.Millisecond)

	var val int
	err = cache.Get(ctx, "short-lived", &val)
	if err != ErrCacheMiss {
		t.Errorf("expected cache miss, got: %v", err)
	}
}

func TestInMemoryCache_EvictionWorker(t *testing.T) {
	cache := NewInMemoryCache()
	ctx := context.Background()

	_ = cache.Set(ctx, "k1", "v1", 2*time.Millisecond)
	_ = cache.Set(ctx, "k2", "v2", 10*time.Hour) // long TTL

	cache.Start(1 * time.Millisecond)
	time.Sleep(10 * time.Millisecond)
	cache.Stop()

	// k1 should be permanently pruned by background thread
	cache.mu.RLock()
	_, k1Exists := cache.items["k1"]
	_, k2Exists := cache.items["k2"]
	cache.mu.RUnlock()

	if k1Exists {
		t.Error("k1 was not purged by eviction worker")
	}
	if !k2Exists {
		t.Error("k2 should not have been purged")
	}
}

func TestInMemoryCache_Concurrency(t *testing.T) {
	cache := NewInMemoryCache()
	ctx := context.Background()

	wg := sync.WaitGroup{}
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			_ = cache.Set(ctx, "shared", id, 1*time.Minute)
			var val int
			_ = cache.Get(ctx, "shared", &val)
		}(i)
	}
	wg.Wait()
}
