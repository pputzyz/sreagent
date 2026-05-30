package handler

import (
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/sreagent/sreagent/internal/service"
)

// AuditLogHandler handles audit log read endpoints.
type AuditLogHandler struct {
	svc *service.AuditLogService
}

func NewAuditLogHandler(svc *service.AuditLogService) *AuditLogHandler {
	return &AuditLogHandler{svc: svc}
}

// List returns a paginated list of audit log entries.
// GET /api/v1/audit-logs
// Query params: page, page_size, user_id, action, resource_type, start_time, end_time (RFC3339)
func (h *AuditLogHandler) List(c *gin.Context) {
	pq := GetPageQuery(c)

	f := service.AuditLogFilter{}

	if v := c.Query("user_id"); v != "" {
		if uid, err := strconv.ParseUint(v, 10, 64); err == nil {
			u := uint(uid)
			f.UserID = &u
		}
	}
	if v := c.Query("action"); v != "" {
		f.Action = v
	}
	if v := c.Query("resource_type"); v != "" {
		f.ResourceType = v
	}
	if v := c.Query("start_time"); v != "" {
		if t, err := time.Parse(time.RFC3339, v); err == nil {
			f.StartTime = &t
		}
	}
	if v := c.Query("end_time"); v != "" {
		if t, err := time.Parse(time.RFC3339, v); err == nil {
			f.EndTime = &t
		}
	}

	logs, total, err := h.svc.List(c.Request.Context(), f, pq.Page, pq.PageSize)
	if err != nil {
		Error(c, err)
		return
	}

	SuccessPage(c, logs, total, pq.Page, pq.PageSize)
}
