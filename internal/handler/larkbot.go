package handler

import (
	"io"

	"github.com/gin-gonic/gin"

	apperr "github.com/sreagent/sreagent/internal/pkg/errors"
	"github.com/sreagent/sreagent/internal/service"
)

// LarkBotHandler handles Lark bot API endpoints.
type LarkBotHandler struct {
	svc *service.LarkBotService
}

// NewLarkBotHandler creates a new LarkBotHandler.
func NewLarkBotHandler(svc *service.LarkBotService) *LarkBotHandler {
	return &LarkBotHandler{svc: svc}
}

// EventCallback handles incoming Lark event subscription callbacks.
// This endpoint receives both URL verification challenges and message events.
func (h *LarkBotHandler) EventCallback(c *gin.Context) {
	body, err := io.ReadAll(io.LimitReader(c.Request.Body, 1<<20)) // 1 MB max
	if err != nil {
		Error(c, apperr.WithMessage(apperr.ErrInvalidParam, "failed to read request body"))
		return
	}

	// Extract Lark signature headers for HMAC-SHA256 verification.
	result, err := h.svc.HandleEvent(c.Request.Context(), body,
		c.GetHeader("X-Lark-Signature"),
		c.GetHeader("X-Lark-Request-Timestamp"),
		c.GetHeader("X-Lark-Request-Nonce"),
	)
	if err != nil {
		Error(c, apperr.WithMessage(apperr.ErrMissingParam, err.Error()))
		return
	}

	// Return raw result (important for URL verification challenge)
	c.JSON(200, result)
}

// GetConfig returns the current Lark bot configuration.
func (h *LarkBotHandler) GetConfig(c *gin.Context) {
	cfg, err := h.svc.GetConfig(c.Request.Context())
	if err != nil {
		Error(c, apperr.WithMessage(apperr.ErrExternalAPI, "failed to load Lark config: "+err.Error()))
		return
	}
	Success(c, cfg)
}

// UpdateConfig updates the Lark bot configuration.
func (h *LarkBotHandler) UpdateConfig(c *gin.Context) {
	var req service.LarkConfig
	if err := c.ShouldBindJSON(&req); err != nil {
		Error(c, apperr.WithMessage(apperr.ErrInvalidParam, err.Error()))
		return
	}

	if err := h.svc.UpdateConfig(c.Request.Context(), req); err != nil {
		Error(c, apperr.WithMessage(apperr.ErrExternalAPI, "failed to save Lark config: "+err.Error()))
		return
	}
	Success(c, gin.H{"message": "Lark bot configuration updated"})
}

// TestBotAPI tests connectivity to the Lark Bot API.
func (h *LarkBotHandler) TestBotAPI(c *gin.Context) {
	if err := h.svc.TestBotAPI(c.Request.Context()); err != nil {
		Error(c, apperr.WithMessage(apperr.ErrExternalAPI, "Lark bot API test failed: "+err.Error()))
		return
	}
	Success(c, gin.H{"message": "Lark bot API connection successful"})
}

// GetBotStatus returns the current bot connection status and diagnostics.
func (h *LarkBotHandler) GetBotStatus(c *gin.Context) {
	status, err := h.svc.GetBotStatus(c.Request.Context())
	if err != nil {
		Error(c, apperr.WithMessage(apperr.ErrExternalAPI, "failed to get bot status: "+err.Error()))
		return
	}
	Success(c, status)
}
