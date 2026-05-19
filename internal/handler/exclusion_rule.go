package handler

import (
	"strconv"

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

	rule := &model.ChannelExclusionRule{
		ChannelID:   channelID,
		Name:        req.Name,
		Description: req.Description,
		Conditions:  req.Conditions,
		IsEnabled:   req.IsEnabled,
		Priority:    req.Priority,
	}
	if rule.Conditions == "" {
		rule.Conditions = "[]"
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

// UpdateNoiseCfg updates a channel's aggregation and flapping configs.
// PUT /api/v1/channels/:id/noise-config
func UpdateNoiseCfg(channelSvc *service.ChannelService) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := GetIDParam(c, "id")
		if err != nil {
			Error(c, err)
			return
		}

		var req struct {
			AggregationConfig string `json:"aggregation_config"`
			FlappingConfig    string `json:"flapping_config"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			Error(c, apperr.WithMessage(apperr.ErrInvalidParam, err.Error()))
			return
		}

		ch, err := channelSvc.Update(c.Request.Context(), id, &model.Channel{
			AggregationConfig: req.AggregationConfig,
			FlappingConfig:    req.FlappingConfig,
		})
		if err != nil {
			Error(c, err)
			return
		}
		Success(c, ch)
	}
}

// GetNoiseCfg returns a channel's noise reduction config.
// GET /api/v1/channels/:id/noise-config
func GetNoiseCfg(channelSvc *service.ChannelService) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := GetIDParam(c, "id")
		if err != nil {
			Error(c, err)
			return
		}
		ch, err := channelSvc.GetByID(c.Request.Context(), id)
		if err != nil {
			Error(c, err)
			return
		}
		Success(c, gin.H{
			"aggregation_config": ch.AggregationConfig,
			"flapping_config":    ch.FlappingConfig,
		})
	}
}

// helper: parse string channel id from path (for sub-resource routes)
func parseChannelID(c *gin.Context) (uint, error) {
	s := c.Param("channel_id")
	if s == "" {
		s = c.Param("id")
	}
	v, err := strconv.ParseUint(s, 10, 64)
	if err != nil {
		return 0, err
	}
	return uint(v), nil
}
