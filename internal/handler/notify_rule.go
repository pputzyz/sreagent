package handler

import (
	"github.com/gin-gonic/gin"

	"github.com/sreagent/sreagent/internal/model"
	apperr "github.com/sreagent/sreagent/internal/pkg/errors"
	"github.com/sreagent/sreagent/internal/service"
)

// NotifyRuleHandler handles HTTP requests for notify rules.
type NotifyRuleHandler struct {
	svc *service.NotifyRuleService
}

// NewNotifyRuleHandler creates a new NotifyRuleHandler.
func NewNotifyRuleHandler(svc *service.NotifyRuleService) *NotifyRuleHandler {
	return &NotifyRuleHandler{svc: svc}
}

// CreateNotifyRuleRequest is the request body for creating a notify rule.
type CreateNotifyRuleRequest struct {
	Name           string           `json:"name" binding:"required"`
	Description    string           `json:"description"`
	IsEnabled      *bool            `json:"is_enabled"`
	Severities     string           `json:"severities"`
	MatchLabels    model.JSONLabels `json:"match_labels"`
	Pipeline       string           `json:"pipeline"`
	NotifyConfigs  string           `json:"notify_configs"`
	RepeatInterval int              `json:"repeat_interval"`
	CallbackURL    string           `json:"callback_url"`
}

// UpdateNotifyRuleRequest is the request body for updating a notify rule.
type UpdateNotifyRuleRequest struct {
	Name           string           `json:"name" binding:"required"`
	Description    string           `json:"description"`
	IsEnabled      *bool            `json:"is_enabled"`
	Severities     string           `json:"severities"`
	MatchLabels    model.JSONLabels `json:"match_labels"`
	Pipeline       string           `json:"pipeline"`
	NotifyConfigs  string           `json:"notify_configs"`
	RepeatInterval int              `json:"repeat_interval"`
	CallbackURL    string           `json:"callback_url"`
}

// Create creates a new notify rule.
func (h *NotifyRuleHandler) Create(c *gin.Context) {
	var req CreateNotifyRuleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		Error(c, apperr.WithMessage(apperr.ErrInvalidParam, err.Error()))
		return
	}

	isEnabled := true
	if req.IsEnabled != nil {
		isEnabled = *req.IsEnabled
	}

	rule := &model.NotifyRule{
		Name:           req.Name,
		Description:    req.Description,
		IsEnabled:      isEnabled,
		Severities:     req.Severities,
		MatchLabels:    req.MatchLabels,
		Pipeline:       req.Pipeline,
		NotifyConfigs:  req.NotifyConfigs,
		RepeatInterval: req.RepeatInterval,
		CallbackURL:    req.CallbackURL,
		CreatedBy:      GetCurrentUserID(c),
	}

	if err := h.svc.Create(c.Request.Context(), rule); err != nil {
		Error(c, err)
		return
	}

	Success(c, rule)
}

// Get returns a single notify rule by ID.
func (h *NotifyRuleHandler) Get(c *gin.Context) {
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

// List returns a paginated list of notify rules.
func (h *NotifyRuleHandler) List(c *gin.Context) {
	pq := GetPageQuery(c)

	list, total, err := h.svc.List(c.Request.Context(), pq.Page, pq.PageSize)
	if err != nil {
		Error(c, err)
		return
	}

	SuccessPage(c, list, total, pq.Page, pq.PageSize)
}

// Update updates a notify rule.
func (h *NotifyRuleHandler) Update(c *gin.Context) {
	id, err := GetIDParam(c, "id")
	if err != nil {
		Error(c, err)
		return
	}

	var req UpdateNotifyRuleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		Error(c, apperr.WithMessage(apperr.ErrInvalidParam, err.Error()))
		return
	}

	isEnabled := true
	if req.IsEnabled != nil {
		isEnabled = *req.IsEnabled
	}

	rule := &model.NotifyRule{
		Name:           req.Name,
		Description:    req.Description,
		IsEnabled:      isEnabled,
		Severities:     req.Severities,
		MatchLabels:    req.MatchLabels,
		Pipeline:       req.Pipeline,
		NotifyConfigs:  req.NotifyConfigs,
		RepeatInterval: req.RepeatInterval,
		CallbackURL:    req.CallbackURL,
	}
	rule.ID = id

	if err := h.svc.Update(c.Request.Context(), rule); err != nil {
		Error(c, err)
		return
	}

	Success(c, rule)
}

// Delete deletes a notify rule.
func (h *NotifyRuleHandler) Delete(c *gin.Context) {
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

// notifyBatchIDsReq is the request body for batch operations.
type notifyBatchIDsReq struct {
	IDs []uint `json:"ids" binding:"required,min=1"`
}

// BatchEnable enables multiple notify rules.
func (h *NotifyRuleHandler) BatchEnable(c *gin.Context) {
	var req notifyBatchIDsReq
	if err := c.ShouldBindJSON(&req); err != nil {
		Error(c, apperr.WithMessage(apperr.ErrInvalidParam, err.Error()))
		return
	}
	if err := h.svc.BatchEnable(c.Request.Context(), req.IDs); err != nil {
		Error(c, err)
		return
	}
	Success(c, nil)
}

// BatchDisable disables multiple notify rules.
func (h *NotifyRuleHandler) BatchDisable(c *gin.Context) {
	var req notifyBatchIDsReq
	if err := c.ShouldBindJSON(&req); err != nil {
		Error(c, apperr.WithMessage(apperr.ErrInvalidParam, err.Error()))
		return
	}
	if err := h.svc.BatchDisable(c.Request.Context(), req.IDs); err != nil {
		Error(c, err)
		return
	}
	Success(c, nil)
}

// BatchDelete deletes multiple notify rules.
func (h *NotifyRuleHandler) BatchDelete(c *gin.Context) {
	var req notifyBatchIDsReq
	if err := c.ShouldBindJSON(&req); err != nil {
		Error(c, apperr.WithMessage(apperr.ErrInvalidParam, err.Error()))
		return
	}
	if err := h.svc.BatchDelete(c.Request.Context(), req.IDs); err != nil {
		Error(c, err)
		return
	}
	Success(c, nil)
}
