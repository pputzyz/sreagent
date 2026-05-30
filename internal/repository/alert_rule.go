package repository

import (
	"context"
	"time"

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

func (r *AlertRuleRepository) List(ctx context.Context, severity, status, groupName, category, keyword string, datasourceID *uint, page, pageSize int) ([]model.AlertRule, int64, error) {
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
	if keyword != "" {
		kw := "%" + keyword + "%"
		query = query.Where("name LIKE ? OR display_name LIKE ? OR expression LIKE ?", kw, kw, kw)
	}
	if datasourceID != nil {
		query = query.Where("datasource_id = ?", *datasourceID)
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

// GetByIDs returns all rules whose ID is in the given slice.
// Returns an empty slice (not an error) when ids is empty.
func (r *AlertRuleRepository) GetByIDs(ctx context.Context, ids []uint) ([]model.AlertRule, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	var rules []model.AlertRule
	err := r.db.WithContext(ctx).
		Where("id IN ?", ids).
		Find(&rules).Error
	return rules, err
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

// ListByTeamIDs is like List but restricted to rules whose team_id is in teamIDs.
// When teamIDs is empty, returns no results (the caller should guard against this).
func (r *AlertRuleRepository) ListByTeamIDs(ctx context.Context, teamIDs []uint, severity, status, groupName, category, keyword string, datasourceID *uint, page, pageSize int) ([]model.AlertRule, int64, error) {
	var list []model.AlertRule
	var total int64

	query := r.db.WithContext(ctx).Model(&model.AlertRule{}).Where("team_id IN ?", teamIDs)
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
	if keyword != "" {
		kw := "%" + keyword + "%"
		query = query.Where("name LIKE ? OR display_name LIKE ? OR expression LIKE ?", kw, kw, kw)
	}
	if datasourceID != nil {
		query = query.Where("datasource_id = ?", *datasourceID)
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

// ListCategories returns all distinct non-empty category values.
func (r *AlertRuleRepository) ListCategories(ctx context.Context) ([]string, error) {
	var categories []string
	err := r.db.WithContext(ctx).Model(&model.AlertRule{}).
		Where("category != ''").
		Distinct("category").Pluck("category", &categories).Error
	return categories, err
}

func (r *AlertRuleRepository) Update(ctx context.Context, rule *model.AlertRule) error {
	return r.db.WithContext(ctx).Save(rule).Error
}

// UpdateVersion performs an optimistic-lock update: the row is only updated when
// the current version in the database equals oldVersion. On success the version
// column is atomically incremented by 1. Returns (true, nil) if the update
// succeeded, (false, nil) if the version didn't match (conflict), or (false, err)
// on DB errors.
func (r *AlertRuleRepository) UpdateVersion(ctx context.Context, rule *model.AlertRule, oldVersion int) (bool, error) {
	result := r.db.WithContext(ctx).
		Model(rule).
		Where("id = ? AND version = ?", rule.ID, oldVersion).
		Updates(map[string]interface{}{
			"name":                  rule.Name,
			"display_name":          rule.DisplayName,
			"description":           rule.Description,
			"datasource_id":         rule.DataSourceID,
			"datasource_type":       rule.DatasourceType,
			"expression":            rule.Expression,
			"for_duration":          rule.ForDuration,
			"severity":              rule.Severity,
			"labels":                rule.Labels,
			"annotations":           rule.Annotations,
			"group_name":            rule.GroupName,
			"category":              rule.Category,
			"group_wait_seconds":    rule.GroupWaitSeconds,
			"group_interval_seconds": rule.GroupIntervalSeconds,
			"updated_by":            rule.UpdatedBy,
			"eval_interval":         rule.EvalInterval,
			"recovery_hold":         rule.RecoveryHold,
			"nodata_enabled":        rule.NoDataEnabled,
			"nodata_duration":       rule.NoDataDuration,
			"suppress_enabled":      rule.SuppressEnabled,
			"biz_group_id":          rule.BizGroupID,
			"rule_type":             rule.RuleType,
			"heartbeat_token":       rule.HeartbeatToken,
			"heartbeat_interval":    rule.HeartbeatInterval,
			"ack_sla_minutes":       rule.AckSlaMinutes,
			"status":                rule.Status,
			"var_config":            rule.VarConfig,
			"version":               gorm.Expr("version + 1"),
		})
	if result.Error != nil {
		return false, result.Error
	}
	return result.RowsAffected > 0, nil
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

// UpdateHeartbeatLastAt updates only the heartbeat_last_at column for a single rule.
// This avoids a full-row Save that would overwrite concurrent UI edits.
func (r *AlertRuleRepository) UpdateHeartbeatLastAt(ctx context.Context, ruleID uint, ts time.Time) error {
	return r.db.WithContext(ctx).Model(&model.AlertRule{}).
		Where("id = ?", ruleID).
		Update("heartbeat_last_at", ts).Error
}

// CountByDataSourceID counts alert rules referencing the given datasource (P1-11).
func (r *AlertRuleRepository) CountByDataSourceID(ctx context.Context, dsID uint) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&model.AlertRule{}).Where("data_source_id = ?", dsID).Count(&count).Error
	return count, err
}
