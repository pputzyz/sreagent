package repository

import (
	"context"
	"time"

	"gorm.io/gorm"

	"github.com/sreagent/sreagent/internal/model"
)

// IncidentRepository handles CRUD for incidents (故障).
type IncidentRepository struct {
	db *gorm.DB
}

func NewIncidentRepository(db *gorm.DB) *IncidentRepository {
	return &IncidentRepository{db: db}
}

func (r *IncidentRepository) Create(ctx context.Context, inc *model.Incident) error {
	return r.db.WithContext(ctx).Create(inc).Error
}

func (r *IncidentRepository) GetByID(ctx context.Context, id uint) (*model.Incident, error) {
	var inc model.Incident
	err := r.db.WithContext(ctx).
		Preload("Channel").
		Preload("AssignedUser").
		Preload("EscalationPolicy").
		First(&inc, id).Error
	if err != nil {
		return nil, err
	}
	return &inc, nil
}

// List returns paginated incidents with optional filters.
func (r *IncidentRepository) List(ctx context.Context, channelID uint, status, severity, query string, assignedTo uint, page, pageSize int) ([]model.Incident, int64, error) {
	var list []model.Incident
	var total int64

	q := r.db.WithContext(ctx).Model(&model.Incident{})
	if channelID > 0 {
		q = q.Where("channel_id = ?", channelID)
	}
	if status != "" {
		q = q.Where("status = ?", status)
	}
	if severity != "" {
		q = q.Where("severity = ?", severity)
	}
	if assignedTo > 0 {
		q = q.Where("assigned_to = ?", assignedTo)
	}
	if query != "" {
		like := "%" + query + "%"
		q = q.Where("title LIKE ? OR description LIKE ?", like, like)
	}

	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	if err := q.Preload("Channel").Preload("AssignedUser").
		Offset(offset).Limit(pageSize).Order("id DESC").Find(&list).Error; err != nil {
		return nil, 0, err
	}

	return list, total, nil
}

func (r *IncidentRepository) Update(ctx context.Context, inc *model.Incident) error {
	return r.db.WithContext(ctx).Save(inc).Error
}

func (r *IncidentRepository) Delete(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Delete(&model.Incident{}, id).Error
}

// UpdateStatus updates status and related timestamps atomically.
func (r *IncidentRepository) UpdateStatus(ctx context.Context, id uint, status model.IncidentStatus, updates map[string]interface{}) error {
	updates["status"] = status
	return r.db.WithContext(ctx).Model(&model.Incident{}).Where("id = ?", id).Updates(updates).Error
}

// --- IncidentAssignee ---

func (r *IncidentRepository) AddAssignee(ctx context.Context, assignee *model.IncidentAssignee) error {
	return r.db.WithContext(ctx).Create(assignee).Error
}

func (r *IncidentRepository) ListAssignees(ctx context.Context, incidentID uint) ([]model.IncidentAssignee, error) {
	var list []model.IncidentAssignee
	err := r.db.WithContext(ctx).Preload("User").Where("incident_id = ?", incidentID).Find(&list).Error
	return list, err
}

func (r *IncidentRepository) AcknowledgeAssignee(ctx context.Context, incidentID, userID uint) error {
	now := time.Now()
	return r.db.WithContext(ctx).
		Model(&model.IncidentAssignee{}).
		Where("incident_id = ? AND user_id = ?", incidentID, userID).
		Updates(map[string]interface{}{
			"is_acknowledged": true,
			"acknowledged_at": now,
		}).Error
}

func (r *IncidentRepository) RemoveAssignees(ctx context.Context, incidentID uint) error {
	return r.db.WithContext(ctx).Where("incident_id = ?", incidentID).Delete(&model.IncidentAssignee{}).Error
}

// --- IncidentTimeline ---

func (r *IncidentRepository) AddTimeline(ctx context.Context, entry *model.IncidentTimeline) error {
	return r.db.WithContext(ctx).Create(entry).Error
}

func (r *IncidentRepository) ListTimeline(ctx context.Context, incidentID uint) ([]model.IncidentTimeline, error) {
	var list []model.IncidentTimeline
	err := r.db.WithContext(ctx).Preload("Actor").
		Where("incident_id = ?", incidentID).
		Order("created_at ASC").Find(&list).Error
	return list, err
}

// ListForAutoClose returns open incidents eligible for auto-close.
// Joins with channels to check auto_close_enabled and auto_close_minutes.
func (r *IncidentRepository) ListForAutoClose(ctx context.Context, now time.Time) ([]model.Incident, error) {
	var list []model.Incident
	err := r.db.WithContext(ctx).
		Joins("JOIN channels ON channels.id = incidents.channel_id AND channels.deleted_at IS NULL").
		Where("incidents.status IN ? AND channels.auto_close_enabled = ? AND incidents.closed_at IS NULL", []string{"triggered", "processing"}, true).
		Where("DATE_ADD(incidents.triggered_at, INTERVAL channels.auto_close_minutes MINUTE) < ?", now).
		Where("channels.auto_close_minutes > 0").
		Find(&list).Error
	return list, err
}

// --- Counts ---

func (r *IncidentRepository) CountByStatus(ctx context.Context, channelID uint) (map[string]int64, error) {
	type result struct {
		Status string
		Count  int64
	}
	var results []result
	q := r.db.WithContext(ctx).Model(&model.Incident{}).Select("status, count(*) as count")
	if channelID > 0 {
		q = q.Where("channel_id = ?", channelID)
	}
	if err := q.Group("status").Find(&results).Error; err != nil {
		return nil, err
	}
	m := make(map[string]int64)
	for _, r := range results {
		m[r.Status] = r.Count
	}
	return m, nil
}
