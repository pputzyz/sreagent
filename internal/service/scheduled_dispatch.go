package service

import (
	"context"
	"fmt"
	"time"

	"go.uber.org/zap"

	"github.com/sreagent/sreagent/internal/model"
	"github.com/sreagent/sreagent/internal/repository"
)

// ScheduledDispatchService manages deferred and repeating notification dispatches.
// A background worker calls ProcessDueDispatches periodically to send pending items.
type ScheduledDispatchService struct {
	repo        *repository.ScheduledDispatchRepository
	policyRepo  *repository.DispatchPolicyRepository
	eventRepo   *repository.AlertEventRepository
	mediaSvc    *NotifyMediaService
	templateSvc *MessageTemplateService
	mediaRepo   *repository.NotifyMediaRepository
	logger      *zap.Logger
}

func NewScheduledDispatchService(
	repo *repository.ScheduledDispatchRepository,
	policyRepo *repository.DispatchPolicyRepository,
	eventRepo *repository.AlertEventRepository,
	mediaSvc *NotifyMediaService,
	templateSvc *MessageTemplateService,
	mediaRepo *repository.NotifyMediaRepository,
	logger *zap.Logger,
) *ScheduledDispatchService {
	return &ScheduledDispatchService{
		repo:        repo,
		policyRepo:  policyRepo,
		eventRepo:   eventRepo,
		mediaSvc:    mediaSvc,
		templateSvc: templateSvc,
		mediaRepo:   mediaRepo,
		logger:      logger,
	}
}

// Schedule creates a scheduled dispatch entry for deferred/repeating notification.
func (s *ScheduledDispatchService) Schedule(ctx context.Context, d *model.ScheduledDispatch) error {
	if err := s.repo.Create(ctx, d); err != nil {
		s.logger.Error("failed to create scheduled dispatch",
			zap.Uint("incident_id", d.IncidentID),
			zap.Uint("policy_id", d.PolicyID),
			zap.Error(err),
		)
		return err
	}
	s.logger.Info("scheduled dispatch created",
		zap.Uint("incident_id", d.IncidentID),
		zap.Uint("policy_id", d.PolicyID),
		zap.Time("dispatch_at", d.DispatchAt),
		zap.String("notify_mode", d.NotifyMode),
		zap.Int("max_repeats", d.MaxRepeats),
		zap.Int("repeat_interval", d.RepeatInterval),
	)
	return nil
}

// CancelByIncident cancels all pending dispatches for an incident.
// Called when an incident is acknowledged or closed.
func (s *ScheduledDispatchService) CancelByIncident(ctx context.Context, incidentID uint) error {
	return s.repo.CancelByIncident(ctx, incidentID)
}

// UpdateIncidentIDByFingerprint back-fills the incident_id on pending dispatches
// that were created before the incident aggregator resolved the incident.
func (s *ScheduledDispatchService) UpdateIncidentIDByFingerprint(ctx context.Context, fingerprint string, incidentID uint) error {
	return s.repo.UpdateIncidentIDByFingerprint(ctx, fingerprint, incidentID)
}

// ProcessDueDispatches polls for due dispatches and sends notifications.
// Called periodically by a background worker (every 30s).
func (s *ScheduledDispatchService) ProcessDueDispatches(ctx context.Context) error {
	due, err := s.repo.GetDueDispatches(ctx, time.Now(), 50)
	if err != nil {
		s.logger.Error("failed to get due dispatches", zap.Error(err))
		return err
	}

	for i := range due {
		d := &due[i]
		if sendErr := s.sendDispatch(ctx, d); sendErr != nil {
			s.logger.Error("scheduled dispatch failed",
				zap.Uint("dispatch_id", d.ID),
				zap.Uint("incident_id", d.IncidentID),
				zap.Error(sendErr),
			)
			if markErr := s.repo.MarkFailed(ctx, d.ID, sendErr.Error()); markErr != nil {
				s.logger.Error("failed to mark dispatch as failed",
					zap.Uint("dispatch_id", d.ID), zap.Error(markErr))
			}
			continue
		}

		if markErr := s.repo.MarkDispatched(ctx, d.ID); markErr != nil {
			s.logger.Error("failed to mark dispatch as dispatched",
				zap.Uint("dispatch_id", d.ID), zap.Error(markErr))
		}

		// Schedule next repeat if applicable
		// MaxRepeats=0 means unlimited repeats; MaxRepeats>0 caps the total count
		if d.RepeatInterval > 0 && (d.MaxRepeats == 0 || d.RepeatCount+1 < d.MaxRepeats) {
			nextAt := time.Now().Add(time.Duration(d.RepeatInterval) * time.Second)
			if nextErr := s.repo.ScheduleNext(ctx, d.ID, nextAt); nextErr != nil {
				s.logger.Error("failed to schedule next repeat",
					zap.Uint("dispatch_id", d.ID), zap.Error(nextErr))
			} else {
				s.logger.Info("scheduled next repeat dispatch",
					zap.Uint("dispatch_id", d.ID),
					zap.Int("repeat_count", d.RepeatCount+1),
					zap.Time("next_at", nextAt),
				)
			}
		}
	}

	return nil
}

// sendDispatch sends the actual notification for a scheduled dispatch.
func (s *ScheduledDispatchService) sendDispatch(ctx context.Context, d *model.ScheduledDispatch) error {
	// Load the dispatch policy for media/template config
	policy, err := s.policyRepo.GetByID(ctx, d.PolicyID)
	if err != nil {
		return fmt.Errorf("load dispatch policy %d: %w", d.PolicyID, err)
	}

	// Load the alert event for notification content
	event, err := s.eventRepo.GetByID(ctx, d.EventID)
	if err != nil {
		return fmt.Errorf("load alert event %d: %w", d.EventID, err)
	}

	// Build template data
	templateData := EventToTemplateData(event, nil, nil, nil)

	// Render content using the policy's template if configured
	var renderedContent string
	if policy.UnifiedTemplateID != nil && *policy.UnifiedTemplateID > 0 {
		rendered, err := s.templateSvc.RenderTemplate(ctx, *policy.UnifiedTemplateID, templateData)
		if err != nil {
			s.logger.Warn("failed to render template for scheduled dispatch, using fallback",
				zap.Uint("template_id", *policy.UnifiedTemplateID),
				zap.Error(err),
			)
			renderedContent = fmt.Sprintf("[%s] %s - %s (repeat #%d)",
				event.Severity, event.AlertName, event.Status, d.RepeatCount+1)
		} else {
			renderedContent = rendered
		}
	} else {
		renderedContent = fmt.Sprintf("[%s] %s - %s (repeat #%d)",
			event.Severity, event.AlertName, event.Status, d.RepeatCount+1)
	}

	if d.NotifyMode == "unified" {
		return s.sendUnified(ctx, policy, renderedContent, templateData)
	}

	// Default: personal_preference mode — send via the policy's unified media
	// (personal preference routing is handled by the notify rule pipeline, not here)
	return s.sendUnified(ctx, policy, renderedContent, templateData)
}

// sendUnified sends a notification via the dispatch policy's unified media.
func (s *ScheduledDispatchService) sendUnified(
	ctx context.Context,
	policy *model.DispatchPolicy,
	content string,
	data *TemplateData,
) error {
	if policy.UnifiedMediaID == nil || *policy.UnifiedMediaID == 0 {
		return fmt.Errorf("dispatch policy %d has no unified_media_id configured", policy.ID)
	}

	media, err := s.mediaRepo.GetByID(ctx, *policy.UnifiedMediaID)
	if err != nil {
		return fmt.Errorf("load media %d: %w", *policy.UnifiedMediaID, err)
	}

	return s.mediaSvc.SendNotification(ctx, media, content, data)
}

// StartWorker starts the background worker that processes due dispatches.
// Stops when ctx is cancelled.
func (s *ScheduledDispatchService) StartWorker(ctx context.Context) {
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()
		s.logger.Info("scheduled dispatch worker started")
		for {
			select {
			case <-ticker.C:
				if err := s.ProcessDueDispatches(ctx); err != nil {
					s.logger.Error("scheduled dispatch worker error", zap.Error(err))
				}
			case <-ctx.Done():
				s.logger.Info("scheduled dispatch worker stopped")
				return
			}
		}
	}()
}
