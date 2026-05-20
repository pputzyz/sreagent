package engine

import (
	"context"
	"crypto/md5"
	"errors"
	"fmt"
	"sync"
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/sreagent/sreagent/internal/model"
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
	logger       *zap.Logger

	interval time.Duration
	stopCh   chan struct{}
	once     sync.Once
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

// Start runs the heartbeat check loop in a background goroutine.
func (h *HeartbeatChecker) Start() {
	go func() {
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
}

// Stop signals the background goroutine to exit.
func (h *HeartbeatChecker) Stop() {
	h.once.Do(func() {
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

	missed := rule.HeartbeatLastAt == nil || now.Sub(*rule.HeartbeatLastAt) > interval

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
	missed := rule.HeartbeatLastAt == nil || now.Sub(*rule.HeartbeatLastAt) > interval

	h.evaluateHeartbeat(ctx, rule, fingerprint, existingEvent, missed, now)
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
		Labels:      rule.Labels,
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

	// Invoke notification callback
	if h.onAlert != nil {
		h.onAlert(ctx, event)
	}
}

// resolveHeartbeatAlert transitions an open heartbeat event to resolved.
func (h *HeartbeatChecker) resolveHeartbeatAlert(ctx context.Context, event *model.AlertEvent, now time.Time) {
	event.Status = model.EventStatusResolved
	event.ResolvedAt = &now
	if err := h.eventRepo.Update(ctx, event); err != nil {
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
	h := md5.New()
	fmt.Fprintf(h, "heartbeat:rule:%d", ruleID)
	return fmt.Sprintf("%x", h.Sum(nil))
}
