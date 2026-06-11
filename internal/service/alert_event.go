package service

import (
	"context"
	"fmt"
	"time"

	"go.uber.org/zap"

	"github.com/sreagent/sreagent/internal/model"
	apperr "github.com/sreagent/sreagent/internal/pkg/errors"
	"github.com/sreagent/sreagent/internal/repository"
)

// AlertEventFilter holds the parameters for filtering alert events.
// Defined in service layer so handlers don't need to import repository.
type AlertEventFilter = repository.AlertEventFilter

const defaultDispatchConcurrency = 100

// OnCallResolver is used by AlertEventService to find the current on-call person.
type OnCallResolver interface {
	GetCurrentOnCallForAlert(ctx context.Context, alertLabels map[string]string) (*model.User, error)
}

// AlertWorkerPool is a bounded executor for alert processing callbacks.
type AlertWorkerPool interface {
	Submit(ctx context.Context, fn func(context.Context)) bool
}

type AlertEventService struct {
	repo               *repository.AlertEventRepository
	timelineRepo       *repository.AlertTimelineRepository
	userRepo           *repository.UserRepository
	notifySvc          *NotificationService
	onCallSvc          OnCallResolver
	larkSvc            *LarkService
	incidentAggregator *IncidentAggregator // P1-03: bridges resolve/close to incident lifecycle
	workerPool         AlertWorkerPool
	dispatchSem        chan struct{} // bounds goroutines when no worker pool is configured
	logger             *zap.Logger
	serverCtx          context.Context // server lifecycle context for background goroutines
}


// AlertGroupRawRow is a single row from the grouped alert query.
type AlertGroupRawRow struct {
	AlertName    string
	Source       string
	Severity     string
	Status       string
	Cnt          int64
	LatestFired  time.Time
	OldestFired  time.Time
	MaxFireCount int
}

// ListGrouped returns alert events grouped by (alert_name, source, severity, status).
// Filters: statuses and severities are optional (empty = all).
func (s *AlertEventService) ListGrouped(ctx context.Context, statuses, severities []string) ([]AlertGroupRawRow, error) {
	q := s.repo.DB().WithContext(ctx).Model(&model.AlertEvent{}).
		Select(`alert_name, source, severity, status,
			COUNT(*) AS cnt,
			MAX(fired_at) AS latest_fired,
			MIN(fired_at) AS oldest_fired,
			MAX(fire_count) AS max_fire_count`).
		Where("deleted_at IS NULL")

	if len(statuses) > 0 {
		q = q.Where("status IN ?", statuses)
	}
	if len(severities) > 0 {
		q = q.Where("severity IN ?", severities)
	}

	var rows []AlertGroupRawRow
	if err := q.Group("alert_name, source, severity, status").
		Order("latest_fired DESC").
		Scan(&rows).Error; err != nil {
		return nil, apperr.Wrap(apperr.ErrDatabase, err)
	}
	return rows, nil
}

func NewAlertEventService(
	repo *repository.AlertEventRepository,
	timelineRepo *repository.AlertTimelineRepository,
	userRepo *repository.UserRepository,
	notifySvc *NotificationService,
	onCallSvc OnCallResolver,
	larkSvc *LarkService,
	workerPool AlertWorkerPool,
	logger *zap.Logger,
) *AlertEventService {
	return &AlertEventService{
		repo:         repo,
		timelineRepo: timelineRepo,
		userRepo:     userRepo,
		notifySvc:    notifySvc,
		onCallSvc:    onCallSvc,
		larkSvc:      larkSvc,
		workerPool:   workerPool,
		dispatchSem:  make(chan struct{}, defaultDispatchConcurrency),
		logger:       logger,
	}
}

// SetIncidentAggregator attaches an IncidentAggregator for P1-03 manual resolve/close linking.
func (s *AlertEventService) SetIncidentAggregator(agg *IncidentAggregator) {
	s.incidentAggregator = agg
}

// WithServerContext sets the server lifecycle context for background goroutines.
func (s *AlertEventService) WithServerContext(ctx context.Context) {
	s.serverCtx = ctx
}

// bgCtx returns the server context if set, otherwise context.Background().
func (s *AlertEventService) bgCtx() context.Context {
	if s.serverCtx != nil {
		return s.serverCtx
	}
	return context.Background()
}

func (s *AlertEventService) List(ctx context.Context, status, severity string, page, pageSize int) ([]model.AlertEvent, int64, error) {
	return s.repo.List(ctx, status, severity, page, pageSize)
}

// ListWithFilter returns alert events using the advanced filter (view mode, time range, etc.).
func (s *AlertEventService) ListWithFilter(ctx context.Context, filter repository.AlertEventFilter) ([]model.AlertEvent, int64, error) {
	return s.repo.ListWithFilter(ctx, filter)
}

func (s *AlertEventService) GetByID(ctx context.Context, id uint) (*model.AlertEvent, error) {
	event, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, apperr.ErrEventNotFound
	}
	return event, nil
}

// Acknowledge marks an alert as acknowledged.
func (s *AlertEventService) Acknowledge(ctx context.Context, eventID, userID uint) error {
	now := time.Now()
	ok, err := s.repo.TransitionStatus(ctx, eventID,
		[]model.AlertEventStatus{model.EventStatusFiring, model.EventStatusAssigned},
		map[string]interface{}{
			"status":   model.EventStatusAcknowledged,
			"acked_by": userID,
			"acked_at": now,
		})
	if err != nil {
		return apperr.Wrap(apperr.ErrDatabase, err)
	}
	if !ok {
		return apperr.WithMessage(apperr.ErrBadRequest, "alert is not in a state that can be acknowledged")
	}

	event, err := s.repo.GetByID(ctx, eventID)
	if err != nil {
		s.logger.Warn("failed to fetch event for Lark update after acknowledge",
			zap.Uint("event_id", eventID), zap.Error(err))
	}

	// Add timeline entry
	s.addTimeline(ctx, eventID, model.TimelineActionAcknowledged, &userID, "Alert acknowledged")

	if event != nil {
		s.triggerLarkCardUpdate(event)
	}
	return nil
}

// Assign assigns an alert to a specific user.
func (s *AlertEventService) Assign(ctx context.Context, eventID, assignTo, operatorID uint, note string) error {
	// Validate that the target user exists.
	if _, err := s.userRepo.GetByID(ctx, assignTo); err != nil {
		return apperr.ErrNotFound
	}

	// Use atomic CAS (TransitionStatus) to prevent race conditions.
	ok, err := s.repo.TransitionStatus(ctx, eventID,
		[]model.AlertEventStatus{model.EventStatusFiring, model.EventStatusAcknowledged, model.EventStatusAssigned},
		map[string]interface{}{
			"status":      model.EventStatusAssigned,
			"assigned_to": assignTo,
		})
	if err != nil {
		return apperr.Wrap(apperr.ErrDatabase, err)
	}
	if !ok {
		return apperr.WithMessage(apperr.ErrBadRequest, "alert cannot be assigned from current state")
	}

	event, err := s.repo.GetByID(ctx, eventID)
	if err != nil {
		s.logger.Warn("failed to fetch event for Lark update after assign",
			zap.Uint("event_id", eventID), zap.Error(err))
	}

	if note == "" {
		note = "Alert assigned"
	}
	s.addTimeline(ctx, eventID, model.TimelineActionAssigned, &operatorID, note)

	if event != nil {
		s.triggerLarkCardUpdate(event)
	}

	return nil
}

// Resolve marks an alert as resolved.
func (s *AlertEventService) Resolve(ctx context.Context, eventID, userID uint, resolution string) error {
	now := time.Now()
	ok, err := s.repo.TransitionStatus(ctx, eventID,
		[]model.AlertEventStatus{
			model.EventStatusFiring,
			model.EventStatusAcknowledged,
			model.EventStatusAssigned,
			model.EventStatusSilenced,
		},
		map[string]interface{}{
			"status":      model.EventStatusResolved,
			"resolved_at": now,
			"resolution":  resolution,
		})
	if err != nil {
		return apperr.Wrap(apperr.ErrDatabase, err)
	}
	if !ok {
		return apperr.WithMessage(apperr.ErrBadRequest, "alert cannot be resolved from current state")
	}

	event, err := s.repo.GetByID(ctx, eventID)
	if err != nil {
		s.logger.Warn("failed to fetch event for Lark update after resolve",
			zap.Uint("event_id", eventID), zap.Error(err))
	}
	s.addTimeline(ctx, eventID, model.TimelineActionResolved, &userID, resolution)
	if event != nil {
		s.triggerLarkCardUpdate(event)
		// P1-03: Notify incident aggregator on manual resolve
		if s.incidentAggregator != nil {
			s.incidentAggregator.OnEventResolved(ctx, event)
		}
	}
	return nil
}

// Close marks an alert as closed.
func (s *AlertEventService) Close(ctx context.Context, eventID, userID uint, note string) error {
	now := time.Now()
	ok, err := s.repo.TransitionStatus(ctx, eventID,
		[]model.AlertEventStatus{
			model.EventStatusFiring,
			model.EventStatusAcknowledged,
			model.EventStatusAssigned,
			model.EventStatusSilenced,
			model.EventStatusResolved,
		},
		map[string]interface{}{
			"status":    model.EventStatusClosed,
			"closed_at": now,
		})
	if err != nil {
		return apperr.Wrap(apperr.ErrDatabase, err)
	}
	if !ok {
		return apperr.WithMessage(apperr.ErrBadRequest, "alert cannot be closed from current state")
	}

	if note == "" {
		note = "Alert closed"
	}
	event, err := s.repo.GetByID(ctx, eventID)
	if err != nil {
		s.logger.Warn("failed to fetch event for Lark update after close",
			zap.Uint("event_id", eventID), zap.Error(err))
	}
	s.addTimeline(ctx, eventID, model.TimelineActionClosed, &userID, note)
	if event != nil {
		s.triggerLarkCardUpdate(event)
		// P1-03: Notify incident aggregator on manual close
		if s.incidentAggregator != nil {
			s.incidentAggregator.OnEventResolved(ctx, event)
		}
	}
	return nil
}

// Silence silences an alert for a specified duration.
func (s *AlertEventService) Silence(ctx context.Context, eventID, userID uint, durationMinutes int, reason string) error {
	now := time.Now()
	silencedUntil := now.Add(time.Duration(durationMinutes) * time.Minute)

	// Use atomic CAS (TransitionStatus) to prevent race conditions.
	ok, err := s.repo.TransitionStatus(ctx, eventID,
		[]model.AlertEventStatus{model.EventStatusFiring, model.EventStatusAcknowledged, model.EventStatusAssigned},
		map[string]interface{}{
			"status":          model.EventStatusSilenced,
			"silenced_until":  silencedUntil,
			"silence_reason":  reason,
		})
	if err != nil {
		return apperr.Wrap(apperr.ErrDatabase, err)
	}
	if !ok {
		return apperr.WithMessage(apperr.ErrBadRequest, "alert cannot be silenced from current state")
	}

	event, err := s.repo.GetByID(ctx, eventID)
	if err != nil {
		s.logger.Warn("failed to fetch event for Lark update after silence",
			zap.Uint("event_id", eventID), zap.Error(err))
	}

	note := fmt.Sprintf("Alert silenced for %d minutes. Reason: %s", durationMinutes, reason)
	s.addTimeline(ctx, eventID, model.TimelineActionSilenced, &userID, note)

	if event != nil {
		s.triggerLarkCardUpdate(event)
	}
	return nil
}

// AddComment adds a comment to the alert timeline.
func (s *AlertEventService) AddComment(ctx context.Context, eventID, userID uint, note string) error {
	if _, err := s.repo.GetByID(ctx, eventID); err != nil {
		return apperr.ErrEventNotFound
	}

	s.addTimeline(ctx, eventID, model.TimelineActionCommented, &userID, note)
	return nil
}

// GetTimeline returns the timeline for an alert event.
func (s *AlertEventService) GetTimeline(ctx context.Context, eventID uint) ([]model.AlertTimeline, error) {
	return s.timelineRepo.ListByEventID(ctx, eventID)
}

// GetTimelinePaged returns a paginated timeline for an alert event.
func (s *AlertEventService) GetTimelinePaged(ctx context.Context, eventID uint, page, pageSize int) ([]model.AlertTimeline, int64, error) {
	return s.timelineRepo.ListByEventIDPaged(ctx, eventID, page, pageSize)
}
