package service

import (
	"context"

	"go.uber.org/zap"

	"github.com/sreagent/sreagent/internal/model"
	apperr "github.com/sreagent/sreagent/internal/pkg/errors"
	"github.com/sreagent/sreagent/internal/repository"
)

type UserTeamNotifyPrefService struct {
	repo   *repository.UserTeamNotifyPrefRepository
	logger *zap.Logger
}

func NewUserTeamNotifyPrefService(repo *repository.UserTeamNotifyPrefRepository, logger *zap.Logger) *UserTeamNotifyPrefService {
	return &UserTeamNotifyPrefService{repo: repo, logger: logger}
}

func (s *UserTeamNotifyPrefService) Upsert(ctx context.Context, pref *model.UserTeamNotifyPref) error {
	existing, _ := s.repo.GetByUserTeamMedia(ctx, pref.UserID, pref.TeamID, pref.MediaID)
	if existing != nil {
		existing.IsMuted = pref.IsMuted
		return s.repo.Update(ctx, existing)
	}
	return s.repo.Create(ctx, pref)
}

func (s *UserTeamNotifyPrefService) ListByUser(ctx context.Context, userID uint) ([]model.UserTeamNotifyPref, error) {
	return s.repo.ListByUser(ctx, userID)
}

func (s *UserTeamNotifyPrefService) Delete(ctx context.Context, id, userID uint) error {
	if err := s.repo.DeleteByUser(ctx, id, userID); err != nil {
		return apperr.Wrap(apperr.ErrNotFound, err)
	}
	return nil
}
