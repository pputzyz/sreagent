package repository

import (
	"context"

	"gorm.io/gorm"

	"github.com/sreagent/sreagent/internal/model"
)

// MetricViewRepository handles metric views persistence.
type MetricViewRepository struct {
	db *gorm.DB
}

// NewMetricViewRepository creates a new MetricViewRepository.
func NewMetricViewRepository(db *gorm.DB) *MetricViewRepository {
	return &MetricViewRepository{db: db}
}

// Create creates a new metric view.
func (r *MetricViewRepository) Create(ctx context.Context, v *model.MetricView) error {
	return r.db.WithContext(ctx).Create(v).Error
}

// GetByID returns a metric view by its ID.
func (r *MetricViewRepository) GetByID(ctx context.Context, id uint) (*model.MetricView, error) {
	var v model.MetricView
	if err := r.db.WithContext(ctx).First(&v, id).Error; err != nil {
		return nil, err
	}
	v.DB2FE()
	return &v, nil
}

// Update updates an existing metric view.
func (r *MetricViewRepository) Update(ctx context.Context, v *model.MetricView) error {
	return r.db.WithContext(ctx).Save(v).Error
}

// Delete soft-deletes a metric view by ID.
func (r *MetricViewRepository) Delete(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Delete(&model.MetricView{}, id).Error
}

// List returns metric views filtered by created_by, with pagination.
func (r *MetricViewRepository) List(ctx context.Context, createdBy uint, page, pageSize int) ([]model.MetricView, int64, error) {
	var list []model.MetricView
	var total int64

	query := r.db.WithContext(ctx).Model(&model.MetricView{})
	if createdBy > 0 {
		query = query.Where("created_by = ?", createdBy)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	if err := query.Order("updated_at DESC").Offset(offset).Limit(pageSize).Find(&list).Error; err != nil {
		return nil, 0, err
	}

	for i := range list {
		list[i].DB2FE()
	}

	return list, total, nil
}
