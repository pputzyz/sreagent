package service

import (
	"context"

	"go.uber.org/zap"

	"github.com/sreagent/sreagent/internal/model"
	apperr "github.com/sreagent/sreagent/internal/pkg/errors"
	"github.com/sreagent/sreagent/internal/repository"
)

// MetricViewService provides business logic for metric views.
type MetricViewService struct {
	repo   *repository.MetricViewRepository
	logger *zap.Logger
}

// NewMetricViewService creates a new MetricViewService.
func NewMetricViewService(
	repo *repository.MetricViewRepository,
	logger *zap.Logger,
) *MetricViewService {
	return &MetricViewService{
		repo:   repo,
		logger: logger,
	}
}

// Create creates a new metric view.
func (s *MetricViewService) Create(ctx context.Context, v *model.MetricView) error {
	v.FE2DB()
	if err := v.Verify(); err != nil {
		return apperr.WithMessage(apperr.ErrInvalidParam, err.Error())
	}
	if err := s.repo.Create(ctx, v); err != nil {
		s.logger.Error("failed to create metric view", zap.Error(err))
		return apperr.Wrap(apperr.ErrDatabase, err)
	}
	return nil
}

// GetByID returns a metric view by its ID.
func (s *MetricViewService) GetByID(ctx context.Context, id uint) (*model.MetricView, error) {
	v, err := s.repo.GetByID(ctx, id)
	if err != nil {
		s.logger.Error("failed to get metric view", zap.Uint("id", id), zap.Error(err))
		return nil, apperr.Wrap(apperr.ErrNotFound, err)
	}
	return v, nil
}

// Update updates an existing metric view.
func (s *MetricViewService) Update(ctx context.Context, existing *model.MetricView, input *model.MetricView) error {
	input.FE2DB()
	if err := input.Verify(); err != nil {
		return apperr.WithMessage(apperr.ErrInvalidParam, err.Error())
	}

	// Preserve immutable fields
	input.ID = existing.ID
	input.CreatedBy = existing.CreatedBy
	input.CreatedAt = existing.CreatedAt

	if err := s.repo.Update(ctx, input); err != nil {
		s.logger.Error("failed to update metric view", zap.Uint("id", existing.ID), zap.Error(err))
		return apperr.Wrap(apperr.ErrDatabase, err)
	}
	return nil
}

// Delete deletes a metric view by ID.
func (s *MetricViewService) Delete(ctx context.Context, id uint) error {
	if _, err := s.repo.GetByID(ctx, id); err != nil {
		return apperr.ErrNotFound
	}
	if err := s.repo.Delete(ctx, id); err != nil {
		s.logger.Error("failed to delete metric view", zap.Uint("id", id), zap.Error(err))
		return apperr.Wrap(apperr.ErrDatabase, err)
	}
	return nil
}

// List returns a paginated list of metric views for a user.
func (s *MetricViewService) List(ctx context.Context, createdBy uint, page, pageSize int) ([]model.MetricView, int64, error) {
	list, total, err := s.repo.List(ctx, createdBy, page, pageSize)
	if err != nil {
		s.logger.Error("failed to list metric views", zap.Error(err))
		return nil, 0, apperr.Wrap(apperr.ErrDatabase, err)
	}
	return list, total, nil
}
