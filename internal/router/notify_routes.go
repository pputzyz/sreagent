package router

import (
	"github.com/gin-gonic/gin"

	"github.com/sreagent/sreagent/internal/middleware"
)

// registerNotifyRoutes registers all notification-related routes:
// notify rules, media, templates, subscriptions, alert channels, and user notify configs.
func (h *Handlers) registerNotifyRoutes(auth *gin.RouterGroup, manage, operate gin.HandlerFunc) {
	// Notify Rules (v2)
	notifyRules := auth.Group("/notify-rules")
	{
		notifyRules.GET("", h.NotifyRule.List)
		notifyRules.GET("/:id", h.NotifyRule.Get)
		notifyRules.POST("", manage, middleware.RequirePerm("notify.write"), h.NotifyRule.Create)
		notifyRules.PUT("/:id", manage, middleware.RequirePerm("notify.write"), h.NotifyRule.Update)
		notifyRules.DELETE("/:id", manage, middleware.RequirePerm("notify.write"), h.NotifyRule.Delete)
		notifyRules.POST("/batch/enable", manage, middleware.RequirePerm("notify.write"), h.NotifyRule.BatchEnable)
		notifyRules.POST("/batch/disable", manage, middleware.RequirePerm("notify.write"), h.NotifyRule.BatchDisable)
		notifyRules.POST("/batch/delete", manage, middleware.RequirePerm("notify.write"), h.NotifyRule.BatchDelete)
		notifyRules.POST("/batch", manage, middleware.RequirePerm("notify.write"), h.NotifyRule.BatchCreate)
		notifyRules.POST("/:id/test", manage, middleware.RequirePerm("notify.write"), h.NotifyRule.Test)
	}

	// Notify Media
	notifyMedia := auth.Group("/notify-media")
	{
		notifyMedia.GET("", h.NotifyMedia.List)
		notifyMedia.GET("/:id", h.NotifyMedia.Get)
		notifyMedia.POST("", manage, middleware.RequirePerm("notify.write"), h.NotifyMedia.Create)
		notifyMedia.PUT("/:id", manage, middleware.RequirePerm("notify.write"), h.NotifyMedia.Update)
		notifyMedia.DELETE("/:id", manage, middleware.RequirePerm("notify.write"), h.NotifyMedia.Delete)
		notifyMedia.POST("/:id/test", manage, middleware.RequirePerm("notify.write"), h.NotifyMedia.Test)
	}

	// Message Templates
	msgTemplates := auth.Group("/message-templates")
	{
		msgTemplates.GET("", h.MessageTemplate.List)
		msgTemplates.GET("/:id", h.MessageTemplate.Get)
		msgTemplates.POST("", manage, middleware.RequirePerm("notify.write"), h.MessageTemplate.Create)
		msgTemplates.PUT("/:id", manage, middleware.RequirePerm("notify.write"), h.MessageTemplate.Update)
		msgTemplates.DELETE("/:id", manage, middleware.RequirePerm("notify.write"), h.MessageTemplate.Delete)
		msgTemplates.POST("/preview", h.MessageTemplate.Preview)
	}

	// Subscribe Rules — members can manage their own subscriptions
	subscribes := auth.Group("/subscribe-rules")
	{
		subscribes.GET("", h.SubscribeRule.List)
		subscribes.GET("/:id", h.SubscribeRule.Get)
		subscribes.POST("", operate, h.SubscribeRule.Create)
		subscribes.PUT("/:id", operate, h.SubscribeRule.Update)
		subscribes.DELETE("/:id", operate, h.SubscribeRule.Delete)
	}

	// Alert Channels (virtual receivers)
	if h.AlertChannel != nil {
		alertChannels := auth.Group("/alert-channels")
		{
			alertChannels.GET("", h.AlertChannel.List)
			alertChannels.GET("/:id", h.AlertChannel.Get)
			alertChannels.POST("", manage, middleware.RequirePerm("channels.write"), h.AlertChannel.Create)
			alertChannels.PUT("/:id", manage, middleware.RequirePerm("channels.write"), h.AlertChannel.Update)
			alertChannels.DELETE("/:id", manage, middleware.RequirePerm("channels.write"), h.AlertChannel.Delete)
			alertChannels.POST("/:id/test", manage, middleware.RequirePerm("channels.write"), h.AlertChannel.Test)
		}
	}

	// User personal notify configs (multi-media, current user)
	if h.UserNotifyConfig != nil {
		auth.GET("/me/notify-configs", h.UserNotifyConfig.List)
		auth.PUT("/me/notify-configs", h.UserNotifyConfig.Upsert)
		auth.DELETE("/me/notify-configs/:mediaType", h.UserNotifyConfig.DeleteByMediaType)
	}

}
