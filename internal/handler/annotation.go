package handler

import (
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/sreagent/sreagent/internal/model"
	apperr "github.com/sreagent/sreagent/internal/pkg/errors"
	"github.com/sreagent/sreagent/internal/service"
)

// AnnotationHandler handles HTTP requests for dashboard annotations.
type AnnotationHandler struct {
	svc      *service.AnnotationService
	auditSvc *service.AuditLogService
	log      *zap.Logger
}

// NewAnnotationHandler creates a new AnnotationHandler.
func NewAnnotationHandler(svc *service.AnnotationService, logger ...*zap.Logger) *AnnotationHandler {
	l := zap.NewNop()
	if len(logger) > 0 && logger[0] != nil {
		l = logger[0]
	}
	return &AnnotationHandler{svc: svc, log: l}
}

// SetAuditService injects the audit log service.
func (h *AnnotationHandler) SetAuditService(svc *service.AuditLogService) {
	h.auditSvc = svc
}

// --- Request types ---

// CreateAnnotationRequest is the request body for creating an annotation.
type CreateAnnotationRequest struct {
	DashboardID uint       `json:"dashboard_id" binding:"required"`
	Time        time.Time  `json:"time" binding:"required"`
	EndTime     *time.Time `json:"end_time"`
	Text        string     `json:"text"`
	Tags        JSONMap    `json:"tags"`
	Source      string     `json:"source"`
}

// UpdateAnnotationRequest is the request body for updating an annotation.
type UpdateAnnotationRequest struct {
	Text    string     `json:"text"`
	Tags    JSONMap    `json:"tags"`
	Time    *time.Time `json:"time"`
	EndTime *time.Time `json:"end_time"`
}

// BatchCreateAnnotationRequest is the request body for batch creating annotations.
type BatchCreateAnnotationRequest struct {
	Annotations []CreateAnnotationRequest `json:"annotations" binding:"required,min=1"`
}

// JSONMap is a convenience alias for map[string]string used in request binding.
type JSONMap map[string]string

// --- Handler methods ---

// List returns annotations, optionally filtered by dashboard and time range.
// GET /annotations?dashboard_id=X&from=T1&to=T2&page=1&page_size=20
func (h *AnnotationHandler) List(c *gin.Context) {
	var dashboardID uint
	if dashboardIDStr := c.Query("dashboard_id"); dashboardIDStr != "" {
		id, err := parseUint64(dashboardIDStr)
		if err != nil {
			Error(c, apperr.WithMessage(apperr.ErrInvalidParam, "invalid dashboard_id"))
			return
		}
		dashboardID = uint(id)
	}

	var from, to time.Time
	if fromStr := c.Query("from"); fromStr != "" {
		ts, err := parseUint64(fromStr)
		if err == nil {
			from = time.Unix(int64(ts), 0)
		}
	}
	if toStr := c.Query("to"); toStr != "" {
		ts, err := parseUint64(toStr)
		if err == nil {
			to = time.Unix(int64(ts), 0)
		}
	}

	// Pagination
	page, _ := strconv.ParseUint(c.DefaultQuery("page", "1"), 10, 32)
	pageSize, _ := strconv.ParseUint(c.DefaultQuery("page_size", "20"), 10, 32)
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	annotations, total, err := h.svc.List(c.Request.Context(), dashboardID, from, to, uint(page), uint(pageSize))
	if err != nil {
		Error(c, err)
		return
	}

	SuccessPage(c, annotations, total, int(page), int(pageSize))
}

// Create creates a new annotation.
// POST /annotations
func (h *AnnotationHandler) Create(c *gin.Context) {
	var req CreateAnnotationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		Error(c, apperr.WithMessage(apperr.ErrInvalidParam, err.Error()))
		return
	}

	source := req.Source
	if source == "" {
		source = "user"
	}

	var tags model.JSONLabels
	if req.Tags != nil {
		tags = model.JSONLabels(req.Tags)
	}

	userID := GetCurrentUserID(c)
	ann := &model.Annotation{
		DashboardID: req.DashboardID,
		Time:        req.Time,
		EndTime:     req.EndTime,
		Text:        req.Text,
		Tags:        tags,
		Source:      source,
		CreatedBy:   userID,
	}

	if err := h.svc.Create(c.Request.Context(), ann); err != nil {
		Error(c, err)
		return
	}

	if h.auditSvc != nil {
		uid := GetCurrentUserID(c)
		h.auditSvc.Record(&model.AuditLog{
			UserID:       &uid,
			Action:       model.AuditActionCreate,
			ResourceType: "annotation",
			ResourceID:   &ann.ID,
			IP:           c.ClientIP(),
		})
	}

	Success(c, ann)
}

// Update updates an existing annotation.
// PUT /annotations/:id
func (h *AnnotationHandler) Update(c *gin.Context) {
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

	var req UpdateAnnotationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		Error(c, apperr.WithMessage(apperr.ErrInvalidParam, err.Error()))
		return
	}

	if req.Text != "" {
		existing.Text = req.Text
	}
	if req.Tags != nil {
		existing.Tags = model.JSONLabels(req.Tags)
	}
	if req.Time != nil {
		existing.Time = *req.Time
	}
	if req.EndTime != nil {
		existing.EndTime = req.EndTime
	}

	if err := h.svc.Update(c.Request.Context(), existing); err != nil {
		Error(c, err)
		return
	}

	if h.auditSvc != nil {
		uid := GetCurrentUserID(c)
		h.auditSvc.Record(&model.AuditLog{
			UserID:       &uid,
			Action:       model.AuditActionUpdate,
			ResourceType: "annotation",
			ResourceID:   &id,
			IP:           c.ClientIP(),
		})
	}

	Success(c, existing)
}

// Delete deletes an annotation.
// DELETE /annotations/:id
func (h *AnnotationHandler) Delete(c *gin.Context) {
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
			ResourceType: "annotation",
			ResourceID:   &id,
			IP:           c.ClientIP(),
		})
	}

	Success(c, nil)
}

// BatchCreate creates multiple annotations at once.
// POST /annotations/batch
func (h *AnnotationHandler) BatchCreate(c *gin.Context) {
	var req BatchCreateAnnotationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		Error(c, apperr.WithMessage(apperr.ErrInvalidParam, err.Error()))
		return
	}

	userID := GetCurrentUserID(c)
	annotations := make([]model.Annotation, 0, len(req.Annotations))
	for _, a := range req.Annotations {
		source := a.Source
		if source == "" {
			source = "user"
		}
		var tags model.JSONLabels
		if a.Tags != nil {
			tags = model.JSONLabels(a.Tags)
		}
		annotations = append(annotations, model.Annotation{
			DashboardID: a.DashboardID,
			Time:        a.Time,
			EndTime:     a.EndTime,
			Text:        a.Text,
			Tags:        tags,
			Source:      source,
			CreatedBy:   userID,
		})
	}

	if err := h.svc.BatchCreate(c.Request.Context(), annotations); err != nil {
		Error(c, err)
		return
	}

	Success(c, nil)
}

// parseUint64 is a helper to parse a string to uint64.
func parseUint64(s string) (uint64, error) {
	var n uint64
	for _, c := range s {
		if c < '0' || c > '9' {
			return 0, apperr.ErrInvalidParam
		}
		n = n*10 + uint64(c-'0')
	}
	return n, nil
}
