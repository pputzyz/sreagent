package repository

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"

	"github.com/sreagent/sreagent/internal/model"
)

// OnCallShiftRepository handles persistence for OnCallShift records.
type OnCallShiftRepository struct {
	db *gorm.DB
}

func NewOnCallShiftRepository(db *gorm.DB) *OnCallShiftRepository {
	return &OnCallShiftRepository{db: db}
}

// Create inserts a new OnCallShift record.
func (r *OnCallShiftRepository) Create(ctx context.Context, shift *model.OnCallShift) error {
	return r.db.WithContext(ctx).Create(shift).Error
}

// GetByID retrieves a shift by primary key.
func (r *OnCallShiftRepository) GetByID(ctx context.Context, id uint) (*model.OnCallShift, error) {
	var shift model.OnCallShift
	err := r.db.WithContext(ctx).Preload("User").First(&shift, id).Error
	if err != nil {
		return nil, err
	}
	return &shift, nil
}

// Update saves changes to an existing shift.
func (r *OnCallShiftRepository) Update(ctx context.Context, shift *model.OnCallShift) error {
	return r.db.WithContext(ctx).Save(shift).Error
}

// Delete soft-deletes a shift.
func (r *OnCallShiftRepository) Delete(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Delete(&model.OnCallShift{}, id).Error
}

// DeleteByScheduleID deletes all shifts for a given schedule.
func (r *OnCallShiftRepository) DeleteByScheduleID(ctx context.Context, scheduleID uint) error {
	return r.db.WithContext(ctx).Where("schedule_id = ?", scheduleID).Delete(&model.OnCallShift{}).Error
}

// ListBySchedule returns all shifts for a schedule that overlap with [start, end).
func (r *OnCallShiftRepository) ListBySchedule(ctx context.Context, scheduleID uint, start, end time.Time) ([]model.OnCallShift, error) {
	var list []model.OnCallShift
	err := r.db.WithContext(ctx).
		Where("schedule_id = ? AND start_time < ? AND end_time > ?", scheduleID, end, start).
		Order("start_time ASC").
		Preload("User").
		Find(&list).Error
	return list, err
}

// GetCurrentShift returns the active shift for a schedule at the given time.
// Returns nil (no error) when no shift is active.
func (r *OnCallShiftRepository) GetCurrentShift(ctx context.Context, scheduleID uint, now time.Time) (*model.OnCallShift, error) {
	var shift model.OnCallShift
	err := r.db.WithContext(ctx).
		Where("schedule_id = ? AND start_time <= ? AND end_time > ?", scheduleID, now, now).
		Order("start_time DESC").
		Preload("User").
		First(&shift).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &shift, nil
}

// GetCurrentOnCallUser returns the user on call right now for the given schedule.
// Returns nil, nil when nobody is on call.
func (r *OnCallShiftRepository) GetCurrentOnCallUser(ctx context.Context, scheduleID uint) (*model.User, error) {
	shift, err := r.GetCurrentShift(ctx, scheduleID, time.Now())
	if err != nil {
		return nil, err
	}
	if shift == nil {
		return nil, nil
	}
	return &shift.User, nil
}

// ListUpcoming returns the next N shifts for a schedule starting from now.
func (r *OnCallShiftRepository) ListUpcoming(ctx context.Context, scheduleID uint, limit int) ([]model.OnCallShift, error) {
	var list []model.OnCallShift
	now := time.Now()
	err := r.db.WithContext(ctx).
		Where("schedule_id = ? AND end_time > ?", scheduleID, now).
		Order("start_time ASC").
		Limit(limit).
		Preload("User").
		Find(&list).Error
	return list, err
}

// BulkCreate inserts multiple shifts in a single operation.
func (r *OnCallShiftRepository) BulkCreate(ctx context.Context, shifts []model.OnCallShift) error {
	if len(shifts) == 0 {
		return nil
	}
	return r.db.WithContext(ctx).Create(&shifts).Error
}

// DeleteByScheduleAndTimeRange removes all shifts for a schedule within [start, end).
// Used to clean up auto-generated shifts before regenerating.
func (r *OnCallShiftRepository) DeleteByScheduleAndTimeRange(ctx context.Context, scheduleID uint, start, end time.Time) error {
	return r.db.WithContext(ctx).
		Where("schedule_id = ? AND start_time >= ? AND start_time < ?", scheduleID, start, end).
		Delete(&model.OnCallShift{}).Error
}

// GetCurrentShiftsForSchedules returns the active shift for each of the given
// schedule IDs at the given time, in a single query. Returns a map keyed by scheduleID.
func (r *OnCallShiftRepository) GetCurrentShiftsForSchedules(ctx context.Context, scheduleIDs []uint, now time.Time) (map[uint]*model.OnCallShift, error) {
	if len(scheduleIDs) == 0 {
		return nil, nil
	}
	var shifts []model.OnCallShift
	err := r.db.WithContext(ctx).
		Where("schedule_id IN ? AND start_time <= ? AND end_time > ?", scheduleIDs, now, now).
		Order("schedule_id ASC, start_time DESC").
		Preload("User").
		Find(&shifts).Error
	if err != nil {
		return nil, err
	}
	// Take the most recent active shift per schedule.
	m := make(map[uint]*model.OnCallShift, len(scheduleIDs))
	for i := range shifts {
		if _, exists := m[shifts[i].ScheduleID]; !exists {
			m[shifts[i].ScheduleID] = &shifts[i]
		}
	}
	return m, nil
}

// HasOverlapShift checks if any shift exists for the given schedule that overlaps the time range.
// excludeID can be set to exclude a specific shift (used during updates).
func (r *OnCallShiftRepository) HasOverlapShift(ctx context.Context, scheduleID uint, start, end time.Time, excludeID uint) (bool, error) {
	var count int64
	query := r.db.WithContext(ctx).Model(&model.OnCallShift{}).
		Where("schedule_id = ? AND start_time < ? AND end_time > ?", scheduleID, end, start)
	if excludeID > 0 {
		query = query.Where("id != ?", excludeID)
	}
	if err := query.Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}
