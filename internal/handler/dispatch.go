package handler

import (
	"github.com/gin-gonic/gin"

	"github.com/sreagent/sreagent/internal/model"
	"github.com/sreagent/sreagent/internal/service"
)

// DispatchHandler manages dispatch policies via HTTP.
type DispatchHandler struct {
	svc *service.DispatchService
}

func NewDispatchHandler(svc *service.DispatchService) *DispatchHandler {
	return &DispatchHandler{svc: svc}
}

type CreateDispatchPolicyRequest struct {
	Name                  string `json:"name" binding:"required"`
	Description           string `json:"description"`
	IsEnabled             bool   `json:"is_enabled"`
	Priority              int    `json:"priority"`
	MatchConditions       string `json:"match_conditions"`
	ActiveTimeConfig      string `json:"active_time_config"`
	DelaySeconds          int    `json:"delay_seconds"`
	EscalationPolicyID    *uint  `json:"escalation_policy_id"`
	RepeatIntervalSeconds int    `json:"repeat_interval_seconds"`
	MaxRepeats            int    `json:"max_repeats"`
	NotifyMode            string `json:"notify_mode"`
	UnifiedMediaID        *uint  `json:"unified_media_id"`
	LabelEnhancementRules string `json:"label_enhancement_rules"`
}

// List returns dispatch policies for a channel.
// GET /api/v1/channels/:id/dispatch-policies
func (h *DispatchHandler) List(c *gin.Context) {
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

// Create creates a dispatch policy for a channel.
// POST /api/v1/channels/:id/dispatch-policies
func (h *DispatchHandler) Create(c *gin.Context) {
	channelID, err := GetIDParam(c, "id")
	if err != nil {
		Error(c, err)
		return
	}

	var req CreateDispatchPolicyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		ErrorWithMessage(c, 10001, err.Error())
		return
	}

	notifyMode := req.NotifyMode
	if notifyMode == "" {
		notifyMode = "personal_preference"
	}

	p := &model.DispatchPolicy{
		ChannelID:             channelID,
		Name:                  req.Name,
		Description:           req.Description,
		IsEnabled:             req.IsEnabled,
		Priority:              req.Priority,
		MatchConditions:       req.MatchConditions,
		ActiveTimeConfig:      req.ActiveTimeConfig,
		DelaySeconds:          req.DelaySeconds,
		EscalationPolicyID:    req.EscalationPolicyID,
		RepeatIntervalSeconds: req.RepeatIntervalSeconds,
		MaxRepeats:            req.MaxRepeats,
		NotifyMode:            notifyMode,
		UnifiedMediaID:        req.UnifiedMediaID,
		LabelEnhancementRules: req.LabelEnhancementRules,
	}

	if err := h.svc.Create(c.Request.Context(), p); err != nil {
		Error(c, err)
		return
	}
	Success(c, p)
}

// Get returns a single dispatch policy.
// GET /api/v1/dispatch-policies/:id
func (h *DispatchHandler) Get(c *gin.Context) {
	id, err := GetIDParam(c, "id")
	if err != nil {
		Error(c, err)
		return
	}
	p, err := h.svc.GetByID(c.Request.Context(), id)
	if err != nil {
		Error(c, err)
		return
	}
	Success(c, p)
}

// Update updates a dispatch policy.
// PUT /api/v1/dispatch-policies/:id
func (h *DispatchHandler) Update(c *gin.Context) {
	id, err := GetIDParam(c, "id")
	if err != nil {
		Error(c, err)
		return
	}

	var req CreateDispatchPolicyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		ErrorWithMessage(c, 10001, err.Error())
		return
	}

	updates := &model.DispatchPolicy{
		Name:                  req.Name,
		Description:           req.Description,
		IsEnabled:             req.IsEnabled,
		Priority:              req.Priority,
		MatchConditions:       req.MatchConditions,
		ActiveTimeConfig:      req.ActiveTimeConfig,
		DelaySeconds:          req.DelaySeconds,
		EscalationPolicyID:    req.EscalationPolicyID,
		RepeatIntervalSeconds: req.RepeatIntervalSeconds,
		MaxRepeats:            req.MaxRepeats,
		NotifyMode:            req.NotifyMode,
		UnifiedMediaID:        req.UnifiedMediaID,
		LabelEnhancementRules: req.LabelEnhancementRules,
	}

	p, err := h.svc.Update(c.Request.Context(), id, updates)
	if err != nil {
		Error(c, err)
		return
	}
	Success(c, p)
}

// Delete deletes a dispatch policy.
// DELETE /api/v1/dispatch-policies/:id
func (h *DispatchHandler) Delete(c *gin.Context) {
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
