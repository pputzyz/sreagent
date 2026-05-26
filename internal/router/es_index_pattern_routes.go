package router

import (
	"github.com/gin-gonic/gin"
)

// registerESIndexPatternRoutes registers ES index pattern API routes.
func (h *Handlers) registerESIndexPatternRoutes(auth *gin.RouterGroup, manage gin.HandlerFunc) {
	if h.ESIndexPattern == nil {
		return
	}
	eip := auth.Group("/es-index-patterns")
	{
		eip.GET("", h.ESIndexPattern.List)
		eip.GET("/:id", h.ESIndexPattern.Get)
		eip.POST("", manage, h.ESIndexPattern.Create)
		eip.PUT("/:id", manage, h.ESIndexPattern.Update)
		eip.DELETE("/:id", manage, h.ESIndexPattern.Delete)
	}
}
