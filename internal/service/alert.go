package service

import (
	"context"
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/sreagent/sreagent/internal/model"
	apperr "github.com/sreagent/sreagent/internal/pkg/errors"
	"github.com/sreagent/sreagent/internal/repository"
)

// AlertV2NotFoundError code
var ErrAlertV2NotFound = apperr.WithMessage(apperr.ErrNotFound, "alert not found")

// AlertV2Service provides business logic for the v2 Alert + Event model.
type AlertV2Service struct {
	repo   *repository.AlertRepository
	logger *zap.Logger
}

func NewAlertV2Service(repo *repository.AlertRepository, logger *zap.Logger) *AlertV2Service {
	return &AlertV2Service{repo: repo, logger: logger}
}

// GetByID returns an alert by ID.
func (s *AlertV2Service) GetByID(ctx context.Context, id uint) (*model.Alert, error) {
	alert, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, ErrAlertV2NotFound
		}
		return nil, apperr.Wrap(apperr.ErrDatabase, err)
	}
	return alert, nil
}

// List returns paginated alerts with optional filters.
func (s *AlertV2Service) List(ctx context.Context, channelID, incidentID uint, status, severity, query string, page, pageSize int) ([]model.Alert, int64, error) {
	list, total, err := s.repo.List(ctx, channelID, incidentID, status, severity, query, page, pageSize)
	if err != nil {
		s.logger.Error("failed to list alerts", zap.Error(err))
		return nil, 0, apperr.Wrap(apperr.ErrDatabase, err)
	}
	return list, total, nil
}

// UpsertFromEvent is the core ingestion path: receives a raw event, upserts the
// corresponding Alert, creates an Event record, and returns the Alert.
// This is called by the alert engine / webhook pipeline.
func (s *AlertV2Service) UpsertFromEvent(ctx context.Context, alertKey, title, source, generatorURL string, severity model.AlertSeverity, status model.AlertEventV2Status, labels, annotations model.JSONLabels, value float64, ruleID, channelID *uint) (*model.Alert, error) {
	now := time.Now()

	// Try to find existing alert by key
	existing, err := s.repo.GetByAlertKey(ctx, alertKey)
	if err != nil && err != gorm.ErrRecordNotFound {
		return nil, apperr.Wrap(apperr.ErrDatabase, err)
	}

	if existing != nil {
		// Update existing alert
		if status == model.AlertEventV2StatusFiring {
			if err := s.repo.IncrementFireCount(ctx, existing.ID, now); err != nil {
				return nil, apperr.Wrap(apperr.ErrDatabase, err)
			}
			// Update labels/annotations to latest
			existing.Labels = labels
			existing.Annotations = annotations
			existing.Severity = severity
			existing.LastFiredAt = now
			existing.Status = model.AlertStatusFiring
			if err := s.repo.Update(ctx, existing); err != nil {
				return nil, apperr.Wrap(apperr.ErrDatabase, err)
			}
		} else {
			// Resolved
			resolvedAt := now
			if err := s.repo.UpdateStatus(ctx, existing.ID, model.AlertStatusResolved, &resolvedAt); err != nil {
				return nil, apperr.Wrap(apperr.ErrDatabase, err)
			}
			existing.Status = model.AlertStatusResolved
			existing.ResolvedAt = &resolvedAt
		}

		// Create event record
		event := &model.AlertEventV2{
			AlertID:       existing.ID,
			EventStatus:   status,
			EventSeverity: severity,
			Labels:        labels,
			Annotations:   annotations,
			Value:         value,
			Timestamp:     now,
		}
		if err := s.repo.CreateEvent(ctx, event); err != nil {
			s.logger.Error("failed to create event for existing alert", zap.Error(err), zap.String("alert_key", alertKey))
		}

		return existing, nil
	}

	// Create new alert
	alert := &model.Alert{
		AlertKey:     alertKey,
		Title:        title,
		Severity:     severity,
		Status:       model.AlertStatusFiring,
		RuleID:       ruleID,
		Labels:       labels,
		Annotations:  annotations,
		ChannelID:    channelID,
		Source:       source,
		GeneratorURL: generatorURL,
		FirstFiredAt: now,
		LastFiredAt:  now,
		EventCount:   1,
		FireCount:    1,
	}
	if status == model.AlertEventV2StatusResolved {
		alert.Status = model.AlertStatusResolved
		alert.ResolvedAt = &now
	}

	if err := s.repo.Create(ctx, alert); err != nil {
		return nil, apperr.Wrap(apperr.ErrDatabase, err)
	}

	// Create initial event
	event := &model.AlertEventV2{
		AlertID:       alert.ID,
		EventStatus:   status,
		EventSeverity: severity,
		Labels:        labels,
		Annotations:   annotations,
		Value:         value,
		Timestamp:     now,
	}
	if err := s.repo.CreateEvent(ctx, event); err != nil {
		s.logger.Error("failed to create initial event", zap.Error(err), zap.String("alert_key", alertKey))
	}

	s.logger.Info("alert created", zap.Uint("id", alert.ID), zap.String("alert_key", alertKey))
	return alert, nil
}

// ListEvents returns paginated events for an alert.
func (s *AlertV2Service) ListEvents(ctx context.Context, alertID uint, page, pageSize int) ([]model.AlertEventV2, int64, error) {
	// Verify alert exists
	_, err := s.repo.GetByID(ctx, alertID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, 0, ErrAlertV2NotFound
		}
		return nil, 0, apperr.Wrap(apperr.ErrDatabase, err)
	}

	list, total, err := s.repo.ListEvents(ctx, alertID, page, pageSize)
	if err != nil {
		return nil, 0, apperr.Wrap(apperr.ErrDatabase, err)
	}
	return list, total, nil
}

// LinkToIncident links an alert to an incident.
func (s *AlertV2Service) LinkToIncident(ctx context.Context, alertID, incidentID uint) error {
	return s.repo.LinkToIncident(ctx, alertID, incidentID)
}
