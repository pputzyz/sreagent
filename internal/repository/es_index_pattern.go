package repository

import (
	"context"

	"gorm.io/gorm"

	"github.com/sreagent/sreagent/internal/model"
)

// ESIndexPatternRepository handles ES index pattern persistence.
type ESIndexPatternRepository struct {
	db *gorm.DB
}

// NewESIndexPatternRepository creates a new ESIndexPatternRepository.
func NewESIndexPatternRepository(db *gorm.DB) *ESIndexPatternRepository {
	return &ESIndexPatternRepository{db: db}
}

// Create creates a new ES index pattern.
func (r *ESIndexPatternRepository) Create(ctx context.Context, v *model.ESIndexPattern) error {
	return r.db.WithContext(ctx).Create(v).Error
}

// GetByID returns an ES index pattern by its ID.
func (r *ESIndexPatternRepository) GetByID(ctx context.Context, id uint) (*model.ESIndexPattern, error) {
	var v model.ESIndexPattern
	if err := r.db.WithContext(ctx).First(&v, id).Error; err != nil {
		return nil, err
	}
	return &v, nil
}

// Update updates an existing ES index pattern.
func (r *ESIndexPatternRepository) Update(ctx context.Context, v *model.ESIndexPattern) error {
	return r.db.WithContext(ctx).Save(v).Error
}

// Delete soft-deletes an ES index pattern by ID.
func (r *ESIndexPatternRepository) Delete(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Delete(&model.ESIndexPattern{}, id).Error
}

// List returns ES index patterns filtered by datasource_id.
// If datasourceID is 0, returns all patterns.
func (r *ESIndexPatternRepository) List(ctx context.Context, datasourceID uint) ([]model.ESIndexPattern, error) {
	var list []model.ESIndexPattern
	query := r.db.WithContext(ctx)
	if datasourceID > 0 {
		query = query.Where("datasource_id = ?", datasourceID)
	}
	if err := query.Order("name ASC").Find(&list).Error; err != nil {
		return nil, err
	}
	return list, nil
}

// ExistsByName checks if a pattern with the given name already exists for the
// datasource, optionally excluding a specific ID (for update checks).
func (r *ESIndexPatternRepository) ExistsByName(ctx context.Context, datasourceID uint, name string, excludeID uint) (bool, error) {
	var count int64
	query := r.db.WithContext(ctx).Model(&model.ESIndexPattern{}).
		Where("datasource_id = ? AND name = ?", datasourceID, name)
	if excludeID > 0 {
		query = query.Where("id != ?", excludeID)
	}
	if err := query.Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}
