package service

import (
	"context"
	"time"

	"go.uber.org/zap"

	"github.com/sreagent/sreagent/internal/model"
	apperr "github.com/sreagent/sreagent/internal/pkg/errors"
	"github.com/sreagent/sreagent/internal/repository"
)

// AnnotationService provides business logic for dashboard annotations.
type AnnotationService struct {
	repo   *repository.AnnotationRepository
	logger *zap.Logger
}

// NewAnnotationService creates a new AnnotationService.
func NewAnnotationService(
	repo *repository.AnnotationRepository,
	logger *zap.Logger,
) *AnnotationService {
	return &AnnotationService{
		repo:   repo,
		logger: logger,
	}
}

// Create creates a new annotation.
func (s *AnnotationService) Create(ctx context.Context, annotation *model.Annotation) error {
	if err := s.repo.Create(ctx, annotation); err != nil {
		s.logger.Error("failed to create annotation", zap.Error(err))
		return apperr.Wrap(apperr.ErrDatabase, err)
	}
	return nil
}

// GetByID returns an annotation by its ID.
func (s *AnnotationService) GetByID(ctx context.Context, id uint) (*model.Annotation, error) {
	ann, err := s.repo.GetByID(ctx, id)
	if err != nil {
		s.logger.Error("failed to get annotation", zap.Uint("id", id), zap.Error(err))
		return nil, apperr.Wrap(apperr.ErrDatabase, err)
	}
	return ann, nil
}

// Update updates an existing annotation.
func (s *AnnotationService) Update(ctx context.Context, annotation *model.Annotation) error {
	if err := s.repo.Update(ctx, annotation); err != nil {
		s.logger.Error("failed to update annotation", zap.Uint("id", annotation.ID), zap.Error(err))
		return apperr.Wrap(apperr.ErrDatabase, err)
	}
	return nil
}

// Delete deletes an annotation by ID.
func (s *AnnotationService) Delete(ctx context.Context, id uint) error {
	if err := s.repo.Delete(ctx, id); err != nil {
		s.logger.Error("failed to delete annotation", zap.Uint("id", id), zap.Error(err))
		return apperr.Wrap(apperr.ErrDatabase, err)
	}
	return nil
}

// ListByDashboard returns annotations for a dashboard within a time range.
func (s *AnnotationService) ListByDashboard(ctx context.Context, dashboardID uint, from, to time.Time) ([]model.Annotation, error) {
	list, err := s.repo.ListByDashboard(ctx, dashboardID, from, to)
	if err != nil {
		s.logger.Error("failed to list annotations",
			zap.Uint("dashboard_id", dashboardID),
			zap.Error(err),
		)
		return nil, apperr.Wrap(apperr.ErrDatabase, err)
	}
	return list, nil
}

// List returns annotations with optional filters and pagination.
func (s *AnnotationService) List(ctx context.Context, dashboardID uint, from, to time.Time, page, pageSize uint) ([]model.Annotation, int64, error) {
	list, total, err := s.repo.List(ctx, dashboardID, from, to, page, pageSize)
	if err != nil {
		s.logger.Error("failed to list annotations", zap.Error(err))
		return nil, 0, apperr.Wrap(apperr.ErrDatabase, err)
	}
	return list, total, nil
}

// BatchCreate inserts multiple annotations at once.
func (s *AnnotationService) BatchCreate(ctx context.Context, annotations []model.Annotation) error {
	if len(annotations) == 0 {
		return nil
	}
	if err := s.repo.BatchCreate(ctx, annotations); err != nil {
		s.logger.Error("failed to batch create annotations", zap.Int("count", len(annotations)), zap.Error(err))
		return apperr.Wrap(apperr.ErrDatabase, err)
	}
	return nil
}
