package api

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// rateLimiter implements a simple token bucket rate limiter per client IP.
type rateLimiter struct {
	mu       sync.Mutex
	clients  map[string]*bucket
	rate     int           // tokens per interval
	interval time.Duration // refill interval
	burst    int           // max tokens (burst capacity)
}

type bucket struct {
	tokens   int
	lastFill time.Time
}

func newRateLimiter(rate int, interval time.Duration, burst int) *rateLimiter {
	rl := &rateLimiter{
		clients:  make(map[string]*bucket),
		rate:     rate,
		interval: interval,
		burst:    burst,
	}
	// Cleanup stale entries periodically
	go rl.cleanup()
	return rl
}

func (rl *rateLimiter) allow(key string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	b, exists := rl.clients[key]
	if !exists {
		b = &bucket{tokens: rl.burst, lastFill: time.Now()}
		rl.clients[key] = b
	}

	// Refill tokens based on elapsed time
	now := time.Now()
	elapsed := now.Sub(b.lastFill)
	refillCount := int(elapsed/rl.interval) * rl.rate
	if refillCount > 0 {
		b.tokens += refillCount
		if b.tokens > rl.burst {
			b.tokens = rl.burst
		}
		b.lastFill = now
	}

	if b.tokens <= 0 {
		return false
	}

	b.tokens--
	return true
}

func (rl *rateLimiter) cleanup() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	for range ticker.C {
		rl.mu.Lock()
		staleThreshold := time.Now().Add(-10 * time.Minute)
		for key, b := range rl.clients {
			if b.lastFill.Before(staleThreshold) {
				delete(rl.clients, key)
			}
		}
		rl.mu.Unlock()
	}
}

// RateLimitMiddleware creates a rate limiter middleware.
// rate: number of requests allowed per interval
// interval: time interval for rate calculation
// burst: maximum burst capacity
func RateLimitMiddleware(rate int, interval time.Duration, burst int) gin.HandlerFunc {
	limiter := newRateLimiter(rate, interval, burst)
	return func(c *gin.Context) {
		key := c.ClientIP()
		if !limiter.allow(key) {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"error": "Rate limit exceeded. Please slow down.",
			})
			return
		}
		c.Next()
	}
}

// LoginRateLimitMiddleware is a stricter rate limiter specifically for login attempts.
// Limits to 5 attempts per minute per IP to prevent brute-force attacks.
func LoginRateLimitMiddleware() gin.HandlerFunc {
	return RateLimitMiddleware(5, time.Minute, 10)
}
