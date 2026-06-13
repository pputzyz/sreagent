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
	"github.com/sreagent/sreagent/internal/pkg/metrics"
	"github.com/sreagent/sreagent/internal/repository"
)

const incidentAutoCloseInterval = 5 * time.Minute

// IncidentService provides business logic for incidents (故障).
type IncidentService struct {
	repo        *repository.IncidentRepository
	channelSvc  *ChannelService
	alertRepo   *repository.AlertRepository          // optional, for merge alert migration
	escStepRepo *repository.EscalationStepRepository // optional, for escalation upper-bound check
	logger      *zap.Logger

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
//	ack (processing)  → open (triggered), close (closed), snooze (snoozed)
//	close (closed)    → reopen (triggered)
//	snooze (snoozed)  → open (triggered), close (closed)
var validTransitions = map[model.IncidentStatus][]model.IncidentStatus{
	model.IncidentStatusTriggered:  {model.IncidentStatusProcessing, model.IncidentStatusClosed, model.IncidentStatusSnoozed},
	model.IncidentStatusProcessing: {model.IncidentStatusTriggered, model.IncidentStatusClosed, model.IncidentStatusSnoozed},
	model.IncidentStatusClosed:     {model.IncidentStatusTriggered},
	model.IncidentStatusSnoozed:    {model.IncidentStatusTriggered, model.IncidentStatusClosed},
}

// allowedActionStates defines which statuses allow non-status-changing actions.
var allowedActionStates = map[string][]model.IncidentStatus{
	"reassign": {model.IncidentStatusTriggered, model.IncidentStatusProcessing},
	"escalate": {model.IncidentStatusTriggered, model.IncidentStatusProcessing},
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
		s.logger.Error("failed to add timeline", zap.Error(err), zap.Uint("incident_id", inc.ID))
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

// CanAccess reports whether a non-admin user (identified by their team IDs) may
// act on the given incident. Admins bypass this check at the handler layer.
// Mirrors the team isolation enforced by ListScoped so single-resource operations
// cannot be used to bypass team boundaries (IDOR).
func (s *IncidentService) CanAccess(ctx context.Context, id uint, teamIDs []uint) (bool, error) {
	ok, err := s.repo.IncidentInTeams(ctx, id, teamIDs)
	if err != nil {
		return false, apperr.Wrap(apperr.ErrDatabase, err)
	}
	return ok, nil
}

// Acknowledge marks the incident as processing and records the ack.
// Uses atomic CAS: UPDATE ... WHERE status = 'triggered'. No read-then-write race.
func (s *IncidentService) Acknowledge(ctx context.Context, id, userID uint) error {
	now := time.Now()
	updates := map[string]interface{}{
		"acknowledged_at": now,
	}
	err := s.repo.TransitionStatus(ctx, id, model.IncidentStatusTriggered, model.IncidentStatusProcessing, updates)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			// Could be: not found, or concurrent status change. Check which.
			inc, getErr := s.repo.GetByID(ctx, id)
			if getErr != nil {
				return apperr.ErrIncidentNotFound
			}
			return validateTransition(inc.Status, model.IncidentStatusProcessing)
		}
		return apperr.Wrap(apperr.ErrDatabase, err)
	}

	// Record assignee ack
	if err := s.repo.AcknowledgeAssignee(ctx, id, userID); err != nil {
		s.logger.Error("failed to acknowledge assignee", zap.Error(err), zap.Uint("incident_id", id))
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
		s.logger.Error("failed to add timeline", zap.Error(err), zap.Uint("incident_id", id))
	}

	s.logger.Info("incident acknowledged", zap.Uint("id", id), zap.Uint("user_id", userID))
	return nil
}

// Close closes an incident.
// Uses atomic CAS: UPDATE ... WHERE status = expected. No read-then-write race.
func (s *IncidentService) Close(ctx context.Context, id, userID uint) error {
	now := time.Now()
	updates := map[string]interface{}{
		"closed_at": now,
	}
	// Try from triggered first (most common), then processing, then snoozed.
	for _, from := range []model.IncidentStatus{
		model.IncidentStatusTriggered,
		model.IncidentStatusProcessing,
		model.IncidentStatusSnoozed,
	} {
		err := s.repo.TransitionStatus(ctx, id, from, model.IncidentStatusClosed, updates)
		if err == nil {
			// Success
			if s.onStatusChange != nil {
				s.onStatusChange(ctx, id, model.IncidentStatusClosed)
			}
			if addErr := s.repo.AddTimeline(ctx, &model.IncidentTimeline{
				IncidentID: id,
				Action:     model.IncidentActionClosed,
				ActorID:    &userID,
				Content:    "Incident closed",
			}); addErr != nil {
				s.logger.Error("failed to add timeline", zap.Error(addErr), zap.Uint("incident_id", id))
			}
			s.logger.Info("incident closed", zap.Uint("id", id), zap.Uint("user_id", userID))
			return nil
		}
		if err != gorm.ErrRecordNotFound {
			return apperr.Wrap(apperr.ErrDatabase, err)
		}
	}
	// All attempts returned ErrRecordNotFound: either not found or already closed.
	inc, getErr := s.repo.GetByID(ctx, id)
	if getErr != nil {
		return apperr.ErrIncidentNotFound
	}
	if inc.Status == model.IncidentStatusClosed {
		return nil // idempotent
	}
	return validateTransition(inc.Status, model.IncidentStatusClosed)
}

// Reopen re-opens a closed incident.
// Uses atomic CAS: UPDATE ... WHERE status = 'closed'. No read-then-write race.
func (s *IncidentService) Reopen(ctx context.Context, id, userID uint) error {
	updates := map[string]interface{}{
		"closed_at":               nil,
		"current_escalation_step": 0,
	}
	err := s.repo.TransitionStatus(ctx, id, model.IncidentStatusClosed, model.IncidentStatusTriggered, updates)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			inc, getErr := s.repo.GetByID(ctx, id)
			if getErr != nil {
				return apperr.ErrIncidentNotFound
			}
			return validateTransition(inc.Status, model.IncidentStatusTriggered)
		}
		return apperr.Wrap(apperr.ErrDatabase, err)
	}

	if err := s.repo.AddTimeline(ctx, &model.IncidentTimeline{
		IncidentID: id,
		Action:     model.IncidentActionReopened,
		ActorID:    &userID,
		Content:    "Incident reopened",
	}); err != nil {
		s.logger.Error("failed to add timeline", zap.Error(err), zap.Uint("incident_id", id))
	}

	s.logger.Info("incident reopened", zap.Uint("id", id), zap.Uint("user_id", userID))
	return nil
}

// Snooze puts an incident on hold until a specified time.
// Uses atomic CAS: UPDATE ... WHERE status = 'triggered'. No read-then-write race.
func (s *IncidentService) Snooze(ctx context.Context, id, userID uint, until time.Time) error {
	if until.Before(time.Now()) {
		return apperr.WithMessage(apperr.ErrInvalidParam, "snooze time must be in the future")
	}

	updates := map[string]interface{}{
		"snoozed_until": until,
	}
	// Allow snooze from both triggered and processing states
	err := s.repo.TransitionStatus(ctx, id, model.IncidentStatusTriggered, model.IncidentStatusSnoozed, updates)
	if err != nil {
		// Try from processing state
		err = s.repo.TransitionStatus(ctx, id, model.IncidentStatusProcessing, model.IncidentStatusSnoozed, updates)
		if err != nil {
			if err == gorm.ErrRecordNotFound {
				inc, getErr := s.repo.GetByID(ctx, id)
				if getErr != nil {
					return apperr.ErrIncidentNotFound
				}
				return validateTransition(inc.Status, model.IncidentStatusSnoozed)
			}
			return apperr.Wrap(apperr.ErrDatabase, err)
		}
	}

	if err := s.repo.AddTimeline(ctx, &model.IncidentTimeline{
		IncidentID: id,
		Action:     model.IncidentActionSnoozed,
		ActorID:    &userID,
		Content:    "Incident snoozed until " + until.Format(time.RFC3339),
	}); err != nil {
		s.logger.Error("failed to add timeline", zap.Error(err), zap.Uint("incident_id", id))
	}

	return nil
}

// Reassign reassigns the incident to a different user.
// Uses a targeted column update on assigned_to to avoid lost-update races
// that would occur with a full-row Save().
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

	if err := s.repo.UpdateAssignees(ctx, id, &newAssignee); err != nil {
		return apperr.Wrap(apperr.ErrDatabase, err)
	}

	// Add new assignee record
	if err := s.repo.AddAssignee(ctx, &model.IncidentAssignee{
		IncidentID: id,
		UserID:     newAssignee,
		AssignedAt: time.Now(),
		Source:     "manual",
	}); err != nil {
		s.logger.Error("failed to add assignee", zap.Error(err), zap.Uint("incident_id", id))
	}

	if err := s.repo.AddTimeline(ctx, &model.IncidentTimeline{
		IncidentID: id,
		Action:     model.IncidentActionReassigned,
		ActorID:    &userID,
		Content:    "Incident reassigned",
	}); err != nil {
		s.logger.Error("failed to add timeline", zap.Error(err), zap.Uint("incident_id", id))
	}

	s.logger.Info("incident reassigned", zap.Uint("id", id), zap.Uint("new_assignee", newAssignee))
	return nil
}

// Merge merges a source incident into a target incident.
// All steps (close source, migrate alerts, update target count, add timelines)
// are wrapped in a single database transaction for atomicity.
// The transaction is propagated via context — no shared mutable state.
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

	if err := validateTransition(source.Status, model.IncidentStatusClosed); err != nil {
		return err
	}

	err = s.repo.Transaction(ctx, func(ctx context.Context) error {
		// Close source incident
		source.MergedIntoID = &targetID
		now := time.Now()
		source.Status = model.IncidentStatusClosed
		source.ClosedAt = &now
		if err := s.repo.Update(ctx, source); err != nil {
			return apperr.Wrap(apperr.ErrDatabase, err)
		}

		// Migrate alerts from source to target (alertRepo uses context tx)
		if s.alertRepo != nil {
			if err := s.alertRepo.BulkUpdateIncidentID(ctx, source.ID, target.ID); err != nil {
				s.logger.Error("failed to migrate alerts during merge",
					zap.Uint("source", sourceID), zap.Uint("target", targetID), zap.Error(err))
				return apperr.Wrap(apperr.ErrDatabase, err)
			}
			// Recount alerts on target
			count, err := s.alertRepo.CountByIncidentID(ctx, target.ID)
			if err != nil {
				s.logger.Error("failed to count alerts on target after merge",
					zap.Uint("target", targetID), zap.Error(err))
				return apperr.Wrap(apperr.ErrDatabase, err)
			}
			target.AlertCount = count
			if err := s.repo.Update(ctx, target); err != nil {
				s.logger.Error("failed to update target alert count after merge",
					zap.Uint("target", targetID), zap.Error(err))
				return apperr.Wrap(apperr.ErrDatabase, err)
			}
		}

		// Timeline on source
		if err := s.repo.AddTimeline(ctx, &model.IncidentTimeline{
			IncidentID: sourceID,
			Action:     model.IncidentActionMerged,
			ActorID:    &userID,
			Content:    "Merged into another incident",
		}); err != nil {
			s.logger.Error("failed to add timeline", zap.Error(err), zap.Uint("incident_id", sourceID))
			return apperr.Wrap(apperr.ErrDatabase, err)
		}

		// Timeline on target
		if err := s.repo.AddTimeline(ctx, &model.IncidentTimeline{
			IncidentID: targetID,
			Action:     model.IncidentActionAlertMerged,
			ActorID:    &userID,
			Content:    fmt.Sprintf("Incident #%d merged into this incident", sourceID),
		}); err != nil {
			s.logger.Error("failed to add timeline on target", zap.Error(err), zap.Uint("incident_id", targetID))
			return apperr.Wrap(apperr.ErrDatabase, err)
		}

		return nil
	})

	if err != nil {
		return err
	}

	s.logger.Info("incident merged", zap.Uint("source", sourceID), zap.Uint("target", targetID))
	return nil
}

// BulkAcknowledge acknowledges multiple incidents.
func (s *IncidentService) BulkAcknowledge(ctx context.Context, ids []uint, userID uint) error {
	for _, id := range ids {
		if err := s.Acknowledge(ctx, id, userID); err != nil {
			s.logger.Error("bulk acknowledge: failed on incident",
				zap.Uint("id", id), zap.Error(err))
		}
	}
	return nil
}

// BulkClose closes multiple incidents.
func (s *IncidentService) BulkClose(ctx context.Context, ids []uint, userID uint) error {
	for _, id := range ids {
		if err := s.Close(ctx, id, userID); err != nil {
			s.logger.Error("bulk close: failed on incident",
				zap.Uint("id", id), zap.Error(err))
		}
	}
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
// Uses atomic CAS on current_escalation_step to prevent concurrent double-escalation.
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

	// Atomic CAS: only update if current_escalation_step hasn't changed since we read it.
	expectedStep := inc.CurrentEscalationStep
	newStep := expectedStep + 1
	if err := s.repo.TransitionEscalationStep(ctx, id, expectedStep, newStep); err != nil {
		if err == gorm.ErrRecordNotFound {
			// Another goroutine escalated concurrently. Re-read and report current state.
			current, getErr := s.repo.GetByID(ctx, id)
			if getErr != nil {
				return apperr.Wrap(apperr.ErrDatabase, getErr)
			}
			return apperr.WithMessage(apperr.ErrVersionConflict,
				fmt.Sprintf("escalation step was concurrently modified (expected %d, now %d)",
					expectedStep, current.CurrentEscalationStep))
		}
		return apperr.Wrap(apperr.ErrDatabase, err)
	}

	if err := s.repo.AddTimeline(ctx, &model.IncidentTimeline{
		IncidentID: id,
		Action:     model.IncidentActionEscalated,
		ActorID:    &userID,
		Content:    "Incident escalated",
	}); err != nil {
		s.logger.Error("failed to add timeline", zap.Error(err), zap.Uint("incident_id", id))
	}

	s.logger.Info("incident escalated", zap.Uint("id", id), zap.Int("step", newStep))
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
			s.logger.Error("failed to add timeline", zap.Error(err), zap.Uint("incident_id", inc.ID))
		}
		metrics.IncIncidentAutoClose()
		s.logger.Info("auto-close: incident closed", zap.Uint("id", inc.ID))
	}
}

// WakeExpiredSnoozed transitions snoozed incidents whose snooze window has elapsed
// back to "triggered" so escalation and repeat notifications resume. Without this,
// a snoozed incident would stay snoozed forever (it is also excluded from auto-close).
func (s *IncidentService) WakeExpiredSnoozed(ctx context.Context) {
	now := time.Now()
	incidents, err := s.repo.ListExpiredSnoozed(ctx, now)
	if err != nil {
		s.logger.Error("snooze-wake: failed to list expired snoozed incidents", zap.Error(err))
		return
	}
	for _, inc := range incidents {
		// CAS from snoozed -> triggered, clearing the snooze timestamp.
		updates := map[string]interface{}{"snoozed_until": nil}
		if err := s.repo.TransitionStatus(ctx, inc.ID, model.IncidentStatusSnoozed, model.IncidentStatusTriggered, updates); err != nil {
			if err != gorm.ErrRecordNotFound { // not-found = status changed concurrently; skip quietly
				s.logger.Error("snooze-wake: failed to wake incident", zap.Uint("id", inc.ID), zap.Error(err))
			}
			continue
		}
		if err := s.repo.AddTimeline(ctx, &model.IncidentTimeline{
			IncidentID: inc.ID,
			Action:     model.IncidentActionReopened,
			Content:    "Incident snooze expired; returned to triggered",
		}); err != nil {
			s.logger.Error("failed to add timeline", zap.Error(err), zap.Uint("incident_id", inc.ID))
		}
		s.logger.Info("snooze-wake: incident returned to triggered", zap.Uint("id", inc.ID))
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
				s.WakeExpiredSnoozed(ctx)
				s.CloseExpiredIncidents(ctx)
			case <-ctx.Done():
				s.logger.Info("incident auto-close worker stopped")
				return
			}
		}
	}()
}
