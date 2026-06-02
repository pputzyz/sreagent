package handler

import (
	"encoding/json"

	"github.com/gin-gonic/gin"
	apperr "github.com/sreagent/sreagent/internal/pkg/errors"

	"github.com/sreagent/sreagent/internal/model"
	"github.com/sreagent/sreagent/internal/service"
)

// ExclusionRuleHandler manages channel exclusion rules via HTTP.
type ExclusionRuleHandler struct {
	svc *service.ExclusionRuleService
}

func NewExclusionRuleHandler(svc *service.ExclusionRuleService) *ExclusionRuleHandler {
	return &ExclusionRuleHandler{svc: svc}
}

type CreateExclusionRuleRequest struct {
	ChannelID   uint   `json:"channel_id" binding:"required"`
	Name        string `json:"name" binding:"required"`
	Description string `json:"description"`
	Conditions  string `json:"conditions"` // JSON array of FilterCondition
	IsEnabled   bool   `json:"is_enabled"`
	Priority    int    `json:"priority"`
}

type UpdateExclusionRuleRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Conditions  string `json:"conditions"`
	IsEnabled   bool   `json:"is_enabled"`
	Priority    int    `json:"priority"`
}

// List returns exclusion rules for a channel.
// GET /api/v1/channels/:id/exclusion-rules
func (h *ExclusionRuleHandler) List(c *gin.Context) {
	channelID, err := GetIDParam(c, "id")
	if err != nil {
		Error(c, err)
		return
	}
	list, err := h.svc.ListByChannel(c.Request.Context(), channelID)
	if err != nil {
		Error(c, err)
		return
	}
	Success(c, list)
}

// Create creates an exclusion rule for a channel.
// POST /api/v1/channels/:id/exclusion-rules
func (h *ExclusionRuleHandler) Create(c *gin.Context) {
	channelID, err := GetIDParam(c, "id")
	if err != nil {
		Error(c, err)
		return
	}

	var req CreateExclusionRuleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		Error(c, apperr.WithMessage(apperr.ErrInvalidParam, err.Error()))
		return
	}

	// Validate conditions is valid JSON
	cond := req.Conditions
	if cond == "" {
		cond = "[]"
	}
	if !json.Valid([]byte(cond)) {
		Error(c, apperr.WithMessage(apperr.ErrInvalidParam, "conditions must be valid JSON"))
		return
	}

	rule := &model.ChannelExclusionRule{
		ChannelID:   channelID,
		Name:        req.Name,
		Description: req.Description,
		Conditions:  cond,
		IsEnabled:   req.IsEnabled,
		Priority:    req.Priority,
	}

	if err := h.svc.Create(c.Request.Context(), rule); err != nil {
		Error(c, err)
		return
	}
	Success(c, rule)
}

// Update updates an exclusion rule.
// PUT /api/v1/exclusion-rules/:id
func (h *ExclusionRuleHandler) Update(c *gin.Context) {
	id, err := GetIDParam(c, "id")
	if err != nil {
		Error(c, err)
		return
	}

	var req UpdateExclusionRuleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		Error(c, apperr.WithMessage(apperr.ErrInvalidParam, err.Error()))
		return
	}

	// Validate conditions is valid JSON if provided
	if req.Conditions != "" && !json.Valid([]byte(req.Conditions)) {
		Error(c, apperr.WithMessage(apperr.ErrInvalidParam, "conditions must be valid JSON"))
		return
	}

	updates := &model.ChannelExclusionRule{
		Name:        req.Name,
		Description: req.Description,
		Conditions:  req.Conditions,
		IsEnabled:   req.IsEnabled,
		Priority:    req.Priority,
	}

	rule, err := h.svc.Update(c.Request.Context(), id, updates)
	if err != nil {
		Error(c, err)
		return
	}
	Success(c, rule)
}

// Delete deletes an exclusion rule.
// DELETE /api/v1/exclusion-rules/:id
func (h *ExclusionRuleHandler) Delete(c *gin.Context) {
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
