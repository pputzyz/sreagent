package handler

import (
	"io"

	"github.com/gin-gonic/gin"

	"github.com/sreagent/sreagent/internal/model"
	"github.com/sreagent/sreagent/internal/service"
)

// IntegrationHandler manages webhook integrations.
type IntegrationHandler struct {
	svc *service.IntegrationService
}

func NewIntegrationHandler(svc *service.IntegrationService) *IntegrationHandler {
	return &IntegrationHandler{svc: svc}
}

type CreateIntegrationRequest struct {
	Name                   string `json:"name" binding:"required"`
	Description            string `json:"description"`
	Type                   string `json:"type" binding:"required"` // standard | alertmanager | grafana
	Mode                   string `json:"mode"`                    // exclusive | shared
	ChannelID              *uint  `json:"channel_id"`
	PipelineConfig         string `json:"pipeline_config"`
	LabelEnhancementConfig string `json:"label_enhancement_config"`
	IsEnabled              bool   `json:"is_enabled"`
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
		ErrorWithMessage(c, 10001, err.Error())
		return
	}

	mode := model.IntegrationMode(req.Mode)
	if mode == "" {
		mode = model.IntegrationModeExclusive
	}

	integ := &model.Integration{
		Name:                   req.Name,
		Description:            req.Description,
		Type:                   model.IntegrationType(req.Type),
		Mode:                   mode,
		ChannelID:              req.ChannelID,
		PipelineConfig:         req.PipelineConfig,
		LabelEnhancementConfig: req.LabelEnhancementConfig,
		IsEnabled:              req.IsEnabled,
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
		ErrorWithMessage(c, 10001, err.Error())
		return
	}
	updates := &model.Integration{
		Name:                   req.Name,
		Description:            req.Description,
		IsEnabled:              req.IsEnabled,
		PipelineConfig:         req.PipelineConfig,
		LabelEnhancementConfig: req.LabelEnhancementConfig,
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
		ErrorWithMessage(c, 10001, "missing integration token")
		return
	}

	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		ErrorWithMessage(c, 10001, "failed to read request body")
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
