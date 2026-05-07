package repository

import (
	"context"

	"gorm.io/gorm"

	"github.com/sreagent/sreagent/internal/model"
)

// ExclusionRuleRepository handles CRUD for channel exclusion rules.
type ExclusionRuleRepository struct {
	db *gorm.DB
}

func NewExclusionRuleRepository(db *gorm.DB) *ExclusionRuleRepository {
	return &ExclusionRuleRepository{db: db}
}

func (r *ExclusionRuleRepository) Create(ctx context.Context, rule *model.ChannelExclusionRule) error {
	return r.db.WithContext(ctx).Create(rule).Error
}

func (r *ExclusionRuleRepository) GetByID(ctx context.Context, id uint) (*model.ChannelExclusionRule, error) {
	var rule model.ChannelExclusionRule
	err := r.db.WithContext(ctx).First(&rule, id).Error
	if err != nil {
		return nil, err
	}
	return &rule, nil
}

func (r *ExclusionRuleRepository) ListByChannel(ctx context.Context, channelID uint) ([]model.ChannelExclusionRule, error) {
	var list []model.ChannelExclusionRule
	err := r.db.WithContext(ctx).
		Where("channel_id = ?", channelID).
		Order("priority ASC, id ASC").
		Find(&list).Error
	return list, err
}

func (r *ExclusionRuleRepository) ListEnabledByChannel(ctx context.Context, channelID uint) ([]model.ChannelExclusionRule, error) {
	var list []model.ChannelExclusionRule
	err := r.db.WithContext(ctx).
		Where("channel_id = ? AND is_enabled = ?", channelID, true).
		Order("priority ASC, id ASC").
		Find(&list).Error
	return list, err
}

func (r *ExclusionRuleRepository) Update(ctx context.Context, rule *model.ChannelExclusionRule) error {
	return r.db.WithContext(ctx).Save(rule).Error
}

func (r *ExclusionRuleRepository) Delete(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Delete(&model.ChannelExclusionRule{}, id).Error
}
