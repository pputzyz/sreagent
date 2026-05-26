package router

import (
	"github.com/gin-gonic/gin"
)

// registerLLMConfigRoutes registers LLM config API routes.
func (h *Handlers) registerLLMConfigRoutes(auth *gin.RouterGroup, manage gin.HandlerFunc) {
	if h.LLMConfig == nil {
		return
	}
	lc := auth.Group("/llm-configs")
	{
		lc.GET("", h.LLMConfig.List)
		lc.GET("/:id", h.LLMConfig.Get)
		lc.POST("", manage, h.LLMConfig.Create)
		lc.PUT("/:id", manage, h.LLMConfig.Update)
		lc.DELETE("/:id", manage, h.LLMConfig.Delete)
		lc.POST("/test", manage, h.LLMConfig.TestConnection)
	}
}
