package repository

import (
	"context"

	"gorm.io/gorm"

	"github.com/sreagent/sreagent/internal/model"
)

type AlertRuleRepository struct {
	db *gorm.DB
}

func NewAlertRuleRepository(db *gorm.DB) *AlertRuleRepository {
	return &AlertRuleRepository{db: db}
}

func (r *AlertRuleRepository) Create(ctx context.Context, rule *model.AlertRule) error {
	return r.db.WithContext(ctx).Create(rule).Error
}

func (r *AlertRuleRepository) GetByID(ctx context.Context, id uint) (*model.AlertRule, error) {
	var rule model.AlertRule
	err := r.db.WithContext(ctx).Preload("DataSource").First(&rule, id).Error
	if err != nil {
		return nil, err
	}
	return &rule, nil
}

func (r *AlertRuleRepository) List(ctx context.Context, severity, status, groupName, category string, page, pageSize int) ([]model.AlertRule, int64, error) {
	var list []model.AlertRule
	var total int64

	query := r.db.WithContext(ctx).Model(&model.AlertRule{})
	if severity != "" {
		query = query.Where("severity = ?", severity)
	}
	if status != "" {
		query = query.Where("status = ?", status)
	}
	if groupName != "" {
		query = query.Where("group_name = ?", groupName)
	}
	if category != "" {
		query = query.Where("category = ?", category)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	if err := query.Preload("DataSource").Offset(offset).Limit(pageSize).Order("id DESC").Find(&list).Error; err != nil {
		return nil, 0, err
	}

	return list, total, nil
}

// ListHeartbeat returns all heartbeat-type alert rules (regardless of pagination).
func (r *AlertRuleRepository) ListHeartbeat(ctx context.Context) ([]model.AlertRule, int64, error) {
	var list []model.AlertRule
	var total int64
	q := r.db.WithContext(ctx).Model(&model.AlertRule{}).Where("rule_type = ?", model.RuleTypeHeartbeat)
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if err := q.Preload("DataSource").Find(&list).Error; err != nil {
		return nil, 0, err
	}
	return list, total, nil
}

// GetByHeartbeatToken returns the rule whose heartbeat_token matches the given token.
func (r *AlertRuleRepository) GetByHeartbeatToken(ctx context.Context, token string) (*model.AlertRule, error) {
	var rule model.AlertRule
	err := r.db.WithContext(ctx).
		Where("heartbeat_token = ? AND rule_type = ?", token, model.RuleTypeHeartbeat).
		First(&rule).Error
	if err != nil {
		return nil, err
	}
	return &rule, nil
}

// ListCategories returns all distinct non-empty category values.
func (r *AlertRuleRepository) ListCategories(ctx context.Context) ([]string, error) {
	var categories []string
	err := r.db.WithContext(ctx).Model(&model.AlertRule{}).
		Where("category != '' AND deleted_at IS NULL").
		Distinct("category").Pluck("category", &categories).Error
	return categories, err
}

func (r *AlertRuleRepository) Update(ctx context.Context, rule *model.AlertRule) error {
	return r.db.WithContext(ctx).Save(rule).Error
}

func (r *AlertRuleRepository) Delete(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Delete(&model.AlertRule{}, id).Error
}

// BatchUpdateStatus sets status for all rules whose IDs are in ids.
func (r *AlertRuleRepository) BatchUpdateStatus(ctx context.Context, ids []uint, status model.AlertRuleStatus) error {
	return r.db.WithContext(ctx).Model(&model.AlertRule{}).
		Where("id IN ?", ids).
		Updates(map[string]interface{}{"status": status, "version": gorm.Expr("version + 1")}).Error
}

// BatchDelete soft-deletes all rules whose IDs are in ids.
func (r *AlertRuleRepository) BatchDelete(ctx context.Context, ids []uint) error {
	return r.db.WithContext(ctx).Where("id IN ?", ids).Delete(&model.AlertRule{}).Error
}
