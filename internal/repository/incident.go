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
	// Tx is an optional per-request transaction. When set, all repository
	// methods use Tx instead of db. The caller is responsible for commit/rollback.
	Tx *gorm.DB
}

func NewIncidentRepository(db *gorm.DB) *IncidentRepository {
	return &IncidentRepository{db: db}
}

// withTx returns the active transaction if set, otherwise the default db.
func (r *IncidentRepository) withTx() *gorm.DB {
	if r.Tx != nil {
		return r.Tx
	}
	return r.db
}

// Transaction executes fn inside a database transaction. On success the
// transaction is committed; on error it is rolled back. The repository's Tx
// field is set for the duration of fn so that all repository methods
// automatically participate in the transaction.
func (r *IncidentRepository) Transaction(fn func(tx *gorm.DB) error) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		r.Tx = tx
		defer func() { r.Tx = nil }()
		return fn(tx)
	})
}

func (r *IncidentRepository) Create(ctx context.Context, inc *model.Incident) error {
	return r.withTx().WithContext(ctx).Create(inc).Error
}

func (r *IncidentRepository) GetByID(ctx context.Context, id uint) (*model.Incident, error) {
	var inc model.Incident
	err := r.withTx().WithContext(ctx).
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

	q := r.withTx().WithContext(ctx).Model(&model.Incident{})
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

// ListByTeamIDs is like List but restricted to incidents whose channel belongs to
// one of the given team IDs. Uses a subquery on the channels table.
func (r *IncidentRepository) ListByTeamIDs(ctx context.Context, teamIDs []uint, channelID uint, status, severity, query string, assignedTo uint, page, pageSize int) ([]model.Incident, int64, error) {
	var list []model.Incident
	var total int64

	q := r.withTx().WithContext(ctx).Model(&model.Incident{}).
		Where("channel_id IN (SELECT id FROM channels WHERE team_id IN ? AND deleted_at IS NULL)", teamIDs)
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
	return r.withTx().WithContext(ctx).Save(inc).Error
}

func (r *IncidentRepository) Delete(ctx context.Context, id uint) error {
	return r.withTx().WithContext(ctx).Delete(&model.Incident{}, id).Error
}

// UpdateStatus updates status and related timestamps atomically.
func (r *IncidentRepository) UpdateStatus(ctx context.Context, id uint, status model.IncidentStatus, updates map[string]interface{}) error {
	updates["status"] = status
	return r.withTx().WithContext(ctx).Model(&model.Incident{}).Where("id = ?", id).Updates(updates).Error
}

// UpdateAssignees updates only the assigned_to column to avoid lost-update races
// from concurrent full-row Save() calls.
func (r *IncidentRepository) UpdateAssignees(ctx context.Context, id uint, assignee *uint) error {
	return r.withTx().WithContext(ctx).Model(&model.Incident{}).Where("id = ?", id).
		Update("assigned_to", assignee).Error
}

// UpdateEscalationStep updates only the current_escalation_step column to avoid
// lost-update races from concurrent full-row Save() calls.
func (r *IncidentRepository) UpdateEscalationStep(ctx context.Context, id uint, step int) error {
	return r.withTx().WithContext(ctx).Model(&model.Incident{}).Where("id = ?", id).
		Update("current_escalation_step", step).Error
}

// --- IncidentAssignee ---

func (r *IncidentRepository) AddAssignee(ctx context.Context, assignee *model.IncidentAssignee) error {
	return r.withTx().WithContext(ctx).Create(assignee).Error
}

func (r *IncidentRepository) ListAssignees(ctx context.Context, incidentID uint) ([]model.IncidentAssignee, error) {
	var list []model.IncidentAssignee
	err := r.withTx().WithContext(ctx).Preload("User").Where("incident_id = ?", incidentID).Find(&list).Error
	return list, err
}

func (r *IncidentRepository) AcknowledgeAssignee(ctx context.Context, incidentID, userID uint) error {
	now := time.Now()
	return r.withTx().WithContext(ctx).
		Model(&model.IncidentAssignee{}).
		Where("incident_id = ? AND user_id = ?", incidentID, userID).
		Updates(map[string]interface{}{
			"is_acknowledged": true,
			"acknowledged_at": now,
		}).Error
}

func (r *IncidentRepository) RemoveAssignees(ctx context.Context, incidentID uint) error {
	return r.withTx().WithContext(ctx).Where("incident_id = ?", incidentID).Delete(&model.IncidentAssignee{}).Error
}

// --- IncidentTimeline ---

func (r *IncidentRepository) AddTimeline(ctx context.Context, entry *model.IncidentTimeline) error {
	return r.withTx().WithContext(ctx).Create(entry).Error
}

func (r *IncidentRepository) ListTimeline(ctx context.Context, incidentID uint) ([]model.IncidentTimeline, error) {
	var list []model.IncidentTimeline
	err := r.withTx().WithContext(ctx).Preload("Actor").
		Where("incident_id = ?", incidentID).
		Order("created_at ASC").
		Limit(1000).Find(&list).Error
	return list, err
}

// ListForAutoClose returns open incidents eligible for auto-close.
// Joins with channels to check auto_close_enabled and auto_close_minutes.
// LIMIT prevents unbounded result sets; auto-close worker re-polls periodically.
func (r *IncidentRepository) ListForAutoClose(ctx context.Context, now time.Time) ([]model.Incident, error) {
	var list []model.Incident
	err := r.withTx().WithContext(ctx).
		Joins("JOIN channels ON channels.id = incidents.channel_id AND channels.deleted_at IS NULL").
		Where("incidents.status IN ? AND channels.auto_close_enabled = ? AND incidents.closed_at IS NULL", []string{"triggered", "processing"}, true).
		// Keep column on the left side for index sargability:
		// triggered_at < DATE_SUB(?, INTERVAL auto_close_minutes MINUTE)
		Where("incidents.triggered_at < DATE_SUB(?, INTERVAL channels.auto_close_minutes MINUTE)", now).
		Where("channels.auto_close_minutes > 0").
		Limit(1000).
		Find(&list).Error
	return list, err
}

// --- Counts ---

// FindOpenByFingerprint returns the first open incident (triggered/processing) for a fingerprint.
func (r *IncidentRepository) FindOpenByFingerprint(ctx context.Context, fingerprint string) (*model.Incident, error) {
	var inc model.Incident
	err := r.withTx().WithContext(ctx).
		Where("fingerprint = ? AND status IN ?", fingerprint, []string{
			string(model.IncidentStatusTriggered),
			string(model.IncidentStatusProcessing),
		}).
		Order("id DESC").
		First(&inc).Error
	if err != nil {
		return nil, err
	}
	return &inc, nil
}

// CountActiveByChannel returns the count of active (triggered/processing) incidents for a channel.
func (r *IncidentRepository) CountActiveByChannel(ctx context.Context, channelID uint) (int64, error) {
	var count int64
	err := r.withTx().WithContext(ctx).
		Model(&model.Incident{}).
		Where("channel_id = ? AND status IN ?", channelID, []string{
			string(model.IncidentStatusTriggered),
			string(model.IncidentStatusProcessing),
		}).
		Count(&count).Error
	return count, err
}

func (r *IncidentRepository) CountByStatus(ctx context.Context, channelID uint) (map[string]int64, error) {
	type result struct {
		Status string
		Count  int64
	}
	var results []result
	q := r.withTx().WithContext(ctx).Model(&model.Incident{}).Select("status, count(*) as count")
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
