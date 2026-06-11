package service

import (
	"context"
	"runtime/debug"

	larksdk "github.com/larksuite/oapi-sdk-go/v3"
	"github.com/larksuite/oapi-sdk-go/v3/event/dispatcher"
	larkim "github.com/larksuite/oapi-sdk-go/v3/service/im/v1"
	larkws "github.com/larksuite/oapi-sdk-go/v3/ws"
	"go.uber.org/zap"
)

// WSEventSource receives events via Lark's WebSocket long-connection.
// Uses larkws.NewClient from oapi-sdk-go/v3/ws with auto-reconnect.
type WSEventSource struct {
	appID     string
	appSecret string
	domain    string
	handler   EventHandler
	dedup     *EventDedup
	logger    *zap.Logger
	client    *larkws.Client
	cancel    context.CancelFunc
}

// NewWSEventSource creates a new WebSocket-based Lark event source.
func NewWSEventSource(appID, appSecret, domain string, handler EventHandler, dedup *EventDedup, logger *zap.Logger) *WSEventSource {
	return &WSEventSource{
		appID:     appID,
		appSecret: appSecret,
		domain:    domain,
		handler:   handler,
		dedup:     dedup,
		logger:    logger,
	}
}

// Start launches the WebSocket client in a background goroutine.
// It blocks until the context is cancelled or Stop() is called.
func (w *WSEventSource) Start(ctx context.Context) error {
	ctx, w.cancel = context.WithCancel(ctx)

	// Build the event dispatcher with message handler.
	// Verification token and encrypt key are left empty — the WS transport
	// handles authentication at the connection level.
	eventDispatcher := dispatcher.NewEventDispatcher("", "").
		OnP2MessageReceiveV1(w.onMessageReceive)

	// Determine the Lark domain endpoint.
	// "feishu" → China (open.feishu.cn), anything else → International (open.larksuite.com).
	domain := larksdk.LarkBaseUrl
	if w.domain == "feishu" {
		domain = larksdk.FeishuBaseUrl
	}

	w.client = larkws.NewClient(w.appID, w.appSecret,
		larkws.WithEventHandler(eventDispatcher),
		larkws.WithDomain(domain),
		larkws.WithAutoReconnect(true),
	)

	w.logger.Info("WebSocket event source starting",
		zap.String("app_id", w.appID),
		zap.String("domain", w.domain),
	)

	// Start() blocks until the connection is closed or context is cancelled.
	// Run it in a goroutine so Start() itself is non-blocking.
	go func() {
		defer func() {
			if r := recover(); r != nil {
				w.logger.Error("WebSocket event source panic recovered",
					zap.Any("panic", r),
					zap.String("stack", string(debug.Stack())),
				)
			}
		}()
		if err := w.client.Start(ctx); err != nil {
			w.logger.Error("WebSocket event source exited with error", zap.Error(err))
		}
		w.logger.Info("WebSocket event source goroutine exited")
	}()

	return nil
}

// Stop gracefully shuts down the WebSocket connection.
func (w *WSEventSource) Stop() {
	w.logger.Info("WebSocket event source stopping")
	if w.cancel != nil {
		w.cancel()
	}
}

// onMessageReceive handles im.message.receive_v1 events from the WebSocket dispatcher.
func (w *WSEventSource) onMessageReceive(ctx context.Context, event *larkim.P2MessageReceiveV1) error {
	defer func() {
		if r := recover(); r != nil {
			w.logger.Error("message event handler panic recovered",
				zap.Any("panic", r),
				zap.String("stack", string(debug.Stack())),
			)
		}
	}()

	if event == nil || event.Event == nil {
		return nil
	}

	// Dedup using the event header if available.
	// Use EventV2Base.Header explicitly to avoid ambiguity with EventReq.Header.
	eventID := ""
	if event.EventV2Base != nil && event.EventV2Base.Header != nil {
		eventID = event.EventV2Base.Header.EventID
	}
	if eventID != "" {
		dup, err := w.dedup.IsDuplicate(ctx, eventID)
		if err != nil {
			w.logger.Warn("event dedup check failed, processing anyway", zap.Error(err))
		} else if dup {
			w.logger.Debug("duplicate WebSocket event skipped", zap.String("event_id", eventID))
			return nil
		}
	}

	// Convert the SDK event to our internal LarkEventRequest format.
	req := w.convertMessageEvent(event)

	if err := w.handler.HandleMessageEvent(ctx, req); err != nil {
		w.logger.Error("WebSocket message event handling failed",
			zap.String("event_id", eventID),
			zap.Error(err),
		)
		return err
	}

	return nil
}

// derefStr safely dereferences a *string, returning "" if nil.
func derefStr(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// convertMessageEvent converts the SDK P2MessageReceiveV1 event to our internal LarkEventRequest.
func (w *WSEventSource) convertMessageEvent(event *larkim.P2MessageReceiveV1) *LarkEventRequest {
	req := &LarkEventRequest{
		Schema: "2.0",
	}

	// Map header from the embedded EventV2Base.
	if event.EventV2Base != nil && event.EventV2Base.Header != nil {
		h := event.EventV2Base.Header
		req.Header = &LarkEventHeader{
			EventID:    h.EventID,
			Token:      h.Token,
			CreateTime: h.CreateTime,
			EventType:  h.EventType,
			TenantKey:  h.TenantKey,
			AppID:      h.AppID,
		}
	}

	// Map event body.
	if event.Event != nil {
		body := &LarkEventBody{}

		if event.Event.Sender != nil {
			sender := &LarkSender{
				SenderType: derefStr(event.Event.Sender.SenderType),
			}
			if event.Event.Sender.SenderId != nil {
				sender.SenderID = &LarkSenderID{
					UnionID: derefStr(event.Event.Sender.SenderId.UnionId),
					UserID:  derefStr(event.Event.Sender.SenderId.UserId),
					OpenID:  derefStr(event.Event.Sender.SenderId.OpenId),
				}
			}
			body.Sender = sender
		}

		if event.Event.Message != nil {
			msg := &LarkMessage{
				MessageID:   derefStr(event.Event.Message.MessageId),
				RootID:      derefStr(event.Event.Message.RootId),
				ParentID:    derefStr(event.Event.Message.ParentId),
				CreateTime:  derefStr(event.Event.Message.CreateTime),
				ChatID:      derefStr(event.Event.Message.ChatId),
				ChatType:    derefStr(event.Event.Message.ChatType),
				MessageType: derefStr(event.Event.Message.MessageType),
				Content:     derefStr(event.Event.Message.Content),
			}

			// Convert mentions from SDK format.
			if event.Event.Message.Mentions != nil {
				mentions := make([]LarkMention, 0, len(event.Event.Message.Mentions))
				for _, m := range event.Event.Message.Mentions {
					mention := LarkMention{
						Key:  derefStr(m.Key),
						Name: derefStr(m.Name),
					}
					if m.Id != nil {
						mention.ID = &LarkSenderID{
							UnionID: derefStr(m.Id.UnionId),
							UserID:  derefStr(m.Id.UserId),
							OpenID:  derefStr(m.Id.OpenId),
						}
					}
					mentions = append(mentions, mention)
				}
				msg.Mentions = mentions
			}

			body.Message = msg
		}

		req.Event = body
	}

	return req
}

// Ensure WSEventSource implements EventSource at compile time.
var _ EventSource = (*WSEventSource)(nil)
