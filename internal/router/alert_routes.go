package router

import (
	"github.com/gin-gonic/gin"
)

// registerAlertRoutes registers alert rule, alert event, heartbeat, and alert action routes.
// root is the top-level engine (for unauthenticated endpoints like heartbeat).
// auth is the JWT-authenticated group.
func (h *Handlers) registerAlertRoutes(root *gin.Engine, auth *gin.RouterGroup, manage, operate gin.HandlerFunc) {
	// Alert Rules
	rules := auth.Group("/alert-rules")
	{
		rules.GET("", h.AlertRule.List)
		rules.GET("/:id", h.AlertRule.Get)
		rules.GET("/categories", h.AlertRule.ListCategories)
		rules.GET("/export", h.AlertRule.Export)
		rules.POST("", manage, h.AlertRule.Create)
		rules.PUT("/:id", manage, h.AlertRule.Update)
		rules.DELETE("/:id", manage, h.AlertRule.Delete)
		rules.PATCH("/:id/status", manage, h.AlertRule.ToggleStatus)
		rules.POST("/import", manage, h.AlertRule.Import)
		rules.POST("/batch/enable", manage, h.AlertRule.BatchEnable)
		rules.POST("/batch/disable", manage, h.AlertRule.BatchDisable)
		rules.POST("/batch/delete", manage, h.AlertRule.BatchDelete)
	}

	// Alert Rule Templates
	if h.AlertRuleTemplate != nil {
		templates := auth.Group("/alert-rule-templates")
		{
			templates.GET("", h.AlertRuleTemplate.List)
			templates.GET("/categories", h.AlertRuleTemplate.ListCategories)
			templates.GET("/:id", h.AlertRuleTemplate.Get)
			templates.POST("", manage, h.AlertRuleTemplate.Create)
			templates.PUT("/:id", manage, h.AlertRuleTemplate.Update)
			templates.DELETE("/:id", manage, h.AlertRuleTemplate.Delete)
			templates.POST("/:id/apply", manage, h.AlertRuleTemplate.Apply)
		}
	}

	// Preset Rules
	if h.PresetRule != nil {
		presets := auth.Group("/preset-rules")
		{
			presets.GET("", h.PresetRule.List)
			presets.GET("/categories", h.PresetRule.Categories)
			presets.GET("/:id", h.PresetRule.Get)
			presets.POST("/:id/apply", manage, h.PresetRule.Apply)
			presets.POST("/import", manage, h.PresetRule.Import)
			presets.DELETE("/:id", manage, h.PresetRule.Delete)
		}
	}

	// Alert Events
	events := auth.Group("/alert-events")
	{
		events.GET("", h.AlertEvent.List)
		events.GET("/export", h.AlertEvent.Export)
		events.GET("/groups", h.AlertEvent.ListGroups)
		events.GET("/:id", h.AlertEvent.Get)
		events.GET("/:id/timeline", h.AlertEvent.GetTimeline)
		events.POST("/:id/acknowledge", operate, h.AlertEvent.Acknowledge)
		events.POST("/:id/assign", operate, h.AlertEvent.Assign)
		events.POST("/:id/resolve", operate, h.AlertEvent.Resolve)
		events.POST("/:id/close", operate, h.AlertEvent.Close)
		events.POST("/:id/comment", operate, h.AlertEvent.AddComment)
		events.POST("/:id/silence", operate, h.AlertEvent.Silence)
		events.POST("/batch/acknowledge", operate, h.AlertEvent.BatchAcknowledge)
		events.POST("/batch/close", operate, h.AlertEvent.BatchClose)
	}

	// Alerts v2 (告警)
	if h.AlertV2 != nil {
		alertsV2 := auth.Group("/alerts")
		{
			alertsV2.GET("", h.AlertV2.List)
			alertsV2.GET("/:id", h.AlertV2.Get)
			alertsV2.GET("/:id/events", h.AlertV2.ListEvents)
		}
	}

	// Heartbeat ping endpoint (no auth — token authenticates the sender)
	if h.Heartbeat != nil {
		root.POST("/heartbeat/:token", h.Heartbeat.Ping)
	}

	// Alert action page (no auth - token-based)
	if h.AlertAction != nil {
		root.GET("/alert-action/:token", h.AlertAction.ActionPage)
		root.POST("/alert-action/:token", h.AlertAction.ExecuteAction)
	}
}
