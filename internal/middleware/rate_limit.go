package middleware

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// RateLimit provides per-IP token bucket rate limiting for sensitive
// endpoints (login, register, forgot-password). This prevents brute-force
// attacks and credential stuffing.
//
// Uses an in-memory store with automatic cleanup. For multi-instance
// deployments, swap for a Redis-backed implementation.
func RateLimit(maxRequests int, window time.Duration) gin.HandlerFunc {
	type bucket struct {
		tokens    int
		lastReset time.Time
	}

	var (
		mu      sync.Mutex
		buckets = make(map[string]*bucket)
	)

	// Background cleanup every 5 minutes to prevent memory leak from stale IPs.
	go func() {
		for {
			time.Sleep(5 * time.Minute)
			mu.Lock()
			cutoff := time.Now().Add(-2 * window)
			for ip, b := range buckets {
				if b.lastReset.Before(cutoff) {
					delete(buckets, ip)
				}
			}
			mu.Unlock()
		}
	}()

	return func(c *gin.Context) {
		ip := c.ClientIP()

		mu.Lock()
		b, exists := buckets[ip]
		if !exists {
			b = &bucket{tokens: maxRequests, lastReset: time.Now()}
			buckets[ip] = b
		}

		// Reset window if expired.
		if time.Since(b.lastReset) > window {
			b.tokens = maxRequests
			b.lastReset = time.Now()
		}

		if b.tokens <= 0 {
			mu.Unlock()
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"error": gin.H{
					"code":    "RATE_LIMITED",
					"message": "Too many requests. Please try again later.",
				},
			})
			return
		}

		b.tokens--
		mu.Unlock()
		c.Next()
	}
}
