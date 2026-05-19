package handler

import (
	"github.com/gin-gonic/gin"
	apperr "github.com/sreagent/sreagent/internal/pkg/errors"

	"github.com/sreagent/sreagent/internal/model"
	"github.com/sreagent/sreagent/internal/service"
)

// SubscribeRuleHandler handles HTTP requests for subscribe rules.
type SubscribeRuleHandler struct {
	svc *service.SubscribeRuleService
}

// NewSubscribeRuleHandler creates a new SubscribeRuleHandler.
func NewSubscribeRuleHandler(svc *service.SubscribeRuleService) *SubscribeRuleHandler {
	return &SubscribeRuleHandler{svc: svc}
}

// CreateSubscribeRuleRequest is the request body for creating a subscribe rule.
type CreateSubscribeRuleRequest struct {
	Name         string           `json:"name" binding:"required"`
	Description  string           `json:"description"`
	IsEnabled    *bool            `json:"is_enabled"`
	MatchLabels  model.JSONLabels `json:"match_labels"`
	Severities   string           `json:"severities"`
	NotifyRuleID uint             `json:"notify_rule_id" binding:"required"`
	UserID       *uint            `json:"user_id"`
	TeamID       *uint            `json:"team_id"`
}

// UpdateSubscribeRuleRequest is the request body for updating a subscribe rule.
type UpdateSubscribeRuleRequest struct {
	Name         string           `json:"name" binding:"required"`
	Description  string           `json:"description"`
	IsEnabled    *bool            `json:"is_enabled"`
	MatchLabels  model.JSONLabels `json:"match_labels"`
	Severities   string           `json:"severities"`
	NotifyRuleID uint             `json:"notify_rule_id" binding:"required"`
}

// Create creates a new subscribe rule.
func (h *SubscribeRuleHandler) Create(c *gin.Context) {
	var req CreateSubscribeRuleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		Error(c, apperr.WithMessage(apperr.ErrInvalidParam, err.Error()))
		return
	}

	isEnabled := true
	if req.IsEnabled != nil {
		isEnabled = *req.IsEnabled
	}

	rule := &model.SubscribeRule{
		Name:         req.Name,
		Description:  req.Description,
		IsEnabled:    isEnabled,
		MatchLabels:  req.MatchLabels,
		Severities:   req.Severities,
		NotifyRuleID: req.NotifyRuleID,
		UserID:       req.UserID,
		TeamID:       req.TeamID,
		CreatedBy:    GetCurrentUserID(c),
	}

	if err := h.svc.Create(c.Request.Context(), rule); err != nil {
		Error(c, err)
		return
	}

	Success(c, rule)
}

// Get returns a single subscribe rule by ID.
func (h *SubscribeRuleHandler) Get(c *gin.Context) {
	id, err := GetIDParam(c, "id")
	if err != nil {
		Error(c, err)
		return
	}

	rule, err := h.svc.GetByID(c.Request.Context(), id)
	if err != nil {
		Error(c, err)
		return
	}

	Success(c, rule)
}

// List returns a paginated list of subscribe rules.
func (h *SubscribeRuleHandler) List(c *gin.Context) {
	pq := GetPageQuery(c)

	list, total, err := h.svc.List(c.Request.Context(), pq.Page, pq.PageSize)
	if err != nil {
		Error(c, err)
		return
	}

	SuccessPage(c, list, total, pq.Page, pq.PageSize)
}

// Update updates a subscribe rule.
func (h *SubscribeRuleHandler) Update(c *gin.Context) {
	id, err := GetIDParam(c, "id")
	if err != nil {
		Error(c, err)
		return
	}

	var req UpdateSubscribeRuleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		Error(c, apperr.WithMessage(apperr.ErrInvalidParam, err.Error()))
		return
	}

	isEnabled := true
	if req.IsEnabled != nil {
		isEnabled = *req.IsEnabled
	}

	rule := &model.SubscribeRule{
		Name:         req.Name,
		Description:  req.Description,
		IsEnabled:    isEnabled,
		MatchLabels:  req.MatchLabels,
		Severities:   req.Severities,
		NotifyRuleID: req.NotifyRuleID,
	}
	rule.ID = id

	if err := h.svc.Update(c.Request.Context(), rule); err != nil {
		Error(c, err)
		return
	}

	Success(c, rule)
}

// Delete deletes a subscribe rule.
func (h *SubscribeRuleHandler) Delete(c *gin.Context) {
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
