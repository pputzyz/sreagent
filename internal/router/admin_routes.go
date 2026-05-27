package router

import (
	"github.com/gin-gonic/gin"

	"github.com/sreagent/sreagent/internal/middleware"
)

// registerAdminRoutes registers admin and management routes: mute rules, inhibition rules,
// collaboration channels, integrations, routing rules, incidents, and post-mortems.
// Data source, team/user, and settings routes are in their own domain files.
func (h *Handlers) registerAdminRoutes(auth *gin.RouterGroup, adminOnly, manage, operate gin.HandlerFunc) {
	// Mute Rules
	mutes := auth.Group("/mute-rules")
	{
		mutes.GET("", h.MuteRule.List)
		mutes.GET("/preview", h.MuteRule.Preview)
		mutes.GET("/:id", h.MuteRule.Get)
		mutes.GET("/:id/preview", operate, h.MuteRule.PreviewOne)
		mutes.POST("", manage, middleware.RequirePerm("mute.write"), h.MuteRule.Create)
		mutes.PUT("/:id", manage, middleware.RequirePerm("mute.write"), h.MuteRule.Update)
		mutes.DELETE("/:id", manage, middleware.RequirePerm("mute.write"), h.MuteRule.Delete)
		mutes.POST("/batch/enable", manage, middleware.RequirePerm("mute.write"), h.MuteRule.BatchEnable)
		mutes.POST("/batch/disable", manage, middleware.RequirePerm("mute.write"), h.MuteRule.BatchDisable)
		mutes.POST("/batch/delete", manage, middleware.RequirePerm("mute.write"), h.MuteRule.BatchDelete)
	}

	// Inhibition Rules
	if h.InhibitionRule != nil {
		inhibitions := auth.Group("/inhibition-rules")
		{
			inhibitions.GET("", h.InhibitionRule.List)
			inhibitions.GET("/:id", h.InhibitionRule.Get)
			inhibitions.POST("", manage, middleware.RequirePerm("inhibition.write"), h.InhibitionRule.Create)
			inhibitions.PUT("/:id", manage, middleware.RequirePerm("inhibition.write"), h.InhibitionRule.Update)
			inhibitions.DELETE("/:id", manage, middleware.RequirePerm("inhibition.write"), h.InhibitionRule.Delete)
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
				chv2.POST("/:id/dispatch-policies", manage, middleware.RequirePerm("dispatch.write"), h.DispatchPolicy.Create)
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
			dp.PUT("/:id", manage, middleware.RequirePerm("dispatch.write"), h.DispatchPolicy.Update)
			dp.DELETE("/:id", manage, middleware.RequirePerm("dispatch.write"), h.DispatchPolicy.Delete)
		}
	}

	// Integrations CRUD (webhook receive is registered in Setup without auth)
	if h.Integration != nil {
		integrations := auth.Group("/integrations")
		{
			integrations.GET("", h.Integration.List)
			integrations.GET("/:id", h.Integration.Get)
			integrations.POST("", manage, middleware.RequirePerm("integration.write"), h.Integration.Create)
			integrations.PUT("/:id", manage, middleware.RequirePerm("integration.write"), h.Integration.Update)
			integrations.DELETE("/:id", manage, middleware.RequirePerm("integration.write"), h.Integration.Delete)
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
			// Dispatch logs
			if h.DispatchPolicy != nil {
				incidents.GET("/:id/dispatch-logs", h.DispatchPolicy.ListLogs)
			}
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

	// Knowledge Base (知识库)
	if h.Knowledge != nil {
		kb := auth.Group("/knowledge")
		{
			kb.GET("", h.Knowledge.List)
			kb.GET("/:id", h.Knowledge.Get)
			kb.POST("", manage, h.Knowledge.Create)
			kb.PUT("/:id", manage, h.Knowledge.Update)
			kb.DELETE("/:id", manage, h.Knowledge.Delete)
			kb.POST("/search", h.Knowledge.Search)
			kb.POST("/:id/helpful", operate, h.Knowledge.Helpful)
		}
	}

	// Diagnostic Workflows (诊断工作流 — AIOps Phase 2)
	if h.DiagnosticWorkflow != nil {
		diag := auth.Group("/diagnostic-workflows")
		{
			diag.GET("", h.DiagnosticWorkflow.List)
			diag.GET("/:id", h.DiagnosticWorkflow.Get)
			diag.POST("", manage, h.DiagnosticWorkflow.Create)
			diag.PUT("/:id", manage, h.DiagnosticWorkflow.Update)
			diag.DELETE("/:id", manage, h.DiagnosticWorkflow.Delete)
			diag.PUT("/:id/steps", manage, h.DiagnosticWorkflow.ReplaceSteps)
			diag.POST("/:id/run", operate, h.DiagnosticWorkflow.StartRun)
			diag.POST("/match", operate, h.DiagnosticWorkflow.MatchWorkflows)
		}
		// Diagnostic Runs
		diagRuns := auth.Group("/diagnostic-runs")
		{
			diagRuns.GET("", h.DiagnosticWorkflow.ListRuns)
			diagRuns.GET("/:id", h.DiagnosticWorkflow.GetRun)
		}
	}

	// Change Events (变更事件 — AIOps Phase 2)
	if h.ChangeEvent != nil {
		changes := auth.Group("/change-events")
		{
			changes.GET("", h.ChangeEvent.List)
			changes.GET("/:id", h.ChangeEvent.Get)
			changes.POST("", manage, h.ChangeEvent.Ingest)
			changes.DELETE("/:id", manage, h.ChangeEvent.Delete)
		}
	}

	// Inspection Tasks (定时巡检 Agent)
	if h.Inspection != nil {
		inspTasks := auth.Group("/inspection/tasks")
		{
			inspTasks.GET("", h.Inspection.ListTasks)
			inspTasks.GET("/:id", h.Inspection.GetTask)
			inspTasks.POST("", manage, h.Inspection.CreateTask)
			inspTasks.PUT("/:id", manage, h.Inspection.UpdateTask)
			inspTasks.DELETE("/:id", manage, h.Inspection.DeleteTask)
			inspTasks.POST("/:id/run", operate, h.Inspection.RunNow)
		}
		inspRuns := auth.Group("/inspection/runs")
		{
			inspRuns.GET("", h.Inspection.ListRuns)
			inspRuns.GET("/:id", h.Inspection.GetRun)
		}
		inspUtil := auth.Group("/inspection")
		{
			inspUtil.POST("/validate-cron", h.Inspection.ValidateCron)
		}
	}
}
