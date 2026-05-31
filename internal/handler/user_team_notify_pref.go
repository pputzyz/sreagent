package handler

import (
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/sreagent/sreagent/internal/model"
	apperr "github.com/sreagent/sreagent/internal/pkg/errors"
	"github.com/sreagent/sreagent/internal/service"
)

type UserTeamNotifyPrefHandler struct {
	svc    *service.UserTeamNotifyPrefService
	logger *zap.Logger
}

func NewUserTeamNotifyPrefHandler(svc *service.UserTeamNotifyPrefService, logger *zap.Logger) *UserTeamNotifyPrefHandler {
	return &UserTeamNotifyPrefHandler{svc: svc, logger: logger}
}

type userTeamNotifyPrefReq struct {
	TeamID  uint `json:"team_id" binding:"required"`
	MediaID uint `json:"media_id" binding:"required"`
	IsMuted bool `json:"is_muted"`
}

func (h *UserTeamNotifyPrefHandler) Upsert(c *gin.Context) {
	userID, ok := GetCurrentUserIDOK(c)
	if !ok {
		Error(c, apperr.ErrUnauthorized)
		return
	}
	var req userTeamNotifyPrefReq
	if err := c.ShouldBindJSON(&req); err != nil {
		Error(c, apperr.WithMessage(apperr.ErrInvalidParam, err.Error()))
		return
	}
	pref := &model.UserTeamNotifyPref{
		UserID:  userID,
		TeamID:  req.TeamID,
		MediaID: req.MediaID,
		IsMuted: req.IsMuted,
	}
	if err := h.svc.Upsert(c.Request.Context(), pref); err != nil {
		Error(c, err)
		return
	}
	Success(c, pref)
}

func (h *UserTeamNotifyPrefHandler) List(c *gin.Context) {
	userID, ok := GetCurrentUserIDOK(c)
	if !ok {
		Error(c, apperr.ErrUnauthorized)
		return
	}
	prefs, err := h.svc.ListByUser(c.Request.Context(), userID)
	if err != nil {
		Error(c, err)
		return
	}
	Success(c, prefs)
}

func (h *UserTeamNotifyPrefHandler) Delete(c *gin.Context) {
	userID, ok := GetCurrentUserIDOK(c)
	if !ok {
		Error(c, apperr.ErrUnauthorized)
		return
	}
	id, err := GetIDParam(c, "id")
	if err != nil {
		Error(c, err)
		return
	}
	if err := h.svc.Delete(c.Request.Context(), id, userID); err != nil {
		Error(c, err)
		return
	}
	Success(c, nil)
}
