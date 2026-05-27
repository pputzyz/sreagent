package handler

import (
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/sreagent/sreagent/internal/model"
	apperr "github.com/sreagent/sreagent/internal/pkg/errors"
	"github.com/sreagent/sreagent/internal/service"
)

type TeamNotifyChannelHandler struct {
	svc    *service.TeamNotifyChannelService
	logger *zap.Logger
}

func NewTeamNotifyChannelHandler(svc *service.TeamNotifyChannelService, logger *zap.Logger) *TeamNotifyChannelHandler {
	return &TeamNotifyChannelHandler{svc: svc, logger: logger}
}

type teamNotifyChannelReq struct {
	TeamID    uint `json:"team_id" binding:"required"`
	MediaID   uint `json:"media_id" binding:"required"`
	IsDefault bool `json:"is_default"`
}

func (h *TeamNotifyChannelHandler) Create(c *gin.Context) {
	var req teamNotifyChannelReq
	if err := c.ShouldBindJSON(&req); err != nil {
		Error(c, apperr.WithMessage(apperr.ErrInvalidParam, err.Error()))
		return
	}
	ch := &model.TeamNotifyChannel{
		TeamID:    req.TeamID,
		MediaID:   req.MediaID,
		IsDefault: req.IsDefault,
	}
	if err := h.svc.Create(c.Request.Context(), ch); err != nil {
		Error(c, err)
		return
	}
	Success(c, ch)
}

func (h *TeamNotifyChannelHandler) List(c *gin.Context) {
	teamID, err := GetIDParam(c, "teamId")
	if err != nil {
		Error(c, err)
		return
	}
	channels, err := h.svc.ListByTeam(c.Request.Context(), teamID)
	if err != nil {
		Error(c, err)
		return
	}
	Success(c, channels)
}

func (h *TeamNotifyChannelHandler) Update(c *gin.Context) {
	id, err := GetIDParam(c, "id")
	if err != nil {
		Error(c, err)
		return
	}
	var req teamNotifyChannelReq
	if err := c.ShouldBindJSON(&req); err != nil {
		Error(c, apperr.WithMessage(apperr.ErrInvalidParam, err.Error()))
		return
	}
	ch := &model.TeamNotifyChannel{
		TeamID:    req.TeamID,
		MediaID:   req.MediaID,
		IsDefault: req.IsDefault,
	}
	ch.ID = id
	if err := h.svc.Update(c.Request.Context(), ch); err != nil {
		Error(c, err)
		return
	}
	Success(c, ch)
}

func (h *TeamNotifyChannelHandler) SetDefault(c *gin.Context) {
	id, err := GetIDParam(c, "id")
	if err != nil {
		Error(c, err)
		return
	}
	if err := h.svc.SetDefault(c.Request.Context(), id); err != nil {
		Error(c, err)
		return
	}
	Success(c, nil)
}

func (h *TeamNotifyChannelHandler) Delete(c *gin.Context) {
	id, err := GetIDParam(c, "id")
	if err != nil {
		Error(c, err)
		return
	}
	if err := h.svc.Delete(c.Request.Context(), id); err != nil {
		Error(c, err)
		return
	}
	Success(c, nil)
}
