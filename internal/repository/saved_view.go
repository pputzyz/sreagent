package repository

import (
	"context"

	"gorm.io/gorm"

	"github.com/sreagent/sreagent/internal/model"
)

// SavedViewRepository handles saved views persistence.
type SavedViewRepository struct {
	db *gorm.DB
}

// NewSavedViewRepository creates a new SavedViewRepository.
func NewSavedViewRepository(db *gorm.DB) *SavedViewRepository {
	return &SavedViewRepository{db: db}
}

// Create creates a new saved view.
func (r *SavedViewRepository) Create(ctx context.Context, v *model.SavedView) error {
	return r.db.WithContext(ctx).Create(v).Error
}

// GetByID returns a saved view by its ID.
func (r *SavedViewRepository) GetByID(ctx context.Context, id uint) (*model.SavedView, error) {
	var v model.SavedView
	err := r.db.WithContext(ctx).First(&v, id).Error
	if err != nil {
		return nil, err
	}
	return &v, nil
}

// Update updates an existing saved view.
func (r *SavedViewRepository) Update(ctx context.Context, v *model.SavedView) error {
	return r.db.WithContext(ctx).Save(v).Error
}

// Delete soft-deletes a saved view by ID.
func (r *SavedViewRepository) Delete(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Delete(&model.SavedView{}, id).Error
}

// ListQuery holds optional filters for listing saved views.
type ListQuery struct {
	Tab       string
	IsPublic  *bool
	CreatedBy uint
}

// List returns a paginated, filtered list of saved views.
func (r *SavedViewRepository) List(ctx context.Context, q ListQuery, page, pageSize int) ([]model.SavedView, int64, error) {
	var list []model.SavedView
	var total int64

	query := r.db.WithContext(ctx).Model(&model.SavedView{})

	if q.Tab != "" {
		query = query.Where("tab = ?", q.Tab)
	}
	if q.IsPublic != nil {
		query = query.Where("is_public = ?", *q.IsPublic)
	}
	if q.CreatedBy > 0 {
		query = query.Where("created_by = ?", q.CreatedBy)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	if err := query.Order("updated_at DESC").Offset(offset).Limit(pageSize).Find(&list).Error; err != nil {
		return nil, 0, err
	}

	return list, total, nil
}
