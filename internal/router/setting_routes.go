package router

import (
	"github.com/gin-gonic/gin"

	"github.com/sreagent/sreagent/internal/middleware"
)

// registerSettingRoutes registers settings, AI, audit, and system routes.
func (h *Handlers) registerSettingRoutes(auth *gin.RouterGroup, adminOnly, manage, operate gin.HandlerFunc) {
	// Audit Logs — admin only
	auth.GET("/audit-logs", adminOnly, h.AuditLog.List)

	// SMTP settings — admin only
	if h.SMTPSettings != nil {
		smtpSettings := auth.Group("/settings/smtp")
		{
			smtpSettings.GET("", adminOnly, h.SMTPSettings.GetConfig)
			smtpSettings.PUT("", adminOnly, h.SMTPSettings.UpdateConfig)
			smtpSettings.POST("/test", adminOnly, h.SMTPSettings.TestConnection)
		}
	}

	// Security settings — admin only
	if h.SecuritySettings != nil {
		secSettings := auth.Group("/settings/security")
		{
			secSettings.GET("", adminOnly, h.SecuritySettings.GetConfig)
			secSettings.PUT("", adminOnly, h.SecuritySettings.UpdateConfig)
		}
	}

	// AI — config is admin only, usage is for all
	// Rate limit: 1 RPS, burst 10 for AI inference endpoints
	aiRL := middleware.RateLimit(func(c *gin.Context) string {
		return "ai:" + c.ClientIP()
	}, 1, 10)
	ai := auth.Group("/ai")
	{
		ai.POST("/alert-report", aiRL, h.AI.GenerateReport)
		ai.POST("/suggest-sop", aiRL, h.AI.SuggestSOP)
		ai.POST("/test", manage, h.AI.TestConnection)
		ai.GET("/config", adminOnly, h.AI.GetConfig)
		ai.PUT("/config", adminOnly, h.AI.UpdateConfig)
		ai.POST("/chat", aiRL, h.AI.Chat)
		ai.GET("/history", h.AI.GetHistory)
		ai.DELETE("/history", h.AI.ClearHistory)
		ai.GET("/modules", adminOnly, h.AI.GetModules)
		ai.PUT("/modules", adminOnly, h.AI.UpdateModules)
		ai.GET("/providers", adminOnly, h.AI.GetProviders)
		ai.PUT("/providers", adminOnly, h.AI.SaveProviders)
		ai.POST("/test-provider", adminOnly, h.AI.TestProvider)
	}

	// Engine status (simple, no process management)
	if h.Engine != nil {
		auth.GET("/engine/status", h.Engine.GetStatus)
	}

	// AI Rule Generation — rate limited
	if h.AIRule != nil {
		aiRules := auth.Group("/ai/rules", operate, aiRL)
		{
			aiRules.POST("/generate", h.AIRule.Generate)
			aiRules.POST("/validate", h.AIRule.Validate)
			aiRules.POST("/suggest-labels", h.AIRule.SuggestLabels)
			aiRules.POST("/generate-inhibition", h.AIRule.GenerateInhibition)
		}
	}

	// Lark Bot config — admin only
	larkBot := auth.Group("/lark/bot")
	{
		larkBot.GET("/config", adminOnly, h.LarkBot.GetConfig)
		larkBot.PUT("/config", adminOnly, h.LarkBot.UpdateConfig)
	}

	// Status Page services (状态页面)
	if h.StatusService != nil {
		statusSvc := auth.Group("/status-services")
		{
			statusSvc.GET("", h.StatusService.List)
			statusSvc.GET("/:id", h.StatusService.Get)
			statusSvc.POST("", adminOnly, h.StatusService.Create)
			statusSvc.PUT("/:id", adminOnly, h.StatusService.Update)
			statusSvc.DELETE("/:id", adminOnly, h.StatusService.Delete)
		}
	}
}
