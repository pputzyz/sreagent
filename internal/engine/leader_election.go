package engine

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/go-redis/redis/v8"
	"go.uber.org/zap"

	"github.com/sreagent/sreagent/internal/pkg/metrics"
)

const (
	leaderLockKey  = "sreagent:leader:engine"
	leaderLockTTL  = 15 * time.Second
	renewInterval  = 5 * time.Second // renew at TTL/3
)

// LeaderElection defines the interface for distributed leader election.
type LeaderElection interface {
	// TryAcquire attempts to acquire leadership. Returns true if this instance
	// became the leader (or was already the leader).
	TryAcquire(ctx context.Context) bool
	// IsLeader returns whether this instance currently holds leadership.
	IsLeader() bool
	// Release voluntarily releases leadership (e.g. during graceful shutdown).
	Release(ctx context.Context)
	// Start begins the background renewal loop. Must be called after TryAcquire.
	Start(ctx context.Context)
	// Stop stops the background renewal loop and releases the lock.
	Stop()
}

// RedisLeaderElection implements LeaderElection using Redis SET NX EX.
type RedisLeaderElection struct {
	rdb       *redis.Client
	logger    *zap.Logger
	value     string // unique instance identifier
	isLeader  bool
	cancel    context.CancelFunc
}

// NewRedisLeaderElection creates a new Redis-based leader election instance.
// The value is a unique identifier for this instance (hostname:pid).
func NewRedisLeaderElection(rdb *redis.Client, logger *zap.Logger) *RedisLeaderElection {
	hostname, _ := os.Hostname()
	value := fmt.Sprintf("%s:%d", hostname, os.Getpid())
	return &RedisLeaderElection{
		rdb:    rdb,
		logger: logger.With(zap.String("leader_value", value)),
		value:  value,
	}
}

// TryAcquire attempts to acquire the leader lock using SET NX EX.
func (l *RedisLeaderElection) TryAcquire(ctx context.Context) bool {
	ok, err := l.rdb.SetNX(ctx, leaderLockKey, l.value, leaderLockTTL).Result()
	if err != nil {
		l.logger.Warn("leader election: failed to acquire lock", zap.Error(err))
		return false
	}
	if ok {
		l.isLeader = true
		l.logger.Info("leader election: acquired leadership")
		return true
	}

	// Check if we already hold the lock (e.g. after a restart within TTL)
	val, err := l.rdb.Get(ctx, leaderLockKey).Result()
	if err != nil {
		return false
	}
	if val == l.value {
		// We already hold it — extend TTL
		l.rdb.Set(ctx, leaderLockKey, l.value, leaderLockTTL)
		l.isLeader = true
		l.logger.Info("leader election: re-acquired existing lock")
		return true
	}

	l.logger.Debug("leader election: another instance holds the lock", zap.String("holder", val))
	return false
}

// IsLeader returns whether this instance is the current leader.
func (l *RedisLeaderElection) IsLeader() bool {
	return l.isLeader
}

// Start begins the background renewal goroutine.
// It periodically renews the lock and re-acquires leadership if lost.
func (l *RedisLeaderElection) Start(ctx context.Context) {
	ctx, l.cancel = context.WithCancel(ctx)
	go l.renewLoop(ctx)
}

// renewLoop periodically renews the leader lock. If the lock is lost,
// it attempts to re-acquire it.
func (l *RedisLeaderElection) renewLoop(ctx context.Context) {
	ticker := time.NewTicker(renewInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			l.renew(ctx)
		case <-ctx.Done():
			return
		}
	}
}

// renew extends the lock TTL if we still hold it, or attempts to re-acquire.
func (l *RedisLeaderElection) renew(ctx context.Context) {
	if l.isLeader {
		// Use a Lua script to atomically check-and-extend
		script := redis.NewScript(`
			if redis.call("GET", KEYS[1]) == ARGV[1] then
				redis.call("EXPIRE", KEYS[1], ARGV[2])
				return 1
			else
				return 0
			end
		`)
		result, err := script.Run(ctx, l.rdb, []string{leaderLockKey}, l.value, int(leaderLockTTL.Seconds())).Int()
		if err != nil {
			l.logger.Warn("leader election: failed to renew lock", zap.Error(err))
			return
		}
		if result == 0 {
			l.isLeader = false
			l.logger.Warn("leader election: lost leadership, will attempt re-acquisition")
			// Export metric
			metrics.SetEngineLeaderStatus(false)
		} else {
			l.logger.Debug("leader election: renewed lock")
		}
	} else {
		// Try to acquire
		if l.TryAcquire(ctx) {
			l.logger.Info("leader election: re-acquired leadership")
		}
	}
	// Export metric regardless
	metrics.SetEngineLeaderStatus(l.isLeader)
}

// Release releases the leader lock atomically using a Lua script.
func (l *RedisLeaderElection) Release(ctx context.Context) {
	if !l.isLeader {
		return
	}
	script := redis.NewScript(`
		if redis.call("GET", KEYS[1]) == ARGV[1] then
			return redis.call("DEL", KEYS[1])
		else
			return 0
		end
	`)
	_, err := script.Run(ctx, l.rdb, []string{leaderLockKey}, l.value).Result()
	if err != nil {
		l.logger.Warn("leader election: failed to release lock", zap.Error(err))
	}
	l.isLeader = false
	metrics.SetEngineLeaderStatus(false)
	l.logger.Info("leader election: released leadership")
}

// Stop cancels the renewal loop and releases the lock.
func (l *RedisLeaderElection) Stop() {
	if l.cancel != nil {
		l.cancel()
	}
	l.Release(context.Background())
}
