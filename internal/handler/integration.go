package handler

import (
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
	apperr "github.com/sreagent/sreagent/internal/pkg/errors"
	"go.uber.org/zap"

	"github.com/sreagent/sreagent/internal/model"
	"github.com/sreagent/sreagent/internal/service"
)

// IntegrationHandler manages webhook integrations.
type IntegrationHandler struct {
	svc *service.IntegrationService
	log *zap.Logger
}

func NewIntegrationHandler(svc *service.IntegrationService, logger ...*zap.Logger) *IntegrationHandler {
	l := zap.NewNop()
	if len(logger) > 0 && logger[0] != nil {
		l = logger[0]
	}
	return &IntegrationHandler{svc: svc, log: l}
}

type CreateIntegrationRequest struct {
	Name                   string `json:"name" binding:"required"`
	Description            string `json:"description"`
	Type                   string `json:"type" binding:"required"` // standard | alertmanager | grafana
	Mode                   string `json:"mode"`                    // exclusive | shared
	ChannelID              *uint  `json:"channel_id"`
	PipelineConfig         string `json:"pipeline_config"`
	LabelEnhancementConfig string `json:"label_enhancement_config"`
	IsEnabled              *bool  `json:"is_enabled"`
}

// List returns integrations, optionally filtered by channel.
// GET /api/v1/integrations?channel_id=&page=&page_size=
func (h *IntegrationHandler) List(c *gin.Context) {
	pq := GetPageQuery(c)
	var channelID uint
	if id, err := GetIDParam(c, "channel_id"); err == nil {
		channelID = id
	}
	// Also accept query param
	if v := c.Query("channel_id"); v != "" {
		var cid uint
		if _, err := parseUintStr(v, &cid); err == nil {
			channelID = cid
		}
	}

	list, total, err := h.svc.List(c.Request.Context(), channelID, pq.Page, pq.PageSize)
	if err != nil {
		Error(c, err)
		return
	}
	SuccessPage(c, list, total, pq.Page, pq.PageSize)
}

// Create creates a new integration.
// POST /api/v1/integrations
func (h *IntegrationHandler) Create(c *gin.Context) {
	var req CreateIntegrationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		Error(c, apperr.WithMessage(apperr.ErrInvalidParam, err.Error()))
		return
	}

	// Validate integration type
	validTypes := map[string]bool{
		string(model.IntegrationTypeStandard):     true,
		string(model.IntegrationTypeAlertManager): true,
		string(model.IntegrationTypeGrafana):      true,
	}
	if !validTypes[req.Type] {
		Error(c, apperr.WithMessage(apperr.ErrInvalidParam, "unsupported integration type: must be one of standard, alertmanager, grafana"))
		return
	}

	mode := model.IntegrationMode(req.Mode)
	if mode == "" {
		mode = model.IntegrationModeExclusive
	}
	if mode != model.IntegrationModeExclusive && mode != model.IntegrationModeShared {
		Error(c, apperr.WithMessage(apperr.ErrInvalidParam, "unsupported integration mode: must be one of exclusive, shared"))
		return
	}

	h.log.Info("integration create",
		zap.Uint("user_id", GetCurrentUserID(c)),
		zap.String("name", req.Name),
		zap.String("type", req.Type),
		zap.String("request_id", c.GetString("request_id")))

	isEnabled := true // default to enabled
	if req.IsEnabled != nil {
		isEnabled = *req.IsEnabled
	}

	integ := &model.Integration{
		Name:                   req.Name,
		Description:            req.Description,
		Type:                   model.IntegrationType(req.Type),
		Mode:                   mode,
		ChannelID:              req.ChannelID,
		PipelineConfig:         req.PipelineConfig,
		LabelEnhancementConfig: req.LabelEnhancementConfig,
		IsEnabled:              isEnabled,
	}

	if err := h.svc.Create(c.Request.Context(), integ); err != nil {
		Error(c, err)
		return
	}
	Success(c, integ)
}

// Get returns a single integration.
// GET /api/v1/integrations/:id
func (h *IntegrationHandler) Get(c *gin.Context) {
	id, err := GetIDParam(c, "id")
	if err != nil {
		Error(c, err)
		return
	}
	integ, err := h.svc.GetByID(c.Request.Context(), id)
	if err != nil {
		Error(c, err)
		return
	}
	Success(c, integ)
}

// Update updates an integration.
// PUT /api/v1/integrations/:id
func (h *IntegrationHandler) Update(c *gin.Context) {
	id, err := GetIDParam(c, "id")
	if err != nil {
		Error(c, err)
		return
	}
	var req CreateIntegrationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		Error(c, apperr.WithMessage(apperr.ErrInvalidParam, err.Error()))
		return
	}
	h.log.Info("integration update",
		zap.Uint("user_id", GetCurrentUserID(c)),
		zap.Uint("integration_id", id),
		zap.String("name", req.Name),
		zap.String("request_id", c.GetString("request_id")))

	updates := &model.Integration{
		Name:                   req.Name,
		Description:            req.Description,
		PipelineConfig:         req.PipelineConfig,
		LabelEnhancementConfig: req.LabelEnhancementConfig,
		Mode:                   model.IntegrationMode(req.Mode),
		ChannelID:              req.ChannelID,
	}
	if req.IsEnabled != nil {
		updates.IsEnabled = *req.IsEnabled
	} else {
		// Preserve existing IsEnabled when not explicitly provided.
		existing, err := h.svc.GetByID(c.Request.Context(), id)
		if err != nil {
			Error(c, err)
			return
		}
		updates.IsEnabled = existing.IsEnabled
	}
	integ, err := h.svc.Update(c.Request.Context(), id, updates)
	if err != nil {
		Error(c, err)
		return
	}
	Success(c, integ)
}

// Delete deletes an integration.
// DELETE /api/v1/integrations/:id
func (h *IntegrationHandler) Delete(c *gin.Context) {
	id, err := GetIDParam(c, "id")
	if err != nil {
		Error(c, err)
		return
	}

	h.log.Info("integration delete",
		zap.Uint("user_id", GetCurrentUserID(c)),
		zap.Uint("integration_id", id),
		zap.String("request_id", c.GetString("request_id")))

	if err := h.svc.Delete(c.Request.Context(), id); err != nil {
		Error(c, err)
		return
	}
	Success(c, nil)
}

// Receive is the webhook entry point.
// POST /api/v1/integrations/:token/alerts
func (h *IntegrationHandler) Receive(c *gin.Context) {
	token := c.Param("token")
	if token == "" {
		Error(c, apperr.WithMessage(apperr.ErrInvalidParam, "missing integration token"))
		return
	}

	body, err := io.ReadAll(http.MaxBytesReader(nil, c.Request.Body, 1<<20)) // 1 MB max
	if err != nil {
		Error(c, apperr.WithMessage(apperr.ErrInvalidParam, "failed to read request body"))
		return
	}

	if err := h.svc.ReceiveAlerts(c.Request.Context(), token, body); err != nil {
		Error(c, err)
		return
	}
	Success(c, gin.H{"received": true})
}

// parseUintStr is a small helper used locally.
func parseUintStr(s string, out *uint) (int, error) {
	var v uint64
	n, err := parseUintFromString(s, &v)
	if err == nil {
		*out = uint(v)
	}
	return n, err
}

func parseUintFromString(s string, out *uint64) (int, error) {
	var v uint64
	for i, c := range s {
		if c < '0' || c > '9' {
			return i, &parseError{}
		}
		v = v*10 + uint64(c-'0')
	}
	*out = v
	return len(s), nil
}

type parseError struct{}

func (e *parseError) Error() string { return "invalid integer" }
