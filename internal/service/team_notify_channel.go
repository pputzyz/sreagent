package service

import (
	"context"

	"go.uber.org/zap"

	"github.com/sreagent/sreagent/internal/model"
	apperr "github.com/sreagent/sreagent/internal/pkg/errors"
	"github.com/sreagent/sreagent/internal/repository"
)

type TeamNotifyChannelService struct {
	repo      *repository.TeamNotifyChannelRepository
	mediaRepo *repository.NotifyMediaRepository
	logger    *zap.Logger
}

func NewTeamNotifyChannelService(
	repo *repository.TeamNotifyChannelRepository,
	mediaRepo *repository.NotifyMediaRepository,
	logger *zap.Logger,
) *TeamNotifyChannelService {
	return &TeamNotifyChannelService{repo: repo, mediaRepo: mediaRepo, logger: logger}
}

func (s *TeamNotifyChannelService) Create(ctx context.Context, ch *model.TeamNotifyChannel) error {
	if _, err := s.mediaRepo.GetByID(ctx, ch.MediaID); err != nil {
		return apperr.WithMessage(apperr.ErrNotFound, "notification media not found")
	}
	if err := s.repo.Create(ctx, ch); err != nil {
		return apperr.Wrap(apperr.ErrDatabase, err)
	}
	return nil
}

func (s *TeamNotifyChannelService) GetByID(ctx context.Context, id uint) (*model.TeamNotifyChannel, error) {
	ch, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, apperr.ErrNotFound
	}
	return ch, nil
}

func (s *TeamNotifyChannelService) ListByTeam(ctx context.Context, teamID uint) ([]model.TeamNotifyChannel, error) {
	return s.repo.ListByTeam(ctx, teamID)
}

func (s *TeamNotifyChannelService) Update(ctx context.Context, ch *model.TeamNotifyChannel) error {
	if _, err := s.repo.GetByID(ctx, ch.ID); err != nil {
		return apperr.ErrNotFound
	}
	if err := s.repo.Update(ctx, ch); err != nil {
		return apperr.Wrap(apperr.ErrDatabase, err)
	}
	return nil
}

func (s *TeamNotifyChannelService) SetDefault(ctx context.Context, id uint) error {
	ch, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return apperr.ErrNotFound
	}
	if err := s.repo.ClearDefault(ctx, ch.TeamID); err != nil {
		return apperr.Wrap(apperr.ErrDatabase, err)
	}
	ch.IsDefault = true
	return s.repo.Update(ctx, ch)
}

func (s *TeamNotifyChannelService) Delete(ctx context.Context, id uint) error {
	if err := s.repo.Delete(ctx, id); err != nil {
		return apperr.Wrap(apperr.ErrDatabase, err)
	}
	return nil
}
