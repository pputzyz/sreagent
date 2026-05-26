package router

import (
	"github.com/gin-gonic/gin"
)

// registerMCPServerRoutes registers MCP server API routes.
func (h *Handlers) registerMCPServerRoutes(auth *gin.RouterGroup, manage gin.HandlerFunc) {
	if h.MCPServer == nil {
		return
	}
	mcp := auth.Group("/mcp-servers")
	{
		mcp.GET("", h.MCPServer.List)
		mcp.GET("/:id", h.MCPServer.Get)
		mcp.POST("", manage, h.MCPServer.Create)
		mcp.PUT("/:id", manage, h.MCPServer.Update)
		mcp.DELETE("/:id", manage, h.MCPServer.Delete)
		mcp.POST("/:id/test", manage, h.MCPServer.TestConnection)
		mcp.GET("/:id/tools", h.MCPServer.ListTools)
	}
}
