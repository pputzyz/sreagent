package service

import (
	"context"
	"fmt"
	"time"

	"go.uber.org/zap"

	"github.com/sreagent/sreagent/internal/model"
	"github.com/sreagent/sreagent/internal/repository"
)

// NotificationService is the notification routing engine.
// It dispatches alert events through the v2 notify-rule pipeline.
type NotificationService struct {
	subscribeSvc    *SubscribeRuleService
	notifyRuleSvc   *NotifyRuleService
	ruleRepo        *repository.AlertRuleRepository
	inhibitionSvc   *InhibitionRuleService // optional — inhibition check before routing
	muteSvc         *MuteRuleService       // optional — mute rule check before routing
	eventRepo       *repository.AlertEventRepository // optional — for fetching firing events
	logger          *zap.Logger
}

// NewNotificationService creates a new NotificationService.
func NewNotificationService(
	subscribeSvc *SubscribeRuleService,
	notifyRuleSvc *NotifyRuleService,
	ruleRepo *repository.AlertRuleRepository,
	logger *zap.Logger,
) *NotificationService {
	return &NotificationService{
		subscribeSvc:  subscribeSvc,
		notifyRuleSvc: notifyRuleSvc,
		ruleRepo:      ruleRepo,
		logger:        logger,
	}
}

// SetInhibitionService injects the inhibition rule service for pre-routing checks.
func (s *NotificationService) SetInhibitionService(svc *InhibitionRuleService) {
	s.inhibitionSvc = svc
}

// SetAlertEventRepository injects the event repository for fetching firing events (inhibition check).
func (s *NotificationService) SetAlertEventRepository(repo *repository.AlertEventRepository) {
	s.eventRepo = repo
}

// SetMuteRuleService injects the mute rule service for pre-routing mute checks.
func (s *NotificationService) SetMuteRuleService(svc *MuteRuleService) {
	s.muteSvc = svc
}

// RouteAlert is the main routing function. It finds matching notify rules by
// alert labels/severity, processes each through the v2 pipeline (throttle,
// dedup, template, media dispatch), and also processes user/team subscriptions.
//
// NOTE: EscalationPolicyID on the event is NOT checked here. Escalation is
// handled separately by the EscalationExecutor, which periodically scans
// firing events and dispatches escalation steps based on the matched policy's
// delay schedule. This separation keeps the notification path (immediate)
// distinct from the escalation path (delayed, policy-driven).
//
// NOTE: Global concurrency cap is implemented via maxConcurrentSend (semaphore)
// in NotifyMediaService. See notify_media.go:76 for the limiter initialization.
// semaphore (e.g. buffered channel of size 50) that limits concurrent
// ProcessEvent calls across all rules and subscriptions.
func (s *NotificationService) RouteAlert(ctx context.Context, event *model.AlertEvent) error {
	// Skip notification for silenced alerts
	if event.Status == model.EventStatusSilenced && event.SilencedUntil != nil && event.SilencedUntil.After(time.Now()) {
		s.logger.Info("skipping notification for silenced alert",
			zap.Uint("event_id", event.ID),
			zap.Time("silenced_until", *event.SilencedUntil),
		)
		return nil
	}

	// Inhibition check: suppress notification if a higher-priority alert is firing.
	if s.inhibitionSvc != nil && s.eventRepo != nil {
		firingEvents, _, err := s.eventRepo.List(ctx, "firing", "", 1, 500)
		if err == nil && len(firingEvents) > 0 {
			if s.inhibitionSvc.IsInhibited(ctx, event, firingEvents) {
				s.logger.Info("notification inhibited by inhibition rule",
					zap.Uint("event_id", event.ID),
					zap.String("alert_name", event.AlertName),
				)
				return nil
			}
		}
	}

	// B4-5: Mute rule check — suppress notification if any active mute rule matches.
	if s.muteSvc != nil && s.muteSvc.IsAlertMuted(ctx, event) {
		s.logger.Info("notification suppressed by mute rule",
			zap.Uint("event_id", event.ID),
			zap.String("alert_name", event.AlertName),
		)
		return nil
	}

	// Resolve datasource_id from the event's alert rule for routing.
	var dataSourceID *uint
	if s.ruleRepo != nil && event.RuleID != nil {
		rule, err := s.ruleRepo.GetByID(ctx, *event.RuleID)
		if err != nil {
			s.logger.Warn("failed to load alert rule for routing, skipping datasource filter",
				zap.Uint("event_id", event.ID),
				zap.Uint("rule_id", *event.RuleID),
				zap.Error(err),
			)
		} else {
			dataSourceID = rule.DataSourceID
		}
	}

	// --- V2 Notify Rule Pipeline ---
	// Match notify rules by labels + severity, process each through the pipeline.
	if s.notifyRuleSvc != nil {
		rules, err := s.notifyRuleSvc.FindMatchingRules(ctx, event, dataSourceID)
		if err != nil {
			s.logger.Error("failed to find matching notify rules",
				zap.Uint("event_id", event.ID),
				zap.Error(err),
			)
			return fmt.Errorf("failed to find matching notify rules: %w", err)
		} else if len(rules) > 0 {
			s.logger.Info("routing alert through notify rules",
				zap.Uint("event_id", event.ID),
				zap.String("alert_name", event.AlertName),
				zap.Int("matching_rules", len(rules)),
			)
			for _, rule := range rules {
				// M4: Deep copy event labels before each rule to prevent cross-rule contamination.
				eventCopy := shallowCopyEvent(event)
				if err := s.notifyRuleSvc.ProcessEvent(ctx, eventCopy, rule.ID); err != nil {
					s.logger.Error("failed to process event through notify rule",
						zap.Uint("event_id", event.ID),
						zap.Uint("rule_id", rule.ID),
						zap.Error(err),
					)
				}
			}
		}
	}

	// --- V2 Subscription Pipeline ---
	// Process user/team subscribe rules that reference notify rules.
	if s.subscribeSvc != nil && s.notifyRuleSvc != nil {
		subscriptions, err := s.subscribeSvc.FindSubscriptions(ctx, event)
		if err != nil {
			s.logger.Error("failed to find matching subscriptions",
				zap.Uint("event_id", event.ID),
				zap.Error(err),
			)
			return nil
		}

		if len(subscriptions) > 0 {
			s.logger.Info("processing event through subscription rules",
				zap.Uint("event_id", event.ID),
				zap.Int("matching_subscriptions", len(subscriptions)),
			)
			for _, sub := range subscriptions {
				if sub.NotifyRuleID == 0 {
					continue
				}
				eventCopy := shallowCopyEvent(event)
				if err := s.notifyRuleSvc.ProcessEvent(ctx, eventCopy, sub.NotifyRuleID); err != nil {
					s.logger.Error("failed to process event through subscribed notify rule",
						zap.Uint("event_id", event.ID),
						zap.Uint("subscribe_rule_id", sub.ID),
						zap.Uint("notify_rule_id", sub.NotifyRuleID),
						zap.Error(err),
					)
				}
			}
		}
	}

	return nil
}

// RouteAggregatedAlerts routes a batch of grouped events through the notification pipeline.
// This is called by AlertGroupManager.flushGroup when the batch route function is configured.
// It delegates to NotifyRuleService.ProcessEventBatch which handles per-rule aggregation logic.
func (s *NotificationService) RouteAggregatedAlerts(ctx context.Context, events []*model.AlertEvent) error {
	if len(events) == 0 {
		return nil
	}

	if len(events) == 1 {
		// Single event — use the normal routing path.
		return s.RouteAlert(ctx, events[0])
	}

	s.logger.Info("routing aggregated alert batch",
		zap.Int("event_count", len(events)),
		zap.String("first_alert", events[0].AlertName),
	)

	if s.notifyRuleSvc != nil {
		if err := s.notifyRuleSvc.ProcessEventBatch(ctx, events); err != nil {
			s.logger.Error("failed to process event batch",
				zap.Int("event_count", len(events)),
				zap.Error(err),
			)
			return err
		}
	}

	// Subscription pipeline: route each event individually (subscriptions are per-user).
	if s.subscribeSvc != nil && s.notifyRuleSvc != nil {
		for _, event := range events {
			subscriptions, err := s.subscribeSvc.FindSubscriptions(ctx, event)
			if err != nil {
				s.logger.Error("failed to find matching subscriptions for batch event",
					zap.Uint("event_id", event.ID),
					zap.Error(err),
				)
				continue
			}
			for _, sub := range subscriptions {
				if sub.NotifyRuleID == 0 {
					continue
				}
				eventCopy := shallowCopyEvent(event)
				if err := s.notifyRuleSvc.ProcessEvent(ctx, eventCopy, sub.NotifyRuleID); err != nil {
					s.logger.Error("failed to process batch event through subscribed notify rule",
						zap.Uint("event_id", event.ID),
						zap.Uint("subscribe_rule_id", sub.ID),
						zap.Error(err),
					)
				}
			}
		}
	}

	return nil
}

// shallowCopyEvent creates a shallow copy of the event with deep-copied labels/annotations.
// This prevents relabel steps in one notify rule from contaminating subsequent rules.
func shallowCopyEvent(event *model.AlertEvent) *model.AlertEvent {
	cp := *event
	if event.Labels != nil {
		cp.Labels = make(model.JSONLabels, len(event.Labels))
		for k, v := range event.Labels {
			cp.Labels[k] = v
		}
	}
	if event.Annotations != nil {
		cp.Annotations = make(model.JSONLabels, len(event.Annotations))
		for k, v := range event.Annotations {
			cp.Annotations[k] = v
		}
	}
	return &cp
}
