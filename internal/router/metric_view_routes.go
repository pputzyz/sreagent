package router

import (
	"github.com/gin-gonic/gin"
)

// registerMetricViewRoutes registers metric view API routes.
func (h *Handlers) registerMetricViewRoutes(auth *gin.RouterGroup, manage gin.HandlerFunc) {
	if h.MetricView == nil {
		return
	}
	mv := auth.Group("/metric-views")
	{
		mv.GET("", h.MetricView.List)
		mv.GET("/:id", h.MetricView.Get)
		mv.POST("", manage, h.MetricView.Create)
		mv.PUT("/:id", manage, h.MetricView.Update)
		mv.DELETE("/:id", manage, h.MetricView.Delete)
	}
}
