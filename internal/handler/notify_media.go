package handler

import (
	"github.com/gin-gonic/gin"
	apperr "github.com/sreagent/sreagent/internal/pkg/errors"

	"github.com/sreagent/sreagent/internal/model"
	"github.com/sreagent/sreagent/internal/service"
)

// NotifyMediaHandler handles HTTP requests for notify medias.
type NotifyMediaHandler struct {
	svc *service.NotifyMediaService
}

// NewNotifyMediaHandler creates a new NotifyMediaHandler.
func NewNotifyMediaHandler(svc *service.NotifyMediaService) *NotifyMediaHandler {
	return &NotifyMediaHandler{svc: svc}
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
