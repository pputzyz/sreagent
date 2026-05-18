package router

import (
	"github.com/gin-gonic/gin"
)

// registerAdminRoutes registers admin and management routes: data sources, users, teams,
// mute rules, inhibition rules, business groups, label registry, audit logs,
// SMTP/security settings, dashboard, AI, engine, collaboration channels, incidents,
// integrations, routing rules, post-mortems, pet, and status services.
func (h *Handlers) registerAdminRoutes(auth *gin.RouterGroup, adminOnly, manage, operate gin.HandlerFunc) {
	// DataSources
	ds := auth.Group("/datasources")
	{
		ds.GET("", h.DataSource.List)
		ds.GET("/:id", h.DataSource.Get)
		ds.POST("", adminOnly, h.DataSource.Create)
		ds.PUT("/:id", adminOnly, h.DataSource.Update)
		ds.DELETE("/:id", adminOnly, h.DataSource.Delete)
		ds.POST("/:id/health-check", manage, h.DataSource.HealthCheck)
		ds.POST("/:id/query", manage, h.DataSource.Query)
		ds.POST("/:id/query-range", manage, h.DataSource.RangeQuery)
		ds.POST("/:id/log-query", manage, h.DataSource.LogQuery)
		ds.GET("/:id/labels/keys", h.DataSource.LabelKeys)
		ds.GET("/:id/labels/values", h.DataSource.LabelValues)
		ds.GET("/:id/metrics", h.DataSource.MetricNames)
	}

	// Users — admin only for management
	users := auth.Group("/users")
	{
		users.GET("", h.User.List)
		users.GET("/:id", h.User.Get)
		users.POST("", adminOnly, h.User.Create)
		users.POST("/virtual", adminOnly, h.User.CreateVirtual)
		users.PUT("/:id", adminOnly, h.User.Update)
		users.PATCH("/:id/active", adminOnly, h.User.ToggleActive)
		users.PATCH("/:id/password", adminOnly, h.User.ChangePassword)
		users.DELETE("/:id", adminOnly, h.User.DeleteUser)
	}

	// Teams
	teams := auth.Group("/teams")
	{
		teams.GET("", h.Team.List)
		teams.GET("/:id", h.Team.Get)
		teams.GET("/:id/members", h.Team.ListMembers)
		teams.POST("", manage, h.Team.Create)
		teams.PUT("/:id", manage, h.Team.Update)
		teams.DELETE("/:id", manage, h.Team.Delete)
		teams.POST("/:id/members", manage, h.Team.AddMember)
		teams.DELETE("/:id/members/:uid", manage, h.Team.RemoveMember)
	}

	// Mute Rules
	mutes := auth.Group("/mute-rules")
	{
		mutes.GET("", h.MuteRule.List)
		mutes.GET("/preview", h.MuteRule.Preview)
		mutes.GET("/:id", h.MuteRule.Get)
		mutes.POST("", manage, h.MuteRule.Create)
		mutes.PUT("/:id", manage, h.MuteRule.Update)
		mutes.DELETE("/:id", manage, h.MuteRule.Delete)
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

	// Business Groups
	bizGroups := auth.Group("/biz-groups")
	{
		bizGroups.GET("", h.BizGroup.List)
		bizGroups.GET("/tree", h.BizGroup.ListTree)
		bizGroups.GET("/:id", h.BizGroup.Get)
		bizGroups.GET("/:id/members", h.BizGroup.ListMembers)
		bizGroups.POST("", manage, h.BizGroup.Create)
		bizGroups.PUT("/:id", manage, h.BizGroup.Update)
		bizGroups.DELETE("/:id", manage, h.BizGroup.Delete)
		bizGroups.POST("/:id/members", manage, h.BizGroup.AddMember)
		bizGroups.DELETE("/:id/members/:uid", manage, h.BizGroup.RemoveMember)
	}

	// Label Registry (autocomplete for match_labels)
	if h.LabelRegistry != nil {
		labelReg := auth.Group("/label-registry")
		{
			labelReg.GET("/keys", h.LabelRegistry.GetKeys)
			labelReg.GET("/values", h.LabelRegistry.GetValues)
			labelReg.GET("/datasource-keys", h.LabelRegistry.GetKeysByDatasource)
			labelReg.GET("/datasource-values", h.LabelRegistry.GetValuesByDatasource)
			labelReg.POST("/sync", adminOnly, h.LabelRegistry.Sync)
		}
	}

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

	// Dashboard — all authenticated users
	auth.GET("/dashboard/stats", h.Dashboard.GetStats)
	auth.GET("/dashboard/mtta-mttr", h.Dashboard.GetMTTRStats)
	auth.GET("/dashboard/mttr-trend", h.Dashboard.GetMTTRTrend)
	auth.GET("/dashboard/alert-trend", h.Dashboard.GetAlertTrend)
	auth.GET("/dashboard/top-rules", h.Dashboard.GetTopRules)
	auth.GET("/dashboard/severity-history", h.Dashboard.GetSeverityHistory)
	auth.GET("/dashboard/export", h.Dashboard.ExportReport)
	// v2 dashboard stats (incident/channel/team dimensions)
	auth.GET("/dashboard/incident-stats", h.Dashboard.IncidentStats)
	auth.GET("/dashboard/channel-stats", h.Dashboard.ChannelStats)
	auth.GET("/dashboard/team-stats", h.Dashboard.TeamStats)
	auth.GET("/dashboard/incident-trend", h.Dashboard.IncidentTrend)

	// Dashboard v2 (panel/variable dashboards)
	dashV2 := auth.Group("/dashboards")
	{
		dashV2.GET("", h.DashboardV2.List)
		dashV2.GET("/:id", h.DashboardV2.Get)
		dashV2.POST("", manage, h.DashboardV2.Create)
		dashV2.PUT("/:id", manage, h.DashboardV2.Update)
		dashV2.DELETE("/:id", manage, h.DashboardV2.Delete)
	}

	// AI — config is admin only, usage is for all
	ai := auth.Group("/ai")
	{
		ai.POST("/alert-report", h.AI.GenerateReport)
		ai.POST("/suggest-sop", h.AI.SuggestSOP)
		ai.POST("/test", manage, h.AI.TestConnection)
		ai.GET("/config", adminOnly, h.AI.GetConfig)
		ai.PUT("/config", adminOnly, h.AI.UpdateConfig)
		ai.POST("/chat", h.AI.Chat)
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

	// AI Rule Generation
	if h.AIRule != nil {
		aiRules := auth.Group("/ai/rules", operate)
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
			// Post-mortem (复盘)
			if h.PostMortem != nil {
				incidents.GET("/:id/post-mortem", h.PostMortem.Get)
				incidents.PUT("/:id/post-mortem", operate, h.PostMortem.Update)
				incidents.POST("/:id/post-mortem/publish", manage, h.PostMortem.Publish)
				incidents.POST("/:id/post-mortem/ai-generate", operate, h.PostMortem.AIGenerate)
				incidents.POST("/:id/post-mortem/ai-summary", operate, h.PostMortem.AISummary)
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
