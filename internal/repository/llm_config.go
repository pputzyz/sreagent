package repository

import (
	"context"

	"gorm.io/gorm"

	"github.com/sreagent/sreagent/internal/model"
)

// LLMConfigRepository handles LLM config persistence.
type LLMConfigRepository struct {
	db *gorm.DB
}

// NewLLMConfigRepository creates a new LLMConfigRepository.
func NewLLMConfigRepository(db *gorm.DB) *LLMConfigRepository {
	return &LLMConfigRepository{db: db}
}

// Create creates a new LLM config.
func (r *LLMConfigRepository) Create(ctx context.Context, v *model.LLMConfig) error {
	return r.db.WithContext(ctx).Create(v).Error
}

// GetByID returns an LLM config by its ID.
func (r *LLMConfigRepository) GetByID(ctx context.Context, id uint) (*model.LLMConfig, error) {
	var v model.LLMConfig
	if err := r.db.WithContext(ctx).First(&v, id).Error; err != nil {
		return nil, err
	}
	return &v, nil
}

// Update updates an existing LLM config.
func (r *LLMConfigRepository) Update(ctx context.Context, v *model.LLMConfig) error {
	return r.db.WithContext(ctx).Save(v).Error
}

// Delete soft-deletes an LLM config by ID.
func (r *LLMConfigRepository) Delete(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Delete(&model.LLMConfig{}, id).Error
}

// List returns all LLM configs with pagination, ordered by name.
func (r *LLMConfigRepository) List(ctx context.Context, page, pageSize int) ([]model.LLMConfig, int64, error) {
	var list []model.LLMConfig
	var total int64

	query := r.db.WithContext(ctx).Model(&model.LLMConfig{})

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	if err := query.Order("is_default DESC, name ASC").Offset(offset).Limit(pageSize).Find(&list).Error; err != nil {
		return nil, 0, err
	}

	return list, total, nil
}

// PickDefault returns the config where is_default=true, or nil if none is set.
func (r *LLMConfigRepository) PickDefault(ctx context.Context) (*model.LLMConfig, error) {
	var v model.LLMConfig
	err := r.db.WithContext(ctx).Where("is_default = ?", true).First(&v).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &v, nil
}

// ClearDefault sets is_default=false for all configs. Caller must pass a
// transaction handle so this can be combined with the update that sets a
// new default atomically.
func (r *LLMConfigRepository) ClearDefault(tx *gorm.DB) error {
	return tx.Model(&model.LLMConfig{}).Where("is_default = ?", true).Update("is_default", false).Error
}
