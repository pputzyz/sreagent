package engine

import (
	"context"
	"sync"
	"time"

	"go.uber.org/zap"

	"github.com/sreagent/sreagent/internal/model"
	"github.com/sreagent/sreagent/internal/repository"
)

// SilenceExpiryChecker periodically expires silenced alerts whose
// silenced_until timestamp has passed, transitioning them back to firing.
type SilenceExpiryChecker struct {
	eventRepo    *repository.AlertEventRepository
	timelineRepo *repository.AlertTimelineRepository
	leader       LeaderElection // optional; nil = always run
	logger       *zap.Logger

	interval  time.Duration
	stopCh    chan struct{}
	startOnce sync.Once
	stopOnce  sync.Once
}

// NewSilenceExpiryChecker creates a checker that runs every 60 seconds.
func NewSilenceExpiryChecker(
	eventRepo *repository.AlertEventRepository,
	timelineRepo *repository.AlertTimelineRepository,
	logger *zap.Logger,
) *SilenceExpiryChecker {
	return &SilenceExpiryChecker{
		eventRepo:    eventRepo,
		timelineRepo: timelineRepo,
		logger:       logger,
		interval:     60 * time.Second,
		stopCh:       make(chan struct{}),
	}
}

// SetInterval overrides the default 60-second check interval.
func (s *SilenceExpiryChecker) SetInterval(d time.Duration) { s.interval = d }

// SetLeaderElection sets an optional distributed leader election mechanism.
func (s *SilenceExpiryChecker) SetLeaderElection(le LeaderElection) { s.leader = le }

// Start runs the expiry check loop in a background goroutine.
func (s *SilenceExpiryChecker) Start() {
	s.startOnce.Do(func() {
		go func() {
			ticker := time.NewTicker(s.interval)
			defer ticker.Stop()
			s.logger.Info("silence expiry checker started", zap.Duration("interval", s.interval))
			for {
				select {
				case <-ticker.C:
					func() {
						defer func() {
							if r := recover(); r != nil {
								s.logger.Error("silence expiry checker tick panic recovered", zap.Any("recover", r))
							}
						}()
						if s.leader != nil && !s.leader.IsLeader() {
							return
						}
						ctx, cancel := context.WithTimeout(context.Background(), 55*time.Second)
						defer cancel()
						s.runOnce(ctx)
					}()
				case <-s.stopCh:
					s.logger.Info("silence expiry checker stopped")
					return
				}
			}
		}()
	})
}

// Stop signals the background goroutine to exit.
func (s *SilenceExpiryChecker) Stop() {
	s.stopOnce.Do(func() {
		select {
		case <-s.stopCh:
		default:
			close(s.stopCh)
		}
	})
}

// runOnce expires all silenced events whose silenced_until has passed.
// Uses a bulk UPDATE for efficiency, then records timeline entries for each.
func (s *SilenceExpiryChecker) runOnce(ctx context.Context) {
	now := time.Now()

	// Step 1: Find IDs of events to expire BEFORE updating, so we can write timeline entries.
	var events []model.AlertEvent
	if err := s.eventRepo.DB().WithContext(ctx).
		Where("status = ? AND silenced_until IS NOT NULL AND silenced_until < ?",
			model.EventStatusSilenced, now).
		Select("id").
		Find(&events).Error; err != nil {
		s.logger.Error("silence expiry: failed to query expired events", zap.Error(err))
		return
	}

	if len(events) == 0 {
		return
	}

	ids := make([]uint, len(events))
	for i := range events {
		ids[i] = events[i].ID
	}

	// Step 2: Bulk update to firing. The status guard makes this a CAS:
	// an event resolved/closed by a user between Step 1 and here must NOT
	// be forced back to firing.
	result := s.eventRepo.DB().WithContext(ctx).
		Model(&model.AlertEvent{}).
		Where("id IN ? AND status = ?", ids, model.EventStatusSilenced).
		Updates(map[string]interface{}{
			"status":          model.EventStatusFiring,
			"silenced_until":  nil,
			"silence_reason":  "",
		})
	if result.Error != nil {
		s.logger.Error("silence expiry: failed to update events", zap.Error(result.Error))
		return
	}

	s.logger.Info("silence expiry: expired silenced events",
		zap.Int("count", len(ids)),
		zap.Int64("rows_affected", result.RowsAffected),
	)

	// Step 3: Record timeline entries for each expired event.
	entries := make([]model.AlertTimeline, 0, len(ids))
	for _, id := range ids {
		entries = append(entries, model.AlertTimeline{
			EventID: id,
			Action:  model.TimelineActionUnsilenced,
			Note:    "Silence expired — alert returned to firing",
		})
	}
	if err := s.timelineRepo.BulkCreate(ctx, entries); err != nil {
		s.logger.Error("silence expiry: failed to record timeline entries", zap.Error(err))
	}
}
