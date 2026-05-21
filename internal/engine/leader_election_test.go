package engine

import (
	"sync/atomic"
	"testing"
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
