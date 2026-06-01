package service

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"go.uber.org/zap"

	"github.com/sreagent/sreagent/internal/pkg/lark"
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
	userRepo    *repository.UserRepository // optional; enables OpenID→User mapping
	client      *http.Client
	tokenCache  *lark.TokenCache // shared token cache (optional)
	logger      *zap.Logger

	// Runtime lifecycle metrics (in-memory, not persisted).
	lastMessageAt       time.Time
	lastError           string
	lastErrorAt         time.Time
	consecutiveErrors   int
	mu                  sync.Mutex
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
	Operator  *LarkCardOperator `json:"operator"`
	Action    *LarkCardAction   `json:"action"`
	Token     string            `json:"token"`
	Type      string            `json:"type"`
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

	// Verify HMAC-SHA256 signature (preferred over plaintext token verification).
	if cfg.EncryptKey != "" && signature != "" {
		if !verifyLarkSignature(timestamp, nonce, cfg.EncryptKey, body, signature) {
			return nil, fmt.Errorf("invalid lark event signature")
		}
	} else if cfg.VerificationToken != "" && req.Header != nil && req.Header.Token != cfg.VerificationToken {
		// Fallback: plaintext token verification when no encrypt key configured.
		return nil, fmt.Errorf("invalid event token")
	} else if cfg.EncryptKey != "" && signature == "" {
		// EncryptKey configured but no signature provided — reject.
		return nil, fmt.Errorf("missing lark event signature")
	}

	// Handle message events
	if req.Header != nil && req.Header.EventType == "im.message.receive_v1" {
		if err := s.handleMessageEvent(ctx, &req, cfg); err != nil {
			s.logger.Error("failed to handle message event", zap.Error(err))
			return nil, err
		}
	}

	return map[string]string{"status": "ok"}, nil
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

// HandleCardAction processes a Lark card action callback (button clicks on alert cards).
// The callback body contains the operator (who clicked) and the action value (what to do).
// Returns an updated card to replace the original, or an error card.
func (s *LarkBotService) HandleCardAction(ctx context.Context, body []byte, signature, timestamp, nonce string) (*lark.CardMessage, error) {
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

	// Anti-replay check
	if timestamp != "" {
		ts, parseErr := strconv.ParseInt(timestamp, 10, 64)
		if parseErr != nil {
			return nil, fmt.Errorf("invalid lark request timestamp")
		}
		if diff := time.Now().Unix() - ts; diff < -300 || diff > 300 {
			return nil, fmt.Errorf("lark request timestamp out of acceptable window")
		}
	}

	if cfg.EncryptKey != "" && signature != "" {
		if !verifyLarkSignature(timestamp, nonce, cfg.EncryptKey, body, signature) {
			return nil, fmt.Errorf("invalid lark card action signature")
		}
	} else if cfg.VerificationToken != "" && req.Token != cfg.VerificationToken {
		return nil, fmt.Errorf("invalid card action token")
	} else if cfg.EncryptKey != "" && signature == "" {
		return nil, fmt.Errorf("missing lark card action signature")
	}

	// Validate action payload
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

	// Resolve operator to system user
	operatorID, err := s.resolveUserID(ctx, req.Operator.OpenID)
	if err != nil {
		s.logger.Warn("card action: operator not mapped",
			zap.String("open_id", req.Operator.OpenID), zap.Error(err))
		return lark.BuildErrorResponseCard("未绑定系统账号，请先在 SREAgent 中绑定 Lark 账号"), nil
	}

	// Fetch event for response card
	event, err := s.eventSvc.GetByID(ctx, eventID)
	if err != nil {
		return lark.BuildErrorResponseCard(fmt.Sprintf("告警 #%d 不存在", eventID)), nil
	}

	// Route to the appropriate action
	switch action {
	case "ack":
		if err := s.eventSvc.Acknowledge(ctx, eventID, operatorID); err != nil {
			s.logger.Warn("card action ack failed",
				zap.Uint("event_id", eventID), zap.Error(err))
			return lark.BuildErrorResponseCard(fmt.Sprintf("认领失败: %v", err)), nil
		}
		s.logger.Info("alert acknowledged via card action",
			zap.Uint("event_id", eventID), zap.Uint("operator", operatorID))
		return lark.BuildAckResponseCard(event.AlertName), nil

	case "silence":
		// Default silence duration: 60 minutes
		if err := s.eventSvc.Silence(ctx, eventID, operatorID, 60, "Lark card action"); err != nil {
			s.logger.Warn("card action silence failed",
				zap.Uint("event_id", eventID), zap.Error(err))
			return lark.BuildErrorResponseCard(fmt.Sprintf("静默失败: %v", err)), nil
		}
		s.logger.Info("alert silenced via card action",
			zap.Uint("event_id", eventID), zap.Uint("operator", operatorID))
		return lark.BuildSilenceResponseCard(event.AlertName), nil

	default:
		return lark.BuildErrorResponseCard(fmt.Sprintf("未知操作: %s", action)), nil
	}
}

// handleMessageEvent processes a received message event.
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

	// If commands are disabled, ignore all messages
	if !cfg.CommandsEnabled {
		s.logger.Debug("lark bot commands disabled, ignoring message")
		return nil
	}

	// Parse command and args
	parts := strings.Fields(text)
	if len(parts) == 0 {
		return s.SendMessage(ctx, chatID, "请发送命令。可用命令: /health, /oncall, /ack, /status\n或直接用自然语言描述您的问题。")
	}

	command := parts[0]
	args := parts[1:]

	// If not a slash command and natural language is enabled, try to map it
	if !strings.HasPrefix(command, "/") && cfg.NaturalLanguageEnabled {
		mappedCmd, mappedArgs := s.mapNaturalLanguage(text)
		if mappedCmd != "" {
			command = mappedCmd
			args = mappedArgs
			s.logger.Debug("natural language mapped",
				zap.String("input", text),
				zap.String("mapped_command", mappedCmd),
			)
		}
	}

	return s.HandleCommand(ctx, command, args, chatID, userID)
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
			bot = lark.NewBotClientWithCache(cfg.AppID, cfg.AppSecret, s.tokenCache)
		} else {
			bot = lark.NewBotClient(cfg.AppID, cfg.AppSecret)
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
	bot := lark.NewBotClient(cfg.AppID, cfg.AppSecret)
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
	Configured       bool   `json:"configured"`
	AppID            string `json:"app_id,omitempty"`
	WebhookSet       bool   `json:"webhook_set"`
	CommandsEnabled  bool   `json:"commands_enabled"`
	NLEnabled        bool   `json:"natural_language_enabled"`
	DebugMode        bool   `json:"debug_mode"`
	LastMessageAt    string `json:"last_message_at,omitempty"`
	LastError        string `json:"last_error,omitempty"`
	LastErrorAt      string `json:"last_error_at,omitempty"`
	ConsecutiveErrors int   `json:"consecutive_errors"`
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
		Configured:       cfg.AppID != "" && cfg.AppSecret != "",
		AppID:            cfg.AppID,
		WebhookSet:       cfg.DefaultWebhook != "",
		CommandsEnabled:  cfg.CommandsEnabled,
		NLEnabled:        cfg.NaturalLanguageEnabled,
		DebugMode:        cfg.DebugMode,
		ConsecutiveErrors: consecErrs,
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
