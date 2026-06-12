package handler

import (
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/sreagent/sreagent/internal/model"
	apperr "github.com/sreagent/sreagent/internal/pkg/errors"
	"github.com/sreagent/sreagent/internal/service"
)

// AlertForwarderHandler handles HTTP requests for alert forwarders.
type AlertForwarderHandler struct {
	svc      *service.AlertForwarderService
	auditSvc *service.AuditLogService
	log      *zap.Logger
}

// NewAlertForwarderHandler creates a new AlertForwarderHandler.
func NewAlertForwarderHandler(svc *service.AlertForwarderService, logger ...*zap.Logger) *AlertForwarderHandler {
	l := zap.NewNop()
	if len(logger) > 0 && logger[0] != nil {
		l = logger[0]
	}
	return &AlertForwarderHandler{svc: svc, log: l}
}

// SetAuditService injects the audit log service.
func (h *AlertForwarderHandler) SetAuditService(svc *service.AuditLogService) {
	h.auditSvc = svc
}

// CreateAlertForwarderRequest is the request body for creating an alert forwarder.
type CreateAlertForwarderRequest struct {
	Name                     string                          `json:"name" binding:"required"`
	Description              string                          `json:"description"`
	Enabled                  *bool                           `json:"enabled"`
	Direction                string                          `json:"direction" binding:"required"`
	Priority                 int                             `json:"priority"`
	InboundConfig            *model.InboundConfig            `json:"inbound_config"`
	OutboundConfig           *model.OutboundConfig           `json:"outbound_config"`
	InboundSeverityMapping   *model.SeverityMappingConfig    `json:"inbound_severity_mapping"`
	OutboundSeverityMapping  *model.SeverityMappingConfig    `json:"outbound_severity_mapping"`
	PlatformCapabilities     *model.PlatformCapabilitiesConfig `json:"platform_capabilities"`
	MatchLabels              model.JSONLabels                `json:"match_labels"`
}

// UpdateAlertForwarderRequest is the request body for updating an alert forwarder.
type UpdateAlertForwarderRequest struct {
	Name                     string                          `json:"name" binding:"required"`
	Description              string                          `json:"description"`
	Enabled                  *bool                           `json:"enabled"`
	Direction                string                          `json:"direction" binding:"required"`
	Priority                 int                             `json:"priority"`
	InboundConfig            *model.InboundConfig            `json:"inbound_config"`
	OutboundConfig           *model.OutboundConfig           `json:"outbound_config"`
	InboundSeverityMapping   *model.SeverityMappingConfig    `json:"inbound_severity_mapping"`
	OutboundSeverityMapping  *model.SeverityMappingConfig    `json:"outbound_severity_mapping"`
	PlatformCapabilities     *model.PlatformCapabilitiesConfig `json:"platform_capabilities"`
	MatchLabels              model.JSONLabels                `json:"match_labels"`
}

// BatchRequest is the request body for batch operations.
type BatchRequest struct {
	IDs []uint `json:"ids" binding:"required"`
}

// Create creates a new alert forwarder.
func (h *AlertForwarderHandler) Create(c *gin.Context) {
	var req CreateAlertForwarderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		Error(c, apperr.WithMessage(apperr.ErrInvalidParam, err.Error()))
		return
	}

	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}

	userID := GetCurrentUserID(c)
	h.log.Info("alert forwarder create",
		zap.Uint("user_id", userID),
		zap.String("name", req.Name),
		zap.String("direction", req.Direction),
	)

	forwarder := &model.AlertForwarder{
		Name:                 req.Name,
		Description:          req.Description,
		Enabled:              enabled,
		Direction:            model.ForwarderDirection(req.Direction),
		Priority:             req.Priority,
		InboundConfig:            req.InboundConfig,
		OutboundConfig:           req.OutboundConfig,
		InboundSeverityMapping:   req.InboundSeverityMapping,
		OutboundSeverityMapping:  req.OutboundSeverityMapping,
		PlatformCapabilities:     req.PlatformCapabilities,
		MatchLabels:              req.MatchLabels,
	}

	if err := h.svc.Create(c.Request.Context(), forwarder); err != nil {
		Error(c, err)
		return
	}

	Success(c, forwarder)
}

// GetByID returns an alert forwarder by its ID.
func (h *AlertForwarderHandler) GetByID(c *gin.Context) {
	id, err := GetIDParam(c, "id")
	if err != nil {
		Error(c, apperr.WithMessage(apperr.ErrInvalidParam, "invalid id"))
		return
	}

	forwarder, err := h.svc.GetByID(c.Request.Context(), id)
	if err != nil {
		Error(c, err)
		return
	}

	Success(c, forwarder)
}

// List returns a paginated list of alert forwarders.
func (h *AlertForwarderHandler) List(c *gin.Context) {
	pq := GetPageQuery(c)
	page, pageSize := pq.Page, pq.PageSize
	direction := c.Query("direction")

	var enabled *bool
	if enabledStr := c.Query("enabled"); enabledStr != "" {
		e := enabledStr == "true"
		enabled = &e
	}

	forwarders, total, err := h.svc.List(c.Request.Context(), page, pageSize, direction, enabled)
	if err != nil {
		Error(c, err)
		return
	}

	SuccessPage(c, forwarders, total, page, pageSize)
}

// Update updates an existing alert forwarder.
func (h *AlertForwarderHandler) Update(c *gin.Context) {
	id, err := GetIDParam(c, "id")
	if err != nil {
		Error(c, apperr.WithMessage(apperr.ErrInvalidParam, "invalid id"))
		return
	}

	var req UpdateAlertForwarderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		Error(c, apperr.WithMessage(apperr.ErrInvalidParam, err.Error()))
		return
	}

	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}

	userID := GetCurrentUserID(c)
	h.log.Info("alert forwarder update",
		zap.Uint("user_id", userID),
		zap.Uint("id", id),
		zap.String("name", req.Name),
	)

	forwarder := &model.AlertForwarder{
		ID:                   id,
		Name:                 req.Name,
		Description:          req.Description,
		Enabled:              enabled,
		Direction:            model.ForwarderDirection(req.Direction),
		Priority:             req.Priority,
		InboundConfig:            req.InboundConfig,
		OutboundConfig:           req.OutboundConfig,
		InboundSeverityMapping:   req.InboundSeverityMapping,
		OutboundSeverityMapping:  req.OutboundSeverityMapping,
		PlatformCapabilities:     req.PlatformCapabilities,
		MatchLabels:              req.MatchLabels,
	}

	if err := h.svc.Update(c.Request.Context(), forwarder); err != nil {
		Error(c, err)
		return
	}

	Success(c, forwarder)
}

// Delete deletes an alert forwarder.
func (h *AlertForwarderHandler) Delete(c *gin.Context) {
	id, err := GetIDParam(c, "id")
	if err != nil {
		Error(c, apperr.WithMessage(apperr.ErrInvalidParam, "invalid id"))
		return
	}

	userID := GetCurrentUserID(c)
	h.log.Info("alert forwarder delete",
		zap.Uint("user_id", userID),
		zap.Uint("id", id),
	)

	if err := h.svc.Delete(c.Request.Context(), id); err != nil {
		Error(c, err)
		return
	}

	Success(c, nil)
}

// Enable enables an alert forwarder.
func (h *AlertForwarderHandler) Enable(c *gin.Context) {
	id, err := GetIDParam(c, "id")
	if err != nil {
		Error(c, apperr.WithMessage(apperr.ErrInvalidParam, "invalid id"))
		return
	}

	userID := GetCurrentUserID(c)
	h.log.Info("alert forwarder enable",
		zap.Uint("user_id", userID),
		zap.Uint("id", id),
	)

	if err := h.svc.Enable(c.Request.Context(), id); err != nil {
		Error(c, err)
		return
	}

	Success(c, nil)
}

// Disable disables an alert forwarder.
func (h *AlertForwarderHandler) Disable(c *gin.Context) {
	id, err := GetIDParam(c, "id")
	if err != nil {
		Error(c, apperr.WithMessage(apperr.ErrInvalidParam, "invalid id"))
		return
	}

	userID := GetCurrentUserID(c)
	h.log.Info("alert forwarder disable",
		zap.Uint("user_id", userID),
		zap.Uint("id", id),
	)

	if err := h.svc.Disable(c.Request.Context(), id); err != nil {
		Error(c, err)
		return
	}

	Success(c, nil)
}

// BatchEnable enables multiple alert forwarders.
func (h *AlertForwarderHandler) BatchEnable(c *gin.Context) {
	var req BatchRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		Error(c, apperr.WithMessage(apperr.ErrInvalidParam, err.Error()))
		return
	}

	userID := GetCurrentUserID(c)
	h.log.Info("alert forwarder batch enable",
		zap.Uint("user_id", userID),
		zap.Int("count", len(req.IDs)),
	)

	if err := h.svc.BatchEnable(c.Request.Context(), req.IDs); err != nil {
		Error(c, err)
		return
	}

	Success(c, nil)
}

// BatchDisable disables multiple alert forwarders.
func (h *AlertForwarderHandler) BatchDisable(c *gin.Context) {
	var req BatchRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		Error(c, apperr.WithMessage(apperr.ErrInvalidParam, err.Error()))
		return
	}

	userID := GetCurrentUserID(c)
	h.log.Info("alert forwarder batch disable",
		zap.Uint("user_id", userID),
		zap.Int("count", len(req.IDs)),
	)

	if err := h.svc.BatchDisable(c.Request.Context(), req.IDs); err != nil {
		Error(c, err)
		return
	}

	Success(c, nil)
}

// BatchDelete deletes multiple alert forwarders.
func (h *AlertForwarderHandler) BatchDelete(c *gin.Context) {
	var req BatchRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		Error(c, apperr.WithMessage(apperr.ErrInvalidParam, err.Error()))
		return
	}

	userID := GetCurrentUserID(c)
	h.log.Info("alert forwarder batch delete",
		zap.Uint("user_id", userID),
		zap.Int("count", len(req.IDs)),
	)

	if err := h.svc.BatchDelete(c.Request.Context(), req.IDs); err != nil {
		Error(c, err)
		return
	}

	Success(c, nil)
}

// GetStats returns statistics about alert forwarders.
func (h *AlertForwarderHandler) GetStats(c *gin.Context) {
	stats, err := h.svc.GetStats(c.Request.Context())
	if err != nil {
		Error(c, err)
		return
	}

	Success(c, stats)
}

// TestForwarder tests a forwarder configuration.
func (h *AlertForwarderHandler) TestForwarder(c *gin.Context) {
	id, err := GetIDParam(c, "id")
	if err != nil {
		Error(c, apperr.WithMessage(apperr.ErrInvalidParam, "invalid id"))
		return
	}

	userID := GetCurrentUserID(c)
	h.log.Info("alert forwarder test",
		zap.Uint("user_id", userID),
		zap.Uint("id", id),
	)

	result, err := h.svc.TestForwarder(c.Request.Context(), id)
	if err != nil {
		Error(c, err)
		return
	}

	Success(c, result)
}

// HandleInbound handles inbound alert webhook requests.
func (h *AlertForwarderHandler) HandleInbound(c *gin.Context) {
	id, err := GetIDParam(c, "id")
	if err != nil {
		Error(c, apperr.WithMessage(apperr.ErrInvalidParam, "invalid id"))
		return
	}

	h.log.Info("inbound alert received",
		zap.Uint("forwarder_id", id),
		zap.String("remote_addr", c.ClientIP()),
	)

	if err := h.svc.ProcessInbound(c.Request.Context(), id, c.Request); err != nil {
		Error(c, err)
		return
	}

	Success(c, gin.H{"status": "accepted"})
}

