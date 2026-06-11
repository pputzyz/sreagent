package middleware

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// RateLimiterBackend is the interface that both in-memory and Redis-backed
// rate limiters implement. This allows the middleware to be agnostic of the
// underlying storage.
type RateLimiterBackend interface {
	Allow(key string) bool
	RecordFailure(key string, maxFailures int, lockoutDuration time.Duration)
	ResetFailures(key string)
}

// LoginRateLimiter is the pluggable backend for login rate limiting.
// Defaults to in-memory; set to a Redis-backed implementation for
// multi-instance deployments.
var LoginRateLimiter RateLimiterBackend

// RateLimiter provides per-key rate limiting using a token bucket algorithm.
type RateLimiter struct {
	mu      sync.Mutex
	buckets map[string]*bucket
	rate    float64 // tokens per second
	burst   int     // max tokens
	cleanup time.Duration
}

type bucket struct {
	tokens    float64
	lastTime  time.Time
	failCount int
	locked    bool
	lockUntil time.Time
}

// NewRateLimiter creates a rate limiter with the given rate (tokens/sec) and burst size.
func NewRateLimiter(rate float64, burst int) *RateLimiter {
	rl := &RateLimiter{
		buckets: make(map[string]*bucket),
		rate:    rate,
		burst:   burst,
		cleanup: 10 * time.Minute,
	}
	go rl.cleanupLoop()
	return rl
}

func (rl *RateLimiter) cleanupLoop() {
	ticker := time.NewTicker(rl.cleanup)
	defer ticker.Stop()
	for range ticker.C {
		rl.mu.Lock()
		now := time.Now()
		for k, b := range rl.buckets {
			if now.Sub(b.lastTime) > rl.cleanup {
				delete(rl.buckets, k)
			}
		}
		rl.mu.Unlock()
	}
}

// Allow checks if a request for the given key should be allowed.
func (rl *RateLimiter) Allow(key string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	b, exists := rl.buckets[key]
	if !exists {
		b = &bucket{tokens: float64(rl.burst), lastTime: now}
		rl.buckets[key] = b
	}

	// Check lockout
	if b.locked && now.Before(b.lockUntil) {
		return false
	}
	if b.locked && !now.Before(b.lockUntil) {
		b.locked = false
		b.failCount = 0
	}

	// Refill tokens
	elapsed := now.Sub(b.lastTime).Seconds()
	b.tokens += elapsed * rl.rate
	if b.tokens > float64(rl.burst) {
		b.tokens = float64(rl.burst)
	}
	b.lastTime = now

	if b.tokens < 1 {
		return false
	}
	b.tokens--
	return true
}

// RecordFailure records a failed attempt. After maxFailures consecutive failures,
// the key is locked for lockoutDuration.
func (rl *RateLimiter) RecordFailure(key string, maxFailures int, lockoutDuration time.Duration) {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	b, exists := rl.buckets[key]
	if !exists {
		b = &bucket{lastTime: time.Now()}
		rl.buckets[key] = b
	}
	b.failCount++
	if b.failCount >= maxFailures {
		b.locked = true
		b.lockUntil = time.Now().Add(lockoutDuration)
	}
}

// ResetFailures resets the failure counter for a key (e.g., on successful login).
func (rl *RateLimiter) ResetFailures(key string) {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	if b, exists := rl.buckets[key]; exists {
		b.failCount = 0
		b.locked = false
	}
}

// RateLimit returns a Gin middleware that rate-limits by the given key function.
func RateLimit(keyFunc func(*gin.Context) string, rate float64, burst int) gin.HandlerFunc {
	limiter := NewRateLimiter(rate, burst)
	return func(c *gin.Context) {
		key := keyFunc(c)
		if !limiter.Allow(key) {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"code":    42900,
				"message": "rate limit exceeded, please try again later",
			})
			return
		}
		c.Next()
	}
}

// LoginRateLimit returns a Gin middleware for login brute-force protection.
// Allows `rate` requests/sec with burst, and locks out for `lockoutDuration`
// after `maxFailures` consecutive failures.
// If LoginRateLimiter is set (e.g., Redis-backed), it is used instead of
// the in-memory limiter, enabling multi-instance rate limiting.
func LoginRateLimit(rate float64, burst int, maxFailures int, lockoutDuration time.Duration) gin.HandlerFunc {
	var backend RateLimiterBackend
	if LoginRateLimiter != nil {
		backend = LoginRateLimiter
	} else {
		backend = NewRateLimiter(rate, burst)
	}
	return loginRateLimitMiddleware(backend, maxFailures, lockoutDuration)
}

// NewLoginRateLimitWithBackend returns a login rate-limit middleware using the
// explicitly provided backend. Use this when you want to inject a specific
// limiter (e.g., Redis-backed) without relying on the global LoginRateLimiter.
func NewLoginRateLimitWithBackend(backend RateLimiterBackend, maxFailures int, lockoutDuration time.Duration) gin.HandlerFunc {
	return loginRateLimitMiddleware(backend, maxFailures, lockoutDuration)
}

func loginRateLimitMiddleware(backend RateLimiterBackend, maxFailures int, lockoutDuration time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		key := "login:" + c.ClientIP()
		if !backend.Allow(key) {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"code":    42900,
				"message": "too many login attempts, please try again later",
			})
			return
		}
		c.Next()

		// Check response status to record failures
		if c.Writer.Status() == http.StatusUnauthorized || c.Writer.Status() == http.StatusBadRequest {
			backend.RecordFailure(key, maxFailures, lockoutDuration)
		} else if c.Writer.Status() == http.StatusOK {
			backend.ResetFailures(key)
		}
	}
}
