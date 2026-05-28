package repository

import (
	"context"

	"gorm.io/gorm"

	"github.com/sreagent/sreagent/internal/model"
)

type DataSourceRepository struct {
	db *gorm.DB
}

func NewDataSourceRepository(db *gorm.DB) *DataSourceRepository {
	return &DataSourceRepository{db: db}
}

func (r *DataSourceRepository) Create(ctx context.Context, ds *model.DataSource) error {
	return r.db.WithContext(ctx).Create(ds).Error
}

func (r *DataSourceRepository) GetByID(ctx context.Context, id uint) (*model.DataSource, error) {
	var ds model.DataSource
	err := r.db.WithContext(ctx).First(&ds, id).Error
	if err != nil {
		return nil, err
	}
	return &ds, nil
}

func (r *DataSourceRepository) GetByName(ctx context.Context, name string) (*model.DataSource, error) {
	var ds model.DataSource
	err := r.db.WithContext(ctx).Where("name = ?", name).First(&ds).Error
	if err != nil {
		return nil, err
	}
	return &ds, nil
}

func (r *DataSourceRepository) List(ctx context.Context, dsType string, page, pageSize int) ([]model.DataSource, int64, error) {
	var list []model.DataSource
	var total int64

	query := r.db.WithContext(ctx).Model(&model.DataSource{})
	if dsType != "" {
		query = query.Where("type = ?", dsType)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	if err := query.Offset(offset).Limit(pageSize).Order("id DESC").Find(&list).Error; err != nil {
		return nil, 0, err
	}

	return list, total, nil
}

func (r *DataSourceRepository) Update(ctx context.Context, ds *model.DataSource) error {
	return r.db.WithContext(ctx).Save(ds).Error
}

func (r *DataSourceRepository) Delete(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Delete(&model.DataSource{}, id).Error
}

// UpdateHealthStatus updates only the status and version fields (avoids full Save overwriting concurrent edits).
func (r *DataSourceRepository) UpdateHealthStatus(ctx context.Context, id uint, status model.DataSourceStatus, version string) error {
	updates := map[string]interface{}{"status": status}
	if version != "" {
		updates["version"] = version
	}
	return r.db.WithContext(ctx).Model(&model.DataSource{}).Where("id = ?", id).Updates(updates).Error
}

// ListEnabled returns all enabled datasources.
func (r *DataSourceRepository) ListEnabled(ctx context.Context) ([]model.DataSource, error) {
	var list []model.DataSource
	err := r.db.WithContext(ctx).Where("is_enabled = ?", true).Find(&list).Error
	return list, err
}

// ListEnabledByType returns all enabled datasources of a specific type.
func (r *DataSourceRepository) ListEnabledByType(ctx context.Context, dsType model.DataSourceType) ([]model.DataSource, error) {
	var list []model.DataSource
	err := r.db.WithContext(ctx).
		Where("is_enabled = ? AND type = ?", true, dsType).
		Find(&list).Error
	return list, err
}
