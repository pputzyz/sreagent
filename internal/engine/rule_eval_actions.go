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

	re.logger.Info("alert fired",
		zap.Uint("event_id", event.ID),
		zap.String("alert_name", re.rule.Name),
		zap.String("severity", string(re.rule.Severity)),
		zap.Float64("value", state.Value),
	)

	// Call the onAlert callback to trigger notification routing
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
		if re.workerPool != nil {
			if !re.workerPool.Submit(re.ctx, fn) {
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
					fn(re.ctx)
				}()
			default:
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
func (re *RuleEvaluator) resolveAlertEvent(state *AlertState) error {
	if state.EventID == 0 {
		return nil
	}

	ctx, cancel := context.WithTimeout(re.ctx, 10*time.Second)
	defer cancel()

	event, err := re.eventRepo.GetByID(ctx, state.EventID)
	if err != nil {
		re.logger.Warn("failed to get event for resolution",
			zap.Uint("event_id", state.EventID),
			zap.Error(err),
		)
		return err
	}

	if event.Status == model.EventStatusClosed || event.Status == model.EventStatusResolved {
		return nil
	}

	now := time.Now()
	event.Status = model.EventStatusResolved
	event.ResolvedAt = &now

	if err := re.eventRepo.Update(ctx, event); err != nil {
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

	re.logger.Info("alert resolved",
		zap.Uint("event_id", state.EventID),
		zap.String("alert_name", re.rule.Name),
	)

	// Notify about resolution
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
		if re.workerPool != nil {
			if !re.workerPool.Submit(re.ctx, fn) {
				re.logger.Warn("worker pool full, onAlert (resolve) deferred to next eval cycle",
					zap.Uint("event_id", ev.ID),
				)
			}
		} else {
			go fn(re.ctx)
		}
	}
	return nil
}
