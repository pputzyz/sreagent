package router

import (
	"github.com/gin-gonic/gin"

	"github.com/sreagent/sreagent/internal/middleware"
)

// registerDatasourceRoutes registers data source, label registry, and dashboard routes.
func (h *Handlers) registerDatasourceRoutes(auth *gin.RouterGroup, adminOnly, manage gin.HandlerFunc) {
	// DataSources
	ds := auth.Group("/datasources")
	{
		ds.GET("", h.DataSource.List)
		ds.GET("/:id", h.DataSource.Get)
		ds.POST("", adminOnly, middleware.RequirePerm("datasource.write"), h.DataSource.Create)
		ds.PUT("/:id", adminOnly, middleware.RequirePerm("datasource.write"), h.DataSource.Update)
		ds.DELETE("/:id", adminOnly, middleware.RequirePerm("datasource.write"), h.DataSource.Delete)
		ds.POST("/:id/health-check", manage, h.DataSource.HealthCheck)
		ds.POST("/:id/query", manage, h.DataSource.Query)
		ds.POST("/:id/query-range", manage, h.DataSource.RangeQuery)
		ds.POST("/:id/log-query", manage, h.DataSource.LogQuery)
		ds.POST("/:id/log-histogram", manage, h.DataSource.LogHistogram)
		ds.GET("/:id/labels/keys", h.DataSource.LabelKeys)
		ds.GET("/:id/labels/values", h.DataSource.LabelValues)
		ds.GET("/:id/metrics", h.DataSource.MetricNames)
		ds.GET("/:id/es-indices", manage, h.DataSource.GetESIndices)
		ds.GET("/:id/es-fields", manage, h.DataSource.GetESFields)
		// Generic proxy: GET /datasources/:id/proxy/*path (Nightingale pattern)
		// P1-7: Restricted to GET only for security
		ds.GET("/:id/proxy/*path", manage, h.DataSource.Proxy)
	}

	// Label Registry (autocomplete for match_labels)

	// Unified query endpoint (Nightingale ds-query pattern)
	auth.POST("/ds-query", manage, h.DataSource.DsQuery)
	if h.LabelRegistry != nil {
		labelReg := auth.Group("/label-registry")
		{
			labelReg.GET("/keys", h.LabelRegistry.GetKeys)
			labelReg.GET("/values", h.LabelRegistry.GetValues)
			labelReg.POST("/sync", adminOnly, h.LabelRegistry.Sync)
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
		// Dashboard biz group bindings
		dashV2.GET("/:id/biz-groups", h.DashboardV2.ListBizGroups)
		dashV2.POST("/:id/biz-groups", manage, h.DashboardV2.BindBizGroup)
		dashV2.DELETE("/:id/biz-groups/:gid", manage, h.DashboardV2.UnbindBizGroup)
	}

}
