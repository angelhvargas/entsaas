package event

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog/log"
)

// EventBroker defines decoupled publish/subscribe operations.
type EventBroker interface {
	Publish(ctx context.Context, channel string, message any) error
	Subscribe(ctx context.Context, channel string, handler func(payload []byte) error) error
}

// InMemoryBroker is a zero-dependency in-memory EventBroker fallback.
type InMemoryBroker struct {
	mu          sync.RWMutex
	subscribers map[string][]func([]byte) error
}

// NewInMemoryBroker creates a new InMemoryBroker.
func NewInMemoryBroker() *InMemoryBroker {
	return &InMemoryBroker{
		subscribers: make(map[string][]func([]byte) error),
	}
}

// Publish distributes the message payload to all matching in-memory channel subscribers in background goroutines.
func (b *InMemoryBroker) Publish(ctx context.Context, channel string, message any) error {
	data, err := json.Marshal(message)
	if err != nil {
		return err
	}

	b.mu.RLock()
	handlers, exists := b.subscribers[channel]
	b.mu.RUnlock()

	if !exists || len(handlers) == 0 {
		log.Debug().Str("channel", channel).Msg("Published event to 0 in-memory subscribers")
		return nil
	}

	for _, h := range handlers {
		go func(handler func([]byte) error) {
			if err := handler(data); err != nil {
				log.Error().Err(err).Str("channel", channel).Msg("In-memory event handler failed")
			}
		}(h)
	}

	return nil
}

// Subscribe registers an event handler function for the given in-memory channel.
func (b *InMemoryBroker) Subscribe(ctx context.Context, channel string, handler func(payload []byte) error) error {
	b.mu.Lock()
	b.subscribers[channel] = append(b.subscribers[channel], handler)
	b.mu.Unlock()
	return nil
}

// RedisStreamBroker implements EventBroker backed by Redis Streams.
type RedisStreamBroker struct {
	client *redis.Client
	mu     sync.Mutex
	wg     sync.WaitGroup
	ctx    context.Context
	cancel context.CancelFunc
}

// NewRedisStreamBroker creates a new RedisStreamBroker.
func NewRedisStreamBroker(client *redis.Client) *RedisStreamBroker {
	ctx, cancel := context.WithCancel(context.Background())
	return &RedisStreamBroker{
		client: client,
		ctx:    ctx,
		cancel: cancel,
	}
}

// Publish appends a message to the corresponding Redis Stream channel.
func (b *RedisStreamBroker) Publish(ctx context.Context, channel string, message any) error {
	data, err := json.Marshal(message)
	if err != nil {
		return err
	}

	// Add entry to Redis Stream
	err = b.client.XAdd(ctx, &redis.XAddArgs{
		Stream: channel,
		Values: map[string]interface{}{"payload": data},
	}).Err()

	return err
}

// Subscribe polls the Redis Stream in a background loop and dispatches entries to the handler.
func (b *RedisStreamBroker) Subscribe(ctx context.Context, channel string, handler func(payload []byte) error) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.wg.Add(1)
	go func() {
		defer b.wg.Done()
		lastID := "$" // Read only new entries added from now

		for {
			select {
			case <-b.ctx.Done():
				return
			default:
				// Poll stream with a timeout to avoid tight loop CPU usage
				streams, err := b.client.XRead(b.ctx, &redis.XReadArgs{
					Streams: []string{channel, lastID},
					Count:   10,
					Block:   2 * time.Second,
				}).Result()

				if err != nil {
					if errors.Is(err, redis.Nil) || errors.Is(err, context.Canceled) {
						continue
					}
					log.Error().Err(err).Str("channel", channel).Msg("Redis stream XRead polling failed")
					time.Sleep(1 * time.Second) // backoff
					continue
				}

				for _, stream := range streams {
					for _, message := range stream.Messages {
						lastID = message.ID
						payloadStr, exists := message.Values["payload"].(string)
						if !exists {
							continue
						}

						if err := handler([]byte(payloadStr)); err != nil {
							log.Error().Err(err).Str("channel", channel).Msg("Redis stream event handler failed")
						}
					}
				}
			}
		}
	}()

	return nil
}

// Close cancels all background subscription workers gracefully.
func (b *RedisStreamBroker) Close() {
	b.cancel()
	b.wg.Wait()
}
