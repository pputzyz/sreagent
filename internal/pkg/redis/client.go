package redis

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/go-redis/redis/v8"

	"github.com/sreagent/sreagent/internal/config"
)

// Client wraps redis.Client with helper methods for SREAgent use cases.
type Client struct {
	rdb *redis.Client
}

// New creates a new Redis client from config.
func New(cfg *config.RedisConfig) (*Client, error) {
	rdb := redis.NewClient(&redis.Options{
		Addr:     cfg.Addr(),
		Password: cfg.Password,
		DB:       cfg.DB,
		PoolSize: cfg.PoolSize,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := rdb.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to redis: %w", err)
	}

	return &Client{rdb: rdb}, nil
}

// Close closes the Redis connection.
func (c *Client) Close() error {
	return c.rdb.Close()
}

// Raw returns the underlying redis.Client for advanced operations.
func (c *Client) Raw() *redis.Client {
	return c.rdb
}

// --- Cache helpers ---

// Set stores a key-value pair with TTL.
func (c *Client) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	return c.rdb.Set(ctx, key, value, ttl).Err()
}

// Get retrieves a value by key.
func (c *Client) Get(ctx context.Context, key string) (string, error) {
	return c.rdb.Get(ctx, key).Result()
}

// Del deletes one or more keys.
func (c *Client) Del(ctx context.Context, keys ...string) error {
	return c.rdb.Del(ctx, keys...).Err()
}

// Exists checks if a key exists.
func (c *Client) Exists(ctx context.Context, key string) (bool, error) {
	n, err := c.rdb.Exists(ctx, key).Result()
	return n > 0, err
}

// --- Throttle helpers (for notification rate limiting) ---

// ThrottleKey returns a throttle cache key.
func ThrottleKey(channelID uint, fingerprint string) string {
	return fmt.Sprintf("throttle:%d:%s", channelID, fingerprint)
}

// IsThrottled checks if a notification is currently throttled.
func (c *Client) IsThrottled(ctx context.Context, key string) (bool, error) {
	return c.Exists(ctx, key)
}

// SetThrottle marks a notification as throttled for the given duration.
func (c *Client) SetThrottle(ctx context.Context, key string, ttl time.Duration) error {
	return c.Set(ctx, key, "1", ttl)
}

// --- Login rate-limit helpers ---

const loginFailPrefix = "sreagent:login:fail:"

// LoginFailKey returns the Redis key for tracking login failures for a username.
func LoginFailKey(username string) string {
	return loginFailPrefix + username
}

// IncrLoginFail atomically increments the login-failure counter for a username.
// Uses Redis INCR + EXPIRE to avoid race conditions under concurrent requests.
func (c *Client) IncrLoginFail(ctx context.Context, username string, ttl time.Duration) {
	key := LoginFailKey(username)
	pipe := c.rdb.Pipeline()
	pipe.Incr(ctx, key)
	pipe.Expire(ctx, key, ttl)
	_, _ = pipe.Exec(ctx)
}

// GetLoginFailCount returns the current login-failure count for a username.
// Returns 0 if the key does not exist.
func (c *Client) GetLoginFailCount(ctx context.Context, username string) (int64, error) {
	key := LoginFailKey(username)
	val, err := c.rdb.Get(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return 0, nil
		}
		return 0, err
	}
	return strconv.ParseInt(val, 10, 64)
}

// ClearLoginFail deletes the login-failure counter for a username.
func (c *Client) ClearLoginFail(ctx context.Context, username string) error {
	return c.rdb.Del(ctx, LoginFailKey(username)).Err()
}

// --- Captcha helpers ---

const captchaPrefix = "sreagent:captcha:"

// CaptchaKey returns the Redis key for a captcha answer.
func CaptchaKey(captchaID string) string {
	return captchaPrefix + captchaID
}

// SetCaptcha stores a captcha answer in Redis with the given TTL.
func (c *Client) SetCaptcha(ctx context.Context, captchaID, answer string, ttl time.Duration) error {
	return c.rdb.Set(ctx, CaptchaKey(captchaID), answer, ttl).Err()
}

// GetCaptcha retrieves a captcha answer and deletes it (one-time use).
func (c *Client) GetCaptcha(ctx context.Context, captchaID string) (string, error) {
	key := CaptchaKey(captchaID)
	val, err := c.rdb.GetDel(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return "", nil
		}
		return "", err
	}
	return val, nil
}

// --- Stream helpers (for alert event bus) ---

const AlertEventStream = "sreagent:alert_events"

// PublishAlertEvent publishes an alert event to the Redis stream.
func (c *Client) PublishAlertEvent(ctx context.Context, eventID uint, alertName, severity, fingerprint string) error {
	return c.rdb.XAdd(ctx, &redis.XAddArgs{
		Stream: AlertEventStream,
		Values: map[string]interface{}{
			"event_id":    eventID,
			"alert_name":  alertName,
			"severity":    severity,
			"fingerprint": fingerprint,
			"timestamp":   time.Now().Unix(),
		},
	}).Err()
}
