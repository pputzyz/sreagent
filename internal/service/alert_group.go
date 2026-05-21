package service

import (
	"context"
	"fmt"
	"sync"
	"time"

	"go.uber.org/zap"

	"github.com/sreagent/sreagent/internal/model"
	"github.com/sreagent/sreagent/internal/repository"
)

// alertGroup holds buffered events for a single notification group.
type alertGroup struct {
	key           string
	events        []*model.AlertEvent
	groupWait     time.Duration
	groupInterval time.Duration
	timer         *time.Timer
	lastFlush     time.Time
	mu            sync.Mutex
}

// AlertGroupManager buffers firing alerts by group key and flushes them
// after group_wait (first batch) or group_interval (subsequent batches),
// implementing Alertmanager-style notification grouping.
type AlertGroupManager struct {
	groups    map[string]*alertGroup
	mu        sync.RWMutex
	routeFunc func(ctx context.Context, event *model.AlertEvent) error
	ruleRepo  *repository.AlertRuleRepository
	logger    *zap.Logger
	stopCh    chan struct{}
	stopped   bool
	serverCtx context.Context // server lifecycle context for timer callbacks
}

// NewAlertGroupManager creates a new AlertGroupManager.
// routeFunc is the downstream notification dispatch function (typically notifySvc.RouteAlert).
func NewAlertGroupManager(
	routeFunc func(ctx context.Context, event *model.AlertEvent) error,
	ruleRepo *repository.AlertRuleRepository,
	logger *zap.Logger,
) *AlertGroupManager {
	return &AlertGroupManager{
		groups:    make(map[string]*alertGroup),
		routeFunc: routeFunc,
		ruleRepo:  ruleRepo,
		logger:    logger,
		stopCh:    make(chan struct{}),
		serverCtx: context.Background(),
	}
}

// WithServerContext sets the server lifecycle context for timer-based flushes.
func (m *AlertGroupManager) WithServerContext(ctx context.Context) {
	m.serverCtx = ctx
}

// ProcessEvent is the main entry point. For firing events it buffers them
// according to group_wait/group_interval settings on the alert rule.
// For resolved events it dispatches immediately (no grouping).
func (m *AlertGroupManager) ProcessEvent(ctx context.Context, event *model.AlertEvent) error {
	// Resolution events bypass grouping — send immediately.
	if event.Status == model.EventStatusResolved {
		return m.routeFunc(ctx, event)
	}

	// Look up the rule's grouping config.
	groupWait, groupInterval := m.getGroupTiming(ctx, event)

	// If both are zero, grouping is disabled — pass through.
	if groupWait == 0 && groupInterval == 0 {
		return m.routeFunc(ctx, event)
	}

	// Derive group key.
	groupKey := m.getGroupKey(ctx, event)

	m.mu.Lock()
	g, exists := m.groups[groupKey]
	if !exists {
		g = &alertGroup{
			key:           groupKey,
			groupWait:     groupWait,
			groupInterval: groupInterval,
		}
		m.groups[groupKey] = g
	}
	m.mu.Unlock()

	g.mu.Lock()
	g.events = append(g.events, event)

	// If no timer is running, start one.
	if g.timer == nil {
		var delay time.Duration
		if g.lastFlush.IsZero() {
			// First notification for this group — use group_wait.
			delay = g.groupWait
		} else {
			// Subsequent notification — use group_interval.
			delay = g.groupInterval
		}

		if delay <= 0 {
			// No delay needed — flush immediately.
			g.mu.Unlock()
			m.flushGroup(groupKey)
			return nil
		}

		g.timer = time.AfterFunc(delay, func() {
			m.flushGroup(groupKey)
		})
		m.logger.Debug("group timer started",
			zap.String("group_key", groupKey),
			zap.Duration("delay", delay),
		)
	}
	g.mu.Unlock()

	return nil
}

// flushGroup dispatches all buffered events in a group through routeFunc.
func (m *AlertGroupManager) flushGroup(key string) {
	m.mu.RLock()
	g, exists := m.groups[key]
	m.mu.RUnlock()
	if !exists {
		return
	}

	g.mu.Lock()
	events := g.events
	g.events = nil
	g.timer = nil
	g.lastFlush = time.Now()
	g.mu.Unlock()

	if len(events) == 0 {
		return
	}

	m.logger.Info("flushing alert group",
		zap.String("group_key", key),
		zap.Int("event_count", len(events)),
	)

	flushCtx, cancel := context.WithTimeout(m.serverCtx, 30*time.Second)
	defer cancel()

	for _, event := range events {
		if err := m.routeFunc(flushCtx, event); err != nil {
			m.logger.Error("failed to route grouped alert",
				zap.String("group_key", key),
				zap.Uint("event_id", event.ID),
				zap.Error(err),
			)
		}
	}
}

// getGroupTiming reads group_wait_seconds and group_interval_seconds from the
// alert rule associated with the event.
func (m *AlertGroupManager) getGroupTiming(ctx context.Context, event *model.AlertEvent) (time.Duration, time.Duration) {
	if event.RuleID == nil || *event.RuleID == 0 {
		return 0, 0
	}

	rule, err := m.ruleRepo.GetByID(ctx, *event.RuleID)
	if err != nil {
		m.logger.Warn("failed to load rule for group timing, disabling grouping",
			zap.Uint("rule_id", *event.RuleID),
			zap.Error(err),
		)
		return 0, 0
	}

	return time.Duration(rule.GroupWaitSeconds) * time.Second,
		time.Duration(rule.GroupIntervalSeconds) * time.Second
}

// getGroupKey derives the notification group key for an event.
// If the rule has a GroupName, the key is "{GroupName}:{RuleID}".
// Otherwise it's "rule:{RuleID}" (each rule is its own group).
func (m *AlertGroupManager) getGroupKey(ctx context.Context, event *model.AlertEvent) string {
	ruleID := uint(0)
	if event.RuleID != nil {
		ruleID = *event.RuleID
	}

	// Try to get GroupName from event labels or load from rule.
	if ruleID > 0 {
		rule, err := m.ruleRepo.GetByID(ctx, ruleID)
		if err == nil && rule.GroupName != "" {
			return fmt.Sprintf("%s:%d", rule.GroupName, ruleID)
		}
	}

	return fmt.Sprintf("rule:%d", ruleID)
}

// Stop cancels all pending timers and prevents new events from being buffered.
func (m *AlertGroupManager) Stop() {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.stopped {
		return
	}
	m.stopped = true
	close(m.stopCh)

	// Flush all remaining groups.
	for key, g := range m.groups {
		g.mu.Lock()
		if g.timer != nil {
			g.timer.Stop()
			g.timer = nil
		}
		events := g.events
		g.events = nil
		g.mu.Unlock()

		// Best-effort flush of remaining events.
		if len(events) > 0 {
			m.logger.Info("flushing remaining group on shutdown",
				zap.String("group_key", key),
				zap.Int("event_count", len(events)),
			)
			for _, event := range events {
				_ = m.routeFunc(m.serverCtx, event)
			}
		}
	}

	m.logger.Info("alert group manager stopped")
}
