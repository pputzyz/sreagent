package router

import (
	"net/http"
	"os"
	"path"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/sreagent/sreagent/internal/config"
	"github.com/sreagent/sreagent/internal/handler"
	"github.com/sreagent/sreagent/internal/middleware"
)

// startedAt records the server startup timestamp (RFC3339).
// Exposed via /healthz so the frontend can detect server restarts.
var startedAt = time.Now().UTC().Format(time.RFC3339)

// Handlers aggregates all handler instances.
type Handlers struct {
	Auth               *handler.AuthHandler
	OIDC               *handler.OIDCHandler // nil if OIDC is not configured
	OIDCSettings       *handler.OIDCSettingsHandler
	OAuth2             *handler.OAuth2Handler      // nil if OAuth2 is not configured
	SSOSettings        *handler.SSOSettingsHandler // LDAP + OAuth2 settings
	DataSource         *handler.DataSourceHandler
	AlertRule          *handler.AlertRuleHandler
	AlertEvent         *handler.AlertEventHandler
	User               *handler.UserHandler
	Team               *handler.TeamHandler
	Schedule           *handler.ScheduleHandler
	Dashboard          *handler.DashboardHandler
	AI                 *handler.AIHandler
	LarkBot            *handler.LarkBotHandler
	Engine             *handler.EngineHandler
	AlertAction        *handler.AlertActionHandler
	MuteRule           *handler.MuteRuleHandler
	NotifyRule         *handler.NotifyRuleHandler
	NotifyMedia        *handler.NotifyMediaHandler
	MessageTemplate    *handler.MessageTemplateHandler
	SubscribeRule      *handler.SubscribeRuleHandler
	BizGroup           *handler.BizGroupHandler
	AlertChannel       *handler.AlertChannelHandler
	UserNotifyConfig   *handler.UserNotifyConfigHandler
	AuditLog           *handler.AuditLogHandler
	SMTPSettings       *handler.SMTPSettingsHandler
	SecuritySettings   *handler.SecuritySettingsHandler
	InhibitionRule     *handler.InhibitionRuleHandler
	Heartbeat          *handler.HeartbeatHandler
	LabelRegistry      *handler.LabelRegistryHandler
	DashboardV2        *handler.DashboardV2Handler
	AlertRuleTemplate  *handler.AlertRuleTemplateHandler
	ChannelV2          *handler.ChannelHandler            // v2 collaboration channels (协作空间)
	IncidentV2         *handler.IncidentHandler           // v2 incidents (故障)
	AlertV2            *handler.AlertV2Handler            // v2 alerts (告警)
	ExclusionRule      *handler.ExclusionRuleHandler      // channel exclusion rules (排除规则)
	DispatchPolicy     *handler.DispatchHandler           // channel dispatch policies (分派策略)
	Integration        *handler.IntegrationHandler        // webhook integrations (集成中心)
	RoutingRule        *handler.RoutingRuleHandler        // routing rules for shared integrations (路由规则)
	PostMortem         *handler.PostMortemHandler         // incident post-mortems (故障复盘)
	StatusService      *handler.StatusServiceHandler      // status page services (状态页面)
	PresetRule         *handler.PresetRuleHandler         // preset rules (预设规则)
	AIRule             *handler.AIRuleHandler             // AI rule generation (AI 规则生成)
	AlertmanagerImport *handler.AlertmanagerImportHandler // alertmanager config import
	UserPreference     *handler.UserPreferenceHandler     // user preferences (用户偏好)
	UserNotification   *handler.UserNotificationHandler   // notification center (通知中心)
	Permissions        *handler.PermissionsHandler        // RBAC permissions (权限查询)
	Agent              *handler.AgentHandler              // AI Agent (自主执行)
	Knowledge          *handler.KnowledgeHandler          // 知识库 (Knowledge Base)
	DiagnosticWorkflow *handler.DiagnosticWorkflowHandler // 诊断工作流 (AIOps Phase 2)
	ChangeEvent        *handler.ChangeEventHandler        // 变更事件 (AIOps Phase 2)
	Inspection         *handler.InspectionHandler         // 定时巡检 Agent
	ReportTask         *handler.ReportTaskHandler         // 定时报告任务
	RecordingRule      *handler.RecordingRuleHandler      // 录制规则 (Recording Rules)
	BuiltinMetric      *handler.BuiltinMetricHandler      // 内置指标目录 (Metrics Builtin)
	EventPipeline      *handler.EventPipelineHandler      // 事件管道 (Event Pipeline)
	Annotation         *handler.AnnotationHandler         // 仪表盘标注 (Annotations)
	SavedView          *handler.SavedViewHandler          // 快捷视图 (Saved Views)
	MetricView         *handler.MetricViewHandler         // 指标视图 (Metric Views)
	LLMConfig          *handler.LLMConfigHandler          // LLM 配置管理 (LLM Configs)
	MCPServer          *handler.MCPServerHandler          // MCP 服务器管理 (MCP Servers)
	AISkill            *handler.AISkillHandler            // AI 技能管理 (AI Skills)
	ESIndexPattern     *handler.ESIndexPatternHandler     // ES 索引模式 (ES Index Patterns)
	SiteInfo           *handler.SiteInfoHandler           // 站点信息 (Site Info)
	TaskTpl            *handler.TaskTplHandler            // 任务模板管理 (Task Templates)
	Task               *handler.TaskHandler               // 任务执行 (Task Execution)
	BuiltinDashboard   *handler.BuiltinDashboardHandler   // 内置仪表盘库 (Builtin Dashboards)
	UserContact        *handler.UserContactHandler        // 用户联系人 (User Contacts)
	StatusSubscription *handler.StatusSubscriptionHandler // 状态页邮件订阅 (Status Page Subscriptions)
	TeamNotifyChannel  *handler.TeamNotifyChannelHandler  // 团队通知渠道 (Team Notify Channels)
	UserTeamNotifyPref *handler.UserTeamNotifyPrefHandler // 用户团队通知偏好 (User Team Notify Prefs)
}

// Setup initializes the Gin router with all routes and middleware.
func Setup(cfg *config.Config, handlers *Handlers, logger *zap.Logger) *gin.Engine {
	if cfg.Server.Mode == "release" {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.New()

	// Global middleware — use zap-based recovery instead of gin's default stderr logger
	r.Use(func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				// Try to get request-scoped logger, fall back to the package-level one
				var zapLogger *zap.Logger
				if l, exists := c.Get("logger"); exists {
					if l2, ok := l.(*zap.Logger); ok {
						zapLogger = l2
					}
				}
				if zapLogger == nil {
					zapLogger = logger
				}
				zapLogger.Error("panic recovered",
					zap.Any("error", err),
					zap.String("path", c.Request.URL.Path),
					zap.String("method", c.Request.Method),
				)
				c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
					"code":    50000,
					"message": "internal server error",
				})
			}
		}()
		c.Next()
	})
	r.Use(middleware.CORS(cfg.CORSAllowedOrigins))
	r.Use(middleware.RequestLogger(logger))

	// Limit request body size to 10MB
	r.Use(func(c *gin.Context) {
		if c.Request.Body != nil {
			c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 10<<20)
		}
		c.Next()
	})

	// Health check (no auth)
	// NOTE (B1-18): Add rate limiting to this endpoint to prevent abuse.
	// Currently unprotected — a flood of /healthz requests could consume
	// server resources. Consider adding middleware.RateLimit or a dedicated
	// health-check limiter (e.g. 100 req/s per IP).
	r.GET("/healthz", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok", "started_at": startedAt})
	})

	// Prometheus metrics endpoint (no auth) — exposes Go runtime + app metrics
	r.GET("/metrics", handler.NewMetricsHandler(cfg.MetricsToken))

	// Webhook endpoint — authenticated by shared secret (X-Webhook-Secret header)
	webhooks := r.Group("/webhooks", middleware.WebhookAuth(cfg.Server.WebhookSecret))
	{
		webhooks.POST("/alertmanager", handlers.AlertEvent.WebhookReceive)
	}

	// Lark Bot callback (no auth middleware — verified by HMAC signature or token)
	r.POST("/lark/event", handlers.LarkBot.EventCallback)

	// Lark card action callback (button clicks on alert cards)
	r.POST("/lark/card-action", handlers.LarkBot.CardActionCallback)

	// Integration webhook receive endpoint (no auth — authenticated by token in URL)
	if handlers.Integration != nil {
		r.POST("/api/v1/integrations/:token/alerts", handlers.Integration.Receive)
	}

	// API v1 routes
	api := r.Group("/api/v1")
	{
		// Public routes — login rate limited (5 RPS, burst 5, lockout after 5 failures for 15 min)
		// In testing mode, disable rate limiting entirely
		loginRL := middleware.LoginRateLimit(5, 5, 5, 15*time.Minute)
		if os.Getenv("SREAGENT_TESTING") == "true" {
			loginRL = func(c *gin.Context) { c.Next() } // no-op in testing mode
		}
		api.POST("/auth/login", loginRL, handlers.Auth.Login)
		refreshRL := middleware.RateLimit(func(c *gin.Context) string { return "refresh:" + c.ClientIP() }, 1, 5)
		api.POST("/auth/refresh", refreshRL, handlers.Auth.Refresh)
		api.GET("/auth/captcha", handlers.Auth.Captcha)

		// OIDC routes (public — before JWT middleware)
		if handlers.OIDC != nil {
			// #7: Rate limit OIDC callback and token endpoints
			oidcCallbackRL := middleware.RateLimit(func(c *gin.Context) string {
				return "oidc-cb:" + c.ClientIP()
			}, 10.0/60.0, 10)
			api.GET("/auth/oidc/config", handlers.OIDC.OIDCConfig)
			api.GET("/auth/oidc/login", handlers.OIDC.LoginRedirect)
			api.GET("/auth/oidc/callback", oidcCallbackRL, handlers.OIDC.Callback)
			api.POST("/auth/oidc/token", oidcCallbackRL, handlers.OIDC.CallbackJSON)
		} else {
			// Return disabled status when OIDC is not configured
			api.GET("/auth/oidc/config", func(c *gin.Context) {
				c.JSON(200, gin.H{"code": 0, "message": "ok", "data": gin.H{"enabled": false}})
			})
		}

		// OAuth2 routes (public — before JWT middleware)
		if handlers.OAuth2 != nil {
			// #7: Rate limit OAuth2 callback and token endpoints
			oauth2CallbackRL := middleware.RateLimit(func(c *gin.Context) string {
				return "oauth2-cb:" + c.ClientIP()
			}, 10.0/60.0, 10)
			api.GET("/auth/oauth2/config", handlers.OAuth2.OAuth2Config)
			api.GET("/auth/oauth2/login", handlers.OAuth2.LoginRedirect)
			api.GET("/auth/oauth2/callback", oauth2CallbackRL, handlers.OAuth2.Callback)
			api.POST("/auth/oauth2/token", oauth2CallbackRL, handlers.OAuth2.CallbackJSON)
		} else {
			// Return disabled status when OAuth2 is not configured
			api.GET("/auth/oauth2/config", func(c *gin.Context) {
				c.JSON(200, gin.H{"code": 0, "message": "ok", "data": gin.H{"enabled": false}})
			})
		}

		// Status page public endpoints (no auth — read-only for external status pages)
		if handlers.StatusService != nil {
			api.GET("/status-services", handlers.StatusService.List)
			api.GET("/status-services/:id", handlers.StatusService.Get)
		}

		// Status page subscription public endpoints (no auth — email subscribe/unsubscribe)
		if handlers.StatusSubscription != nil {
			// #13: Rate limit subscribe/unsubscribe to prevent abuse
			statusSubRL := middleware.RateLimit(func(c *gin.Context) string {
				return "status-sub:" + c.ClientIP()
			}, 10.0/60.0, 10)
			api.POST("/status-subscriptions", statusSubRL, handlers.StatusSubscription.Subscribe)
			api.DELETE("/status-subscriptions", statusSubRL, handlers.StatusSubscription.Unsubscribe)
		}

		// ----- Authenticated routes (JWT required) -----
		auth := api.Group("")
		auth.Use(middleware.JWTAuth(&cfg.JWT))
		auth.Use(middleware.TeamScoped())
		{
			// --- Role shorthand for readability ---
			adminOnly := middleware.RequireRole("admin")
			manage := middleware.RequireRole("admin", "team_lead")            // create/update/delete config objects
			operate := middleware.RequireRole("admin", "team_lead", "member") // operational actions (ack, resolve, etc.)

			// Register routes by module
			handlers.registerAuthRoutes(auth, adminOnly)
			handlers.registerAlertRoutes(r, auth, adminOnly, manage, operate)
			handlers.registerNotifyRoutes(auth, manage, operate)
			handlers.registerScheduleRoutes(auth, manage)
			handlers.registerDatasourceRoutes(auth, adminOnly, manage)
			handlers.registerTeamRoutes(auth, adminOnly, manage)
			handlers.registerSettingRoutes(auth, adminOnly, manage, operate)
			handlers.registerAdminRoutes(auth, adminOnly, manage, operate)
			handlers.registerRecordingRuleRoutes(auth, adminOnly, manage)
			handlers.registerBuiltinMetricRoutes(auth, adminOnly, manage)
			handlers.registerEventPipelineRoutes(auth, adminOnly, manage)
			handlers.registerAnnotationRoutes(auth, manage)
			handlers.registerSavedViewRoutes(auth, manage)
			handlers.registerMetricViewRoutes(auth, manage)
			handlers.registerLLMConfigRoutes(auth, manage)
			handlers.registerMCPServerRoutes(auth, manage)
			handlers.registerAISkillRoutes(auth, manage)
			handlers.registerESIndexPatternRoutes(auth, manage)
			handlers.registerBuiltinDashboardRoutes(auth, adminOnly, manage)
			handlers.registerTaskRoutes(auth, manage, operate)
		}
	}

	// Serve frontend static files in production
	distPath := "web/dist"
	if _, err := os.Stat(distPath); err == nil {
		r.Static("/assets", path.Join(distPath, "assets"))
		r.StaticFile("/favicon.ico", path.Join(distPath, "favicon.ico"))
		r.StaticFile("/logo.svg", path.Join(distPath, "logo.svg"))

		r.NoRoute(func(c *gin.Context) {
			reqPath := c.Request.URL.Path

			// Never serve index.html for backend API paths — return proper 404 JSON.
			if strings.HasPrefix(reqPath, "/api/") ||
				strings.HasPrefix(reqPath, "/webhooks/") ||
				strings.HasPrefix(reqPath, "/lark/") ||
				strings.HasPrefix(reqPath, "/heartbeat/") ||
				strings.HasPrefix(reqPath, "/alert-action/") ||
				reqPath == "/healthz" ||
				reqPath == "/metrics" {
				c.JSON(http.StatusNotFound, gin.H{
					"code":    40400,
					"message": "endpoint not found",
				})
				return
			}

			// If it looks like a static file request, try to serve it
			if strings.Contains(reqPath, ".") {
				filePath := path.Join(distPath, reqPath)
				// #5: Prevent path traversal — resolved path must stay under distPath
				if !strings.HasPrefix(filePath, distPath) {
					c.Status(http.StatusNotFound)
					return
				}
				if _, err := os.Stat(filePath); err == nil {
					c.File(filePath)
					return
				}
				c.Status(http.StatusNotFound)
				return
			}
			// SPA fallback: serve index.html for all non-API routes
			c.File(path.Join(distPath, "index.html"))
		})
	}

	return r
}
