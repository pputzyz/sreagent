package service

import (
	"context"

	"go.uber.org/zap"

	"github.com/sreagent/sreagent/internal/model"
	apperr "github.com/sreagent/sreagent/internal/pkg/errors"
	"github.com/sreagent/sreagent/internal/repository"
)

// SavedViewListQuery holds optional filters for listing saved views.
// Defined in service layer so handlers don't need to import repository.
type SavedViewListQuery = repository.ListQuery

// SavedViewService provides business logic for saved views.
type SavedViewService struct {
	repo   *repository.SavedViewRepository
	logger *zap.Logger
}

// NewSavedViewService creates a new SavedViewService.
func NewSavedViewService(
	repo *repository.SavedViewRepository,
	logger *zap.Logger,
) *SavedViewService {
	return &SavedViewService{
		repo:   repo,
		logger: logger,
	}
}

// Create creates a new saved view.
func (s *SavedViewService) Create(ctx context.Context, v *model.SavedView) error {
	if err := s.repo.Create(ctx, v); err != nil {
		s.logger.Error("failed to create saved view", zap.Error(err))
		return apperr.Wrap(apperr.ErrDatabase, err)
	}
	return nil
}

// GetByID returns a saved view by its ID.
func (s *SavedViewService) GetByID(ctx context.Context, id uint) (*model.SavedView, error) {
	v, err := s.repo.GetByID(ctx, id)
	if err != nil {
		s.logger.Error("failed to get saved view", zap.Uint("id", id), zap.Error(err))
		return nil, apperr.Wrap(apperr.ErrNotFound, err)
	}
	return v, nil
}

// Update updates an existing saved view.
func (s *SavedViewService) Update(ctx context.Context, v *model.SavedView) error {
	existing, err := s.repo.GetByID(ctx, v.ID)
	if err != nil {
		return apperr.ErrNotFound
	}

	existing.Name = v.Name
	existing.Description = v.Description
	existing.Tab = v.Tab
	existing.DatasourceID = v.DatasourceID
	existing.Expression = v.Expression
	existing.QueryConfig = v.QueryConfig
	existing.IsPublic = v.IsPublic
	existing.UpdatedBy = v.UpdatedBy

	if err := s.repo.Update(ctx, existing); err != nil {
		s.logger.Error("failed to update saved view", zap.Uint("id", v.ID), zap.Error(err))
		return apperr.Wrap(apperr.ErrDatabase, err)
	}
	return nil
}

// Delete deletes a saved view by ID.
func (s *SavedViewService) Delete(ctx context.Context, id uint) error {
	if _, err := s.repo.GetByID(ctx, id); err != nil {
		return apperr.ErrNotFound
	}
	if err := s.repo.Delete(ctx, id); err != nil {
		s.logger.Error("failed to delete saved view", zap.Uint("id", id), zap.Error(err))
		return apperr.Wrap(apperr.ErrDatabase, err)
	}
	return nil
}

// List returns a paginated, filtered list of saved views.
func (s *SavedViewService) List(ctx context.Context, q repository.ListQuery, page, pageSize int) ([]model.SavedView, int64, error) {
	list, total, err := s.repo.List(ctx, q, page, pageSize)
	if err != nil {
		s.logger.Error("failed to list saved views", zap.Error(err))
		return nil, 0, apperr.Wrap(apperr.ErrDatabase, err)
	}
	return list, total, nil
}

// Copy clones an existing saved view with a new name.
func (s *SavedViewService) Copy(ctx context.Context, id uint, userID uint) (*model.SavedView, error) {
	original, err := s.repo.GetByID(ctx, id)
	if err != nil {
		s.logger.Error("failed to get saved view for copy", zap.Uint("id", id), zap.Error(err))
		return nil, apperr.Wrap(apperr.ErrNotFound, err)
	}

	clone := &model.SavedView{
		Name:         original.Name + " (copy)",
		Description:  original.Description,
		Tab:          original.Tab,
		DatasourceID: original.DatasourceID,
		Expression:   original.Expression,
		QueryConfig:  original.QueryConfig,
		IsPublic:     false, // cloned views are always private
		CreatedBy:    userID,
		UpdatedBy:    userID,
	}

	if err := s.repo.Create(ctx, clone); err != nil {
		s.logger.Error("failed to copy saved view", zap.Uint("id", id), zap.Error(err))
		return nil, apperr.Wrap(apperr.ErrDatabase, err)
	}
	return clone, nil
}
