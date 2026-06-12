package service

import (
	"context"
	"fmt"
	"time"

	"github.com/go-redis/redis/v8"
	"go.uber.org/zap"
)

// EventSource abstracts how Lark events arrive (WebSocket vs HTTP callback).
// The HTTP path is served by the existing gin endpoints (handler/larkbot.go);
// the WebSocket path is WSEventSource. Both feed the SAME LarkBotService
// handlers — only transport-level concerns (signature verification vs
// connection-level auth) differ.
type EventSource interface {
	Start(ctx context.Context) error
	Stop()
}

// EventHandler processes parsed Lark events, decoupled from transport.
// Implemented by LarkBotService.
type EventHandler interface {
	// HandleMessageEvent must return quickly (≤3s ack window); implementations
	// deduplicate by event_id and process asynchronously.
	HandleMessageEvent(ctx context.Context, req *LarkEventRequest) error
	// HandleCardActionEvent routes a pre-verified card action and returns the
	// callback response (toast map for v2 cards / replacement card for v1).
	HandleCardActionEvent(ctx context.Context, req *LarkCardActionRequest) (interface{}, error)
}

// EventDedup prevents duplicate event processing using Redis SETNX.
type EventDedup struct {
	rdb    *redis.Client
	logger *zap.Logger
}

const (
	eventDedupPrefix = "sreagent:lark:event_dedup:"
	eventDedupTTL    = 1 * time.Hour
)

// NewEventDedup creates a new EventDedup backed by Redis.
// Accepts the raw go-redis client to avoid import cycles (service -> pkg/redis -> engine -> service).
func NewEventDedup(rdb *redis.Client, logger *zap.Logger) *EventDedup {
	return &EventDedup{rdb: rdb, logger: logger}
}

// IsDuplicate returns true if the eventID has already been processed.
// Uses Redis SETNX with a 1-hour TTL to prevent duplicate processing.
func (d *EventDedup) IsDuplicate(ctx context.Context, eventID string) (bool, error) {
	key := eventDedupPrefix + eventID
	ok, err := d.rdb.SetNX(ctx, key, "1", eventDedupTTL).Result()
	if err != nil {
		return false, fmt.Errorf("event dedup SETNX failed: %w", err)
	}
	// SetNX returns true when the key was set (i.e. NOT a duplicate).
	return !ok, nil
}
