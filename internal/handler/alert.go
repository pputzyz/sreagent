package handler

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/sreagent/sreagent/internal/service"
)

// AlertV2Handler handles HTTP requests for the v2 Alert model.
type AlertV2Handler struct {
	svc *service.AlertV2Service
}

func NewAlertV2Handler(svc *service.AlertV2Service) *AlertV2Handler {
	return &AlertV2Handler{svc: svc}
}

// Get returns a single alert by ID.
// GET /api/v1/alerts/:id
func (h *AlertV2Handler) Get(c *gin.Context) {
	id, err := GetIDParam(c, "id")
	if err != nil {
		Error(c, err)
		return
	}

	alert, err := h.svc.GetByID(c.Request.Context(), id)
	if err != nil {
		Error(c, err)
		return
	}

	Success(c, alert)
}

// List returns a paginated list of alerts.
// GET /api/v1/alerts?channel_id=&incident_id=&status=&severity=&query=&page=&page_size=
func (h *AlertV2Handler) List(c *gin.Context) {
	pq := GetPageQuery(c)
	var channelID, incidentID uint
	if v := c.Query("channel_id"); v != "" {
		if id, err := strconv.ParseUint(v, 10, 64); err == nil {
			channelID = uint(id)
		}
	}
	if v := c.Query("incident_id"); v != "" {
		if id, err := strconv.ParseUint(v, 10, 64); err == nil {
			incidentID = uint(id)
		}
	}
	status := c.Query("status")
	severity := c.Query("severity")
	query := c.Query("query")

	list, total, err := h.svc.List(c.Request.Context(), channelID, incidentID, status, severity, query, pq.Page, pq.PageSize)
	if err != nil {
		Error(c, err)
		return
	}

	SuccessPage(c, list, total, pq.Page, pq.PageSize)
}

// ListEvents returns paginated events for an alert.
// GET /api/v1/alerts/:id/events?page=&page_size=
func (h *AlertV2Handler) ListEvents(c *gin.Context) {
	id, err := GetIDParam(c, "id")
	if err != nil {
		Error(c, err)
		return
	}

	pq := GetPageQuery(c)
	list, total, err := h.svc.ListEvents(c.Request.Context(), id, pq.Page, pq.PageSize)
	if err != nil {
		Error(c, err)
		return
	}

	SuccessPage(c, list, total, pq.Page, pq.PageSize)
}
