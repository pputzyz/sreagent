package repository

import (
	"context"

	"gorm.io/gorm"

	"github.com/sreagent/sreagent/internal/model"
)

// NotifyRuleRepository handles notify_rules persistence.
type NotifyRuleRepository struct {
	db *gorm.DB
}

// NewNotifyRuleRepository creates a new NotifyRuleRepository.
func NewNotifyRuleRepository(db *gorm.DB) *NotifyRuleRepository {
	return &NotifyRuleRepository{db: db}
}

// Create creates a new notify rule.
func (r *NotifyRuleRepository) Create(ctx context.Context, rule *model.NotifyRule) error {
	return r.db.WithContext(ctx).Create(rule).Error
}

// GetByID returns a notify rule by its ID.
func (r *NotifyRuleRepository) GetByID(ctx context.Context, id uint) (*model.NotifyRule, error) {
	var rule model.NotifyRule
	err := r.db.WithContext(ctx).First(&rule, id).Error
	if err != nil {
		return nil, err
	}
	return &rule, nil
}

// List returns a paginated list of notify rules.
func (r *NotifyRuleRepository) List(ctx context.Context, page, pageSize int) ([]model.NotifyRule, int64, error) {
	var list []model.NotifyRule
	var total int64

	query := r.db.WithContext(ctx).Model(&model.NotifyRule{})

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	if err := query.Offset(offset).Limit(pageSize).Order("id DESC").Find(&list).Error; err != nil {
		return nil, 0, err
	}

	return list, total, nil
}

// Update updates an existing notify rule.
func (r *NotifyRuleRepository) Update(ctx context.Context, rule *model.NotifyRule) error {
	return r.db.WithContext(ctx).Save(rule).Error
}

// Delete soft-deletes a notify rule by ID.
func (r *NotifyRuleRepository) Delete(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Delete(&model.NotifyRule{}, id).Error
}

// ListEnabled returns all enabled notify rules.
func (r *NotifyRuleRepository) ListEnabled(ctx context.Context) ([]model.NotifyRule, error) {
	var list []model.NotifyRule
	err := r.db.WithContext(ctx).
		Where("is_enabled = ?", true).
		Order("id ASC").
		Find(&list).Error
	return list, err
}

// BatchUpdateEnabled sets is_enabled for all rules whose IDs are in ids.
func (r *NotifyRuleRepository) BatchUpdateEnabled(ctx context.Context, ids []uint, enabled bool) error {
	if len(ids) == 0 {
		return nil
	}
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return tx.Model(&model.NotifyRule{}).
			Where("id IN ?", ids).
			Update("is_enabled", enabled).Error
	})
}

// BatchDelete soft-deletes all rules whose IDs are in ids.
func (r *NotifyRuleRepository) BatchDelete(ctx context.Context, ids []uint) error {
	if len(ids) == 0 {
		return nil
	}
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return tx.Where("id IN ?", ids).Delete(&model.NotifyRule{}).Error
	})
}

// FindMatchingRules returns all enabled notify rules whose match_labels are a subset
// of the given event labels, and whose severity filter matches (or is empty for all).
func (r *NotifyRuleRepository) FindMatchingRules(ctx context.Context, labels map[string]string, severity string) ([]model.NotifyRule, error) {
	allRules, err := r.ListEnabled(ctx)
	if err != nil {
		return nil, err
	}

	var matched []model.NotifyRule
	for _, rule := range allRules {
		// Check label matching
		if !labelsMatch(rule.MatchLabels, labels) {
			continue
		}

		// Check severity filter
		if rule.Severities != "" {
			if !severityMatches(rule.Severities, severity) {
				continue
			}
		}

		matched = append(matched, rule)
	}

	return matched, nil
}
