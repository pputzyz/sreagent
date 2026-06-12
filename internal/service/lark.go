package service

import (
	"context"
	"fmt"
	"sync"
	"time"

	"go.uber.org/zap"

	"github.com/sreagent/sreagent/internal/model"
	"github.com/sreagent/sreagent/internal/pkg/lark"
)

// LarkService wraps the Lark client for sending alert notifications.
type LarkService struct {
	client *lark.Client
	logger *zap.Logger
	// platformBaseURL is the base URL of the SREAgent web UI for deep-linking.
	platformBaseURL string
	// jwtSecret is used to sign alert action tokens.
	jwtSecret string
	// settingSvc provides Lark bot credentials for Bot API calls.
	settingSvc *SystemSettingService
	// Cached bot client to avoid fetching a new token on every call.
	botClientMu     sync.Mutex
	botClient       *lark.BotClient
	botClientAppID  string
	botClientSecret string
	// Shared token cache — can be injected so multiple services share one cache.
	tokenCache *lark.TokenCache
	// cardSvc manages CardKit card entities (v2 path). Nil when CardKit not configured.
	cardSvc *LarkCardStateService
}

// SetCardService injects the CardKit card state service for the v2 send path.
func (s *LarkService) SetCardService(cardSvc *LarkCardStateService) {
	s.cardSvc = cardSvc
}

// NewLarkService creates a new LarkService.
func NewLarkService(logger *zap.Logger, platformBaseURL, jwtSecret string, settingSvc *SystemSettingService) *LarkService {
	return &LarkService{
		client:          lark.NewClient(logger),
		logger:          logger,
		platformBaseURL: platformBaseURL,
		jwtSecret:       jwtSecret,
		settingSvc:      settingSvc,
	}
}

// SetTokenCache injects a shared token cache so that multiple services (LarkService,
// NotifyMediaService, LarkBotService) share a single cache, avoiding redundant
// tenant_access_token fetches.
func (s *LarkService) SetTokenCache(cache *lark.TokenCache) {
	s.botClientMu.Lock()
	defer s.botClientMu.Unlock()
	s.tokenCache = cache
	// Invalidate the cached bot client so it is recreated with the shared cache.
	s.botClient = nil
}

// getBotClient returns a cached BotClient, creating a new one if credentials changed.
func (s *LarkService) getBotClient(appID, appSecret, baseURL string) *lark.BotClient {
	s.botClientMu.Lock()
	defer s.botClientMu.Unlock()
	if s.botClient != nil && s.botClientAppID == appID && s.botClientSecret == appSecret {
		return s.botClient
	}
	if s.tokenCache != nil {
		s.botClient = lark.NewBotClientWithCache(appID, appSecret, s.tokenCache, baseURL)
	} else {
		s.botClient = lark.NewBotClient(appID, appSecret, baseURL)
	}
	s.botClientAppID = appID
	s.botClientSecret = appSecret
	return s.botClient
}

// SendAlertNotification prepares and sends an alert notification via Lark webhook.
func (s *LarkService) SendAlertNotification(ctx context.Context, event *model.AlertEvent, webhookURL string) error {
	// Build the platform link for this alert event
	platformURL := ""
	if s.platformBaseURL != "" {
		platformURL = fmt.Sprintf("%s/alert-events/%d", s.platformBaseURL, event.ID)
	}

	card := lark.BuildAlertCard(
		event.AlertName,
		string(event.Severity),
		string(event.Status),
		event.Labels,
		event.Annotations,
		event.FiredAt,
		platformURL,
	)

	resp, err := s.client.SendWebhook(ctx, webhookURL, card)
	if err != nil {
		s.logger.Error("failed to send lark alert notification",
			zap.Uint("event_id", event.ID),
			zap.String("alert_name", event.AlertName),
			zap.Error(err),
		)
		return fmt.Errorf("lark webhook failed: %w", err)
	}

	s.logger.Info("lark alert notification sent",
		zap.Uint("event_id", event.ID),
		zap.String("alert_name", event.AlertName),
		zap.Int("resp_code", resp.Code),
	)
	return nil
}

// SendEnrichedAlertNotification sends an alert notification with AI analysis via Lark webhook.
func (s *LarkService) SendEnrichedAlertNotification(ctx context.Context, event *model.AlertEvent, analysis *AlertAnalysis, webhookURL string) error {
	// Build the platform link for this alert event
	platformURL := ""
	if s.platformBaseURL != "" {
		platformURL = fmt.Sprintf("%s/alert-events/%d", s.platformBaseURL, event.ID)
	}

	// Generate an action token for no-auth alert action page
	actionBaseURL := ""
	if s.platformBaseURL != "" && s.jwtSecret != "" {
		token, err := GenerateAlertActionToken(event.ID, s.jwtSecret)
		if err != nil {
			s.logger.Warn("failed to generate alert action token",
				zap.Uint("event_id", event.ID),
				zap.Error(err),
			)
		} else {
			actionBaseURL = fmt.Sprintf("%s/alert-action/%s", s.platformBaseURL, token)
		}
	}

	// Convert service.AlertAnalysis to lark.AIAnalysisResult (nil-safe)
	var aiResult *lark.AIAnalysisResult
	if analysis != nil {
		aiResult = &lark.AIAnalysisResult{
			Summary:          analysis.Summary,
			ProbableCauses:   analysis.ProbableCauses,
			Impact:           analysis.Impact,
			RecommendedSteps: analysis.RecommendedSteps,
		}
	}

	card := lark.BuildEnrichedAlertCard(
		event.AlertName,
		string(event.Severity),
		string(event.Status),
		event.Labels,
		event.Annotations,
		event.FiredAt,
		aiResult,
		platformURL,
		actionBaseURL,
		0, // no callback for webhook delivery
	)

	resp, err := s.client.SendWebhook(ctx, webhookURL, card)
	if err != nil {
		s.logger.Error("failed to send enriched lark alert notification",
			zap.Uint("event_id", event.ID),
			zap.String("alert_name", event.AlertName),
			zap.Error(err),
		)
		return fmt.Errorf("lark webhook failed: %w", err)
	}

	s.logger.Info("enriched lark alert notification sent",
		zap.Uint("event_id", event.ID),
		zap.String("alert_name", event.AlertName),
		zap.Int("resp_code", resp.Code),
		zap.Bool("has_ai_analysis", analysis != nil),
	)
	return nil
}

// SendTestNotification sends a test card to the given webhook URL.
func (s *LarkService) SendTestNotification(ctx context.Context, webhookURL string) error {
	card := lark.BuildTestCard()

	_, err := s.client.SendWebhook(ctx, webhookURL, card)
	if err != nil {
		s.logger.Error("failed to send lark test notification", zap.Error(err))
		return fmt.Errorf("lark test webhook failed: %w", err)
	}

	s.logger.Info("lark test notification sent successfully")
	return nil
}

// SendEnrichedAlertNotificationViaBot sends an alert card via Lark Bot API to a group chat.
// Returns the message_id that can be used to update the card on status changes.
// chatID is the group's chat_id (e.g. "oc_xxxxx").
func (s *LarkService) SendEnrichedAlertNotificationViaBot(ctx context.Context, event *model.AlertEvent, analysis *AlertAnalysis, chatID string) (string, error) {
	if s.settingSvc == nil {
		return "", fmt.Errorf("settingSvc not configured for Bot API")
	}

	larkCfg, err := s.settingSvc.GetLarkConfig(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to load lark config: %w", err)
	}
	if larkCfg.AppID == "" || larkCfg.AppSecret == "" {
		return "", fmt.Errorf("lark bot credentials not configured")
	}

	// v2 path: CardKit entity-based card.
	if larkCfg.CardSchemaVersion == "v2" && s.cardSvc != nil {
		entity, err := s.cardSvc.EnsureCardForEvent(ctx, event, chatID)
		if err != nil {
			s.logger.Error("CardKit send failed, falling back to legacy",
				zap.Uint("event_id", event.ID), zap.Error(err))
			// fall through to legacy path
		} else {
			s.logger.Info("alert card sent via CardKit",
				zap.Uint("event_id", event.ID),
				zap.String("card_id", entity.CardID),
			)
			return entity.CardID, nil
		}
	}

	// v1 path: legacy PATCH-based card.
	card := s.buildEnrichedCard(event, analysis, true)
	botClient := s.getBotClient(larkCfg.AppID, larkCfg.AppSecret, lark.BaseURLForDomain(larkCfg.Domain))

	msgID, err := botClient.SendMessage(ctx, chatID, card)
	if err != nil {
		s.logger.Error("failed to send alert card via Bot API",
			zap.Uint("event_id", event.ID), zap.Error(err))
		return "", fmt.Errorf("lark bot send failed: %w", err)
	}

	s.logger.Info("alert card sent via Bot API",
		zap.Uint("event_id", event.ID),
		zap.String("message_id", msgID),
	)
	return msgID, nil
}

// SendTestNotificationViaBot sends a test card to a Lark chat via Bot API (chat_id).
func (s *LarkService) SendTestNotificationViaBot(ctx context.Context, chatID string) error {
	if s.settingSvc == nil {
		return fmt.Errorf("settingSvc not configured for Bot API")
	}
	larkCfg, err := s.settingSvc.GetLarkConfig(ctx)
	if err != nil {
		return fmt.Errorf("failed to load lark config: %w", err)
	}
	if larkCfg.AppID == "" || larkCfg.AppSecret == "" {
		return fmt.Errorf("lark bot credentials not configured")
	}
	card := lark.BuildTestCard()
	bot := s.getBotClient(larkCfg.AppID, larkCfg.AppSecret, lark.BaseURLForDomain(larkCfg.Domain))
	if _, err := bot.SendMessage(ctx, chatID, card); err != nil {
		s.logger.Error("failed to send lark test card via Bot API",
			zap.String("chat_id", chatID), zap.Error(err))
		return fmt.Errorf("lark bot test send failed: %w", err)
	}
	s.logger.Info("lark test card sent via Bot API", zap.String("chat_id", chatID))
	return nil
}

// SendAlertCardToUser sends an enriched alert card directly to a Lark user (DM) via Bot API.
// receiveIDType is typically "user_id" (from UserNotifyConfig) or "open_id".
// Returns the message_id (not persisted to the event — DMs are per-recipient).
func (s *LarkService) SendAlertCardToUser(ctx context.Context, event *model.AlertEvent, analysis *AlertAnalysis, receiveIDType, receiveID string) (string, error) {
	if s.settingSvc == nil {
		return "", fmt.Errorf("settingSvc not configured for Bot API")
	}
	if receiveID == "" {
		return "", fmt.Errorf("receiveID is empty")
	}

	larkCfg, err := s.settingSvc.GetLarkConfig(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to load lark config: %w", err)
	}
	if larkCfg.AppID == "" || larkCfg.AppSecret == "" {
		return "", fmt.Errorf("lark bot credentials not configured")
	}

	card := s.buildEnrichedCard(event, analysis, true)
	botClient := s.getBotClient(larkCfg.AppID, larkCfg.AppSecret, lark.BaseURLForDomain(larkCfg.Domain))

	msgID, err := botClient.SendDirectMessage(ctx, receiveIDType, receiveID, card)
	if err != nil {
		s.logger.Error("failed to send alert DM via Bot API",
			zap.Uint("event_id", event.ID),
			zap.String("receive_id_type", receiveIDType),
			zap.Error(err))
		return "", fmt.Errorf("lark bot DM failed: %w", err)
	}

	s.logger.Info("alert DM sent via Bot API",
		zap.Uint("event_id", event.ID),
		zap.String("receive_id_type", receiveIDType),
		zap.String("message_id", msgID),
	)
	return msgID, nil
}

// UpdateAlertCard patches the content of an existing card when the alert status changes.
// messageID is the value stored in alert_events.lark_message_id.
func (s *LarkService) UpdateAlertCard(ctx context.Context, event *model.AlertEvent, messageID string) error {
	if s.settingSvc == nil {
		return fmt.Errorf("settingSvc not configured for Bot API")
	}
	if messageID == "" {
		return nil // nothing to update
	}

	larkCfg, err := s.settingSvc.GetLarkConfig(ctx)
	if err != nil {
		return fmt.Errorf("failed to load lark config: %w", err)
	}
	if larkCfg.AppID == "" || larkCfg.AppSecret == "" {
		return fmt.Errorf("lark bot credentials not configured")
	}

	card := s.buildEnrichedCard(event, nil, true)
	botClient := s.getBotClient(larkCfg.AppID, larkCfg.AppSecret, lark.BaseURLForDomain(larkCfg.Domain))

	if err := botClient.UpdateMessage(ctx, messageID, card); err != nil {
		s.logger.Error("failed to update lark card",
			zap.Uint("event_id", event.ID),
			zap.String("message_id", messageID),
			zap.Error(err),
		)
		return fmt.Errorf("lark card update failed: %w", err)
	}

	s.logger.Info("lark card updated",
		zap.Uint("event_id", event.ID),
		zap.String("message_id", messageID),
		zap.String("new_status", string(event.Status)),
	)
	return nil
}

// HandleCardLifecycle handles card updates or deletions based on the Lark config strategy.
// Called by AlertEventService.triggerLarkCardUpdate on status changes.
func (s *LarkService) HandleCardLifecycle(ctx context.Context, event *model.AlertEvent) error {
	if s.settingSvc == nil {
		return nil
	}

	larkCfg, err := s.settingSvc.GetLarkConfig(ctx)
	if err != nil {
		return fmt.Errorf("failed to load lark config: %w", err)
	}
	if larkCfg.AppID == "" || larkCfg.AppSecret == "" {
		return nil
	}

	if !larkCfg.UpdateOnStateChange {
		return nil
	}

	// v2 path: CardKit entity-based status sync. v2 cards are tracked in
	// lark_card_entities, NOT via event.LarkMessageID — gating on the message
	// ID here would silently disable all v2 status updates.
	if larkCfg.CardSchemaVersion == "v2" && s.cardSvc != nil {
		return s.cardSvc.SyncCardStatus(ctx, event)
	}

	// v1 path requires the stored message ID.
	if event.LarkMessageID == "" {
		return nil
	}

	// v1 path: legacy PATCH-based card lifecycle.
	// For resolved/closed alerts, check the resolve strategy
	if event.Status == model.EventStatusResolved || event.Status == model.EventStatusClosed {
		if larkCfg.ResolveStrategy == "delete" {
			// Check business hours restriction
			if larkCfg.DeleteOnlyInBusinessHours && !isWithinBusinessHours(larkCfg.BusinessHoursStart, larkCfg.BusinessHoursEnd) {
				s.logger.Debug("card lifecycle: skipping delete outside business hours",
					zap.Uint("event_id", event.ID),
				)
				return nil
			}
			bot := s.getBotClient(larkCfg.AppID, larkCfg.AppSecret, lark.BaseURLForDomain(larkCfg.Domain))
			if err := bot.DeleteMessage(ctx, event.LarkMessageID); err != nil {
				return fmt.Errorf("delete lark card: %w", err)
			}
			s.logger.Info("lark card deleted on resolve",
				zap.Uint("event_id", event.ID),
				zap.String("message_id", event.LarkMessageID),
			)
			return nil
		}
	}

	// Default: update the card
	return s.UpdateAlertCard(ctx, event, event.LarkMessageID)
}

// isWithinBusinessHours checks if the current time is within the configured business hours.
func isWithinBusinessHours(start, end string) bool {
	if start == "" || end == "" {
		return true
	}

	now := time.Now()
	currentMinutes := now.Hour()*60 + now.Minute()

	var startH, startM, endH, endM int
	if n, _ := fmt.Sscanf(start, "%d:%d", &startH, &startM); n != 2 {
		return true // malformed input, safe default
	}
	if n, _ := fmt.Sscanf(end, "%d:%d", &endH, &endM); n != 2 {
		return true
	}
	if startH < 0 || startH > 23 || startM < 0 || startM > 59 || endH < 0 || endH > 23 || endM < 0 || endM > 59 {
		return true // out of range, safe default
	}

	startMinutes := startH*60 + startM
	endMinutes := endH*60 + endM

	if startMinutes <= endMinutes {
		return currentMinutes >= startMinutes && currentMinutes < endMinutes
	}
	// Overnight range (e.g., 22:00-06:00)
	return currentMinutes >= startMinutes || currentMinutes < endMinutes
}

// buildEnrichedCard constructs the Lark interactive card for an alert event.
// When useCallback is true, action buttons use Lark card callback (behaviour: "callback")
// instead of URL links. This is appropriate for Bot API delivery where the server can
// receive card.action.trigger events.
func (s *LarkService) buildEnrichedCard(event *model.AlertEvent, analysis *AlertAnalysis, useCallback bool) *lark.CardMessage {
	platformURL := ""
	if s.platformBaseURL != "" {
		platformURL = fmt.Sprintf("%s/alert-events/%d", s.platformBaseURL, event.ID)
	}
	actionBaseURL := ""
	var eventID uint
	if useCallback {
		eventID = event.ID
	} else if s.platformBaseURL != "" && s.jwtSecret != "" {
		token, err := GenerateAlertActionToken(event.ID, s.jwtSecret)
		if err == nil {
			actionBaseURL = fmt.Sprintf("%s/alert-action/%s", s.platformBaseURL, token)
		}
	}

	var aiResult *lark.AIAnalysisResult
	if analysis != nil {
		aiResult = &lark.AIAnalysisResult{
			Summary:          analysis.Summary,
			ProbableCauses:   analysis.ProbableCauses,
			Impact:           analysis.Impact,
			RecommendedSteps: analysis.RecommendedSteps,
		}
	}

	return lark.BuildEnrichedAlertCard(
		event.AlertName,
		string(event.Severity),
		string(event.Status),
		event.Labels,
		event.Annotations,
		event.FiredAt,
		aiResult,
		platformURL,
		actionBaseURL,
		eventID,
	)
}
