package repository

import (
	"context"
	"time"

	"gorm.io/gorm"

	"github.com/sreagent/sreagent/internal/model"
)

// txContextKey is used to store a *gorm.DB transaction in context.Context.
// Defined at package level so all repositories in this package share the same key.
type txContextKey struct{}

// ContextWithTx returns a new context carrying the given transaction.
func ContextWithTx(ctx context.Context, tx *gorm.DB) context.Context {
	return context.WithValue(ctx, txContextKey{}, tx)
}

// txFromContext extracts a *gorm.DB transaction from context, or returns nil.
func txFromContext(ctx context.Context) *gorm.DB {
	if tx, ok := ctx.Value(txContextKey{}).(*gorm.DB); ok {
		return tx
	}
	return nil
}

// IncidentRepository handles CRUD for incidents (故障).
type IncidentRepository struct {
	db *gorm.DB
}

func NewIncidentRepository(db *gorm.DB) *IncidentRepository {
	return &IncidentRepository{db: db}
}

// withTx returns the transaction stored in ctx, or falls back to r.db.WithContext(ctx).
func (r *IncidentRepository) withTx(ctx context.Context) *gorm.DB {
	if tx := txFromContext(ctx); tx != nil {
		return tx
	}
	return r.db.WithContext(ctx)
}

// Transaction executes fn inside a database transaction. The transaction is
// propagated via context so all repository methods called within fn
// automatically participate in the transaction without any shared mutable state.
func (r *IncidentRepository) Transaction(ctx context.Context, fn func(ctx context.Context) error) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return fn(ContextWithTx(ctx, tx))
	})
}

func (r *IncidentRepository) Create(ctx context.Context, inc *model.Incident) error {
	return r.withTx(ctx).Create(inc).Error
}

func (r *IncidentRepository) GetByID(ctx context.Context, id uint) (*model.Incident, error) {
	var inc model.Incident
	err := r.withTx(ctx).
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

	q := r.withTx(ctx).Model(&model.Incident{})
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

	q := r.withTx(ctx).Model(&model.Incident{}).
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
	return r.withTx(ctx).Save(inc).Error
}

func (r *IncidentRepository) Delete(ctx context.Context, id uint) error {
	return r.withTx(ctx).Delete(&model.Incident{}, id).Error
}

// UpdateStatus updates status and related timestamps atomically.
func (r *IncidentRepository) UpdateStatus(ctx context.Context, id uint, status model.IncidentStatus, updates map[string]interface{}) error {
	updates["status"] = status
	return r.withTx(ctx).Model(&model.Incident{}).Where("id = ?", id).Updates(updates).Error
}

// TransitionStatus performs an atomic compare-and-swap status transition.
// Updates status only if the current status matches expectedStatus.
// Returns ErrInvalidTransition if no rows were affected (concurrent modification or wrong state).
func (r *IncidentRepository) TransitionStatus(ctx context.Context, id uint, expectedStatus, newStatus model.IncidentStatus, updates map[string]interface{}) error {
	updates["status"] = newStatus
	result := r.withTx(ctx).Model(&model.Incident{}).
		Where("id = ? AND status = ?", id, expectedStatus).
		Updates(updates)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

// UpdateAssignees updates only the assigned_to column to avoid lost-update races
// from concurrent full-row Save() calls.
func (r *IncidentRepository) UpdateAssignees(ctx context.Context, id uint, assignee *uint) error {
	return r.withTx(ctx).Model(&model.Incident{}).Where("id = ?", id).
		Update("assigned_to", assignee).Error
}

// UpdateEscalationStep updates only the current_escalation_step column to avoid
// lost-update races from concurrent full-row Save() calls.
func (r *IncidentRepository) UpdateEscalationStep(ctx context.Context, id uint, step int) error {
	return r.withTx(ctx).Model(&model.Incident{}).Where("id = ?", id).
		Update("current_escalation_step", step).Error
}

// TransitionEscalationStep performs an atomic compare-and-swap escalation step update.
// Updates current_escalation_step only if the current value matches expectedStep.
// Returns gorm.ErrRecordNotFound if no rows were affected (concurrent modification).
func (r *IncidentRepository) TransitionEscalationStep(ctx context.Context, id uint, expectedStep, newStep int) error {
	result := r.withTx(ctx).Model(&model.Incident{}).
		Where("id = ? AND current_escalation_step = ?", id, expectedStep).
		Update("current_escalation_step", newStep)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

// --- IncidentAssignee ---

func (r *IncidentRepository) AddAssignee(ctx context.Context, assignee *model.IncidentAssignee) error {
	return r.withTx(ctx).Create(assignee).Error
}

func (r *IncidentRepository) ListAssignees(ctx context.Context, incidentID uint) ([]model.IncidentAssignee, error) {
	var list []model.IncidentAssignee
	err := r.withTx(ctx).Preload("User").Where("incident_id = ?", incidentID).Find(&list).Error
	return list, err
}

func (r *IncidentRepository) AcknowledgeAssignee(ctx context.Context, incidentID, userID uint) error {
	now := time.Now()
	return r.withTx(ctx).
		Model(&model.IncidentAssignee{}).
		Where("incident_id = ? AND user_id = ?", incidentID, userID).
		Updates(map[string]interface{}{
			"is_acknowledged": true,
			"acknowledged_at": now,
		}).Error
}

func (r *IncidentRepository) RemoveAssignees(ctx context.Context, incidentID uint) error {
	return r.withTx(ctx).Where("incident_id = ?", incidentID).Delete(&model.IncidentAssignee{}).Error
}

// --- IncidentTimeline ---

func (r *IncidentRepository) AddTimeline(ctx context.Context, entry *model.IncidentTimeline) error {
	// Ensure JSON fields have valid values (MySQL JSON columns don't accept empty strings)
	if entry.Extra == "" {
		entry.Extra = "{}"
	}
	return r.withTx(ctx).Create(entry).Error
}

func (r *IncidentRepository) ListTimeline(ctx context.Context, incidentID uint) ([]model.IncidentTimeline, error) {
	var list []model.IncidentTimeline
	err := r.withTx(ctx).Preload("Actor").
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
	err := r.withTx(ctx).
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

// IncidentInTeams reports whether the incident's channel belongs to one of the
// given teams. Used for per-incident authorization, mirroring ListByTeamIDs.
func (r *IncidentRepository) IncidentInTeams(ctx context.Context, id uint, teamIDs []uint) (bool, error) {
	if len(teamIDs) == 0 {
		return false, nil
	}
	var count int64
	err := r.withTx(ctx).Model(&model.Incident{}).
		Where("id = ? AND channel_id IN (SELECT id FROM channels WHERE team_id IN ? AND deleted_at IS NULL)", id, teamIDs).
		Count(&count).Error
	return count > 0, err
}

// ListExpiredSnoozed returns snoozed incidents whose snooze window has elapsed.
func (r *IncidentRepository) ListExpiredSnoozed(ctx context.Context, now time.Time) ([]model.Incident, error) {
	var list []model.Incident
	err := r.withTx(ctx).
		Where("status = ? AND snoozed_until IS NOT NULL AND snoozed_until <= ?",
			string(model.IncidentStatusSnoozed), now).
		Limit(1000).
		Find(&list).Error
	return list, err
}

// --- Counts ---

// FindOpenByFingerprint returns the first open incident (triggered/processing) for a fingerprint.
func (r *IncidentRepository) FindOpenByFingerprint(ctx context.Context, fingerprint string) (*model.Incident, error) {
	var inc model.Incident
	err := r.withTx(ctx).
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
	err := r.withTx(ctx).
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
	q := r.withTx(ctx).Model(&model.Incident{}).Select("status, count(*) as count")
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
