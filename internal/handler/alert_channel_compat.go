package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/sreagent/sreagent/pkg/types"
)

// AlertChannelCompatHandler provides backward-compatible endpoints for
// the old /api/v1/alert-channels/* routes. All operations log deprecation
// warnings. Create/update/delete return HTTP 410 Gone.
// This handler will be removed in v4.44.0.
//
// Deprecated: Use DispatchPolicy directly.
type AlertChannelCompatHandler struct {
	logger *zap.Logger
}

func NewAlertChannelCompatHandler(logger *zap.Logger) *AlertChannelCompatHandler {
	return &AlertChannelCompatHandler{logger: logger}
}

func (h *AlertChannelCompatHandler) List(c *gin.Context) {
	h.logger.Warn("deprecated endpoint called: GET /alert-channels, use /dispatch-policies instead")
	Success(c, gin.H{"data": []interface{}{}, "total": 0, "message": "AlertChannel is deprecated. Use DispatchPolicy. Data migration completed in v4.42.0."})
}

func (h *AlertChannelCompatHandler) Get(c *gin.Context) {
	h.logger.Warn("deprecated endpoint called: GET /alert-channels/:id")
	Success(c, gin.H{"message": "AlertChannel is deprecated. Use DispatchPolicy."})
}

func (h *AlertChannelCompatHandler) Create(c *gin.Context) {
	h.logger.Warn("deprecated endpoint called: POST /alert-channels")
	c.JSON(http.StatusGone, types.Response{
		Code:    10002,
		Message: "AlertChannel creation is deprecated. Use DispatchPolicy instead.",
	})
}

func (h *AlertChannelCompatHandler) Update(c *gin.Context) {
	h.logger.Warn("deprecated endpoint called: PUT /alert-channels/:id")
	c.JSON(http.StatusGone, types.Response{
		Code:    10002,
		Message: "AlertChannel update is deprecated. Use DispatchPolicy instead.",
	})
}

func (h *AlertChannelCompatHandler) Delete(c *gin.Context) {
	h.logger.Warn("deprecated endpoint called: DELETE /alert-channels/:id")
	c.JSON(http.StatusGone, types.Response{
		Code:    10002,
		Message: "AlertChannel deletion is deprecated. Use DispatchPolicy instead.",
	})
}

func (h *AlertChannelCompatHandler) Test(c *gin.Context) {
	h.logger.Warn("deprecated endpoint called: POST /alert-channels/:id/test")
	c.JSON(http.StatusGone, types.Response{
		Code:    10002,
		Message: "AlertChannel test is deprecated. Use DispatchPolicy instead.",
	})
}
