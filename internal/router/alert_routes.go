package router

import (
	"github.com/gin-gonic/gin"

	"github.com/sreagent/sreagent/internal/middleware"
)

// registerAlertRoutes registers alert rule, alert event, heartbeat, and alert action routes.
// root is the top-level engine (for unauthenticated endpoints like heartbeat).
// auth is the JWT-authenticated group.
func (h *Handlers) registerAlertRoutes(root *gin.Engine, auth *gin.RouterGroup, adminOnly, manage, operate gin.HandlerFunc) {
	// Alert Rules
	rules := auth.Group("/alert-rules")
	{
		rules.GET("", h.AlertRule.List)
		rules.GET("/:id", h.AlertRule.Get)
		rules.GET("/categories", h.AlertRule.ListCategories)
		rules.GET("/export", manage, h.AlertRule.Export)
		rules.GET("/label-validation-preview", manage, h.AlertRule.LabelValidationPreview)
		rules.POST("", manage, middleware.RequirePerm("rules.write"), h.AlertRule.Create)
		rules.PUT("/:id", manage, middleware.RequirePerm("rules.write"), h.AlertRule.Update)
		rules.DELETE("/:id", manage, middleware.RequirePerm("rules.write"), h.AlertRule.Delete)
		rules.PATCH("/:id/status", manage, middleware.RequirePerm("rules.write"), h.AlertRule.ToggleStatus)
		rules.POST("/import", manage, middleware.RequirePerm("rules.write"), h.AlertRule.Import)
		rules.POST("/batch/enable", manage, middleware.RequirePerm("rules.write"), h.AlertRule.BatchEnable)
		rules.POST("/batch/disable", manage, middleware.RequirePerm("rules.write"), h.AlertRule.BatchDisable)
		rules.POST("/batch/delete", manage, middleware.RequirePerm("rules.write"), h.AlertRule.BatchDelete)
		rules.GET("/:id/heartbeat-token", adminOnly, h.AlertRule.GetHeartbeatToken)
	}

	// Alert Rule Templates
	if h.AlertRuleTemplate != nil {
		templates := auth.Group("/alert-rule-templates")
		{
			templates.GET("", h.AlertRuleTemplate.List)
			templates.GET("/categories", h.AlertRuleTemplate.ListCategories)
			templates.GET("/:id", h.AlertRuleTemplate.Get)
			templates.POST("", manage, middleware.RequirePerm("rules.write"), h.AlertRuleTemplate.Create)
			templates.PUT("/:id", manage, middleware.RequirePerm("rules.write"), h.AlertRuleTemplate.Update)
			templates.DELETE("/:id", manage, middleware.RequirePerm("rules.write"), h.AlertRuleTemplate.Delete)
			templates.POST("/:id/apply", manage, middleware.RequirePerm("rules.write"), h.AlertRuleTemplate.Apply)
		}
	}

	// Preset Rules
	if h.PresetRule != nil {
		presets := auth.Group("/preset-rules")
		{
			presets.GET("", h.PresetRule.List)
			presets.GET("/categories", h.PresetRule.Categories)
			presets.GET("/:id", h.PresetRule.Get)
			presets.POST("/:id/apply", manage, middleware.RequirePerm("rules.write"), h.PresetRule.Apply)
			presets.POST("/batch-apply", manage, middleware.RequirePerm("rules.write"), h.PresetRule.BatchApply)
			presets.POST("/import", manage, middleware.RequirePerm("rules.write"), h.PresetRule.Import)
			presets.DELETE("/:id", manage, middleware.RequirePerm("rules.write"), h.PresetRule.Delete)
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
