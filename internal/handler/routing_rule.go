package handler

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/sreagent/sreagent/internal/model"
	"github.com/sreagent/sreagent/internal/repository"
)

// RoutingRuleHandler manages routing rules for shared integrations.
type RoutingRuleHandler struct {
	repo *repository.RoutingRuleRepository
}

func NewRoutingRuleHandler(repo *repository.RoutingRuleRepository) *RoutingRuleHandler {
	return &RoutingRuleHandler{repo: repo}
}

type createRoutingRuleReq struct {
	IntegrationID   uint   `json:"integration_id" binding:"required"`
	TargetChannelID uint   `json:"target_channel_id" binding:"required"`
	Conditions      string `json:"conditions"`
	Priority        int    `json:"priority"`
	IsEnabled       bool   `json:"is_enabled"`
}

type updateRoutingRuleReq struct {
	TargetChannelID uint   `json:"target_channel_id"`
	Conditions      string `json:"conditions"`
	Priority        int    `json:"priority"`
	IsEnabled       *bool  `json:"is_enabled"`
}

// ListByIntegration returns all routing rules for a shared integration.
// GET /api/v1/routing-rules?integration_id=X
func (h *RoutingRuleHandler) ListByIntegration(c *gin.Context) {
	integID, err := strconv.ParseUint(c.Query("integration_id"), 10, 64)
	if err != nil || integID == 0 {
		ErrorWithMessage(c, 10001, "missing or invalid integration_id query param")
		return
	}
	rules, err := h.repo.ListByIntegration(c.Request.Context(), uint(integID))
	if err != nil {
		ErrorWithMessage(c, 50001, err.Error())
		return
	}
	Success(c, rules)
}

// Create adds a new routing rule.
// POST /api/v1/routing-rules  (integration_id in body)
func (h *RoutingRuleHandler) Create(c *gin.Context) {
	var req createRoutingRuleReq
	if err := c.ShouldBindJSON(&req); err != nil {
		ErrorWithMessage(c, 10001, err.Error())
		return
	}
	if req.IntegrationID == 0 {
		ErrorWithMessage(c, 10001, "integration_id is required")
		return
	}
	rule := &model.RoutingRule{
		IntegrationID:   req.IntegrationID,
		TargetChannelID: req.TargetChannelID,
		Conditions:      req.Conditions,
		Priority:        req.Priority,
		IsEnabled:       req.IsEnabled,
	}
	if err := h.repo.Create(c.Request.Context(), rule); err != nil {
		ErrorWithMessage(c, 50001, err.Error())
		return
	}
	Success(c, rule)
}

// Update modifies a routing rule.
// PUT /api/v1/routing-rules/:id
func (h *RoutingRuleHandler) Update(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		ErrorWithMessage(c, 10001, "invalid id")
		return
	}
	var req updateRoutingRuleReq
	if err := c.ShouldBindJSON(&req); err != nil {
		ErrorWithMessage(c, 10001, err.Error())
		return
	}
	rule, err := h.repo.GetByID(c.Request.Context(), uint(id))
	if err != nil {
		ErrorWithMessage(c, 50001, err.Error())
		return
	}
	if req.TargetChannelID != 0 {
		rule.TargetChannelID = req.TargetChannelID
	}
	if req.Conditions != "" {
		rule.Conditions = req.Conditions
	}
	rule.Priority = req.Priority
	if req.IsEnabled != nil {
		rule.IsEnabled = *req.IsEnabled
	}
	if err := h.repo.Update(c.Request.Context(), rule); err != nil {
		ErrorWithMessage(c, 50001, err.Error())
		return
	}
	Success(c, rule)
}

// Delete removes a routing rule.
// DELETE /api/v1/routing-rules/:id
func (h *RoutingRuleHandler) Delete(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		ErrorWithMessage(c, 10001, "invalid id")
		return
	}
	if err := h.repo.Delete(c.Request.Context(), uint(id)); err != nil {
		ErrorWithMessage(c, 50001, err.Error())
		return
	}
	Success(c, nil)
}
