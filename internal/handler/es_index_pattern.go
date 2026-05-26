package handler

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/sreagent/sreagent/internal/model"
	apperr "github.com/sreagent/sreagent/internal/pkg/errors"
	"github.com/sreagent/sreagent/internal/service"
)

// ESIndexPatternHandler handles HTTP requests for ES index patterns.
type ESIndexPatternHandler struct {
	svc      *service.ESIndexPatternService
	auditSvc *service.AuditLogService
	log      *zap.Logger
}

// NewESIndexPatternHandler creates a new ESIndexPatternHandler.
func NewESIndexPatternHandler(svc *service.ESIndexPatternService, logger *zap.Logger) *ESIndexPatternHandler {
	return &ESIndexPatternHandler{svc: svc, log: logger}
}

// SetAuditService injects the audit log service.
func (h *ESIndexPatternHandler) SetAuditService(svc *service.AuditLogService) {
	h.auditSvc = svc
}

// --- Request types ---

// CreateESIndexPatternRequest is the request body for creating an ES index pattern.
type CreateESIndexPatternRequest struct {
	DatasourceID           uint   `json:"datasource_id" binding:"required"`
	Name                   string `json:"name" binding:"required,max=191"`
	TimeField              string `json:"time_field" binding:"max=128"`
	AllowHideSystemIndices bool   `json:"allow_hide_system_indices"`
	FieldsFormat           string `json:"fields_format"`
	CrossClusterEnabled    bool   `json:"cross_cluster_enabled"`
	Note                   string `json:"note" binding:"max=512"`
}

// UpdateESIndexPatternRequest is the request body for updating an ES index pattern.
type UpdateESIndexPatternRequest struct {
	DatasourceID           uint   `json:"datasource_id" binding:"required"`
	Name                   string `json:"name" binding:"required,max=191"`
	TimeField              string `json:"time_field" binding:"max=128"`
	AllowHideSystemIndices bool   `json:"allow_hide_system_indices"`
	FieldsFormat           string `json:"fields_format"`
	CrossClusterEnabled    bool   `json:"cross_cluster_enabled"`
	Note                   string `json:"note" binding:"max=512"`
}

// --- Handler methods ---

// List returns ES index patterns, optionally filtered by datasource_id.
// GET /es-index-patterns?datasource_id=1
func (h *ESIndexPatternHandler) List(c *gin.Context) {
	var datasourceID uint
	if dsStr := c.Query("datasource_id"); dsStr != "" {
		if v, err := strconv.ParseUint(dsStr, 10, 64); err == nil {
			datasourceID = uint(v)
		}
	}

	list, err := h.svc.List(c.Request.Context(), datasourceID)
	if err != nil {
		Error(c, err)
		return
	}

	Success(c, list)
}

// Get returns a single ES index pattern by ID.
// GET /es-index-patterns/:id
func (h *ESIndexPatternHandler) Get(c *gin.Context) {
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

// Create creates a new ES index pattern.
// POST /es-index-patterns
func (h *ESIndexPatternHandler) Create(c *gin.Context) {
	var req CreateESIndexPatternRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		Error(c, apperr.WithMessage(apperr.ErrInvalidParam, err.Error()))
		return
	}

	userID := GetCurrentUserID(c)

	timeField := req.TimeField
	if timeField == "" {
		timeField = "@timestamp"
	}

	v := &model.ESIndexPattern{
		DatasourceID:           req.DatasourceID,
		Name:                   req.Name,
		TimeField:              timeField,
		AllowHideSystemIndices: req.AllowHideSystemIndices,
		FieldsFormat:           req.FieldsFormat,
		CrossClusterEnabled:    req.CrossClusterEnabled,
		Note:                   req.Note,
		CreatedBy:              strconv.FormatUint(uint64(userID), 10),
		UpdatedBy:              strconv.FormatUint(uint64(userID), 10),
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
			ResourceType: "es_index_pattern",
			ResourceID:   &v.ID,
			IP:           c.ClientIP(),
		})
	}

	Success(c, v)
}

// Update updates an existing ES index pattern.
// PUT /es-index-patterns/:id
func (h *ESIndexPatternHandler) Update(c *gin.Context) {
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

	var req UpdateESIndexPatternRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		Error(c, apperr.WithMessage(apperr.ErrInvalidParam, err.Error()))
		return
	}

	userID := GetCurrentUserID(c)

	timeField := req.TimeField
	if timeField == "" {
		timeField = "@timestamp"
	}

	input := &model.ESIndexPattern{
		DatasourceID:           req.DatasourceID,
		Name:                   req.Name,
		TimeField:              timeField,
		AllowHideSystemIndices: req.AllowHideSystemIndices,
		FieldsFormat:           req.FieldsFormat,
		CrossClusterEnabled:    req.CrossClusterEnabled,
		Note:                   req.Note,
		UpdatedBy:              strconv.FormatUint(uint64(userID), 10),
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
			ResourceType: "es_index_pattern",
			ResourceID:   &rid,
			IP:           c.ClientIP(),
		})
	}

	Success(c, nil)
}

// Delete deletes an ES index pattern.
// DELETE /es-index-patterns/:id
func (h *ESIndexPatternHandler) Delete(c *gin.Context) {
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
			ResourceType: "es_index_pattern",
			ResourceID:   &rid,
			IP:           c.ClientIP(),
		})
	}

	Success(c, nil)
}
