package handler

import (
	"github.com/gin-gonic/gin"

	apperr "github.com/sreagent/sreagent/internal/pkg/errors"
	"github.com/sreagent/sreagent/internal/service"
)

type UserPreferenceHandler struct {
	svc *service.UserPreferenceService
}

func NewUserPreferenceHandler(svc *service.UserPreferenceService) *UserPreferenceHandler {
	return &UserPreferenceHandler{svc: svc}
}

// Get returns the current user's preferences.
func (h *UserPreferenceHandler) Get(c *gin.Context) {
	userID, ok := GetCurrentUserIDOK(c)
	if !ok {
		Error(c, apperr.WithMessage(apperr.ErrUnauthorized, "unauthorized"))
		return
	}

	pref, err := h.svc.Get(c.Request.Context(), userID)
	if err != nil {
		Error(c, err)
		return
	}
	Success(c, pref)
}

// Update creates or updates the current user's preferences.
func (h *UserPreferenceHandler) Update(c *gin.Context) {
	userID, ok := GetCurrentUserIDOK(c)
	if !ok {
		Error(c, apperr.WithMessage(apperr.ErrUnauthorized, "unauthorized"))
		return
	}

	var req struct {
		Theme                  *string `json:"theme"`
		Language               *string `json:"language"`
		Timezone               *string `json:"timezone"`
		DefaultTimeRange       *string `json:"default_time_range"`
		NotificationSeverities *string `json:"notification_severities"`
		AIChatMode             *string `json:"ai_chat_mode"`
		AccentColor            *string `json:"accent_color"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		Error(c, apperr.WithMessage(apperr.ErrInvalidParam, err.Error()))
		return
	}

	// Get current prefs, then apply partial updates
	pref, err := h.svc.Get(c.Request.Context(), userID)
	if err != nil {
		Error(c, err)
		return
	}

	if req.Theme != nil {
		pref.Theme = *req.Theme
	}
	if req.Language != nil {
		pref.Language = *req.Language
	}
	if req.Timezone != nil {
		pref.Timezone = *req.Timezone
	}
	if req.DefaultTimeRange != nil {
		pref.DefaultTimeRange = *req.DefaultTimeRange
	}
	if req.NotificationSeverities != nil {
		pref.NotificationSeverities = *req.NotificationSeverities
	}
	if req.AIChatMode != nil {
		pref.AIChatMode = *req.AIChatMode
	}
	if req.AccentColor != nil {
		pref.AccentColor = *req.AccentColor
	}

	if err := h.svc.Update(c.Request.Context(), pref); err != nil {
		Error(c, err)
		return
	}
	Success(c, pref)
}
