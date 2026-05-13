package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/sreagent/sreagent/internal/service"
)

// OIDCHandler handles OIDC login flow endpoints.
type OIDCHandler struct {
	svc *service.OIDCService
}

// NewOIDCHandler creates a new OIDC handler.
func NewOIDCHandler(svc *service.OIDCService) *OIDCHandler {
	return &OIDCHandler{svc: svc}
}

// LoginRedirect initiates the OIDC authorization code flow.
// GET /api/v1/auth/oidc/login
// Redirects the browser to the IdP's authorization endpoint.
func (h *OIDCHandler) LoginRedirect(c *gin.Context) {
	authURL, state, err := h.svc.GenerateAuthURL()
	if err != nil {
		ErrorWithMessage(c, 50000, "failed to generate OIDC auth URL: "+err.Error())
		return
	}

	// Store state in a secure cookie for CSRF protection
	c.SetSameSite(http.SameSiteLaxMode)
	secure := c.Request.TLS != nil || c.GetHeader("X-Forwarded-Proto") == "https"
	c.SetCookie("oidc_state", state, 300, "/", "", secure, true)

	c.Redirect(http.StatusFound, authURL)
}

// Callback handles the OIDC callback after IdP authentication.
// GET /api/v1/auth/oidc/callback?code=...&state=...
// On success, redirects to the frontend with a token parameter.
func (h *OIDCHandler) Callback(c *gin.Context) {
	// Verify state for CSRF protection
	expectedState, err := c.Cookie("oidc_state")
	if err != nil || expectedState == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    10100,
			"message": "missing or expired OIDC state cookie",
		})
		return
	}

	actualState := c.Query("state")
	if actualState != expectedState {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    10100,
			"message": "OIDC state mismatch (possible CSRF)",
		})
		return
	}

	// Clear the state cookie
	secure := c.Request.TLS != nil || c.GetHeader("X-Forwarded-Proto") == "https"
	c.SetCookie("oidc_state", "", -1, "/", "", secure, true)

	// Check for error from IdP
	if errParam := c.Query("error"); errParam != "" {
		errDesc := c.Query("error_description")
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    10102,
			"message": "OIDC authentication failed: " + errParam + " - " + errDesc,
		})
		return
	}

	code := c.Query("code")
	if code == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    10001,
			"message": "missing authorization code",
		})
		return
	}

	// Exchange code for tokens and create/login user
	token, expiresIn, err := h.svc.ExchangeAndLogin(c.Request.Context(), code)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"code":    10102,
			"message": "OIDC login failed: " + err.Error(),
		})
		return
	}

	// For SPA: redirect to frontend with token as URL fragment (not query param for security)
	// The frontend will extract the token from the fragment and store it.
	redirectURL := "/#oidc_token=" + token + "&expires_in=" + strconv.Itoa(expiresIn)
	c.Redirect(http.StatusFound, redirectURL)
}

// CallbackJSON is an alternative callback that returns JSON instead of redirecting.
// POST /api/v1/auth/oidc/token
// Accepts {"code": "...", "state": "..."} and returns {"token": "...", "expires_in": ...}
// This is useful for SPAs that handle the redirect themselves.
func (h *OIDCHandler) CallbackJSON(c *gin.Context) {
	var req struct {
		Code  string `json:"code" binding:"required"`
		State string `json:"state"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		ErrorWithMessage(c, 10001, err.Error())
		return
	}

	// Validate CSRF state (mandatory)
	if req.State == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    10100,
			"message": "missing OIDC state parameter (required for CSRF protection)",
		})
		return
	}
	expectedState, err := c.Cookie("oidc_state")
	if err != nil || expectedState == "" || req.State != expectedState {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    10100,
			"message": "OIDC state mismatch (possible CSRF)",
		})
		return
	}
	// Clear the state cookie
	secure := c.Request.TLS != nil || c.GetHeader("X-Forwarded-Proto") == "https"
	c.SetCookie("oidc_state", "", -1, "/", "", secure, true)

	token, expiresIn, err := h.svc.ExchangeAndLogin(c.Request.Context(), req.Code)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"code":    10102,
			"message": "OIDC login failed: " + err.Error(),
		})
		return
	}

	Success(c, LoginResponse{
		Token:     token,
		ExpiresIn: expiresIn,
	})
}

// OIDCConfig returns the OIDC configuration for the frontend.
// GET /api/v1/auth/oidc/config
// Returns issuer, client_id, scopes (no secrets).
func (h *OIDCHandler) OIDCConfig(c *gin.Context) {
	enabled := h.svc != nil && h.svc.Enabled()
	if !enabled {
		Success(c, gin.H{"enabled": false})
		return
	}

	Success(c, gin.H{
		"enabled":   true,
		"login_url": "/api/v1/auth/oidc/login",
	})
}
