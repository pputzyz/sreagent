package handler

import (
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/sreagent/sreagent/internal/model"
	apperr "github.com/sreagent/sreagent/internal/pkg/errors"
	"github.com/sreagent/sreagent/internal/service"
)

// SavedViewHandler handles HTTP requests for saved views.
type SavedViewHandler struct {
	svc      *service.SavedViewService
	auditSvc *service.AuditLogService
	log      *zap.Logger
}

// NewSavedViewHandler creates a new SavedViewHandler.
func NewSavedViewHandler(svc *service.SavedViewService, logger ...*zap.Logger) *SavedViewHandler {
	l := zap.NewNop()
	if len(logger) > 0 && logger[0] != nil {
		l = logger[0]
	}
	return &SavedViewHandler{svc: svc, log: l}
}

// SetAuditService injects the audit log service.
func (h *SavedViewHandler) SetAuditService(svc *service.AuditLogService) {
	h.auditSvc = svc
}

// --- Request types ---

// CreateSavedViewRequest is the request body for creating a saved view.
type CreateSavedViewRequest struct {
	Name         string `json:"name" binding:"required,max=200"`
	Description  string `json:"description" binding:"max=500"`
	Tab          string `json:"tab" binding:"required,oneof=metrics logs"`
	DatasourceID uint   `json:"datasource_id"`
	Expression   string `json:"expression" binding:"required"`
	QueryConfig  string `json:"query_config"`
	IsPublic     bool   `json:"is_public"`
}

// UpdateSavedViewRequest is the request body for updating a saved view.
type UpdateSavedViewRequest struct {
	Name         string `json:"name" binding:"required,max=200"`
	Description  string `json:"description" binding:"max=500"`
	Tab          string `json:"tab" binding:"required,oneof=metrics logs"`
	DatasourceID uint   `json:"datasource_id"`
	Expression   string `json:"expression" binding:"required"`
	QueryConfig  string `json:"query_config"`
	IsPublic     bool   `json:"is_public"`
}

// --- Handler methods ---

// List returns a paginated list of saved views.
// GET /saved-views?tab=metrics&is_public=true&page=1&page_size=20
func (h *SavedViewHandler) List(c *gin.Context) {
	pq := GetPageQuery(c)

	q := service.SavedViewListQuery{
		Tab: c.Query("tab"),
	}
	if createdByStr := c.Query("created_by"); createdByStr != "" {
		if id, err := parseUint64(createdByStr); err == nil {
			q.CreatedBy = uint(id)
		}
	}
	if isPublicStr := c.Query("is_public"); isPublicStr != "" {
		b := isPublicStr == "true" || isPublicStr == "1"
		q.IsPublic = &b
	}

	list, total, err := h.svc.List(c.Request.Context(), q, pq.Page, pq.PageSize)
	if err != nil {
		Error(c, err)
		return
	}

	SuccessPage(c, list, total, pq.Page, pq.PageSize)
}

// Get returns a single saved view by ID.
// GET /saved-views/:id
func (h *SavedViewHandler) Get(c *gin.Context) {
	id, err := GetIDParam(c, "id")
	if err != nil {
		Error(c, err)
		return
	}

	v, err := h.svc.GetByID(c.Request.Context(), id)
	if err != nil {
		Error(c, err)
		return
	}

	Success(c, v)
}

// Create creates a new saved view.
// POST /saved-views
func (h *SavedViewHandler) Create(c *gin.Context) {
	var req CreateSavedViewRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		Error(c, apperr.WithMessage(apperr.ErrInvalidParam, err.Error()))
		return
	}

	userID := GetCurrentUserID(c)

	v := &model.SavedView{
		Name:         req.Name,
		Description:  req.Description,
		Tab:          req.Tab,
		DatasourceID: req.DatasourceID,
		Expression:   req.Expression,
		QueryConfig:  req.QueryConfig,
		IsPublic:     req.IsPublic,
		CreatedBy:    userID,
		UpdatedBy:    userID,
	}

	if err := h.svc.Create(c.Request.Context(), v); err != nil {
		Error(c, err)
		return
	}

	if h.auditSvc != nil {
		uid := GetCurrentUserID(c)
		h.auditSvc.Record(&model.AuditLog{
			UserID:       &uid,
			Action:       model.AuditActionCreate,
			ResourceType: "saved_view",
			ResourceID:   &v.ID,
			IP:           c.ClientIP(),
		})
	}

	Success(c, v)
}

// Update updates an existing saved view.
// PUT /saved-views/:id
func (h *SavedViewHandler) Update(c *gin.Context) {
	id, err := GetIDParam(c, "id")
	if err != nil {
		Error(c, err)
		return
	}

	var req UpdateSavedViewRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		Error(c, apperr.WithMessage(apperr.ErrInvalidParam, err.Error()))
		return
	}

	userID := GetCurrentUserID(c)

	v := &model.SavedView{
		Name:         req.Name,
		Description:  req.Description,
		Tab:          req.Tab,
		DatasourceID: req.DatasourceID,
		Expression:   req.Expression,
		QueryConfig:  req.QueryConfig,
		IsPublic:     req.IsPublic,
		UpdatedBy:    userID,
	}
	v.ID = id

	if err := h.svc.Update(c.Request.Context(), v); err != nil {
		Error(c, err)
		return
	}

	if h.auditSvc != nil {
		uid := GetCurrentUserID(c)
		h.auditSvc.Record(&model.AuditLog{
			UserID:       &uid,
			Action:       model.AuditActionUpdate,
			ResourceType: "saved_view",
			ResourceID:   &id,
			IP:           c.ClientIP(),
		})
	}

	Success(c, v)
}

// Delete deletes a saved view.
// DELETE /saved-views/:id
func (h *SavedViewHandler) Delete(c *gin.Context) {
	id, err := GetIDParam(c, "id")
	if err != nil {
		Error(c, err)
		return
	}

	if err := h.svc.Delete(c.Request.Context(), id); err != nil {
		Error(c, err)
		return
	}

	if h.auditSvc != nil {
		uid := GetCurrentUserID(c)
		h.auditSvc.Record(&model.AuditLog{
			UserID:       &uid,
			Action:       model.AuditActionDelete,
			ResourceType: "saved_view",
			ResourceID:   &id,
			IP:           c.ClientIP(),
		})
	}

	Success(c, nil)
}

// Copy clones an existing saved view.
// POST /saved-views/:id/copy
func (h *SavedViewHandler) Copy(c *gin.Context) {
	id, err := GetIDParam(c, "id")
	if err != nil {
		Error(c, err)
		return
	}

	userID := GetCurrentUserID(c)

	v, err := h.svc.Copy(c.Request.Context(), id, userID)
	if err != nil {
		Error(c, err)
		return
	}

	if h.auditSvc != nil {
		uid := GetCurrentUserID(c)
		h.auditSvc.Record(&model.AuditLog{
			UserID:       &uid,
			Action:       model.AuditActionCreate,
			ResourceType: "saved_view",
			ResourceID:   &v.ID,
			IP:           c.ClientIP(),
		})
	}

	Success(c, v)
}
