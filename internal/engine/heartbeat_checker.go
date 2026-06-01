package engine

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/sreagent/sreagent/internal/model"
	"github.com/sreagent/sreagent/internal/pkg/fingerprint"
	"github.com/sreagent/sreagent/internal/pkg/metrics"
	"github.com/sreagent/sreagent/internal/repository"
)

// HeartbeatChecker monitors heartbeat-type alert rules.
// For each enabled heartbeat rule, it periodically checks whether a ping has
// been received within the configured interval. If not, it fires an alert event.
// When pings resume, it resolves the event automatically.
type HeartbeatChecker struct {
	ruleRepo     *repository.AlertRuleRepository
	eventRepo    *repository.AlertEventRepository
	timelineRepo *repository.AlertTimelineRepository
	onAlert      func(ctx context.Context, event *model.AlertEvent)
	leader       LeaderElection // optional; nil = always run
	pool         *AlertWorkerPool // optional; nil = use fallback semaphore
	logger       *zap.Logger

	interval    time.Duration
	stopCh      chan struct{}
	startOnce   sync.Once
	stopOnce    sync.Once
	fallbackSem chan struct{} // semaphore for onAlert when pool is nil (cap=16)
}

// NewHeartbeatChecker creates a HeartbeatChecker that runs every checkInterval.
func NewHeartbeatChecker(
	ruleRepo *repository.AlertRuleRepository,
	eventRepo *repository.AlertEventRepository,
	timelineRepo *repository.AlertTimelineRepository,
	logger *zap.Logger,
) *HeartbeatChecker {
	return &HeartbeatChecker{
		ruleRepo:     ruleRepo,
		eventRepo:    eventRepo,
		timelineRepo: timelineRepo,
		logger:       logger,
		interval:     60 * time.Second,
		stopCh:       make(chan struct{}),
		fallbackSem:  make(chan struct{}, 16),
	}
}

// SetInterval overrides the default 60-second check interval.
func (h *HeartbeatChecker) SetInterval(d time.Duration) { h.interval = d }

// SetOnAlert registers the callback invoked when a new heartbeat alert fires.
func (h *HeartbeatChecker) SetOnAlert(fn func(ctx context.Context, event *model.AlertEvent)) {
	h.onAlert = fn
}

// SetLeaderElection sets an optional distributed leader election mechanism.
// When set, only the leader instance will run heartbeat checks.
func (h *HeartbeatChecker) SetLeaderElection(le LeaderElection) {
	h.leader = le
}

// SetWorkerPool sets the bounded goroutine pool for onAlert callbacks.
func (h *HeartbeatChecker) SetWorkerPool(p *AlertWorkerPool) {
	h.pool = p
}

// Start runs the heartbeat check loop in a background goroutine.
func (h *HeartbeatChecker) Start() {
	h.startOnce.Do(func() {
		go func() {
			defer func() {
				if r := recover(); r != nil {
					h.logger.Error("heartbeat checker goroutine panic recovered", zap.Any("recover", r))
				}
			}()
			ticker := time.NewTicker(h.interval)
			defer ticker.Stop()
			h.logger.Info("heartbeat checker started", zap.Duration("interval", h.interval))
			for {
				select {
				case <-ticker.C:
					if h.leader != nil && !h.leader.IsLeader() {
						continue
					}
					ctx, cancel := context.WithTimeout(context.Background(), 55*time.Second)
					h.runOnce(ctx)
					cancel()
				case <-h.stopCh:
					h.logger.Info("heartbeat checker stopped")
					return
				}
			}
		}()
	})
}

// Stop signals the background goroutine to exit.
func (h *HeartbeatChecker) Stop() {
	h.stopOnce.Do(func() {
		select {
		case <-h.stopCh:
		default:
			close(h.stopCh)
		}
	})
}

// runOnce performs a single heartbeat check pass over all enabled heartbeat rules.
func (h *HeartbeatChecker) runOnce(ctx context.Context) {
	rules, _, err := h.ruleRepo.ListHeartbeat(ctx)
	if err != nil {
		h.logger.Error("heartbeat: failed to list heartbeat rules", zap.Error(err))
		metrics.IncHeartbeatChecks("error")
		return
	}

	// Collect fingerprints for all enabled rules and batch-query events.
	enabled := make([]*model.AlertRule, 0, len(rules))
	fingerprints := make([]string, 0, len(rules))
	for i := range rules {
		if rules[i].Status == model.RuleStatusActive {
			rule := &rules[i]
			enabled = append(enabled, rule)
			fingerprints = append(fingerprints, heartbeatFingerprint(rule.ID))
		}
	}

	metrics.SetHeartbeatActiveRules(len(enabled))

	eventMap, err := h.eventRepo.GetLatestByFingerprints(ctx, fingerprints)
	if err != nil {
		h.logger.Error("heartbeat: failed to batch-load events, falling back to per-rule queries",
			zap.Error(err),
		)
		metrics.IncHeartbeatChecks("error")
		// Fallback: per-rule query (original behaviour).
		now := time.Now()
		for _, rule := range enabled {
			h.checkRule(ctx, rule, now)
		}
		metrics.SetEngineLastHeartbeatTimestamp()
		return
	}

	now := time.Now()
	for _, rule := range enabled {
		fp := heartbeatFingerprint(rule.ID)
		h.checkRuleWithEvent(ctx, rule, eventMap[fp], now)
	}

	// Deadman switch: record successful heartbeat pass
	metrics.SetEngineLastHeartbeatTimestamp()
}

// checkRule evaluates a single heartbeat rule.
func (h *HeartbeatChecker) checkRule(ctx context.Context, rule *model.AlertRule, now time.Time) {
	fingerprint := heartbeatFingerprint(rule.ID)
	interval := time.Duration(rule.HeartbeatInterval) * time.Second

	missed, skip := h.computeMissed(rule, interval, now)
	if skip {
		return
	}

	// Look up any existing active event for this rule.
	existingEvent, err := h.eventRepo.GetByFingerprint(ctx, fingerprint)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		h.logger.Error("heartbeat: DB error looking up event",
			zap.Uint("rule_id", rule.ID), zap.Error(err))
		return
	}

	h.evaluateHeartbeat(ctx, rule, fingerprint, existingEvent, missed, now)
}

// checkRuleWithEvent evaluates a heartbeat rule using a pre-fetched event (batch path).
// The event parameter may be nil if no active event exists for the fingerprint.
func (h *HeartbeatChecker) checkRuleWithEvent(ctx context.Context, rule *model.AlertRule, existingEvent *model.AlertEvent, now time.Time) {
	fingerprint := heartbeatFingerprint(rule.ID)
	interval := time.Duration(rule.HeartbeatInterval) * time.Second
	missed, skip := h.computeMissed(rule, interval, now)
	if skip {
		return
	}

	h.evaluateHeartbeat(ctx, rule, fingerprint, existingEvent, missed, now)
}

// computeMissed determines whether a heartbeat is missed, with clock-skew tolerance.
// Returns (missed, skip). skip=true means the check should be skipped due to
// suspicious clock skew (e.g., NTP jump or VM migration).
func (h *HeartbeatChecker) computeMissed(rule *model.AlertRule, interval time.Duration, now time.Time) (bool, bool) {
	if rule.HeartbeatLastAt == nil {
		return true, false
	}
	gap := now.Sub(*rule.HeartbeatLastAt)
	if gap < 0 {
		// Clock jumped backward — skip this cycle.
		h.logger.Warn("heartbeat: clock skew detected (negative gap), skipping check",
			zap.Uint("rule_id", rule.ID),
			zap.Duration("gap", gap),
		)
		return false, true
	}
	if gap > 10*interval {
		// Gap implausibly large — likely NTP forward jump or VM migration.
		// 10x threshold is generous enough to tolerate brief monitoring gaps
		// while still catching genuine clock skew.
		h.logger.Warn("heartbeat: suspicious clock skew detected (gap >> interval), skipping check",
			zap.Uint("rule_id", rule.ID),
			zap.Duration("gap", gap),
			zap.Duration("interval", interval),
		)
		return false, true
	}
	return gap > interval, false
}

// evaluateHeartbeat contains the shared decision logic for heartbeat rules.
func (h *HeartbeatChecker) evaluateHeartbeat(ctx context.Context, rule *model.AlertRule, fingerprint string, existingEvent *model.AlertEvent, missed bool, now time.Time) {
	if missed {
		// Heartbeat is overdue — ensure a firing event exists.
		if existingEvent == nil || existingEvent.Status == model.EventStatusResolved || existingEvent.Status == model.EventStatusClosed {
			h.fireHeartbeatAlert(ctx, rule, fingerprint, now)
		}
		// Already firing — nothing to do.
	} else {
		// Heartbeat is healthy — auto-resolve any open event.
		if existingEvent != nil &&
			existingEvent.Status != model.EventStatusResolved &&
			existingEvent.Status != model.EventStatusClosed {
			h.resolveHeartbeatAlert(ctx, existingEvent, now)
		}
	}
}

// fireHeartbeatAlert creates a new firing alert event for a missed heartbeat.
func (h *HeartbeatChecker) fireHeartbeatAlert(ctx context.Context, rule *model.AlertRule, fingerprint string, now time.Time) {
	lastSeen := "never"
	if rule.HeartbeatLastAt != nil {
		lastSeen = rule.HeartbeatLastAt.Format(time.RFC3339)
	}

	event := &model.AlertEvent{
		Fingerprint: fingerprint,
		RuleID:      &rule.ID,
		AlertName:   rule.Name,
		Severity:    rule.Severity,
		Status:      model.EventStatusFiring,
		Labels:      copyLabels(rule.Labels),
		Annotations: model.JSONLabels{
			"summary":     fmt.Sprintf("Heartbeat missing for rule '%s'", rule.Name),
			"description": fmt.Sprintf("No ping received for %ds (last: %s)", rule.HeartbeatInterval, lastSeen),
		},
		Source:    "heartbeat",
		FiredAt:   now,
		FireCount: 1,
	}

	if err := h.eventRepo.Create(ctx, event); err != nil {
		h.logger.Error("heartbeat: failed to create alert event",
			zap.Uint("rule_id", rule.ID), zap.Error(err))
		return
	}

	h.logger.Warn("heartbeat alert fired",
		zap.String("rule", rule.Name),
		zap.Uint("rule_id", rule.ID),
		zap.String("fingerprint", fingerprint),
	)
	metrics.IncHeartbeatChecks("missed")

	// Record in timeline
	h.recordTimeline(ctx, event.ID, "Heartbeat alert fired — ping timeout exceeded")

	// Invoke notification callback asynchronously to avoid blocking the heartbeat loop.
	// Use an independent context because the caller's ctx is cancelled after runOnce returns.
	if h.onAlert != nil {
		notifyCtx, notifyCancel := context.WithTimeout(context.Background(), 30*time.Second)
		fn := func(ctx context.Context) {
			defer notifyCancel()
			h.onAlert(ctx, event)
		}
		if h.pool != nil {
			if !h.pool.Submit(notifyCtx, fn) {
				notifyCancel()
				h.logger.Warn("heartbeat: worker pool full, onAlert deferred",
					zap.Uint("event_id", event.ID),
				)
			}
		} else {
			// Fallback: use semaphore to limit concurrency
			select {
			case h.fallbackSem <- struct{}{}:
				go func() {
					defer func() { <-h.fallbackSem }()
					fn(notifyCtx)
				}()
			default:
				notifyCancel()
				h.logger.Warn("heartbeat: fallback semaphore full, onAlert deferred",
					zap.Uint("event_id", event.ID),
				)
			}
		}
	}
}

// resolveHeartbeatAlert transitions an open heartbeat event to resolved.
// Creates a local copy to avoid mutating the caller's event pointer, which may
// still be referenced by other goroutines or caches.
func (h *HeartbeatChecker) resolveHeartbeatAlert(ctx context.Context, event *model.AlertEvent, now time.Time) {
	eventCopy := *event
	eventCopy.Status = model.EventStatusResolved
	eventCopy.ResolvedAt = &now
	if err := h.eventRepo.Update(ctx, &eventCopy); err != nil {
		h.logger.Error("heartbeat: failed to resolve alert event",
			zap.Uint("event_id", event.ID), zap.Error(err))
		return
	}
	h.recordTimeline(ctx, event.ID, "Heartbeat recovered — ping received")
	h.logger.Info("heartbeat alert resolved", zap.Uint("event_id", event.ID))
	metrics.IncHeartbeatChecks("resolved")
}

// recordTimeline appends a heartbeat action to the event timeline.
func (h *HeartbeatChecker) recordTimeline(ctx context.Context, eventID uint, note string) {
	t := &model.AlertTimeline{
		EventID: eventID,
		Action:  model.TimelineActionCreated,
		Note:    note,
	}
	if err := h.timelineRepo.Create(ctx, t); err != nil {
		h.logger.Error("heartbeat: failed to record timeline", zap.Error(err))
	}
}

// heartbeatFingerprint produces a stable fingerprint for a heartbeat rule's alert event.
func heartbeatFingerprint(ruleID uint) string {
	return fingerprint.Compute(map[string]string{
		"__heartbeat_rule__": fmt.Sprintf("%d", ruleID),
	})
}

// copyLabels creates a deep copy of a JSONLabels map to avoid shared-map mutations.
func copyLabels(src model.JSONLabels) model.JSONLabels {
	if src == nil {
		return nil
	}
	dst := make(model.JSONLabels, len(src))
	for k, v := range src {
		dst[k] = v
	}
	return dst
}
