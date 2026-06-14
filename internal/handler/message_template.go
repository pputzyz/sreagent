package handler

import (
	"github.com/gin-gonic/gin"
	apperr "github.com/sreagent/sreagent/internal/pkg/errors"
	"go.uber.org/zap"

	"github.com/sreagent/sreagent/internal/model"
	"github.com/sreagent/sreagent/internal/service"
)

// MessageTemplateHandler handles HTTP requests for message templates.
type MessageTemplateHandler struct {
	svc *service.MessageTemplateService
	log *zap.Logger
}

// NewMessageTemplateHandler creates a new MessageTemplateHandler.
func NewMessageTemplateHandler(svc *service.MessageTemplateService, logger ...*zap.Logger) *MessageTemplateHandler {
	l := zap.NewNop()
	if len(logger) > 0 && logger[0] != nil {
		l = logger[0]
	}
	return &MessageTemplateHandler{svc: svc, log: l}
}

// CreateMessageTemplateRequest is the request body for creating a message template.
type CreateMessageTemplateRequest struct {
	Name        string `json:"name" binding:"required"`
	Description string `json:"description"`
	Content     string `json:"content" binding:"required"`
	ContentEN   string `json:"content_en"` // optional English variant
	Type        string `json:"type"`       // text, html, markdown, lark_card
}

// UpdateMessageTemplateRequest is the request body for updating a message template.
type UpdateMessageTemplateRequest struct {
	Name        string `json:"name" binding:"required"`
	Description string `json:"description"`
	Content     string `json:"content" binding:"required"`
	ContentEN   string `json:"content_en"`
	Type        string `json:"type"`
}

// PreviewMessageTemplateRequest is the request body for previewing a template rendering.
type PreviewMessageTemplateRequest struct {
	Content string `json:"content" binding:"required"`
}

// Create creates a new message template.
func (h *MessageTemplateHandler) Create(c *gin.Context) {
	var req CreateMessageTemplateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		Error(c, apperr.WithMessage(apperr.ErrInvalidParam, err.Error()))
		return
	}

	tmplType := req.Type
	if tmplType == "" {
		tmplType = "text"
	}

	h.log.Info("message template create",
		zap.Uint("user_id", GetCurrentUserID(c)),
		zap.String("name", req.Name),
		zap.String("request_id", c.GetString("request_id")))

	tmpl := &model.MessageTemplate{
		Name:        req.Name,
		Description: req.Description,
		Content:     req.Content,
		ContentEN:   req.ContentEN,
		Type:        tmplType,
	}

	if err := h.svc.Create(c.Request.Context(), tmpl); err != nil {
		Error(c, err)
		return
	}

	Success(c, tmpl)
}

// Get returns a single message template by ID.
func (h *MessageTemplateHandler) Get(c *gin.Context) {
	id, err := GetIDParam(c, "id")
	if err != nil {
		Error(c, err)
		return
	}

	tmpl, err := h.svc.GetByID(c.Request.Context(), id)
	if err != nil {
		Error(c, err)
		return
	}

	Success(c, tmpl)
}

// List returns a paginated list of message templates.
func (h *MessageTemplateHandler) List(c *gin.Context) {
	pq := GetPageQuery(c)

	list, total, err := h.svc.List(c.Request.Context(), pq.Page, pq.PageSize)
	if err != nil {
		Error(c, err)
		return
	}

	SuccessPage(c, list, total, pq.Page, pq.PageSize)
}

// Update updates a message template.
func (h *MessageTemplateHandler) Update(c *gin.Context) {
	id, err := GetIDParam(c, "id")
	if err != nil {
		Error(c, err)
		return
	}

	var req UpdateMessageTemplateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		Error(c, apperr.WithMessage(apperr.ErrInvalidParam, err.Error()))
		return
	}

	tmplType := req.Type
	if tmplType == "" {
		tmplType = "text"
	}

	h.log.Info("message template update",
		zap.Uint("user_id", GetCurrentUserID(c)),
		zap.Uint("template_id", id),
		zap.String("name", req.Name),
		zap.String("request_id", c.GetString("request_id")))

	tmpl := &model.MessageTemplate{
		Name:        req.Name,
		Description: req.Description,
		Content:     req.Content,
		ContentEN:   req.ContentEN,
		Type:        tmplType,
	}
	tmpl.ID = id

	if err := h.svc.Update(c.Request.Context(), tmpl); err != nil {
		Error(c, err)
		return
	}

	Success(c, tmpl)
}

// Delete deletes a message template.
func (h *MessageTemplateHandler) Delete(c *gin.Context) {
	id, err := GetIDParam(c, "id")
	if err != nil {
		Error(c, err)
		return
	}

	h.log.Info("message template delete",
		zap.Uint("user_id", GetCurrentUserID(c)),
		zap.Uint("template_id", id),
		zap.String("request_id", c.GetString("request_id")))

	if err := h.svc.Delete(c.Request.Context(), id); err != nil {
		Error(c, err)
		return
	}

	Success(c, nil)
}

// Preview renders a template with sample data and returns the result.
func (h *MessageTemplateHandler) Preview(c *gin.Context) {
	var req PreviewMessageTemplateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		Error(c, apperr.WithMessage(apperr.ErrInvalidParam, err.Error()))
		return
	}

	rendered, err := h.svc.RenderPreview(c.Request.Context(), req.Content)
	if err != nil {
		Error(c, err)
		return
	}

	Success(c, gin.H{"rendered": rendered})
}
