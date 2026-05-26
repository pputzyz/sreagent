package router

import (
	"github.com/gin-gonic/gin"
)

// registerAISkillRoutes registers AI skill API routes.
func (h *Handlers) registerAISkillRoutes(auth *gin.RouterGroup, manage gin.HandlerFunc) {
	if h.AISkill == nil {
		return
	}
	skills := auth.Group("/ai-skills")
	{
		skills.GET("", h.AISkill.List)
		skills.GET("/:id", h.AISkill.Get)
		skills.POST("", manage, h.AISkill.Create)
		skills.PUT("/:id", manage, h.AISkill.Update)
		skills.DELETE("/:id", manage, h.AISkill.Delete)
		skills.POST("/import", manage, h.AISkill.Import)
		skills.GET("/:id/files", h.AISkill.GetFiles)
		skills.POST("/:id/files", manage, h.AISkill.AddFile)
		skills.GET("/files/:fileId", h.AISkill.GetFile)
		skills.DELETE("/files/:fileId", manage, h.AISkill.DeleteFile)
	}
}
