package router

import (
	"github.com/gin-gonic/gin"

	"github.com/sreagent/sreagent/internal/middleware"
)

func (h *Handlers) registerBuiltinMetricRoutes(auth *gin.RouterGroup, adminOnly, manage gin.HandlerFunc) {
	bm := auth.Group("/builtin-metrics")
	{
		bm.GET("", h.BuiltinMetric.List)
		bm.GET("/:id", h.BuiltinMetric.Get)
		bm.POST("", manage, middleware.RequirePerm("metrics.write"), h.BuiltinMetric.Create)
		bm.PUT("", manage, middleware.RequirePerm("metrics.write"), h.BuiltinMetric.Update)
		bm.POST("/delete", manage, middleware.RequirePerm("metrics.write"), h.BuiltinMetric.Delete)
		bm.GET("/types", h.BuiltinMetric.Types)
		bm.GET("/collectors", h.BuiltinMetric.Collectors)
		bm.POST("/batch", manage, middleware.RequirePerm("metrics.write"), h.BuiltinMetric.BatchCreate)
	}

	// Metric filters
	mf := auth.Group("/builtin-metric-filters")
	{
		mf.GET("", h.BuiltinMetric.ListFilters)
		mf.POST("", h.BuiltinMetric.CreateFilter)
		mf.PUT("", h.BuiltinMetric.UpdateFilter)
		mf.POST("/delete", h.BuiltinMetric.DeleteFilter)
	}
}
