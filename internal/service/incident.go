package service

import (
	"context"
	"fmt"
	"runtime/debug"
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/sreagent/sreagent/internal/model"
	apperr "github.com/sreagent/sreagent/internal/pkg/errors"
	"github.com/sreagent/sreagent/internal/repository"
)

const incidentAutoCloseInterval = 5 * time.Minute

// IncidentService provides business logic for incidents (故障).
type IncidentService struct {
	repo          *repository.IncidentRepository
	channelSvc    *ChannelService
	alertRepo     *repository.AlertRepository          // optional, for merge alert migration
	escStepRepo   *repository.EscalationStepRepository // optional, for escalation upper-bound check
	logger        *zap.Logger

	// onStatusChange is called when an incident transitions to processing (ack) or closed.
	// Used to cancel pending scheduled dispatches.
	onStatusChange func(ctx context.Context, incidentID uint, newStatus model.IncidentStatus)
}

func NewIncidentService(repo *repository.IncidentRepository, channelSvc *ChannelService, logger *zap.Logger) *IncidentService {
	return &IncidentService{repo: repo, channelSvc: channelSvc, logger: logger}
}

// SetAlertRepository injects the alert repository for incident merge operations.
func (s *IncidentService) SetAlertRepository(ar *repository.AlertRepository) {
	s.alertRepo = ar
}

// SetEscalationStepRepository injects the escalation step repository for escalation upper-bound checks.
func (s *IncidentService) SetEscalationStepRepository(sr *repository.EscalationStepRepository) {
	s.escStepRepo = sr
}

// SetOnStatusChange sets a callback that fires when an incident is acknowledged or closed.
func (s *IncidentService) SetOnStatusChange(fn func(ctx context.Context, incidentID uint, newStatus model.IncidentStatus)) {
	s.onStatusChange = fn
}

// validTransitions defines the allowed status transitions for incidents.
// Keys are source statuses, values are the set of allowed target statuses.
//
//	open (triggered)  → ack (processing), close (closed), snooze (snoozed)
//	ack (processing)  → open (triggered), close (closed)
//	close (closed)    → reopen (triggered)
//	snooze (snoozed)  → open (triggered), close (closed)
var validTransitions = map[model.IncidentStatus][]model.IncidentStatus{
	model.IncidentStatusTriggered:  {model.IncidentStatusProcessing, model.IncidentStatusClosed, model.IncidentStatusSnoozed},
	model.IncidentStatusProcessing: {model.IncidentStatusTriggered, model.IncidentStatusClosed},
	model.IncidentStatusClosed:     {model.IncidentStatusTriggered},
	model.IncidentStatusSnoozed:    {model.IncidentStatusTriggered, model.IncidentStatusClosed},
}

// allowedActionStates defines which statuses allow non-status-changing actions.
var allowedActionStates = map[string][]model.IncidentStatus{
	"reassign":  {model.IncidentStatusProcessing},
	"escalate":  {model.IncidentStatusProcessing},
}

// validateTransition checks whether a status transition is allowed.
func validateTransition(from, to model.IncidentStatus) error {
	targets, ok := validTransitions[from]
	if !ok {
		return apperr.WithMessage(apperr.ErrInvalidTransition,
			fmt.Sprintf("unknown source status %q", from))
	}
	for _, t := range targets {
		if t == to {
			return nil
		}
	}
	return apperr.WithMessage(apperr.ErrInvalidTransition,
		fmt.Sprintf("cannot transition from %q to %q", from, to))
}

// validateActionAllowed checks whether an action is allowed in the current status.
func validateActionAllowed(action string, current model.IncidentStatus) error {
	states, ok := allowedActionStates[action]
	if !ok {
		return nil // no restriction
	}
	for _, s := range states {
		if s == current {
			return nil
		}
	}
	return apperr.WithMessage(apperr.ErrInvalidTransition,
		fmt.Sprintf("action %q not allowed in status %q", action, current))
}

// Create creates a new incident and updates the channel's active incident count.
func (s *IncidentService) Create(ctx context.Context, inc *model.Incident) error {
	if inc.TriggeredAt.IsZero() {
		inc.TriggeredAt = time.Now()
	}
	if inc.Status == "" {
		inc.Status = model.IncidentStatusTriggered
	}

	if err := s.repo.Create(ctx, inc); err != nil {
		s.logger.Error("failed to create incident", zap.Error(err))
		return apperr.Wrap(apperr.ErrDatabase, err)
	}

	// Add triggered timeline entry
	if err := s.repo.AddTimeline(ctx, &model.IncidentTimeline{
		IncidentID: inc.ID,
		Action:     model.IncidentActionTriggered,
		Content:    "Incident triggered",
	}); err != nil {
		zap.L().Error("failed to add timeline", zap.Error(err), zap.Uint("incident_id", inc.ID))
	}

	s.logger.Info("incident created", zap.Uint("id", inc.ID), zap.String("title", inc.Title))
	return nil
}

// GetByID returns an incident by ID.
func (s *IncidentService) GetByID(ctx context.Context, id uint) (*model.Incident, error) {
	inc, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, apperr.ErrIncidentNotFound
		}
		return nil, apperr.Wrap(apperr.ErrDatabase, err)
	}
	return inc, nil
}

// List returns paginated incidents.
func (s *IncidentService) List(ctx context.Context, channelID uint, status, severity, query string, assignedTo uint, page, pageSize int) ([]model.Incident, int64, error) {
	list, total, err := s.repo.List(ctx, channelID, status, severity, query, assignedTo, page, pageSize)
	if err != nil {
		s.logger.Error("failed to list incidents", zap.Error(err))
		return nil, 0, apperr.Wrap(apperr.ErrDatabase, err)
	}
	return list, total, nil
}

// ListScoped returns paginated incidents with team-level data isolation.
// If isAdmin is true, the regular List is called (no filtering).
// Otherwise, only incidents whose channel belongs to the given teamIDs are returned.
// When teamIDs is empty for a non-admin user, an empty result is returned.
func (s *IncidentService) ListScoped(ctx context.Context, isAdmin bool, teamIDs []uint, channelID uint, status, severity, query string, assignedTo uint, page, pageSize int) ([]model.Incident, int64, error) {
	if isAdmin {
		return s.List(ctx, channelID, status, severity, query, assignedTo, page, pageSize)
	}
	if len(teamIDs) == 0 {
		return []model.Incident{}, 0, nil
	}
	list, total, err := s.repo.ListByTeamIDs(ctx, teamIDs, channelID, status, severity, query, assignedTo, page, pageSize)
	if err != nil {
		s.logger.Error("failed to list incidents (scoped)", zap.Error(err))
		return nil, 0, apperr.Wrap(apperr.ErrDatabase, err)
	}
	return list, total, nil
}

// Acknowledge marks the incident as processing and records the ack.
func (s *IncidentService) Acknowledge(ctx context.Context, id, userID uint) error {
	inc, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return apperr.ErrIncidentNotFound
		}
		return apperr.Wrap(apperr.ErrDatabase, err)
	}

	if err := validateTransition(inc.Status, model.IncidentStatusProcessing); err != nil {
		return err
	}

	now := time.Now()
	updates := map[string]interface{}{
		"acknowledged_at": now,
	}
	if err := s.repo.UpdateStatus(ctx, id, model.IncidentStatusProcessing, updates); err != nil {
		return apperr.Wrap(apperr.ErrDatabase, err)
	}

	// Record assignee ack
	if err := s.repo.AcknowledgeAssignee(ctx, id, userID); err != nil {
		zap.L().Error("failed to acknowledge assignee", zap.Error(err), zap.Uint("incident_id", id))
	}

	// Fire status change callback (cancels scheduled dispatches)
	if s.onStatusChange != nil {
		s.onStatusChange(ctx, id, model.IncidentStatusProcessing)
	}

	// Timeline
	if err := s.repo.AddTimeline(ctx, &model.IncidentTimeline{
		IncidentID: id,
		Action:     model.IncidentActionAcknowledged,
		ActorID:    &userID,
		Content:    "Incident acknowledged",
	}); err != nil {
		zap.L().Error("failed to add timeline", zap.Error(err), zap.Uint("incident_id", id))
	}

	s.logger.Info("incident acknowledged", zap.Uint("id", id), zap.Uint("user_id", userID))
	return nil
}

// Close closes an incident.
func (s *IncidentService) Close(ctx context.Context, id, userID uint) error {
	inc, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return apperr.ErrIncidentNotFound
		}
		return apperr.Wrap(apperr.ErrDatabase, err)
	}

	if inc.Status == model.IncidentStatusClosed {
		return nil // idempotent
	}

	if err := validateTransition(inc.Status, model.IncidentStatusClosed); err != nil {
		return err
	}

	now := time.Now()
	updates := map[string]interface{}{
		"closed_at": now,
	}
	if err := s.repo.UpdateStatus(ctx, id, model.IncidentStatusClosed, updates); err != nil {
		return apperr.Wrap(apperr.ErrDatabase, err)
	}

	// Fire status change callback (cancels scheduled dispatches)
	if s.onStatusChange != nil {
		s.onStatusChange(ctx, id, model.IncidentStatusClosed)
	}

	if err := s.repo.AddTimeline(ctx, &model.IncidentTimeline{
		IncidentID: id,
		Action:     model.IncidentActionClosed,
		ActorID:    &userID,
		Content:    "Incident closed",
	}); err != nil {
		zap.L().Error("failed to add timeline", zap.Error(err), zap.Uint("incident_id", id))
	}

	s.logger.Info("incident closed", zap.Uint("id", id), zap.Uint("user_id", userID))
	return nil
}

// Reopen re-opens a closed incident.
func (s *IncidentService) Reopen(ctx context.Context, id, userID uint) error {
	inc, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return apperr.ErrIncidentNotFound
		}
		return apperr.Wrap(apperr.ErrDatabase, err)
	}

	if err := validateTransition(inc.Status, model.IncidentStatusTriggered); err != nil {
		return err
	}

	updates := map[string]interface{}{
		"closed_at": nil,
	}
	if err := s.repo.UpdateStatus(ctx, id, model.IncidentStatusTriggered, updates); err != nil {
		return apperr.Wrap(apperr.ErrDatabase, err)
	}

	if err := s.repo.AddTimeline(ctx, &model.IncidentTimeline{
		IncidentID: id,
		Action:     model.IncidentActionReopened,
		ActorID:    &userID,
		Content:    "Incident reopened",
	}); err != nil {
		zap.L().Error("failed to add timeline", zap.Error(err), zap.Uint("incident_id", id))
	}

	s.logger.Info("incident reopened", zap.Uint("id", id), zap.Uint("user_id", userID))
	return nil
}

// Snooze puts an incident on hold until a specified time.
func (s *IncidentService) Snooze(ctx context.Context, id, userID uint, until time.Time) error {
	if until.Before(time.Now()) {
		return apperr.WithMessage(apperr.ErrInvalidParam, "snooze time must be in the future")
	}

	inc, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return apperr.ErrIncidentNotFound
		}
		return apperr.Wrap(apperr.ErrDatabase, err)
	}

	if err := validateTransition(inc.Status, model.IncidentStatusSnoozed); err != nil {
		return err
	}

	updates := map[string]interface{}{
		"snoozed_until": until,
	}
	if err := s.repo.UpdateStatus(ctx, id, model.IncidentStatusSnoozed, updates); err != nil {
		return apperr.Wrap(apperr.ErrDatabase, err)
	}

	if err := s.repo.AddTimeline(ctx, &model.IncidentTimeline{
		IncidentID: id,
		Action:     model.IncidentActionSnoozed,
		ActorID:    &userID,
		Content:    "Incident snoozed until " + until.Format(time.RFC3339),
	}); err != nil {
		zap.L().Error("failed to add timeline", zap.Error(err), zap.Uint("incident_id", id))
	}

	return nil
}

// Reassign reassigns the incident to a different user.
func (s *IncidentService) Reassign(ctx context.Context, id, userID, newAssignee uint) error {
	inc, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return apperr.ErrIncidentNotFound
		}
		return apperr.Wrap(apperr.ErrDatabase, err)
	}

	if err := validateActionAllowed("reassign", inc.Status); err != nil {
		return err
	}

	inc.AssignedTo = &newAssignee
	if err := s.repo.Update(ctx, inc); err != nil {
		return apperr.Wrap(apperr.ErrDatabase, err)
	}

	// Add new assignee record
	if err := s.repo.AddAssignee(ctx, &model.IncidentAssignee{
		IncidentID: id,
		UserID:     newAssignee,
		AssignedAt: time.Now(),
		Source:     "manual",
	}); err != nil {
		zap.L().Error("failed to add assignee", zap.Error(err), zap.Uint("incident_id", id))
	}

	if err := s.repo.AddTimeline(ctx, &model.IncidentTimeline{
		IncidentID: id,
		Action:     model.IncidentActionReassigned,
		ActorID:    &userID,
		Content:    "Incident reassigned",
	}); err != nil {
		zap.L().Error("failed to add timeline", zap.Error(err), zap.Uint("incident_id", id))
	}

	s.logger.Info("incident reassigned", zap.Uint("id", id), zap.Uint("new_assignee", newAssignee))
	return nil
}

// Merge merges a source incident into a target incident.
func (s *IncidentService) Merge(ctx context.Context, sourceID, targetID, userID uint) error {
	if sourceID == targetID {
		return apperr.WithMessage(apperr.ErrInvalidParam, "cannot merge incident into itself")
	}

	source, err := s.repo.GetByID(ctx, sourceID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return apperr.WithMessage(apperr.ErrIncidentNotFound, "source incident not found")
		}
		return apperr.Wrap(apperr.ErrDatabase, err)
	}

	target, err := s.repo.GetByID(ctx, targetID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return apperr.WithMessage(apperr.ErrIncidentNotFound, "target incident not found")
		}
		return apperr.Wrap(apperr.ErrDatabase, err)
	}

	source.MergedIntoID = &targetID
	now := time.Now()
	if err := validateTransition(source.Status, model.IncidentStatusClosed); err != nil {
		return err
	}
	source.Status = model.IncidentStatusClosed
	source.ClosedAt = &now
	if err := s.repo.Update(ctx, source); err != nil {
		return apperr.Wrap(apperr.ErrDatabase, err)
	}

	// Migrate alerts from source to target
	if s.alertRepo != nil {
		if err := s.alertRepo.BulkUpdateIncidentID(ctx, source.ID, target.ID); err != nil {
			s.logger.Error("failed to migrate alerts during merge",
				zap.Uint("source", sourceID), zap.Uint("target", targetID), zap.Error(err))
		} else {
			// Recount alerts on target
			if count, err := s.alertRepo.CountByIncidentID(ctx, target.ID); err == nil {
				target.AlertCount = count
				if err := s.repo.Update(ctx, target); err != nil {
					s.logger.Error("failed to update target alert count after merge",
						zap.Uint("target", targetID), zap.Error(err))
				}
			}
		}
	}

	if err := s.repo.AddTimeline(ctx, &model.IncidentTimeline{
		IncidentID: sourceID,
		Action:     model.IncidentActionMerged,
		ActorID:    &userID,
		Content:    "Merged into another incident",
	}); err != nil {
		zap.L().Error("failed to add timeline", zap.Error(err), zap.Uint("incident_id", sourceID))
	}

	// Record timeline on the target incident so the merge is visible there too.
	if err := s.repo.AddTimeline(ctx, &model.IncidentTimeline{
		IncidentID: targetID,
		Action:     model.IncidentActionAlertMerged,
		ActorID:    &userID,
		Content:    fmt.Sprintf("Incident #%d merged into this incident", sourceID),
	}); err != nil {
		zap.L().Error("failed to add timeline on target", zap.Error(err), zap.Uint("incident_id", targetID))
	}

	s.logger.Info("incident merged", zap.Uint("source", sourceID), zap.Uint("target", targetID))
	return nil
}

// AddComment adds a comment to the incident timeline.
func (s *IncidentService) AddComment(ctx context.Context, incidentID, userID uint, content string) error {
	_, err := s.repo.GetByID(ctx, incidentID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return apperr.ErrIncidentNotFound
		}
		return apperr.Wrap(apperr.ErrDatabase, err)
	}

	if err := s.repo.AddTimeline(ctx, &model.IncidentTimeline{
		IncidentID: incidentID,
		Action:     model.IncidentActionCommented,
		ActorID:    &userID,
		Content:    content,
	}); err != nil {
		return apperr.Wrap(apperr.ErrDatabase, err)
	}

	return nil
}

// ListTimeline returns the timeline entries for an incident.
func (s *IncidentService) ListTimeline(ctx context.Context, incidentID uint) ([]model.IncidentTimeline, error) {
	// Verify incident exists
	_, err := s.repo.GetByID(ctx, incidentID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, apperr.ErrIncidentNotFound
		}
		return nil, apperr.Wrap(apperr.ErrDatabase, err)
	}

	list, err := s.repo.ListTimeline(ctx, incidentID)
	if err != nil {
		return nil, apperr.Wrap(apperr.ErrDatabase, err)
	}
	return list, nil
}

// ListAssignees returns all assignees for an incident.
func (s *IncidentService) ListAssignees(ctx context.Context, incidentID uint) ([]model.IncidentAssignee, error) {
	list, err := s.repo.ListAssignees(ctx, incidentID)
	if err != nil {
		return nil, apperr.Wrap(apperr.ErrDatabase, err)
	}
	return list, nil
}

// Escalate moves the incident to the next escalation step.
func (s *IncidentService) Escalate(ctx context.Context, id, userID uint) error {
	inc, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return apperr.ErrIncidentNotFound
		}
		return apperr.Wrap(apperr.ErrDatabase, err)
	}

	if err := validateActionAllowed("escalate", inc.Status); err != nil {
		return err
	}

	// Upper-bound check: ensure we don't exceed the escalation policy's step count.
	if s.escStepRepo != nil && inc.EscalationPolicyID != nil {
		steps, err := s.escStepRepo.ListByPolicyID(ctx, *inc.EscalationPolicyID)
		if err == nil && inc.CurrentEscalationStep >= len(steps) {
			return apperr.WithMessage(apperr.ErrBusiness, "already at the last escalation step")
		}
	}

	inc.CurrentEscalationStep++
	if err := s.repo.Update(ctx, inc); err != nil {
		return apperr.Wrap(apperr.ErrDatabase, err)
	}

	if err := s.repo.AddTimeline(ctx, &model.IncidentTimeline{
		IncidentID: id,
		Action:     model.IncidentActionEscalated,
		ActorID:    &userID,
		Content:    "Incident escalated",
	}); err != nil {
		zap.L().Error("failed to add timeline", zap.Error(err), zap.Uint("incident_id", id))
	}

	s.logger.Info("incident escalated", zap.Uint("id", id), zap.Int("step", inc.CurrentEscalationStep))
	return nil
}

// CloseExpiredIncidents closes all incidents that have exceeded their channel's
// auto_close_minutes timeout. Called by StartAutoCloseWorker.
func (s *IncidentService) CloseExpiredIncidents(ctx context.Context) {
	now := time.Now()
	incidents, err := s.repo.ListForAutoClose(ctx, now)
	if err != nil {
		s.logger.Error("auto-close: failed to list incidents", zap.Error(err))
		return
	}
	for _, inc := range incidents {
		updates := map[string]interface{}{
			"closed_at": now,
		}
		if err := s.repo.UpdateStatus(ctx, inc.ID, model.IncidentStatusClosed, updates); err != nil {
			s.logger.Error("auto-close: failed to close incident",
				zap.Uint("id", inc.ID), zap.Error(err))
			continue
		}
		if err := s.repo.AddTimeline(ctx, &model.IncidentTimeline{
			IncidentID: inc.ID,
			Action:     model.IncidentActionClosed,
			Content:    "Incident auto-closed due to timeout",
		}); err != nil {
			zap.L().Error("failed to add timeline", zap.Error(err), zap.Uint("incident_id", inc.ID))
		}
		s.logger.Info("auto-close: incident closed", zap.Uint("id", inc.ID))
	}
}

// StartAutoCloseWorker starts a background goroutine that periodically closes
// timed-out incidents. It stops when ctx is cancelled.
func (s *IncidentService) StartAutoCloseWorker(ctx context.Context) {
	go func() {
		defer func() {
			if r := recover(); r != nil {
				s.logger.Error("incident auto-close worker panic recovered", zap.Any("recover", r), zap.String("stack", string(debug.Stack())))
			}
		}()
		ticker := time.NewTicker(incidentAutoCloseInterval)
		defer ticker.Stop()
		s.logger.Info("incident auto-close worker started",
			zap.Duration("interval", incidentAutoCloseInterval))
		for {
			select {
			case <-ticker.C:
				s.CloseExpiredIncidents(ctx)
			case <-ctx.Done():
				s.logger.Info("incident auto-close worker stopped")
				return
			}
		}
	}()
}
