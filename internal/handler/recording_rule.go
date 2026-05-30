package handler

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/sreagent/sreagent/internal/model"
	apperr "github.com/sreagent/sreagent/internal/pkg/errors"
	"github.com/sreagent/sreagent/internal/service"
)

type RecordingRuleHandler struct {
	svc      *service.RecordingRuleService
	auditSvc *service.AuditLogService
	log      *zap.Logger
}

func NewRecordingRuleHandler(svc *service.RecordingRuleService, log *zap.Logger) *RecordingRuleHandler {
	return &RecordingRuleHandler{svc: svc, log: log}
}

func (h *RecordingRuleHandler) SetAuditService(svc *service.AuditLogService) {
	h.auditSvc = svc
}

// List returns recording rules with optional filtering and pagination.
func (h *RecordingRuleHandler) List(c *gin.Context) {
	groupIDStr := c.Query("group_id")
	query := c.Query("query")
	disabledStr := c.Query("disabled")
	pq := GetPageQuery(c)

	var groupID uint
	if groupIDStr != "" {
		gid, err := strconv.ParseUint(groupIDStr, 10, 64)
		if err != nil {
			Error(c, apperr.WithMessage(apperr.ErrInvalidParam, "invalid group_id"))
			return
		}
		groupID = uint(gid)
	}

	var disabled *int
	if disabledStr != "" {
		d, err := strconv.Atoi(disabledStr)
		if err == nil {
			disabled = &d
		}
	}

	rules, total, err := h.svc.ListWithFilter(c.Request.Context(), groupID, query, disabled, pq.Page, pq.PageSize)
	if err != nil {
		Error(c, apperr.WithMessage(apperr.ErrInternal, err.Error()))
		return
	}
	SuccessPage(c, rules, total, pq.Page, pq.PageSize)
}

// Get returns a single recording rule by ID.
func (h *RecordingRuleHandler) Get(c *gin.Context) {
	id, err := GetIDParam(c, "id")
	if err != nil {
		Error(c, apperr.WithMessage(apperr.ErrInvalidParam, "invalid id"))
		return
	}
	rule, err := h.svc.GetByID(c.Request.Context(), id)
	if err != nil {
		Error(c, err)
		return
	}
	Success(c, rule)
}

// Create creates a new recording rule.
func (h *RecordingRuleHandler) Create(c *gin.Context) {
	var req CreateRecordingRuleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		Error(c, apperr.WithMessage(apperr.ErrInvalidParam, err.Error()))
		return
	}

	userID := GetCurrentUserID(c)

	writeBack := 1 // default: write results back to datasource
	if req.WriteBack != nil {
		writeBack = *req.WriteBack
	}

	rule := &model.RecordingRule{
		GroupID:           req.GroupID,
		Name:              req.Name,
		PromQL:            req.PromQL,
		DatasourceIDsJSON: req.DatasourceIDs,
		CronPattern:       req.CronPattern,
		Disabled:          req.Disabled,
		WriteBack:         writeBack,
		AppendTagsJSON:    req.AppendTags,
		Note:              req.Note,
		QueryConfigsJSON:  req.QueryConfigs,
		CreatedBy:         strconv.FormatUint(uint64(userID), 10),
		UpdatedBy:         strconv.FormatUint(uint64(userID), 10),
	}
	if rule.CronPattern == "" {
		rule.CronPattern = "@every 60s"
	}

	if err := h.svc.Create(c.Request.Context(), rule); err != nil {
		Error(c, apperr.WithMessage(apperr.ErrInternal, err.Error()))
		return
	}
	if h.auditSvc != nil {
		uid := userID
		h.auditSvc.Record(&model.AuditLog{
			UserID:       &uid,
			Action:       model.AuditActionCreate,
			ResourceType: "recording_rule",
			ResourceID:   &rule.ID,
			IP:           c.ClientIP(),
		})
	}
	Success(c, gin.H{
		"rule": rule,
		"warning": "Recording rules are in Phase 1: queries are validated and executed but results are NOT written back to the datasource as new time series. This is an experimental feature.",
	})
}

// Update updates an existing recording rule.
func (h *RecordingRuleHandler) Update(c *gin.Context) {
	id, err := GetIDParam(c, "id")
	if err != nil {
		Error(c, apperr.WithMessage(apperr.ErrInvalidParam, "invalid id"))
		return
	}

	existing, err := h.svc.GetByID(c.Request.Context(), id)
	if err != nil {
		Error(c, err)
		return
	}

	var req UpdateRecordingRuleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		Error(c, apperr.WithMessage(apperr.ErrInvalidParam, err.Error()))
		return
	}

	userID := GetCurrentUserID(c)

	input := &model.RecordingRule{
		Name:              req.Name,
		PromQL:            req.PromQL,
		DatasourceIDsJSON: req.DatasourceIDs,
		CronPattern:       req.CronPattern,
		Disabled:          req.Disabled,
		WriteBack:         existing.WriteBack, // preserve existing by default
		AppendTagsJSON:    req.AppendTags,
		Note:              req.Note,
		QueryConfigsJSON:  req.QueryConfigs,
		UpdatedBy:         strconv.FormatUint(uint64(userID), 10),
	}
	if req.WriteBack != nil {
		input.WriteBack = *req.WriteBack
	}
	if input.CronPattern == "" {
		input.CronPattern = "@every 60s"
	}

	if err := h.svc.Update(c.Request.Context(), existing, input); err != nil {
		Error(c, apperr.WithMessage(apperr.ErrInternal, err.Error()))
		return
	}
	if h.auditSvc != nil {
		uid := userID
		rid := id
		h.auditSvc.Record(&model.AuditLog{
			UserID:       &uid,
			Action:       model.AuditActionUpdate,
			ResourceType: "recording_rule",
			ResourceID:   &rid,
			IP:           c.ClientIP(),
		})
	}
	Success(c, nil)
}

// Delete deletes a recording rule by ID.
func (h *RecordingRuleHandler) Delete(c *gin.Context) {
	id, err := GetIDParam(c, "id")
	if err != nil {
		Error(c, apperr.WithMessage(apperr.ErrInvalidParam, "invalid id"))
		return
	}

	rule, err := h.svc.GetByID(c.Request.Context(), id)
	if err != nil {
		Error(c, err)
		return
	}

	if err := h.svc.Delete(c.Request.Context(), id, rule.GroupID); err != nil {
		Error(c, apperr.WithMessage(apperr.ErrInternal, err.Error()))
		return
	}
	if h.auditSvc != nil {
		uid := GetCurrentUserID(c)
		rid := id
		h.auditSvc.Record(&model.AuditLog{
			UserID:       &uid,
			Action:       model.AuditActionDelete,
			ResourceType: "recording_rule",
			ResourceID:   &rid,
			IP:           c.ClientIP(),
		})
	}
	Success(c, nil)
}

// BatchDelete deletes multiple recording rules by IDs.
func (h *RecordingRuleHandler) BatchDelete(c *gin.Context) {
	var req BatchDeleteRecordingRuleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		Error(c, apperr.WithMessage(apperr.ErrInvalidParam, err.Error()))
		return
	}

	if err := h.svc.DeleteByIDs(c.Request.Context(), req.IDs, req.GroupID); err != nil {
		Error(c, apperr.WithMessage(apperr.ErrInternal, err.Error()))
		return
	}
	Success(c, nil)
}

// BatchCreate creates multiple recording rules (for import).
func (h *RecordingRuleHandler) BatchCreate(c *gin.Context) {
	var req BatchCreateRecordingRuleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		Error(c, apperr.WithMessage(apperr.ErrInvalidParam, err.Error()))
		return
	}

	userID := GetCurrentUserID(c)
	uidStr := strconv.FormatUint(uint64(userID), 10)

	for i := range req.Rules {
		req.Rules[i].GroupID = req.GroupID
		req.Rules[i].CreatedBy = uidStr
		req.Rules[i].UpdatedBy = uidStr
	}

	results := h.svc.BatchCreate(c.Request.Context(), req.Rules)
	Success(c, results)
}

// UpdateFields batch-updates specific fields across multiple rules.
func (h *RecordingRuleHandler) UpdateFields(c *gin.Context) {
	var req UpdateFieldsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		Error(c, apperr.WithMessage(apperr.ErrInvalidParam, err.Error()))
		return
	}

	userID := GetCurrentUserID(c)
	req.Fields["updated_by"] = strconv.FormatUint(uint64(userID), 10)

	var lastErr error
	for _, id := range req.IDs {
		if err := h.svc.UpdateFields(c.Request.Context(), id, req.Fields); err != nil {
			h.log.Error("failed to update recording rule fields",
				zap.Uint("id", id), zap.Error(err))
			lastErr = err
		}
	}
	if lastErr != nil {
		Error(c, apperr.WithMessage(apperr.ErrInternal, lastErr.Error()))
		return
	}
	Success(c, nil)
}

// --------------- Request types ---------------

type CreateRecordingRuleRequest struct {
	GroupID       uint               `json:"group_id" binding:"required"`
	Name          string             `json:"name" binding:"required"`
	PromQL        string             `json:"prom_ql" binding:"required"`
	DatasourceIDs []int64            `json:"datasource_ids"`
	CronPattern   string             `json:"cron_pattern"`
	Disabled      int                `json:"disabled"`
	WriteBack     *int               `json:"write_back"`
	AppendTags    []string           `json:"append_tags"`
	Note          string             `json:"note"`
	QueryConfigs  []model.QueryConfig `json:"query_configs"`
}

type UpdateRecordingRuleRequest struct {
	Name          string             `json:"name" binding:"required"`
	PromQL        string             `json:"prom_ql" binding:"required"`
	DatasourceIDs []int64            `json:"datasource_ids"`
	CronPattern   string             `json:"cron_pattern"`
	Disabled      int                `json:"disabled"`
	WriteBack     *int               `json:"write_back"`
	AppendTags    []string           `json:"append_tags"`
	Note          string             `json:"note"`
	QueryConfigs  []model.QueryConfig `json:"query_configs"`
}

type BatchDeleteRecordingRuleRequest struct {
	GroupID uint   `json:"group_id" binding:"required"`
	IDs     []uint `json:"ids" binding:"required"`
}

type BatchCreateRecordingRuleRequest struct {
	GroupID uint                  `json:"group_id" binding:"required"`
	Rules   []model.RecordingRule `json:"rules" binding:"required"`
}

type UpdateFieldsRequest struct {
	IDs    []uint                 `json:"ids" binding:"required"`
	Fields map[string]interface{} `json:"fields" binding:"required"`
}
