package repository

import (
	"context"

	"gorm.io/gorm"

	"github.com/sreagent/sreagent/internal/model"
)

// DashboardBizGroupRepository handles dashboard-to-biz-group binding persistence.
type DashboardBizGroupRepository struct {
	db *gorm.DB
}

// NewDashboardBizGroupRepository creates a new DashboardBizGroupRepository.
func NewDashboardBizGroupRepository(db *gorm.DB) *DashboardBizGroupRepository {
	return &DashboardBizGroupRepository{db: db}
}

// BindDashboardToGroup creates a binding between a dashboard and a business group.
func (r *DashboardBizGroupRepository) BindDashboardToGroup(ctx context.Context, dashboardID, bizGroupID uint, permFlag string) error {
	binding := &model.DashboardBizGroup{
		DashboardID: dashboardID,
		BizGroupID:  bizGroupID,
		PermFlag:    permFlag,
	}
	return r.db.WithContext(ctx).Create(binding).Error
}

// UnbindDashboardFromGroup removes a binding between a dashboard and a business group.
func (r *DashboardBizGroupRepository) UnbindDashboardFromGroup(ctx context.Context, dashboardID, bizGroupID uint) error {
	return r.db.WithContext(ctx).
		Where("dashboard_id = ? AND biz_group_id = ?", dashboardID, bizGroupID).
		Delete(&model.DashboardBizGroup{}).Error
}

// ListGroupsByDashboard returns all biz group bindings for a dashboard.
func (r *DashboardBizGroupRepository) ListGroupsByDashboard(ctx context.Context, dashboardID uint) ([]model.DashboardBizGroup, error) {
	var bindings []model.DashboardBizGroup
	err := r.db.WithContext(ctx).
		Where("dashboard_id = ?", dashboardID).
		Find(&bindings).Error
	return bindings, err
}

// ListDashboardsByGroup returns all dashboard IDs accessible to a business group.
func (r *DashboardBizGroupRepository) ListDashboardsByGroup(ctx context.Context, bizGroupID uint) ([]uint, error) {
	var ids []uint
	err := r.db.WithContext(ctx).
		Model(&model.DashboardBizGroup{}).
		Where("biz_group_id = ?", bizGroupID).
		Pluck("dashboard_id", &ids).Error
	return ids, err
}

// GetBinding returns a specific binding.
func (r *DashboardBizGroupRepository) GetBinding(ctx context.Context, dashboardID, bizGroupID uint) (*model.DashboardBizGroup, error) {
	var binding model.DashboardBizGroup
	err := r.db.WithContext(ctx).
		Where("dashboard_id = ? AND biz_group_id = ?", dashboardID, bizGroupID).
		First(&binding).Error
	if err != nil {
		return nil, err
	}
	return &binding, nil
}

// UpdatePermFlag updates the permission flag for a binding.
func (r *DashboardBizGroupRepository) UpdatePermFlag(ctx context.Context, dashboardID, bizGroupID uint, permFlag string) error {
	return r.db.WithContext(ctx).
		Model(&model.DashboardBizGroup{}).
		Where("dashboard_id = ? AND biz_group_id = ?", dashboardID, bizGroupID).
		Update("perm_flag", permFlag).Error
}

// DeleteByDashboard removes all bindings for a dashboard (used when deleting a dashboard).
func (r *DashboardBizGroupRepository) DeleteByDashboard(ctx context.Context, dashboardID uint) error {
	return r.db.WithContext(ctx).
		Where("dashboard_id = ?", dashboardID).
		Delete(&model.DashboardBizGroup{}).Error
}
