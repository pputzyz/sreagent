package engine

import (
	"context"
	"fmt"
	"time"

	"go.uber.org/zap"

	"github.com/sreagent/sreagent/internal/model"
)

// checkTimeWindowMute checks if the alert should be muted by engine-level time-window rules.
// Returns (true, muteRuleID) if muted, (false, 0) otherwise.
func (re *RuleEvaluator) checkTimeWindowMute(stateLabels map[string]string, severity string) (bool, uint) {
	if re.suppressor == nil {
		return false, 0
	}
	ruleID := re.rule.ID
	return re.suppressor.IsMutedByAnyRule(re.ctx, stateLabels, severity, &ruleID)
}

// createAlertEvent creates a new alert event in the database.
// On success, state.EventID is set to the new event's ID.
// On failure, state.Status is reverted to "pending" and state.FiredAt is zeroed
// so the caller can detect the failure via state.EventID == 0 and handle it.
func (re *RuleEvaluator) createAlertEvent(state *AlertState, status model.AlertEventStatus) {
	ctx, cancel := context.WithTimeout(re.ctx, 10*time.Second)
	defer cancel()

	fp := generateFingerprint(state.Labels)

	// Merge labels — priority (high → low):
	//   1. query result labels  (state.Labels)
	//   2. rule static labels   (re.rule.Labels)
	//   3. datasource labels    (re.datasource.Labels — e.g. biz_project, tenant, project)
	labels := make(model.JSONLabels)
	// Lowest priority: datasource static labels (biz_project, tenant, project, etc.)
	for k, v := range re.datasource.Labels {
		labels[k] = v
	}
	// Rule labels override datasource labels
	for k, v := range re.rule.Labels {
		labels[k] = v
	}
	// Query result labels have highest priority
	for k, v := range state.Labels {
		labels[k] = v
	}
	// Ensure severity and alertname are in labels
	if _, ok := labels["severity"]; !ok {
		labels["severity"] = string(re.rule.Severity)
	}
	if _, ok := labels["alertname"]; !ok {
		labels["alertname"] = re.rule.Name
	}
	// Inject _channel_id hint for v2 pipeline routing (4.3)
	if re.rule.ChannelID != nil && *re.rule.ChannelID > 0 {
		labels["_channel_id"] = fmt.Sprintf("%d", *re.rule.ChannelID)
	}

	annotations := make(model.JSONLabels)
	for k, v := range re.rule.Annotations {
		annotations[k] = v
	}
	for k, v := range state.Annotations {
		annotations[k] = v
	}

	ruleID := re.rule.ID
	dsID := re.datasource.ID
	event := &model.AlertEvent{
		Fingerprint:  fp,
		RuleID:       &ruleID,
		AlertName:    re.rule.Name,
		Severity:     re.rule.Severity,
		Status:       status,
		Labels:       labels,
		Annotations:  annotations,
		Source:       re.datasource.Name,
		DataSourceID: &dsID,
		FiredAt:      state.FiredAt,
		FireCount:    1,
	}

	if err := re.eventRepo.Create(ctx, event); err != nil {
		// Check if the event already exists (timeout after commit edge case).
		existing, queryErr := re.eventRepo.GetByFingerprintAndStatus(ctx, fp, model.EventStatusFiring)
		if queryErr == nil && existing != nil {
			re.logger.Warn("alert event already exists after create error, reusing",
				zap.String("fingerprint", fp),
				zap.Uint("existing_id", existing.ID),
			)
			state.EventID = existing.ID
			return
		}
		re.logger.Error("failed to create alert event, reverting to pending for retry",
			zap.String("fingerprint", fp),
			zap.Error(err),
		)
		// Revert state so the next eval cycle retries the create.
		state.Status = "pending"
		state.FiredAt = time.Time{}
		return
	}

	state.EventID = event.ID

	// B1-2: Passively record labels into the label registry so non-Prometheus
	// datasources (Zabbix, VictoriaLogs, etc.) also get autocomplete values.
	if re.onLabelRecord != nil {
		re.onLabelRecord(re.datasource.ID, state.Labels)
	}

	re.logger.Info("alert fired",
		zap.Uint("event_id", event.ID),
		zap.String("alert_name", re.rule.Name),
		zap.String("severity", string(re.rule.Severity)),
		zap.Float64("value", state.Value),
	)

	// Call the onAlert callback to trigger notification routing.
	// Use an independent context (not re.ctx) so notifications complete even
	// during graceful shutdown. A 60s timeout prevents goroutine leaks if the
	// notification path hangs indefinitely.
	if re.onAlert != nil {
		ev := event
		fn := func(ctx context.Context) {
			defer func() {
				if r := recover(); r != nil {
					re.logger.Error("panic in onAlert callback", zap.Any("recover", r))
				}
			}()
			re.onAlert(ctx, ev)
		}
		notifyCtx, notifyCancel := context.WithTimeout(context.Background(), 60*time.Second)
		// Wrap fn so that notifyCancel is always called when the callback finishes.
		wrappedFn := func(ctx context.Context) {
			defer notifyCancel()
			fn(ctx)
		}
		if re.workerPool != nil {
			if !re.workerPool.Submit(notifyCtx, wrappedFn) {
				notifyCancel()
				re.logger.Warn("worker pool full, onAlert deferred to next eval cycle",
					zap.Uint("event_id", ev.ID),
				)
			}
		} else {
			// Use fallback semaphore to limit concurrency when no worker pool is configured
			select {
			case re.fallbackSem <- struct{}{}:
				go func() {
					defer func() { <-re.fallbackSem }()
					wrappedFn(notifyCtx)
				}()
			default:
				notifyCancel()
				re.logger.Warn("fallback semaphore full, onAlert deferred to next eval cycle",
					zap.Uint("event_id", ev.ID),
				)
			}
		}
	}
}

// updateFiringEvent atomically increments fire_count for a firing/acknowledged event
// using a single targeted UPDATE, avoiding a prior SELECT round-trip.
func (re *RuleEvaluator) updateFiringEvent(state *AlertState) {
	if state.EventID == 0 {
		re.logger.Warn("updateFiringEvent called with EventID=0, skipping — event was never created",
			zap.String("status", state.Status),
			zap.Uint("rule_id", re.rule.ID),
		)
		return
	}

	ctx, cancel := context.WithTimeout(re.ctx, 10*time.Second)
	defer cancel()

	if err := re.eventRepo.IncrFireCount(ctx, state.EventID); err != nil {
		re.logger.Warn("failed to increment fire count",
			zap.Uint("event_id", state.EventID),
			zap.Error(err),
		)
	}
}

// resolveAlertEvent resolves an existing alert event.
// Uses TransitionStatus for an atomic check-and-update to avoid TOCTOU races
// where the event status changes between the read and the write.
func (re *RuleEvaluator) resolveAlertEvent(state *AlertState) error {
	if state.EventID == 0 {
		return nil
	}

	ctx, cancel := context.WithTimeout(re.ctx, 10*time.Second)
	defer cancel()

	now := time.Now()
	ok, err := re.eventRepo.TransitionStatus(ctx, state.EventID,
		[]model.AlertEventStatus{model.EventStatusFiring},
		map[string]interface{}{
			"status":      model.EventStatusResolved,
			"resolved_at": now,
		},
	)
	if err != nil {
		re.logger.Error("failed to resolve alert event, reverting to firing for retry",
			zap.Uint("event_id", state.EventID),
			zap.String("alert_name", re.rule.Name),
			zap.Error(err),
		)
		// Revert state so the next eval cycle retries the resolve.
		state.Status = "firing"
		state.ResolvedAt = time.Time{}
		return err
	}
	if !ok {
		// Status didn't match (already resolved/closed/acknowledged/assigned/silenced).
		// Engine should not overwrite user-driven states.
		return nil
	}

	// Reload the event for the notification callback.
	event, err := re.eventRepo.GetByID(ctx, state.EventID)
	if err != nil {
		re.logger.Warn("failed to reload event after resolution for notification",
			zap.Uint("event_id", state.EventID),
			zap.Error(err),
		)
		// Resolution succeeded in DB; just skip the notification.
		return nil
	}

	re.logger.Info("alert resolved",
		zap.Uint("event_id", state.EventID),
		zap.String("alert_name", re.rule.Name),
	)

	// Notify about resolution — use an independent context so notifications
	// complete even during graceful shutdown. A 60s timeout prevents goroutine
	// leaks if the notification path hangs indefinitely.
	if re.onAlert != nil {
		ev := event
		fn := func(ctx context.Context) {
			defer func() {
				if r := recover(); r != nil {
					re.logger.Error("panic in onAlert callback (resolve)", zap.Any("recover", r))
				}
			}()
			re.onAlert(ctx, ev)
		}
		notifyCtx, notifyCancel := context.WithTimeout(context.Background(), 60*time.Second)
		// Wrap fn so that notifyCancel is always called when the callback finishes.
		wrappedFn := func(ctx context.Context) {
			defer notifyCancel()
			fn(ctx)
		}
		if re.workerPool != nil {
			if !re.workerPool.Submit(notifyCtx, wrappedFn) {
				notifyCancel()
				re.logger.Warn("worker pool full, onAlert (resolve) deferred to next eval cycle",
					zap.Uint("event_id", ev.ID),
				)
			}
		} else {
			// Use fallback semaphore to limit concurrency when no worker pool is configured
			select {
			case re.fallbackSem <- struct{}{}:
				go func() {
					defer func() { <-re.fallbackSem }()
					wrappedFn(notifyCtx)
				}()
			default:
				notifyCancel()
				re.logger.Warn("fallback semaphore full, onAlert (resolve) deferred to next eval cycle",
					zap.Uint("event_id", ev.ID),
				)
			}
		}
	}
	return nil
}
