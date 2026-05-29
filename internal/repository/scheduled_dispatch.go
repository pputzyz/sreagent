package repository

import (
	"context"
	"time"

	"gorm.io/gorm"

	"github.com/sreagent/sreagent/internal/model"
)

// ScheduledDispatchRepository handles CRUD for scheduled dispatches.
type ScheduledDispatchRepository struct {
	db *gorm.DB
}

func NewScheduledDispatchRepository(db *gorm.DB) *ScheduledDispatchRepository {
	return &ScheduledDispatchRepository{db: db}
}

// Create persists a new scheduled dispatch entry.
func (r *ScheduledDispatchRepository) Create(ctx context.Context, d *model.ScheduledDispatch) error {
	return r.db.WithContext(ctx).Create(d).Error
}

// GetDueDispatches returns up to limit pending dispatches whose dispatch_at <= now.
func (r *ScheduledDispatchRepository) GetDueDispatches(ctx context.Context, now time.Time, limit int) ([]model.ScheduledDispatch, error) {
	var list []model.ScheduledDispatch
	err := r.db.WithContext(ctx).
		Where("dispatch_at <= ? AND status = ?", now, model.ScheduledDispatchPending).
		Order("dispatch_at ASC").
		Limit(limit).
		Find(&list).Error
	return list, err
}

// MarkDispatched sets status = 'dispatched'.
func (r *ScheduledDispatchRepository) MarkDispatched(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).
		Model(&model.ScheduledDispatch{}).
		Where("id = ?", id).
		Update("status", model.ScheduledDispatchDispatched).Error
}

// MarkFailed sets status = 'failed' and records the error message.
func (r *ScheduledDispatchRepository) MarkFailed(ctx context.Context, id uint, lastError string) error {
	return r.db.WithContext(ctx).
		Model(&model.ScheduledDispatch{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"status":     model.ScheduledDispatchFailed,
			"last_error": lastError,
		}).Error
}

// ScheduleNext resets the dispatch for the next repeat cycle:
// increments repeat_count, updates dispatch_at, and sets status back to 'pending'.
func (r *ScheduledDispatchRepository) ScheduleNext(ctx context.Context, id uint, nextDispatchAt time.Time) error {
	return r.db.WithContext(ctx).
		Model(&model.ScheduledDispatch{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"repeat_count": gorm.Expr("repeat_count + 1"),
			"dispatch_at":  nextDispatchAt,
			"status":       model.ScheduledDispatchPending,
		}).Error
}

// CancelByIncident cancels all pending dispatches for an incident.
// Called when an incident is acknowledged or closed.
func (r *ScheduledDispatchRepository) CancelByIncident(ctx context.Context, incidentID uint) error {
	return r.db.WithContext(ctx).
		Model(&model.ScheduledDispatch{}).
		Where("incident_id = ? AND status = ?", incidentID, model.ScheduledDispatchPending).
		Update("status", model.ScheduledDispatchCancelled).Error
}
