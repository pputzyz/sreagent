package handler

import (
	"github.com/gin-gonic/gin"

	apperr "github.com/sreagent/sreagent/internal/pkg/errors"
	"github.com/sreagent/sreagent/internal/service"
)

// SSOSettingsHandler manages LDAP and OAuth2 configuration stored in the DB.
type SSOSettingsHandler struct {
	ldapSvc   *service.LDAPService
	oauth2Svc *service.OAuth2Service
}

// NewSSOSettingsHandler creates a new SSOSettingsHandler.
func NewSSOSettingsHandler(ldapSvc *service.LDAPService, oauth2Svc *service.OAuth2Service) *SSOSettingsHandler {
	return &SSOSettingsHandler{ldapSvc: ldapSvc, oauth2Svc: oauth2Svc}
}

// ---- LDAP Settings ----

// GetLDAPConfig returns the current LDAP configuration.
// The bind_password is masked if set.
func (h *SSOSettingsHandler) GetLDAPConfig(c *gin.Context) {
	if h.ldapSvc == nil {
		Error(c, apperr.WithMessage(apperr.ErrInternal, "LDAP service not available"))
		return
	}
	cfg, err := h.ldapSvc.GetConfig(c.Request.Context())
	if err != nil {
		Error(c, apperr.WithMessage(apperr.ErrExternalAPI, "failed to load LDAP config: "+err.Error()))
		return
	}
	// Mask the password — never send it back to the browser.
	if cfg.BindPassword != "" {
		cfg.BindPassword = "********"
	}
	Success(c, cfg)
}

// UpdateLDAPConfig updates the LDAP configuration.
// If bind_password is empty or "********", the existing stored password is preserved.
func (h *SSOSettingsHandler) UpdateLDAPConfig(c *gin.Context) {
	if h.ldapSvc == nil {
		Error(c, apperr.WithMessage(apperr.ErrInternal, "LDAP service not available"))
		return
	}
	var req service.LDAPConfig
	if err := c.ShouldBindJSON(&req); err != nil {
		Error(c, apperr.WithMessage(apperr.ErrInvalidParam, err.Error()))
		return
	}
	// Treat the masked placeholder as "don't change the password".
	if req.BindPassword == "********" {
		req.BindPassword = ""
	}
	if err := h.ldapSvc.SaveConfig(c.Request.Context(), req); err != nil {
		Error(c, apperr.WithMessage(apperr.ErrExternalAPI, "failed to save LDAP config: "+err.Error()))
		return
	}
	Success(c, gin.H{"message": "LDAP configuration updated successfully."})
}

// TestLDAPConnection tests the LDAP connection by performing a bind.
func (h *SSOSettingsHandler) TestLDAPConnection(c *gin.Context) {
	if h.ldapSvc == nil {
		Error(c, apperr.WithMessage(apperr.ErrInternal, "LDAP service not available"))
		return
	}
	msg, err := h.ldapSvc.TestConnection(c.Request.Context())
	if err != nil {
		Success(c, gin.H{"success": false, "message": err.Error()})
		return
	}
	Success(c, gin.H{"success": true, "message": msg})
}

// ---- OAuth2 Settings ----

// GetOAuth2Config returns the current OAuth2 configuration.
// The client_secret is masked if set.
func (h *SSOSettingsHandler) GetOAuth2Config(c *gin.Context) {
	if h.oauth2Svc == nil {
		Error(c, apperr.WithMessage(apperr.ErrInternal, "OAuth2 service not available"))
		return
	}
	cfg, err := h.oauth2Svc.GetConfig(c.Request.Context())
	if err != nil {
		Error(c, apperr.WithMessage(apperr.ErrExternalAPI, "failed to load OAuth2 config: "+err.Error()))
		return
	}
	// Mask the secret — never send it back to the browser.
	if cfg.ClientSecret != "" {
		cfg.ClientSecret = "********"
	}
	Success(c, cfg)
}

// UpdateOAuth2Config updates the OAuth2 configuration.
// If client_secret is empty or "********", the existing stored secret is preserved.
func (h *SSOSettingsHandler) UpdateOAuth2Config(c *gin.Context) {
	if h.oauth2Svc == nil {
		Error(c, apperr.WithMessage(apperr.ErrInternal, "OAuth2 service not available"))
		return
	}
	var req service.OAuth2Config
	if err := c.ShouldBindJSON(&req); err != nil {
		Error(c, apperr.WithMessage(apperr.ErrInvalidParam, err.Error()))
		return
	}
	// Treat the masked placeholder as "don't change the secret".
	if req.ClientSecret == "********" {
		req.ClientSecret = ""
	}
	if err := h.oauth2Svc.SaveConfig(c.Request.Context(), req); err != nil {
		Error(c, apperr.WithMessage(apperr.ErrExternalAPI, "failed to save OAuth2 config: "+err.Error()))
		return
	}
	Success(c, gin.H{"message": "OAuth2 configuration updated successfully."})
}
