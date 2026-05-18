package repository

import (
	"context"
	"time"

	"gorm.io/gorm"

	"github.com/sreagent/sreagent/internal/model"
)

// ---------------------------------------------------------------------------
// ScheduleRepository
// ---------------------------------------------------------------------------

type ScheduleRepository struct {
	db *gorm.DB
}

func NewScheduleRepository(db *gorm.DB) *ScheduleRepository {
	return &ScheduleRepository{db: db}
}

func (r *ScheduleRepository) Create(ctx context.Context, schedule *model.Schedule) error {
	return r.db.WithContext(ctx).Create(schedule).Error
}

func (r *ScheduleRepository) GetByID(ctx context.Context, id uint) (*model.Schedule, error) {
	var schedule model.Schedule
	err := r.db.WithContext(ctx).Preload("Team").First(&schedule, id).Error
	if err != nil {
		return nil, err
	}
	return &schedule, nil
}

func (r *ScheduleRepository) List(ctx context.Context, teamID uint, page, pageSize int) ([]model.Schedule, int64, error) {
	var list []model.Schedule
	var total int64

	query := r.db.WithContext(ctx).Model(&model.Schedule{})
	if teamID > 0 {
		query = query.Where("team_id = ?", teamID)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	if err := query.Offset(offset).Limit(pageSize).Order("id DESC").Preload("Team").Find(&list).Error; err != nil {
		return nil, 0, err
	}

	return list, total, nil
}

func (r *ScheduleRepository) Update(ctx context.Context, schedule *model.Schedule) error {
	return r.db.WithContext(ctx).Save(schedule).Error
}

func (r *ScheduleRepository) Delete(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Delete(&model.Schedule{}, id).Error
}

// ---------------------------------------------------------------------------
// ScheduleParticipantRepository
// ---------------------------------------------------------------------------

type ScheduleParticipantRepository struct {
	db *gorm.DB
}

func NewScheduleParticipantRepository(db *gorm.DB) *ScheduleParticipantRepository {
	return &ScheduleParticipantRepository{db: db}
}

func (r *ScheduleParticipantRepository) Create(ctx context.Context, participant *model.ScheduleParticipant) error {
	return r.db.WithContext(ctx).Create(participant).Error
}

func (r *ScheduleParticipantRepository) ListByScheduleID(ctx context.Context, scheduleID uint) ([]model.ScheduleParticipant, error) {
	var list []model.ScheduleParticipant
	err := r.db.WithContext(ctx).
		Where("schedule_id = ?", scheduleID).
		Order("position ASC").
		Preload("User").
		Find(&list).Error
	return list, err
}

func (r *ScheduleParticipantRepository) DeleteByScheduleID(ctx context.Context, scheduleID uint) error {
	return r.db.WithContext(ctx).
		Where("schedule_id = ?", scheduleID).
		Delete(&model.ScheduleParticipant{}).Error
}

func (r *ScheduleParticipantRepository) UpdatePositions(ctx context.Context, participants []model.ScheduleParticipant) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for _, p := range participants {
			if err := tx.Model(&model.ScheduleParticipant{}).
				Where("id = ?", p.ID).
				Update("position", p.Position).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

// ---------------------------------------------------------------------------
// ScheduleOverrideRepository
// ---------------------------------------------------------------------------

type ScheduleOverrideRepository struct {
	db *gorm.DB
}

func NewScheduleOverrideRepository(db *gorm.DB) *ScheduleOverrideRepository {
	return &ScheduleOverrideRepository{db: db}
}

func (r *ScheduleOverrideRepository) Create(ctx context.Context, override *model.ScheduleOverride) error {
	return r.db.WithContext(ctx).Create(override).Error
}

func (r *ScheduleOverrideRepository) ListByScheduleID(ctx context.Context, scheduleID uint) ([]model.ScheduleOverride, error) {
	var list []model.ScheduleOverride
	err := r.db.WithContext(ctx).
		Where("schedule_id = ?", scheduleID).
		Order("start_time DESC").
		Preload("User").
		Find(&list).Error
	return list, err
}

func (r *ScheduleOverrideRepository) Delete(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Delete(&model.ScheduleOverride{}, id).Error
}

// GetActiveOverride returns the currently active override for a schedule at the given time.
func (r *ScheduleOverrideRepository) GetActiveOverride(ctx context.Context, scheduleID uint, at time.Time) (*model.ScheduleOverride, error) {
	var override model.ScheduleOverride
	err := r.db.WithContext(ctx).
		Where("schedule_id = ? AND start_time <= ? AND end_time > ?", scheduleID, at, at).
		Preload("User").
		Order("created_at DESC").
		First(&override).Error
	if err != nil {
		return nil, err
	}
	return &override, nil
}

// ---------------------------------------------------------------------------
// EscalationPolicyRepository
// ---------------------------------------------------------------------------

type EscalationPolicyRepository struct {
	db *gorm.DB
}

func NewEscalationPolicyRepository(db *gorm.DB) *EscalationPolicyRepository {
	return &EscalationPolicyRepository{db: db}
}

func (r *EscalationPolicyRepository) Create(ctx context.Context, policy *model.EscalationPolicy) error {
	return r.db.WithContext(ctx).Create(policy).Error
}

func (r *EscalationPolicyRepository) GetByID(ctx context.Context, id uint) (*model.EscalationPolicy, error) {
	var policy model.EscalationPolicy
	err := r.db.WithContext(ctx).Preload("Team").First(&policy, id).Error
	if err != nil {
		return nil, err
	}
	return &policy, nil
}

// GetByIDs returns all escalation policies whose ID is in the given slice.
// Returns nil (not an error) when ids is empty.
func (r *EscalationPolicyRepository) GetByIDs(ctx context.Context, ids []uint) ([]model.EscalationPolicy, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	var policies []model.EscalationPolicy
	err := r.db.WithContext(ctx).
		Where("id IN ?", ids).
		Find(&policies).Error
	return policies, err
}

func (r *EscalationPolicyRepository) ListByTeamID(ctx context.Context, teamID uint) ([]model.EscalationPolicy, error) {
	var list []model.EscalationPolicy
	query := r.db.WithContext(ctx).Model(&model.EscalationPolicy{})
	if teamID > 0 {
		query = query.Where("team_id = ?", teamID)
	}
	err := query.Order("id DESC").Preload("Team").Find(&list).Error
	return list, err
}

func (r *EscalationPolicyRepository) Update(ctx context.Context, policy *model.EscalationPolicy) error {
	return r.db.WithContext(ctx).Save(policy).Error
}

func (r *EscalationPolicyRepository) Delete(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Delete(&model.EscalationPolicy{}, id).Error
}

// ---------------------------------------------------------------------------
// EscalationStepRepository
// ---------------------------------------------------------------------------

type EscalationStepRepository struct {
	db *gorm.DB
}

func NewEscalationStepRepository(db *gorm.DB) *EscalationStepRepository {
	return &EscalationStepRepository{db: db}
}

func (r *EscalationStepRepository) Create(ctx context.Context, step *model.EscalationStep) error {
	return r.db.WithContext(ctx).Create(step).Error
}

func (r *EscalationStepRepository) ListByPolicyID(ctx context.Context, policyID uint) ([]model.EscalationStep, error) {
	var list []model.EscalationStep
	err := r.db.WithContext(ctx).
		Where("policy_id = ?", policyID).
		Order("step_order ASC").
		Find(&list).Error
	return list, err
}

func (r *EscalationStepRepository) Update(ctx context.Context, step *model.EscalationStep) error {
	return r.db.WithContext(ctx).Save(step).Error
}

func (r *EscalationStepRepository) Delete(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Delete(&model.EscalationStep{}, id).Error
}
