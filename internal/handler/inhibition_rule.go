package handler

import (
	"github.com/gin-gonic/gin"
	apperr "github.com/sreagent/sreagent/internal/pkg/errors"

	"github.com/sreagent/sreagent/internal/model"
	"github.com/sreagent/sreagent/internal/service"
)

// InhibitionRuleHandler handles inhibition rule API requests.
type InhibitionRuleHandler struct {
	svc *service.InhibitionRuleService
}

// NewInhibitionRuleHandler creates a new InhibitionRuleHandler.
func NewInhibitionRuleHandler(svc *service.InhibitionRuleService) *InhibitionRuleHandler {
	return &InhibitionRuleHandler{svc: svc}
}

// InhibitionRuleRequest is the request body for create/update.
type InhibitionRuleRequest struct {
	Name        string           `json:"name" binding:"required"`
	Description string           `json:"description"`
	SourceMatch model.JSONLabels `json:"source_match"`
	TargetMatch model.JSONLabels `json:"target_match"`
	EqualLabels string           `json:"equal_labels"`
	IsEnabled   bool             `json:"is_enabled"`
}

// Create creates a new inhibition rule.
func (h *InhibitionRuleHandler) Create(c *gin.Context) {
	var req InhibitionRuleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		Error(c, apperr.WithMessage(apperr.ErrInvalidParam, err.Error()))
		return
	}
	rule := &model.InhibitionRule{
		Name:        req.Name,
		Description: req.Description,
		SourceMatch: req.SourceMatch,
		TargetMatch: req.TargetMatch,
		EqualLabels: req.EqualLabels,
		IsEnabled:   req.IsEnabled,
		CreatedBy:   GetCurrentUserID(c),
	}
	if rule.SourceMatch == nil {
		rule.SourceMatch = model.JSONLabels{}
	}
	if rule.TargetMatch == nil {
		rule.TargetMatch = model.JSONLabels{}
	}
	if err := h.svc.Create(c.Request.Context(), rule); err != nil {
		Error(c, err)
		return
	}
	Success(c, rule)
}

// Get returns an inhibition rule by ID.
func (h *InhibitionRuleHandler) Get(c *gin.Context) {
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

// List returns a paginated list of inhibition rules.
func (h *InhibitionRuleHandler) List(c *gin.Context) {
	pq := GetPageQuery(c)
	list, total, err := h.svc.List(c.Request.Context(), pq.Page, pq.PageSize)
	if err != nil {
		Error(c, err)
		return
	}
	SuccessPage(c, list, total, pq.Page, pq.PageSize)
}

// Update updates an existing inhibition rule.
func (h *InhibitionRuleHandler) Update(c *gin.Context) {
	id, err := GetIDParam(c, "id")
	if err != nil {
		Error(c, err)
		return
	}
	var req InhibitionRuleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		Error(c, apperr.WithMessage(apperr.ErrInvalidParam, err.Error()))
		return
	}
	rule := &model.InhibitionRule{
		Name:        req.Name,
		Description: req.Description,
		SourceMatch: req.SourceMatch,
		TargetMatch: req.TargetMatch,
		EqualLabels: req.EqualLabels,
		IsEnabled:   req.IsEnabled,
	}
	rule.ID = id
	if rule.SourceMatch == nil {
		rule.SourceMatch = model.JSONLabels{}
	}
	if rule.TargetMatch == nil {
		rule.TargetMatch = model.JSONLabels{}
	}
	if err := h.svc.Update(c.Request.Context(), rule); err != nil {
		Error(c, err)
		return
	}
	Success(c, rule)
}

// Delete soft-deletes an inhibition rule.
func (h *InhibitionRuleHandler) Delete(c *gin.Context) {
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
