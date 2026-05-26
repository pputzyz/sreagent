package handler

import (
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/sreagent/sreagent/internal/model"
	apperr "github.com/sreagent/sreagent/internal/pkg/errors"
	"github.com/sreagent/sreagent/internal/service"
)

// LLMConfigHandler handles HTTP requests for LLM configs.
type LLMConfigHandler struct {
	svc      *service.LLMConfigService
	auditSvc *service.AuditLogService
	log      *zap.Logger
}

// NewLLMConfigHandler creates a new LLMConfigHandler.
func NewLLMConfigHandler(svc *service.LLMConfigService, logger *zap.Logger) *LLMConfigHandler {
	return &LLMConfigHandler{svc: svc, log: logger}
}

// SetAuditService injects the audit log service.
func (h *LLMConfigHandler) SetAuditService(svc *service.AuditLogService) {
	h.auditSvc = svc
}

// --- Request types ---

// CreateLLMConfigRequest is the request body for creating an LLM config.
type CreateLLMConfigRequest struct {
	Name        string `json:"name" binding:"required,max=128"`
	Provider    string `json:"provider" binding:"required,oneof=openai azure ollama anthropic custom"`
	APIURL      string `json:"api_url" binding:"max=512"`
	APIKey      string `json:"api_key"`
	Model       string `json:"model" binding:"max=128"`
	ExtraConfig string `json:"extra_config"`
	Enabled     *bool  `json:"enabled"`
	IsDefault   *bool  `json:"is_default"`
	Description string `json:"description" binding:"max=512"`
}

// UpdateLLMConfigRequest is the request body for updating an LLM config.
type UpdateLLMConfigRequest struct {
	Name        string `json:"name" binding:"required,max=128"`
	Provider    string `json:"provider" binding:"required,oneof=openai azure ollama anthropic custom"`
	APIURL      string `json:"api_url" binding:"max=512"`
	APIKey      string `json:"api_key"`
	Model       string `json:"model" binding:"max=128"`
	ExtraConfig string `json:"extra_config"`
	Enabled     *bool  `json:"enabled"`
	IsDefault   *bool  `json:"is_default"`
	Description string `json:"description" binding:"max=512"`
}

// TestLLMConfigRequest is the request body for testing a connection.
type TestLLMConfigRequest struct {
	ID uint `json:"id"`
}

// --- Handler methods ---

// List returns a paginated list of LLM configs.
// GET /llm-configs?page=1&page_size=20
func (h *LLMConfigHandler) List(c *gin.Context) {
	pq := GetPageQuery(c)

	list, total, err := h.svc.List(c.Request.Context(), pq.Page, pq.PageSize)
	if err != nil {
		Error(c, err)
		return
	}

	SuccessPage(c, list, total, pq.Page, pq.PageSize)
}

// Get returns a single LLM config by ID.
// GET /llm-configs/:id
func (h *LLMConfigHandler) Get(c *gin.Context) {
	id, err := GetIDParam(c, "id")
	if err != nil {
		Error(c, err)
		return
	}

	v, err := h.svc.GetByID(c.Request.Context(), id)
	if err != nil {
		Error(c, err)
		return
	}

	// Mask API key in response
	v.APIKey = v.MaskAPIKey()
	Success(c, v)
}

// Create creates a new LLM config.
// POST /llm-configs
func (h *LLMConfigHandler) Create(c *gin.Context) {
	var req CreateLLMConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		Error(c, apperr.WithMessage(apperr.ErrInvalidParam, err.Error()))
		return
	}

	userID := GetCurrentUserID(c)

	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}
	isDefault := false
	if req.IsDefault != nil {
		isDefault = *req.IsDefault
	}

	v := &model.LLMConfig{
		Name:        req.Name,
		Provider:    req.Provider,
		APIURL:      req.APIURL,
		APIKey:      req.APIKey,
		ModelName:   req.Model,
		ExtraConfig: req.ExtraConfig,
		Enabled:     enabled,
		IsDefault:   isDefault,
		Description: req.Description,
		CreatedBy:   userID,
		UpdatedBy:   userID,
	}

	if err := h.svc.Create(c.Request.Context(), v); err != nil {
		Error(c, err)
		return
	}

	if h.auditSvc != nil {
		uid := userID
		h.auditSvc.Record(&model.AuditLog{
			UserID:       &uid,
			Action:       model.AuditActionCreate,
			ResourceType: "llm_config",
			ResourceID:   &v.ID,
			IP:           c.ClientIP(),
		})
	}

	// Mask API key in response
	v.APIKey = v.MaskAPIKey()
	Success(c, v)
}

// Update updates an existing LLM config.
// PUT /llm-configs/:id
func (h *LLMConfigHandler) Update(c *gin.Context) {
	id, err := GetIDParam(c, "id")
	if err != nil {
		Error(c, err)
		return
	}

	existing, err := h.svc.GetByID(c.Request.Context(), id)
	if err != nil {
		Error(c, err)
		return
	}

	var req UpdateLLMConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		Error(c, apperr.WithMessage(apperr.ErrInvalidParam, err.Error()))
		return
	}

	userID := GetCurrentUserID(c)

	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}
	isDefault := false
	if req.IsDefault != nil {
		isDefault = *req.IsDefault
	}

	input := &model.LLMConfig{
		Name:        req.Name,
		Provider:    req.Provider,
		APIURL:      req.APIURL,
		APIKey:      req.APIKey,
		ModelName:   req.Model,
		ExtraConfig: req.ExtraConfig,
		Enabled:     enabled,
		IsDefault:   isDefault,
		Description: req.Description,
		UpdatedBy:   userID,
	}

	if err := h.svc.Update(c.Request.Context(), existing, input); err != nil {
		Error(c, err)
		return
	}

	if h.auditSvc != nil {
		uid := userID
		rid := id
		h.auditSvc.Record(&model.AuditLog{
			UserID:       &uid,
			Action:       model.AuditActionUpdate,
			ResourceType: "llm_config",
			ResourceID:   &rid,
			IP:           c.ClientIP(),
		})
	}

	Success(c, nil)
}

// Delete deletes an LLM config.
// DELETE /llm-configs/:id
func (h *LLMConfigHandler) Delete(c *gin.Context) {
	id, err := GetIDParam(c, "id")
	if err != nil {
		Error(c, err)
		return
	}

	if err := h.svc.Delete(c.Request.Context(), id); err != nil {
		Error(c, err)
		return
	}

	if h.auditSvc != nil {
		uid := GetCurrentUserID(c)
		rid := id
		h.auditSvc.Record(&model.AuditLog{
			UserID:       &uid,
			Action:       model.AuditActionDelete,
			ResourceType: "llm_config",
			ResourceID:   &rid,
			IP:           c.ClientIP(),
		})
	}

	Success(c, nil)
}

// TestConnection tests connectivity for an LLM config by ID.
// POST /llm-configs/test
func (h *LLMConfigHandler) TestConnection(c *gin.Context) {
	var req TestLLMConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		Error(c, apperr.WithMessage(apperr.ErrInvalidParam, err.Error()))
		return
	}

	v, err := h.svc.GetByID(c.Request.Context(), req.ID)
	if err != nil {
		Error(c, err)
		return
	}

	if err := h.svc.TestConnection(c.Request.Context(), v); err != nil {
		Error(c, err)
		return
	}

	Success(c, gin.H{"message": "connection successful"})
}
