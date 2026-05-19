package repository

import (
	"context"

	"gorm.io/gorm"

	"github.com/sreagent/sreagent/internal/model"
	"github.com/sreagent/sreagent/internal/pkg/labelmatch"
)

// SubscribeRuleRepository handles subscribe_rules persistence.
type SubscribeRuleRepository struct {
	db *gorm.DB
}

// NewSubscribeRuleRepository creates a new SubscribeRuleRepository.
func NewSubscribeRuleRepository(db *gorm.DB) *SubscribeRuleRepository {
	return &SubscribeRuleRepository{db: db}
}

// Create creates a new subscribe rule.
func (r *SubscribeRuleRepository) Create(ctx context.Context, rule *model.SubscribeRule) error {
	return r.db.WithContext(ctx).Create(rule).Error
}

// GetByID returns a subscribe rule by its ID.
func (r *SubscribeRuleRepository) GetByID(ctx context.Context, id uint) (*model.SubscribeRule, error) {
	var rule model.SubscribeRule
	err := r.db.WithContext(ctx).First(&rule, id).Error
	if err != nil {
		return nil, err
	}
	return &rule, nil
}

// List returns a paginated list of subscribe rules.
func (r *SubscribeRuleRepository) List(ctx context.Context, page, pageSize int) ([]model.SubscribeRule, int64, error) {
	var list []model.SubscribeRule
	var total int64

	query := r.db.WithContext(ctx).Model(&model.SubscribeRule{})

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	if err := query.Offset(offset).Limit(pageSize).Order("id DESC").Find(&list).Error; err != nil {
		return nil, 0, err
	}

	return list, total, nil
}

// Update updates an existing subscribe rule.
func (r *SubscribeRuleRepository) Update(ctx context.Context, rule *model.SubscribeRule) error {
	return r.db.WithContext(ctx).Save(rule).Error
}

// Delete soft-deletes a subscribe rule by ID.
func (r *SubscribeRuleRepository) Delete(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Delete(&model.SubscribeRule{}, id).Error
}

// FindMatchingSubscriptions returns all enabled subscribe rules whose match_labels
// are a subset of the given event labels, and whose severity filter matches.
func (r *SubscribeRuleRepository) FindMatchingSubscriptions(ctx context.Context, labels map[string]string, severity string) ([]model.SubscribeRule, error) {
	var allRules []model.SubscribeRule
	err := r.db.WithContext(ctx).
		Where("is_enabled = ?", true).
		Order("id ASC").
		Find(&allRules).Error
	if err != nil {
		return nil, err
	}

	var matched []model.SubscribeRule
	for _, rule := range allRules {
		// Check label matching
		if !labelmatch.Match(labels, map[string]string(rule.MatchLabels)) {
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

// ListByUser returns all subscribe rules for a specific user.
func (r *SubscribeRuleRepository) ListByUser(ctx context.Context, userID uint) ([]model.SubscribeRule, error) {
	var list []model.SubscribeRule
	err := r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Order("id DESC").
		Find(&list).Error
	return list, err
}

// ListByTeam returns all subscribe rules for a specific team.
func (r *SubscribeRuleRepository) ListByTeam(ctx context.Context, teamID uint) ([]model.SubscribeRule, error) {
	var list []model.SubscribeRule
	err := r.db.WithContext(ctx).
		Where("team_id = ?", teamID).
		Order("id DESC").
		Find(&list).Error
	return list, err
}
