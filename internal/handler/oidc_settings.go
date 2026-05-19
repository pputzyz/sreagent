package handler

import (
	"github.com/gin-gonic/gin"

	apperr "github.com/sreagent/sreagent/internal/pkg/errors"
	"github.com/sreagent/sreagent/internal/service"
)

// OIDCSettingsHandler manages OIDC configuration stored in the DB.
// This is separate from OIDCHandler, which handles the actual SSO auth flow.
type OIDCSettingsHandler struct {
	settingSvc *service.SystemSettingService
	onReload   func() // called after config save to trigger hot-reload (may be nil)
}

// NewOIDCSettingsHandler creates a new OIDCSettingsHandler.
func NewOIDCSettingsHandler(settingSvc *service.SystemSettingService, onReload func()) *OIDCSettingsHandler {
	return &OIDCSettingsHandler{settingSvc: settingSvc, onReload: onReload}
}

// GetConfig returns the current OIDC configuration.
// The client_secret is masked if set (non-empty placeholder "********").
func (h *OIDCSettingsHandler) GetConfig(c *gin.Context) {
	cfg, err := h.settingSvc.GetOIDCConfig(c.Request.Context())
	if err != nil {
		Error(c, apperr.WithMessage(apperr.ErrExternalAPI, "failed to load OIDC config: "+err.Error()))
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
		Error(c, apperr.WithMessage(apperr.ErrInvalidParam, err.Error()))
		return
	}
	// Treat the masked placeholder as "don't change the secret".
	if req.ClientSecret == "********" {
		req.ClientSecret = ""
	}
	if err := h.settingSvc.SaveOIDCConfig(c.Request.Context(), req); err != nil {
		Error(c, apperr.WithMessage(apperr.ErrExternalAPI, "failed to save OIDC config: "+err.Error()))
		return
	}

	// Trigger hot-reload of the OIDC service if a callback is configured.
	if h.onReload != nil {
		h.onReload()
	}

	Success(c, gin.H{"message": "OIDC configuration updated successfully."})
}
