package handler

import (
	"encoding/csv"
	"fmt"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/sreagent/sreagent/internal/model"
	apperr "github.com/sreagent/sreagent/internal/pkg/errors"
	"github.com/sreagent/sreagent/internal/service"
)

type AlertEventHandler struct {
	svc      *service.AlertEventService
	auditSvc *service.AuditLogService
	log      *zap.Logger
}

func NewAlertEventHandler(svc *service.AlertEventService, logger ...*zap.Logger) *AlertEventHandler {
	l := zap.NewNop()
	if len(logger) > 0 && logger[0] != nil {
		l = logger[0]
	}
	return &AlertEventHandler{svc: svc, log: l}
}

func (h *AlertEventHandler) SetAuditService(svc *service.AuditLogService) {
	h.auditSvc = svc
}

// List returns paginated alert events with optional filters.
// Supports view_mode=mine|unassigned|all and user_id for role-based visibility.
func (h *AlertEventHandler) List(c *gin.Context) {
	pq := GetPageQuery(c)

	filter := service.AlertEventFilter{
		Status:    c.Query("status"),
		Severity:  c.Query("severity"),
		AlertName: c.Query("alert_name"), // FE4-1: wire frontend search to backend
		ViewMode:  c.Query("view_mode"),
		Page:      pq.Page,
		PageSize:  pq.PageSize,
	}

	// FE4-2/4-3: Wire time range params from frontend filter bar
	if startStr := c.Query("start_time"); startStr != "" {
		if t, err := time.Parse(time.RFC3339, startStr); err == nil {
			filter.StartTime = &t
		}
	}
	if endStr := c.Query("end_time"); endStr != "" {
		if t, err := time.Parse(time.RFC3339, endStr); err == nil {
			filter.EndTime = &t
		}
	}

	// FE4-4: Wire rule_id filter
	if ruleStr := c.Query("rule_id"); ruleStr != "" {
		if rid, err := strconv.ParseUint(ruleStr, 10, 64); err == nil {
			ruleID := uint(rid)
			filter.RuleID = &ruleID
		}
	}

	// user_id param overrides current user (admin use); default to current user.
	// Non-admin users can only query their own data — silently ignore user_id param.
	currentUserID := GetCurrentUserID(c)
	currentRole, _ := c.Get("role")
	isAdmin := currentRole == "admin"

	if isAdmin {
		if uidStr := c.Query("user_id"); uidStr != "" {
			if uid, err := strconv.ParseUint(uidStr, 10, 64); err == nil {
				filter.UserID = uint(uid)
			}
		}
	}
	if filter.UserID == 0 {
		filter.UserID = currentUserID
	}

	list, total, err := h.svc.ListWithFilter(c.Request.Context(), filter)
	if err != nil {
		Error(c, err)
		return
	}

	SuccessPage(c, list, total, pq.Page, pq.PageSize)
}

// Get returns a single alert event with its timeline.
func (h *AlertEventHandler) Get(c *gin.Context) {
	id, err := GetIDParam(c, "id")
	if err != nil {
		Error(c, err)
		return
	}

	event, err := h.svc.GetByID(c.Request.Context(), id)
	if err != nil {
		Error(c, err)
		return
	}

	Success(c, event)
}

// Acknowledge marks an alert as acknowledged by the current user.
func (h *AlertEventHandler) Acknowledge(c *gin.Context) {
	id, err := GetIDParam(c, "id")
	if err != nil {
		Error(c, err)
		return
	}

	userID := GetCurrentUserID(c)
	h.log.Info("alert event acknowledge",
		zap.Uint("user_id", userID),
		zap.Uint("event_id", id),
		zap.String("request_id", c.GetString("request_id")))

	if err := h.svc.Acknowledge(c.Request.Context(), id, userID); err != nil {
		Error(c, err)
		return
	}

	if h.auditSvc != nil {
		h.auditSvc.Record(&model.AuditLog{
			UserID: &userID, Username: GetCurrentUsername(c),
			Action: model.AuditActionAck, ResourceType: model.AuditResourceAlertEvent,
			ResourceID: &id, IP: c.ClientIP(),
		})
	}
	Success(c, nil)
}

// Assign assigns an alert event to a specific user.
func (h *AlertEventHandler) Assign(c *gin.Context) {
	id, err := GetIDParam(c, "id")
	if err != nil {
		Error(c, err)
		return
	}

	var req struct {
		AssignTo uint   `json:"assign_to" binding:"required"`
		Note     string `json:"note"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		Error(c, apperr.WithMessage(apperr.ErrInvalidParam, err.Error()))
		return
	}

	operatorID := GetCurrentUserID(c)
	h.log.Info("alert event assign",
		zap.Uint("user_id", operatorID),
		zap.Uint("event_id", id),
		zap.Uint("assign_to", req.AssignTo),
		zap.String("request_id", c.GetString("request_id")))

	if err := h.svc.Assign(c.Request.Context(), id, req.AssignTo, operatorID, req.Note); err != nil {
		Error(c, err)
		return
	}

	if h.auditSvc != nil {
		h.auditSvc.Record(&model.AuditLog{
			UserID: &operatorID, Username: GetCurrentUsername(c),
			Action: model.AuditActionAssign, ResourceType: model.AuditResourceAlertEvent,
			ResourceID: &id, IP: c.ClientIP(),
		})
	}
	Success(c, nil)
}

// Resolve marks an alert as resolved.
func (h *AlertEventHandler) Resolve(c *gin.Context) {
	id, err := GetIDParam(c, "id")
	if err != nil {
		Error(c, err)
		return
	}

	var req struct {
		Resolution string `json:"resolution"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		Error(c, apperr.WithMessage(apperr.ErrInvalidParam, err.Error()))
		return
	}

	userID := GetCurrentUserID(c)
	h.log.Info("alert event resolve",
		zap.Uint("user_id", userID),
		zap.Uint("event_id", id),
		zap.String("request_id", c.GetString("request_id")))

	if err := h.svc.Resolve(c.Request.Context(), id, userID, req.Resolution); err != nil {
		Error(c, err)
		return
	}

	if h.auditSvc != nil {
		h.auditSvc.Record(&model.AuditLog{
			UserID: &userID, Username: GetCurrentUsername(c),
			Action: model.AuditActionResolve, ResourceType: model.AuditResourceAlertEvent,
			ResourceID: &id, IP: c.ClientIP(),
		})
	}
	Success(c, nil)
}

// Close closes an alert event.
func (h *AlertEventHandler) Close(c *gin.Context) {
	id, err := GetIDParam(c, "id")
	if err != nil {
		Error(c, err)
		return
	}

	var req struct {
		Note string `json:"note"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		Error(c, apperr.WithMessage(apperr.ErrInvalidParam, err.Error()))
		return
	}

	userID := GetCurrentUserID(c)
	h.log.Info("alert event close",
		zap.Uint("user_id", userID),
		zap.Uint("event_id", id),
		zap.String("request_id", c.GetString("request_id")))

	if err := h.svc.Close(c.Request.Context(), id, userID, req.Note); err != nil {
		Error(c, err)
		return
	}

	if h.auditSvc != nil {
		h.auditSvc.Record(&model.AuditLog{
			UserID: &userID, Username: GetCurrentUsername(c),
			Action: model.AuditActionClose, ResourceType: model.AuditResourceAlertEvent,
			ResourceID: &id, IP: c.ClientIP(),
		})
	}
	Success(c, nil)
}

// AddComment adds a comment to an alert event timeline.
func (h *AlertEventHandler) AddComment(c *gin.Context) {
	id, err := GetIDParam(c, "id")
	if err != nil {
		Error(c, err)
		return
	}

	var req struct {
		Note string `json:"note" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		Error(c, apperr.WithMessage(apperr.ErrInvalidParam, err.Error()))
		return
	}

	userID := GetCurrentUserID(c)
	h.log.Info("alert event add comment",
		zap.Uint("user_id", userID),
		zap.Uint("event_id", id),
		zap.String("request_id", c.GetString("request_id")))

	if err := h.svc.AddComment(c.Request.Context(), id, userID, req.Note); err != nil {
		Error(c, err)
		return
	}

	Success(c, nil)
}

// GetTimeline returns the timeline for an alert event.
func (h *AlertEventHandler) GetTimeline(c *gin.Context) {
	id, err := GetIDParam(c, "id")
	if err != nil {
		Error(c, err)
		return
	}

	timeline, err := h.svc.GetTimeline(c.Request.Context(), id)
	if err != nil {
		Error(c, err)
		return
	}

	Success(c, timeline)
}

// Silence silences an alert for a specified duration.
func (h *AlertEventHandler) Silence(c *gin.Context) {
	id, err := GetIDParam(c, "id")
	if err != nil {
		Error(c, err)
		return
	}

	var req struct {
		DurationMinutes int    `json:"duration_minutes" binding:"required,min=1"`
		Reason          string `json:"reason"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		Error(c, apperr.WithMessage(apperr.ErrInvalidParam, err.Error()))
		return
	}

	userID := GetCurrentUserID(c)
	h.log.Info("alert event silence",
		zap.Uint("user_id", userID),
		zap.Uint("event_id", id),
		zap.Int("duration_minutes", req.DurationMinutes),
		zap.String("request_id", c.GetString("request_id")))

	if err := h.svc.Silence(c.Request.Context(), id, userID, req.DurationMinutes, req.Reason); err != nil {
		Error(c, err)
		return
	}

	if h.auditSvc != nil {
		h.auditSvc.Record(&model.AuditLog{
			UserID: &userID, Username: GetCurrentUsername(c),
			Action: model.AuditActionSilence, ResourceType: model.AuditResourceAlertEvent,
			ResourceID: &id, IP: c.ClientIP(),
		})
	}
	Success(c, nil)
}

// BatchAcknowledge acknowledges multiple alerts at once.
func (h *AlertEventHandler) BatchAcknowledge(c *gin.Context) {
	var req struct {
		IDs []uint `json:"ids" binding:"required,min=1"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		Error(c, apperr.WithMessage(apperr.ErrInvalidParam, err.Error()))
		return
	}

	userID := GetCurrentUserID(c)
	h.log.Info("alert event batch acknowledge",
		zap.Uint("user_id", userID),
		zap.Int("count", len(req.IDs)),
		zap.String("request_id", c.GetString("request_id")))

	success, failed, err := h.svc.BatchAcknowledge(c.Request.Context(), req.IDs, userID)
	if err != nil {
		Error(c, err)
		return
	}

	if h.auditSvc != nil {
		h.auditSvc.Record(&model.AuditLog{
			UserID: &userID, Username: GetCurrentUsername(c),
			Action: model.AuditActionAck, ResourceType: model.AuditResourceAlertEvent,
			Detail: "batch", IP: c.ClientIP(),
		})
	}
	Success(c, gin.H{"success": success, "failed": failed})
}

// BatchClose closes multiple alerts at once.
func (h *AlertEventHandler) BatchClose(c *gin.Context) {
	var req struct {
		IDs []uint `json:"ids" binding:"required,min=1"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		Error(c, apperr.WithMessage(apperr.ErrInvalidParam, err.Error()))
		return
	}

	userID := GetCurrentUserID(c)
	h.log.Info("alert event batch close",
		zap.Uint("user_id", userID),
		zap.Int("count", len(req.IDs)),
		zap.String("request_id", c.GetString("request_id")))

	success, failed, err := h.svc.BatchClose(c.Request.Context(), req.IDs, userID)
	if err != nil {
		Error(c, err)
		return
	}

	if h.auditSvc != nil {
		h.auditSvc.Record(&model.AuditLog{
			UserID: &userID, Username: GetCurrentUsername(c),
			Action: model.AuditActionClose, ResourceType: model.AuditResourceAlertEvent,
			Detail: "batch", IP: c.ClientIP(),
		})
	}
	Success(c, gin.H{"success": success, "failed": failed})
}

// Export streams alert events as a CSV file.
// GET /api/v1/alert-events/export?status=firing&severity=critical&start=RFC3339&end=RFC3339
func (h *AlertEventHandler) Export(c *gin.Context) {
	filter := service.AlertEventFilter{
		Status:   c.Query("status"),
		Severity: c.Query("severity"),
		Page:     1,
		PageSize: 10000, // cap at 10k rows
	}
	// user_id param: only admins can export other users' data
	currentUserID := GetCurrentUserID(c)
	currentRole, _ := c.Get("role")
	isAdmin := currentRole == "admin"

	if isAdmin {
		if uidStr := c.Query("user_id"); uidStr != "" {
			if uid, err := strconv.ParseUint(uidStr, 10, 64); err == nil {
				filter.UserID = uint(uid)
			}
		}
	}
	if filter.UserID == 0 {
		filter.UserID = currentUserID
	}
	filter.ViewMode = c.DefaultQuery("view_mode", "all")

	if startStr := c.Query("start"); startStr != "" {
		if t, err := time.Parse(time.RFC3339, startStr); err == nil {
			filter.StartTime = &t
		}
	}
	if endStr := c.Query("end"); endStr != "" {
		if t, err := time.Parse(time.RFC3339, endStr); err == nil {
			filter.EndTime = &t
		}
	}

	events, _, err := h.svc.ListWithFilter(c.Request.Context(), filter)
	if err != nil {
		Error(c, err)
		return
	}

	filename := fmt.Sprintf("alert-events-%s.csv", time.Now().Format("20060102-150405"))
	c.Header("Content-Type", "text/csv; charset=utf-8")
	c.Header("Content-Disposition", "attachment; filename="+filename)
	c.Header("Transfer-Encoding", "chunked")

	w := csv.NewWriter(c.Writer)
	if err := w.Write([]string{
		"ID", "AlertName", "Severity", "Status", "Source",
		"FiredAt", "AckedAt", "ResolvedAt", "ClosedAt",
		"Labels", "Annotations", "Resolution", "FireCount",
	}); err != nil {
		h.log.Error("failed to write CSV header", zap.Error(err))
		return
	}

	fmtT := func(t *time.Time) string {
		if t == nil {
			return ""
		}
		return t.Format(time.RFC3339)
	}
	fmtLabels := func(m model.JSONLabels) string {
		s := ""
		for k, v := range m {
			if s != "" {
				s += "; "
			}
			s += k + "=" + v
		}
		return s
	}

	for _, ev := range events {
		if err := w.Write([]string{
			strconv.FormatUint(uint64(ev.ID), 10),
			ev.AlertName,
			string(ev.Severity),
			string(ev.Status),
			ev.Source,
			ev.FiredAt.Format(time.RFC3339),
			fmtT(ev.AckedAt),
			fmtT(ev.ResolvedAt),
			fmtT(ev.ClosedAt),
			fmtLabels(ev.Labels),
			fmtLabels(ev.Annotations),
			ev.Resolution,
			strconv.Itoa(ev.FireCount),
		}); err != nil {
			h.log.Error("failed to write CSV row", zap.Error(err))
			return
		}
	}
	w.Flush()
	if err := w.Error(); err != nil {
		h.log.Error("CSV flush error", zap.Error(err))
	}
}

// AlertGroupItem represents a set of alerts grouped by alert_name + source.
type AlertGroupItem struct {
	AlertName         string           `json:"alert_name"`
	Source            string           `json:"source"`
	TotalCount        int64            `json:"total_count"`
	SeverityBreakdown map[string]int64 `json:"severity_breakdown"`
	StatusBreakdown   map[string]int64 `json:"status_breakdown"`
	LatestFiredAt     time.Time        `json:"latest_fired_at"`
	OldestFiredAt     time.Time        `json:"oldest_fired_at"`
	MaxFireCount      int              `json:"max_fire_count"` // noisiest single event in group
}

// ListGroups aggregates alert events by alert_name + source so operators can
// spot noisy rules at a glance.
// GET /api/v1/alert-events/groups?status=firing,acknowledged&severity=critical,warning
func (h *AlertEventHandler) ListGroups(c *gin.Context) {
	// Status filter — default to active states
	statusParam := c.DefaultQuery("status", "firing,acknowledged,assigned")
	var statuses []string
	for _, s := range splitCSV(statusParam) {
		if s != "" {
			statuses = append(statuses, s)
		}
	}

	severityParam := c.Query("severity")
	var severities []string
	for _, s := range splitCSV(severityParam) {
		if s != "" {
			severities = append(severities, s)
		}
	}

	rows, err := h.svc.ListGrouped(c.Request.Context(), statuses, severities)
	if err != nil {
		Error(c, err)
		return
	}

	// Merge into groups keyed by (alert_name, source).
	type key struct{ name, source string }
	order := []key{}
	groups := map[key]*AlertGroupItem{}

	for _, r := range rows {
		k := key{r.AlertName, r.Source}
		g, exists := groups[k]
		if !exists {
			g = &AlertGroupItem{
				AlertName:         r.AlertName,
				Source:            r.Source,
				SeverityBreakdown: map[string]int64{"critical": 0, "warning": 0, "info": 0},
				StatusBreakdown:   map[string]int64{},
				OldestFiredAt:     r.OldestFired,
				LatestFiredAt:     r.LatestFired,
			}
			groups[k] = g
			order = append(order, k)
		}
		g.TotalCount += r.Cnt
		g.SeverityBreakdown[r.Severity] += r.Cnt
		g.StatusBreakdown[r.Status] += r.Cnt
		if r.LatestFired.After(g.LatestFiredAt) {
			g.LatestFiredAt = r.LatestFired
		}
		if r.OldestFired.Before(g.OldestFiredAt) {
			g.OldestFiredAt = r.OldestFired
		}
		if r.MaxFireCount > g.MaxFireCount {
			g.MaxFireCount = r.MaxFireCount
		}
	}

	result := make([]AlertGroupItem, 0, len(order))
	for _, k := range order {
		result = append(result, *groups[k])
	}
	Success(c, result)
}

// splitCSV splits a comma-separated string into trimmed non-empty parts.
func splitCSV(s string) []string {
	if s == "" {
		return nil
	}
	parts := make([]string, 0)
	start := 0
	for i := 0; i <= len(s); i++ {
		if i == len(s) || s[i] == ',' {
			p := s[start:i]
			if len(p) > 0 {
				parts = append(parts, p)
			}
			start = i + 1
		}
	}
	return parts
}

// WebhookReceive handles incoming alert webhooks (AlertManager compatible).
func (h *AlertEventHandler) WebhookReceive(c *gin.Context) {
	var payload model.AlertManagerPayload
	if err := c.ShouldBindJSON(&payload); err != nil {
		Error(c, apperr.WithMessage(apperr.ErrInvalidParam, err.Error()))
		return
	}

	if err := h.svc.ProcessWebhook(c.Request.Context(), &payload); err != nil {
		Error(c, err)
		return
	}

	Success(c, nil)
}
