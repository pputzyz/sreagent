package engine

import (
	"time"

	"github.com/sreagent/sreagent/internal/model"
)

// invalidateFiringCache clears the cached firing events.
// Called when rules are added/removed or evaluators are stopped.
func (e *Evaluator) invalidateFiringCache() {
	e.firingCacheMu.Lock()
	e.firingCache = nil
	e.firingCacheAt = time.Time{}
	e.firingCacheMu.Unlock()
}

// GetFiringEvents returns a snapshot of all currently firing alert states
// across all active rule evaluators. This is a cheap in-memory operation
// that replaces the costly DB scan of eventSvc.List("firing").
func (e *Evaluator) GetFiringEvents() []*AlertState {
	// Check TTL cache first.
	ttl := e.firingCacheTTL
	if ttl == 0 {
		ttl = 5 * time.Second
	}
	e.firingCacheMu.RLock()
	if e.firingCache != nil && time.Since(e.firingCacheAt) < ttl {
		// Return a shallow copy of the cached slice (states are already deep-copied on cache fill).
		out := make([]*AlertState, len(e.firingCache))
		copy(out, e.firingCache)
		e.firingCacheMu.RUnlock()
		return out
	}
	e.firingCacheMu.RUnlock()

	// Cache miss — rebuild under write lock with double-check.
	e.firingCacheMu.Lock()
	defer e.firingCacheMu.Unlock()
	if e.firingCache != nil && time.Since(e.firingCacheAt) < ttl {
		out := make([]*AlertState, len(e.firingCache))
		copy(out, e.firingCache)
		return out
	}

	// Collect all rule evaluators — from both direct map and per-DS buckets.
	evals := e.collectAllEvaluators()

	var result []*AlertState
	for _, re := range evals {
		re.rangeStates(func(_ string, sl *stateLock) bool {
			sl.mu.Lock()
			if sl.state != nil && sl.state.Status == "firing" {
				result = append(result, copyAlertState(sl.state))
			}
			sl.mu.Unlock()
			return true
		})
	}

	e.firingCache = result
	e.firingCacheAt = time.Now()
	return result
}

// copyAlertState deep-copies an AlertState including map fields.
func copyAlertState(s *AlertState) *AlertState {
	cp := *s
	if s.Labels != nil {
		cp.Labels = make(map[string]string, len(s.Labels))
		for k, v := range s.Labels {
			cp.Labels[k] = v
		}
	}
	if s.Annotations != nil {
		cp.Annotations = make(map[string]string, len(s.Annotations))
		for k, v := range s.Annotations {
			cp.Annotations[k] = v
		}
	}
	return &cp
}

// GetFiringAlertEvents returns firing alerts as []model.AlertEvent,
// a lightweight adapter for callers that need model.AlertEvent
// (e.g. inhibition rule matching). Only ID, Status, and Labels are populated.
func (e *Evaluator) GetFiringAlertEvents() []model.AlertEvent {
	states := e.GetFiringEvents()
	events := make([]model.AlertEvent, 0, len(states))
	for _, s := range states {
		events = append(events, model.AlertEvent{
			BaseModel: model.BaseModel{ID: s.EventID},
			Status:    model.AlertEventStatus(s.Status),
			Labels:    model.JSONLabels(s.Labels),
		})
	}
	return events
}

// collectAllEvaluators returns all active RuleEvaluator instances from both
// the direct evaluator map and per-datasource buckets.
func (e *Evaluator) collectAllEvaluators() []*RuleEvaluator {
	var evals []*RuleEvaluator

	// Direct evaluators (non-perDS mode or mixed).
	e.mu.RLock()
	for _, re := range e.evaluators {
		evals = append(evals, re)
	}
	e.mu.RUnlock()

	// Per-datasource bucket evaluators.
	e.perDS.Range(func(_, v any) bool {
		bucket := v.(*PerDatasourceEvaluator)
		bucket.rules.Range(func(_, rv any) bool {
			evals = append(evals, rv.(*RuleEvaluator))
			return true
		})
		return true
	})

	return evals
}

// GetStatus returns status of the evaluation engine.
func (e *Evaluator) GetStatus() EngineStatus {
	evals := e.collectAllEvaluators()

	activeAlerts := 0
	for _, re := range evals {
		re.rangeStates(func(_ string, sl *stateLock) bool {
			sl.mu.Lock()
			if sl.state != nil && sl.state.Status == "firing" {
				activeAlerts++
			}
			sl.mu.Unlock()
			return true
		})
	}

	uptime := ""
	running := false
	select {
	case <-e.stopCh:
		running = false
	default:
		running = !e.startedAt.IsZero()
	}

	if running && !e.startedAt.IsZero() {
		uptime = time.Since(e.startedAt).Truncate(time.Second).String()
	}

	isLeader := e.leader == nil // single-instance mode = always leader
	if e.leader != nil {
		isLeader = e.leader.IsLeader()
	}

	return EngineStatus{
		Running:      running,
		TotalRules:   len(evals),
		ActiveAlerts: activeAlerts,
		Uptime:       uptime,
		IsLeader:     isLeader,
	}
}
