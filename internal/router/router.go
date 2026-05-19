package router

import (
	"net/http"
	"os"
	"path"
	"strings"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/sreagent/sreagent/internal/config"
	"github.com/sreagent/sreagent/internal/handler"
	"github.com/sreagent/sreagent/internal/middleware"
)

// Handlers aggregates all handler instances.
type Handlers struct {
	Auth             *handler.AuthHandler
	OIDC             *handler.OIDCHandler // nil if OIDC is not configured
	OIDCSettings     *handler.OIDCSettingsHandler
	DataSource       *handler.DataSourceHandler
	AlertRule        *handler.AlertRuleHandler
	AlertEvent       *handler.AlertEventHandler
	Notification     *handler.NotificationHandler
	User             *handler.UserHandler
	Team             *handler.TeamHandler
	Schedule         *handler.ScheduleHandler
	Dashboard        *handler.DashboardHandler
	AI               *handler.AIHandler
	LarkBot          *handler.LarkBotHandler
	Engine           *handler.EngineHandler
	AlertAction      *handler.AlertActionHandler
	MuteRule         *handler.MuteRuleHandler
	NotifyRule       *handler.NotifyRuleHandler
	NotifyMedia      *handler.NotifyMediaHandler
	MessageTemplate  *handler.MessageTemplateHandler
	SubscribeRule    *handler.SubscribeRuleHandler
	BizGroup         *handler.BizGroupHandler
	AlertChannel     *handler.AlertChannelHandler
	UserNotifyConfig *handler.UserNotifyConfigHandler
	AuditLog         *handler.AuditLogHandler
	SMTPSettings     *handler.SMTPSettingsHandler
	SecuritySettings *handler.SecuritySettingsHandler
	InhibitionRule   *handler.InhibitionRuleHandler
	Heartbeat        *handler.HeartbeatHandler
	LabelRegistry    *handler.LabelRegistryHandler
	DashboardV2         *handler.DashboardV2Handler
	AlertRuleTemplate   *handler.AlertRuleTemplateHandler
	ChannelV2      *handler.ChannelHandler        // v2 collaboration channels (协作空间)
	IncidentV2     *handler.IncidentHandler       // v2 incidents (故障)
	AlertV2        *handler.AlertV2Handler        // v2 alerts (告警)
	ExclusionRule  *handler.ExclusionRuleHandler  // channel exclusion rules (排除规则)
	DispatchPolicy *handler.DispatchHandler       // channel dispatch policies (分派策略)
	Integration    *handler.IntegrationHandler    // webhook integrations (集成中心)
	RoutingRule    *handler.RoutingRuleHandler    // routing rules for shared integrations (路由规则)
	PostMortem     *handler.PostMortemHandler     // incident post-mortems (故障复盘)
	Pet            *handler.PetHandler            // virtual pet system (宠物系统)
	StatusService  *handler.StatusServiceHandler  // status page services (状态页面)
	PresetRule          *handler.PresetRuleHandler          // preset rules (预设规则)
	AIRule              *handler.AIRuleHandler              // AI rule generation (AI 规则生成)
	AlertmanagerImport  *handler.AlertmanagerImportHandler  // alertmanager config import
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
	r.GET("/healthz", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	// Prometheus metrics endpoint (no auth) — exposes Go runtime + app metrics
	r.GET("/metrics", handler.NewMetricsHandler(cfg.MetricsToken))

	// Webhook endpoint — authenticated by shared secret (X-Webhook-Secret header)
	webhooks := r.Group("/webhooks", middleware.WebhookAuth(cfg.Server.WebhookSecret))
	{
		webhooks.POST("/alertmanager", handlers.AlertEvent.WebhookReceive)
	}

	// Lark Bot callback (no auth - verified by token)
	r.POST("/lark/event", handlers.LarkBot.EventCallback)

	// Integration webhook receive endpoint (no auth — authenticated by token in URL)
	if handlers.Integration != nil {
		r.POST("/api/v1/integrations/:token/alerts", handlers.Integration.Receive)
	}

	// API v1 routes
	api := r.Group("/api/v1")
	{
		// Public routes
		api.POST("/auth/login", handlers.Auth.Login)
		api.POST("/auth/refresh", handlers.Auth.Refresh)

		// OIDC routes (public — before JWT middleware)
		if handlers.OIDC != nil {
			api.GET("/auth/oidc/config", handlers.OIDC.OIDCConfig)
			api.GET("/auth/oidc/login", handlers.OIDC.LoginRedirect)
			api.GET("/auth/oidc/callback", handlers.OIDC.Callback)
			api.POST("/auth/oidc/token", handlers.OIDC.CallbackJSON)
		} else {
			// Return disabled status when OIDC is not configured
			api.GET("/auth/oidc/config", func(c *gin.Context) {
				c.JSON(200, gin.H{"code": 0, "message": "ok", "data": gin.H{"enabled": false}})
			})
		}

		// ----- Authenticated routes (JWT required) -----
		auth := api.Group("")
		auth.Use(middleware.JWTAuth(&cfg.JWT))
		{
			// --- Role shorthand for readability ---
			adminOnly := middleware.RequireRole("admin")
			manage := middleware.RequireRole("admin", "team_lead")            // create/update/delete config objects
			operate := middleware.RequireRole("admin", "team_lead", "member") // operational actions (ack, resolve, etc.)

			// Register routes by module
			handlers.registerAuthRoutes(auth, adminOnly)
			handlers.registerAlertRoutes(r, auth, manage, operate)
			handlers.registerNotifyRoutes(auth, manage, operate)
			handlers.registerScheduleRoutes(auth, manage)
			handlers.registerAdminRoutes(auth, adminOnly, manage, operate)
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
			// If it looks like a static file request, try to serve it
			if strings.Contains(reqPath, ".") {
				filePath := path.Join(distPath, reqPath)
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
