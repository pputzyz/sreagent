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

// DeleteCascade deletes a schedule and all its child records in a single transaction.
// Order: shifts → overrides → participants → schedule.
func (r *ScheduleRepository) DeleteCascade(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("schedule_id = ?", id).Delete(&model.OnCallShift{}).Error; err != nil {
			return err
		}
		if err := tx.Where("schedule_id = ?", id).Delete(&model.ScheduleOverride{}).Error; err != nil {
			return err
		}
		if err := tx.Where("schedule_id = ?", id).Delete(&model.ScheduleParticipant{}).Error; err != nil {
			return err
		}
		if err := tx.Delete(&model.Schedule{}, id).Error; err != nil {
			return err
		}
		return nil
	})
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

// UpdatePositions updates participant positions in a single transaction.
// NOTE: Schedule participants per schedule are typically <20, so per-row UPDATE
// is acceptable. If the dataset grows, consider CASE WHEN batch update.
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

// DeleteByScheduleID deletes all overrides for a given schedule.
func (r *ScheduleOverrideRepository) DeleteByScheduleID(ctx context.Context, scheduleID uint) error {
	return r.db.WithContext(ctx).Where("schedule_id = ?", scheduleID).Delete(&model.ScheduleOverride{}).Error
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

// ListAllEnabled returns all enabled escalation policies.
func (r *EscalationPolicyRepository) ListAllEnabled(ctx context.Context) ([]model.EscalationPolicy, error) {
	var list []model.EscalationPolicy
	err := r.db.WithContext(ctx).Where("is_enabled = ?", true).Order("id DESC").Preload("Team").Find(&list).Error
	return list, err
}

func (r *EscalationPolicyRepository) Update(ctx context.Context, policy *model.EscalationPolicy) error {
	return r.db.WithContext(ctx).Save(policy).Error
}

func (r *EscalationPolicyRepository) Delete(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Delete(&model.EscalationPolicy{}, id).Error
}

// DeleteCascade deletes an escalation policy and all its steps in a single transaction.
func (r *EscalationPolicyRepository) DeleteCascade(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("policy_id = ?", id).Delete(&model.EscalationStep{}).Error; err != nil {
			return err
		}
		if err := tx.Delete(&model.EscalationPolicy{}, id).Error; err != nil {
			return err
		}
		return nil
	})
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

func (r *EscalationStepRepository) Delete(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Delete(&model.EscalationStep{}, id).Error
}

// DeleteByPolicyID deletes all escalation steps for a given policy.
func (r *EscalationStepRepository) DeleteByPolicyID(ctx context.Context, policyID uint) error {
	return r.db.WithContext(ctx).
		Where("policy_id = ?", policyID).
		Delete(&model.EscalationStep{}).Error
}

// BatchLoadByPolicyIDs loads all escalation steps for the given policy IDs in a single query.
// Returns a map keyed by policyID. If policyIDs is empty, returns nil.
func (r *EscalationStepRepository) BatchLoadByPolicyIDs(ctx context.Context, policyIDs []uint) (map[uint][]model.EscalationStep, error) {
	if len(policyIDs) == 0 {
		return nil, nil
	}
	var steps []model.EscalationStep
	if err := r.db.WithContext(ctx).
		Where("policy_id IN ?", policyIDs).
		Order("policy_id ASC, step_order ASC").
		Find(&steps).Error; err != nil {
		return nil, err
	}
	m := make(map[uint][]model.EscalationStep, len(policyIDs))
	for _, s := range steps {
		m[s.PolicyID] = append(m[s.PolicyID], s)
	}
	return m, nil
}

// ReplaceByPolicyID replaces all steps for a policy in a single transaction.
// Uses CreateInBatches to avoid N individual INSERT statements.
func (r *EscalationStepRepository) ReplaceByPolicyID(ctx context.Context, policyID uint, steps []model.EscalationStep) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("policy_id = ?", policyID).Delete(&model.EscalationStep{}).Error; err != nil {
			return err
		}
		if len(steps) > 0 {
			if err := tx.CreateInBatches(&steps, len(steps)).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

// ---------------------------------------------------------------------------
// EscalationStepExecutionRepository
// ---------------------------------------------------------------------------

type EscalationStepExecutionRepository struct {
	db *gorm.DB
}

func NewEscalationStepExecutionRepository(db *gorm.DB) *EscalationStepExecutionRepository {
	return &EscalationStepExecutionRepository{db: db}
}

// InsertIgnore atomically records a step execution using INSERT IGNORE.
// Returns true if the row was inserted (i.e., this is the first execution).
// The row is inserted with status='pending'.
func (r *EscalationStepExecutionRepository) InsertIgnore(ctx context.Context, eventID, stepID uint) (bool, error) {
	exec := &model.EscalationStepExecution{
		EventID:    eventID,
		StepID:     stepID,
		Status:     "pending",
		ExecutedAt: time.Now(),
	}
	// INSERT IGNORE: if the unique key (event_id, step_id) already exists, the row is silently ignored.
	result := r.db.WithContext(ctx).Exec(
		"INSERT IGNORE INTO escalation_step_executions (event_id, step_id, status, executed_at) VALUES (?, ?, ?, ?)",
		exec.EventID, exec.StepID, exec.Status, exec.ExecutedAt,
	)
	if result.Error != nil {
		return false, result.Error
	}
	return result.RowsAffected > 0, nil
}

// HasExecuted checks if a step has already been successfully executed for an event.
// Only returns true for status='success', allowing failed steps to be retried.
func (r *EscalationStepExecutionRepository) HasExecuted(ctx context.Context, eventID, stepID uint) (bool, error) {
	var count int64
	if err := r.db.WithContext(ctx).Model(&model.EscalationStepExecution{}).
		Where("event_id = ? AND step_id = ? AND status = ?", eventID, stepID, "success").
		Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

// MarkSuccess updates a step execution record to status='success'.
func (r *EscalationStepExecutionRepository) MarkSuccess(ctx context.Context, eventID, stepID uint) error {
	return r.db.WithContext(ctx).
		Model(&model.EscalationStepExecution{}).
		Where("event_id = ? AND step_id = ?", eventID, stepID).
		Update("status", "success").Error
}

// MarkFailed updates a step execution record to status='failed'.
func (r *EscalationStepExecutionRepository) MarkFailed(ctx context.Context, eventID, stepID uint) error {
	return r.db.WithContext(ctx).
		Model(&model.EscalationStepExecution{}).
		Where("event_id = ? AND step_id = ?", eventID, stepID).
		Update("status", "failed").Error
}

// DeleteByEventAndStep removes a step execution record so it can be retried on the next cycle.
func (r *EscalationStepExecutionRepository) DeleteByEventAndStep(ctx context.Context, eventID, stepID uint) error {
	return r.db.WithContext(ctx).
		Where("event_id = ? AND step_id = ?", eventID, stepID).
		Delete(&model.EscalationStepExecution{}).Error
}
