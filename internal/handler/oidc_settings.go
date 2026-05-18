package handler

import (
	"context"

	"github.com/gin-gonic/gin"

	"github.com/sreagent/sreagent/internal/service"
)

// OIDCSettingsHandler manages OIDC configuration stored in the DB.
// This is separate from OIDCHandler, which handles the actual SSO auth flow.
// Supports hot-reload: after saving config, admins can call POST /settings/oidc/reload
// to apply changes without restarting the pod.
type OIDCSettingsHandler struct {
	settingSvc *service.SystemSettingService
	reloadFn   func(ctx context.Context) error // set via SetReloadFn
}

// NewOIDCSettingsHandler creates a new OIDCSettingsHandler.
func NewOIDCSettingsHandler(settingSvc *service.SystemSettingService) *OIDCSettingsHandler {
	return &OIDCSettingsHandler{settingSvc: settingSvc}
}

// SetReloadFn sets the function called by the Reload endpoint (P1-9 hot-reload).
func (h *OIDCSettingsHandler) SetReloadFn(fn func(ctx context.Context) error) {
	h.reloadFn = fn
}

// GetConfig returns the current OIDC configuration.
// The client_secret is masked if set (non-empty placeholder "********").
func (h *OIDCSettingsHandler) GetConfig(c *gin.Context) {
	cfg, err := h.settingSvc.GetOIDCConfig(c.Request.Context())
	if err != nil {
		ErrorWithMessage(c, 50003, "failed to load OIDC config: "+err.Error())
		return
	}
	// Mask the secret — never send it back to the browser.
	if cfg.ClientSecret != "" {
		cfg.ClientSecret = "********"
	}
	Success(c, cfg)
}

// UpdateConfig updates the OIDC configuration.
// If client_secret is empty or "********", the existing stored secret is preserved.
func (h *OIDCSettingsHandler) UpdateConfig(c *gin.Context) {
	var req service.OIDCConfigDB
	if err := c.ShouldBindJSON(&req); err != nil {
		ErrorWithMessage(c, 10001, err.Error())
		return
	}
	// Treat the masked placeholder as "don't change the secret".
	if req.ClientSecret == "********" {
		req.ClientSecret = ""
	}
	if err := h.settingSvc.SaveOIDCConfig(c.Request.Context(), req); err != nil {
		ErrorWithMessage(c, 50003, "failed to save OIDC config: "+err.Error())
		return
	}
	Success(c, gin.H{"message": "OIDC configuration updated. Call POST /settings/oidc/reload to apply without restart."})
}

// Reload reinitializes the OIDC service from the current DB config.
// POST /api/v1/settings/oidc/reload (admin only)
func (h *OIDCSettingsHandler) Reload(c *gin.Context) {
	if h.reloadFn == nil {
		ErrorWithMessage(c, 50003, "OIDC reload not available")
		return
	}
	if err := h.reloadFn(c.Request.Context()); err != nil {
		ErrorWithMessage(c, 50003, "failed to reload OIDC: "+err.Error())
		return
	}
	Success(c, gin.H{"message": "OIDC service reloaded successfully"})
}
