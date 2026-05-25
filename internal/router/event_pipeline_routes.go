package router

import (
	"github.com/gin-gonic/gin"

	"github.com/sreagent/sreagent/internal/middleware"
)

func (h *Handlers) registerEventPipelineRoutes(auth *gin.RouterGroup, adminOnly, manage gin.HandlerFunc) {
	ep := auth.Group("/event-pipelines")
	{
		ep.GET("", h.EventPipeline.List)
		ep.GET("/:id", h.EventPipeline.Get)
		ep.POST("", manage, middleware.RequirePerm("pipeline.write"), h.EventPipeline.Create)
		ep.PUT("/:id", manage, middleware.RequirePerm("pipeline.write"), h.EventPipeline.Update)
		ep.DELETE("/:id", manage, middleware.RequirePerm("pipeline.write"), h.EventPipeline.Delete)
		ep.GET("/:id/executions", h.EventPipeline.ListExecutions)
		ep.POST("/:id/tryrun", manage, h.EventPipeline.TryRun)
		ep.GET("/processor-types", h.EventPipeline.ListProcessorTypes)
	}

	// Execution records
	exec := auth.Group("/event-pipeline-executions")
	{
		exec.GET("/:id", h.EventPipeline.GetExecution)
		exec.POST("/clean", adminOnly, h.EventPipeline.CleanExecutions)
	}
}
