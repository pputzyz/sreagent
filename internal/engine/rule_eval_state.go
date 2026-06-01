package engine

import (
	"context"
	"time"

	"go.uber.org/zap"

	"github.com/sreagent/sreagent/internal/pkg/metrics"
)

// lockState returns (or creates) the per-fingerprint stateLock.
// This is the primary entry point for fine-grained locking.
func (re *RuleEvaluator) lockState(fp string) *stateLock {
	sl, _ := re.states.LoadOrStore(fp, &stateLock{})
	if sl == nil {
		return &stateLock{}
	}
	if s, ok := sl.(*stateLock); ok {
		return s
	}
	return &stateLock{}
}

// deleteState removes a fingerprint's state (used after alert resolved cleanup).
func (re *RuleEvaluator) deleteState(fp string) {
	re.states.Delete(fp)
}

// rangeStates iterates all states. fn returns false to stop early.
// fn must lock sl.mu itself if it reads/writes sl.state.
func (re *RuleEvaluator) rangeStates(fn func(fp string, sl *stateLock) bool) {
	re.states.Range(func(k, v any) bool {
		fp, ok1 := k.(string)
		sl, ok2 := v.(*stateLock)
		if !ok1 || !ok2 {
			return true
		}
		return fn(fp, sl)
	})
}

// loadPersistedState restores alert states from the StateStore on startup.
// If no StateStore is configured or loading fails, this is a no-op (in-memory only).
func (re *RuleEvaluator) loadPersistedState() {
	if re.stateStore == nil {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	entries, err := re.stateStore.LoadStates(ctx, re.rule.ID)
	if err != nil {
		re.logger.Warn("failed to load persisted states, starting fresh",
			zap.Error(err),
		)
		return
	}

	if len(entries) == 0 {
		return
	}

	restored := 0
	for fp, entry := range entries {
		state := fromStateEntry(entry)
		sl := re.lockState(fp)
		sl.mu.Lock()
		sl.state = state
		sl.mu.Unlock()
		restored++

		// Restore suppressor entries for firing and pending states
		if (state.Status == "firing" || state.Status == "pending") && re.suppressor != nil {
			re.suppressor.UpdateSeverity(re.rule.ID, fp, string(re.rule.Severity))
		}
	}

	re.logger.Info("restored persisted alert states",
		zap.Int("count", restored),
	)
}

// stateTTL returns the TTL for persisted state entries.
// Uses max(1 hour, 10x eval interval) to survive longer crash-recovery windows.
// Previous 3x interval (e.g. 180s for 60s eval) was too aggressive — a brief
// restart would lose all firing state and cause duplicate alerts.
func (re *RuleEvaluator) stateTTL() time.Duration {
	interval := time.Duration(re.rule.EvalInterval) * time.Second
	if interval <= 0 {
		interval = 60 * time.Second
	}
	ttl := 10 * interval
	if ttl < time.Hour {
		ttl = time.Hour
	}
	return ttl
}

// persistState saves a state entry to the StateStore (if configured).
// Errors are logged but not propagated — Redis is best-effort.
func (re *RuleEvaluator) persistState(fp string, state *AlertState) {
	if re.stateStore == nil {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	entry := toStateEntry(fp, state)
	if err := re.stateStore.SaveState(ctx, re.rule.ID, fp, entry, re.stateTTL()); err != nil {
		metrics.IncStatePersistFailure("save")
		re.logger.Warn("failed to persist state to redis",
			zap.String("fingerprint", fp),
			zap.Error(err),
		)
	}
}

// deletePersistedState removes a state entry from the StateStore (if configured).
func (re *RuleEvaluator) deletePersistedState(fp string) {
	if re.stateStore == nil {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := re.stateStore.DeleteState(ctx, re.rule.ID, fp); err != nil {
		metrics.IncStatePersistFailure("delete")
		re.logger.Warn("failed to delete persisted state from redis",
			zap.String("fingerprint", fp),
			zap.Error(err),
		)
	}
}

// gcStates removes resolved entries whose ResolvedAt is older than 24 hours.
// Called periodically by the evaluator's Run loop.
// Two-phase approach: first mark stale entries (set state=nil), then delete
// them from the map outside the Range callback to avoid concurrent modification.
func (re *RuleEvaluator) gcStates() {
	threshold := time.Now().Add(-24 * time.Hour)
	removed := 0

	// Phase 1: Mark stale entries by setting state=nil under lock.
	re.states.Range(func(k, v any) bool {
		sl, ok := v.(*stateLock)
		if !ok {
			return true
		}
		sl.mu.Lock()
		if sl.state != nil && sl.state.Status == "resolved" && !sl.state.ResolvedAt.IsZero() && sl.state.ResolvedAt.Before(threshold) {
			sl.state = nil // Mark for deletion
			sl.mu.Unlock()
			if fp, ok := k.(string); ok {
				re.deletePersistedState(fp)
			}
			removed++
			return true
		}
		sl.mu.Unlock()
		return true
	})

	// Phase 2: Delete entries whose state is nil (marked in phase 1).
	// This is safe because once state is nil, lockState will create a new stateLock
	// via LoadOrStore, and any subsequent operation will use the fresh one.
	if removed > 0 {
		re.states.Range(func(k, v any) bool {
			sl, ok := v.(*stateLock)
			if !ok {
				return true
			}
			sl.mu.Lock()
			isEmpty := sl.state == nil
			sl.mu.Unlock()
			if isEmpty {
				re.states.Delete(k)
			}
			return true
		})

		re.logger.Debug("state GC completed",
			zap.Int("removed_entries", removed),
		)
	}
}
