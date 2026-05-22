package handler

import (
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	apperr "github.com/sreagent/sreagent/internal/pkg/errors"

	"github.com/sreagent/sreagent/internal/model"
	"github.com/sreagent/sreagent/internal/service"
)

// InhibitionRuleHandler handles inhibition rule API requests.
type InhibitionRuleHandler struct {
	svc      *service.InhibitionRuleService
	auditSvc *service.AuditLogService
	log      *zap.Logger
}

// NewInhibitionRuleHandler creates a new InhibitionRuleHandler.
func NewInhibitionRuleHandler(svc *service.InhibitionRuleService, logger ...*zap.Logger) *InhibitionRuleHandler {
	l := zap.NewNop()
	if len(logger) > 0 && logger[0] != nil {
		l = logger[0]
	}
	return &InhibitionRuleHandler{svc: svc, log: l}
}

// SetAuditService injects the audit log service (called after construction to avoid circular DI).
func (h *InhibitionRuleHandler) SetAuditService(svc *service.AuditLogService) {
	h.auditSvc = svc
}

// InhibitionRuleRequest is the request body for create/update.
type InhibitionRuleRequest struct {
	Name        string           `json:"name" binding:"required"`
	Description string           `json:"description"`
	SourceMatch model.JSONLabels `json:"source_match"`
	TargetMatch model.JSONLabels `json:"target_match"`
	EqualLabels string           `json:"equal_labels"`
	IsEnabled   bool             `json:"is_enabled"`
}

// Create creates a new inhibition rule.
func (h *InhibitionRuleHandler) Create(c *gin.Context) {
	var req InhibitionRuleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		Error(c, apperr.WithMessage(apperr.ErrInvalidParam, err.Error()))
		return
	}
	userID := GetCurrentUserID(c)
	h.log.Info("inhibition rule create",
		zap.Uint("user_id", userID),
		zap.String("name", req.Name),
		zap.String("request_id", c.GetString("request_id")))

	rule := &model.InhibitionRule{
		Name:        req.Name,
		Description: req.Description,
		SourceMatch: req.SourceMatch,
		TargetMatch: req.TargetMatch,
		EqualLabels: req.EqualLabels,
		IsEnabled:   req.IsEnabled,
		CreatedBy:   userID,
	}
	if rule.SourceMatch == nil {
		rule.SourceMatch = model.JSONLabels{}
	}
	if rule.TargetMatch == nil {
		rule.TargetMatch = model.JSONLabels{}
	}
	if err := h.svc.Create(c.Request.Context(), rule); err != nil {
		Error(c, err)
		return
	}

	if h.auditSvc != nil {
		uid := GetCurrentUserID(c)
		h.auditSvc.Record(&model.AuditLog{
			UserID: &uid, Action: model.AuditActionCreate,
			ResourceType: model.AuditResourceInhibitionRule, ResourceID: &rule.ID, ResourceName: rule.Name,
			IP: c.ClientIP(),
		})
	}

	Success(c, rule)
}

// Get returns an inhibition rule by ID.
func (h *InhibitionRuleHandler) Get(c *gin.Context) {
	id, err := GetIDParam(c, "id")
	if err != nil {
		Error(c, err)
		return
	}
	rule, err := h.svc.GetByID(c.Request.Context(), id)
	if err != nil {
		Error(c, err)
		return
	}
	Success(c, rule)
}

// List returns a paginated list of inhibition rules.
func (h *InhibitionRuleHandler) List(c *gin.Context) {
	pq := GetPageQuery(c)
	list, total, err := h.svc.List(c.Request.Context(), pq.Page, pq.PageSize)
	if err != nil {
		Error(c, err)
		return
	}
	SuccessPage(c, list, total, pq.Page, pq.PageSize)
}

// Update updates an existing inhibition rule.
func (h *InhibitionRuleHandler) Update(c *gin.Context) {
	id, err := GetIDParam(c, "id")
	if err != nil {
		Error(c, err)
		return
	}
	var req InhibitionRuleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		Error(c, apperr.WithMessage(apperr.ErrInvalidParam, err.Error()))
		return
	}
	h.log.Info("inhibition rule update",
		zap.Uint("user_id", GetCurrentUserID(c)),
		zap.Uint("rule_id", id),
		zap.String("name", req.Name),
		zap.String("request_id", c.GetString("request_id")))

	rule := &model.InhibitionRule{
		Name:        req.Name,
		Description: req.Description,
		SourceMatch: req.SourceMatch,
		TargetMatch: req.TargetMatch,
		EqualLabels: req.EqualLabels,
		IsEnabled:   req.IsEnabled,
	}
	rule.ID = id
	if rule.SourceMatch == nil {
		rule.SourceMatch = model.JSONLabels{}
	}
	if rule.TargetMatch == nil {
		rule.TargetMatch = model.JSONLabels{}
	}
	if err := h.svc.Update(c.Request.Context(), rule); err != nil {
		Error(c, err)
		return
	}

	if h.auditSvc != nil {
		uid := GetCurrentUserID(c)
		h.auditSvc.Record(&model.AuditLog{
			UserID: &uid, Action: model.AuditActionUpdate,
			ResourceType: model.AuditResourceInhibitionRule, ResourceID: &id, ResourceName: req.Name,
			IP: c.ClientIP(),
		})
	}

	Success(c, rule)
}

// Delete soft-deletes an inhibition rule.
func (h *InhibitionRuleHandler) Delete(c *gin.Context) {
	id, err := GetIDParam(c, "id")
	if err != nil {
		Error(c, err)
		return
	}

	h.log.Info("inhibition rule delete",
		zap.Uint("user_id", GetCurrentUserID(c)),
		zap.Uint("rule_id", id),
		zap.String("request_id", c.GetString("request_id")))

	if err := h.svc.Delete(c.Request.Context(), id); err != nil {
		Error(c, err)
		return
	}

	if h.auditSvc != nil {
		uid := GetCurrentUserID(c)
		h.auditSvc.Record(&model.AuditLog{
			UserID: &uid, Action: model.AuditActionDelete,
			ResourceType: model.AuditResourceInhibitionRule, ResourceID: &id,
			IP: c.ClientIP(),
		})
	}

	Success(c, nil)
}
