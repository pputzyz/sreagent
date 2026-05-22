package service

import (
	"context"
	"errors"
	"fmt"
	"runtime/debug"
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/sreagent/sreagent/internal/model"
	apperr "github.com/sreagent/sreagent/internal/pkg/errors"
)

// ProcessWebhook processes an incoming AlertManager webhook payload.
func (s *AlertEventService) ProcessWebhook(ctx context.Context, payload *model.AlertManagerPayload) error {
	for _, alert := range payload.Alerts {
		if err := s.processAlert(ctx, &alert, payload); err != nil {
			s.logger.Error("failed to process alert",
				zap.String("fingerprint", alert.Fingerprint),
				zap.Error(err),
			)
			// Continue processing remaining alerts
		}
	}
	return nil
}

func (s *AlertEventService) processAlert(ctx context.Context, alert *model.AlertManagerAlert, payload *model.AlertManagerPayload) error {
	// Try to find existing event by fingerprint
	existing, err := s.repo.GetByFingerprint(ctx, alert.Fingerprint)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		s.logger.Error("failed to get event by fingerprint", zap.Error(err))
		return apperr.Wrap(apperr.ErrDatabase, err)
	}

	if alert.Status == "resolved" {
		if existing != nil && existing.Status != model.EventStatusClosed {
			now := time.Now()
			existing.Status = model.EventStatusResolved
			existing.ResolvedAt = &now
			if err := s.repo.Update(ctx, existing); err != nil {
				return err
			}
			s.addTimeline(ctx, existing.ID, model.TimelineActionResolved, nil, "Auto-resolved by AlertManager")
			s.triggerLarkCardUpdate(existing)
		}
		return nil
	}

	// Firing alert
	if existing != nil {
		// Dedup: increment fire count
		existing.FireCount++
		return s.repo.Update(ctx, existing)
	}

	// Determine severity from labels
	severity := model.SeverityWarning
	if sev, ok := alert.Labels["severity"]; ok {
		switch sev {
		case "critical":
			severity = model.SeverityCritical
		case "warning":
			severity = model.SeverityWarning
		case "info":
			severity = model.SeverityInfo
		}
	}

	alertName := alert.Labels["alertname"]
	if alertName == "" {
		alertName = "Unknown"
	}

	event := &model.AlertEvent{
		Fingerprint:  alert.Fingerprint,
		AlertName:    alertName,
		Severity:     severity,
		Status:       model.EventStatusFiring,
		Labels:       alert.Labels,
		Annotations:  alert.Annotations,
		Source:       payload.Receiver,
		GeneratorURL: alert.GeneratorURL,
		FiredAt:      alert.StartsAt,
		FireCount:    1,
	}

	if err := s.repo.Create(ctx, event); err != nil {
		return err
	}

	s.addTimeline(ctx, event.ID, model.TimelineActionCreated, nil, "Alert received from "+payload.Receiver)

	// On-call dispatch: find the current on-call person for matching schedules
	if s.onCallSvc != nil {
		if onCallUser, err := s.onCallSvc.GetCurrentOnCallForAlert(ctx, map[string]string(alert.Labels)); err == nil && onCallUser != nil {
			event.OnCallUserID = &onCallUser.ID
			event.IsDispatched = true
			if updateErr := s.repo.Update(ctx, event); updateErr != nil {
				s.logger.Error("failed to set on-call user on event",
					zap.Uint("event_id", event.ID),
					zap.Error(updateErr),
				)
			} else {
				note := fmt.Sprintf("Auto-dispatched to on-call user: %s", onCallUser.DisplayName)
				s.addTimeline(ctx, event.ID, model.TimelineActionDispatched, &onCallUser.ID, note)
			}
		}
	}

	// Trigger notification routing (bounded worker pool)
	if s.notifySvc != nil {
		eventID := event.ID
		dispatch := func(ctx context.Context) {
			if err := s.notifySvc.RouteAlert(ctx, event); err != nil {
				s.logger.Error("failed to route alert notification",
					zap.Uint("event_id", eventID),
					zap.Error(err),
				)
			}
		}
		if s.workerPool != nil {
			if !s.workerPool.Submit(s.bgCtx(), dispatch) {
				s.logger.Warn("worker pool full, notification deferred to next eval cycle",
					zap.Uint("event_id", eventID),
				)
			}
		} else {
			select {
			case s.dispatchSem <- struct{}{}:
				go func() {
					defer func() { <-s.dispatchSem }()
					defer func() {
						if r := recover(); r != nil {
							s.logger.Error("notification dispatch panic recovered", zap.Any("recover", r), zap.String("stack", string(debug.Stack())))
						}
					}()
					dispatch(s.bgCtx())
				}()
			default:
				s.logger.Warn("dispatch semaphore full, dropping notification dispatch",
					zap.Uint("event_id", eventID),
				)
			}
		}
	}

	s.logger.Info("new alert event created",
		zap.String("alert_name", alertName),
		zap.String("severity", string(severity)),
		zap.String("fingerprint", alert.Fingerprint),
	)

	return nil
}

// triggerLarkCardUpdate patches or deletes the Lark card in the background when
// the alert was originally sent via Bot API (LarkMessageID is non-empty).
// Uses the bounded worker pool when available.
func (s *AlertEventService) triggerLarkCardUpdate(event *model.AlertEvent) {
	if s.larkSvc == nil || event.LarkMessageID == "" {
		return
	}
	e := event
	fn := func(ctx context.Context) {
		if err := s.larkSvc.HandleCardLifecycle(ctx, e); err != nil {
			s.logger.Warn("failed to handle lark card lifecycle after status change",
				zap.Uint("event_id", e.ID),
				zap.String("status", string(e.Status)),
				zap.Error(err),
			)
		}
	}
	if s.workerPool != nil {
		s.workerPool.Submit(s.bgCtx(), fn) // best-effort; don't block caller
	} else {
		select {
		case s.dispatchSem <- struct{}{}:
			go func() {
				defer func() { <-s.dispatchSem }()
				defer func() {
					if r := recover(); r != nil {
						s.logger.Error("lark card update panic recovered", zap.Any("recover", r), zap.String("stack", string(debug.Stack())))
					}
				}()
				fn(s.bgCtx())
			}()
		default:
			s.logger.Warn("dispatch semaphore full, dropping lark card update",
				zap.Uint("event_id", e.ID),
			)
		}
	}
}

func (s *AlertEventService) addTimeline(ctx context.Context, eventID uint, action model.AlertTimelineAction, operatorID *uint, note string) {
	timeline := &model.AlertTimeline{
		EventID:    eventID,
		Action:     action,
		OperatorID: operatorID,
		Note:       note,
	}
	if err := s.timelineRepo.Create(ctx, timeline); err != nil {
		s.logger.Error("failed to add timeline entry",
			zap.Uint("event_id", eventID),
			zap.Error(err),
		)
	}
}
