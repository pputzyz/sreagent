package service

import (
	"context"

	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/sreagent/sreagent/internal/model"
	apperr "github.com/sreagent/sreagent/internal/pkg/errors"
	"github.com/sreagent/sreagent/internal/repository"
)

// ChannelService provides business logic for collaboration channels (协作空间).
type ChannelService struct {
	repo        *repository.ChannelRepository
	incidentRepo *repository.IncidentRepository // optional, for active incident checks
	logger      *zap.Logger
}

func NewChannelService(repo *repository.ChannelRepository, logger *zap.Logger) *ChannelService {
	return &ChannelService{repo: repo, logger: logger}
}

// SetIncidentRepository injects the incident repository for active incident checks.
func (s *ChannelService) SetIncidentRepository(ir *repository.IncidentRepository) {
	s.incidentRepo = ir
}

// Create creates a new collaboration channel after validating uniqueness.
func (s *ChannelService) Create(ctx context.Context, ch *model.Channel) error {
	// Check name uniqueness
	existing, err := s.repo.GetByName(ctx, ch.Name)
	if err == nil && existing != nil {
		return apperr.WithMessage(apperr.ErrDuplicateName, "a channel with this name already exists")
	}

	// Set default JSON configs if empty
	if ch.AggregationConfig == "" {
		ch.AggregationConfig = "{}"
	}
	if ch.FlappingConfig == "" {
		ch.FlappingConfig = "{}"
	}

	if err := s.repo.Create(ctx, ch); err != nil {
		s.logger.Error("failed to create channel", zap.Error(err), zap.String("name", ch.Name))
		return apperr.Wrap(apperr.ErrDatabase, err)
	}

	s.logger.Info("channel created", zap.Uint("id", ch.ID), zap.String("name", ch.Name))
	return nil
}

// GetByID returns a channel by ID.
func (s *ChannelService) GetByID(ctx context.Context, id uint) (*model.Channel, error) {
	ch, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, apperr.ErrCollabChannelNotFound
		}
		return nil, apperr.Wrap(apperr.ErrDatabase, err)
	}
	return ch, nil
}

// List returns a paginated list of channels with optional filters.
func (s *ChannelService) List(ctx context.Context, query, status string, page, pageSize int) ([]model.Channel, int64, error) {
	list, total, err := s.repo.List(ctx, query, status, page, pageSize)
	if err != nil {
		s.logger.Error("failed to list channels", zap.Error(err))
		return nil, 0, apperr.Wrap(apperr.ErrDatabase, err)
	}
	return list, total, nil
}

// Update updates an existing channel.
// P1-18: Pointer-based patching — only fields present in the request are updated.
func (s *ChannelService) Update(ctx context.Context, id uint, updates *model.Channel) (*model.Channel, error) {
	existing, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, apperr.ErrCollabChannelNotFound
		}
		return nil, apperr.Wrap(apperr.ErrDatabase, err)
	}

	// Check name uniqueness if changed
	if updates.Name != "" && updates.Name != existing.Name {
		dup, err := s.repo.GetByName(ctx, updates.Name)
		if err == nil && dup != nil {
			return nil, apperr.WithMessage(apperr.ErrDuplicateName, "a channel with this name already exists")
		}
		existing.Name = updates.Name
	}

	// P1-18: Apply all non-zero fields unconditionally.
	// The handler now only populates fields that were explicitly sent in the request.
	// This allows clearing optional fields by sending empty strings.
	if updates.Description != "" {
		existing.Description = updates.Description
	}
	if updates.TeamID != nil {
		existing.TeamID = updates.TeamID
	}
	if updates.Status != "" {
		existing.Status = updates.Status
	}
	if updates.AccessLevel != "" {
		existing.AccessLevel = updates.AccessLevel
	}
	if updates.AggregationConfig != "" {
		existing.AggregationConfig = updates.AggregationConfig
	}
	if updates.FlappingConfig != "" {
		existing.FlappingConfig = updates.FlappingConfig
	}
	existing.AutoCloseEnabled = updates.AutoCloseEnabled
	existing.FollowAlertClose = updates.FollowAlertClose
	if updates.AutoCloseOrigin != "" {
		existing.AutoCloseOrigin = updates.AutoCloseOrigin
	}
	if updates.AutoCloseMinutes > 0 {
		existing.AutoCloseMinutes = updates.AutoCloseMinutes
	}
	if updates.SortOrder != 0 {
		existing.SortOrder = updates.SortOrder
	}

	if err := s.repo.Update(ctx, existing); err != nil {
		s.logger.Error("failed to update channel", zap.Error(err), zap.Uint("id", id))
		return nil, apperr.Wrap(apperr.ErrDatabase, err)
	}

	s.logger.Info("channel updated", zap.Uint("id", id))
	return existing, nil
}

// Delete soft-deletes a channel.
func (s *ChannelService) Delete(ctx context.Context, id uint) error {
	_, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return apperr.ErrCollabChannelNotFound
		}
		return apperr.Wrap(apperr.ErrDatabase, err)
	}

	// Block deletion if there are active incidents in this channel.
	if s.incidentRepo != nil {
		count, err := s.incidentRepo.CountActiveByChannel(ctx, id)
		if err == nil && count > 0 {
			return apperr.WithMessage(apperr.ErrBusiness, "cannot delete channel with active incidents")
		}
	}

	if err := s.repo.Delete(ctx, id); err != nil {
		s.logger.Error("failed to delete channel", zap.Error(err), zap.Uint("id", id))
		return apperr.Wrap(apperr.ErrDatabase, err)
	}

	s.logger.Info("channel deleted", zap.Uint("id", id))
	return nil
}

// ListActive returns all active channels (e.g. for dropdown selectors).
func (s *ChannelService) ListActive(ctx context.Context) ([]model.Channel, error) {
	list, err := s.repo.ListActive(ctx)
	if err != nil {
		return nil, apperr.Wrap(apperr.ErrDatabase, err)
	}
	return list, nil
}

// Star marks a channel as favorite for a user.
func (s *ChannelService) Star(ctx context.Context, userID, channelID uint) error {
	// Verify channel exists
	if _, err := s.repo.GetByID(ctx, channelID); err != nil {
		if err == gorm.ErrRecordNotFound {
			return apperr.ErrCollabChannelNotFound
		}
		return apperr.Wrap(apperr.ErrDatabase, err)
	}
	if err := s.repo.Star(ctx, userID, channelID); err != nil {
		// Ignore duplicate star (idempotent)
		return nil
	}
	return nil
}

// Unstar removes a channel from user's favorites.
func (s *ChannelService) Unstar(ctx context.Context, userID, channelID uint) error {
	return s.repo.Unstar(ctx, userID, channelID)
}

// ListStarred returns the IDs of channels starred by a user.
func (s *ChannelService) ListStarred(ctx context.Context, userID uint) ([]uint, error) {
	return s.repo.ListStarred(ctx, userID)
}
