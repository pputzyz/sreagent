package repository

import (
	"context"
	"time"

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

// FindMatchingSubscriptions returns all enabled subscribe rules that match the
// given alert event across all filter dimensions: labels, tag filters with
// operators, severity, datasource, rule ID, and minimum firing duration.
//
// Matching logic per rule (all must pass):
//   - RuleIDs: empty or contains 0 = global (matches all rules); otherwise event.RuleID must be in list
//   - DatasourceIDs: empty = matches all; otherwise event.DataSourceID must be in list (0 in list = wildcard)
//   - MatchLabels: subset match (existing behavior)
//   - TagFilters: operator-based match (==, !=, =~, !~, in, not in)
//   - Severities: comma-separated list match
//   - ForDuration: event must have been firing for at least this many seconds
//
// NOTE: SubscribeRule table is typically small (<500 rows). Full-scan + in-memory
// filter is acceptable. A LIMIT guard prevents unbounded scans.
func (r *SubscribeRuleRepository) FindMatchingSubscriptions(ctx context.Context, event *model.AlertEvent) ([]model.SubscribeRule, error) {
	const maxScanRows = 5000
	var allRules []model.SubscribeRule
	err := r.db.WithContext(ctx).
		Where("is_enabled = ?", true).
		Order("id ASC").
		Limit(maxScanRows).
		Find(&allRules).Error
	if err != nil {
		return nil, err
	}

	labels := map[string]string(event.Labels)
	severity := string(event.Severity)

	var matched []model.SubscribeRule
	for _, rule := range allRules {
		// Check rule ID filter (global subscription when empty or contains 0)
		if len(rule.RuleIDs) > 0 && event.RuleID != nil && !containsUintOrZero(rule.RuleIDs, *event.RuleID) {
			continue
		}

		// Check datasource filter (empty = all datasources)
		if len(rule.DatasourceIDs) > 0 && event.DataSourceID != nil {
			if !containsUintOrZero(rule.DatasourceIDs, *event.DataSourceID) {
				continue
			}
		}

		// Check legacy label matching (subset match)
		if !labelmatch.Match(labels, map[string]string(rule.MatchLabels)) {
			continue
		}

		// Check advanced tag filters with operators
		if len(rule.TagFilters) > 0 && !model.MatchTagFilters(labels, rule.TagFilters) {
			continue
		}

		// Check severity filter
		if rule.Severities != "" {
			if !severityMatches(rule.Severities, severity) {
				continue
			}
		}

		// Check minimum firing duration
		if rule.ForDuration > 0 {
			firingSeconds := time.Since(event.FiredAt).Seconds()
			if firingSeconds < float64(rule.ForDuration) {
				continue
			}
		}

		matched = append(matched, rule)
	}

	return matched, nil
}

// containsUintOrZero returns true if the slice contains 0 (wildcard) or the target value.
func containsUintOrZero(ids []uint, target uint) bool {
	for _, id := range ids {
		if id == 0 || id == target {
			return true
		}
	}
	return false
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
