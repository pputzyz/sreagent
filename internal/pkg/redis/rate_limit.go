package redis

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/go-redis/redis/v8"
)

const rateLimitPrefix = "sreagent:ratelimit:"

// RedisRateLimiter provides per-key rate limiting backed by Redis.
// It uses a fixed-window token bucket: INCR + EXPIRE on a per-second window key.
type RedisRateLimiter struct {
	client *Client
	burst  int     // max requests per window
	rate   float64 // tokens per second (used to compute window TTL)
}

// NewRedisRateLimiter creates a Redis-backed rate limiter.
// burst = max tokens (max requests per window), rate = tokens per second.
func NewRedisRateLimiter(client *Client, rate float64, burst int) *RedisRateLimiter {
	return &RedisRateLimiter{
		client: client,
		burst:  burst,
		rate:   rate,
	}
}

// windowKey returns the Redis key for the current time window.
// Uses a 1-second window granularity.
func (rl *RedisRateLimiter) windowKey(key string) string {
	window := time.Now().Unix()
	return fmt.Sprintf("%s%s:%d", rateLimitPrefix, key, window)
}

// Allow checks if a request for the given key should be allowed.
// Uses Redis INCR + EXPIRE for atomic token counting within a window.
func (rl *RedisRateLimiter) Allow(key string) bool {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	rdb := rl.client.Raw()
	windowKey := rl.windowKey(key)

	// Check if locked out first
	lockKey := rateLimitPrefix + "lock:" + key
	locked, err := rdb.Exists(ctx, lockKey).Result()
	if err != nil {
		// On Redis error, allow the request (fail open)
		return true
	}
	if locked > 0 {
		return false
	}

	// Atomic INCR + EXPIRE in a pipeline
	pipe := rdb.Pipeline()
	incrCmd := pipe.Incr(ctx, windowKey)
	pipe.Expire(ctx, windowKey, 2*time.Second) // expire after 2s to cover edge cases
	if _, err := pipe.Exec(ctx); err != nil {
		// On Redis error, allow the request (fail open)
		return true
	}

	count := incrCmd.Val()
	return count <= int64(rl.burst)
}

// RecordFailure records a failed attempt. After maxFailures consecutive failures,
// the key is locked for lockoutDuration.
func (rl *RedisRateLimiter) RecordFailure(key string, maxFailures int, lockoutDuration time.Duration) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	rdb := rl.client.Raw()
	failKey := rateLimitPrefix + "fail:" + key

	// Increment failure count with TTL (auto-cleanup)
	pipe := rdb.Pipeline()
	incrCmd := pipe.Incr(ctx, failKey)
	// Set TTL to 2x lockout duration so failures persist across lockout
	pipe.Expire(ctx, failKey, lockoutDuration*2)
	if _, err := pipe.Exec(ctx); err != nil {
		return
	}

	failCount := incrCmd.Val()
	if failCount >= int64(maxFailures) {
		// Set lock key with TTL
		lockKey := rateLimitPrefix + "lock:" + key
		rdb.Set(ctx, lockKey, "1", lockoutDuration)
	}
}

// ResetFailures resets the failure counter and lock for a key (e.g., on successful login).
func (rl *RedisRateLimiter) ResetFailures(key string) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	rdb := rl.client.Raw()
	failKey := rateLimitPrefix + "fail:" + key
	lockKey := rateLimitPrefix + "lock:" + key
	rdb.Del(ctx, failKey, lockKey)
}

// ClearRateLimitKeys removes all rate limit keys for a given key (for testing or admin reset).
func (rl *RedisRateLimiter) ClearRateLimitKeys(key string) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	rdb := rl.client.Raw()
	failKey := rateLimitPrefix + "fail:" + key
	lockKey := rateLimitPrefix + "lock:" + key
	// Also clear current window key
	windowKey := rl.windowKey(key)
	rdb.Del(ctx, failKey, lockKey, windowKey)
}

// GetFailCount returns the current failure count for a key (for diagnostics).
func (rl *RedisRateLimiter) GetFailCount(key string) int64 {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	rdb := rl.client.Raw()
	failKey := rateLimitPrefix + "fail:" + key
	val, err := rdb.Get(ctx, failKey).Result()
	if err != nil {
		if err == redis.Nil {
			return 0
		}
		return 0
	}
	n, _ := strconv.ParseInt(val, 10, 64)
	return n
}
