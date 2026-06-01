package repository

import (
	"context"
	"time"

	"gorm.io/gorm"

	"github.com/sreagent/sreagent/internal/model"
)

// AlertRepository handles CRUD for the v2 Alert model.
type AlertRepository struct {
	db *gorm.DB
}

func NewAlertRepository(db *gorm.DB) *AlertRepository {
	return &AlertRepository{db: db}
}

// withTx returns the transaction stored in ctx, or falls back to r.db.WithContext(ctx).
func (r *AlertRepository) withTx(ctx context.Context) *gorm.DB {
	if tx := txFromContext(ctx); tx != nil {
		return tx
	}
	return r.db.WithContext(ctx)
}

func (r *AlertRepository) Create(ctx context.Context, alert *model.Alert) error {
	return r.db.WithContext(ctx).Create(alert).Error
}

func (r *AlertRepository) GetByID(ctx context.Context, id uint) (*model.Alert, error) {
	var alert model.Alert
	err := r.db.WithContext(ctx).
		Preload("Rule").
		Preload("Channel").
		Preload("Incident").
		First(&alert, id).Error
	if err != nil {
		return nil, err
	}
	return &alert, nil
}

// GetByAlertKey finds an alert by its deduplication key.
func (r *AlertRepository) GetByAlertKey(ctx context.Context, key string) (*model.Alert, error) {
	var alert model.Alert
	err := r.db.WithContext(ctx).Where("alert_key = ?", key).First(&alert).Error
	if err != nil {
		return nil, err
	}
	return &alert, nil
}

// List returns paginated alerts with optional filters.
func (r *AlertRepository) List(ctx context.Context, channelID, incidentID uint, status, severity, query string, page, pageSize int) ([]model.Alert, int64, error) {
	var list []model.Alert
	var total int64

	q := r.db.WithContext(ctx).Model(&model.Alert{})
	if channelID > 0 {
		q = q.Where("channel_id = ?", channelID)
	}
	if incidentID > 0 {
		q = q.Where("incident_id = ?", incidentID)
	}
	if status != "" {
		q = q.Where("status = ?", status)
	}
	if severity != "" {
		q = q.Where("severity = ?", severity)
	}
	if query != "" {
		like := "%" + query + "%"
		q = q.Where("title LIKE ? OR alert_key LIKE ?", like, like)
	}

	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	if err := q.Preload("Channel").Preload("Incident").
		Offset(offset).Limit(pageSize).Order("last_fired_at DESC").Find(&list).Error; err != nil {
		return nil, 0, err
	}

	return list, total, nil
}

func (r *AlertRepository) Update(ctx context.Context, alert *model.Alert) error {
	return r.db.WithContext(ctx).Save(alert).Error
}

// UpdateStatus updates the alert status and related timestamps.
func (r *AlertRepository) UpdateStatus(ctx context.Context, id uint, status model.AlertStatus, resolvedAt *time.Time) error {
	updates := map[string]interface{}{"status": status}
	if resolvedAt != nil {
		updates["resolved_at"] = resolvedAt
	}
	return r.db.WithContext(ctx).Model(&model.Alert{}).Where("id = ?", id).Updates(updates).Error
}

// IncrementFireCount atomically increments event/fire counts and updates last_fired_at.
func (r *AlertRepository) IncrementFireCount(ctx context.Context, id uint, now time.Time) error {
	return r.db.WithContext(ctx).
		Model(&model.Alert{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"fire_count":    gorm.Expr("fire_count + 1"),
			"event_count":   gorm.Expr("event_count + 1"),
			"last_fired_at": now,
			"status":        model.AlertStatusFiring,
		}).Error
}

// LinkToIncident sets the incident_id on an alert.
func (r *AlertRepository) LinkToIncident(ctx context.Context, alertID, incidentID uint) error {
	return r.db.WithContext(ctx).
		Model(&model.Alert{}).
		Where("id = ?", alertID).
		Update("incident_id", incidentID).Error
}

// BulkUpdateIncidentID moves all alerts from one incident to another.
// Used during incident merge to migrate source alerts to the target.
func (r *AlertRepository) BulkUpdateIncidentID(ctx context.Context, fromIncidentID, toIncidentID uint) error {
	return r.withTx(ctx).
		Model(&model.Alert{}).
		Where("incident_id = ?", fromIncidentID).
		Update("incident_id", toIncidentID).Error
}

// CountByIncidentID returns the number of alerts linked to an incident.
func (r *AlertRepository) CountByIncidentID(ctx context.Context, incidentID uint) (int, error) {
	var count int64
	err := r.withTx(ctx).
		Model(&model.Alert{}).
		Where("incident_id = ?", incidentID).
		Count(&count).Error
	return int(count), err
}

// --- AlertEventV2 (unified into alert_events) ---

// CreateEvent inserts a v2 pipeline event into the unified alert_events table.
func (r *AlertRepository) CreateEvent(ctx context.Context, event *model.AlertEvent) error {
	return r.db.WithContext(ctx).Create(event).Error
}

// ListEvents returns paginated v2 pipeline events for an alert,
// read from the unified alert_events table (where alert_id is set).
func (r *AlertRepository) ListEvents(ctx context.Context, alertID uint, page, pageSize int) ([]model.ViewAlertEvent, int64, error) {
	var list []model.AlertEvent
	var total int64

	q := r.db.WithContext(ctx).Model(&model.AlertEvent{}).Where("alert_id = ?", alertID)

	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	if err := q.Offset(offset).Limit(pageSize).Order("fired_at DESC").Find(&list).Error; err != nil {
		return nil, 0, err
	}

	// Convert to view model
	views := make([]model.ViewAlertEvent, len(list))
	for i := range list {
		views[i] = list[i].ToViewAlertEvent()
	}

	return views, total, nil
}
