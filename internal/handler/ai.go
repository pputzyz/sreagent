package handler

import (
	"github.com/gin-gonic/gin"

	"github.com/sreagent/sreagent/internal/model"
	apperr "github.com/sreagent/sreagent/internal/pkg/errors"
	"github.com/sreagent/sreagent/internal/service"
)

// AIHandler handles AI/LLM-related API endpoints.
type AIHandler struct {
	aiSvc          *service.AIService
	eventSvc       *service.AlertEventService
	chatHistorySvc *service.ChatHistoryService
	petSvc         *service.PetService
}

// NewAIHandler creates a new AIHandler.
func NewAIHandler(aiSvc *service.AIService, eventSvc *service.AlertEventService, chatHistorySvc *service.ChatHistoryService, petSvc *service.PetService) *AIHandler {
	return &AIHandler{aiSvc: aiSvc, eventSvc: eventSvc, chatHistorySvc: chatHistorySvc, petSvc: petSvc}
}

// systemPrompts maps chat modes to their system prompts.
var systemPrompts = map[string]string{
	"alert":   "你是一位资深 SRE 助手，擅长分析告警、定位根因、推荐 SOP。用中文回答，简洁专业。",
	"general": "你是一位友好的 AI 助手，可以回答任何问题。用中文回答。",
	"pet":     "你是一只活泼、好奇、偶尔犯傻的狐狸宠物。你的名字由用户决定。回复风格简短有趣，偶尔卖萌。用中文回答。",
}

// Chat handles multi-turn chat conversations.
func (h *AIHandler) Chat(c *gin.Context) {
	var req struct {
		Mode    string `json:"mode" binding:"required"`
		Message string `json:"message" binding:"required,max=4000"`
		Context string `json:"context,omitempty"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		Error(c, apperr.WithMessage(apperr.ErrInvalidParam, err.Error()))
		return
	}

	systemPrompt, ok := systemPrompts[req.Mode]
	if !ok {
		Error(c, apperr.WithMessage(apperr.ErrInvalidParam, "invalid mode: must be alert, general, or pet"))
		return
	}

	userID := GetCurrentUserID(c)

	// Load recent history (20 messages) for context
	historyMsgs, err := h.chatHistorySvc.GetHistory(c.Request.Context(), userID, req.Mode)
	if err != nil {
		Error(c, apperr.WithMessage(apperr.ErrDatabase, "failed to load chat history: "+err.Error()))
		return
	}

	// Convert to ChatMessage slice (limit to last 20)
	history := make([]service.ChatMessage, 0, len(historyMsgs))
	if len(historyMsgs) > 20 {
		historyMsgs = historyMsgs[len(historyMsgs)-20:]
	}
	for _, msg := range historyMsgs {
		history = append(history, service.ChatMessage{
			Role:    msg.Role,
			Content: msg.Content,
		})
	}

	// Call LLM
	reply, err := h.aiSvc.Chat(c.Request.Context(), systemPrompt, history, req.Message)
	if err != nil {
		Error(c, apperr.WithMessage(apperr.ErrExternalAPI, "AI chat failed: "+err.Error()))
		return
	}

	// Save user message
	userMsg := &model.ChatHistory{
		UserID:  userID,
		Mode:    req.Mode,
		Role:    "user",
		Content: req.Message,
		Context: req.Context,
	}
	if err := h.chatHistorySvc.Save(c.Request.Context(), userMsg); err != nil {
		Error(c, apperr.WithMessage(apperr.ErrDatabase, "failed to save user message: "+err.Error()))
		return
	}

	// Save assistant reply
	assistantMsg := &model.ChatHistory{
		UserID:  userID,
		Mode:    req.Mode,
		Role:    "assistant",
		Content: reply,
	}
	if err := h.chatHistorySvc.Save(c.Request.Context(), assistantMsg); err != nil {
		Error(c, apperr.WithMessage(apperr.ErrDatabase, "failed to save assistant message: "+err.Error()))
		return
	}

	// Award pet exp for pet mode chat (non-critical)
	if req.Mode == "pet" {
		_, _ = h.petSvc.AddChatExp(c.Request.Context(), userID)
	}

	Success(c, gin.H{"reply": reply, "mode": req.Mode})
}

// GetHistory returns chat history for the current user and mode.
func (h *AIHandler) GetHistory(c *gin.Context) {
	mode := c.Query("mode")
	if mode == "" {
		Error(c, apperr.WithMessage(apperr.ErrInvalidParam, "mode query parameter is required"))
		return
	}

	userID := GetCurrentUserID(c)

	msgs, err := h.chatHistorySvc.GetHistory(c.Request.Context(), userID, mode)
	if err != nil {
		Error(c, apperr.WithMessage(apperr.ErrDatabase, "failed to load chat history: "+err.Error()))
		return
	}

	Success(c, msgs)
}

// ClearHistory deletes chat history for the current user and mode.
func (h *AIHandler) ClearHistory(c *gin.Context) {
	mode := c.Query("mode")
	if mode == "" {
		Error(c, apperr.WithMessage(apperr.ErrInvalidParam, "mode query parameter is required"))
		return
	}

	userID := GetCurrentUserID(c)

	if err := h.chatHistorySvc.ClearHistory(c.Request.Context(), userID, mode); err != nil {
		Error(c, apperr.WithMessage(apperr.ErrDatabase, "failed to clear chat history: "+err.Error()))
		return
	}

	Success(c, gin.H{"message": "chat history cleared", "mode": mode})
}

// GenerateReport generates an AI-powered alert report.
func (h *AIHandler) GenerateReport(c *gin.Context) {
	var req struct {
		EventID uint `json:"event_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		Error(c, apperr.WithMessage(apperr.ErrInvalidParam, err.Error()))
		return
	}

	event, err := h.eventSvc.GetByID(c.Request.Context(), req.EventID)
	if err != nil {
		Error(c, err)
		return
	}

	report, err := h.aiSvc.GenerateAlertReport(c.Request.Context(), event)
	if err != nil {
		Error(c, apperr.WithMessage(apperr.ErrExternalAPI, err.Error()))
		return
	}

	Success(c, gin.H{"report": report, "event_id": req.EventID})
}

// SuggestSOP suggests Standard Operating Procedure steps for an alert.
func (h *AIHandler) SuggestSOP(c *gin.Context) {
	var req struct {
		EventID uint `json:"event_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		Error(c, apperr.WithMessage(apperr.ErrInvalidParam, err.Error()))
		return
	}

	event, err := h.eventSvc.GetByID(c.Request.Context(), req.EventID)
	if err != nil {
		Error(c, err)
		return
	}

	sop, err := h.aiSvc.SuggestSOP(c.Request.Context(), event)
	if err != nil {
		Error(c, apperr.WithMessage(apperr.ErrExternalAPI, err.Error()))
		return
	}

	Success(c, gin.H{"sop": sop, "event_id": req.EventID})
}

// GetConfig returns the current AI configuration with masked API key.
func (h *AIHandler) GetConfig(c *gin.Context) {
	cfg, err := h.aiSvc.GetConfig(c.Request.Context())
	if err != nil {
		Error(c, apperr.WithMessage(apperr.ErrExternalAPI, "failed to load AI config: "+err.Error()))
		return
	}
	Success(c, cfg)
}

// UpdateConfig updates the AI configuration.
func (h *AIHandler) UpdateConfig(c *gin.Context) {
	var req service.AIConfig
	if err := c.ShouldBindJSON(&req); err != nil {
		Error(c, apperr.WithMessage(apperr.ErrInvalidParam, err.Error()))
		return
	}

	if err := h.aiSvc.UpdateConfig(c.Request.Context(), req); err != nil {
		Error(c, apperr.WithMessage(apperr.ErrExternalAPI, "failed to save AI config: "+err.Error()))
		return
	}
	Success(c, gin.H{"message": "AI configuration updated"})
}

// TestConnection tests connectivity to the configured AI provider.
func (h *AIHandler) TestConnection(c *gin.Context) {
	if err := h.aiSvc.TestConnection(c.Request.Context()); err != nil {
		Error(c, apperr.WithMessage(apperr.ErrExternalAPI, "AI connection test failed: "+err.Error()))
		return
	}

	Success(c, gin.H{"message": "AI connection successful"})
}

// GetModules returns the AI module configuration.
func (h *AIHandler) GetModules(c *gin.Context) {
	cfg, err := h.aiSvc.GetAIModules(c.Request.Context())
	if err != nil {
		Error(c, apperr.WithMessage(apperr.ErrExternalAPI, "failed to load AI modules: "+err.Error()))
		return
	}
	Success(c, cfg)
}

// UpdateModules updates the AI module configuration.
func (h *AIHandler) UpdateModules(c *gin.Context) {
	var req service.AIModuleConfig
	if err := c.ShouldBindJSON(&req); err != nil {
		Error(c, apperr.WithMessage(apperr.ErrInvalidParam, err.Error()))
		return
	}
	if err := h.aiSvc.UpdateAIModules(c.Request.Context(), &req); err != nil {
		Error(c, apperr.WithMessage(apperr.ErrExternalAPI, "failed to save AI modules: "+err.Error()))
		return
	}
	Success(c, gin.H{"message": "AI modules configuration updated"})
}

// GetProviders returns the multi-provider AI configuration with masked API keys.
func (h *AIHandler) GetProviders(c *gin.Context) {
	cfg, err := h.aiSvc.GetProvidersConfig(c.Request.Context())
	if err != nil {
		Error(c, apperr.WithMessage(apperr.ErrExternalAPI, "failed to load AI providers: "+err.Error()))
		return
	}
	Success(c, cfg)
}

// SaveProviders updates the multi-provider AI configuration.
func (h *AIHandler) SaveProviders(c *gin.Context) {
	var req service.AIProvidersConfig
	if err := c.ShouldBindJSON(&req); err != nil {
		Error(c, apperr.WithMessage(apperr.ErrInvalidParam, err.Error()))
		return
	}

	// Validate at least one provider is enabled.
	hasEnabled := false
	for _, p := range req.Providers {
		if p.Enabled {
			hasEnabled = true
			break
		}
	}
	if !hasEnabled {
		Error(c, apperr.WithMessage(apperr.ErrInvalidParam, "at least one AI provider must be enabled"))
		return
	}

	if err := h.aiSvc.SaveProvidersConfig(c.Request.Context(), req); err != nil {
		Error(c, apperr.WithMessage(apperr.ErrExternalAPI, "failed to save AI providers: "+err.Error()))
		return
	}
	Success(c, gin.H{"message": "AI providers configuration updated"})
}

// TestProvider tests connectivity to a specific provider by key.
func (h *AIHandler) TestProvider(c *gin.Context) {
	var req struct {
		Key string `json:"key" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		Error(c, apperr.WithMessage(apperr.ErrInvalidParam, err.Error()))
		return
	}
	if err := h.aiSvc.TestProviderConnection(c.Request.Context(), req.Key); err != nil {
		Error(c, apperr.WithMessage(apperr.ErrExternalAPI, "AI connection test failed: "+err.Error()))
		return
	}
	Success(c, gin.H{"message": "AI connection successful"})
}
