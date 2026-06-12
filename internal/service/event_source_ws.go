package service

import (
	"context"
	"runtime/debug"
	"sync/atomic"

	larksdk "github.com/larksuite/oapi-sdk-go/v3"
	"github.com/larksuite/oapi-sdk-go/v3/event/dispatcher"
	"github.com/larksuite/oapi-sdk-go/v3/event/dispatcher/callback"
	larkim "github.com/larksuite/oapi-sdk-go/v3/service/im/v1"
	larkws "github.com/larksuite/oapi-sdk-go/v3/ws"
	"go.uber.org/zap"
)

// WSEventSource receives events via Lark's WebSocket long-connection.
// Uses larkws.NewClient from oapi-sdk-go/v3/ws with auto-reconnect.
//
// Event-id deduplication is owned by the handler (LarkBotService), NOT here —
// dedup consumes the SETNX slot, so doing it on both transport and handler
// would make the handler see every event as a duplicate.
type WSEventSource struct {
	appID     string
	appSecret string
	domain    string
	handler   EventHandler
	logger    *zap.Logger
	client    *larkws.Client
	cancel    context.CancelFunc

	// status is the connection state surfaced to GetBotStatus / the UI:
	// starting | connected | reconnecting | disconnected | error | stopped
	status atomic.Value // string
}

// NewWSEventSource creates a new WebSocket-based Lark event source.
// handler must be non-nil (it is the only consumer of received events).
func NewWSEventSource(appID, appSecret, domain string, handler EventHandler, logger *zap.Logger) *WSEventSource {
	w := &WSEventSource{
		appID:     appID,
		appSecret: appSecret,
		domain:    domain,
		handler:   handler,
		logger:    logger,
	}
	w.status.Store("starting")
	return w
}

// Status returns the current connection state.
func (w *WSEventSource) Status() string {
	if v, ok := w.status.Load().(string); ok {
		return v
	}
	return "unknown"
}

// Start launches the WebSocket client in a background goroutine.
func (w *WSEventSource) Start(ctx context.Context) error {
	if w.handler == nil {
		w.status.Store("error")
		w.logger.Error("WebSocket event source not started: no event handler wired")
		return nil // degrade gracefully: HTTP callback path still works
	}

	ctx, w.cancel = context.WithCancel(ctx)

	// Build the event dispatcher. Verification token and encrypt key are left
	// empty — the WS transport authenticates at the connection level and
	// delivers plaintext events.
	eventDispatcher := dispatcher.NewEventDispatcher("", "").
		OnP2MessageReceiveV1(w.onMessageReceive).
		OnP2CardActionTrigger(w.onCardActionTrigger)

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
		larkws.WithOnReady(func() { w.status.Store("connected") }),
		larkws.WithOnReconnecting(func() { w.status.Store("reconnecting") }),
		larkws.WithOnReconnected(func() { w.status.Store("connected") }),
		larkws.WithOnDisconnected(func() { w.status.Store("disconnected") }),
		larkws.WithOnError(func(err error) {
			w.logger.Warn("WebSocket event source error", zap.Error(err))
		}),
	)

	w.logger.Info("WebSocket event source starting",
		zap.String("app_id", w.appID),
		zap.String("domain", w.domain),
	)

	// Start() blocks until the connection is closed or context is cancelled.
	// Run it in a goroutine so Start() itself is non-blocking; a startup
	// failure (e.g. the platform not supporting long connections) is logged
	// and reflected in Status() — it must not crash the process.
	go func() {
		defer func() {
			if r := recover(); r != nil {
				w.status.Store("error")
				w.logger.Error("WebSocket event source panic recovered",
					zap.Any("panic", r),
					zap.String("stack", string(debug.Stack())),
				)
			}
		}()
		if err := w.client.Start(ctx); err != nil {
			w.status.Store("error")
			w.logger.Error("WebSocket event source exited with error", zap.Error(err))
			return
		}
		w.status.Store("stopped")
		w.logger.Info("WebSocket event source goroutine exited")
	}()

	return nil
}

// Stop gracefully shuts down the WebSocket connection.
func (w *WSEventSource) Stop() {
	w.logger.Info("WebSocket event source stopping")
	w.status.Store("stopped")
	if w.cancel != nil {
		w.cancel()
	}
}

// onMessageReceive handles im.message.receive_v1 events from the WebSocket dispatcher.
// The handler (LarkBotService.HandleMessageEvent) dedups and processes
// asynchronously, so this returns well within the 3-second ack window.
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

	req := w.convertMessageEvent(event)
	if err := w.handler.HandleMessageEvent(ctx, req); err != nil {
		w.logger.Error("WebSocket message event handling failed", zap.Error(err))
		return err
	}
	return nil
}

// onCardActionTrigger handles card.action.trigger callbacks delivered over the
// long connection (CardInteractionMode=callback_ws). Availability of this
// channel on Lark International is subject to the T0 PoC.
func (w *WSEventSource) onCardActionTrigger(ctx context.Context, event *callback.CardActionTriggerEvent) (*callback.CardActionTriggerResponse, error) {
	defer func() {
		if r := recover(); r != nil {
			w.logger.Error("card action handler panic recovered",
				zap.Any("panic", r),
				zap.String("stack", string(debug.Stack())),
			)
		}
	}()

	if event == nil || event.Event == nil {
		return &callback.CardActionTriggerResponse{}, nil
	}

	req := convertCardActionEvent(event)
	result, err := w.handler.HandleCardActionEvent(ctx, req)
	if err != nil {
		w.logger.Error("WebSocket card action handling failed", zap.Error(err))
		return &callback.CardActionTriggerResponse{
			Toast: &callback.Toast{Type: "error", Content: "操作处理失败"},
		}, nil
	}

	// v2 path returns a toast map; translate it into the SDK response shape.
	if m, ok := result.(map[string]interface{}); ok {
		if toast, ok := m["toast"].(map[string]interface{}); ok {
			t := &callback.Toast{}
			if v, ok := toast["type"].(string); ok {
				t.Type = v
			}
			if v, ok := toast["content"].(string); ok {
				t.Content = v
			}
			return &callback.CardActionTriggerResponse{Toast: t}, nil
		}
	}
	// Legacy v1 replacement cards cannot be expressed over this channel;
	// acknowledge with a neutral toast (the card itself is updated via the
	// platform's card lifecycle path).
	return &callback.CardActionTriggerResponse{
		Toast: &callback.Toast{Type: "success", Content: "操作已执行"},
	}, nil
}

// convertCardActionEvent maps the SDK callback payload to our internal request type.
func convertCardActionEvent(event *callback.CardActionTriggerEvent) *LarkCardActionRequest {
	req := &LarkCardActionRequest{}
	e := event.Event
	if e.Operator != nil {
		req.Operator = &LarkCardOperator{OpenID: e.Operator.OpenID}
	}
	if e.Action != nil {
		req.Action = &LarkCardAction{Value: e.Action.Value}
		req.FormData = e.Action.FormValue
	}
	if e.Context != nil {
		req.OpenMessageID = e.Context.OpenMessageID
		req.OpenChatID = e.Context.OpenChatID
	}
	return req
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

// Ensure LarkBotService implements EventHandler at compile time.
var _ EventHandler = (*LarkBotService)(nil)
