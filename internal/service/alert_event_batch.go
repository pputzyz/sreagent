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

	affected, dbErr := s.repo.BulkAcknowledge(ctx, eventIDs, userID)
	if dbErr != nil {
		return 0, len(eventIDs), apperr.Wrap(apperr.ErrDatabase, dbErr)
	}

	success = int(affected)
	failed = len(eventIDs) - success

	// Only insert timeline entries for events that were actually updated.
	// Re-query to find which IDs transitioned to acknowledged.
	if success > 0 {
		updatedEvents, qErr := s.repo.GetByIDs(ctx, eventIDs)
		if qErr == nil {
			entries := make([]model.AlertTimeline, 0, success)
			for _, ev := range updatedEvents {
				if ev.Status == model.EventStatusAcknowledged {
					entries = append(entries, model.AlertTimeline{
						EventID:    ev.ID,
						Action:     model.TimelineActionAcknowledged,
						OperatorID: &userID,
						Note:       "Alert acknowledged",
					})
				}
			}
			if err2 := s.timelineRepo.BulkCreate(ctx, entries); err2 != nil {
				s.logger.Error("failed to bulk-insert acknowledge timeline", zap.Error(err2))
			}
		}
	}

	return success, failed, nil
}

// BatchClose closes multiple alerts in a single DB round-trip.
func (s *AlertEventService) BatchClose(ctx context.Context, eventIDs []uint, userID uint) (success int, failed int, err error) {
	if len(eventIDs) == 0 {
		return 0, 0, nil
	}

	affected, dbErr := s.repo.BulkClose(ctx, eventIDs)
	if dbErr != nil {
		return 0, len(eventIDs), apperr.Wrap(apperr.ErrDatabase, dbErr)
	}

	success = int(affected)
	failed = len(eventIDs) - success

	// Only insert timeline entries for events that were actually updated.
	if success > 0 {
		updatedEvents, qErr := s.repo.GetByIDs(ctx, eventIDs)
		if qErr == nil {
			entries := make([]model.AlertTimeline, 0, success)
			for _, ev := range updatedEvents {
				if ev.Status == model.EventStatusClosed {
					entries = append(entries, model.AlertTimeline{
						EventID:    ev.ID,
						Action:     model.TimelineActionClosed,
						OperatorID: &userID,
						Note:       "Batch close",
					})
				}
			}
			if err2 := s.timelineRepo.BulkCreate(ctx, entries); err2 != nil {
				s.logger.Error("failed to bulk-insert close timeline", zap.Error(err2))
			}
		}
	}

	return success, failed, nil
}
