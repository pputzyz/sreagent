package service

import (
	"context"
	"fmt"
	"sync"
	"time"

	"go.uber.org/zap"

	"github.com/sreagent/sreagent/internal/model"
	"github.com/sreagent/sreagent/internal/pkg/lark"
	"github.com/sreagent/sreagent/internal/repository"
)

const (
	// cardDebounceWindow is the time window to merge rapid status changes.
	cardDebounceWindow = 500 * time.Millisecond
	// cardExpiryDuration is the default card entity lifetime.
	cardExpiryDuration = 14 * 24 * time.Hour // 14 days
)

// LarkCardStateService manages CardKit card entities for alert events.
// It handles card creation, status synchronization (6-state mapping),
// debounced updates, and expiry.
type LarkCardStateService struct {
	cardRepo   *repository.LarkCardRepository
	cardKit    *lark.CardKitClient
	settingSvc *SystemSettingService
	logger     *zap.Logger

	mu            sync.Mutex
	debounceTimers map[uint]*time.Timer // eventID → pending update timer
}

// NewLarkCardStateService creates a new LarkCardStateService.
func NewLarkCardStateService(
	cardRepo *repository.LarkCardRepository,
	cardKit *lark.CardKitClient,
	settingSvc *SystemSettingService,
	logger *zap.Logger,
) *LarkCardStateService {
	return &LarkCardStateService{
		cardRepo:       cardRepo,
		cardKit:        cardKit,
		settingSvc:     settingSvc,
		logger:         logger,
		debounceTimers: make(map[uint]*time.Timer),
	}
}

// EnsureCardForEvent returns the active card entity for an event, or creates a new one
// and sends it to the specified chat. If the entity already exists, it is returned as-is.
func (s *LarkCardStateService) EnsureCardForEvent(ctx context.Context, event *model.AlertEvent, chatID string) (*model.LarkCardEntity, error) {
	// Check for existing active entity.
	entity, err := s.cardRepo.GetEntityByEventID(ctx, event.ID)
	if err == nil && entity != nil {
		return entity, nil
	}

	// Create a new card entity.
	cardJSON, err := s.buildEventCardJSON(event)
	if err != nil {
		return nil, fmt.Errorf("build card json: %w", err)
	}

	cardID, err := s.cardKit.CreateCardEntity(ctx, cardJSON)
	if err != nil {
		return nil, fmt.Errorf("create card entity: %w", err)
	}

	entity = &model.LarkCardEntity{
		EventID:    &event.ID,
		CardID:     cardID,
		Sequence:   1,
		CardStatus: "active",
		ExpiresAt:  time.Now().Add(cardExpiryDuration),
	}
	if err := s.cardRepo.CreateEntity(ctx, entity); err != nil {
		return nil, fmt.Errorf("save card entity: %w", err)
	}

	// Send to the target chat.
	msgID, err := s.cardKit.SendCardByID(ctx, chatID, cardID)
	if err != nil {
		s.logger.Warn("failed to send card to chat",
			zap.Uint("event_id", event.ID),
			zap.String("chat_id", chatID),
			zap.Error(err))
	} else {
		_ = s.cardRepo.CreateMessage(ctx, &model.LarkCardMessage{
			CardEntityID: entity.ID,
			ChatID:       chatID,
			MessageID:    msgID,
		})
	}

	return entity, nil
}

// SyncCardStatus updates the card to reflect the current alert status.
// Debounces rapid status changes: within cardDebounceWindow, only the final status is applied.
func (s *LarkCardStateService) SyncCardStatus(ctx context.Context, event *model.AlertEvent) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Cancel any pending debounce timer for this event.
	if timer, ok := s.debounceTimers[event.ID]; ok {
		timer.Stop()
	}

	// Schedule the update after the debounce window.
	s.debounceTimers[event.ID] = time.AfterFunc(cardDebounceWindow, func() {
		s.mu.Lock()
		delete(s.debounceTimers, event.ID)
		s.mu.Unlock()

		if err := s.doSyncCardStatus(context.Background(), event); err != nil {
			s.logger.Error("card status sync failed",
				zap.Uint("event_id", event.ID),
				zap.Error(err))
		}
	})
	return nil
}

// doSyncCardStatus performs the actual card update (called after debounce).
func (s *LarkCardStateService) doSyncCardStatus(ctx context.Context, event *model.AlertEvent) error {
	entity, err := s.cardRepo.GetEntityByEventID(ctx, event.ID)
	if err != nil {
		return fmt.Errorf("get card entity: %w", err)
	}
	if entity == nil {
		return nil // no card entity for this event
	}

	// Check if entity has expired.
	if time.Now().After(entity.ExpiresAt) {
		_ = s.cardRepo.UpdateStatus(ctx, entity.ID, "expired")
		return fmt.Errorf("card entity expired for event %d", event.ID)
	}

	// Build updated card JSON.
	cardJSON, err := s.buildEventCardJSON(event)
	if err != nil {
		return fmt.Errorf("build card json: %w", err)
	}

	// Increment sequence atomically.
	seq, err := s.cardRepo.IncrementSequence(ctx, entity.ID)
	if err != nil {
		return fmt.Errorf("increment sequence: %w", err)
	}

	// Update the card entity.
	if err := s.cardKit.UpdateCardEntity(ctx, entity.CardID, cardJSON, seq, ""); err != nil {
		// Handle sequence error: re-sync and retry once.
		if _, ok := err.(*lark.CardKitSequenceError); ok {
			s.logger.Warn("card sequence error, retrying", zap.Uint("event_id", event.ID))
			seq, _ = s.cardRepo.IncrementSequence(ctx, entity.ID)
			return s.cardKit.UpdateCardEntity(ctx, entity.CardID, cardJSON, seq, "")
		}
		// Handle expired error: mark as expired.
		if _, ok := err.(*lark.CardKitExpiredError); ok {
			_ = s.cardRepo.UpdateStatus(ctx, entity.ID, "expired")
			return nil
		}
		return err
	}

	return nil
}

// ExpireOldCards marks all expired card entities. Should be called periodically.
func (s *LarkCardStateService) ExpireOldCards(ctx context.Context) (int64, error) {
	return s.cardRepo.ExpireOldCards(ctx)
}

// buildEventCardJSON builds a Card 2.0 JSON string for an alert event.
func (s *LarkCardStateService) buildEventCardJSON(event *model.AlertEvent) (string, error) {
	template := lark.StatusToTemplate(string(event.Status))

	builder := lark.NewCardV2Builder().
		Config(&lark.CardV2Config{
			WideScreenMode: true,
			Summary:        &lark.CardV2Text{Tag: "plain_text", Content: fmt.Sprintf("[%s] %s", event.Severity, event.AlertName)},
		}).
		Header(fmt.Sprintf("%s %s", severityEmoji(string(event.Severity)), event.AlertName), template)

	// Status info.
	statusText := fmt.Sprintf("**Status:** %s", event.Status)
	if event.Status == model.EventStatusSilenced && event.SilencedUntil != nil {
		statusText += fmt.Sprintf(" (until %s)", event.SilencedUntil.Format("15:04"))
	}
	builder.AddMarkdown(statusText)

	// Labels in collapsible panel if present.
	if len(event.Labels) > 0 {
		labelsMD := ""
		for k, v := range event.Labels {
			labelsMD += fmt.Sprintf("**%s:** %s\n", k, v)
		}
		builder.AddCollapsiblePanel("Labels", false, lark.NewMarkdown(labelsMD))
	}

	// Action buttons based on status.
	switch event.Status {
	case model.EventStatusFiring, model.EventStatusAcknowledged:
		builder.AddActions(
			lark.NewButton("✓ Acknowledge", "callback", "primary", map[string]interface{}{
				"action": "ack", "event_id": event.ID,
			}),
			lark.NewButton("🔇 Silence", "callback", "default", map[string]interface{}{
				"action": "silence_form", "event_id": event.ID,
			}),
		)
	}

	return builder.BuildJSON()
}

func severityEmoji(severity string) string {
	switch severity {
	case "critical":
		return "🔴"
	case "error":
		return "🟠"
	case "warning":
		return "🟡"
	case "info":
		return "🔵"
	default:
		return "⚪"
	}
}

// --- Streaming card support for T4-3 ---

// CreateStreamingCard creates a new CardKit card entity in "streaming" mode
// and sends it to the given chat. Returns the card entity ID for subsequent
// AppendContent / FinalizeCard calls.
func (s *LarkCardStateService) CreateStreamingCard(ctx context.Context, chatID, question string) (*model.LarkCardEntity, error) {
	cardJSON, err := lark.NewCardV2Builder().
		Config(&lark.CardV2Config{
			WideScreenMode: true,
			Summary:        &lark.CardV2Text{Tag: "plain_text", Content: "🤔 分析中..."},
		}).
		Header("🤖 SRE Agent", "blue").
		AddMarkdown("🤔 **正在分析您的问题...**\n\n" + question).
		BuildJSON()
	if err != nil {
		return nil, fmt.Errorf("build streaming card: %w", err)
	}

	cardID, err := s.cardKit.CreateCardEntity(ctx, cardJSON)
	if err != nil {
		return nil, fmt.Errorf("create streaming card entity: %w", err)
	}

	entity := &model.LarkCardEntity{
		CardID:     cardID,
		Sequence:   1,
		CardStatus: "active",
		ExpiresAt:  time.Now().Add(cardExpiryDuration),
	}
	if err := s.cardRepo.CreateEntity(ctx, entity); err != nil {
		return nil, fmt.Errorf("save streaming card entity: %w", err)
	}

	// Send to the target chat.
	msgID, err := s.cardKit.SendCardByID(ctx, chatID, cardID)
	if err != nil {
		s.logger.Warn("failed to send streaming card to chat",
			zap.String("chat_id", chatID), zap.Error(err))
	} else {
		_ = s.cardRepo.CreateMessage(ctx, &model.LarkCardMessage{
			CardEntityID: entity.ID,
			ChatID:       chatID,
			MessageID:    msgID,
		})
	}

	return entity, nil
}

// AppendStreamingContent appends a tool step result to a streaming card.
// Each call updates the card entity with the accumulated content so far.
func (s *LarkCardStateService) AppendStreamingContent(ctx context.Context, entityID uint, step int, toolName, content string) error {
	entity, err := s.cardRepo.GetEntityByID(ctx, entityID)
	if err != nil {
		return fmt.Errorf("get streaming entity: %w", err)
	}

	// Build updated card with the new step appended.
	cardJSON, err := lark.NewCardV2Builder().
		Config(&lark.CardV2Config{
			WideScreenMode: true,
			Summary:        &lark.CardV2Text{Tag: "plain_text", Content: fmt.Sprintf("⏳ 步骤 %d: %s", step, toolName)},
		}).
		Header("🤖 SRE Agent", "blue").
		AddMarkdown(fmt.Sprintf("⏳ **步骤 %d: `%s`** 已完成", step, toolName)).
		AddCollapsiblePanel(fmt.Sprintf("步骤 %d 结果", step), false,
			lark.NewMarkdown(truncateForCard(content, 2000))).
		BuildJSON()
	if err != nil {
		return fmt.Errorf("build append card: %w", err)
	}

	seq, err := s.cardRepo.IncrementSequence(ctx, entity.ID)
	if err != nil {
		return fmt.Errorf("increment sequence: %w", err)
	}

	return s.cardKit.UpdateCardEntity(ctx, entity.CardID, cardJSON, seq, "")
}

// FinalizeStreamingCard updates the streaming card with the final answer.
func (s *LarkCardStateService) FinalizeStreamingCard(ctx context.Context, entityID uint, question, answer string) error {
	entity, err := s.cardRepo.GetEntityByID(ctx, entityID)
	if err != nil {
		return fmt.Errorf("get streaming entity: %w", err)
	}

	cardJSON, err := lark.NewCardV2Builder().
		Config(&lark.CardV2Config{
			WideScreenMode: true,
			Summary:        &lark.CardV2Text{Tag: "plain_text", Content: truncateForCard(answer, 100)},
		}).
		Header("🤖 SRE Agent", "green").
		AddMarkdown(fmt.Sprintf("**问题:** %s\n\n---\n\n**回答:**\n%s", question, answer)).
		BuildJSON()
	if err != nil {
		return fmt.Errorf("build finalize card: %w", err)
	}

	seq, err := s.cardRepo.IncrementSequence(ctx, entity.ID)
	if err != nil {
		return fmt.Errorf("increment sequence: %w", err)
	}

	return s.cardKit.UpdateCardEntity(ctx, entity.CardID, cardJSON, seq, "")
}

// truncateForCard truncates text to a maximum length for card display.
func truncateForCard(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}
