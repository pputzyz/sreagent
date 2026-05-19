package main

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/sreagent/sreagent/internal/config"
	"github.com/sreagent/sreagent/internal/engine"
	"github.com/sreagent/sreagent/internal/handler"
	"github.com/sreagent/sreagent/internal/model"
	"github.com/sreagent/sreagent/internal/pkg/datasource"
	sredis "github.com/sreagent/sreagent/internal/pkg/redis"
	"github.com/sreagent/sreagent/internal/repository"
	"github.com/sreagent/sreagent/internal/router"
	"github.com/sreagent/sreagent/internal/service"
)

// Dependencies holds all initialized repositories, services, handlers, and
// engine components. Created by initDependencies and consumed by main() for
// startup wiring and graceful shutdown.
type Dependencies struct {
	// Repositories
	DSRepo          *repository.DataSourceRepository
	RuleRepo        *repository.AlertRuleRepository
	EventRepo       *repository.AlertEventRepository
	TimelineRepo    *repository.AlertTimelineRepository
	UserRepo        *repository.UserRepository
	ChannelRepo     *repository.NotifyChannelRepository
	TeamRepo        *repository.TeamRepository
	EscPolicyRepo   *repository.EscalationPolicyRepository
	EscStepRepo     *repository.EscalationStepRepository
	OnCallShiftRepo *repository.OnCallShiftRepository
	AlertRuleHistoryRepo *repository.AlertRuleHistoryRepository

	// Services
	SettingSvc      *service.SystemSettingService
	EventSvc        *service.AlertEventService
	NotifySvc       *service.NotificationService
	ScheduleSvc     *service.ScheduleService
	MuteRuleSvc     *service.MuteRuleService
	InhibRuleSvc    *service.InhibitionRuleService
	BizGroupSvc     *service.BizGroupService
	AlertV2Pipeline *service.AlertV2Pipeline
	LabelRegistrySvc *service.LabelRegistryService

	// Engine components
	AlertWorkerPool    *engine.AlertWorkerPool
	HeartbeatChecker   *engine.HeartbeatChecker
	AlertGroupMgr      *service.AlertGroupManager
	EscalationExecutor *engine.EscalationExecutor
	Evaluator          *engine.Evaluator // nil if engine disabled

	// Optional
	RedisClient *sredis.Client
	StateStore  engine.StateStore

	// Handlers (for router)
	Handlers *router.Handlers

	// OIDC hot reload (P1-9)
	cfg     *config.Config
	logger  *zap.Logger
	db      *gorm.DB
	oidcSvc *service.OIDCService // current OIDC service (may be nil)
	oidcMu  sync.RWMutex         // protects oidcSvc during reload

	// Shutdown
	appCtx    context.Context    // cancelled on shutdown
	appCancel context.CancelFunc // cancels background workers
}

// initDependencies creates all repositories, services, handlers, and engine
// components. This is the single DI wiring function extracted from main.go.
func initDependencies(cfg *config.Config, db *gorm.DB, zapLogger *zap.Logger) (*Dependencies, error) {
	d := &Dependencies{
		cfg:    cfg,
		logger: zapLogger,
		db:     db,
	}

	// --------------- Repositories ---------------
	dsRepo := repository.NewDataSourceRepository(db)
	ruleRepo := repository.NewAlertRuleRepository(db)
	eventRepo := repository.NewAlertEventRepository(db)
	timelineRepo := repository.NewAlertTimelineRepository(db)
	userRepo := repository.NewUserRepository(db)
	channelRepo := repository.NewNotifyChannelRepository(db)
	policyRepo := repository.NewNotifyPolicyRepository(db)
	recordRepo := repository.NewNotifyRecordRepository(db)
	scheduleRepo := repository.NewScheduleRepository(db)
	participantRepo := repository.NewScheduleParticipantRepository(db)
	overrideRepo := repository.NewScheduleOverrideRepository(db)
	onCallShiftRepo := repository.NewOnCallShiftRepository(db)
	escalationPolicyRepo := repository.NewEscalationPolicyRepository(db)
	escalationStepRepo := repository.NewEscalationStepRepository(db)
	teamRepo := repository.NewTeamRepository(db)
	muteRuleRepo := repository.NewMuteRuleRepository(db)
	inhibitionRuleRepo := repository.NewInhibitionRuleRepository(db)
	alertRuleHistoryRepo := repository.NewAlertRuleHistoryRepository(db)

	// Phase 2 repositories
	notifyRuleRepo := repository.NewNotifyRuleRepository(db)
	notifyMediaRepo := repository.NewNotifyMediaRepository(db)
	messageTemplateRepo := repository.NewMessageTemplateRepository(db)
	subscribeRuleRepo := repository.NewSubscribeRuleRepository(db)
	bizGroupRepo := repository.NewBizGroupRepository(db)

	// Label registry repository
	labelRegistryRepo := repository.NewLabelRegistryRepository(db)

	// Audit log repository
	auditLogRepo := repository.NewAuditLogRepository(db)

	// Dashboard v2 repository
	dashboardV2Repo := repository.NewDashboardRepository(db)
	templateRepo := repository.NewAlertRuleTemplateRepository(db)

	// v2 collaboration channel, incident, alert, noise-reduction, dispatch, integration & postmortem repositories
	channelV2Repo := repository.NewChannelV2Repository(db)
	incidentRepo := repository.NewIncidentRepository(db)
	alertV2Repo := repository.NewAlertRepository(db)
	exclusionRuleRepo := repository.NewExclusionRuleRepository(db)
	dispatchPolicyRepo := repository.NewDispatchPolicyRepository(db)
	dispatchLogRepo := repository.NewDispatchLogRepository(db)
	integrationRepo := repository.NewIntegrationRepository(db)
	routingRuleRepo := repository.NewRoutingRuleRepository(db)
	postMortemRepo := repository.NewPostMortemRepository(db)

	// Dispatch repositories
	alertChannelRepo := repository.NewAlertChannelRepository(db)
	userNotifyConfigRepo := repository.NewUserNotifyConfigRepository(db)
	systemSettingRepo := repository.NewSystemSettingRepository(db)

	// Pet repository
	petRepo := repository.NewPetRepository(db)

	// Status service repository
	statusServiceRepo := repository.NewStatusServiceRepository(db)

	// Chat history repository
	chatHistoryRepo := repository.NewChatHistoryRepository(db)

	// Preset rule repository
	presetRuleRepo := repository.NewPresetRuleRepository(db)

	// --------------- Services ---------------
	settingSvc := service.NewSystemSettingService(systemSettingRepo, zapLogger)
	dsSvc := service.NewDataSourceService(dsRepo, zapLogger)
	ruleSvc := service.NewAlertRuleService(ruleRepo, alertRuleHistoryRepo, dsRepo, zapLogger)
	ruleSvc.SetSystemSettingService(settingSvc)
	authSvc := service.NewAuthService(userRepo, &cfg.JWT, settingSvc, zapLogger)
	larkSvc := service.NewLarkService(zapLogger, cfg.Server.ExternalURL(), cfg.JWT.Secret, settingSvc)
	aiSvc := service.NewAIService(settingSvc, zapLogger)
	queryClient := datasource.NewQueryClient()
	contextBuilder := service.NewAlertContextBuilder(ruleRepo, dsRepo, queryClient, zapLogger)
	alertPipeline := service.NewAlertPipeline(contextBuilder, aiSvc, zapLogger)

	// Phase 2 services (created before notifySvc so they can be passed as constructor params)
	notifyMediaSvc := service.NewNotifyMediaService(notifyMediaRepo, zapLogger)
	messageTemplateSvc := service.NewMessageTemplateService(messageTemplateRepo, zapLogger)
	notifyRuleSvc := service.NewNotifyRuleService(
		notifyRuleRepo, notifyMediaRepo, messageTemplateRepo, recordRepo,
		notifyMediaSvc, messageTemplateSvc, alertPipeline, zapLogger,
	)
	subscribeRuleSvc := service.NewSubscribeRuleService(subscribeRuleRepo, zapLogger)

	notifySvc := service.NewNotificationService(channelRepo, policyRepo, recordRepo, eventRepo, larkSvc, alertPipeline, subscribeRuleSvc, notifyRuleSvc, zapLogger)
	userSvc := service.NewUserService(userRepo, zapLogger)
	teamSvc := service.NewTeamService(teamRepo, zapLogger)
	scheduleSvc := service.NewScheduleService(scheduleRepo, participantRepo, overrideRepo, onCallShiftRepo, escalationPolicyRepo, escalationStepRepo, zapLogger)
	muteRuleSvc := service.NewMuteRuleService(muteRuleRepo, zapLogger)
	inhibitionRuleSvc := service.NewInhibitionRuleService(inhibitionRuleRepo, zapLogger)
	bizGroupSvc := service.NewBizGroupService(bizGroupRepo, zapLogger)

	// Label registry service
	labelRegistrySvc := service.NewLabelRegistryService(labelRegistryRepo, dsRepo, zapLogger)

	// Audit log service
	auditLogSvc := service.NewAuditLogService(auditLogRepo, zapLogger)

	// Dashboard v2 service
	dashboardV2Svc := service.NewDashboardService(dashboardV2Repo, zapLogger)

	// Alert rule template service
	templateSvc := service.NewAlertRuleTemplateService(templateRepo, zapLogger)

	// v2 collaboration channel, incident, alert, noise-reduction & dispatch services
	channelV2Svc := service.NewChannelService(channelV2Repo, zapLogger)
	incidentSvc := service.NewIncidentService(incidentRepo, channelV2Svc, zapLogger)
	alertV2Svc := service.NewAlertV2Service(alertV2Repo, zapLogger)
	exclusionRuleSvc := service.NewExclusionRuleService(exclusionRuleRepo, zapLogger)
	noiseReducer := service.NewNoiseReducer(channelV2Repo, exclusionRuleRepo, zapLogger)
	dispatchSvc := service.NewDispatchService(dispatchPolicyRepo, dispatchLogRepo, zapLogger)
	postMortemSvc := service.NewPostMortemService(postMortemRepo, incidentRepo, zapLogger)

	// Dispatch services
	alertChannelSvc := service.NewAlertChannelService(alertChannelRepo, notifyMediaRepo, zapLogger)
	userNotifyConfigSvc := service.NewUserNotifyConfigService(userNotifyConfigRepo, zapLogger)

	// Pet service
	petSvc := service.NewPetService(petRepo, zapLogger)

	// Status service
	statusServiceSvc := service.NewStatusServiceService(statusServiceRepo, zapLogger)

	// Chat history service
	chatHistorySvc := service.NewChatHistoryService(chatHistoryRepo)

	// Preset rule service
	presetRuleSvc := service.NewPresetRuleService(presetRuleRepo, ruleRepo, dsRepo, zapLogger)

	// AI rule generation service
	ruleGenSvc := service.NewRuleGeneratorService(aiSvc, labelRegistrySvc, dsSvc, ruleSvc, presetRuleRepo, dsRepo, zapLogger)

	// Alertmanager config import service
	alertmanagerImportSvc := service.NewAlertmanagerImportService(channelV2Svc, inhibitionRuleSvc, zapLogger)

	// Seed default notification media and templates
	seedSvc := service.NewSeedService(notifyMediaRepo, messageTemplateRepo, zapLogger)
	if err := seedSvc.SeedDefaults(context.Background()); err != nil {
		zapLogger.Error("failed to seed default notification data", zap.Error(err))
	}

	// Initialize bounded worker pool for onAlert callbacks.
	// Prevents goroutine exhaustion during alert storms (e.g. 500+ firing at once).
	alertWorkerPool := engine.NewAlertWorkerPool(64)

	// AlertEventService — all dependencies resolved via constructor (no setters).
	eventSvc := service.NewAlertEventService(eventRepo, timelineRepo, notifySvc, scheduleSvc, larkSvc, alertWorkerPool, zapLogger)

	larkBotSvc := service.NewLarkBotService(settingSvc, eventSvc, scheduleSvc, userRepo, zapLogger)

	// --------------- OIDC service (optional) ---------------
	oidcSvc := d.initOIDCService(cfg, settingSvc, userRepo, zapLogger)

	// --------------- Redis (optional) ---------------
	var redisClient *sredis.Client
	var stateStore engine.StateStore
	if cfg.Redis.Host != "" {
		rc, err := sredis.New(&cfg.Redis)
		if err != nil {
			zapLogger.Warn("redis unavailable, engine will use in-memory state only",
				zap.String("addr", cfg.Redis.Addr()),
				zap.Error(err),
			)
		} else {
			redisClient = rc
			stateStore = sredis.NewRedisStateStore(rc, zapLogger)
			zapLogger.Info("redis connected, engine state persistence enabled",
				zap.String("addr", cfg.Redis.Addr()),
			)
		}
	} else {
		zapLogger.Info("redis not configured, engine will use in-memory state only")
	}

	// --------------- Engine components ---------------

	// Initialize and start the escalation executor
	escalationExecutor := engine.NewEscalationExecutor(
		escalationPolicyRepo,
		escalationStepRepo,
		eventRepo,
		timelineRepo,
		channelRepo,
		userRepo,
		notifySvc,
		userNotifyConfigRepo,
		teamRepo,
		onCallShiftRepo,
		larkSvc,
		settingSvc,
		ruleRepo,
		zapLogger,
	)
	escalationExecutor.Start()

	// Initialize and start the heartbeat checker
	heartbeatChecker := engine.NewHeartbeatChecker(ruleRepo, eventRepo, timelineRepo, zapLogger)

	// Initialize alert group manager (group_wait / group_interval)
	alertGroupMgr := service.NewAlertGroupManager(
		func(ctx context.Context, event *model.AlertEvent) error {
			return notifySvc.RouteAlert(ctx, event)
		},
		ruleRepo,
		zapLogger,
	)

	// Evaluator pointer — declared here so the onAlertFn closure can capture it.
	var evaluator *engine.Evaluator

	// Shared onAlert callback used by both the evaluator and heartbeat checker.
	// Pipeline: inhibition → mute → bizgroup → group → notify.
	onAlertFn := func(ctx context.Context, event *model.AlertEvent) {
		// 1. Check inhibition rules (suppress target alerts when source is firing).
		var firingEvents []model.AlertEvent
		if evaluator != nil {
			firingEvents = evaluator.GetFiringAlertEvents()
		} else {
			firingEvents, _, _ = eventSvc.List(ctx, "firing", "", 1, 2000)
		}
		if inhibitionRuleSvc.IsInhibited(ctx, event, firingEvents) {
			zapLogger.Info("alert inhibited by inhibition rule, skipping notification",
				zap.Uint("event_id", event.ID),
				zap.String("alert_name", event.AlertName),
			)
			return
		}

		// 2. Check mute rules.
		if muteRuleSvc.IsAlertMuted(ctx, event) {
			zapLogger.Info("alert muted, skipping notification",
				zap.Uint("event_id", event.ID),
				zap.String("alert_name", event.AlertName),
			)
			return
		}

		// 3. Annotate event with matching BizGroup scope.
		if groups, err := bizGroupSvc.FindMatchingGroups(ctx, map[string]string(event.Labels)); err == nil && len(groups) > 0 {
			g := groups[0] // most specific match
			if event.Labels == nil {
				event.Labels = make(model.JSONLabels)
			}
			event.Labels["biz_group"] = g.Name
			if g.ID != 0 {
				event.Labels["biz_group_id"] = fmt.Sprintf("%d", g.ID)
			}
			for k, v := range g.Labels {
				if _, exists := event.Labels[k]; !exists {
					event.Labels[k] = v
				}
			}
			_ = eventRepo.UpdateLabels(ctx, event.ID, event.Labels)
		}

		// 4. Route notification (through group manager for group_wait/group_interval).
		if err := alertGroupMgr.ProcessEvent(ctx, event); err != nil {
			zapLogger.Error("failed to route alert notification",
				zap.Uint("event_id", event.ID),
				zap.Error(err),
			)
		}
	}

	// App-level context for long-running background workers (cancelled on shutdown).
	appCtx, appCancel := context.WithCancel(context.Background())
	d.appCtx = appCtx
	d.appCancel = appCancel

	// Initialize v2 alert pipeline (Alert → Incident lifecycle).
	alertV2Pipeline := service.NewAlertV2Pipeline(alertV2Repo, incidentRepo, channelV2Repo, zapLogger)
	alertV2Pipeline.InitDefaultChannel(context.Background())
	alertV2Pipeline.SetNoiseReducer(noiseReducer)
	alertV2Pipeline.SetDispatchService(dispatchSvc)
	onAlertFn = alertV2Pipeline.WrapOnAlert(onAlertFn)

	// Integration service needs the pipeline (must be after pipeline setup)
	integrationSvc := service.NewIntegrationService(integrationRepo, routingRuleRepo, alertV2Pipeline, zapLogger)

	// Start the incident auto-close background worker.
	incidentSvc.StartAutoCloseWorker(appCtx)

	// Wire the heartbeat checker into the notification pipeline.
	heartbeatChecker.SetOnAlert(onAlertFn)
	heartbeatChecker.Start()

	// Initialize alert evaluator
	var engineHandler *handler.EngineHandler

	if cfg.Engine.Enabled {
		evaluator = engine.NewEvaluator(
			db, dsRepo, ruleRepo, eventRepo, timelineRepo, queryClient, zapLogger,
		)

		if stateStore != nil {
			evaluator.SetStateStore(stateStore)
		}
		evaluator.SetWorkerPool(alertWorkerPool)
		if cfg.Engine.SyncInterval > 0 {
			evaluator.SetSyncInterval(time.Duration(cfg.Engine.SyncInterval) * time.Second)
		}
		evaluator.SetOnAlert(onAlertFn)
		evaluator.Start()

		engineHandler = handler.NewEngineHandler(evaluator)
	}

	// --------------- Handlers ---------------

	// OIDC handler — uses getter function for hot-reload support (P1-9)
	var oidcHandler *handler.OIDCHandler
	if oidcSvc != nil {
		oidcHandler = handler.NewOIDCHandler(oidcSvc)
		d.oidcSvc = oidcSvc
	}

	handlers := &router.Handlers{
		Auth:             func() *handler.AuthHandler { h := handler.NewAuthHandler(authSvc); h.SetUserService(userSvc); return h }(),
		OIDC:             oidcHandler,
		OIDCSettings:     handler.NewOIDCSettingsHandler(settingSvc),
		DataSource:       handler.NewDataSourceHandler(dsSvc),
		AlertRule:        handler.NewAlertRuleHandler(ruleSvc),
		AlertEvent:       handler.NewAlertEventHandler(eventSvc),
		User:             handler.NewUserHandler(userSvc),
		Team:             handler.NewTeamHandler(teamSvc),
		Schedule:         handler.NewScheduleHandler(scheduleSvc),
		Dashboard:        handler.NewDashboardHandler(db, zapLogger),
		AI:               handler.NewAIHandler(aiSvc, eventSvc, chatHistorySvc, petSvc),
		LarkBot:          handler.NewLarkBotHandler(larkBotSvc),
		Engine:           engineHandler,
		AlertAction:      handler.NewAlertActionHandler(eventSvc, userRepo, cfg.JWT.Secret, zapLogger),
		MuteRule:         handler.NewMuteRuleHandler(muteRuleSvc),
		NotifyRule:       handler.NewNotifyRuleHandler(notifyRuleSvc),
		NotifyMedia:      handler.NewNotifyMediaHandler(notifyMediaSvc),
		MessageTemplate:  handler.NewMessageTemplateHandler(messageTemplateSvc),
		SubscribeRule:    handler.NewSubscribeRuleHandler(subscribeRuleSvc),
		BizGroup:         handler.NewBizGroupHandler(bizGroupSvc),
		AlertChannel:     handler.NewAlertChannelHandler(alertChannelSvc),
		UserNotifyConfig: handler.NewUserNotifyConfigHandler(userNotifyConfigSvc),
		AuditLog:         handler.NewAuditLogHandler(auditLogSvc),
		SMTPSettings:     handler.NewSMTPSettingsHandler(settingSvc),
		SecuritySettings: handler.NewSecuritySettingsHandler(settingSvc, &cfg.JWT),
		InhibitionRule:   handler.NewInhibitionRuleHandler(inhibitionRuleSvc),
		Heartbeat:        handler.NewHeartbeatHandler(ruleSvc),
		LabelRegistry:    handler.NewLabelRegistryHandler(labelRegistrySvc),
		DashboardV2:      handler.NewDashboardV2Handler(dashboardV2Svc),
		AlertRuleTemplate:   handler.NewAlertRuleTemplateHandler(templateSvc),
		ChannelV2:           handler.NewChannelHandler(channelV2Svc),
		IncidentV2:          handler.NewIncidentHandler(incidentSvc),
		AlertV2:             handler.NewAlertV2Handler(alertV2Svc),
		ExclusionRule:       handler.NewExclusionRuleHandler(exclusionRuleSvc),
		DispatchPolicy:      handler.NewDispatchHandler(dispatchSvc),
		Integration:         handler.NewIntegrationHandler(integrationSvc),
		RoutingRule:         handler.NewRoutingRuleHandler(routingRuleRepo),
		PostMortem:          handler.NewPostMortemHandler(postMortemSvc, aiSvc),
		Pet:                 handler.NewPetHandler(petSvc),
		StatusService:       handler.NewStatusServiceHandler(statusServiceSvc),
		PresetRule:          handler.NewPresetRuleHandler(presetRuleSvc),
		AIRule:              handler.NewAIRuleHandler(ruleGenSvc),
		AlertmanagerImport:  handler.NewAlertmanagerImportHandler(alertmanagerImportSvc),
	}

	// Inject audit service into handlers that support it
	handlers.AlertRule.SetAuditService(auditLogSvc)
	handlers.AlertEvent.SetAuditService(auditLogSvc)
	handlers.User.SetAuditService(auditLogSvc)
	// Inject event service into mute rule handler for preview endpoint
	handlers.MuteRule.SetAlertEventService(eventSvc)


	// Store references needed for shutdown and hot reload
	d.DSRepo = dsRepo
	d.RuleRepo = ruleRepo
	d.EventRepo = eventRepo
	d.TimelineRepo = timelineRepo
	d.UserRepo = userRepo
	d.ChannelRepo = channelRepo
	d.TeamRepo = teamRepo
	d.EscPolicyRepo = escalationPolicyRepo
	d.EscStepRepo = escalationStepRepo
	d.OnCallShiftRepo = onCallShiftRepo
	d.AlertRuleHistoryRepo = alertRuleHistoryRepo
	d.SettingSvc = settingSvc
	d.EventSvc = eventSvc
	d.NotifySvc = notifySvc
	d.ScheduleSvc = scheduleSvc
	d.MuteRuleSvc = muteRuleSvc
	d.InhibRuleSvc = inhibitionRuleSvc
	d.BizGroupSvc = bizGroupSvc
	d.AlertV2Pipeline = alertV2Pipeline
	d.LabelRegistrySvc = labelRegistrySvc
	d.AlertWorkerPool = alertWorkerPool
	d.HeartbeatChecker = heartbeatChecker
	d.AlertGroupMgr = alertGroupMgr
	d.EscalationExecutor = escalationExecutor
	d.Evaluator = evaluator
	d.RedisClient = redisClient
	d.StateStore = stateStore
	d.Handlers = handlers

	return d, nil
}

// initOIDCService initializes the OIDC service from config and DB settings.
// DB settings take precedence over configmap/env values.
func (d *Dependencies) initOIDCService(
	cfg *config.Config,
	settingSvc *service.SystemSettingService,
	userRepo *repository.UserRepository,
	zapLogger *zap.Logger,
) *service.OIDCService {
	oidcCfg := &cfg.OIDC // start with configmap/env values as baseline

	// Attempt to load from DB; merge if DB has a record.
	dbOIDC, err := settingSvc.GetOIDCConfig(context.Background())
	if err != nil {
		zapLogger.Warn("could not load OIDC config from DB, using configmap values", zap.Error(err))
	} else if dbOIDC.IssuerURL != "" || dbOIDC.Enabled {
		merged := config.OIDCConfig{
			Enabled:       dbOIDC.Enabled,
			IssuerURL:     firstNonEmpty(dbOIDC.IssuerURL, cfg.OIDC.IssuerURL),
			ClientID:      firstNonEmpty(dbOIDC.ClientID, cfg.OIDC.ClientID),
			ClientSecret:  firstNonEmpty(dbOIDC.ClientSecret, cfg.OIDC.ClientSecret),
			RedirectURL:   firstNonEmpty(dbOIDC.RedirectURL, cfg.OIDC.RedirectURL),
			RoleClaim:     firstNonEmpty(dbOIDC.RoleClaim, cfg.OIDC.RoleClaim),
			DefaultRole:   firstNonEmpty(dbOIDC.DefaultRole, cfg.OIDC.DefaultRole),
			AutoProvision: dbOIDC.AutoProvision,
		}
		if dbOIDC.Scopes != "" {
			merged.Scopes = splitScopes(dbOIDC.Scopes)
		} else {
			merged.Scopes = cfg.OIDC.Scopes
		}
		if dbOIDC.RoleMapping != "" {
			if rm, parseErr := parseRoleMapping(dbOIDC.RoleMapping); parseErr != nil {
				zapLogger.Warn("invalid OIDC role_mapping in DB, ignoring", zap.Error(parseErr))
				merged.RoleMapping = cfg.OIDC.RoleMapping
			} else {
				merged.RoleMapping = rm
			}
		} else {
			merged.RoleMapping = cfg.OIDC.RoleMapping
		}
		oidcCfg = &merged
		zapLogger.Info("OIDC config loaded from DB (DB values take precedence over configmap)")
	}

	if !oidcCfg.Enabled {
		return nil
	}

	svc, err := service.NewOIDCService(context.Background(), oidcCfg, &cfg.JWT, userRepo, zapLogger)
	if err != nil {
		zapLogger.Error("failed to initialize OIDC service, SSO login will be unavailable", zap.Error(err))
		return nil
	}
	zapLogger.Info("OIDC service initialized",
		zap.String("issuer", oidcCfg.IssuerURL),
		zap.String("client_id", oidcCfg.ClientID),
	)
	return svc
}

// Shutdown stops all background workers and closes connections in the correct
// order. Called from main() during graceful shutdown.
func (d *Dependencies) Shutdown() {
	zapLogger := d.logger

	// 1. Stop evaluator FIRST — no more onAlert callbacks will fire
	if d.Evaluator != nil {
		zapLogger.Info("stopping alert evaluator...")
		d.Evaluator.Stop()
	}

	// 2. Stop heartbeat checker
	d.HeartbeatChecker.Stop()

	// 3. Stop alert group manager (flush remaining buffered alerts)
	d.AlertGroupMgr.Stop()

	// 4. Stop escalation executor
	d.EscalationExecutor.Stop()

	// 5. Wait for in-flight worker pool tasks to complete
	d.AlertWorkerPool.Wait()

	// 6. Cancel app-level context (label registry sync worker, incident auto-close)
	if d.appCancel != nil {
		d.appCancel()
	}

	// 7. Close Redis connection
	if d.RedisClient != nil {
		if err := d.RedisClient.Close(); err != nil {
			zapLogger.Warn("failed to close redis connection", zap.Error(err))
		}
	}
}

// --------------- Helper functions ---------------

// firstNonEmpty returns the first non-empty string from the arguments.
func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
}

// splitScopes splits a comma-separated scopes string into a slice, trimming spaces.
func splitScopes(s string) []string {
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if t := strings.TrimSpace(p); t != "" {
			out = append(out, t)
		}
	}
	return out
}

// parseRoleMapping parses a JSON object string into a map[string]string.
func parseRoleMapping(s string) (map[string]string, error) {
	var m map[string]string
	if err := json.Unmarshal([]byte(s), &m); err != nil {
		return nil, err
	}
	return m, nil
}

