package service

import (
	"context"
	"time"

	"go.uber.org/zap"

	"github.com/sreagent/sreagent/internal/model"
)

// NotificationService is the notification routing engine.
// It dispatches alert events through the v2 notify-rule pipeline.
type NotificationService struct {
	subscribeSvc  *SubscribeRuleService
	notifyRuleSvc *NotifyRuleService
	logger        *zap.Logger
}

// NewNotificationService creates a new NotificationService.
func NewNotificationService(
	subscribeSvc *SubscribeRuleService,
	notifyRuleSvc *NotifyRuleService,
	logger *zap.Logger,
) *NotificationService {
	return &NotificationService{
		subscribeSvc:  subscribeSvc,
		notifyRuleSvc: notifyRuleSvc,
		logger:        logger,
	}
}

// RouteAlert is the main routing function. It finds matching notify rules by
// alert labels/severity, processes each through the v2 pipeline (throttle,
// dedup, template, media dispatch), and also processes user/team subscriptions.
func (s *NotificationService) RouteAlert(ctx context.Context, event *model.AlertEvent) error {
	// Skip notification for silenced alerts
	if event.Status == model.EventStatusSilenced && event.SilencedUntil != nil && event.SilencedUntil.After(time.Now()) {
		s.logger.Info("skipping notification for silenced alert",
			zap.Uint("event_id", event.ID),
			zap.Time("silenced_until", *event.SilencedUntil),
		)
		return nil
	}

	// --- V2 Notify Rule Pipeline ---
	// Match notify rules by labels + severity, process each through the pipeline.
	if s.notifyRuleSvc != nil {
		rules, err := s.notifyRuleSvc.FindMatchingRules(ctx, event)
		if err != nil {
			s.logger.Error("failed to find matching notify rules",
				zap.Uint("event_id", event.ID),
				zap.Error(err),
			)
		} else if len(rules) > 0 {
			s.logger.Info("routing alert through notify rules",
				zap.Uint("event_id", event.ID),
				zap.String("alert_name", event.AlertName),
				zap.Int("matching_rules", len(rules)),
			)
			for _, rule := range rules {
				if err := s.notifyRuleSvc.ProcessEvent(ctx, event, rule.ID); err != nil {
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
				if err := s.notifyRuleSvc.ProcessEvent(ctx, event, sub.NotifyRuleID); err != nil {
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
