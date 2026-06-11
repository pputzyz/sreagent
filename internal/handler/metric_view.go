package handler

import (
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/sreagent/sreagent/internal/model"
	apperr "github.com/sreagent/sreagent/internal/pkg/errors"
	"github.com/sreagent/sreagent/internal/service"
)

// MetricViewHandler handles HTTP requests for metric views.
type MetricViewHandler struct {
	svc      *service.MetricViewService
	auditSvc *service.AuditLogService
	log      *zap.Logger
}

// NewMetricViewHandler creates a new MetricViewHandler.
func NewMetricViewHandler(svc *service.MetricViewService, logger *zap.Logger) *MetricViewHandler {
	return &MetricViewHandler{svc: svc, log: logger}
}

// SetAuditService injects the audit log service.
func (h *MetricViewHandler) SetAuditService(svc *service.AuditLogService) {
	h.auditSvc = svc
}

// --- Request types ---

// CreateMetricViewRequest is the request body for creating a metric view.
type CreateMetricViewRequest struct {
	Name    string                  `json:"name" binding:"required,max=200"`
	Configs *model.MetricViewConfig `json:"configs" binding:"required"`
}

// UpdateMetricViewRequest is the request body for updating a metric view.
type UpdateMetricViewRequest struct {
	Name    string                  `json:"name" binding:"required,max=200"`
	Configs *model.MetricViewConfig `json:"configs" binding:"required"`
}

// --- Handler methods ---

// List returns a paginated list of metric views.
// GET /metric-views?page=1&page_size=20
func (h *MetricViewHandler) List(c *gin.Context) {
	pq := GetPageQuery(c)

	var createdBy uint
	if createdByStr := c.Query("created_by"); createdByStr != "" {
		if id, err := parseUint64(createdByStr); err == nil {
			createdBy = uint(id)
		}
	}

	list, total, err := h.svc.List(c.Request.Context(), createdBy, pq.Page, pq.PageSize)
	if err != nil {
		Error(c, err)
		return
	}

	SuccessPage(c, list, total, pq.Page, pq.PageSize)
}

// Get returns a single metric view by ID.
// GET /metric-views/:id
func (h *MetricViewHandler) Get(c *gin.Context) {
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

// Create creates a new metric view.
// POST /metric-views
func (h *MetricViewHandler) Create(c *gin.Context) {
	var req CreateMetricViewRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		Error(c, apperr.WithMessage(apperr.ErrInvalidParam, err.Error()))
		return
	}

	userID := GetCurrentUserID(c)

	v := &model.MetricView{
		Name:        req.Name,
		ConfigsJSON: req.Configs,
		CreatedBy:   userID,
		UpdatedBy:   userID,
	}

	if err := h.svc.Create(c.Request.Context(), v); err != nil {
		Error(c, err)
		return
	}

	if h.auditSvc != nil {
		uid := userID
		h.auditSvc.Record(&model.AuditLog{
			UserID:       &uid,
			Action:       model.AuditActionCreate,
			ResourceType: "metric_view",
			ResourceID:   &v.ID,
			IP:           c.ClientIP(),
		})
	}

	Success(c, v)
}

// Update updates an existing metric view.
// PUT /metric-views/:id
func (h *MetricViewHandler) Update(c *gin.Context) {
	id, err := GetIDParam(c, "id")
	if err != nil {
		Error(c, err)
		return
	}

	existing, err := h.svc.GetByID(c.Request.Context(), id)
	if err != nil {
		Error(c, err)
		return
	}

	var req UpdateMetricViewRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		Error(c, apperr.WithMessage(apperr.ErrInvalidParam, err.Error()))
		return
	}

	userID := GetCurrentUserID(c)

	input := &model.MetricView{
		Name:        req.Name,
		ConfigsJSON: req.Configs,
		UpdatedBy:   userID,
	}

	if err := h.svc.Update(c.Request.Context(), existing, input); err != nil {
		Error(c, err)
		return
	}

	if h.auditSvc != nil {
		uid := userID
		rid := id
		h.auditSvc.Record(&model.AuditLog{
			UserID:       &uid,
			Action:       model.AuditActionUpdate,
			ResourceType: "metric_view",
			ResourceID:   &rid,
			IP:           c.ClientIP(),
		})
	}

	Success(c, nil)
}

// Delete deletes a metric view.
// DELETE /metric-views/:id
func (h *MetricViewHandler) Delete(c *gin.Context) {
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
		rid := id
		h.auditSvc.Record(&model.AuditLog{
			UserID:       &uid,
			Action:       model.AuditActionDelete,
			ResourceType: "metric_view",
			ResourceID:   &rid,
			IP:           c.ClientIP(),
		})
	}

	Success(c, nil)
}
