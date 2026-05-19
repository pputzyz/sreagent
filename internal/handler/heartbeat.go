package handler

import (
	"github.com/gin-gonic/gin"
	apperr "github.com/sreagent/sreagent/internal/pkg/errors"

	"github.com/sreagent/sreagent/internal/service"
)

// HeartbeatHandler handles the public heartbeat ping endpoint.
// POST /heartbeat/:token — no authentication required; the token itself authenticates the source.
type HeartbeatHandler struct {
	ruleSvc *service.AlertRuleService
}

// NewHeartbeatHandler creates a HeartbeatHandler.
func NewHeartbeatHandler(ruleSvc *service.AlertRuleService) *HeartbeatHandler {
	return &HeartbeatHandler{ruleSvc: ruleSvc}
}

// Ping records a heartbeat ping for the rule identified by the token.
// Returns 200 OK on success, 404 if the token is unknown.
func (h *HeartbeatHandler) Ping(c *gin.Context) {
	token := c.Param("token")
	if token == "" {
		Error(c, apperr.WithMessage(apperr.ErrInvalidParam, "token is required"))
		return
	}

	if err := h.ruleSvc.RecordHeartbeatPing(c.Request.Context(), token); err != nil {
		Error(c, err)
		return
	}

	Success(c, gin.H{"status": "ok"})
}
