package repository

import (
	"context"

	"gorm.io/gorm"

	"github.com/sreagent/sreagent/internal/model"
)

// MuteRuleRepository handles mute rule persistence.
type MuteRuleRepository struct {
	db *gorm.DB
}

// NewMuteRuleRepository creates a new MuteRuleRepository.
func NewMuteRuleRepository(db *gorm.DB) *MuteRuleRepository {
	return &MuteRuleRepository{db: db}
}

// Create creates a new mute rule.
func (r *MuteRuleRepository) Create(ctx context.Context, rule *model.MuteRule) error {
	return r.db.WithContext(ctx).Create(rule).Error
}

// GetByID returns a mute rule by ID.
func (r *MuteRuleRepository) GetByID(ctx context.Context, id uint) (*model.MuteRule, error) {
	var rule model.MuteRule
	err := r.db.WithContext(ctx).First(&rule, id).Error
	if err != nil {
		return nil, err
	}
	return &rule, nil
}

// List returns a paginated list of mute rules.
func (r *MuteRuleRepository) List(ctx context.Context, page, pageSize int) ([]model.MuteRule, int64, error) {
	var list []model.MuteRule
	var total int64

	query := r.db.WithContext(ctx).Model(&model.MuteRule{})

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	if err := query.Offset(offset).Limit(pageSize).Order("id DESC").Find(&list).Error; err != nil {
		return nil, 0, err
	}

	return list, total, nil
}

// Update performs a column-level update to avoid full-row overwrite (last-writer-wins).
func (r *MuteRuleRepository) Update(ctx context.Context, rule *model.MuteRule) error {
	return r.db.WithContext(ctx).Model(rule).Updates(map[string]interface{}{
		"name":           rule.Name,
		"description":    rule.Description,
		"match_labels":   rule.MatchLabels,
		"severities":     rule.Severities,
		"start_time":     rule.StartTime,
		"end_time":       rule.EndTime,
		"periodic_start": rule.PeriodicStart,
		"periodic_end":   rule.PeriodicEnd,
		"days_of_week":   rule.DaysOfWeek,
		"timezone":       rule.Timezone,
		"rule_ids":       rule.RuleIDs,
		"is_enabled":     rule.IsEnabled,
	}).Error
}

// Delete deletes a mute rule by ID.
func (r *MuteRuleRepository) Delete(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Delete(&model.MuteRule{}, id).Error
}

// FindAllEnabled returns all enabled mute rules.
func (r *MuteRuleRepository) FindAllEnabled(ctx context.Context) ([]model.MuteRule, error) {
	var rules []model.MuteRule
	err := r.db.WithContext(ctx).Where("is_enabled = ?", true).Find(&rules).Error
	return rules, err
}

// BatchUpdateEnabled sets is_enabled for all rules whose IDs are in ids.
func (r *MuteRuleRepository) BatchUpdateEnabled(ctx context.Context, ids []uint, enabled bool) error {
	if len(ids) == 0 {
		return nil
	}
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return tx.Model(&model.MuteRule{}).
			Where("id IN ?", ids).
			Update("is_enabled", enabled).Error
	})
}

// BatchDelete soft-deletes all rules whose IDs are in ids.
func (r *MuteRuleRepository) BatchDelete(ctx context.Context, ids []uint) error {
	if len(ids) == 0 {
		return nil
	}
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return tx.Where("id IN ?", ids).Delete(&model.MuteRule{}).Error
	})
}
