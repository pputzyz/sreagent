package service

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm"

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
// It handles card creation, multi-chat delivery, status synchronization
// (6-state mapping), debounced updates, and expiry re-issue.
type LarkCardStateService struct {
	cardRepo    *repository.LarkCardRepository
	cardKit     *lark.CardKitClient
	settingSvc  *SystemSettingService
	userRepo    *repository.UserRepository // optional: resolves acked-by/assignee names
	externalURL string                     // platform base URL for open_url buttons
	logger      *zap.Logger

	mu             sync.Mutex
	stopped        bool
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

// SetUserRepo injects the user repository so cards can show who acked/assigned.
func (s *LarkCardStateService) SetUserRepo(repo *repository.UserRepository) { s.userRepo = repo }

// SetExternalURL sets the platform base URL used by open_url card buttons.
func (s *LarkCardStateService) SetExternalURL(url string) {
	s.externalURL = strings.TrimRight(url, "/")
}

// Stop cancels all pending debounce timers. Called during graceful shutdown.
func (s *LarkCardStateService) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.stopped = true
	for id, timer := range s.debounceTimers {
		timer.Stop()
		delete(s.debounceTimers, id)
	}
}

// EnsureCardForEvent guarantees the event has an active card entity AND that
// the entity has been delivered to chatID. The same entity is reused across
// chats (one card, N messages) so a single entity update refreshes every chat.
func (s *LarkCardStateService) EnsureCardForEvent(ctx context.Context, event *model.AlertEvent, chatID string) (*model.LarkCardEntity, error) {
	entity, err := s.cardRepo.GetEntityByEventID(ctx, event.ID)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, fmt.Errorf("get card entity: %w", err)
	}

	if entity == nil {
		cfg, cfgErr := s.settingSvc.GetLarkConfig(ctx)
		if cfgErr != nil {
			return nil, fmt.Errorf("load lark config: %w", cfgErr)
		}
		cardJSON, buildErr := s.buildEventCardJSON(ctx, event, cfg)
		if buildErr != nil {
			return nil, fmt.Errorf("build card json: %w", buildErr)
		}
		cardID, createErr := s.cardKit.CreateCardEntity(ctx, cardJSON)
		if createErr != nil {
			return nil, fmt.Errorf("create card entity: %w", createErr)
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
	}

	// Deliver to this chat if not already delivered (multi-chat support).
	if err := s.deliverToChat(ctx, entity, chatID); err != nil {
		s.logger.Warn("failed to deliver card to chat",
			zap.Uint("event_id", event.ID),
			zap.String("chat_id", chatID),
			zap.Error(err))
	}

	return entity, nil
}

// deliverToChat sends the card entity to chatID unless a delivery record exists.
func (s *LarkCardStateService) deliverToChat(ctx context.Context, entity *model.LarkCardEntity, chatID string) error {
	if chatID == "" {
		return nil
	}
	msgs, err := s.cardRepo.GetMessagesByEntityID(ctx, entity.ID)
	if err != nil {
		return fmt.Errorf("list card messages: %w", err)
	}
	for _, m := range msgs {
		if m.ChatID == chatID {
			return nil // already delivered to this chat
		}
	}
	msgID, err := s.cardKit.SendCardByID(ctx, chatID, entity.CardID)
	if err != nil {
		return fmt.Errorf("send card to chat: %w", err)
	}
	return s.cardRepo.CreateMessage(ctx, &model.LarkCardMessage{
		CardEntityID: entity.ID,
		ChatID:       chatID,
		MessageID:    msgID,
	})
}

// SyncCardStatus updates the card to reflect the current alert status.
// Debounces rapid status changes: within cardDebounceWindow, only the final
// status snapshot is applied. A value copy of the event is captured so later
// mutations by the caller don't race with the timer goroutine.
func (s *LarkCardStateService) SyncCardStatus(ctx context.Context, event *model.AlertEvent) error {
	evCopy := *event

	s.mu.Lock()
	defer s.mu.Unlock()
	if s.stopped {
		return nil
	}

	// Cancel any pending debounce timer for this event (newer snapshot wins).
	if timer, ok := s.debounceTimers[evCopy.ID]; ok {
		timer.Stop()
	}

	s.debounceTimers[evCopy.ID] = time.AfterFunc(cardDebounceWindow, func() {
		s.mu.Lock()
		delete(s.debounceTimers, evCopy.ID)
		s.mu.Unlock()

		syncCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		if err := s.doSyncCardStatus(syncCtx, &evCopy); err != nil {
			s.logger.Error("card status sync failed",
				zap.Uint("event_id", evCopy.ID),
				zap.Error(err))
		}
	})
	return nil
}

// doSyncCardStatus performs the actual card update (called after debounce).
func (s *LarkCardStateService) doSyncCardStatus(ctx context.Context, event *model.AlertEvent) error {
	entity, err := s.cardRepo.GetEntityByEventID(ctx, event.ID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil // no card entity for this event — nothing to sync
		}
		return fmt.Errorf("get card entity: %w", err)
	}

	cfg, err := s.settingSvc.GetLarkConfig(ctx)
	if err != nil {
		return fmt.Errorf("load lark config: %w", err)
	}

	// Locally-known expiry: re-issue a fresh card to all chats.
	if time.Now().After(entity.ExpiresAt) {
		return s.reissueCard(ctx, entity, event, cfg)
	}

	cardJSON, err := s.buildEventCardJSON(ctx, event, cfg)
	if err != nil {
		return fmt.Errorf("build card json: %w", err)
	}

	seq, err := s.cardRepo.IncrementSequence(ctx, entity.ID)
	if err != nil {
		return fmt.Errorf("increment sequence: %w", err)
	}

	err = s.cardKit.UpdateCardEntity(ctx, entity.CardID, cardJSON, seq, "")
	if err == nil {
		return nil
	}

	var seqErr *lark.CardKitSequenceError
	if errors.As(err, &seqErr) {
		// Local counter drifted behind the remote: jump forward and retry once.
		s.logger.Warn("card sequence out of order, re-syncing", zap.Uint("event_id", event.ID))
		seq, seqSyncErr := s.cardRepo.JumpSequence(ctx, entity.ID, 100)
		if seqSyncErr != nil {
			return fmt.Errorf("sequence re-sync: %w", seqSyncErr)
		}
		if retryErr := s.cardKit.UpdateCardEntity(ctx, entity.CardID, cardJSON, seq, ""); retryErr != nil {
			// Still failing: re-issue a fresh entity rather than staying stuck.
			s.logger.Warn("sequence retry failed, re-issuing card", zap.Uint("event_id", event.ID), zap.Error(retryErr))
			return s.reissueCard(ctx, entity, event, cfg)
		}
		return nil
	}

	var expErr *lark.CardKitExpiredError
	if errors.As(err, &expErr) {
		// Remote says the entity passed its 14-day window: re-issue.
		return s.reissueCard(ctx, entity, event, cfg)
	}
	return err
}

// reissueCard supersedes an expired/stuck entity with a fresh one and delivers
// it to every chat the old entity was sent to.
func (s *LarkCardStateService) reissueCard(ctx context.Context, old *model.LarkCardEntity, event *model.AlertEvent, cfg LarkConfig) error {
	_ = s.cardRepo.UpdateStatus(ctx, old.ID, "superseded")

	cardJSON, err := s.buildEventCardJSON(ctx, event, cfg)
	if err != nil {
		return fmt.Errorf("build card json: %w", err)
	}
	cardID, err := s.cardKit.CreateCardEntity(ctx, cardJSON)
	if err != nil {
		return fmt.Errorf("re-create card entity: %w", err)
	}
	fresh := &model.LarkCardEntity{
		EventID:    &event.ID,
		CardID:     cardID,
		Sequence:   1,
		CardStatus: "active",
		ExpiresAt:  time.Now().Add(cardExpiryDuration),
	}
	if err := s.cardRepo.CreateEntity(ctx, fresh); err != nil {
		return fmt.Errorf("save re-issued card entity: %w", err)
	}

	msgs, err := s.cardRepo.GetMessagesByEntityID(ctx, old.ID)
	if err != nil {
		return fmt.Errorf("list old card chats: %w", err)
	}
	seen := make(map[string]bool, len(msgs))
	for _, m := range msgs {
		if seen[m.ChatID] {
			continue
		}
		seen[m.ChatID] = true
		if err := s.deliverToChat(ctx, fresh, m.ChatID); err != nil {
			s.logger.Warn("re-issue delivery failed",
				zap.Uint("event_id", event.ID), zap.String("chat_id", m.ChatID), zap.Error(err))
		}
	}
	s.logger.Info("card re-issued after expiry",
		zap.Uint("event_id", event.ID), zap.String("new_card_id", cardID))
	return nil
}

// ExpireOldCards marks all expired card entities. Should be called periodically.
func (s *LarkCardStateService) ExpireOldCards(ctx context.Context) (int64, error) {
	return s.cardRepo.ExpireOldCards(ctx)
}

// HasEntityForEvent reports whether the event has an active v2 card entity.
// Used by the card-action callback to decide between toast (v2, content is
// refreshed via CardKit) and legacy response-card (v1) replies.
func (s *LarkCardStateService) HasEntityForEvent(ctx context.Context, eventID uint) bool {
	entity, err := s.cardRepo.GetEntityByEventID(ctx, eventID)
	return err == nil && entity != nil
}

// buildEventCardJSON builds a Card 2.0 JSON string for an alert event.
// The card content covers all six platform states and the action buttons
// follow cfg.CardInteractionMode (callback vs open_url).
func (s *LarkCardStateService) buildEventCardJSON(ctx context.Context, event *model.AlertEvent, cfg LarkConfig) (string, error) {
	template := lark.StatusToTemplate(string(event.Status))

	builder := lark.NewCardV2Builder().
		Config(&lark.CardV2Config{
			WideScreenMode: true,
			Summary: &lark.CardV2Text{Tag: "plain_text",
				Content: fmt.Sprintf("[%s·%s] %s", event.Severity, event.Status, event.AlertName)},
		}).
		Header(fmt.Sprintf("%s %s", severityEmoji(string(event.Severity)), event.AlertName), template)

	// Key facts row.
	left := fmt.Sprintf("**严重等级:** %s\n**状态:** %s", event.Severity, statusLabel(event.Status))
	right := fmt.Sprintf("**触发时间:** %s\n**触发次数:** %d", event.FiredAt.Format("01-02 15:04:05"), event.FireCount)
	builder.AddColumnSet(
		lark.NewColumn(1, lark.NewMarkdown(left)),
		lark.NewColumn(1, lark.NewMarkdown(right)),
	)

	// State-specific context line (who acked/assigned, silence window, etc.).
	if line := s.stateContextLine(ctx, event); line != "" {
		builder.AddMarkdown(line)
	}

	// AI summary from the notification pipeline, when present.
	if aiSummary := event.Annotations["ai_summary"]; aiSummary != "" {
		builder.AddMarkdown("**🤖 AI 分析:** " + aiSummary)
	}

	// Labels / annotations folded away so the card stays compact.
	if len(event.Labels) > 0 {
		builder.AddCollapsiblePanel("Labels", false, lark.NewMarkdown(sortedKVMarkdown(event.Labels)))
	}
	if len(event.Annotations) > 0 {
		annos := make(map[string]string, len(event.Annotations))
		for k, v := range event.Annotations {
			if k == "ai_summary" {
				continue // already shown above
			}
			annos[k] = v
		}
		if len(annos) > 0 {
			builder.AddCollapsiblePanel("Annotations", false, lark.NewMarkdown(sortedKVMarkdown(annos)))
		}
	}

	s.addActionButtons(builder, event, cfg)

	return builder.BuildJSON()
}

// stateContextLine renders the per-state context (operator names, silence window).
func (s *LarkCardStateService) stateContextLine(ctx context.Context, event *model.AlertEvent) string {
	switch event.Status {
	case model.EventStatusAcknowledged:
		return "✅ **已认领**" + s.userSuffix(ctx, event.AckedBy)
	case model.EventStatusAssigned:
		return "👤 **已指派**" + s.userSuffix(ctx, event.AssignedTo)
	case model.EventStatusSilenced:
		line := "🔇 **已静默**"
		if event.SilencedUntil != nil {
			line += fmt.Sprintf("，至 %s", event.SilencedUntil.Format("01-02 15:04"))
		}
		if event.SilenceReason != "" {
			line += fmt.Sprintf("（%s）", event.SilenceReason)
		}
		return line
	case model.EventStatusResolved:
		line := "🟢 **已恢复**"
		if event.ResolvedAt != nil {
			line += fmt.Sprintf("，%s", event.ResolvedAt.Format("01-02 15:04:05"))
		}
		return line
	case model.EventStatusClosed:
		return "⚫ **已关闭**"
	}
	return ""
}

// userSuffix renders "：DisplayName <at>" for the given user ID, best-effort.
func (s *LarkCardStateService) userSuffix(ctx context.Context, userID *uint) string {
	if userID == nil || *userID == 0 || s.userRepo == nil {
		return ""
	}
	user, err := s.userRepo.GetByID(ctx, *userID)
	if err != nil || user == nil {
		return ""
	}
	suffix := "：" + user.DisplayName
	if user.LarkUserID != "" {
		suffix += fmt.Sprintf(" <at id=%s></at>", user.LarkUserID)
	}
	return suffix
}

// addActionButtons appends state- and mode-appropriate buttons.
func (s *LarkCardStateService) addActionButtons(builder *lark.CardV2Builder, event *model.AlertEvent, cfg LarkConfig) {
	detailURL := ""
	if s.externalURL != "" {
		detailURL = fmt.Sprintf("%s/alerts/events/%d", s.externalURL, event.ID)
	}

	openURLOnly := cfg.CardInteractionMode == "" || cfg.CardInteractionMode == "open_url"

	var buttons []interface{}
	switch event.Status {
	case model.EventStatusFiring:
		if openURLOnly {
			if detailURL != "" {
				buttons = append(buttons, lark.NewButton("处理告警", "open_url", "primary",
					map[string]interface{}{"default_url": detailURL}))
			}
		} else {
			buttons = append(buttons,
				lark.NewButton("✓ 认领", "callback", "primary", map[string]interface{}{
					"action": "ack", "event_id": event.ID,
				}),
				lark.NewButton("解决", "callback", "default", map[string]interface{}{
					"action": "resolve", "event_id": event.ID,
				}),
				lark.NewButton("静默 1h", "callback", "default", map[string]interface{}{
					"action": "silence", "event_id": event.ID, "duration": 60,
				}),
				lark.NewButton("静默 24h", "callback", "default", map[string]interface{}{
					"action": "silence", "event_id": event.ID, "duration": 1440,
				}),
			)
		}
	case model.EventStatusAcknowledged, model.EventStatusAssigned:
		if !openURLOnly {
			buttons = append(buttons,
				lark.NewButton("解决", "callback", "primary", map[string]interface{}{
					"action": "resolve", "event_id": event.ID,
				}),
				lark.NewButton("静默 1h", "callback", "default", map[string]interface{}{
					"action": "silence", "event_id": event.ID, "duration": 60,
				}),
			)
		}
	case model.EventStatusSilenced:
		if !openURLOnly {
			buttons = append(buttons,
				lark.NewButton("解决", "callback", "default", map[string]interface{}{
					"action": "resolve", "event_id": event.ID,
				}),
			)
		}
		// resolved / closed: terminal display, no operational buttons.
	}

	// Detail link is always useful (except when it IS the only button already).
	if detailURL != "" && !openURLOnly {
		buttons = append(buttons, lark.NewButton("查看详情", "open_url", "default",
			map[string]interface{}{"default_url": detailURL}))
	}

	if len(buttons) > 0 {
		builder.AddActions(buttons...)
	}
}

// statusLabel renders a human-readable Chinese status label.
func statusLabel(status model.AlertEventStatus) string {
	switch status {
	case model.EventStatusFiring:
		return "🔴 告警中"
	case model.EventStatusAcknowledged:
		return "🟠 已认领"
	case model.EventStatusAssigned:
		return "🔵 已指派"
	case model.EventStatusSilenced:
		return "🟡 已静默"
	case model.EventStatusResolved:
		return "🟢 已恢复"
	case model.EventStatusClosed:
		return "⚫ 已关闭"
	default:
		return string(status)
	}
}

// sortedKVMarkdown renders a label map as deterministic markdown lines.
func sortedKVMarkdown(kv map[string]string) string {
	keys := make([]string, 0, len(kv))
	for k := range kv {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var b strings.Builder
	for _, k := range keys {
		fmt.Fprintf(&b, "**%s:** %s\n", k, kv[k])
	}
	return b.String()
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

// --- Streaming-style reply cards (@bot conversations, T4-3) ---
//
// NOTE: this is accumulate-and-replace pseudo-streaming via CardKit entity
// updates (rate-limited per card). The dedicated CardKit element-content
// streaming endpoint is intentionally NOT used until its exact contract is
// PoC-verified on a Lark International tenant — see docs/lark-assistant-plan.md.

// StreamStep is one completed agent step shown on a streaming reply card.
type StreamStep struct {
	Step     int
	ToolName string
	Content  string
}

// CreateStreamingCard creates a CardKit reply card and sends it to the chat.
func (s *LarkCardStateService) CreateStreamingCard(ctx context.Context, chatID, question string) (*model.LarkCardEntity, error) {
	cardJSON, err := lark.NewCardV2Builder().
		Config(&lark.CardV2Config{
			WideScreenMode: true,
			Summary:        &lark.CardV2Text{Tag: "plain_text", Content: "🤔 分析中..."},
		}).
		Header("🤖 SRE Agent", "blue").
		AddMarkdown("**问题:** " + question + "\n\n⏳ 正在分析...").
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

	if err := s.deliverToChat(ctx, entity, chatID); err != nil {
		s.logger.Warn("failed to send streaming card to chat",
			zap.String("chat_id", chatID), zap.Error(err))
	}
	return entity, nil
}

// UpdateStreamingProgress re-renders the reply card with ALL completed steps
// so far (accumulate-and-replace; earlier steps remain visible).
func (s *LarkCardStateService) UpdateStreamingProgress(ctx context.Context, entityID uint, question string, steps []StreamStep) error {
	builder := lark.NewCardV2Builder().
		Config(&lark.CardV2Config{
			WideScreenMode: true,
			Summary:        &lark.CardV2Text{Tag: "plain_text", Content: fmt.Sprintf("⏳ 已完成 %d 步", len(steps))},
		}).
		Header("🤖 SRE Agent", "blue").
		AddMarkdown("**问题:** " + question + "\n\n⏳ 正在分析...")
	for _, st := range steps {
		builder.AddCollapsiblePanel(
			fmt.Sprintf("步骤 %d: %s", st.Step, st.ToolName), false,
			lark.NewMarkdown(truncateForCard(st.Content, 2000)),
		)
	}
	return s.updateStreamingEntity(ctx, entityID, builder)
}

// FinalizeStreamingCard renders the final answer (steps folded below it).
func (s *LarkCardStateService) FinalizeStreamingCard(ctx context.Context, entityID uint, question, answer string, steps []StreamStep) error {
	builder := lark.NewCardV2Builder().
		Config(&lark.CardV2Config{
			WideScreenMode: true,
			Summary:        &lark.CardV2Text{Tag: "plain_text", Content: truncateForCard(answer, 100)},
		}).
		Header("🤖 SRE Agent", "green").
		AddMarkdown(fmt.Sprintf("**问题:** %s\n\n---\n\n%s", question, answer))
	for _, st := range steps {
		builder.AddCollapsiblePanel(
			fmt.Sprintf("步骤 %d: %s", st.Step, st.ToolName), false,
			lark.NewMarkdown(truncateForCard(st.Content, 2000)),
		)
	}
	return s.updateStreamingEntity(ctx, entityID, builder)
}

// updateStreamingEntity builds and pushes a card update with sequence handling.
func (s *LarkCardStateService) updateStreamingEntity(ctx context.Context, entityID uint, builder *lark.CardV2Builder) error {
	cardJSON, err := builder.BuildJSON()
	if err != nil {
		return fmt.Errorf("build streaming update: %w", err)
	}
	entity, err := s.cardRepo.GetEntityByID(ctx, entityID)
	if err != nil {
		return fmt.Errorf("get streaming entity: %w", err)
	}
	seq, err := s.cardRepo.IncrementSequence(ctx, entity.ID)
	if err != nil {
		return fmt.Errorf("increment sequence: %w", err)
	}
	return s.cardKit.UpdateCardEntity(ctx, entity.CardID, cardJSON, seq, "")
}

// truncateForCard truncates text (rune-safe) to a maximum length for card display.
func truncateForCard(s string, maxLen int) string {
	runes := []rune(s)
	if len(runes) <= maxLen {
		return s
	}
	return string(runes[:maxLen-1]) + "…"
}
