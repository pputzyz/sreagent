package handler

import (
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	apperr "github.com/sreagent/sreagent/internal/pkg/errors"

	"github.com/sreagent/sreagent/internal/model"
	"github.com/sreagent/sreagent/internal/service"
)

// NotifyMediaHandler handles HTTP requests for notify medias.
type NotifyMediaHandler struct {
	svc *service.NotifyMediaService
	log *zap.Logger
}

// NewNotifyMediaHandler creates a new NotifyMediaHandler.
func NewNotifyMediaHandler(svc *service.NotifyMediaService, logger ...*zap.Logger) *NotifyMediaHandler {
	l := zap.NewNop()
	if len(logger) > 0 && logger[0] != nil {
		l = logger[0]
	}
	return &NotifyMediaHandler{svc: svc, log: l}
}

// CreateNotifyMediaRequest is the request body for creating a notify media.
type CreateNotifyMediaRequest struct {
	Name        string                `json:"name" binding:"required"`
	Type        model.NotifyMediaType `json:"type" binding:"required"`
	Description string                `json:"description"`
	IsEnabled   *bool                 `json:"is_enabled"`
	Config      string                `json:"config" binding:"required"`
	Variables   string                `json:"variables"`
}

// UpdateNotifyMediaRequest is the request body for updating a notify media.
type UpdateNotifyMediaRequest struct {
	Name        string                `json:"name" binding:"required"`
	Type        model.NotifyMediaType `json:"type" binding:"required"`
	Description string                `json:"description"`
	IsEnabled   *bool                 `json:"is_enabled"`
	Config      string                `json:"config"`
	Variables   string                `json:"variables"`
}

// Create creates a new notify media.
func (h *NotifyMediaHandler) Create(c *gin.Context) {
	var req CreateNotifyMediaRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		Error(c, apperr.WithMessage(apperr.ErrInvalidParam, err.Error()))
		return
	}

	isEnabled := true
	if req.IsEnabled != nil {
		isEnabled = *req.IsEnabled
	}

	h.log.Info("notify media create",
		zap.Uint("user_id", GetCurrentUserID(c)),
		zap.String("name", req.Name),
		zap.String("type", string(req.Type)),
		zap.String("request_id", c.GetString("request_id")))

	media := &model.NotifyMedia{
		Name:        req.Name,
		Type:        req.Type,
		Description: req.Description,
		IsEnabled:   isEnabled,
		Config:      req.Config,
		Variables:   req.Variables,
	}

	if err := h.svc.Create(c.Request.Context(), media); err != nil {
		Error(c, err)
		return
	}

	Success(c, media)
}

// Get returns a single notify media by ID.
func (h *NotifyMediaHandler) Get(c *gin.Context) {
	id, err := GetIDParam(c, "id")
	if err != nil {
		Error(c, err)
		return
	}

	media, err := h.svc.GetByID(c.Request.Context(), id)
	if err != nil {
		Error(c, err)
		return
	}

	Success(c, media)
}

// List returns a paginated list of notify medias.
func (h *NotifyMediaHandler) List(c *gin.Context) {
	pq := GetPageQuery(c)

	list, total, err := h.svc.List(c.Request.Context(), pq.Page, pq.PageSize)
	if err != nil {
		Error(c, err)
		return
	}

	SuccessPage(c, list, total, pq.Page, pq.PageSize)
}

// Update updates a notify media.
func (h *NotifyMediaHandler) Update(c *gin.Context) {
	id, err := GetIDParam(c, "id")
	if err != nil {
		Error(c, err)
		return
	}

	var req UpdateNotifyMediaRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		Error(c, apperr.WithMessage(apperr.ErrInvalidParam, err.Error()))
		return
	}

	isEnabled := true
	if req.IsEnabled != nil {
		isEnabled = *req.IsEnabled
	}

	h.log.Info("notify media update",
		zap.Uint("user_id", GetCurrentUserID(c)),
		zap.Uint("media_id", id),
		zap.String("name", req.Name),
		zap.String("request_id", c.GetString("request_id")))

	media := &model.NotifyMedia{
		Name:        req.Name,
		Type:        req.Type,
		Description: req.Description,
		IsEnabled:   isEnabled,
		Config:      req.Config,
		Variables:   req.Variables,
	}
	media.ID = id

	if err := h.svc.Update(c.Request.Context(), media); err != nil {
		Error(c, err)
		return
	}

	Success(c, media)
}

// Delete deletes a notify media.
func (h *NotifyMediaHandler) Delete(c *gin.Context) {
	id, err := GetIDParam(c, "id")
	if err != nil {
		Error(c, err)
		return
	}

	h.log.Info("notify media delete",
		zap.Uint("user_id", GetCurrentUserID(c)),
		zap.Uint("media_id", id),
		zap.String("request_id", c.GetString("request_id")))

	if err := h.svc.Delete(c.Request.Context(), id); err != nil {
		Error(c, err)
		return
	}

	Success(c, nil)
}

// Test sends a test notification via a media.
func (h *NotifyMediaHandler) Test(c *gin.Context) {
	id, err := GetIDParam(c, "id")
	if err != nil {
		Error(c, err)
		return
	}

	if err := h.svc.TestMedia(c.Request.Context(), id); err != nil {
		Error(c, err)
		return
	}

	Success(c, gin.H{"message": "test notification sent"})
}
