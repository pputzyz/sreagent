package router

import (
	"github.com/gin-gonic/gin"
)

// registerAuthRoutes registers authenticated user profile and OIDC settings routes.
func (h *Handlers) registerAuthRoutes(auth *gin.RouterGroup, admin gin.HandlerFunc) {
	// Current user (self) — any authenticated user
	auth.GET("/auth/profile", h.Auth.GetProfile)
	auth.PUT("/me/profile", h.Auth.UpdateMe)
	auth.POST("/me/password", h.Auth.ChangeMyPassword)
	auth.PUT("/me/lark-bind", h.Auth.BindLark)

	// OIDC settings — admin only (separate from /auth/oidc/* which is the SSO auth flow)
	if h.OIDCSettings != nil {
		oidcSettings := auth.Group("/settings/oidc")
		{
			oidcSettings.GET("", admin, h.OIDCSettings.GetConfig)
			oidcSettings.PUT("", admin, h.OIDCSettings.UpdateConfig)
			oidcSettings.POST("/reload", admin, h.OIDCSettings.Reload)
		}
	}
}
