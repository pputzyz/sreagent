package handler

import (
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	apperr "github.com/sreagent/sreagent/internal/pkg/errors"

	"github.com/sreagent/sreagent/internal/model"
	"github.com/sreagent/sreagent/internal/service"
)

// AlertChannelHandler handles HTTP requests for alert channels.
type AlertChannelHandler struct {
	svc *service.AlertChannelService
	log *zap.Logger
}

// NewAlertChannelHandler creates a new AlertChannelHandler.
func NewAlertChannelHandler(svc *service.AlertChannelService, logger ...*zap.Logger) *AlertChannelHandler {
	l := zap.NewNop()
	if len(logger) > 0 && logger[0] != nil {
		l = logger[0]
	}
	return &AlertChannelHandler{svc: svc, log: l}
}

// CreateAlertChannelRequest is the request body for creating an alert channel.
type CreateAlertChannelRequest struct {
	Name        string           `json:"name" binding:"required"`
	Description string           `json:"description"`
	MatchLabels model.JSONLabels `json:"match_labels"`
	Severities  string           `json:"severities"`
	MediaID     uint             `json:"media_id" binding:"required"`
	TemplateID  *uint            `json:"template_id"`
	ThrottleMin int              `json:"throttle_min"`
	IsEnabled   *bool            `json:"is_enabled"`
}

// UpdateAlertChannelRequest is the request body for updating an alert channel.
type UpdateAlertChannelRequest struct {
	Name        string           `json:"name" binding:"required"`
	Description string           `json:"description"`
	MatchLabels model.JSONLabels `json:"match_labels"`
	Severities  string           `json:"severities"`
	MediaID     uint             `json:"media_id" binding:"required"`
	TemplateID  *uint            `json:"template_id"`
	ThrottleMin int              `json:"throttle_min"`
	IsEnabled   *bool            `json:"is_enabled"`
}

// Create creates a new alert channel.
func (h *AlertChannelHandler) Create(c *gin.Context) {
	var req CreateAlertChannelRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		Error(c, apperr.WithMessage(apperr.ErrInvalidParam, err.Error()))
		return
	}

	isEnabled := true
	if req.IsEnabled != nil {
		isEnabled = *req.IsEnabled
	}

	userID := GetCurrentUserID(c)
	h.log.Info("alert channel create",
		zap.Uint("user_id", userID),
		zap.String("name", req.Name),
		zap.String("request_id", c.GetString("request_id")))

	ch := &model.AlertChannel{
		Name:        req.Name,
		Description: req.Description,
		MatchLabels: req.MatchLabels,
		Severities:  req.Severities,
		MediaID:     req.MediaID,
		TemplateID:  req.TemplateID,
		ThrottleMin: req.ThrottleMin,
		IsEnabled:   isEnabled,
		CreatedBy:   userID,
	}

	if err := h.svc.Create(c.Request.Context(), ch); err != nil {
		Error(c, err)
		return
	}

	Success(c, ch)
}

// Get returns a single alert channel by ID.
func (h *AlertChannelHandler) Get(c *gin.Context) {
	id, err := GetIDParam(c, "id")
	if err != nil {
		Error(c, err)
		return
	}

	ch, err := h.svc.GetByID(c.Request.Context(), id)
	if err != nil {
		Error(c, err)
		return
	}

	Success(c, ch)
}

// List returns a paginated list of alert channels.
func (h *AlertChannelHandler) List(c *gin.Context) {
	pq := GetPageQuery(c)

	list, total, err := h.svc.List(c.Request.Context(), pq.Page, pq.PageSize)
	if err != nil {
		Error(c, err)
		return
	}

	SuccessPage(c, list, total, pq.Page, pq.PageSize)
}

// Update updates an alert channel.
func (h *AlertChannelHandler) Update(c *gin.Context) {
	id, err := GetIDParam(c, "id")
	if err != nil {
		Error(c, err)
		return
	}

	var req UpdateAlertChannelRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		Error(c, apperr.WithMessage(apperr.ErrInvalidParam, err.Error()))
		return
	}

	isEnabled := true
	if req.IsEnabled != nil {
		isEnabled = *req.IsEnabled
	}

	h.log.Info("alert channel update",
		zap.Uint("user_id", GetCurrentUserID(c)),
		zap.Uint("channel_id", id),
		zap.String("name", req.Name),
		zap.String("request_id", c.GetString("request_id")))

	ch := &model.AlertChannel{
		Name:        req.Name,
		Description: req.Description,
		MatchLabels: req.MatchLabels,
		Severities:  req.Severities,
		MediaID:     req.MediaID,
		TemplateID:  req.TemplateID,
		ThrottleMin: req.ThrottleMin,
		IsEnabled:   isEnabled,
	}
	ch.ID = id

	if err := h.svc.Update(c.Request.Context(), ch); err != nil {
		Error(c, err)
		return
	}

	Success(c, ch)
}

// Delete deletes an alert channel.
func (h *AlertChannelHandler) Delete(c *gin.Context) {
	id, err := GetIDParam(c, "id")
	if err != nil {
		Error(c, err)
		return
	}

	h.log.Info("alert channel delete",
		zap.Uint("user_id", GetCurrentUserID(c)),
		zap.Uint("channel_id", id),
		zap.String("request_id", c.GetString("request_id")))

	if err := h.svc.Delete(c.Request.Context(), id); err != nil {
		Error(c, err)
		return
	}

	Success(c, nil)
}

// Test validates the channel config and sends a test notification.
func (h *AlertChannelHandler) Test(c *gin.Context) {
	id, err := GetIDParam(c, "id")
	if err != nil {
		Error(c, err)
		return
	}

	if err := h.svc.TestChannel(c.Request.Context(), id); err != nil {
		Error(c, err)
		return
	}

	Success(c, gin.H{"success": true, "message": "channel test passed"})
}
