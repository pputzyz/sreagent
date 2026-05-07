package service

import (
	"context"
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/sreagent/sreagent/internal/model"
	apperr "github.com/sreagent/sreagent/internal/pkg/errors"
	"github.com/sreagent/sreagent/internal/repository"
)

// IncidentService provides business logic for incidents (故障).
type IncidentService struct {
	repo       *repository.IncidentRepository
	channelSvc *ChannelService
	logger     *zap.Logger
}

func NewIncidentService(repo *repository.IncidentRepository, channelSvc *ChannelService, logger *zap.Logger) *IncidentService {
	return &IncidentService{repo: repo, channelSvc: channelSvc, logger: logger}
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
	_ = s.repo.AddTimeline(ctx, &model.IncidentTimeline{
		IncidentID: inc.ID,
		Action:     model.IncidentActionTriggered,
		Content:    "Incident triggered",
	})

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

// Acknowledge marks the incident as processing and records the ack.
func (s *IncidentService) Acknowledge(ctx context.Context, id, userID uint) error {
	inc, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return apperr.ErrIncidentNotFound
		}
		return apperr.Wrap(apperr.ErrDatabase, err)
	}

	if inc.Status == model.IncidentStatusClosed {
		return apperr.WithMessage(apperr.ErrBadRequest, "cannot acknowledge a closed incident")
	}

	now := time.Now()
	updates := map[string]interface{}{
		"acknowledged_at": now,
	}
	if err := s.repo.UpdateStatus(ctx, id, model.IncidentStatusProcessing, updates); err != nil {
		return apperr.Wrap(apperr.ErrDatabase, err)
	}

	// Record assignee ack
	_ = s.repo.AcknowledgeAssignee(ctx, id, userID)

	// Timeline
	_ = s.repo.AddTimeline(ctx, &model.IncidentTimeline{
		IncidentID: id,
		Action:     model.IncidentActionAcknowledged,
		ActorID:    &userID,
		Content:    "Incident acknowledged",
	})

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

	now := time.Now()
	updates := map[string]interface{}{
		"closed_at": now,
	}
	if err := s.repo.UpdateStatus(ctx, id, model.IncidentStatusClosed, updates); err != nil {
		return apperr.Wrap(apperr.ErrDatabase, err)
	}

	_ = s.repo.AddTimeline(ctx, &model.IncidentTimeline{
		IncidentID: id,
		Action:     model.IncidentActionClosed,
		ActorID:    &userID,
		Content:    "Incident closed",
	})

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

	if inc.Status != model.IncidentStatusClosed {
		return apperr.WithMessage(apperr.ErrBadRequest, "can only reopen a closed incident")
	}

	updates := map[string]interface{}{
		"closed_at": nil,
	}
	if err := s.repo.UpdateStatus(ctx, id, model.IncidentStatusTriggered, updates); err != nil {
		return apperr.Wrap(apperr.ErrDatabase, err)
	}

	_ = s.repo.AddTimeline(ctx, &model.IncidentTimeline{
		IncidentID: id,
		Action:     model.IncidentActionReopened,
		ActorID:    &userID,
		Content:    "Incident reopened",
	})

	s.logger.Info("incident reopened", zap.Uint("id", id), zap.Uint("user_id", userID))
	return nil
}

// Snooze puts an incident on hold until a specified time.
func (s *IncidentService) Snooze(ctx context.Context, id, userID uint, until time.Time) error {
	_, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return apperr.ErrIncidentNotFound
		}
		return apperr.Wrap(apperr.ErrDatabase, err)
	}

	updates := map[string]interface{}{
		"snoozed_until": until,
	}
	// Keep current status; snooze is informational
	if err := s.repo.UpdateStatus(ctx, id, model.IncidentStatusProcessing, updates); err != nil {
		return apperr.Wrap(apperr.ErrDatabase, err)
	}

	_ = s.repo.AddTimeline(ctx, &model.IncidentTimeline{
		IncidentID: id,
		Action:     model.IncidentActionSnoozed,
		ActorID:    &userID,
		Content:    "Incident snoozed until " + until.Format(time.RFC3339),
	})

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

	inc.AssignedTo = &newAssignee
	if err := s.repo.Update(ctx, inc); err != nil {
		return apperr.Wrap(apperr.ErrDatabase, err)
	}

	// Add new assignee record
	_ = s.repo.AddAssignee(ctx, &model.IncidentAssignee{
		IncidentID: id,
		UserID:     newAssignee,
		AssignedAt: time.Now(),
		Source:     "manual",
	})

	_ = s.repo.AddTimeline(ctx, &model.IncidentTimeline{
		IncidentID: id,
		Action:     model.IncidentActionReassigned,
		ActorID:    &userID,
		Content:    "Incident reassigned",
	})

	s.logger.Info("incident reassigned", zap.Uint("id", id), zap.Uint("new_assignee", newAssignee))
	return nil
}

// Merge merges a source incident into a target incident.
func (s *IncidentService) Merge(ctx context.Context, sourceID, targetID, userID uint) error {
	source, err := s.repo.GetByID(ctx, sourceID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return apperr.WithMessage(apperr.ErrIncidentNotFound, "source incident not found")
		}
		return apperr.Wrap(apperr.ErrDatabase, err)
	}

	_, err = s.repo.GetByID(ctx, targetID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return apperr.WithMessage(apperr.ErrIncidentNotFound, "target incident not found")
		}
		return apperr.Wrap(apperr.ErrDatabase, err)
	}

	source.MergedIntoID = &targetID
	now := time.Now()
	source.Status = model.IncidentStatusClosed
	source.ClosedAt = &now
	if err := s.repo.Update(ctx, source); err != nil {
		return apperr.Wrap(apperr.ErrDatabase, err)
	}

	_ = s.repo.AddTimeline(ctx, &model.IncidentTimeline{
		IncidentID: sourceID,
		Action:     model.IncidentActionMerged,
		ActorID:    &userID,
		Content:    "Merged into another incident",
	})

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

	inc.CurrentEscalationStep++
	if err := s.repo.Update(ctx, inc); err != nil {
		return apperr.Wrap(apperr.ErrDatabase, err)
	}

	_ = s.repo.AddTimeline(ctx, &model.IncidentTimeline{
		IncidentID: id,
		Action:     model.IncidentActionEscalated,
		ActorID:    &userID,
		Content:    "Incident escalated",
	})

	s.logger.Info("incident escalated", zap.Uint("id", id), zap.Int("step", inc.CurrentEscalationStep))
	return nil
}
