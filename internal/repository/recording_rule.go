package repository

import (
	"context"
	"fmt"

	"gorm.io/gorm"

	"github.com/sreagent/sreagent/internal/model"
)

type RecordingRuleRepository struct {
	db *gorm.DB
}

func NewRecordingRuleRepository(db *gorm.DB) *RecordingRuleRepository {
	return &RecordingRuleRepository{db: db}
}

func (r *RecordingRuleRepository) Create(ctx context.Context, rule *model.RecordingRule) error {
	return r.db.WithContext(ctx).Create(rule).Error
}

func (r *RecordingRuleRepository) GetByID(ctx context.Context, id uint) (*model.RecordingRule, error) {
	var rule model.RecordingRule
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&rule).Error; err != nil {
		return nil, err
	}
	rule.DB2FE()
	return &rule, nil
}

func (r *RecordingRuleRepository) ListByGroupID(ctx context.Context, groupID uint) ([]model.RecordingRule, error) {
	var rules []model.RecordingRule
	q := r.db.WithContext(ctx).Order("name")
	if groupID > 0 {
		q = q.Where("group_id = ?", groupID)
	}
	if err := q.Find(&rules).Error; err != nil {
		return nil, err
	}
	for i := range rules {
		rules[i].DB2FE()
	}
	return rules, nil
}

func (r *RecordingRuleRepository) ListByGroupIDs(ctx context.Context, groupIDs []uint) ([]model.RecordingRule, error) {
	if len(groupIDs) == 0 {
		return nil, nil
	}
	var rules []model.RecordingRule
	if err := r.db.WithContext(ctx).Where("group_id IN ?", groupIDs).Order("name").Find(&rules).Error; err != nil {
		return nil, err
	}
	for i := range rules {
		rules[i].DB2FE()
	}
	return rules, nil
}

func (r *RecordingRuleRepository) ListEnabled(ctx context.Context) ([]model.RecordingRule, error) {
	var rules []model.RecordingRule
	if err := r.db.WithContext(ctx).Where("disabled = 0").Order("name").Find(&rules).Error; err != nil {
		return nil, err
	}
	for i := range rules {
		rules[i].DB2FE()
	}
	return rules, nil
}

func (r *RecordingRuleRepository) Update(ctx context.Context, rule *model.RecordingRule) error {
	return r.db.WithContext(ctx).Model(rule).Select("*").Updates(rule).Error
}

func (r *RecordingRuleRepository) UpdateFields(ctx context.Context, id uint, fields map[string]interface{}) error {
	return r.db.WithContext(ctx).Model(&model.RecordingRule{}).Where("id = ?", id).Updates(fields).Error
}

func (r *RecordingRuleRepository) Delete(ctx context.Context, id uint, groupID uint) error {
	return r.db.WithContext(ctx).Where("id = ? AND group_id = ?", id, groupID).Delete(&model.RecordingRule{}).Error
}

func (r *RecordingRuleRepository) DeleteByIDs(ctx context.Context, ids []uint, groupID uint) error {
	if len(ids) == 0 {
		return nil
	}
	return r.db.WithContext(ctx).Where("id IN ? AND group_id = ?", ids, groupID).Delete(&model.RecordingRule{}).Error
}

func (r *RecordingRuleRepository) ListWithFilter(ctx context.Context, groupID uint, query string, disabled *int, page, pageSize int) ([]model.RecordingRule, int64, error) {
	var rules []model.RecordingRule
	var total int64

	q := r.db.WithContext(ctx).Model(&model.RecordingRule{})
	if groupID > 0 {
		q = q.Where("group_id = ?", groupID)
	}
	if query != "" {
		like := fmt.Sprintf("%%%s%%", query)
		q = q.Where("(name LIKE ? OR note LIKE ? OR append_tags LIKE ?)", like, like, like)
	}
	if disabled != nil {
		q = q.Where("disabled = ?", *disabled)
	}

	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	if err := q.Order("name").Offset(offset).Limit(pageSize).Find(&rules).Error; err != nil {
		return nil, 0, err
	}

	for i := range rules {
		rules[i].DB2FE()
	}
	return rules, total, nil
}
