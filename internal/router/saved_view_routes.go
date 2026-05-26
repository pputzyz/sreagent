package router

import (
	"github.com/gin-gonic/gin"
)

// registerSavedViewRoutes registers saved view API routes.
func (h *Handlers) registerSavedViewRoutes(auth *gin.RouterGroup, manage gin.HandlerFunc) {
	if h.SavedView == nil {
		return
	}
	sv := auth.Group("/saved-views")
	{
		sv.GET("", h.SavedView.List)
		sv.GET("/:id", h.SavedView.Get)
		sv.POST("", manage, h.SavedView.Create)
		sv.PUT("/:id", manage, h.SavedView.Update)
		sv.DELETE("/:id", manage, h.SavedView.Delete)
		sv.POST("/:id/copy", manage, h.SavedView.Copy)
	}
}
