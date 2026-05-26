package service

import (
	"context"
	"fmt"

	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/sreagent/sreagent/internal/model"
	apperr "github.com/sreagent/sreagent/internal/pkg/errors"
	"github.com/sreagent/sreagent/internal/repository"
)

// ESIndexPatternService provides business logic for ES index patterns.
type ESIndexPatternService struct {
	repo   *repository.ESIndexPatternRepository
	db     *gorm.DB
	logger *zap.Logger
}

// NewESIndexPatternService creates a new ESIndexPatternService.
func NewESIndexPatternService(
	repo *repository.ESIndexPatternRepository,
	db *gorm.DB,
	logger *zap.Logger,
) *ESIndexPatternService {
	return &ESIndexPatternService{
		repo:   repo,
		db:     db,
		logger: logger,
	}
}

// Create creates a new ES index pattern after verifying fields and uniqueness.
func (s *ESIndexPatternService) Create(ctx context.Context, v *model.ESIndexPattern) error {
	if err := v.Verify(); err != nil {
		return apperr.WithMessage(apperr.ErrInvalidParam, err.Error())
	}

	exists, err := s.repo.ExistsByName(ctx, v.DatasourceID, v.Name, 0)
	if err != nil {
		s.logger.Error("failed to check name uniqueness", zap.Error(err))
		return apperr.Wrap(apperr.ErrDatabase, err)
	}
	if exists {
		return apperr.WithMessage(apperr.ErrConflict, fmt.Sprintf("index pattern %q already exists for this datasource", v.Name))
	}

	if err := s.repo.Create(ctx, v); err != nil {
		s.logger.Error("failed to create ES index pattern", zap.Error(err))
		return apperr.Wrap(apperr.ErrDatabase, err)
	}
	return nil
}

// GetByID returns an ES index pattern by its ID.
func (s *ESIndexPatternService) GetByID(ctx context.Context, id uint) (*model.ESIndexPattern, error) {
	v, err := s.repo.GetByID(ctx, id)
	if err != nil {
		s.logger.Error("failed to get ES index pattern", zap.Uint("id", id), zap.Error(err))
		return nil, apperr.Wrap(apperr.ErrNotFound, err)
	}
	return v, nil
}

// Update updates an existing ES index pattern after verifying fields and uniqueness.
func (s *ESIndexPatternService) Update(ctx context.Context, existing *model.ESIndexPattern, input *model.ESIndexPattern) error {
	if err := input.Verify(); err != nil {
		return apperr.WithMessage(apperr.ErrInvalidParam, err.Error())
	}

	// Preserve immutable fields
	input.ID = existing.ID
	input.CreatedAt = existing.CreatedAt
	input.CreatedBy = existing.CreatedBy

	exists, err := s.repo.ExistsByName(ctx, input.DatasourceID, input.Name, existing.ID)
	if err != nil {
		s.logger.Error("failed to check name uniqueness", zap.Error(err))
		return apperr.Wrap(apperr.ErrDatabase, err)
	}
	if exists {
		return apperr.WithMessage(apperr.ErrConflict, fmt.Sprintf("index pattern %q already exists for this datasource", input.Name))
	}

	if err := s.repo.Update(ctx, input); err != nil {
		s.logger.Error("failed to update ES index pattern", zap.Uint("id", existing.ID), zap.Error(err))
		return apperr.Wrap(apperr.ErrDatabase, err)
	}
	return nil
}

// Delete deletes an ES index pattern by ID. It checks whether any alert rules
// reference this pattern in their rule_config JSON before allowing deletion.
func (s *ESIndexPatternService) Delete(ctx context.Context, id uint) error {
	if _, err := s.repo.GetByID(ctx, id); err != nil {
		return apperr.ErrNotFound
	}

	// Check if any alert rules reference this pattern
	var count int64
	err := s.db.WithContext(ctx).
		Model(&model.AlertRule{}).
		Where("JSON_EXTRACT(rule_config, '$.index_pattern') = ?", id).
		Count(&count).Error
	if err != nil {
		s.logger.Error("failed to check alert rule references", zap.Error(err))
		return apperr.Wrap(apperr.ErrDatabase, err)
	}
	if count > 0 {
		return apperr.WithMessage(apperr.ErrBusiness,
			fmt.Sprintf("cannot delete: %d alert rule(s) reference this index pattern", count))
	}

	if err := s.repo.Delete(ctx, id); err != nil {
		s.logger.Error("failed to delete ES index pattern", zap.Uint("id", id), zap.Error(err))
		return apperr.Wrap(apperr.ErrDatabase, err)
	}
	return nil
}

// List returns ES index patterns filtered by datasource_id.
func (s *ESIndexPatternService) List(ctx context.Context, datasourceID uint) ([]model.ESIndexPattern, error) {
	list, err := s.repo.List(ctx, datasourceID)
	if err != nil {
		s.logger.Error("failed to list ES index patterns", zap.Error(err))
		return nil, apperr.Wrap(apperr.ErrDatabase, err)
	}
	return list, nil
}
