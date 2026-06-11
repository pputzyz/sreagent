package service

import (
	"context"

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

// ListEvents returns paginated events for an alert.
func (s *AlertV2Service) ListEvents(ctx context.Context, alertID uint, page, pageSize int) ([]model.ViewAlertEvent, int64, error) {
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
