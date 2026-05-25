package router

import (
	"github.com/gin-gonic/gin"
)

// registerAnnotationRoutes registers annotation API routes.
func (h *Handlers) registerAnnotationRoutes(auth *gin.RouterGroup, manage gin.HandlerFunc) {
	if h.Annotation == nil {
		return
	}
	ann := auth.Group("/annotations")
	{
		ann.GET("", h.Annotation.List)
		ann.POST("", manage, h.Annotation.Create)
		ann.PUT("/:id", manage, h.Annotation.Update)
		ann.DELETE("/:id", manage, h.Annotation.Delete)
		ann.POST("/batch", manage, h.Annotation.BatchCreate)
	}
}
