package lark

import (
	"context"
	"sync"
	"time"
)

// RateLimiter enforces Lark API rate limits:
//   - per-chat: 4 QPS (Lark limit is 5, we use 4 for safety margin), LRU 1000 chats
//   - global: 45 QPS (Lark limit is 50)
//   - per-card: 10 QPS (CardKit entity update limit)
type RateLimiter struct {
	perChat *lruBuckets
	global  *tokenBucket
	perCard *lruBuckets
}

// NewRateLimiter creates a RateLimiter with default limits.
func NewRateLimiter() *RateLimiter {
	return &RateLimiter{
		perChat: newLRUBuckets(1000, 4, 4),  // 4 QPS, burst 4
		global:  newTokenBucket(45, 45),     // 45 QPS, burst 45
		perCard: newLRUBuckets(500, 10, 10), // 10 QPS, burst 10
	}
}

// AllowChat reports whether a request to the given chat is within the per-chat limit.
func (r *RateLimiter) AllowChat(chatID string) bool {
	return r.perChat.Allow(chatID) && r.global.Allow()
}

// AllowCard reports whether an update to the given card entity is within the per-card limit.
func (r *RateLimiter) AllowCard(entityID string) bool {
	return r.perCard.Allow(entityID) && r.global.Allow()
}

// WaitChat blocks until the per-chat rate limit allows a request, or ctx is cancelled.
// Returns ctx.Err() if the context expires before the limit allows.
func (r *RateLimiter) WaitChat(ctx context.Context, chatID string) error {
	for {
		if r.AllowChat(chatID) {
			return nil
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(50 * time.Millisecond):
		}
	}
}

// --- token bucket ---

type tokenBucket struct {
	mu       sync.Mutex
	tokens   float64
	lastTime time.Time
	rate     float64 // tokens per second
	burst    float64
}

func newTokenBucket(rate, burst float64) *tokenBucket {
	return &tokenBucket{
		tokens:   burst,
		lastTime: time.Now(),
		rate:     rate,
		burst:    burst,
	}
}

func (b *tokenBucket) Allow() bool {
	b.mu.Lock()
	defer b.mu.Unlock()

	now := time.Now()
	elapsed := now.Sub(b.lastTime).Seconds()
	b.tokens += elapsed * b.rate
	if b.tokens > b.burst {
		b.tokens = b.burst
	}
	b.lastTime = now

	if b.tokens >= 1 {
		b.tokens--
		return true
	}
	return false
}

// --- LRU-bucketed per-key rate limiter ---

type lruBuckets struct {
	mu      sync.Mutex
	buckets map[string]*tokenBucket
	order   []string // LRU order: newest at end
	maxKeys int
	rate    float64
	burst   float64
}

func newLRUBuckets(maxKeys int, rate, burst float64) *lruBuckets {
	return &lruBuckets{
		buckets: make(map[string]*tokenBucket, maxKeys),
		maxKeys: maxKeys,
		rate:    rate,
		burst:   burst,
	}
}

func (l *lruBuckets) Allow(key string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	b, ok := l.buckets[key]
	if !ok {
		// Evict oldest if at capacity.
		if len(l.buckets) >= l.maxKeys {
			oldest := l.order[0]
			l.order = l.order[1:]
			delete(l.buckets, oldest)
		}
		b = newTokenBucket(l.rate, l.burst)
		l.buckets[key] = b
		l.order = append(l.order, key)
	} else {
		// Move to end (most recently used).
		for i, k := range l.order {
			if k == key {
				l.order = append(l.order[:i], l.order[i+1:]...)
				break
			}
		}
		l.order = append(l.order, key)
	}

	return b.Allow()
}
