package router

import (
	"github.com/gin-gonic/gin"

	"github.com/sreagent/sreagent/internal/middleware"
)

func (h *Handlers) registerRecordingRuleRoutes(auth *gin.RouterGroup, adminOnly, manage gin.HandlerFunc) {
	rr := auth.Group("/recording-rules")
	{
		rr.GET("", h.RecordingRule.List)
		rr.GET("/:id", h.RecordingRule.Get)
		rr.POST("", manage, middleware.RequirePerm("rules.write"), h.RecordingRule.Create)
		rr.PUT("/:id", manage, middleware.RequirePerm("rules.write"), h.RecordingRule.Update)
		rr.DELETE("/:id", manage, middleware.RequirePerm("rules.write"), h.RecordingRule.Delete)
		rr.POST("/batch", manage, middleware.RequirePerm("rules.write"), h.RecordingRule.BatchCreate)
		rr.POST("/batch-delete", manage, middleware.RequirePerm("rules.write"), h.RecordingRule.BatchDelete)
		rr.PUT("/fields", manage, middleware.RequirePerm("rules.write"), h.RecordingRule.UpdateFields)
	}
}
