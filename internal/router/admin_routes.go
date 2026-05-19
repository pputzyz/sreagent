package router

import (
	"github.com/gin-gonic/gin"

	"github.com/sreagent/sreagent/internal/middleware"
)

// registerAdminRoutes registers admin and management routes: mute rules, inhibition rules,
// collaboration channels, integrations, routing rules, incidents, post-mortems, and pet.
// Data source, team/user, and settings routes are in their own domain files.
func (h *Handlers) registerAdminRoutes(auth *gin.RouterGroup, adminOnly, manage, operate gin.HandlerFunc) {
	// Mute Rules
	mutes := auth.Group("/mute-rules")
	{
		mutes.GET("", h.MuteRule.List)
		mutes.GET("/preview", h.MuteRule.Preview)
		mutes.GET("/:id", h.MuteRule.Get)
		mutes.GET("/:id/preview", operate, h.MuteRule.PreviewOne)
		mutes.POST("", manage, h.MuteRule.Create)
		mutes.PUT("/:id", manage, h.MuteRule.Update)
		mutes.DELETE("/:id", manage, h.MuteRule.Delete)
		mutes.POST("/batch/enable", manage, h.MuteRule.BatchEnable)
		mutes.POST("/batch/disable", manage, h.MuteRule.BatchDisable)
		mutes.POST("/batch/delete", manage, h.MuteRule.BatchDelete)
	}

	// Inhibition Rules
	if h.InhibitionRule != nil {
		inhibitions := auth.Group("/inhibition-rules")
		{
			inhibitions.GET("", h.InhibitionRule.List)
			inhibitions.GET("/:id", h.InhibitionRule.Get)
			inhibitions.POST("", manage, h.InhibitionRule.Create)
			inhibitions.PUT("/:id", manage, h.InhibitionRule.Update)
			inhibitions.DELETE("/:id", manage, h.InhibitionRule.Delete)
		}
	}

	// Collaboration Channels (协作空间 v2)
	if h.ChannelV2 != nil {
		chv2 := auth.Group("/channels")
		{
			chv2.GET("", h.ChannelV2.List)
			chv2.GET("/:id", h.ChannelV2.Get)
			chv2.POST("", manage, h.ChannelV2.Create)
			chv2.PUT("/:id", manage, h.ChannelV2.Update)
			chv2.DELETE("/:id", manage, h.ChannelV2.Delete)
			chv2.POST("/:id/star", h.ChannelV2.Star)
			chv2.DELETE("/:id/star", h.ChannelV2.Unstar)
			// Noise reduction config
			if h.ExclusionRule != nil {
				chv2.GET("/:id/exclusion-rules", h.ExclusionRule.List)
				chv2.POST("/:id/exclusion-rules", manage, h.ExclusionRule.Create)
			}
			// Dispatch policies
			if h.DispatchPolicy != nil {
				chv2.GET("/:id/dispatch-policies", h.DispatchPolicy.List)
				chv2.POST("/:id/dispatch-policies", manage, h.DispatchPolicy.Create)
			}
		}
	}

	// Exclusion rule management (update/delete by rule ID)
	if h.ExclusionRule != nil {
		excl := auth.Group("/exclusion-rules")
		{
			excl.PUT("/:id", manage, h.ExclusionRule.Update)
			excl.DELETE("/:id", manage, h.ExclusionRule.Delete)
		}
	}

	// Dispatch policy management (get/update/delete by policy ID)
	if h.DispatchPolicy != nil {
		dp := auth.Group("/dispatch-policies")
		{
			dp.GET("/:id", h.DispatchPolicy.Get)
			dp.PUT("/:id", manage, h.DispatchPolicy.Update)
			dp.DELETE("/:id", manage, h.DispatchPolicy.Delete)
		}
	}

	// Integrations CRUD (webhook receive is registered in Setup without auth)
	if h.Integration != nil {
		integrations := auth.Group("/integrations")
		{
			integrations.GET("", h.Integration.List)
			integrations.GET("/:id", h.Integration.Get)
			integrations.POST("", manage, h.Integration.Create)
			integrations.PUT("/:id", manage, h.Integration.Update)
			integrations.DELETE("/:id", manage, h.Integration.Delete)
		}
	}

	// Alertmanager config import (import receivers as channels + inhibit_rules)
	if h.AlertmanagerImport != nil {
		auth.POST("/integrations/import-alertmanager", manage, h.AlertmanagerImport.Import)
	}

	// Routing rules
	if h.RoutingRule != nil {
		rr := auth.Group("/routing-rules")
		{
			rr.GET("", h.RoutingRule.ListByIntegration) // ?integration_id=X
			rr.POST("", manage, h.RoutingRule.Create)   // body: integration_id
			rr.PUT("/:id", manage, h.RoutingRule.Update)
			rr.DELETE("/:id", manage, h.RoutingRule.Delete)
		}
	}

	// Incidents (故障 v2)
	if h.IncidentV2 != nil {
		incidents := auth.Group("/incidents")
		{
			incidents.GET("", h.IncidentV2.List)
			incidents.GET("/:id", h.IncidentV2.Get)
			incidents.POST("", manage, h.IncidentV2.Create)
			incidents.GET("/:id/timeline", h.IncidentV2.GetTimeline)
			incidents.POST("/:id/acknowledge", operate, h.IncidentV2.Acknowledge)
			incidents.POST("/:id/close", operate, h.IncidentV2.Close)
			incidents.POST("/:id/reopen", operate, h.IncidentV2.Reopen)
			incidents.POST("/:id/snooze", operate, h.IncidentV2.Snooze)
			incidents.POST("/:id/reassign", operate, h.IncidentV2.Reassign)
			incidents.POST("/:id/merge", operate, h.IncidentV2.Merge)
			incidents.POST("/:id/escalate", operate, h.IncidentV2.Escalate)
			incidents.POST("/:id/comment", operate, h.IncidentV2.AddComment)
			// Post-mortem (复盘) — AI endpoints rate limited (0.1 RPS, burst 3)
			pmRL := middleware.RateLimit(func(c *gin.Context) string {
				return "pm:" + c.ClientIP()
			}, 0.1, 3)
			if h.PostMortem != nil {
				incidents.GET("/:id/post-mortem", h.PostMortem.Get)
				incidents.PUT("/:id/post-mortem", operate, h.PostMortem.Update)
				incidents.POST("/:id/post-mortem/publish", manage, h.PostMortem.Publish)
				incidents.POST("/:id/post-mortem/ai-generate", operate, pmRL, h.PostMortem.AIGenerate)
				incidents.POST("/:id/post-mortem/ai-summary", operate, pmRL, h.PostMortem.AISummary)
			}
		}
	}

	// Post-mortems list (global view)
	if h.PostMortem != nil {
		auth.GET("/post-mortems", h.PostMortem.List)
	}

	// Pet — virtual pet system
	pet := auth.Group("/pet")
	{
		pet.GET("", h.Pet.GetPet)
		pet.PUT("", h.Pet.UpdatePet)
		pet.POST("/feed", h.Pet.FeedPet)
		pet.POST("/play", h.Pet.PlayWithPet)
		pet.GET("/interactions", h.Pet.GetInteractions)
	}
}
