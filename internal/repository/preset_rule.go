package repository

import (
	"context"

	"gorm.io/gorm"

	"github.com/sreagent/sreagent/internal/model"
)

type PresetRuleRepository struct {
	db *gorm.DB
}

func NewPresetRuleRepository(db *gorm.DB) *PresetRuleRepository {
	return &PresetRuleRepository{db: db}
}

func (r *PresetRuleRepository) Create(ctx context.Context, rule *model.PresetRule) error {
	return r.db.WithContext(ctx).Create(rule).Error
}

func (r *PresetRuleRepository) GetByID(ctx context.Context, id uint) (*model.PresetRule, error) {
	var rule model.PresetRule
	err := r.db.WithContext(ctx).First(&rule, id).Error
	if err != nil {
		return nil, err
	}
	return &rule, nil
}

func (r *PresetRuleRepository) GetByName(ctx context.Context, name string) (*model.PresetRule, error) {
	var rule model.PresetRule
	err := r.db.WithContext(ctx).Where("name = ?", name).First(&rule).Error
	if err != nil {
		return nil, err
	}
	return &rule, nil
}

func (r *PresetRuleRepository) Delete(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Delete(&model.PresetRule{}, id).Error
}

func (r *PresetRuleRepository) List(ctx context.Context, category, search string, page, pageSize int) ([]model.PresetRule, int64, error) {
	var list []model.PresetRule
	var total int64

	query := r.db.WithContext(ctx).Model(&model.PresetRule{})
	if category != "" {
		query = query.Where("category = ?", category)
	}
	if search != "" {
		query = query.Where("name LIKE ? OR display_name LIKE ? OR description LIKE ?",
			"%"+search+"%", "%"+search+"%", "%"+search+"%")
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

func (r *PresetRuleRepository) BatchCreate(ctx context.Context, rules []model.PresetRule) error {
	if len(rules) == 0 {
		return nil
	}
	return r.db.WithContext(ctx).CreateInBatches(rules, 100).Error
}

func (r *PresetRuleRepository) Categories(ctx context.Context) ([]string, error) {
	var categories []string
	err := r.db.WithContext(ctx).Model(&model.PresetRule{}).
		Where("category != ''").
		Distinct("category").
		Pluck("category", &categories).Error
	return categories, err
}

func (r *PresetRuleRepository) IncrementUsage(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Model(&model.PresetRule{}).
		Where("id = ?", id).
		UpdateColumn("usage_count", gorm.Expr("usage_count + 1")).Error
}
