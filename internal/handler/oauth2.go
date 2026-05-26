package handler

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	apperr "github.com/sreagent/sreagent/internal/pkg/errors"
	"github.com/sreagent/sreagent/internal/service"
)

// OAuth2Handler handles OAuth2 SSO login flow endpoints.
type OAuth2Handler struct {
	svc     *service.OAuth2Service
	jwtCfg  *jwtConfigAdapter
}

// jwtConfigAdapter wraps the JWT config for the OAuth2 service.
type jwtConfigAdapter struct {
	secret string
	expire int
}

// NewOAuth2Handler creates a new OAuth2 handler.
func NewOAuth2Handler(svc *service.OAuth2Service, jwtSecret string, jwtExpire int) *OAuth2Handler {
	return &OAuth2Handler{
		svc: svc,
		jwtCfg: &jwtConfigAdapter{
			secret: jwtSecret,
			expire: jwtExpire,
		},
	}
}

// logError logs an error with full details server-side using the request-scoped zap logger.
func (h *OAuth2Handler) logError(c *gin.Context, msg string, err error) {
	if l, exists := c.Get("logger"); exists {
		if logger, ok := l.(*zap.Logger); ok {
			logger.Error(msg, zap.Error(err))
		}
	}
}

// LoginRedirect initiates the OAuth2 authorization code flow.
// GET /api/v1/auth/oauth2/login
// Redirects the browser to the OAuth2 provider's authorization endpoint.
func (h *OAuth2Handler) LoginRedirect(c *gin.Context) {
	if h.svc == nil || !h.svc.Enabled() {
		Error(c, apperr.WithMessage(apperr.ErrForbidden, "OAuth2 authentication is disabled"))
		return
	}

	authURL, state, err := h.svc.GetAuthURL(c.Request.Context())
	if err != nil {
		h.logError(c, "failed to generate OAuth2 auth URL", err)
		Error(c, apperr.WithMessage(apperr.ErrInternal, "failed to initiate OAuth2 login"))
		return
	}

	// Store state in a secure cookie for CSRF protection
	c.SetSameSite(http.SameSiteLaxMode)
	secure := c.Request.TLS != nil || c.GetHeader("X-Forwarded-Proto") == "https"
	c.SetCookie("oauth2_state", state, 300, "/", "", secure, true)

	c.Redirect(http.StatusFound, authURL)
}

// Callback handles the OAuth2 callback after provider authentication.
// GET /api/v1/auth/oauth2/callback?code=...&state=...
// On success, redirects to the frontend with a token parameter.
func (h *OAuth2Handler) Callback(c *gin.Context) {
	if h.svc == nil || !h.svc.Enabled() {
		Error(c, apperr.WithMessage(apperr.ErrForbidden, "OAuth2 authentication is disabled"))
		return
	}

	// Verify state for CSRF protection
	expectedState, err := c.Cookie("oauth2_state")
	if err != nil || expectedState == "" {
		Error(c, apperr.WithMessage(apperr.ErrUnauthorized, "missing or expired OAuth2 state"))
		return
	}

	actualState := c.Query("state")
	if actualState != expectedState {
		Error(c, apperr.WithMessage(apperr.ErrUnauthorized, "OAuth2 state mismatch"))
		return
	}

	// Clear the state cookie
	secure := c.Request.TLS != nil || c.GetHeader("X-Forwarded-Proto") == "https"
	c.SetCookie("oauth2_state", "", -1, "/", "", secure, true)

	// Check for error from provider
	if errParam := c.Query("error"); errParam != "" {
		errDesc := c.Query("error_description")
		h.logError(c, "OAuth2 provider returned error", fmt.Errorf("%s: %s", errParam, errDesc))
		Error(c, apperr.WithMessage(apperr.ErrInvalidCreds, "OAuth2 authentication failed"))
		return
	}

	code := c.Query("code")
	if code == "" {
		Error(c, apperr.WithMessage(apperr.ErrInvalidParam, "missing authorization code"))
		return
	}

	// Exchange code for tokens and create/login user
	token, expiresIn, err := h.svc.ExchangeAndLogin(c.Request.Context(), code, h.jwtCfg.secret, h.jwtCfg.expire)
	if err != nil {
		h.logError(c, "OAuth2 token exchange failed", err)
		Error(c, apperr.WithMessage(apperr.ErrInvalidCreds, "OAuth2 authentication failed"))
		return
	}

	// For SPA: redirect to frontend with token as URL fragment
	redirectURL := "/#oauth2_token=" + token + "&expires_in=" + fmt.Sprintf("%d", expiresIn)
	c.Redirect(http.StatusFound, redirectURL)
}

// CallbackJSON is an alternative callback that returns JSON instead of redirecting.
// POST /api/v1/auth/oauth2/token
// Accepts {"code": "...", "state": "..."} and returns {"token": "...", "expires_in": ...}
// This is useful for SPAs that handle the redirect themselves.
func (h *OAuth2Handler) CallbackJSON(c *gin.Context) {
	if h.svc == nil || !h.svc.Enabled() {
		Error(c, apperr.WithMessage(apperr.ErrForbidden, "OAuth2 authentication is disabled"))
		return
	}

	var req struct {
		Code  string `json:"code" binding:"required"`
		State string `json:"state"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		Error(c, apperr.WithMessage(apperr.ErrInvalidParam, err.Error()))
		return
	}

	// Validate CSRF state (mandatory)
	if req.State == "" {
		Error(c, apperr.WithMessage(apperr.ErrUnauthorized, "missing OAuth2 state parameter"))
		return
	}
	expectedState, err := c.Cookie("oauth2_state")
	if err != nil || expectedState == "" || req.State != expectedState {
		Error(c, apperr.WithMessage(apperr.ErrUnauthorized, "OAuth2 state mismatch"))
		return
	}
	// Clear the state cookie
	secure := c.Request.TLS != nil || c.GetHeader("X-Forwarded-Proto") == "https"
	c.SetCookie("oauth2_state", "", -1, "/", "", secure, true)

	token, expiresIn, err := h.svc.ExchangeAndLogin(c.Request.Context(), req.Code, h.jwtCfg.secret, h.jwtCfg.expire)
	if err != nil {
		h.logError(c, "OAuth2 token exchange failed", err)
		Error(c, apperr.WithMessage(apperr.ErrInvalidCreds, "OAuth2 authentication failed"))
		return
	}

	Success(c, LoginResponse{
		Token:     token,
		ExpiresIn: expiresIn,
	})
}

// OAuth2Config returns the OAuth2 configuration for the frontend.
// GET /api/v1/auth/oauth2/config
// Returns name, login_url (no secrets).
func (h *OAuth2Handler) OAuth2Config(c *gin.Context) {
	enabled := h.svc != nil && h.svc.Enabled()
	if !enabled {
		Success(c, gin.H{"enabled": false})
		return
	}

	Success(c, gin.H{
		"enabled":   true,
		"name":      h.svc.GetName(),
		"login_url": "/api/v1/auth/oauth2/login",
	})
}
