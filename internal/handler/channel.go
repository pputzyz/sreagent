package handler

import (
	"github.com/gin-gonic/gin"

	"github.com/sreagent/sreagent/internal/model"
	apperr "github.com/sreagent/sreagent/internal/pkg/errors"
	"github.com/sreagent/sreagent/internal/service"
)

// ChannelHandler handles HTTP requests for collaboration channels (协作空间).
type ChannelHandler struct {
	svc      *service.ChannelService
	auditSvc *service.AuditLogService
}

func NewChannelHandler(svc *service.ChannelService) *ChannelHandler {
	return &ChannelHandler{svc: svc}
}

// SetAuditService injects the audit log service (called after construction to avoid circular DI).
func (h *ChannelHandler) SetAuditService(svc *service.AuditLogService) {
	h.auditSvc = svc
}

// --- Request structs ---

type CreateCollabChannelRequest struct {
	Name              string `json:"name" binding:"required"`
	Description       string `json:"description"`
	TeamID            *uint  `json:"team_id"`
	Status            string `json:"status"`
	AccessLevel       string `json:"access_level"`
	AggregationConfig string `json:"aggregation_config"`
	FlappingConfig    string `json:"flapping_config"`
	AutoCloseEnabled  bool   `json:"auto_close_enabled"`
	AutoCloseOrigin   string `json:"auto_close_origin"`
	AutoCloseMinutes  int    `json:"auto_close_minutes"`
	FollowAlertClose  bool   `json:"follow_alert_close"`
	SortOrder         int    `json:"sort_order"`
}

type UpdateCollabChannelRequest struct {
	Name              string `json:"name"`
	Description       string `json:"description"`
	TeamID            *uint  `json:"team_id"`
	Status            string `json:"status"`
	AccessLevel       string `json:"access_level"`
	AggregationConfig string `json:"aggregation_config"`
	FlappingConfig    string `json:"flapping_config"`
	AutoCloseEnabled  bool   `json:"auto_close_enabled"`
	AutoCloseOrigin   string `json:"auto_close_origin"`
	AutoCloseMinutes  int    `json:"auto_close_minutes"`
	FollowAlertClose  bool   `json:"follow_alert_close"`
	SortOrder         int    `json:"sort_order"`
}

// --- Endpoints ---

// Create creates a new collaboration channel.
// POST /api/v1/channels
func (h *ChannelHandler) Create(c *gin.Context) {
	var req CreateCollabChannelRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		Error(c, apperr.WithMessage(apperr.ErrInvalidParam, err.Error()))
		return
	}

	ch := &model.Channel{
		Name:              req.Name,
		Description:       req.Description,
		TeamID:            req.TeamID,
		Status:            model.ChannelStatus(req.Status),
		AccessLevel:       model.ChannelAccessLevel(req.AccessLevel),
		AggregationConfig: req.AggregationConfig,
		FlappingConfig:    req.FlappingConfig,
		AutoCloseEnabled:  req.AutoCloseEnabled,
		AutoCloseOrigin:   req.AutoCloseOrigin,
		AutoCloseMinutes:  req.AutoCloseMinutes,
		FollowAlertClose:  req.FollowAlertClose,
		SortOrder:         req.SortOrder,
	}

	// Defaults
	if ch.Status == "" {
		ch.Status = model.ChannelStatusActive
	}
	if ch.AccessLevel == "" {
		ch.AccessLevel = model.ChannelAccessPublic
	}

	if err := h.svc.Create(c.Request.Context(), ch); err != nil {
		Error(c, err)
		return
	}

	if h.auditSvc != nil {
		uid := GetCurrentUserID(c)
		h.auditSvc.Record(&model.AuditLog{
			UserID: &uid, Action: model.AuditActionCreate,
			ResourceType: model.AuditResourceChannel, ResourceID: &ch.ID, ResourceName: ch.Name,
			IP: c.ClientIP(),
		})
	}

	Success(c, ch)
}

// Get returns a single channel by ID.
// GET /api/v1/channels/:id
func (h *ChannelHandler) Get(c *gin.Context) {
	id, err := GetIDParam(c, "id")
	if err != nil {
		Error(c, err)
		return
	}

	ch, err := h.svc.GetByID(c.Request.Context(), id)
	if err != nil {
		Error(c, err)
		return
	}

	Success(c, ch)
}

// List returns a paginated list of channels.
// GET /api/v1/channels?query=&status=&page=&page_size=
func (h *ChannelHandler) List(c *gin.Context) {
	pq := GetPageQuery(c)
	query := c.Query("query")
	status := c.Query("status")

	list, total, err := h.svc.List(c.Request.Context(), query, status, pq.Page, pq.PageSize)
	if err != nil {
		Error(c, err)
		return
	}

	// Enrich with starred info for current user
	userID := GetCurrentUserID(c)
	if userID != 0 {
		starred, _ := h.svc.ListStarred(c.Request.Context(), userID)
		starredSet := make(map[uint]bool, len(starred))
		for _, sid := range starred {
			starredSet[sid] = true
		}
		type ChannelWithStar struct {
			model.Channel
			IsStarred bool `json:"is_starred"`
		}
		enriched := make([]ChannelWithStar, len(list))
		for i, ch := range list {
			enriched[i] = ChannelWithStar{Channel: ch, IsStarred: starredSet[ch.ID]}
		}
		SuccessPage(c, enriched, total, pq.Page, pq.PageSize)
		return
	}

	SuccessPage(c, list, total, pq.Page, pq.PageSize)
}

// Update updates a channel.
// PUT /api/v1/channels/:id
func (h *ChannelHandler) Update(c *gin.Context) {
	id, err := GetIDParam(c, "id")
	if err != nil {
		Error(c, err)
		return
	}

	var req UpdateCollabChannelRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		Error(c, apperr.WithMessage(apperr.ErrInvalidParam, err.Error()))
		return
	}

	updates := &model.Channel{
		Name:              req.Name,
		Description:       req.Description,
		TeamID:            req.TeamID,
		Status:            model.ChannelStatus(req.Status),
		AccessLevel:       model.ChannelAccessLevel(req.AccessLevel),
		AggregationConfig: req.AggregationConfig,
		FlappingConfig:    req.FlappingConfig,
		AutoCloseEnabled:  req.AutoCloseEnabled,
		AutoCloseOrigin:   req.AutoCloseOrigin,
		AutoCloseMinutes:  req.AutoCloseMinutes,
		FollowAlertClose:  req.FollowAlertClose,
		SortOrder:         req.SortOrder,
	}

	ch, err := h.svc.Update(c.Request.Context(), id, updates)
	if err != nil {
		Error(c, err)
		return
	}

	if h.auditSvc != nil {
		uid := GetCurrentUserID(c)
		h.auditSvc.Record(&model.AuditLog{
			UserID: &uid, Action: model.AuditActionUpdate,
			ResourceType: model.AuditResourceChannel, ResourceID: &id, ResourceName: req.Name,
			IP: c.ClientIP(),
		})
	}

	Success(c, ch)
}

// Delete soft-deletes a channel.
// DELETE /api/v1/channels/:id
func (h *ChannelHandler) Delete(c *gin.Context) {
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
			UserID: &uid, Action: model.AuditActionDelete,
			ResourceType: model.AuditResourceChannel, ResourceID: &id,
			IP: c.ClientIP(),
		})
	}

	Success(c, nil)
}

// Star marks a channel as favorite for the current user.
// POST /api/v1/channels/:id/star
func (h *ChannelHandler) Star(c *gin.Context) {
	id, err := GetIDParam(c, "id")
	if err != nil {
		Error(c, err)
		return
	}

	userID := GetCurrentUserID(c)
	if userID == 0 {
		Error(c, apperr.WithMessage(apperr.ErrUnauthorized, "unauthorized"))
		return
	}

	if err := h.svc.Star(c.Request.Context(), userID, id); err != nil {
		Error(c, err)
		return
	}

	Success(c, nil)
}

// Unstar removes a channel from the current user's favorites.
// DELETE /api/v1/channels/:id/star
func (h *ChannelHandler) Unstar(c *gin.Context) {
	id, err := GetIDParam(c, "id")
	if err != nil {
		Error(c, err)
		return
	}

	userID := GetCurrentUserID(c)
	if userID == 0 {
		Error(c, apperr.WithMessage(apperr.ErrUnauthorized, "unauthorized"))
		return
	}

	if err := h.svc.Unstar(c.Request.Context(), userID, id); err != nil {
		Error(c, err)
		return
	}

	Success(c, nil)
}
