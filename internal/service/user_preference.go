package service

import (
	"context"

	"go.uber.org/zap"

	"github.com/sreagent/sreagent/internal/model"
	"github.com/sreagent/sreagent/internal/repository"
)

type UserPreferenceService struct {
	repo   *repository.UserPreferenceRepository
	logger *zap.Logger
}

func NewUserPreferenceService(repo *repository.UserPreferenceRepository, logger *zap.Logger) *UserPreferenceService {
	return &UserPreferenceService{repo: repo, logger: logger}
}

func (s *UserPreferenceService) Get(ctx context.Context, userID uint) (*model.UserPreference, error) {
	return s.repo.GetByUserID(ctx, userID)
}

func (s *UserPreferenceService) Update(ctx context.Context, pref *model.UserPreference) error {
	return s.repo.Upsert(ctx, pref)
}
