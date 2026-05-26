package repository

import (
	"context"
	"fmt"

	"gorm.io/gorm"

	"github.com/sreagent/sreagent/internal/model"
)

type BuiltinDashboardRepository struct {
	db *gorm.DB
}

func NewBuiltinDashboardRepository(db *gorm.DB) *BuiltinDashboardRepository {
	return &BuiltinDashboardRepository{db: db}
}

func (r *BuiltinDashboardRepository) Create(ctx context.Context, d *model.BuiltinDashboard) error {
	return r.db.WithContext(ctx).Create(d).Error
}

func (r *BuiltinDashboardRepository) GetByID(ctx context.Context, id uint) (*model.BuiltinDashboard, error) {
	var d model.BuiltinDashboard
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&d).Error; err != nil {
		return nil, err
	}
	return &d, nil
}

func (r *BuiltinDashboardRepository) GetByIdent(ctx context.Context, ident string) (*model.BuiltinDashboard, error) {
	var d model.BuiltinDashboard
	if err := r.db.WithContext(ctx).Where("ident = ?", ident).First(&d).Error; err != nil {
		return nil, err
	}
	return &d, nil
}

func (r *BuiltinDashboardRepository) List(ctx context.Context, category, component, query string, page, pageSize int) ([]model.BuiltinDashboard, int64, error) {
	var list []model.BuiltinDashboard
	var total int64

	q := r.db.WithContext(ctx).Model(&model.BuiltinDashboard{})
	if category != "" {
		q = q.Where("category = ?", category)
	}
	if component != "" {
		q = q.Where("component = ?", component)
	}
	if query != "" {
		like := fmt.Sprintf("%%%s%%", query)
		q = q.Where("(name LIKE ? OR ident LIKE ? OR tags LIKE ?)", like, like, like)
	}

	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	if err := q.Order("category, name").Offset(offset).Limit(pageSize).Find(&list).Error; err != nil {
		return nil, 0, err
	}

	return list, total, nil
}

func (r *BuiltinDashboardRepository) ListAll(ctx context.Context) ([]model.BuiltinDashboard, error) {
	var list []model.BuiltinDashboard
	if err := r.db.WithContext(ctx).Order("category, name").Find(&list).Error; err != nil {
		return nil, err
	}
	return list, nil
}

func (r *BuiltinDashboardRepository) Count(ctx context.Context) (int64, error) {
	var count int64
	if err := r.db.WithContext(ctx).Model(&model.BuiltinDashboard{}).Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

func (r *BuiltinDashboardRepository) GetCategories(ctx context.Context) ([]string, error) {
	var categories []string
	if err := r.db.WithContext(ctx).Model(&model.BuiltinDashboard{}).Distinct("category").Pluck("category", &categories).Error; err != nil {
		return nil, err
	}
	return categories, nil
}

func (r *BuiltinDashboardRepository) GetComponents(ctx context.Context) ([]string, error) {
	var components []string
	if err := r.db.WithContext(ctx).Model(&model.BuiltinDashboard{}).Distinct("component").Pluck("component", &components).Error; err != nil {
		return nil, err
	}
	return components, nil
}

func (r *BuiltinDashboardRepository) Update(ctx context.Context, d *model.BuiltinDashboard) error {
	return r.db.WithContext(ctx).Model(d).Select("*").Updates(d).Error
}

func (r *BuiltinDashboardRepository) Delete(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Delete(&model.BuiltinDashboard{}, id).Error
}
