package service

import (
	"context"

	"go.uber.org/zap"

	"github.com/sreagent/sreagent/internal/model"
	apperr "github.com/sreagent/sreagent/internal/pkg/errors"
)

// BatchAcknowledge acknowledges multiple firing alerts in a single DB round-trip.
// Returns the number of rows updated (success) and the number of IDs that were
// not in firing state (failed = len(ids) - rows_affected).
func (s *AlertEventService) BatchAcknowledge(ctx context.Context, eventIDs []uint, userID uint) (success int, failed int, err error) {
	if len(eventIDs) == 0 {
		return 0, 0, nil
	}

	// Pre-query: determine which IDs are actually in firing state and will transition.
	// This prevents writing false timeline entries for events already acknowledged.
	transitioningIDs, qErr := s.repo.GetIDsByStatus(ctx, eventIDs, model.EventStatusFiring)
	if qErr != nil {
		return 0, len(eventIDs), apperr.Wrap(apperr.ErrDatabase, qErr)
	}

	if len(transitioningIDs) == 0 {
		return 0, len(eventIDs), nil
	}

	affected, dbErr := s.repo.BulkAcknowledge(ctx, eventIDs, userID)
	if dbErr != nil {
		return 0, len(eventIDs), apperr.Wrap(apperr.ErrDatabase, dbErr)
	}

	success = int(affected)
	failed = len(eventIDs) - success

	// Write timeline only for events that were in firing state (actually transitioned).
	if success > 0 {
		entries := make([]model.AlertTimeline, 0, len(transitioningIDs))
		for _, id := range transitioningIDs {
			entries = append(entries, model.AlertTimeline{
				EventID:    id,
				Action:     model.TimelineActionAcknowledged,
				OperatorID: &userID,
				Note:       "Alert acknowledged",
			})
		}
		if err2 := s.timelineRepo.BulkCreate(ctx, entries); err2 != nil {
			s.logger.Error("failed to bulk-insert acknowledge timeline", zap.Error(err2))
		}
	}

	return success, failed, nil
}

// BatchClose closes multiple alerts in a single DB round-trip.
func (s *AlertEventService) BatchClose(ctx context.Context, eventIDs []uint, userID uint) (success int, failed int, err error) {
	if len(eventIDs) == 0 {
		return 0, 0, nil
	}

	// Pre-query: determine which IDs are NOT already closed/resolved and will transition.
	// This prevents writing false timeline entries for events already in a terminal state.
	transitioningIDs, qErr := s.repo.GetIDsNotInStatus(ctx, eventIDs,
		[]model.AlertEventStatus{model.EventStatusClosed, model.EventStatusResolved})
	if qErr != nil {
		return 0, len(eventIDs), apperr.Wrap(apperr.ErrDatabase, qErr)
	}

	if len(transitioningIDs) == 0 {
		return 0, len(eventIDs), nil
	}

	affected, dbErr := s.repo.BulkClose(ctx, eventIDs)
	if dbErr != nil {
		return 0, len(eventIDs), apperr.Wrap(apperr.ErrDatabase, dbErr)
	}

	success = int(affected)
	failed = len(eventIDs) - success

	// Write timeline only for events that actually transitioned.
	if success > 0 {
		entries := make([]model.AlertTimeline, 0, len(transitioningIDs))
		for _, id := range transitioningIDs {
			entries = append(entries, model.AlertTimeline{
				EventID:    id,
				Action:     model.TimelineActionClosed,
				OperatorID: &userID,
				Note:       "Batch close",
			})
		}
		if err2 := s.timelineRepo.BulkCreate(ctx, entries); err2 != nil {
			s.logger.Error("failed to bulk-insert close timeline", zap.Error(err2))
		}
	}

	return success, failed, nil
}
