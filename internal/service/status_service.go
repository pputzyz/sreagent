package service

import (
	"context"
	"errors"

	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/sreagent/sreagent/internal/model"
	apperr "github.com/sreagent/sreagent/internal/pkg/errors"
	"github.com/sreagent/sreagent/internal/repository"
)

type StatusServiceService struct {
	repo   *repository.StatusServiceRepository
	logger *zap.Logger
}

func NewStatusServiceService(repo *repository.StatusServiceRepository, logger *zap.Logger) *StatusServiceService {
	return &StatusServiceService{repo: repo, logger: logger}
}

func (s *StatusServiceService) Create(ctx context.Context, svc *model.StatusService) error {
	if err := s.repo.Create(ctx, svc); err != nil {
		s.logger.Error("failed to create status service", zap.Error(err))
		return apperr.Wrap(apperr.ErrDatabase, err)
	}
	return nil
}

func (s *StatusServiceService) GetByID(ctx context.Context, id uint) (*model.StatusService, error) {
	svc, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperr.Wrap(apperr.ErrNotFound, err)
		}
		s.logger.Error("failed to get status service", zap.Error(err))
		return nil, apperr.Wrap(apperr.ErrDatabase, err)
	}
	return svc, nil
}

func (s *StatusServiceService) Update(ctx context.Context, svc *model.StatusService) error {
	if err := s.repo.Update(ctx, svc); err != nil {
		s.logger.Error("failed to update status service", zap.Error(err))
		return apperr.Wrap(apperr.ErrDatabase, err)
	}
	return nil
}

func (s *StatusServiceService) Delete(ctx context.Context, id uint) error {
	if err := s.repo.Delete(ctx, id); err != nil {
		s.logger.Error("failed to delete status service", zap.Error(err))
		return apperr.Wrap(apperr.ErrDatabase, err)
	}
	return nil
}

func (s *StatusServiceService) List(ctx context.Context) ([]model.StatusService, error) {
	services, err := s.repo.List(ctx)
	if err != nil {
		s.logger.Error("failed to list status services", zap.Error(err))
		return nil, apperr.Wrap(apperr.ErrDatabase, err)
	}
	return services, nil
}
