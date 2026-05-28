package service

import (
	"context"
	"fmt"
	"time"

	"github.com/go-redis/redis/v8"
	"go.uber.org/zap"
)

// NotificationDedupService provides cross-path deduplication for notifications.
// Both NotifyRule and Escalation paths call this to prevent duplicate sends.
type NotificationDedupService struct {
	rdb    *redis.Client
	logger *zap.Logger
	ttl    time.Duration
}

func NewNotificationDedupService(rdb *redis.Client, logger *zap.Logger) *NotificationDedupService {
	return &NotificationDedupService{
		rdb:    rdb,
		logger: logger,
		ttl:    4 * time.Hour,
	}
}

// TrySend returns true if this notification should be sent (first time),
// false if it's a duplicate.
func (d *NotificationDedupService) TrySend(ctx context.Context, key string) bool {
	if d.rdb == nil {
		return true
	}
	redisKey := fmt.Sprintf("notify_dedup:%s", key)
	ok, err := d.rdb.SetNX(ctx, redisKey, "1", d.ttl).Result()
	if err != nil {
		d.logger.Warn("notify_dedup: redis error, allowing send",
			zap.String("key", key), zap.Error(err))
		return true
	}
	return ok
}

// BuildNotifyDedupKey creates a dedup key from notification components.
func BuildNotifyDedupKey(eventID uint, mediaID uint, fingerprint, status string) string {
	return fmt.Sprintf("%d:%d:%s:%s", eventID, mediaID, fingerprint, status)
}
