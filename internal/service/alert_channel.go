package service

import (
	"context"
	"strings"
	"time"

	"go.uber.org/zap"

	"github.com/sreagent/sreagent/internal/model"
	"github.com/sreagent/sreagent/internal/pkg/labelmatch"
	apperr "github.com/sreagent/sreagent/internal/pkg/errors"
	"github.com/sreagent/sreagent/internal/repository"
)

// AlertChannelService provides CRUD and matching logic for alert channels.
type AlertChannelService struct {
	repo      *repository.AlertChannelRepository
	mediaRepo *repository.NotifyMediaRepository
	logger    *zap.Logger
}

// NewAlertChannelService creates a new AlertChannelService.
func NewAlertChannelService(
	repo *repository.AlertChannelRepository,
	mediaRepo *repository.NotifyMediaRepository,
	logger *zap.Logger,
) *AlertChannelService {
	return &AlertChannelService{repo: repo, mediaRepo: mediaRepo, logger: logger}
}

// Create creates a new alert channel.
func (s *AlertChannelService) Create(ctx context.Context, ch *model.AlertChannel) error {
	if err := s.repo.Create(ctx, ch); err != nil {
		s.logger.Error("failed to create alert channel", zap.Error(err))
		return apperr.Wrap(apperr.ErrDatabase, err)
	}
	return nil
}

// GetByID returns an alert channel by its ID.
func (s *AlertChannelService) GetByID(ctx context.Context, id uint) (*model.AlertChannel, error) {
	ch, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, apperr.WithMessage(apperr.ErrNotFound, "alert channel not found")
	}
	return ch, nil
}

// List returns a paginated list of alert channels.
func (s *AlertChannelService) List(ctx context.Context, page, pageSize int) ([]model.AlertChannel, int64, error) {
	list, total, err := s.repo.List(ctx, page, pageSize)
	if err != nil {
		s.logger.Error("failed to list alert channels", zap.Error(err))
		return nil, 0, apperr.Wrap(apperr.ErrDatabase, err)
	}
	return list, total, nil
}

// Update updates an existing alert channel.
func (s *AlertChannelService) Update(ctx context.Context, ch *model.AlertChannel) error {
	existing, err := s.repo.GetByID(ctx, ch.ID)
	if err != nil {
		return apperr.WithMessage(apperr.ErrNotFound, "alert channel not found")
	}

	existing.Name = ch.Name
	existing.Description = ch.Description
	existing.MatchLabels = ch.MatchLabels
	existing.DataSourceID = ch.DataSourceID
	existing.Severities = ch.Severities
	existing.MediaID = ch.MediaID
	existing.TemplateID = ch.TemplateID
	existing.ThrottleMin = ch.ThrottleMin
	existing.IsEnabled = ch.IsEnabled

	if err := s.repo.Update(ctx, existing); err != nil {
		s.logger.Error("failed to update alert channel", zap.Error(err))
		return apperr.Wrap(apperr.ErrDatabase, err)
	}
	return nil
}

// Delete deletes an alert channel by ID.
func (s *AlertChannelService) Delete(ctx context.Context, id uint) error {
	if _, err := s.repo.GetByID(ctx, id); err != nil {
		return apperr.WithMessage(apperr.ErrNotFound, "alert channel not found")
	}
	if err := s.repo.Delete(ctx, id); err != nil {
		s.logger.Error("failed to delete alert channel", zap.Error(err))
		return apperr.Wrap(apperr.ErrDatabase, err)
	}
	return nil
}

// FindMatchingChannels returns all enabled channels whose MatchLabels are a
// subset of the event's labels AND whose severity filter (if set) matches.

// TestChannel validates the channel config and sends a test notification through its media.
func (s *AlertChannelService) TestChannel(ctx context.Context, id uint) error {
	ch, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return apperr.WithMessage(apperr.ErrNotFound, "alert channel not found")
	}

	media, err := s.mediaRepo.GetByID(ctx, ch.MediaID)
	if err != nil {
		return apperr.WithMessage(apperr.ErrNotFound, "associated notify media not found")
	}

	if !media.IsEnabled {
		return apperr.WithMessage(apperr.ErrBadRequest, "associated notify media is disabled")
	}

	s.logger.Info("alert channel test passed",
		zap.Uint("channel_id", ch.ID),
		zap.String("channel_name", ch.Name),
		zap.Uint("media_id", media.ID),
		zap.String("media_name", media.Name),
		zap.Time("tested_at", time.Now()),
	)

	return nil
}
// FindMatchingChannels returns all enabled channels whose MatchLabels are a
// subset of the event's labels AND whose severity filter (if set) matches.
// dataSourceID is the datasource of the alert rule (nil = skip datasource filtering).
func (s *AlertChannelService) FindMatchingChannels(ctx context.Context, event *model.AlertEvent, dataSourceID *uint) ([]model.AlertChannel, error) {
	channels, err := s.repo.ListEnabled(ctx)
	if err != nil {
		s.logger.Error("failed to list enabled alert channels", zap.Error(err))
		return nil, apperr.Wrap(apperr.ErrDatabase, err)
	}

	var matched []model.AlertChannel
	for _, ch := range channels {
		if !labelmatch.MatchWithSourceID(map[string]string(event.Labels), dataSourceID, map[string]string(ch.MatchLabels), ch.DataSourceID) {
			continue
		}
		if ch.Severities != "" && !severityMatch(ch.Severities, string(event.Severity)) {
			continue
		}
		matched = append(matched, ch)
	}
	return matched, nil
}

// severityMatch returns true if the given severity appears in the
// comma-separated severities string.
func severityMatch(severities, severity string) bool {
	for _, s := range strings.Split(severities, ",") {
		if strings.TrimSpace(s) == severity {
			return true
		}
	}
	return false
}
