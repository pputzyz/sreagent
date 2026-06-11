package engine

import (
	"context"
	"fmt"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"github.com/go-redis/redis/v8"
	"go.uber.org/zap"

	"github.com/sreagent/sreagent/internal/pkg/metrics"
)

const (
	leaderLockKey = "sreagent:leader:engine"
	leaderLockTTL = 15 * time.Second
	renewInterval = 5 * time.Second // renew at TTL/3
)

// checkAndExtendScript atomically checks if we hold the lock and extends its TTL.
var checkAndExtendScript = redis.NewScript(`
if redis.call("GET", KEYS[1]) == ARGV[1] then
	redis.call("EXPIRE", KEYS[1], ARGV[2])
	return 1
else
	return 0
end
`)

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
	isLeader  atomic.Bool
	cancel    context.CancelFunc
	startOnce sync.Once
	wg        sync.WaitGroup
	// lastRenewOK is the UnixNano timestamp of the last successful lock
	// acquire/extend. Used to fail-safe step down when Redis is unreachable:
	// once we cannot prove lock ownership for longer than the lock TTL,
	// another instance may already hold it — continuing as leader would
	// mean two instances evaluating rules (split-brain, duplicate alerts).
	lastRenewOK atomic.Int64
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
		l.isLeader.Store(true)
		l.lastRenewOK.Store(time.Now().UnixNano())
		l.logger.Info("leader election: acquired leadership")
		return true
	}

	// Check if we already hold the lock (e.g. after a restart within TTL) — atomic check-and-extend
	result, err := checkAndExtendScript.Run(ctx, l.rdb, []string{leaderLockKey}, l.value, int(leaderLockTTL.Seconds())).Int()
	if err != nil {
		l.logger.Warn("leader election: failed to check-and-extend lock", zap.Error(err))
		return false
	}
	if result == 1 {
		l.isLeader.Store(true)
		l.lastRenewOK.Store(time.Now().UnixNano())
		l.logger.Info("leader election: re-acquired existing lock")
		return true
	}

	l.logger.Debug("leader election: another instance holds the lock")
	return false
}

// IsLeader returns whether this instance is the current leader.
func (l *RedisLeaderElection) IsLeader() bool {
	return l.isLeader.Load()
}

// Start begins the background renewal goroutine.
// It periodically renews the lock and re-acquires leadership if lost.
func (l *RedisLeaderElection) Start(ctx context.Context) {
	l.startOnce.Do(func() {
		ctx, l.cancel = context.WithCancel(ctx)
		l.wg.Add(1)
		go func() {
			defer l.wg.Done()
			l.renewLoop(ctx)
		}()
	})
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
	if l.isLeader.Load() {
		result, err := checkAndExtendScript.Run(ctx, l.rdb, []string{leaderLockKey}, l.value, int(leaderLockTTL.Seconds())).Int()
		if err != nil {
			l.logger.Warn("leader election: failed to renew lock", zap.Error(err))
			// Fail-safe degradation: if we haven't successfully renewed for
			// longer than the lock TTL, the lock has expired in Redis and
			// another instance may have taken it. Step down rather than
			// risk split-brain (two leaders evaluating rules concurrently).
			last := l.lastRenewOK.Load()
			if last > 0 && time.Since(time.Unix(0, last)) > leaderLockTTL {
				l.isLeader.Store(false)
				metrics.SetEngineLeaderStatus(false)
				l.logger.Error("leader election: renew has been failing beyond lock TTL, stepping down to avoid split-brain")
			}
			return
		}
		if result == 0 {
			l.isLeader.Store(false)
			l.logger.Warn("leader election: lost leadership, will attempt re-acquisition")
			// Export metric
			metrics.SetEngineLeaderStatus(false)
		} else {
			l.lastRenewOK.Store(time.Now().UnixNano())
			l.logger.Debug("leader election: renewed lock")
		}
	} else {
		// Try to acquire
		if l.TryAcquire(ctx) {
			l.logger.Info("leader election: re-acquired leadership")
		}
	}
	// Export metric regardless
	metrics.SetEngineLeaderStatus(l.isLeader.Load())
}

// Release releases the leader lock atomically using a Lua script.
func (l *RedisLeaderElection) Release(ctx context.Context) {
	if !l.isLeader.Load() {
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
	l.isLeader.Store(false)
	metrics.SetEngineLeaderStatus(false)
	l.logger.Info("leader election: released leadership")
}

// Stop cancels the renewal loop, waits for it to exit, and releases the lock.
func (l *RedisLeaderElection) Stop() {
	if l.cancel != nil {
		l.cancel()
	}
	l.wg.Wait() // wait for renewLoop to exit before releasing the lock
	l.Release(context.Background())
}
