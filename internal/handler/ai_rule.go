package handler

import (
	"github.com/gin-gonic/gin"
	apperr "github.com/sreagent/sreagent/internal/pkg/errors"

	"github.com/sreagent/sreagent/internal/service"
)

// AIRuleHandler handles AI-powered rule generation endpoints.
type AIRuleHandler struct {
	ruleGenSvc *service.RuleGeneratorService
}

// NewAIRuleHandler creates a new AIRuleHandler.
func NewAIRuleHandler(svc *service.RuleGeneratorService) *AIRuleHandler {
	return &AIRuleHandler{ruleGenSvc: svc}
}

// Generate handles POST /ai/rules/generate — generates a rule from natural language.
func (h *AIRuleHandler) Generate(c *gin.Context) {
	var req service.RuleGenerateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		Error(c, apperr.WithMessage(apperr.ErrInvalidParam, err.Error()))
		return
	}

	result, err := h.ruleGenSvc.Generate(c.Request.Context(), &req)
	if err != nil {
		Error(c, err)
		return
	}

	Success(c, result)
}

// Validate handles POST /ai/rules/validate — validates a PromQL expression.
func (h *AIRuleHandler) Validate(c *gin.Context) {
	var req struct {
		DatasourceID uint   `json:"datasource_id" binding:"required"`
		Expression   string `json:"expression" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		Error(c, apperr.WithMessage(apperr.ErrInvalidParam, err.Error()))
		return
	}

	result, err := h.ruleGenSvc.ValidateExpression(c.Request.Context(), req.DatasourceID, req.Expression)
	if err != nil {
		Error(c, err)
		return
	}

	Success(c, result)
}

// SuggestLabels handles POST /ai/rules/suggest-labels — suggests labels for an expression.
func (h *AIRuleHandler) SuggestLabels(c *gin.Context) {
	var req struct {
		DatasourceID uint   `json:"datasource_id" binding:"required"`
		Expression   string `json:"expression" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		Error(c, apperr.WithMessage(apperr.ErrInvalidParam, err.Error()))
		return
	}

	result, err := h.ruleGenSvc.SuggestLabels(c.Request.Context(), req.DatasourceID, req.Expression)
	if err != nil {
		Error(c, err)
		return
	}

	Success(c, result)
}

// GenerateInhibition handles POST /ai/rules/generate-inhibition — generates an inhibition rule.
func (h *AIRuleHandler) GenerateInhibition(c *gin.Context) {
	var req struct {
		Description  string `json:"description" binding:"required"`
		DatasourceID *uint  `json:"datasource_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		Error(c, apperr.WithMessage(apperr.ErrInvalidParam, err.Error()))
		return
	}

	result, err := h.ruleGenSvc.GenerateInhibition(c.Request.Context(), req.Description, req.DatasourceID)
	if err != nil {
		Error(c, err)
		return
	}

	Success(c, result)
}

// GenerateMute handles POST /ai/rules/generate-mute — generates a mute rule.
func (h *AIRuleHandler) GenerateMute(c *gin.Context) {
	var req struct {
		Description string `json:"description" binding:"required"`
		Timezone    string `json:"timezone"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		Error(c, apperr.WithMessage(apperr.ErrInvalidParam, err.Error()))
		return
	}

	result, err := h.ruleGenSvc.GenerateMute(c.Request.Context(), req.Description, req.Timezone)
	if err != nil {
		Error(c, err)
		return
	}

	Success(c, result)
}

// Improve handles POST /ai/rules/improve — improves an existing rule based on feedback.
func (h *AIRuleHandler) Improve(c *gin.Context) {
	var req service.ImproveRuleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		Error(c, apperr.WithMessage(apperr.ErrInvalidParam, err.Error()))
		return
	}

	result, err := h.ruleGenSvc.ImproveRule(c.Request.Context(), &req)
	if err != nil {
		Error(c, err)
		return
	}

	Success(c, result)
}
