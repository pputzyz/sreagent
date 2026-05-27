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

func (s *UserTeamNotifyPrefService) ListByUserTeam(ctx context.Context, userID, teamID uint) ([]model.UserTeamNotifyPref, error) {
	return s.repo.ListByUserTeam(ctx, userID, teamID)
}

func (s *UserTeamNotifyPrefService) Delete(ctx context.Context, id, userID uint) error {
	prefs, err := s.repo.ListByUser(ctx, userID)
	if err != nil {
		return apperr.Wrap(apperr.ErrDatabase, err)
	}
	for _, p := range prefs {
		if p.ID == id {
			return s.repo.Delete(ctx, id)
		}
	}
	return apperr.ErrNotFound
}
