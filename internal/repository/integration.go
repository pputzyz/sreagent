package repository

import (
	"context"

	"gorm.io/gorm"

	"github.com/sreagent/sreagent/internal/model"
)

// IntegrationRepository handles CRUD for webhook integrations.
type IntegrationRepository struct {
	db *gorm.DB
}

func NewIntegrationRepository(db *gorm.DB) *IntegrationRepository {
	return &IntegrationRepository{db: db}
}

func (r *IntegrationRepository) Create(ctx context.Context, integ *model.Integration) error {
	return r.db.WithContext(ctx).Create(integ).Error
}

func (r *IntegrationRepository) GetByID(ctx context.Context, id uint) (*model.Integration, error) {
	var integ model.Integration
	err := r.db.WithContext(ctx).Preload("Channel").First(&integ, id).Error
	if err != nil {
		return nil, err
	}
	return &integ, nil
}

func (r *IntegrationRepository) GetByToken(ctx context.Context, token string) (*model.Integration, error) {
	var integ model.Integration
	err := r.db.WithContext(ctx).Where("webhook_token = ?", token).First(&integ).Error
	if err != nil {
		return nil, err
	}
	return &integ, nil
}

func (r *IntegrationRepository) List(ctx context.Context, channelID uint, page, pageSize int) ([]model.Integration, int64, error) {
	var list []model.Integration
	var total int64

	q := r.db.WithContext(ctx).Model(&model.Integration{})
	if channelID > 0 {
		q = q.Where("channel_id = ?", channelID)
	}
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	offset := (page - 1) * pageSize
	if err := q.Preload("Channel").Offset(offset).Limit(pageSize).Order("id DESC").Find(&list).Error; err != nil {
		return nil, 0, err
	}
	return list, total, nil
}

func (r *IntegrationRepository) Update(ctx context.Context, integ *model.Integration) error {
	return r.db.WithContext(ctx).Save(integ).Error
}

func (r *IntegrationRepository) Delete(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Delete(&model.Integration{}, id).Error
}

func (r *IntegrationRepository) IncrTotalAlerts(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).
		Model(&model.Integration{}).
		Where("id = ?", id).
		UpdateColumn("total_alerts", gorm.Expr("total_alerts + 1")).
		Error
}

// --- RoutingRule ---

type RoutingRuleRepository struct {
	db *gorm.DB
}

func NewRoutingRuleRepository(db *gorm.DB) *RoutingRuleRepository {
	return &RoutingRuleRepository{db: db}
}

func (r *RoutingRuleRepository) Create(ctx context.Context, rule *model.RoutingRule) error {
	return r.db.WithContext(ctx).Create(rule).Error
}

func (r *RoutingRuleRepository) GetByID(ctx context.Context, id uint) (*model.RoutingRule, error) {
	var rule model.RoutingRule
	err := r.db.WithContext(ctx).First(&rule, id).Error
	if err != nil {
		return nil, err
	}
	return &rule, nil
}

func (r *RoutingRuleRepository) ListByIntegration(ctx context.Context, integrationID uint) ([]model.RoutingRule, error) {
	var list []model.RoutingRule
	err := r.db.WithContext(ctx).
		Where("integration_id = ? AND is_enabled = ?", integrationID, true).
		Order("priority ASC, id ASC").
		Find(&list).Error
	return list, err
}

func (r *RoutingRuleRepository) Update(ctx context.Context, rule *model.RoutingRule) error {
	return r.db.WithContext(ctx).Save(rule).Error
}

func (r *RoutingRuleRepository) Delete(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Delete(&model.RoutingRule{}, id).Error
}
