package router

import (
	"github.com/gin-gonic/gin"

	"github.com/sreagent/sreagent/internal/middleware"
)

// registerAlertForwarderRoutes registers all alert forwarder routes.
func (h *Handlers) registerAlertForwarderRoutes(auth *gin.RouterGroup, public *gin.RouterGroup, manage, operate gin.HandlerFunc) {
	if h.AlertForwarder == nil {
		return
	}

	// Authenticated routes (CRUD + batch operations)
	forwarders := auth.Group("/alert-forwarders")
	{
		// List and stats
		forwarders.GET("", h.AlertForwarder.List)
		forwarders.GET("/stats", h.AlertForwarder.GetStats)

		// CRUD
		forwarders.POST("", manage, middleware.RequirePerm("forwarder.write"), h.AlertForwarder.Create)
		forwarders.GET("/:id", h.AlertForwarder.GetByID)
		forwarders.PUT("/:id", manage, middleware.RequirePerm("forwarder.write"), h.AlertForwarder.Update)
		forwarders.DELETE("/:id", manage, middleware.RequirePerm("forwarder.write"), h.AlertForwarder.Delete)

		// Enable/Disable
		forwarders.POST("/:id/enable", manage, middleware.RequirePerm("forwarder.write"), h.AlertForwarder.Enable)
		forwarders.POST("/:id/disable", manage, middleware.RequirePerm("forwarder.write"), h.AlertForwarder.Disable)

		// Batch operations
		forwarders.POST("/batch/enable", manage, middleware.RequirePerm("forwarder.write"), h.AlertForwarder.BatchEnable)
		forwarders.POST("/batch/disable", manage, middleware.RequirePerm("forwarder.write"), h.AlertForwarder.BatchDisable)
		forwarders.POST("/batch/delete", manage, middleware.RequirePerm("forwarder.write"), h.AlertForwarder.BatchDelete)

		// Test
		forwarders.POST("/:id/test", manage, middleware.RequirePerm("forwarder.write"), h.AlertForwarder.TestForwarder)
	}

	// Public inbound webhook endpoint (no auth - forwarder authenticates itself)
	// This endpoint is intentionally outside the auth group because external systems
	// call it directly. Authentication is handled at the forwarder level.
	public.POST("/alert-forwarders/:id/inbound", h.AlertForwarder.HandleInbound)
}
