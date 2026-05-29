package handler

import (
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/sreagent/sreagent/internal/model"
	apperr "github.com/sreagent/sreagent/internal/pkg/errors"
	"github.com/sreagent/sreagent/internal/service"
)

// validateMuteRuleRequest performs cross-field validation for create/update requests.
func validateMuteRuleRequest(req *CreateMuteRuleRequest) error {
	// Bug 1: empty matcher = mute all alerts (dangerous)
	if len(req.MatchLabels) == 0 && req.Severities == "" && req.RuleIDs == "" {
		return apperr.WithMessage(apperr.ErrInvalidParam,
			"mute rule must specify at least one of: match_labels, severities, rule_ids")
	}

	// Bug 4: one-time and periodic windows are mutually exclusive
	hasOneTime := req.StartTime != nil || req.EndTime != nil
	hasPeriodic := req.PeriodicStart != "" || req.PeriodicEnd != ""
	if hasOneTime && hasPeriodic {
		return apperr.WithMessage(apperr.ErrInvalidParam,
			"cannot specify both one-time (start_time/end_time) and periodic (periodic_start/periodic_end) windows")
	}

	// Bug 6: validate timezone
	tz := req.Timezone
	if tz == "" {
		tz = "Asia/Shanghai"
	}
	if _, err := time.LoadLocation(tz); err != nil {
		return apperr.WithMessage(apperr.ErrInvalidParam, "invalid timezone: "+tz)
	}

	return nil
}

// MuteRuleHandler handles mute rule API requests.
type MuteRuleHandler struct {
	svc      *service.MuteRuleService
	eventSvc *service.AlertEventService
	auditSvc *service.AuditLogService
	log      *zap.Logger
}

// NewMuteRuleHandler creates a new MuteRuleHandler.
func NewMuteRuleHandler(svc *service.MuteRuleService, eventSvc *service.AlertEventService, logger *zap.Logger) *MuteRuleHandler {
	return &MuteRuleHandler{svc: svc, eventSvc: eventSvc, log: logger}
}

// SetAuditService injects the audit log service.
func (h *MuteRuleHandler) SetAuditService(svc *service.AuditLogService) {
	h.auditSvc = svc
}

// CreateMuteRuleRequest is the request body for creating a mute rule.
type CreateMuteRuleRequest struct {
	Name          string           `json:"name" binding:"required"`
	Description   string           `json:"description"`
	MatchLabels   model.JSONLabels `json:"match_labels"`
	Severities    string           `json:"severities"`
	StartTime     *time.Time       `json:"start_time"`
	EndTime       *time.Time       `json:"end_time"`
	PeriodicStart string           `json:"periodic_start"`
	PeriodicEnd   string           `json:"periodic_end"`
	DaysOfWeek    string           `json:"days_of_week"`
	Timezone      string           `json:"timezone"`
	IsEnabled     bool             `json:"is_enabled"`
	RuleIDs       string           `json:"rule_ids"`
}

// Create creates a new mute rule.
func (h *MuteRuleHandler) Create(c *gin.Context) {
	var req CreateMuteRuleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		Error(c, apperr.WithMessage(apperr.ErrInvalidParam, err.Error()))
		return
	}

	if err := validateMuteRuleRequest(&req); err != nil {
		Error(c, err)
		return
	}

	tz := req.Timezone
	if tz == "" {
		tz = "Asia/Shanghai"
	}

	userID := GetCurrentUserID(c)
	h.log.Info("mute rule create",
		zap.Uint("user_id", userID),
		zap.String("name", req.Name),
		zap.String("request_id", c.GetString("request_id")))

	rule := &model.MuteRule{
		Name:          req.Name,
		Description:   req.Description,
		MatchLabels:   req.MatchLabels,
		Severities:    req.Severities,
		StartTime:     req.StartTime,
		EndTime:       req.EndTime,
		PeriodicStart: req.PeriodicStart,
		PeriodicEnd:   req.PeriodicEnd,
		DaysOfWeek:    req.DaysOfWeek,
		Timezone:      tz,
		CreatedBy:     userID,
		IsEnabled:     req.IsEnabled,
		RuleIDs:       req.RuleIDs,
	}

	if err := h.svc.Create(c.Request.Context(), rule); err != nil {
		Error(c, err)
		return
	}

	if h.auditSvc != nil {
		h.auditSvc.Record(&model.AuditLog{
			UserID: &userID, Action: model.AuditActionCreate,
			ResourceType: model.AuditResourceMuteRule, ResourceID: &rule.ID, ResourceName: rule.Name,
			IP: c.ClientIP(),
		})
	}

	Success(c, rule)
}

// Get returns a mute rule by ID.
func (h *MuteRuleHandler) Get(c *gin.Context) {
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

// List returns a paginated list of mute rules.
func (h *MuteRuleHandler) List(c *gin.Context) {
	pq := GetPageQuery(c)

	list, total, err := h.svc.List(c.Request.Context(), pq.Page, pq.PageSize)
	if err != nil {
		Error(c, err)
		return
	}

	SuccessPage(c, list, total, pq.Page, pq.PageSize)
}

// Update updates an existing mute rule.
func (h *MuteRuleHandler) Update(c *gin.Context) {
	id, err := GetIDParam(c, "id")
	if err != nil {
		Error(c, err)
		return
	}

	var req CreateMuteRuleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		Error(c, apperr.WithMessage(apperr.ErrInvalidParam, err.Error()))
		return
	}

	if err := validateMuteRuleRequest(&req); err != nil {
		Error(c, err)
		return
	}

	tz := req.Timezone
	if tz == "" {
		tz = "Asia/Shanghai"
	}

	h.log.Info("mute rule update",
		zap.Uint("user_id", GetCurrentUserID(c)),
		zap.Uint("rule_id", id),
		zap.String("name", req.Name),
		zap.String("request_id", c.GetString("request_id")))

	rule := &model.MuteRule{
		Name:          req.Name,
		Description:   req.Description,
		MatchLabels:   req.MatchLabels,
		Severities:    req.Severities,
		StartTime:     req.StartTime,
		EndTime:       req.EndTime,
		PeriodicStart: req.PeriodicStart,
		PeriodicEnd:   req.PeriodicEnd,
		DaysOfWeek:    req.DaysOfWeek,
		Timezone:      tz,
		IsEnabled:     req.IsEnabled,
		RuleIDs:       req.RuleIDs,
	}
	rule.ID = id

	if err := h.svc.Update(c.Request.Context(), rule); err != nil {
		Error(c, err)
		return
	}

	if h.auditSvc != nil {
		uid := GetCurrentUserID(c)
		h.auditSvc.Record(&model.AuditLog{
			UserID: &uid, Action: model.AuditActionUpdate,
			ResourceType: model.AuditResourceMuteRule, ResourceID: &id, ResourceName: req.Name,
			IP: c.ClientIP(),
		})
	}

	Success(c, rule)
}

// Delete deletes a mute rule by ID.
func (h *MuteRuleHandler) Delete(c *gin.Context) {
	id, err := GetIDParam(c, "id")
	if err != nil {
		Error(c, err)
		return
	}

	h.log.Info("mute rule delete",
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
			ResourceType: model.AuditResourceMuteRule, ResourceID: &id,
			IP: c.ClientIP(),
		})
	}

	Success(c, nil)
}

// MutePreviewItem describes which currently-firing alerts a mute rule would suppress.
type MutePreviewItem struct {
	RuleID        uint               `json:"rule_id"`
	RuleName      string             `json:"rule_name"`
	MatchedCount  int                `json:"matched_count"`
	MatchedAlerts []model.AlertEvent `json:"matched_alerts"`
}

// Preview returns a preview of which currently-firing alerts each enabled mute rule
// would suppress right now.
// GET /api/v1/mute-rules/preview
func (h *MuteRuleHandler) Preview(c *gin.Context) {
	if h.eventSvc == nil {
		Error(c, apperr.WithMessage(apperr.ErrInternal, "alert event service not available"))
		return
	}

	ctx := c.Request.Context()

	// Fetch all enabled mute rules
	rules, _, err := h.svc.List(ctx, 1, 1000)
	if err != nil {
		Error(c, err)
		return
	}

	// Fetch all currently firing alerts (up to 500)
	firingEvents, total, err := h.eventSvc.List(ctx, "firing", "", 1, 500)
	if err != nil {
		Error(c, err)
		return
	}
	truncated := total > 500

	now := time.Now()
	result := make([]MutePreviewItem, 0, len(rules))
	for _, rule := range rules {
		if !rule.IsEnabled {
			continue
		}
		item := MutePreviewItem{
			RuleID:        rule.ID,
			RuleName:      rule.Name,
			MatchedAlerts: []model.AlertEvent{},
		}
		for _, ev := range firingEvents {
			if h.svc.MatchesRule(&rule, &ev, now) {
				item.MatchedAlerts = append(item.MatchedAlerts, ev)
			}
		}
		item.MatchedCount = len(item.MatchedAlerts)
		result = append(result, item)
	}

	Success(c, gin.H{
		"preview":      result,
		"total_firing": total,
		"truncated":    truncated,
	})
}

// PreviewOne returns the preview for a single mute rule.
// GET /api/v1/mute-rules/:id/preview
func (h *MuteRuleHandler) PreviewOne(c *gin.Context) {
	id, err := GetIDParam(c, "id")
	if err != nil {
		Error(c, err)
		return
	}

	if h.eventSvc == nil {
		Error(c, apperr.WithMessage(apperr.ErrInternal, "alert event service not available"))
		return
	}

	ctx := c.Request.Context()

	// Fetch the specific mute rule
	rule, err := h.svc.GetByID(ctx, id)
	if err != nil {
		Error(c, err)
		return
	}

	// Fetch all currently firing alerts (up to 500)
	firingEvents, _, err := h.eventSvc.List(ctx, "firing", "", 1, 500)
	if err != nil {
		Error(c, err)
		return
	}

	now := time.Now()
	item := MutePreviewItem{
		RuleID:        rule.ID,
		RuleName:      rule.Name,
		MatchedAlerts: []model.AlertEvent{},
	}
	for _, ev := range firingEvents {
		if h.svc.MatchesRule(rule, &ev, now) {
			item.MatchedAlerts = append(item.MatchedAlerts, ev)
		}
	}
	item.MatchedCount = len(item.MatchedAlerts)

	Success(c, item)
}

// muteBatchIDsReq is the request body for batch operations.
type muteBatchIDsReq struct {
	IDs []uint `json:"ids" binding:"required,min=1"`
}

// BatchEnable enables multiple mute rules.
func (h *MuteRuleHandler) BatchEnable(c *gin.Context) {
	var req muteBatchIDsReq
	if err := c.ShouldBindJSON(&req); err != nil {
		Error(c, apperr.WithMessage(apperr.ErrInvalidParam, err.Error()))
		return
	}
	if err := h.svc.BatchEnable(c.Request.Context(), req.IDs); err != nil {
		Error(c, err)
		return
	}
	Success(c, nil)
}

// BatchDisable disables multiple mute rules.
func (h *MuteRuleHandler) BatchDisable(c *gin.Context) {
	var req muteBatchIDsReq
	if err := c.ShouldBindJSON(&req); err != nil {
		Error(c, apperr.WithMessage(apperr.ErrInvalidParam, err.Error()))
		return
	}
	if err := h.svc.BatchDisable(c.Request.Context(), req.IDs); err != nil {
		Error(c, err)
		return
	}
	Success(c, nil)
}

// BatchDelete deletes multiple mute rules.
func (h *MuteRuleHandler) BatchDelete(c *gin.Context) {
	var req muteBatchIDsReq
	if err := c.ShouldBindJSON(&req); err != nil {
		Error(c, apperr.WithMessage(apperr.ErrInvalidParam, err.Error()))
		return
	}
	if err := h.svc.BatchDelete(c.Request.Context(), req.IDs); err != nil {
		Error(c, err)
		return
	}
	Success(c, nil)
}
