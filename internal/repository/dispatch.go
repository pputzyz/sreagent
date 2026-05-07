package repository

import (
	"context"

	"gorm.io/gorm"

	"github.com/sreagent/sreagent/internal/model"
)

// DispatchPolicyRepository handles CRUD for dispatch policies.
type DispatchPolicyRepository struct {
	db *gorm.DB
}

func NewDispatchPolicyRepository(db *gorm.DB) *DispatchPolicyRepository {
	return &DispatchPolicyRepository{db: db}
}

func (r *DispatchPolicyRepository) Create(ctx context.Context, p *model.DispatchPolicy) error {
	return r.db.WithContext(ctx).Create(p).Error
}

func (r *DispatchPolicyRepository) GetByID(ctx context.Context, id uint) (*model.DispatchPolicy, error) {
	var p model.DispatchPolicy
	err := r.db.WithContext(ctx).
		Preload("Channel").
		Preload("EscalationPolicy").
		First(&p, id).Error
	if err != nil {
		return nil, err
	}
	return &p, nil
}

// ListByChannel returns all dispatch policies for a channel, ordered by priority.
func (r *DispatchPolicyRepository) ListByChannel(ctx context.Context, channelID uint) ([]model.DispatchPolicy, error) {
	var list []model.DispatchPolicy
	err := r.db.WithContext(ctx).
		Where("channel_id = ?", channelID).
		Order("priority ASC, id ASC").
		Find(&list).Error
	return list, err
}

// ListEnabledByChannel returns enabled policies for a channel, ordered by priority.
func (r *DispatchPolicyRepository) ListEnabledByChannel(ctx context.Context, channelID uint) ([]model.DispatchPolicy, error) {
	var list []model.DispatchPolicy
	err := r.db.WithContext(ctx).
		Where("channel_id = ? AND is_enabled = ?", channelID, true).
		Order("priority ASC, id ASC").
		Find(&list).Error
	return list, err
}

func (r *DispatchPolicyRepository) Update(ctx context.Context, p *model.DispatchPolicy) error {
	return r.db.WithContext(ctx).Save(p).Error
}

func (r *DispatchPolicyRepository) Delete(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Delete(&model.DispatchPolicy{}, id).Error
}

// --- DispatchLog ---

type DispatchLogRepository struct {
	db *gorm.DB
}

func NewDispatchLogRepository(db *gorm.DB) *DispatchLogRepository {
	return &DispatchLogRepository{db: db}
}

func (r *DispatchLogRepository) Create(ctx context.Context, log *model.DispatchLog) error {
	return r.db.WithContext(ctx).Create(log).Error
}

func (r *DispatchLogRepository) ListByIncident(ctx context.Context, incidentID uint) ([]model.DispatchLog, error) {
	var list []model.DispatchLog
	err := r.db.WithContext(ctx).
		Where("incident_id = ?", incidentID).
		Order("id DESC").
		Find(&list).Error
	return list, err
}

func (r *DispatchLogRepository) UpdateStatus(ctx context.Context, id uint, status, note string) error {
	return r.db.WithContext(ctx).
		Model(&model.DispatchLog{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{"status": status, "note": note}).
		Error
}
