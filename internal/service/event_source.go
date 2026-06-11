package service

import (
	"context"
	"fmt"
	"time"

	"github.com/go-redis/redis/v8"
	"go.uber.org/zap"

	"github.com/sreagent/sreagent/internal/pkg/lark"
)

// EventSource abstracts how Lark events arrive (WebSocket vs HTTP callback).
type EventSource interface {
	Start(ctx context.Context) error
	Stop()
}

// EventHandler processes parsed Lark events, decoupled from transport.
type EventHandler interface {
	HandleMessageEvent(ctx context.Context, req *LarkEventRequest) error
	HandleCardActionEvent(ctx context.Context, req *LarkCardActionRequest) (*lark.CardMessage, error)
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

// HTTPEventSource wraps the existing HTTP callback path via LarkBotService.
// It implements the EventHandler interface by delegating to the bot service.
type HTTPEventSource struct {
	botSvc *LarkBotService
	dedup  *EventDedup
	logger *zap.Logger
}

// NewHTTPEventSource creates an HTTPEventSource backed by the existing LarkBotService.
func NewHTTPEventSource(botSvc *LarkBotService, dedup *EventDedup, logger *zap.Logger) *HTTPEventSource {
	return &HTTPEventSource{
		botSvc: botSvc,
		dedup:  dedup,
		logger: logger,
	}
}

// Start is a no-op for HTTPEventSource — the HTTP server is managed externally.
func (h *HTTPEventSource) Start(_ context.Context) error {
	h.logger.Info("HTTP event source started (HTTP server managed externally)")
	return nil
}

// Stop is a no-op for HTTPEventSource.
func (h *HTTPEventSource) Stop() {
	h.logger.Info("HTTP event source stopped")
}

// HandleMessageEvent processes a message event via the bot service.
// Performs deduplication before forwarding.
func (h *HTTPEventSource) HandleMessageEvent(ctx context.Context, req *LarkEventRequest) error {
	if req.Header != nil && req.Header.EventID != "" {
		dup, err := h.dedup.IsDuplicate(ctx, req.Header.EventID)
		if err != nil {
			h.logger.Warn("event dedup check failed, processing anyway", zap.Error(err))
		} else if dup {
			h.logger.Debug("duplicate message event skipped", zap.String("event_id", req.Header.EventID))
			return nil
		}
	}

	// Load config to pass to the handler (same as LarkBotService.handleMessageEvent expects).
	cfg, err := h.botSvc.loadConfig(ctx)
	if err != nil {
		return fmt.Errorf("failed to load lark config: %w", err)
	}

	return h.botSvc.handleMessageEvent(ctx, req, cfg)
}

// HandleCardActionEvent processes a card action event via the bot service.
func (h *HTTPEventSource) HandleCardActionEvent(ctx context.Context, req *LarkCardActionRequest) (*lark.CardMessage, error) {
	// Card actions don't have a header with event_id, so no dedup here.
	// Signature verification is expected to be done by the HTTP handler before calling this.
	if req.Action == nil || req.Action.Value == nil {
		return lark.BuildErrorResponseCard("无效的操作请求"), nil
	}
	if req.Operator == nil || req.Operator.OpenID == "" {
		return lark.BuildErrorResponseCard("无法识别操作者"), nil
	}

	action, _ := req.Action.Value["action"].(string)
	eventIDFloat, _ := req.Action.Value["event_id"].(float64)
	eventID := uint(eventIDFloat)

	if action == "" || eventID == 0 {
		return lark.BuildErrorResponseCard("操作参数不完整"), nil
	}

	operatorID, err := h.botSvc.resolveUserID(ctx, req.Operator.OpenID)
	if err != nil {
		h.logger.Warn("card action: operator not mapped",
			zap.String("open_id", req.Operator.OpenID), zap.Error(err))
		return lark.BuildErrorResponseCard("未绑定系统账号，请先在 SREAgent 中绑定 Lark 账号"), nil
	}

	event, err := h.botSvc.eventSvc.GetByID(ctx, eventID)
	if err != nil {
		return lark.BuildErrorResponseCard(fmt.Sprintf("告警 #%d 不存在", eventID)), nil
	}

	switch action {
	case "ack":
		if err := h.botSvc.eventSvc.Acknowledge(ctx, eventID, operatorID); err != nil {
			h.logger.Warn("card action ack failed",
				zap.Uint("event_id", eventID), zap.Error(err))
			return lark.BuildErrorResponseCard(fmt.Sprintf("认领失败: %v", err)), nil
		}
		h.logger.Info("alert acknowledged via card action",
			zap.Uint("event_id", eventID), zap.Uint("operator", operatorID))
		return lark.BuildAckResponseCard(event.AlertName), nil

	case "silence":
		if err := h.botSvc.eventSvc.Silence(ctx, eventID, operatorID, 60, "Lark card action"); err != nil {
			h.logger.Warn("card action silence failed",
				zap.Uint("event_id", eventID), zap.Error(err))
			return lark.BuildErrorResponseCard(fmt.Sprintf("静默失败: %v", err)), nil
		}
		h.logger.Info("alert silenced via card action",
			zap.Uint("event_id", eventID), zap.Uint("operator", operatorID))
		return lark.BuildSilenceResponseCard(event.AlertName), nil

	default:
		return lark.BuildErrorResponseCard(fmt.Sprintf("未知操作: %s", action)), nil
	}
}
