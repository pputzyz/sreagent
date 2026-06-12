package lark

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func Test_TokenBucket_BasicAllow(t *testing.T) {
	b := newTokenBucket(10, 10)
	// Should allow up to burst.
	for i := 0; i < 10; i++ {
		assert.True(t, b.Allow(), "burst should allow 10 requests")
	}
	// 11th should fail (no tokens left).
	assert.False(t, b.Allow(), "should deny when tokens exhausted")
}

func Test_TokenBucket_Refill(t *testing.T) {
	b := newTokenBucket(100, 1) // 100 QPS, burst 1
	assert.True(t, b.Allow())
	assert.False(t, b.Allow())

	// Wait for refill (100 QPS → 1 token in 10ms + margin).
	time.Sleep(20 * time.Millisecond)
	assert.True(t, b.Allow(), "should refill after waiting")
}

func Test_LRU_Eviction(t *testing.T) {
	l := newLRUBuckets(3, 10, 10)
	assert.True(t, l.Allow("a"))
	assert.True(t, l.Allow("b"))
	assert.True(t, l.Allow("c"))
	assert.Len(t, l.buckets, 3)

	// Adding a 4th key should evict "a" (oldest).
	assert.True(t, l.Allow("d"))
	assert.Len(t, l.buckets, 3)
	_, exists := l.buckets["a"]
	assert.False(t, exists, "key 'a' should have been evicted")
}

func Test_LRU_MoveToEnd(t *testing.T) {
	l := newLRUBuckets(3, 10, 10)
	l.Allow("a")
	l.Allow("b")
	l.Allow("c")
	// Access "a" again to move it to end.
	l.Allow("a")
	// Now adding "d" should evict "b" (oldest in LRU order).
	l.Allow("d")
	_, exists := l.buckets["b"]
	assert.False(t, exists, "key 'b' should have been evicted")
	_, exists = l.buckets["a"]
	assert.True(t, exists, "key 'a' should still exist")
}

func Test_RateLimiter_AllowChat(t *testing.T) {
	r := NewRateLimiter()
	// Should allow initial burst.
	assert.True(t, r.AllowChat("chat1"))
	assert.True(t, r.AllowChat("chat1"))
}

func Test_RateLimiter_AllowCard_EnforcesBurst(t *testing.T) {
	r := NewRateLimiter()
	allowed := 0
	for i := 0; i < 30; i++ {
		if r.AllowCard("entity1") {
			allowed++
		}
	}
	// Per-card burst is 10; a tiny refill margin may occur during the loop.
	assert.GreaterOrEqual(t, allowed, 10, "burst of 10 must be allowed")
	assert.LessOrEqual(t, allowed, 12, "requests beyond the per-card burst must be rejected")
}

func Test_RateLimiter_WaitCard_RespectsContext(t *testing.T) {
	r := NewRateLimiter()
	for i := 0; i < 30; i++ {
		r.AllowCard("exhausted-card")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()
	err := r.WaitCard(ctx, "exhausted-card")
	assert.ErrorIs(t, err, context.DeadlineExceeded)
}

func Test_RateLimiter_WaitChat_RespectsContext(t *testing.T) {
	r := NewRateLimiter()
	// Exhaust the global limit quickly.
	for i := 0; i < 50; i++ {
		r.AllowChat("exhaust")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()
	err := r.WaitChat(ctx, "exhaust")
	assert.ErrorIs(t, err, context.DeadlineExceeded)
}

func Test_RateLimiter_ConcurrentAccess(t *testing.T) {
	r := NewRateLimiter()
	var wg sync.WaitGroup
	var chatAllowed, cardAllowed atomic.Int64
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if r.AllowChat("shared-chat") {
				chatAllowed.Add(1)
			}
			if r.AllowCard("shared-card") {
				cardAllowed.Add(1)
			}
		}()
	}
	wg.Wait()

	// Under concurrency the buckets must still enforce their limits: the
	// per-chat burst is 4 and per-card burst is 10 (small refill margin
	// allowed). If locking were broken, far more would slip through.
	assert.GreaterOrEqual(t, chatAllowed.Load(), int64(1))
	assert.LessOrEqual(t, chatAllowed.Load(), int64(6), "per-chat burst must hold under concurrency")
	assert.GreaterOrEqual(t, cardAllowed.Load(), int64(1))
	assert.LessOrEqual(t, cardAllowed.Load(), int64(12), "per-card burst must hold under concurrency")
}
