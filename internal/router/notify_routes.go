package router

import (
	"github.com/gin-gonic/gin"
)

// registerNotifyRoutes registers all notification-related routes:
// notify rules, media, templates, subscriptions, alert channels, and user notify configs.
func (h *Handlers) registerNotifyRoutes(auth *gin.RouterGroup, manage, operate gin.HandlerFunc) {
	// Notify Rules (v2)
	notifyRules := auth.Group("/notify-rules")
	{
		notifyRules.GET("", h.NotifyRule.List)
		notifyRules.GET("/:id", h.NotifyRule.Get)
		notifyRules.POST("", manage, h.NotifyRule.Create)
		notifyRules.PUT("/:id", manage, h.NotifyRule.Update)
		notifyRules.DELETE("/:id", manage, h.NotifyRule.Delete)
		notifyRules.POST("/batch/enable", manage, h.NotifyRule.BatchEnable)
		notifyRules.POST("/batch/disable", manage, h.NotifyRule.BatchDisable)
		notifyRules.POST("/batch/delete", manage, h.NotifyRule.BatchDelete)
	}

	// Notify Media
	notifyMedia := auth.Group("/notify-media")
	{
		notifyMedia.GET("", h.NotifyMedia.List)
		notifyMedia.GET("/:id", h.NotifyMedia.Get)
		notifyMedia.POST("", manage, h.NotifyMedia.Create)
		notifyMedia.PUT("/:id", manage, h.NotifyMedia.Update)
		notifyMedia.DELETE("/:id", manage, h.NotifyMedia.Delete)
		notifyMedia.POST("/:id/test", manage, h.NotifyMedia.Test)
	}

	// Message Templates
	msgTemplates := auth.Group("/message-templates")
	{
		msgTemplates.GET("", h.MessageTemplate.List)
		msgTemplates.GET("/:id", h.MessageTemplate.Get)
		msgTemplates.POST("", manage, h.MessageTemplate.Create)
		msgTemplates.PUT("/:id", manage, h.MessageTemplate.Update)
		msgTemplates.DELETE("/:id", manage, h.MessageTemplate.Delete)
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
			alertChannels.POST("", manage, h.AlertChannel.Create)
			alertChannels.PUT("/:id", manage, h.AlertChannel.Update)
			alertChannels.DELETE("/:id", manage, h.AlertChannel.Delete)
			alertChannels.POST("/:id/test", manage, h.AlertChannel.Test)
		}
	}

	// User personal notify configs (multi-media, current user)
	if h.UserNotifyConfig != nil {
		auth.GET("/me/notify-configs", h.UserNotifyConfig.List)
		auth.PUT("/me/notify-configs", h.UserNotifyConfig.Upsert)
		auth.DELETE("/me/notify-configs/:mediaType", h.UserNotifyConfig.DeleteByMediaType)
	}

}
