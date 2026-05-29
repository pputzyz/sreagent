package handler

import (
	"encoding/json"
	"strings"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"github.com/sreagent/sreagent/internal/model"
	"github.com/sreagent/sreagent/internal/pkg/crypto"
	apperr "github.com/sreagent/sreagent/internal/pkg/errors"

	"github.com/sreagent/sreagent/internal/service"
)

// NotifyMediaHandler handles HTTP requests for notify medias.
type NotifyMediaHandler struct {
	svc      *service.NotifyMediaService
	auditSvc *service.AuditLogService
	log      *zap.Logger
}

// NewNotifyMediaHandler creates a new NotifyMediaHandler.
func NewNotifyMediaHandler(svc *service.NotifyMediaService, logger *zap.Logger) *NotifyMediaHandler {
	return &NotifyMediaHandler{svc: svc, log: logger}
}

// SetAuditService injects the audit log service (called after construction to avoid circular DI).
func (h *NotifyMediaHandler) SetAuditService(svc *service.AuditLogService) {
	h.auditSvc = svc
}

// maskNotifyMediaConfig replaces sensitive fields in Config with asterisks before returning to the API.
// If the config is encrypted, it is decrypted first so that masking can operate on the plaintext JSON.
func maskNotifyMediaConfig(media *model.NotifyMedia) {
	if media.Config == "" {
		return
	}
	// Decrypt if encrypted, so downstream JSON unmarshal and masking work on plaintext.
	config := media.Config
	if crypto.IsEncrypted(config) {
		if decrypted, err := crypto.DecryptString(config); err == nil {
			config = decrypted
		}
		// If decryption fails, fall through with the raw value (mask will be a no-op).
	}
	var cfg map[string]interface{}
	if json.Unmarshal([]byte(config), &cfg) != nil {
		return
	}
	sensitiveKeys := []string{
		"password", "secret", "token", "api_key", "apikey",
		"access_key", "secret_key", "auth_token", "webhook_url", "smtp_password",
	}
	for _, sensitive := range sensitiveKeys {
		for k := range cfg {
			if strings.Contains(strings.ToLower(k), sensitive) {
				cfg[k] = "********"
			}
		}
	}
	if masked, err := json.Marshal(cfg); err == nil {
		media.Config = string(masked)
	}
}

// CreateNotifyMediaRequest is the request body for creating a notify media.
type CreateNotifyMediaRequest struct {
	Name        string                `json:"name" binding:"required"`
	Type        model.NotifyMediaType `json:"type" binding:"required"`
	Description string                `json:"description"`
	IsEnabled   *bool                 `json:"is_enabled"`
	Config      string                `json:"config" binding:"required"`
	Variables   string                `json:"variables"`
}

// UpdateNotifyMediaRequest is the request body for updating a notify media.
type UpdateNotifyMediaRequest struct {
	Name        string                `json:"name" binding:"required"`
	Type        model.NotifyMediaType `json:"type" binding:"required"`
	Description string                `json:"description"`
	IsEnabled   *bool                 `json:"is_enabled"`
	Config      string                `json:"config"`
	Variables   string                `json:"variables"`
}

// Create creates a new notify media.
func (h *NotifyMediaHandler) Create(c *gin.Context) {
	var req CreateNotifyMediaRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		Error(c, apperr.WithMessage(apperr.ErrInvalidParam, err.Error()))
		return
	}

	isEnabled := true
	if req.IsEnabled != nil {
		isEnabled = *req.IsEnabled
	}

	h.log.Info("notify media create",
		zap.Uint("user_id", GetCurrentUserID(c)),
		zap.String("name", req.Name),
		zap.String("type", string(req.Type)),
		zap.String("request_id", c.GetString("request_id")))

	media := &model.NotifyMedia{
		Name:        req.Name,
		Type:        req.Type,
		Description: req.Description,
		IsEnabled:   isEnabled,
		Config:      req.Config,
		Variables:   req.Variables,
	}

	if err := h.svc.Create(c.Request.Context(), media); err != nil {
		Error(c, err)
		return
	}

	if h.auditSvc != nil {
		uid := GetCurrentUserID(c)
		h.auditSvc.Record(&model.AuditLog{
			UserID: &uid, Action: model.AuditActionCreate,
			ResourceType: model.AuditResourceNotifyMedia, ResourceID: &media.ID, ResourceName: media.Name,
			IP: c.ClientIP(),
		})
	}

	maskNotifyMediaConfig(media)
	Success(c, media)
}

// Get returns a single notify media by ID.
func (h *NotifyMediaHandler) Get(c *gin.Context) {
	id, err := GetIDParam(c, "id")
	if err != nil {
		Error(c, err)
		return
	}

	media, err := h.svc.GetByID(c.Request.Context(), id)
	if err != nil {
		Error(c, err)
		return
	}

	maskNotifyMediaConfig(media)
	Success(c, media)
}

// List returns a paginated list of notify medias.
func (h *NotifyMediaHandler) List(c *gin.Context) {
	pq := GetPageQuery(c)

	list, total, err := h.svc.List(c.Request.Context(), pq.Page, pq.PageSize)
	if err != nil {
		Error(c, err)
		return
	}

	for i := range list {
		maskNotifyMediaConfig(&list[i])
	}
	SuccessPage(c, list, total, pq.Page, pq.PageSize)
}

// Update updates a notify media.
func (h *NotifyMediaHandler) Update(c *gin.Context) {
	id, err := GetIDParam(c, "id")
	if err != nil {
		Error(c, err)
		return
	}

	var req UpdateNotifyMediaRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		Error(c, apperr.WithMessage(apperr.ErrInvalidParam, err.Error()))
		return
	}

	isEnabled := true
	if req.IsEnabled != nil {
		isEnabled = *req.IsEnabled
	}

	h.log.Info("notify media update",
		zap.Uint("user_id", GetCurrentUserID(c)),
		zap.Uint("media_id", id),
		zap.String("name", req.Name),
		zap.String("request_id", c.GetString("request_id")))

	media := &model.NotifyMedia{
		Name:        req.Name,
		Type:        req.Type,
		Description: req.Description,
		IsEnabled:   isEnabled,
		Config:      req.Config,
		Variables:   req.Variables,
	}
	media.ID = id

	if err := h.svc.Update(c.Request.Context(), media); err != nil {
		Error(c, err)
		return
	}

	if h.auditSvc != nil {
		uid := GetCurrentUserID(c)
		h.auditSvc.Record(&model.AuditLog{
			UserID: &uid, Action: model.AuditActionUpdate,
			ResourceType: model.AuditResourceNotifyMedia, ResourceID: &id, ResourceName: req.Name,
			IP: c.ClientIP(),
		})
	}

	maskNotifyMediaConfig(media)
	Success(c, media)
}

// Delete deletes a notify media.
func (h *NotifyMediaHandler) Delete(c *gin.Context) {
	id, err := GetIDParam(c, "id")
	if err != nil {
		Error(c, err)
		return
	}

	h.log.Info("notify media delete",
		zap.Uint("user_id", GetCurrentUserID(c)),
		zap.Uint("media_id", id),
		zap.String("request_id", c.GetString("request_id")))

	if err := h.svc.Delete(c.Request.Context(), id); err != nil {
		Error(c, err)
		return
	}

	if h.auditSvc != nil {
		uid := GetCurrentUserID(c)
		h.auditSvc.Record(&model.AuditLog{
			UserID: &uid, Action: model.AuditActionDelete,
			ResourceType: model.AuditResourceNotifyMedia, ResourceID: &id,
			IP: c.ClientIP(),
		})
	}

	Success(c, nil)
}

// Test sends a test notification via a media.
func (h *NotifyMediaHandler) Test(c *gin.Context) {
	id, err := GetIDParam(c, "id")
	if err != nil {
		Error(c, err)
		return
	}

	if err := h.svc.TestMedia(c.Request.Context(), id); err != nil {
		Error(c, err)
		return
	}

	Success(c, gin.H{"message": "test notification sent"})
}
