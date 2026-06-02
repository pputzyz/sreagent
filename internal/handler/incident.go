package handler

import (
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/sreagent/sreagent/internal/middleware"
	"github.com/sreagent/sreagent/internal/model"
	apperr "github.com/sreagent/sreagent/internal/pkg/errors"
	"github.com/sreagent/sreagent/internal/service"
)

// IncidentHandler handles HTTP requests for incidents (故障).
type IncidentHandler struct {
	svc      *service.IncidentService
	auditSvc *service.AuditLogService
}

func NewIncidentHandler(svc *service.IncidentService) *IncidentHandler {
	return &IncidentHandler{svc: svc}
}

// SetAuditService injects the audit log service (called after construction to avoid circular DI).
func (h *IncidentHandler) SetAuditService(svc *service.AuditLogService) {
	h.auditSvc = svc
}

// --- Request structs ---

type CreateIncidentRequest struct {
	Title       string `json:"title" binding:"required"`
	Description string `json:"description"`
	Severity    string `json:"severity"`
	ChannelID   uint   `json:"channel_id"`
	AssignedTo  *uint  `json:"assigned_to"`
}

type SnoozeIncidentRequest struct {
	Until string `json:"until" binding:"required"` // RFC3339
}

type ReassignIncidentRequest struct {
	UserID uint `json:"user_id" binding:"required"`
}

type MergeIncidentRequest struct {
	TargetID uint `json:"target_id" binding:"required"`
}

type CommentRequest struct {
	Content string `json:"content" binding:"required"`
}

type BulkIDsRequest struct {
	IDs []uint `json:"ids" binding:"required"`
}

// --- Endpoints ---

// Create creates a new incident.
// POST /api/v1/incidents
func (h *IncidentHandler) Create(c *gin.Context) {
	var req CreateIncidentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		Error(c, apperr.WithMessage(apperr.ErrInvalidParam, err.Error()))
		return
	}

	// --- Parameter validation ---
	if req.ChannelID == 0 {
		Error(c, apperr.WithMessage(apperr.ErrInvalidParam, "channel_id is required"))
		return
	}
	if len(req.Title) > 256 {
		Error(c, apperr.WithMessage(apperr.ErrInvalidParam, "title must not exceed 256 characters"))
		return
	}
	if req.Severity != "" {
		sev := model.IncidentSeverity(req.Severity)
		if !sev.IsValid() {
			Error(c, apperr.WithMessage(apperr.ErrInvalidParam, "severity must be one of: critical, warning, info"))
			return
		}
	}

	inc := &model.Incident{
		Title:       req.Title,
		Description: req.Description,
		Severity:    model.IncidentSeverity(req.Severity),
		ChannelID:   req.ChannelID,
		AssignedTo:  req.AssignedTo,
		TriggeredAt: time.Now(),
	}
	if inc.Severity == "" {
		inc.Severity = model.IncidentSeverityWarning
	}

	if err := h.svc.Create(c.Request.Context(), inc); err != nil {
		Error(c, err)
		return
	}

	if h.auditSvc != nil {
		uid := GetCurrentUserID(c)
		h.auditSvc.Record(&model.AuditLog{
			UserID: &uid, Username: GetCurrentUsername(c),
			Action: model.AuditActionCreate, ResourceType: model.AuditResourceIncident,
			ResourceID: &inc.ID, ResourceName: inc.Title, IP: c.ClientIP(),
		})
	}

	Success(c, inc)
}

// Get returns a single incident by ID.
// GET /api/v1/incidents/:id
func (h *IncidentHandler) Get(c *gin.Context) {
	id, err := GetIDParam(c, "id")
	if err != nil {
		Error(c, err)
		return
	}

	inc, err := h.svc.GetByID(c.Request.Context(), id)
	if err != nil {
		Error(c, err)
		return
	}

	Success(c, inc)
}

// List returns a paginated list of incidents.
// GET /api/v1/incidents?channel_id=&status=&severity=&query=&assigned_to=&page=&page_size=
func (h *IncidentHandler) List(c *gin.Context) {
	pq := GetPageQuery(c)
	var channelID, assignedTo uint
	if v := c.Query("channel_id"); v != "" {
		if id, err := strconv.ParseUint(v, 10, 64); err == nil {
			channelID = uint(id)
		}
	}
	if v := c.Query("assigned_to"); v != "" {
		if id, err := strconv.ParseUint(v, 10, 64); err == nil {
			assignedTo = uint(id)
		}
	}
	status := c.Query("status")
	severity := c.Query("severity")
	query := c.Query("query")

	// Team-scoped listing: admin sees all, non-admin sees only own team's incidents.
	currentRole, _ := c.Get("role")
	isAdmin := currentRole == "admin"
	teamIDs := middleware.GetUserTeamIDs(c)

	list, total, err := h.svc.ListScoped(c.Request.Context(), isAdmin, teamIDs, channelID, status, severity, query, assignedTo, pq.Page, pq.PageSize)
	if err != nil {
		Error(c, err)
		return
	}

	SuccessPage(c, list, total, pq.Page, pq.PageSize)
}

// Acknowledge acknowledges an incident.
// POST /api/v1/incidents/:id/acknowledge
func (h *IncidentHandler) Acknowledge(c *gin.Context) {
	id, err := GetIDParam(c, "id")
	if err != nil {
		Error(c, err)
		return
	}

	userID := GetCurrentUserID(c)
	if err := h.svc.Acknowledge(c.Request.Context(), id, userID); err != nil {
		Error(c, err)
		return
	}

	if h.auditSvc != nil {
		h.auditSvc.Record(&model.AuditLog{
			UserID: &userID, Username: GetCurrentUsername(c),
			Action: model.AuditActionAck, ResourceType: model.AuditResourceIncident,
			ResourceID: &id, IP: c.ClientIP(),
		})
	}

	Success(c, nil)
}

// Close closes an incident.
// POST /api/v1/incidents/:id/close
func (h *IncidentHandler) Close(c *gin.Context) {
	id, err := GetIDParam(c, "id")
	if err != nil {
		Error(c, err)
		return
	}

	userID := GetCurrentUserID(c)
	if err := h.svc.Close(c.Request.Context(), id, userID); err != nil {
		Error(c, err)
		return
	}

	if h.auditSvc != nil {
		h.auditSvc.Record(&model.AuditLog{
			UserID: &userID, Username: GetCurrentUsername(c),
			Action: model.AuditActionClose, ResourceType: model.AuditResourceIncident,
			ResourceID: &id, IP: c.ClientIP(),
		})
	}

	Success(c, nil)
}

// Reopen re-opens a closed incident.
// POST /api/v1/incidents/:id/reopen
func (h *IncidentHandler) Reopen(c *gin.Context) {
	id, err := GetIDParam(c, "id")
	if err != nil {
		Error(c, err)
		return
	}

	userID := GetCurrentUserID(c)
	if err := h.svc.Reopen(c.Request.Context(), id, userID); err != nil {
		Error(c, err)
		return
	}

	if h.auditSvc != nil {
		h.auditSvc.Record(&model.AuditLog{
			UserID: &userID, Username: GetCurrentUsername(c),
			Action: model.AuditActionReopen, ResourceType: model.AuditResourceIncident,
			ResourceID: &id, IP: c.ClientIP(),
		})
	}

	Success(c, nil)
}

// Snooze pauses an incident until a specified time.
// POST /api/v1/incidents/:id/snooze
func (h *IncidentHandler) Snooze(c *gin.Context) {
	id, err := GetIDParam(c, "id")
	if err != nil {
		Error(c, err)
		return
	}

	var req SnoozeIncidentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		Error(c, apperr.WithMessage(apperr.ErrInvalidParam, err.Error()))
		return
	}

	until, err := time.Parse(time.RFC3339, req.Until)
	if err != nil {
		Error(c, apperr.WithMessage(apperr.ErrInvalidParam, "invalid time format, must be RFC3339"))
		return
	}

	userID := GetCurrentUserID(c)
	if err := h.svc.Snooze(c.Request.Context(), id, userID, until); err != nil {
		Error(c, err)
		return
	}

	Success(c, nil)
}

// Reassign reassigns an incident to a different user.
// POST /api/v1/incidents/:id/reassign
func (h *IncidentHandler) Reassign(c *gin.Context) {
	id, err := GetIDParam(c, "id")
	if err != nil {
		Error(c, err)
		return
	}

	var req ReassignIncidentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		Error(c, apperr.WithMessage(apperr.ErrInvalidParam, err.Error()))
		return
	}

	userID := GetCurrentUserID(c)
	if err := h.svc.Reassign(c.Request.Context(), id, userID, req.UserID); err != nil {
		Error(c, err)
		return
	}

	Success(c, nil)
}

// Merge merges this incident into another one.
// POST /api/v1/incidents/:id/merge
func (h *IncidentHandler) Merge(c *gin.Context) {
	id, err := GetIDParam(c, "id")
	if err != nil {
		Error(c, err)
		return
	}

	var req MergeIncidentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		Error(c, apperr.WithMessage(apperr.ErrInvalidParam, err.Error()))
		return
	}

	userID := GetCurrentUserID(c)
	if err := h.svc.Merge(c.Request.Context(), id, req.TargetID, userID); err != nil {
		Error(c, err)
		return
	}

	Success(c, nil)
}

// Escalate escalates the incident to the next step.
// POST /api/v1/incidents/:id/escalate
func (h *IncidentHandler) Escalate(c *gin.Context) {
	id, err := GetIDParam(c, "id")
	if err != nil {
		Error(c, err)
		return
	}

	userID := GetCurrentUserID(c)
	if err := h.svc.Escalate(c.Request.Context(), id, userID); err != nil {
		Error(c, err)
		return
	}

	Success(c, nil)
}

// GetTimeline returns the timeline for an incident.
// GET /api/v1/incidents/:id/timeline
func (h *IncidentHandler) GetTimeline(c *gin.Context) {
	id, err := GetIDParam(c, "id")
	if err != nil {
		Error(c, err)
		return
	}

	list, err := h.svc.ListTimeline(c.Request.Context(), id)
	if err != nil {
		Error(c, err)
		return
	}

	Success(c, list)
}

// BulkAcknowledge acknowledges multiple incidents.
// POST /api/v1/incidents/bulk-acknowledge
func (h *IncidentHandler) BulkAcknowledge(c *gin.Context) {
	var req BulkIDsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		Error(c, apperr.WithMessage(apperr.ErrInvalidParam, err.Error()))
		return
	}
	if len(req.IDs) == 0 {
		Error(c, apperr.WithMessage(apperr.ErrInvalidParam, "ids is required"))
		return
	}

	userID := GetCurrentUserID(c)
	if err := h.svc.BulkAcknowledge(c.Request.Context(), req.IDs, userID); err != nil {
		Error(c, err)
		return
	}

	Success(c, nil)
}

// BulkClose closes multiple incidents.
// POST /api/v1/incidents/bulk-close
func (h *IncidentHandler) BulkClose(c *gin.Context) {
	var req BulkIDsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		Error(c, apperr.WithMessage(apperr.ErrInvalidParam, err.Error()))
		return
	}
	if len(req.IDs) == 0 {
		Error(c, apperr.WithMessage(apperr.ErrInvalidParam, "ids is required"))
		return
	}

	userID := GetCurrentUserID(c)
	if err := h.svc.BulkClose(c.Request.Context(), req.IDs, userID); err != nil {
		Error(c, err)
		return
	}

	Success(c, nil)
}

// AddComment adds a comment to the incident timeline.
// POST /api/v1/incidents/:id/comment
func (h *IncidentHandler) AddComment(c *gin.Context) {
	id, err := GetIDParam(c, "id")
	if err != nil {
		Error(c, err)
		return
	}

	var req CommentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		Error(c, apperr.WithMessage(apperr.ErrInvalidParam, err.Error()))
		return
	}

	userID := GetCurrentUserID(c)
	if err := h.svc.AddComment(c.Request.Context(), id, userID, req.Content); err != nil {
		Error(c, err)
		return
	}

	Success(c, nil)
}
