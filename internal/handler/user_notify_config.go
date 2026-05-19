package handler

import (
	"github.com/gin-gonic/gin"
	apperr "github.com/sreagent/sreagent/internal/pkg/errors"

	"github.com/sreagent/sreagent/internal/model"
	"github.com/sreagent/sreagent/internal/service"
)

type UserNotifyConfigHandler struct {
	svc *service.UserNotifyConfigService
}

func NewUserNotifyConfigHandler(svc *service.UserNotifyConfigService) *UserNotifyConfigHandler {
	return &UserNotifyConfigHandler{svc: svc}
}

// List returns all notify configs for the current user.
func (h *UserNotifyConfigHandler) List(c *gin.Context) {
	userID := GetCurrentUserID(c)
	cfgs, err := h.svc.ListByUserID(c.Request.Context(), userID)
	if err != nil {
		Error(c, err)
		return
	}
	Success(c, cfgs)
}

// Upsert creates or updates one notify config for the current user (keyed by media_type).
func (h *UserNotifyConfigHandler) Upsert(c *gin.Context) {
	userID := GetCurrentUserID(c)

	var req struct {
		MediaType string `json:"media_type" binding:"required"`
		Config    string `json:"config"`
		IsEnabled *bool  `json:"is_enabled"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		Error(c, apperr.WithMessage(apperr.ErrInvalidParam, err.Error()))
		return
	}

	isEnabled := true
	if req.IsEnabled != nil {
		isEnabled = *req.IsEnabled
	}

	cfg := &model.UserNotifyConfig{
		UserID:    userID,
		MediaType: req.MediaType,
		Config:    req.Config,
		IsEnabled: isEnabled,
	}
	if err := h.svc.Upsert(c.Request.Context(), cfg); err != nil {
		Error(c, err)
		return
	}
	Success(c, cfg)
}

// DeleteByMediaType removes a specific media type config for the current user.
func (h *UserNotifyConfigHandler) DeleteByMediaType(c *gin.Context) {
	userID := GetCurrentUserID(c)
	mediaType := c.Param("mediaType")
	if mediaType == "" {
		Error(c, apperr.WithMessage(apperr.ErrInvalidParam, "media_type is required"))
		return
	}
	if err := h.svc.DeleteByMediaType(c.Request.Context(), userID, mediaType); err != nil {
		Error(c, err)
		return
	}
	Success(c, nil)
}
