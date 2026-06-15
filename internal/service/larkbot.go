package service

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"runtime/debug"
	"strconv"
	"strings"
	"sync"
	"time"

	"go.uber.org/zap"

	"github.com/sreagent/sreagent/internal/model"
	"github.com/sreagent/sreagent/internal/pkg/lark"
	"github.com/sreagent/sreagent/internal/pkg/rbac"
	"github.com/sreagent/sreagent/internal/pkg/safehttp"
	"github.com/sreagent/sreagent/internal/repository"
)

// LarkBotService handles Lark bot event callbacks and commands.
// Configuration is loaded from the DB via SystemSettingService on every call,
// so changes made in the Web UI take effect immediately without a restart.
type LarkBotService struct {
	settingSvc  *SystemSettingService
	eventSvc    *AlertEventService
	scheduleSvc *ScheduleService
	aiSvc       *AIService                 // optional; enables AI conversation in groups
	agentSvc    *AgentService              // optional; enables tool-calling NL conversation
	cardSvc     *LarkCardStateService      // optional; enables streaming card replies
	userRepo    *repository.UserRepository // optional; enables OpenID→User mapping
	eventDedup  *EventDedup                // optional; Redis-backed event_id idempotency
	convStore   *ConversationStore         // optional; multi-turn conversation memory
	client      *http.Client
	tokenCache  *lark.TokenCache // shared token cache (optional)
	logger      *zap.Logger

	// eventSourceStatus reports the WS long-connection state ("connected",
	// "reconnecting", …) when the WebSocket event source is active.
	eventSourceStatus func() string

	// Runtime lifecycle metrics (in-memory, not persisted).
	lastMessageAt     time.Time
	lastError         string
	lastErrorAt       time.Time
	consecutiveErrors int
	mu                sync.Mutex
}

// NewLarkBotService creates a new LarkBotService backed by DB-stored configuration.
func NewLarkBotService(settingSvc *SystemSettingService, eventSvc *AlertEventService, scheduleSvc *ScheduleService, userRepo *repository.UserRepository, logger *zap.Logger) *LarkBotService {
	return &LarkBotService{
		settingSvc:  settingSvc,
		eventSvc:    eventSvc,
		scheduleSvc: scheduleSvc,
		userRepo:    userRepo,
		client:      safehttp.NewSafeClient(10 * time.Second),
		logger:      logger,
	}
}

// SetTokenCache injects a shared token cache for Bot API calls.
func (s *LarkBotService) SetTokenCache(cache *lark.TokenCache) {
	s.tokenCache = cache
}

// SetAIService injects the AI service for natural language conversation in Lark groups.
func (s *LarkBotService) SetAIService(svc *AIService) {
	s.aiSvc = svc
}

// SetAgentService injects the AgentService for tool-calling NL conversation.
func (s *LarkBotService) SetAgentService(svc *AgentService) {
	s.agentSvc = svc
}

// SetCardService injects the LarkCardStateService for streaming card replies.
func (s *LarkBotService) SetCardService(svc *LarkCardStateService) {
	s.cardSvc = svc
}

// SetEventDedup injects the Redis-backed event idempotency checker. Used by
// BOTH transports (WebSocket and HTTP callback) so duplicate deliveries —
// e.g. Lark re-pushing after a slow ack — are processed exactly once.
func (s *LarkBotService) SetEventDedup(d *EventDedup) {
	s.eventDedup = d
}

// SetConversationStore injects the multi-turn conversation memory.
func (s *LarkBotService) SetConversationStore(store *ConversationStore) {
	s.convStore = store
}

// SetEventSourceStatusFn wires a status reporter for the WS event source so
// GetBotStatus can surface the long-connection state to the UI.
func (s *LarkBotService) SetEventSourceStatusFn(fn func() string) {
	s.eventSourceStatus = fn
}

// resolveUserID maps a Lark open_id to a DB user ID.
// Returns an error if the user repo is not configured, the open_id is empty,
// or the open_id is not mapped to any system user. No admin fallback.
func (s *LarkBotService) resolveUserID(ctx context.Context, larkOpenID string) (uint, error) {
	if s.userRepo == nil || larkOpenID == "" {
		return 0, fmt.Errorf("lark user mapping not configured")
	}
	user, err := s.userRepo.GetByLarkUserID(ctx, larkOpenID)
	if err != nil {
		return 0, fmt.Errorf("lark user %s not mapped to system user, please bind account first", larkOpenID)
	}
	return user.ID, nil
}

// authorizeCardAction verifies the operator's platform role grants the given
// permission. Identity (resolveUserID) only proves WHO clicked; this proves
// they are ALLOWED to perform the mutation.
func (s *LarkBotService) authorizeCardAction(ctx context.Context, operatorID uint, perm string) error {
	if s.userRepo == nil {
		return fmt.Errorf("user repo not configured")
	}
	user, err := s.userRepo.GetByID(ctx, operatorID)
	if err != nil {
		return fmt.Errorf("load operator %d: %w", operatorID, err)
	}
	if !rbac.HasPerm(string(user.Role), perm) {
		return fmt.Errorf("role %q lacks permission %q", user.Role, perm)
	}
	return nil
}

// loadConfig fetches the current Lark config from the DB.
func (s *LarkBotService) loadConfig(ctx context.Context) (LarkConfig, error) {
	return s.settingSvc.GetLarkConfig(ctx)
}

// GetConfig returns the current Lark bot configuration with secrets masked.
func (s *LarkBotService) GetConfig(ctx context.Context) (LarkConfig, error) {
	cfg, err := s.loadConfig(ctx)
	if err != nil {
		return LarkConfig{}, err
	}
	// Mask secrets for display
	if cfg.AppSecret != "" {
		cfg.AppSecret = "********"
	}
	if cfg.EncryptKey != "" {
		cfg.EncryptKey = "********"
	}
	if cfg.VerificationToken != "" {
		cfg.VerificationToken = "********"
	}
	return cfg, nil
}

// UpdateConfig persists the Lark bot configuration to the DB.
func (s *LarkBotService) UpdateConfig(ctx context.Context, cfg LarkConfig) error {
	return s.settingSvc.SaveLarkConfig(ctx, cfg)
}

// LarkEventRequest represents the incoming Lark event callback payload.
type LarkEventRequest struct {
	// URL verification fields
	Challenge string `json:"challenge"`
	Token     string `json:"token"`
	Type      string `json:"type"`

	// Event subscription fields
	Schema string           `json:"schema"`
	Header *LarkEventHeader `json:"header"`
	Event  *LarkEventBody   `json:"event"`
}

// LarkEventHeader is the header part of a Lark event.
type LarkEventHeader struct {
	EventID    string `json:"event_id"`
	Token      string `json:"token"`
	CreateTime string `json:"create_time"`
	EventType  string `json:"event_type"`
	TenantKey  string `json:"tenant_key"`
	AppID      string `json:"app_id"`
}

// LarkEventBody is the event body for im.message.receive_v1.
type LarkEventBody struct {
	Sender  *LarkSender  `json:"sender"`
	Message *LarkMessage `json:"message"`
}

// LarkSender represents the message sender.
type LarkSender struct {
	SenderID   *LarkSenderID `json:"sender_id"`
	SenderType string        `json:"sender_type"`
	TenantKey  string        `json:"tenant_key"`
}

// LarkSenderID contains various ID formats for the sender.
type LarkSenderID struct {
	UnionID string `json:"union_id"`
	UserID  string `json:"user_id"`
	OpenID  string `json:"open_id"`
}

// LarkMessage represents the message content.
type LarkMessage struct {
	MessageID   string        `json:"message_id"`
	RootID      string        `json:"root_id"`
	ParentID    string        `json:"parent_id"`
	CreateTime  string        `json:"create_time"`
	ChatID      string        `json:"chat_id"`
	ChatType    string        `json:"chat_type"`
	MessageType string        `json:"message_type"`
	Content     string        `json:"content"`
	Mentions    []LarkMention `json:"mentions"`
}

// LarkMention represents an @mention in the message.
type LarkMention struct {
	Key       string        `json:"key"`
	ID        *LarkSenderID `json:"id"`
	Name      string        `json:"name"`
	TenantKey string        `json:"tenant_key"`
}

// LarkCardActionRequest represents the incoming Lark card action callback payload.
type LarkCardActionRequest struct {
	Operator *LarkCardOperator      `json:"operator"`
	Action   *LarkCardAction        `json:"action"`
	Token    string                 `json:"token"`
	Type     string                 `json:"type"`
	FormData map[string]interface{} `json:"form_data,omitempty"`
	// Context of the click, filled by the WS transport (HTTP payloads carry it
	// in a different envelope; unused there).
	OpenMessageID string `json:"open_message_id,omitempty"`
	OpenChatID    string `json:"open_chat_id,omitempty"`
}

// LarkCardOperator identifies the user who clicked the button.
type LarkCardOperator struct {
	OpenID string `json:"open_id"`
}

// LarkCardAction holds the button action data.
type LarkCardAction struct {
	Value map[string]interface{} `json:"value"`
}

// HandleEvent processes a Lark event callback.
// signature, timestamp, nonce are the X-Lark-Signature headers for HMAC-SHA256 verification.
// Returns (response body, error).
func (s *LarkBotService) HandleEvent(ctx context.Context, body []byte, signature, timestamp, nonce string) (interface{}, error) {
	// Parse the JSON body first — this is a cheap operation and must not be
	// blocked behind a DB call.  Loading config is only needed for token
	// verification which happens after parsing.
	var req LarkEventRequest
	if err := json.Unmarshal(body, &req); err != nil {
		return nil, fmt.Errorf("failed to parse event: %w", err)
	}

	cfg, err := s.loadConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to load lark config: %w", err)
	}

	// Handle URL verification challenge
	if req.Type == "url_verification" {
		if cfg.VerificationToken != "" && req.Token != cfg.VerificationToken {
			return nil, fmt.Errorf("invalid verification token")
		}
		return map[string]string{"challenge": req.Challenge}, nil
	}

	// Fail-closed: require at least one verification method configured.
	if cfg.EncryptKey == "" && cfg.VerificationToken == "" {
		s.logger.Warn("lark event rejected: no verification configured (EncryptKey and VerificationToken both empty)")
		return nil, fmt.Errorf("lark verification not configured")
	}

	// Anti-replay: reject requests with timestamps outside a ±5 minute window.
	// Mandatory when verification is configured — empty timestamp is rejected.
	if timestamp == "" {
		return nil, fmt.Errorf("missing lark request timestamp")
	}
	ts, err := strconv.ParseInt(timestamp, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid lark request timestamp")
	}
	if diff := time.Now().Unix() - ts; diff < -300 || diff > 300 {
		return nil, fmt.Errorf("lark request timestamp out of acceptable window")
	}

	// Reject if no authentication is configured at all.
	if cfg.EncryptKey == "" && cfg.VerificationToken == "" {
		return nil, fmt.Errorf("lark bot authentication not configured: set encrypt_key or verification_token")
	}

	// Verify HMAC-SHA256 signature (preferred over plaintext token verification).
	if cfg.EncryptKey != "" && signature != "" {
		if !verifyLarkSignature(timestamp, nonce, cfg.EncryptKey, body, signature) {
			return nil, fmt.Errorf("invalid lark event signature")
		}
	} else if cfg.EncryptKey == "" && cfg.VerificationToken != "" {
		// Fallback: plaintext token verification when no encrypt key configured.
		if req.Header == nil || req.Header.Token == "" {
			return nil, fmt.Errorf("missing event verification token")
		}
		if subtle.ConstantTimeCompare([]byte(req.Header.Token), []byte(cfg.VerificationToken)) != 1 {
			return nil, fmt.Errorf("invalid event token")
		}
	} else if cfg.EncryptKey != "" && signature == "" {
		// EncryptKey configured but no signature provided — reject.
		return nil, fmt.Errorf("missing lark event signature")
	}

	// Handle message events through the transport-agnostic entry: dedup +
	// async processing, so the HTTP callback is acked within Lark's 3-second
	// window even when the NL path runs a long LLM tool-calling loop.
	if req.Header != nil && req.Header.EventType == "im.message.receive_v1" {
		if err := s.HandleMessageEvent(ctx, &req); err != nil {
			s.logger.Error("failed to handle message event", zap.Error(err))
			return nil, err
		}
	}

	return map[string]string{"status": "ok"}, nil
}

// HandleMessageEvent is the transport-agnostic entry for im.message.receive_v1
// events (called by both the WebSocket source and the HTTP callback path,
// AFTER transport-level verification). It deduplicates by event_id and then
// processes the message asynchronously: Lark requires the event to be acked
// within 3 seconds, while the NL path may run an LLM loop for much longer.
// Returning nil here acks the event; failures inside the async worker are
// logged and answered in-chat, not retried by Lark.
func (s *LarkBotService) HandleMessageEvent(ctx context.Context, req *LarkEventRequest) error {
	if s.eventDedup != nil && req.Header != nil && req.Header.EventID != "" {
		dup, err := s.eventDedup.IsDuplicate(ctx, req.Header.EventID)
		if err != nil {
			s.logger.Warn("event dedup check failed, processing anyway", zap.Error(err))
		} else if dup {
			s.logger.Debug("duplicate message event skipped", zap.String("event_id", req.Header.EventID))
			return nil
		}
	}

	go func() {
		defer func() {
			if r := recover(); r != nil {
				s.logger.Error("async message event handler panic recovered",
					zap.Any("recover", r), zap.String("stack", string(debug.Stack())))
			}
		}()
		// Detach from the request context: the HTTP response (the ack) returns
		// immediately, but processing may legitimately take minutes.
		bgCtx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancel()

		cfg, err := s.loadConfig(bgCtx)
		if err != nil {
			s.logger.Error("async message event: failed to load lark config", zap.Error(err))
			return
		}
		if err := s.handleMessageEvent(bgCtx, req, cfg); err != nil {
			s.logger.Error("async message event handling failed", zap.Error(err))
		}
	}()
	return nil
}

// verifyLarkSignature verifies the HMAC-SHA256 signature from Lark event callbacks.
// The signature is computed as: base64(sha256(timestamp + nonce + encryptKey + body)).
func verifyLarkSignature(timestamp, nonce, encryptKey string, body []byte, expectedSignature string) bool {
	if timestamp == "" || nonce == "" || encryptKey == "" {
		return false
	}
	hash := sha256.New()
	hash.Write([]byte(timestamp))
	hash.Write([]byte(nonce))
	hash.Write([]byte(encryptKey))
	hash.Write(body)
	computed := base64.StdEncoding.EncodeToString(hash.Sum(nil))
	return hmac.Equal([]byte(computed), []byte(expectedSignature))
}

// HandleCardAction processes a Lark card action callback (button clicks on
// alert cards) arriving over HTTP. It performs transport verification and
// delegates the action routing to processCardAction.
// The response is either a legacy replacement card (v1 cards) or a toast
// object (v2 CardKit cards, whose content refreshes via entity updates).
func (s *LarkBotService) HandleCardAction(ctx context.Context, body []byte, signature, timestamp, nonce string) (interface{}, error) {
	var req LarkCardActionRequest
	if err := json.Unmarshal(body, &req); err != nil {
		return nil, fmt.Errorf("failed to parse card action: %w", err)
	}

	// Verify signature
	cfg, err := s.loadConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to load lark config: %w", err)
	}

	if cfg.EncryptKey == "" && cfg.VerificationToken == "" {
		s.logger.Warn("lark card action rejected: no verification configured")
		return nil, fmt.Errorf("lark verification not configured")
	}

	// Anti-replay: mandatory, mirroring HandleEvent (an empty timestamp must
	// not silently skip the window check).
	if timestamp == "" {
		return nil, fmt.Errorf("missing lark request timestamp")
	}
	ts, parseErr := strconv.ParseInt(timestamp, 10, 64)
	if parseErr != nil {
		return nil, fmt.Errorf("invalid lark request timestamp")
	}
	if diff := time.Now().Unix() - ts; diff < -300 || diff > 300 {
		return nil, fmt.Errorf("lark request timestamp out of acceptable window")
	}

	if cfg.EncryptKey != "" && signature != "" {
		if !verifyLarkSignature(timestamp, nonce, cfg.EncryptKey, body, signature) {
			return nil, fmt.Errorf("invalid lark card action signature")
		}
	} else if cfg.EncryptKey != "" && signature == "" {
		return nil, fmt.Errorf("missing lark card action signature")
	} else if cfg.VerificationToken != "" {
		if req.Token == "" || subtle.ConstantTimeCompare([]byte(req.Token), []byte(cfg.VerificationToken)) != 1 {
			return nil, fmt.Errorf("invalid card action token")
		}
	}

	return s.processCardAction(ctx, &req)
}

// HandleCardActionEvent routes a pre-verified card action (e.g. delivered over
// the WebSocket long connection, which authenticates at the connection level).
func (s *LarkBotService) HandleCardActionEvent(ctx context.Context, req *LarkCardActionRequest) (interface{}, error) {
	return s.processCardAction(ctx, req)
}

// cardActionPerms maps card actions to the RBAC permission they require.
// Identity mapping alone is NOT authorization: a viewer-role user with a bound
// Lark account must not be able to mutate alerts from a group chat.
var cardActionPerms = map[string]string{
	"ack":          "events.ack",
	"silence":      "events.ack",
	"silence_form": "events.ack",
	"resolve":      "events.ack",
	"assign_form":  "events.assign",
}

// processCardAction validates, authorizes and executes a card button action.
// Transport verification (HTTP signature / WS connection auth) must already
// have happened.
func (s *LarkBotService) processCardAction(ctx context.Context, req *LarkCardActionRequest) (interface{}, error) {
	// Validate action payload
	if req.Action == nil || req.Action.Value == nil {
		return s.cardActionError(ctx, 0, "无效的操作请求"), nil
	}
	if req.Operator == nil || req.Operator.OpenID == "" {
		return s.cardActionError(ctx, 0, "无法识别操作者"), nil
	}

	action, _ := req.Action.Value["action"].(string)
	eventIDFloat, _ := req.Action.Value["event_id"].(float64)
	eventID := uint(eventIDFloat)

	// "retry" wraps another action — unwrap once (no recursion).
	if action == "retry" {
		original, _ := req.Action.Value["original_action"].(string)
		if original == "" || original == "retry" {
			return s.cardActionError(ctx, eventID, "缺少原始操作类型"), nil
		}
		action = original
	}

	if action == "" || eventID == 0 {
		return s.cardActionError(ctx, eventID, "操作参数不完整"), nil
	}

	// Resolve operator to system user (identity).
	operatorID, err := s.resolveUserID(ctx, req.Operator.OpenID)
	if err != nil {
		s.logger.Warn("card action: operator not mapped",
			zap.String("open_id", req.Operator.OpenID), zap.Error(err))
		return s.cardActionError(ctx, eventID, "未绑定系统账号，请先在 SREAgent 中绑定 Lark 账号"), nil
	}

	// Authorize (RBAC): the operator's platform role must grant the action.
	if perm, known := cardActionPerms[action]; known {
		if err := s.authorizeCardAction(ctx, operatorID, perm); err != nil {
			s.logger.Warn("card action denied by RBAC",
				zap.Uint("operator", operatorID), zap.String("action", action))
			return s.cardActionError(ctx, eventID, "权限不足：当前账号无权执行该操作"), nil
		}
	}

	// Fetch event for response card
	event, err := s.eventSvc.GetByID(ctx, eventID)
	if err != nil {
		return s.cardActionError(ctx, eventID, fmt.Sprintf("告警 #%d 不存在", eventID)), nil
	}

	// Route to the appropriate action
	switch action {
	case "ack":
		if err := s.eventSvc.Acknowledge(ctx, eventID, operatorID); err != nil {
			s.logger.Warn("card action ack failed", zap.Uint("event_id", eventID), zap.Error(err))
			return s.cardActionError(ctx, eventID, fmt.Sprintf("认领失败: %v", err)), nil
		}
		s.logger.Info("alert acknowledged via card action",
			zap.Uint("event_id", eventID), zap.Uint("operator", operatorID))
		return s.cardActionSuccess(ctx, event, "已认领", lark.BuildAckResponseCard), nil

	case "silence":
		// Duration comes from the button value (静默 1h / 24h buttons); the
		// legacy form path is handled by silence_form below.
		durationMinutes := 60
		if d, ok := req.Action.Value["duration"].(float64); ok && d > 0 {
			durationMinutes = int(d)
		}
		if err := s.eventSvc.Silence(ctx, eventID, operatorID, durationMinutes, "Lark card action"); err != nil {
			s.logger.Warn("card action silence failed", zap.Uint("event_id", eventID), zap.Error(err))
			return s.cardActionError(ctx, eventID, fmt.Sprintf("静默失败: %v", err)), nil
		}
		s.logger.Info("alert silenced via card action",
			zap.Uint("event_id", eventID), zap.Int("duration", durationMinutes), zap.Uint("operator", operatorID))
		return s.cardActionSuccess(ctx, event, fmt.Sprintf("已静默 %d 分钟", durationMinutes), lark.BuildSilenceResponseCard), nil

	case "silence_form":
		durationMinutes := 60
		reason := "Lark card action"
		if req.FormData != nil {
			if durStr, ok := req.FormData["duration"].(string); ok {
				if dur, err := strconv.Atoi(durStr); err == nil && dur > 0 {
					durationMinutes = dur
				}
			}
			if r, ok := req.FormData["reason"].(string); ok && r != "" {
				reason = r
			}
		}
		if err := s.eventSvc.Silence(ctx, eventID, operatorID, durationMinutes, reason); err != nil {
			s.logger.Warn("card action silence_form failed", zap.Uint("event_id", eventID), zap.Error(err))
			return s.cardActionError(ctx, eventID, fmt.Sprintf("静默失败: %v", err)), nil
		}
		s.logger.Info("alert silenced via silence form",
			zap.Uint("event_id", eventID), zap.Int("duration", durationMinutes), zap.Uint("operator", operatorID))
		return s.cardActionSuccess(ctx, event, fmt.Sprintf("已静默 %d 分钟", durationMinutes), lark.BuildSilenceResponseCard), nil

	case "assign_form":
		if req.FormData == nil {
			return s.cardActionError(ctx, eventID, "表单数据缺失"), nil
		}
		assigneeStr, _ := req.FormData["assignee"].(string)
		if assigneeStr == "" {
			return s.cardActionError(ctx, eventID, "请选择指派人"), nil
		}
		assignTo, err := strconv.ParseUint(assigneeStr, 10, 64)
		if err != nil || assignTo == 0 {
			return s.cardActionError(ctx, eventID, "无效的指派人"), nil
		}
		note, _ := req.FormData["note"].(string)
		if note == "" {
			note = fmt.Sprintf("Assigned via Lark by operator %d", operatorID)
		}
		if err := s.eventSvc.Assign(ctx, eventID, uint(assignTo), operatorID, note); err != nil {
			s.logger.Warn("card action assign_form failed", zap.Uint("event_id", eventID), zap.Error(err))
			return s.cardActionError(ctx, eventID, fmt.Sprintf("指派失败: %v", err)), nil
		}
		s.logger.Info("alert assigned via assign form",
			zap.Uint("event_id", eventID), zap.Uint64("assign_to", assignTo), zap.Uint("operator", operatorID))
		return s.cardActionSuccess(ctx, event, "已指派", lark.BuildAckResponseCard), nil

	case "resolve":
		if err := s.eventSvc.Resolve(ctx, eventID, operatorID, "Resolved via Lark card action"); err != nil {
			s.logger.Warn("card action resolve failed", zap.Uint("event_id", eventID), zap.Error(err))
			return s.cardActionError(ctx, eventID, fmt.Sprintf("确认解决失败: %v", err)), nil
		}
		s.logger.Info("alert resolved via card action",
			zap.Uint("event_id", eventID), zap.Uint("operator", operatorID))
		return s.cardActionSuccess(ctx, event, "已标记解决", lark.BuildAckResponseCard), nil

	default:
		return s.cardActionError(ctx, eventID, fmt.Sprintf("未知操作: %s", action)), nil
	}
}

// isV2CardEvent reports whether the event's card is CardKit-managed (v2).
func (s *LarkBotService) isV2CardEvent(ctx context.Context, eventID uint) bool {
	return eventID != 0 && s.cardSvc != nil && s.cardSvc.HasEntityForEvent(ctx, eventID)
}

// cardActionSuccess builds the callback response after a successful action.
// v2 (CardKit) cards: toast only — the card content itself is refreshed through
// the entity-update path triggered by the status change, keeping ONE update
// channel (returning a replacement card here would fight with CardKit).
// v1 cards: legacy replacement card via the supplied builder.
func (s *LarkBotService) cardActionSuccess(ctx context.Context, event *model.AlertEvent, toastMsg string, v1Builder func(string) *lark.CardMessage) interface{} {
	if s.isV2CardEvent(ctx, event.ID) {
		return map[string]interface{}{
			"toast": map[string]interface{}{"type": "success", "content": toastMsg},
		}
	}
	return v1Builder(event.AlertName)
}

// cardActionError builds the failure response (toast for v2, error card for v1).
func (s *LarkBotService) cardActionError(ctx context.Context, eventID uint, msg string) interface{} {
	if s.isV2CardEvent(ctx, eventID) {
		return map[string]interface{}{
			"toast": map[string]interface{}{"type": "error", "content": msg},
		}
	}
	return lark.BuildErrorResponseCard(msg)
}

// handleMessageEvent processes a received message event.
// Priority: slash commands → Agent NL → legacy NL mapping → fallback message.
func (s *LarkBotService) handleMessageEvent(ctx context.Context, req *LarkEventRequest, cfg LarkConfig) error {
	if req.Event == nil || req.Event.Message == nil {
		return nil
	}

	msg := req.Event.Message
	chatID := msg.ChatID
	userID := ""
	if req.Event.Sender != nil && req.Event.Sender.SenderID != nil {
		userID = req.Event.Sender.SenderID.OpenID
	}

	// Parse message content (Lark sends content as JSON string)
	var content struct {
		Text string `json:"text"`
	}
	if err := json.Unmarshal([]byte(msg.Content), &content); err != nil {
		s.logger.Warn("failed to parse message content", zap.Error(err))
		return nil
	}

	// Strip @bot mentions from the text
	text := content.Text
	for _, mention := range msg.Mentions {
		text = strings.ReplaceAll(text, mention.Key, "")
	}
	text = strings.TrimSpace(text)

	if text == "" {
		return s.SendMessage(ctx, chatID, "请发送命令。可用命令: /health, /oncall, /ack, /status\n或直接用自然语言描述您的问题。")
	}

	// Fast path 1: explicit slash commands — gated by CommandsEnabled so the
	// admin switch actually disables them.
	if strings.HasPrefix(text, "/") {
		if !cfg.CommandsEnabled {
			return s.SendMessage(ctx, chatID, "命令功能未启用，请在系统设置中开启。")
		}
		parts := strings.Fields(text)
		return s.HandleCommand(ctx, parts[0], parts[1:], chatID, userID)
	}

	// Fast path 2: unambiguous NL patterns ("ack 123") map straight to
	// commands — zero LLM latency/cost for the common cases.
	if cfg.CommandsEnabled {
		if mappedCmd, mappedArgs := s.mapNaturalLanguage(text); mappedCmd != "" {
			s.logger.Debug("natural language mapped",
				zap.String("input", text),
				zap.String("mapped_command", mappedCmd),
			)
			return s.HandleCommand(ctx, mappedCmd, mappedArgs, chatID, userID)
		}
	}

	// Everything else: Agent tool-calling conversation.
	if cfg.NaturalLanguageEnabled && s.aiSvc != nil {
		return s.handleAgentConversation(ctx, text, chatID, msg.ChatID, userID)
	}

	return s.SendMessage(ctx, chatID, "我不太理解您的意思。可用命令: /health, /oncall, /ack, /status")
}

// HandleCommand routes and executes bot commands.
func (s *LarkBotService) HandleCommand(ctx context.Context, command string, args []string, chatID, userID string) error {
	switch command {
	case "/health":
		return s.cmdHealth(ctx, args, chatID)
	case "/oncall":
		return s.cmdOnCall(ctx, chatID)
	case "/ack":
		return s.cmdAck(ctx, args, chatID, userID)
	case "/status":
		return s.cmdStatus(ctx, chatID)
	default:
		return s.SendMessage(ctx, chatID, fmt.Sprintf("Unknown command: %s\nAvailable commands: /health <cluster>, /oncall, /ack <alert_id>, /status", command))
	}
}

// cmdHealth handles the /health <cluster> command.
func (s *LarkBotService) cmdHealth(ctx context.Context, args []string, chatID string) error {
	cluster := ""
	if len(args) > 0 {
		cluster = args[0]
	}

	events, _, err := s.eventSvc.List(ctx, "firing", "", 1, 1000)
	if err != nil {
		return s.SendMessage(ctx, chatID, fmt.Sprintf("Failed to fetch cluster health: %v", err))
	}

	// Filter by cluster label when specified
	var clusterAlerts int
	criticalCount := 0
	warningCount := 0
	for _, e := range events {
		if cluster != "" {
			labels := e.Labels
			if labels != nil {
				if c, ok := labels["cluster"]; !ok || c != cluster {
					continue
				}
			} else {
				continue
			}
		}
		clusterAlerts++
		switch strings.ToLower(string(e.Severity)) {
		case "critical":
			criticalCount++
		case "warning":
			warningCount++
		}
	}

	clusterLabel := cluster
	if clusterLabel == "" {
		clusterLabel = "all clusters"
	}

	var status string
	if criticalCount > 0 {
		status = "CRITICAL"
	} else if warningCount > 0 {
		status = "DEGRADED"
	} else if clusterAlerts > 0 {
		status = "WARNING"
	} else {
		status = "HEALTHY"
	}

	msg := fmt.Sprintf("Cluster Health: %s\n- Status: %s\n- Firing Alerts: %d\n- Critical: %d\n- Warning: %d",
		clusterLabel, status, clusterAlerts, criticalCount, warningCount)
	return s.SendMessage(ctx, chatID, msg)
}

// cmdOnCall handles the /oncall command.
func (s *LarkBotService) cmdOnCall(ctx context.Context, chatID string) error {
	if s.scheduleSvc == nil {
		return s.SendMessage(ctx, chatID, "On-call schedules are not configured.")
	}

	user, err := s.scheduleSvc.GetCurrentOnCallForAlert(ctx, map[string]string{})
	if err != nil || user == nil {
		return s.SendMessage(ctx, chatID, "No on-call user found. Please configure schedules in SREAgent.")
	}

	msg := fmt.Sprintf("Current On-Call:\n- Name: %s\n- Email: %s", user.DisplayName, user.Email)
	if user.Phone != "" {
		msg += fmt.Sprintf("\n- Phone: %s", user.Phone)
	}
	return s.SendMessage(ctx, chatID, msg)
}

// cmdAck handles the /ack <alert_id> command.
// Resolves the Lark sender's open_id to a DB user; rejects if unmapped.
func (s *LarkBotService) cmdAck(ctx context.Context, args []string, chatID, userID string) error {
	if len(args) == 0 {
		return s.SendMessage(ctx, chatID, "Usage: /ack <alert_id>")
	}

	operatorID, err := s.resolveUserID(ctx, userID)
	if err != nil {
		return s.SendMessage(ctx, chatID, "未绑定系统账号，请先在 SREAgent 中绑定 Lark 账号")
	}

	idStr := args[0]
	alertID, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		return s.SendMessage(ctx, chatID, fmt.Sprintf("Invalid alert ID: %s. Please provide a numeric alert ID.", idStr))
	}

	if err := s.eventSvc.Acknowledge(ctx, uint(alertID), operatorID); err != nil {
		return s.SendMessage(ctx, chatID, fmt.Sprintf("Failed to acknowledge alert #%d: %v", alertID, err))
	}

	return s.SendMessage(ctx, chatID, fmt.Sprintf("Alert #%d has been acknowledged.", alertID))
}

// cmdStatus handles the /status command.
func (s *LarkBotService) cmdStatus(ctx context.Context, chatID string) error {
	_, firingTotal, err := s.eventSvc.List(ctx, "firing", "", 1, 1)
	if err != nil {
		return s.SendMessage(ctx, chatID, fmt.Sprintf("Failed to fetch alert status: %v", err))
	}

	_, criticalTotal, err := s.eventSvc.List(ctx, "firing", "critical", 1, 1)
	if err != nil {
		return s.SendMessage(ctx, chatID, fmt.Sprintf("Failed to fetch critical alerts: %v", err))
	}
	_, warningTotal, err := s.eventSvc.List(ctx, "firing", "warning", 1, 1)
	if err != nil {
		return s.SendMessage(ctx, chatID, fmt.Sprintf("Failed to fetch warning alerts: %v", err))
	}
	_, ackedTotal, err := s.eventSvc.List(ctx, "acknowledged", "", 1, 1)
	if err != nil {
		return s.SendMessage(ctx, chatID, fmt.Sprintf("Failed to fetch acknowledged alerts: %v", err))
	}

	msg := fmt.Sprintf("SREAgent Platform Status:\n- Active Alerts: %d\n- Critical: %d\n- Warning: %d\n- Acknowledged: %d",
		firingTotal, criticalTotal, warningTotal, ackedTotal)
	return s.SendMessage(ctx, chatID, msg)
}

// SendCardJSON sends a raw interactive card (e.g. a Card 2.0 JSON string) to a
// chat via the Bot API. Used by the report scheduler for chart-rich report cards.
func (s *LarkBotService) SendCardJSON(ctx context.Context, chatID, cardJSON string) error {
	cfg, err := s.loadConfig(ctx)
	if err != nil {
		return fmt.Errorf("failed to load lark config: %w", err)
	}
	if cfg.AppID == "" || cfg.AppSecret == "" {
		return fmt.Errorf("lark bot credentials not configured")
	}
	if chatID == "" {
		return fmt.Errorf("chat_id is empty")
	}

	var bot *lark.BotClient
	if s.tokenCache != nil {
		bot = lark.NewBotClientWithCache(cfg.AppID, cfg.AppSecret, s.tokenCache, lark.BaseURLForDomain(cfg.Domain))
	} else {
		bot = lark.NewBotClient(cfg.AppID, cfg.AppSecret, lark.BaseURLForDomain(cfg.Domain))
	}
	if _, err := bot.SendInteractiveJSON(ctx, "chat_id", chatID, cardJSON); err != nil {
		s.recordMessageError(err)
		return fmt.Errorf("lark card send failed: %w", err)
	}
	s.recordMessageSuccess()
	return nil
}

// SendMessage sends a text message to a Lark chat.
//
// Routing preference:
//  1. If AppID + AppSecret are configured, use the Bot API to reply into the
//     originating chat (chatID), so @bot commands land in the correct room.
//  2. Otherwise fall back to the legacy incoming webhook (DefaultWebhook) — this
//     ignores chatID and is only useful for one-way broadcast setups.
func (s *LarkBotService) SendMessage(ctx context.Context, chatID, content string) error {
	cfg, err := s.loadConfig(ctx)
	if err != nil {
		s.logger.Warn("lark bot: failed to load config", zap.Error(err))
		return fmt.Errorf("failed to load lark config: %w", err)
	}

	// Preferred path: Bot API with chat_id (correct routing for command replies).
	if cfg.AppID != "" && cfg.AppSecret != "" && chatID != "" {
		var bot *lark.BotClient
		if s.tokenCache != nil {
			bot = lark.NewBotClientWithCache(cfg.AppID, cfg.AppSecret, s.tokenCache, lark.BaseURLForDomain(cfg.Domain))
		} else {
			bot = lark.NewBotClient(cfg.AppID, cfg.AppSecret, lark.BaseURLForDomain(cfg.Domain))
		}
		if _, err := bot.SendText(ctx, "chat_id", chatID, content); err != nil {
			s.recordMessageError(err)
			s.logger.Warn("lark bot: Bot API send failed",
				zap.String("chat_id", chatID), zap.Error(err))
			return fmt.Errorf("lark bot API send failed: %w", err)
		}
		s.recordMessageSuccess()
		s.logger.Debug("lark bot text reply sent via Bot API", zap.String("chat_id", chatID))
		return nil
	}

	// Fallback: incoming webhook (chatID is ignored by webhook targets).
	if cfg.DefaultWebhook == "" {
		s.logger.Warn("lark bot: neither Bot API credentials nor default webhook configured")
		return fmt.Errorf("lark bot not configured (need AppID/AppSecret or DefaultWebhook)")
	}

	payload := map[string]interface{}{
		"msg_type": "text",
		"content": map[string]string{
			"text": content,
		},
	}

	bodyBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, cfg.DefaultWebhook, bytes.NewReader(bodyBytes))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send lark message: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	respBody, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20)) // 1 MB max
	if err != nil {
		return fmt.Errorf("failed to read lark response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("lark API returned status %d: %s", resp.StatusCode, string(respBody))
	}

	s.logger.Debug("lark bot message sent", zap.String("chat_id", chatID))
	return nil
}

// TestBotAPI tests connectivity to the Lark Bot API by fetching a tenant access token.
func (s *LarkBotService) TestBotAPI(ctx context.Context) error {
	cfg, err := s.loadConfig(ctx)
	if err != nil {
		return fmt.Errorf("failed to load lark config: %w", err)
	}
	if cfg.AppID == "" || cfg.AppSecret == "" {
		return fmt.Errorf("AppID and AppSecret must be configured")
	}
	bot := lark.NewBotClient(cfg.AppID, cfg.AppSecret, lark.BaseURLForDomain(cfg.Domain))
	// SendText to an invalid chat_id — if credentials are wrong, we get a
	// LarkError during token acquisition. If credentials are valid, the API
	// returns a routing error (invalid chat_id) which is expected.
	_, err = bot.SendText(ctx, "chat_id", "__test__", "ping")
	if err != nil {
		// Auth errors (bad AppID/AppSecret) are wrapped as *LarkError
		var larkErr *lark.LarkError
		if errors.As(err, &larkErr) {
			return fmt.Errorf("lark bot API authentication failed (code %d): %s", larkErr.Code, larkErr.Message)
		}
		// Network or other unexpected errors
		return fmt.Errorf("lark bot API test failed: %w", err)
	}
	return nil
}

// BotStatus holds diagnostic info about the bot connection.
type BotStatus struct {
	Configured        bool   `json:"configured"`
	AppID             string `json:"app_id,omitempty"`
	WebhookSet        bool   `json:"webhook_set"`
	CommandsEnabled   bool   `json:"commands_enabled"`
	NLEnabled         bool   `json:"natural_language_enabled"`
	DebugMode         bool   `json:"debug_mode"`
	ConnectionMode    string `json:"connection_mode,omitempty"`
	EventSourceStatus string `json:"event_source_status,omitempty"` // WS long connection: connected/reconnecting/...
	LastMessageAt     string `json:"last_message_at,omitempty"`
	LastError         string `json:"last_error,omitempty"`
	LastErrorAt       string `json:"last_error_at,omitempty"`
	ConsecutiveErrors int    `json:"consecutive_errors"`
}

// GetBotStatus returns the current bot connection status and diagnostics.
func (s *LarkBotService) GetBotStatus(ctx context.Context) (*BotStatus, error) {
	cfg, err := s.loadConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to load lark config: %w", err)
	}

	s.mu.Lock()
	lastMsg := s.lastMessageAt
	lastErr := s.lastError
	lastErrAt := s.lastErrorAt
	consecErrs := s.consecutiveErrors
	s.mu.Unlock()

	status := &BotStatus{
		Configured:        cfg.AppID != "" && cfg.AppSecret != "",
		AppID:             cfg.AppID,
		WebhookSet:        cfg.DefaultWebhook != "",
		CommandsEnabled:   cfg.CommandsEnabled,
		NLEnabled:         cfg.NaturalLanguageEnabled,
		DebugMode:         cfg.DebugMode,
		ConnectionMode:    cfg.ConnectionMode,
		ConsecutiveErrors: consecErrs,
	}
	if s.eventSourceStatus != nil {
		status.EventSourceStatus = s.eventSourceStatus()
	}
	if !lastMsg.IsZero() {
		status.LastMessageAt = lastMsg.Format(time.RFC3339)
	}
	if lastErr != "" {
		status.LastError = lastErr
		status.LastErrorAt = lastErrAt.Format(time.RFC3339)
	}
	return status, nil
}

// recordMessageSuccess updates lifecycle metrics on successful send.
func (s *LarkBotService) recordMessageSuccess() {
	s.mu.Lock()
	s.lastMessageAt = time.Now()
	s.consecutiveErrors = 0
	s.mu.Unlock()
}

// recordMessageError updates lifecycle metrics on failed send.
func (s *LarkBotService) recordMessageError(err error) {
	s.mu.Lock()
	s.lastError = err.Error()
	s.lastErrorAt = time.Now()
	s.consecutiveErrors++
	s.mu.Unlock()
}

// mapNaturalLanguage maps natural language input to bot commands.
func (s *LarkBotService) mapNaturalLanguage(text string) (string, []string) {
	lower := strings.ToLower(strings.TrimSpace(text))

	// Status queries
	if strings.Contains(lower, "状态") || strings.Contains(lower, "status") ||
		strings.Contains(lower, "情况") || strings.Contains(lower, "how many") {
		return "/status", nil
	}
	// Health queries
	if strings.Contains(lower, "健康") || strings.Contains(lower, "health") ||
		strings.Contains(lower, "集群") || strings.Contains(lower, "cluster") {
		// Try to extract cluster name
		parts := strings.Fields(lower)
		for _, p := range parts {
			if p != "健康" && p != "health" && p != "集群" && p != "cluster" && p != "的" && p != "查看" {
				return "/health", []string{p}
			}
		}
		return "/health", nil
	}
	// On-call queries
	if strings.Contains(lower, "值班") || strings.Contains(lower, "oncall") ||
		strings.Contains(lower, "on-call") || strings.Contains(lower, "谁在") {
		return "/oncall", nil
	}
	// Acknowledge
	if strings.Contains(lower, "确认") || strings.Contains(lower, "ack") ||
		strings.Contains(lower, "acknowledge") {
		// Try to extract alert ID
		parts := strings.Fields(text)
		for _, p := range parts {
			if n, err := strconv.Atoi(p); err == nil && n > 0 {
				return "/ack", []string{p}
			}
		}
		return "/ack", nil
	}

	return "", nil
}

// defaultLarkBotTools is the tool whitelist used when the admin hasn't
// configured one. Read-only queries + acknowledge — deliberately NOT the full
// registry: an empty allowList means "everything" downstream (including MCP
// write tools), which must never be the default for a chat-driven agent.
var defaultLarkBotTools = []string{
	"query_alert_events",
	"alert_statistics",
	"get_oncall",
	"acknowledge_alert",
	"run_inspection",
	"query_instant",
	"list_alert_rules",
	"search_similar_alerts",
	"search_knowledge",
}

// botAllowedTools resolves the effective tool whitelist for the Lark bot.
func botAllowedTools(cfg LarkConfig) []string {
	raw := strings.TrimSpace(cfg.BotAllowedTools)
	if raw == "" {
		return defaultLarkBotTools
	}
	parts := strings.Split(raw, ",")
	tools := make([]string, 0, len(parts))
	for _, p := range parts {
		if p = strings.TrimSpace(p); p != "" {
			tools = append(tools, p)
		}
	}
	if len(tools) == 0 {
		return defaultLarkBotTools
	}
	return tools
}

// handleAgentConversation routes a natural language message through the
// AgentService tool-calling pipeline. senderOpenID is the Lark open_id of the
// MESSAGE SENDER (not the chat) and is mapped to a platform user for tool
// authorization; unmapped senders run with operatorID 0 (read-only: write
// tools reject a zero operator).
func (s *LarkBotService) handleAgentConversation(ctx context.Context, question, chatID, larkChatID, senderOpenID string) error {
	if s.aiSvc == nil {
		return s.SendMessage(ctx, chatID, "AI 助手未配置，请在系统设置中启用 AI。")
	}

	systemPrompt := `你是 SREAgent 智能运维助手，通过飞书群与用户交互。
你可以查询告警、统计数据、查看值班、触发巡检、认领告警等。
回答简洁专业，使用中文。涉及数据的问题必须使用提供的工具获取实时数据，禁止编造数字。
如果问题不明确，请给出通用的排查建议。`

	// Multi-turn memory: prepend recent turns so follow-up questions
	// ("那 web-01 呢？") keep their context.
	if s.convStore != nil {
		if conv, err := s.convStore.Get(ctx, larkChatID, senderOpenID); err == nil && conv != nil && len(conv.Turns) > 0 {
			var b strings.Builder
			b.WriteString("\n\n以下是本会话此前的对话记录（供理解上下文）：\n")
			for _, turn := range conv.Turns {
				fmt.Fprintf(&b, "[%s] %s\n", turn.Role, truncateForCard(turn.Content, 500))
			}
			systemPrompt += b.String()
		}
	}

	// Streaming path: agentSvc + cardSvc → streaming card with tool step updates.
	var answer string
	var convErr error
	if s.agentSvc != nil && s.cardSvc != nil {
		answer, convErr = s.runStreamingAgentConversation(ctx, question, chatID, larkChatID, senderOpenID, systemPrompt)
	} else {
		// Fallback: simple Chat without streaming/tools.
		answer, convErr = s.aiSvc.Chat(ctx, systemPrompt, nil, question)
		if convErr == nil {
			convErr = s.SendMessage(ctx, chatID, fmt.Sprintf("🤖 **AI 回复:**\n%s", answer))
		}
	}
	if convErr != nil {
		s.logger.Warn("agent conversation failed", zap.Error(convErr))
		return s.SendMessage(ctx, chatID, fmt.Sprintf("AI 助手暂时无法回复: %v", convErr))
	}

	// Persist the turn for follow-up questions (best-effort).
	if s.convStore != nil && answer != "" {
		_ = s.convStore.Append(ctx, larkChatID, senderOpenID, ConversationTurn{Role: "user", Content: question})
		_ = s.convStore.Append(ctx, larkChatID, senderOpenID, ConversationTurn{Role: "assistant", Content: truncateForCard(answer, 1000)})
	}
	return nil
}

// runStreamingAgentConversation runs the agent with accumulating card updates.
// Returns the final answer text.
func (s *LarkBotService) runStreamingAgentConversation(ctx context.Context, question, chatID, larkChatID, senderOpenID, systemPrompt string) (string, error) {
	cfg, err := s.loadConfig(ctx)
	if err != nil {
		return "", fmt.Errorf("load lark config: %w", err)
	}

	// Resolve the SENDER (not the chat) for tool authorization. Unmapped
	// senders get operatorID 0 — write tools reject that.
	var operatorID uint
	if resolved, err := s.resolveUserID(ctx, senderOpenID); err == nil {
		operatorID = resolved
	}

	entity, err := s.cardSvc.CreateStreamingCard(ctx, larkChatID, question)
	if err != nil {
		s.logger.Warn("streaming card creation failed, falling back to text", zap.Error(err))
		answer, chatErr := s.aiSvc.Chat(ctx, systemPrompt, nil, question)
		if chatErr != nil {
			return "", chatErr
		}
		return answer, s.SendMessage(ctx, chatID, fmt.Sprintf("🤖 **AI 回复:**\n%s", answer))
	}

	// Accumulate steps so the card shows the FULL progress history, and let
	// the per-card rate limiter (inside CardKit UpdateCardEntity) pace updates.
	var steps []StreamStep
	result, err := s.agentSvc.RunWithStreaming(ctx, operatorID, systemPrompt, question, botAllowedTools(cfg), 10,
		func(step int, toolName, content string) {
			steps = append(steps, StreamStep{Step: step, ToolName: toolName, Content: content})
			if updateErr := s.cardSvc.UpdateStreamingProgress(ctx, entity.ID, question, steps); updateErr != nil {
				s.logger.Warn("streaming card update failed", zap.Int("step", step), zap.Error(updateErr))
			}
		},
	)
	if err != nil {
		s.logger.Warn("streaming agent failed", zap.Error(err))
		_ = s.cardSvc.FinalizeStreamingCard(ctx, entity.ID, question, fmt.Sprintf("❌ 执行失败: %v", err), steps)
		return "", nil // error already shown on the card; don't double-post
	}

	if err := s.cardSvc.FinalizeStreamingCard(ctx, entity.ID, question, result.FinalAnswer, steps); err != nil {
		s.logger.Warn("streaming card finalize failed", zap.Error(err))
	}
	s.recordMessageSuccess()
	return result.FinalAnswer, nil
}
