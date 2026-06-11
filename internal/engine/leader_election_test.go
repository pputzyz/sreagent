package engine

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

func TestRedisLeaderElection_IsLeader_DefaultFalse(t *testing.T) {
	// Without a real Redis connection, we test the state management logic
	l := &RedisLeaderElection{}
	if l.IsLeader() {
		t.Error("new instance should not be leader")
	}
}

func TestRedisLeaderElection_IsLeader_SetState(t *testing.T) {
	l := &RedisLeaderElection{}
	l.isLeader.Store(true)
	if !l.IsLeader() {
		t.Error("expected IsLeader() = true after Store(true)")
	}
	l.isLeader.Store(false)
	if l.IsLeader() {
		t.Error("expected IsLeader() = false after Store(false)")
	}
}

func TestRedisLeaderElection_ConcurrentIsLeader(t *testing.T) {
	l := &RedisLeaderElection{}
	var ready atomic.Bool

	// Simulate concurrent reads while a write happens
	go func() {
		l.isLeader.Store(true)
		ready.Store(true)
	}()

	// Read concurrently — should not race
	for i := 0; i < 100; i++ {
		_ = l.IsLeader()
	}
}

// ---------------------------------------------------------------------------
// P1-4: renew() must fail-safe step down when Redis is unreachable beyond TTL
// ---------------------------------------------------------------------------

// newUnreachableElection returns an election whose Redis client points at a
// closed port, so every renew() call fails with a connection error. This
// exercises the REAL renew() error path (not a hand-simulated state change).
func newUnreachableElection(t *testing.T) *RedisLeaderElection {
	t.Helper()
	rdb := redis.NewClient(&redis.Options{
		Addr:        "127.0.0.1:1", // nothing listens here
		DialTimeout: 100 * time.Millisecond,
		ReadTimeout: 100 * time.Millisecond,
		MaxRetries:  -1, // fail fast, no backoff
	})
	return &RedisLeaderElection{
		rdb:    rdb,
		logger: zap.NewNop(),
		value:  "test-instance",
	}
}

// Test_LeaderElection_RenewFailsBeyondTTL_StepsDown: the lock in Redis has
// long expired (last successful renew > leaderLockTTL ago) and Redis is
// unreachable — renew() must step down instead of staying a phantom leader
// while another instance may have acquired the lock (split-brain).
func Test_LeaderElection_RenewFailsBeyondTTL_StepsDown(t *testing.T) {
	l := newUnreachableElection(t)
	l.isLeader.Store(true)
	l.lastRenewOK.Store(time.Now().Add(-2 * leaderLockTTL).UnixNano())

	l.renew(context.Background())

	assert.False(t, l.IsLeader(),
		"renew failing beyond the lock TTL must step down to avoid split-brain")
}

// Test_LeaderElection_RenewFailsWithinTTL_StaysLeader: a transient Redis
// blip within the lock TTL must NOT cause a step-down — the lock is still
// held in Redis, so flapping here would cause unnecessary leader churn.
func Test_LeaderElection_RenewFailsWithinTTL_StaysLeader(t *testing.T) {
	l := newUnreachableElection(t)
	l.isLeader.Store(true)
	l.lastRenewOK.Store(time.Now().UnixNano())

	l.renew(context.Background())

	assert.True(t, l.IsLeader(),
		"a single renew failure within the lock TTL must not step down")
}
