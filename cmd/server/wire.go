package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/sreagent/sreagent/internal/config"
	"github.com/sreagent/sreagent/internal/engine"
	ppipeline "github.com/sreagent/sreagent/internal/engine/pipeline"
	pprocessors "github.com/sreagent/sreagent/internal/engine/pipeline/processors"
	"github.com/sreagent/sreagent/internal/handler"
	"github.com/sreagent/sreagent/internal/middleware"
	"github.com/sreagent/sreagent/internal/model"
	"github.com/sreagent/sreagent/internal/pkg/datasource"
	"github.com/sreagent/sreagent/internal/pkg/hashring"
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
	RecordingRuleEngine *engine.RecordingRuleEngine

	// Optional
	RedisClient *sredis.Client
	StateStore  engine.StateStore
	Leader      engine.LeaderElection // nil if no Redis or single-instance

	// Handlers (for router)
	Handlers *router.Handlers

	// OIDC hot reload
	cfg        *config.Config
	logger     *zap.Logger
	db         *gorm.DB
	oidcSvc    *service.OIDCService    // current OIDC service (may be nil)
	oidcMu     sync.RWMutex           // protects oidcSvc during reload
	oidcHdlr   *handler.OIDCHandler   // for SetService on reload

	// Inspection scheduler (for graceful shutdown)
	InspectionSched *service.InspectionScheduler

	// Shutdown
	appCtx    context.Context    // cancelled on shutdown
	appCancel context.CancelFunc // cancels background workers
}

// teamRoleAdapter implements middleware.TeamRoleQuerier using the team repository.
type teamRoleAdapter struct {
	teamRepo *repository.TeamRepository
}

func (a *teamRoleAdapter) ListTeamRoles(userID uint) ([]string, error) {
	members, err := a.teamRepo.ListByUser(context.Background(), userID)
	if err != nil {
		return nil, err
	}
	seen := make(map[string]struct{})
	var roles []string
	for _, m := range members {
		if _, ok := seen[m.Role]; !ok {
			seen[m.Role] = struct{}{}
			roles = append(roles, m.Role)
		}
	}
	return roles, nil
}

// tokenRevocationAdapter implements middleware.TokenRevocationChecker using Redis.
type tokenRevocationAdapter struct {
	rc *sredis.Client
}

func (a *tokenRevocationAdapter) GetUserTokenRevokedAt(userID uint) time.Time {
	ts, err := a.rc.GetUserTokenRevokedAt(context.Background(), userID)
	if err != nil {
		return time.Time{}
	}
	return ts
}

// repoBundle holds all repository instances created from a single *gorm.DB.
type repoBundle struct {
	DS                *repository.DataSourceRepository
	Rule              *repository.AlertRuleRepository
	Event             *repository.AlertEventRepository
	Timeline          *repository.AlertTimelineRepository
	User              *repository.UserRepository
	Channel           *repository.NotifyChannelRepository
	Record            *repository.NotifyRecordRepository
	Schedule          *repository.ScheduleRepository
	Participant       *repository.ScheduleParticipantRepository
	Override          *repository.ScheduleOverrideRepository
	OnCallShift       *repository.OnCallShiftRepository
	EscalationPolicy  *repository.EscalationPolicyRepository
	EscalationStep    *repository.EscalationStepRepository
	StepExec          *repository.EscalationStepExecutionRepository
	Team              *repository.TeamRepository
	MuteRule          *repository.MuteRuleRepository
	InhibitionRule    *repository.InhibitionRuleRepository
	AlertRuleHistory  *repository.AlertRuleHistoryRepository
	NotifyRule        *repository.NotifyRuleRepository
	NotifyMedia       *repository.NotifyMediaRepository
	MessageTemplate   *repository.MessageTemplateRepository
	SubscribeRule     *repository.SubscribeRuleRepository
	BizGroup          *repository.BizGroupRepository
	LabelRegistry     *repository.LabelRegistryRepository
	AuditLog          *repository.AuditLogRepository
	Knowledge         *repository.KnowledgeRepository
	DashboardV2       *repository.DashboardRepository
	DashboardBizGroup *repository.DashboardBizGroupRepository
	AlertRuleTemplate *repository.AlertRuleTemplateRepository
	ChannelV2         *repository.ChannelRepository
	Incident          *repository.IncidentRepository
	AlertV2           *repository.AlertRepository
	ExclusionRule     *repository.ExclusionRuleRepository
	DispatchPolicy    *repository.DispatchPolicyRepository
	DispatchLog       *repository.DispatchLogRepository
	Integration       *repository.IntegrationRepository
	RoutingRule       *repository.RoutingRuleRepository
	PostMortem        *repository.PostMortemRepository
	AlertChannel      *repository.AlertChannelRepository
	UserNotifyConfig  *repository.UserNotifyConfigRepository
	SystemSetting     *repository.SystemSettingRepository
	ScheduledDispatch *repository.ScheduledDispatchRepository
	StatusService     *repository.StatusServiceRepository
	ChatHistory       *repository.ChatHistoryRepository
	AIConversation    *repository.AIConversationRepository
	PresetRule        *repository.PresetRuleRepository
	UserPreference    *repository.UserPreferenceRepository
	UserContact       *repository.UserContactRepository
	UserNotification  *repository.UserNotificationRepository
	DiagnosticWorkflow *repository.DiagnosticWorkflowRepository
	ChangeEvent       *repository.ChangeEventRepository
	Inspection        *repository.InspectionRepository
	RecordingRule     *repository.RecordingRuleRepository
	BuiltinMetric     *repository.BuiltinMetricRepository
	MetricFilter      *repository.MetricFilterRepository
	EventPipeline     *repository.EventPipelineRepository
	EventPipelineExec *repository.EventPipelineExecutionRepository
	Annotation        *repository.AnnotationRepository
	LLMConfig         *repository.LLMConfigRepository
	BuiltinDashboard  *repository.BuiltinDashboardRepository
	TaskTpl           *repository.TaskTplRepository
	TaskRecord        *repository.TaskRecordRepository
	TeamNotifyChannel *repository.TeamNotifyChannelRepository
	UserTeamNotifyPref *repository.UserTeamNotifyPrefRepository
	StatusSubscription *repository.StatusSubscriptionRepository
	SavedView         *repository.SavedViewRepository
	MetricView        *repository.MetricViewRepository
	MCPServer         *repository.MCPServerRepository
	AISkill           *repository.AISkillRepository
	ESIndexPattern    *repository.ESIndexPatternRepository
}

// initRepositories creates all repository instances from a single database connection.
func initRepositories(db *gorm.DB) *repoBundle {
	return &repoBundle{
		DS:                repository.NewDataSourceRepository(db),
		Rule:              repository.NewAlertRuleRepository(db),
		Event:             repository.NewAlertEventRepository(db),
		Timeline:          repository.NewAlertTimelineRepository(db),
		User:              repository.NewUserRepository(db),
		Channel:           repository.NewNotifyChannelRepository(db),
		Record:            repository.NewNotifyRecordRepository(db),
		Schedule:          repository.NewScheduleRepository(db),
		Participant:       repository.NewScheduleParticipantRepository(db),
		Override:          repository.NewScheduleOverrideRepository(db),
		OnCallShift:       repository.NewOnCallShiftRepository(db),
		EscalationPolicy:  repository.NewEscalationPolicyRepository(db),
		EscalationStep:    repository.NewEscalationStepRepository(db),
		StepExec:          repository.NewEscalationStepExecutionRepository(db),
		Team:              repository.NewTeamRepository(db),
		MuteRule:          repository.NewMuteRuleRepository(db),
		InhibitionRule:    repository.NewInhibitionRuleRepository(db),
		AlertRuleHistory:  repository.NewAlertRuleHistoryRepository(db),
		NotifyRule:        repository.NewNotifyRuleRepository(db),
		NotifyMedia:       repository.NewNotifyMediaRepository(db),
		MessageTemplate:   repository.NewMessageTemplateRepository(db),
		SubscribeRule:     repository.NewSubscribeRuleRepository(db),
		BizGroup:          repository.NewBizGroupRepository(db),
		LabelRegistry:     repository.NewLabelRegistryRepository(db),
		AuditLog:          repository.NewAuditLogRepository(db),
		Knowledge:         repository.NewKnowledgeRepository(db),
		DashboardV2:       repository.NewDashboardRepository(db),
		DashboardBizGroup: repository.NewDashboardBizGroupRepository(db),
		AlertRuleTemplate: repository.NewAlertRuleTemplateRepository(db),
		ChannelV2:         repository.NewChannelV2Repository(db),
		Incident:          repository.NewIncidentRepository(db),
		AlertV2:           repository.NewAlertRepository(db),
		ExclusionRule:     repository.NewExclusionRuleRepository(db),
		DispatchPolicy:    repository.NewDispatchPolicyRepository(db),
		DispatchLog:       repository.NewDispatchLogRepository(db),
		Integration:       repository.NewIntegrationRepository(db),
		RoutingRule:       repository.NewRoutingRuleRepository(db),
		PostMortem:        repository.NewPostMortemRepository(db),
		AlertChannel:      repository.NewAlertChannelRepository(db),
		UserNotifyConfig:  repository.NewUserNotifyConfigRepository(db),
		SystemSetting:     repository.NewSystemSettingRepository(db),
		ScheduledDispatch: repository.NewScheduledDispatchRepository(db),
		StatusService:     repository.NewStatusServiceRepository(db),
		ChatHistory:       repository.NewChatHistoryRepository(db),
		AIConversation:    repository.NewAIConversationRepository(db),
		PresetRule:        repository.NewPresetRuleRepository(db),
		UserPreference:    repository.NewUserPreferenceRepository(db),
		UserContact:       repository.NewUserContactRepository(db),
		UserNotification:  repository.NewUserNotificationRepository(db),
		DiagnosticWorkflow: repository.NewDiagnosticWorkflowRepository(db),
		ChangeEvent:       repository.NewChangeEventRepository(db),
		Inspection:        repository.NewInspectionRepository(db),
		RecordingRule:     repository.NewRecordingRuleRepository(db),
		BuiltinMetric:     repository.NewBuiltinMetricRepository(db),
		MetricFilter:      repository.NewMetricFilterRepository(db),
		EventPipeline:     repository.NewEventPipelineRepository(db),
		EventPipelineExec: repository.NewEventPipelineExecutionRepository(db),
		Annotation:        repository.NewAnnotationRepository(db),
		LLMConfig:         repository.NewLLMConfigRepository(db),
		BuiltinDashboard:  repository.NewBuiltinDashboardRepository(db),
		TaskTpl:           repository.NewTaskTplRepository(db),
		TaskRecord:        repository.NewTaskRecordRepository(db),
		TeamNotifyChannel: repository.NewTeamNotifyChannelRepository(db),
		UserTeamNotifyPref: repository.NewUserTeamNotifyPrefRepository(db),
		StatusSubscription: repository.NewStatusSubscriptionRepository(db),
		SavedView:         repository.NewSavedViewRepository(db),
		MetricView:        repository.NewMetricViewRepository(db),
		MCPServer:         repository.NewMCPServerRepository(db),
		AISkill:           repository.NewAISkillRepository(db),
		ESIndexPattern:    repository.NewESIndexPatternRepository(db),
	}
}

// serviceBundle holds all service instances and shared components.
type serviceBundle struct {
	// Core services
	SettingSvc         *service.SystemSettingService
	DSSvc              *service.DataSourceService
	RuleSvc            *service.AlertRuleService
	AuthSvc            *service.AuthService
	LarkSvc            *service.LarkService
	AISvc              *service.AIService
	QueryClient        *datasource.QueryClient
	AlertPipeline      *service.AlertPipeline
	NotifyMediaSvc     *service.NotifyMediaService
	MessageTemplateSvc *service.MessageTemplateService
	NotifyRuleSvc      *service.NotifyRuleService
	SubscribeRuleSvc   *service.SubscribeRuleService
	NotifySvc          *service.NotificationService
	UserSvc            *service.UserService
	TeamSvc            *service.TeamService
	ScheduleSvc        *service.ScheduleService
	MuteRuleSvc        *service.MuteRuleService
	InhibitionRuleSvc  *service.InhibitionRuleService
	BizGroupSvc        *service.BizGroupService
	LabelRegistrySvc   *service.LabelRegistryService
	AuditLogSvc        *service.AuditLogService
	KnowledgeSvc       *service.KnowledgeBaseService
	DashboardV2Svc     *service.DashboardService
	TemplateSvc        *service.AlertRuleTemplateService
	ChannelV2Svc       *service.ChannelService
	IncidentSvc        *service.IncidentService
	AlertV2Svc         *service.AlertV2Service
	ExclusionRuleSvc   *service.ExclusionRuleService
	NoiseReducer       *service.NoiseReducer
	DispatchSvc        *service.DispatchService
	ScheduledDispatchSvc *service.ScheduledDispatchService
	PostMortemSvc      *service.PostMortemService
	AlertChannelSvc    *service.AlertChannelService
	UserNotifyConfigSvc *service.UserNotifyConfigService
	TeamNotifyChannelSvc *service.TeamNotifyChannelService
	UserTeamNotifyPrefSvc *service.UserTeamNotifyPrefService
	StatusServiceSvc   *service.StatusServiceService
	StatusSubSvc       *service.StatusSubscriptionService
	ChatHistorySvc     *service.ChatHistoryService
	PresetRuleSvc      *service.PresetRuleService
	UserPreferenceSvc  *service.UserPreferenceService
	UserNotificationSvc *service.UserNotificationService
	RuleGenSvc         *service.RuleGeneratorService
	AgentSvc           *service.AgentService
	ChangeEventSvc     *service.ChangeEventService
	DiagnosticWorkflowSvc *service.DiagnosticWorkflowService
	InspectionExecutor *service.InspectionExecutor
	RecordingRuleSvc   *service.RecordingRuleService
	AnnotationSvc      *service.AnnotationService
	SavedViewSvc       *service.SavedViewService
	MetricViewSvc      *service.MetricViewService
	MCPServerSvc       *service.MCPServerService
	LLMConfigSvc       *service.LLMConfigService
	AISkillSvc         *service.AISkillService
	ESIndexPatternSvc  *service.ESIndexPatternService
	BuiltinDashboardSvc *service.BuiltinDashboardService
	TaskTplSvc         *service.TaskTplService
	TaskExecutor       *service.TaskExecutor
	BuiltinMetricSvc   *service.BuiltinMetricService
	MetricFilterSvc    *service.MetricFilterService
	AlertmanagerImportSvc *service.AlertmanagerImportService
	EventSvc           *service.AlertEventService
	LarkBotSvc         *service.LarkBotService
	LDAPSvc            *service.LDAPService
	OAuth2Svc          *service.OAuth2Service
	UserContactSvc     *service.UserContactService
	DashboardStatsSvc  *service.DashboardStatsService

	// Shared engine components
	AlertWorkerPool *engine.AlertWorkerPool
	PipelineEngine  *ppipeline.Engine
}

// initServices creates all service instances. Services that require Redis
// (e.g. UserContactSvc) receive redisClient which may be nil.
func initServices(repos *repoBundle, db *gorm.DB, cfg *config.Config, zapLogger *zap.Logger, redisClient *sredis.Client) *serviceBundle {
	svcs := &serviceBundle{}

	// Worker pool — prevents goroutine exhaustion during alert storms.
	svcs.AlertWorkerPool = engine.NewAlertWorkerPool(64, zapLogger)

	// Core services
	svcs.SettingSvc = service.NewSystemSettingService(repos.SystemSetting, zapLogger)
	svcs.DSSvc = service.NewDataSourceService(repos.DS, zapLogger)
	svcs.DSSvc.SetRuleCountFn(repos.Rule.CountByDataSourceID) // P1-11: cascade check on delete
	svcs.RuleSvc = service.NewAlertRuleService(repos.Rule, repos.AlertRuleHistory, repos.DS, zapLogger)
	svcs.RuleSvc.SetSystemSettingService(svcs.SettingSvc)
	svcs.AuthSvc = service.NewAuthService(repos.User, &cfg.JWT, svcs.SettingSvc, zapLogger)
	svcs.LarkSvc = service.NewLarkService(zapLogger, cfg.Server.ExternalURL(), cfg.JWT.Secret, svcs.SettingSvc)
	svcs.AISvc = service.NewAIService(svcs.SettingSvc, zapLogger)
	svcs.QueryClient = datasource.NewQueryClient()
	contextBuilder := service.NewAlertContextBuilder(repos.Rule, repos.DS, svcs.QueryClient, zapLogger)
	svcs.AlertPipeline = service.NewAlertPipeline(contextBuilder, svcs.AISvc, zapLogger)

	// Phase 2 services (created before notifySvc so they can be passed as constructor params)
	svcs.NotifyMediaSvc = service.NewNotifyMediaService(repos.NotifyMedia, zapLogger)
	svcs.MessageTemplateSvc = service.NewMessageTemplateService(repos.MessageTemplate, zapLogger)
	svcs.NotifyRuleSvc = service.NewNotifyRuleService(
		repos.NotifyRule, repos.NotifyMedia, repos.MessageTemplate, repos.Record,
		repos.Rule, repos.DS,
		svcs.NotifyMediaSvc, svcs.MessageTemplateSvc, svcs.AlertPipeline, nil, zapLogger,
	)
	svcs.SubscribeRuleSvc = service.NewSubscribeRuleService(repos.SubscribeRule, zapLogger)

	svcs.NotifySvc = service.NewNotificationService(svcs.SubscribeRuleSvc, svcs.NotifyRuleSvc, repos.Rule, zapLogger)
	svcs.UserSvc = service.NewUserService(repos.User, zapLogger)
	svcs.TeamSvc = service.NewTeamService(repos.Team, zapLogger)
	svcs.ScheduleSvc = service.NewScheduleService(repos.Schedule, repos.Participant, repos.Override, repos.OnCallShift, repos.EscalationPolicy, repos.EscalationStep, repos.User, repos.Team, zapLogger)
	svcs.MuteRuleSvc = service.NewMuteRuleService(repos.MuteRule, zapLogger)
	svcs.InhibitionRuleSvc = service.NewInhibitionRuleService(repos.InhibitionRule, zapLogger)
	svcs.BizGroupSvc = service.NewBizGroupService(repos.BizGroup, zapLogger)

	// Label registry service
	svcs.LabelRegistrySvc = service.NewLabelRegistryService(repos.LabelRegistry, repos.DS, zapLogger)

	// Audit log service
	svcs.AuditLogSvc = service.NewAuditLogService(repos.AuditLog, zapLogger)

	// Knowledge base service
	svcs.KnowledgeSvc = service.NewKnowledgeBaseService(repos.Knowledge, svcs.AISvc, zapLogger)

	// Dashboard v2 service
	svcs.DashboardV2Svc = service.NewDashboardService(repos.DashboardV2, zapLogger)
	svcs.DashboardV2Svc.SetBizGroupRepository(repos.DashboardBizGroup)

	// Alert rule template service
	svcs.TemplateSvc = service.NewAlertRuleTemplateService(repos.AlertRuleTemplate, zapLogger)

	// v2 collaboration channel, incident, alert, noise-reduction & dispatch services
	svcs.ChannelV2Svc = service.NewChannelService(repos.ChannelV2, zapLogger)
	svcs.IncidentSvc = service.NewIncidentService(repos.Incident, svcs.ChannelV2Svc, zapLogger)
	svcs.IncidentSvc.SetAlertRepository(repos.AlertV2) // for incident merge alert migration
	svcs.AlertV2Svc = service.NewAlertV2Service(repos.AlertV2, zapLogger)
	svcs.ExclusionRuleSvc = service.NewExclusionRuleService(repos.ExclusionRule, zapLogger)
	svcs.NoiseReducer = service.NewNoiseReducer(repos.ChannelV2, repos.ExclusionRule, zapLogger)
	svcs.DispatchSvc = service.NewDispatchService(repos.DispatchPolicy, repos.DispatchLog, zapLogger)

	// Scheduled dispatch service (deferred/repeating notifications from dispatch policies)
	svcs.ScheduledDispatchSvc = service.NewScheduledDispatchService(
		repos.ScheduledDispatch, repos.DispatchPolicy, repos.Event,
		svcs.NotifyMediaSvc, svcs.MessageTemplateSvc, repos.NotifyMedia, zapLogger,
	)

	// Cancel scheduled dispatches when incident is acknowledged or closed
	svcs.IncidentSvc.SetOnStatusChange(func(ctx context.Context, incID uint, _ model.IncidentStatus) {
		if err := svcs.ScheduledDispatchSvc.CancelByIncident(ctx, incID); err != nil {
			zapLogger.Warn("failed to cancel scheduled dispatches on incident status change",
				zap.Uint("incident_id", incID), zap.Error(err))
		}
	})

	svcs.PostMortemSvc = service.NewPostMortemService(repos.PostMortem, repos.Incident, zapLogger)

	// Dispatch services
	svcs.AlertChannelSvc = service.NewAlertChannelService(repos.AlertChannel, repos.NotifyMedia, svcs.NotifyMediaSvc, zapLogger)
	svcs.UserNotifyConfigSvc = service.NewUserNotifyConfigService(repos.UserNotifyConfig, zapLogger)

	// Team notify channel service
	svcs.TeamNotifyChannelSvc = service.NewTeamNotifyChannelService(repos.TeamNotifyChannel, repos.NotifyMedia, zapLogger)

	// User team notify preference service
	svcs.UserTeamNotifyPrefSvc = service.NewUserTeamNotifyPrefService(repos.UserTeamNotifyPref, zapLogger)

	// Status service
	svcs.StatusServiceSvc = service.NewStatusServiceService(repos.StatusService, zapLogger)
	svcs.StatusSubSvc = service.NewStatusSubscriptionService(repos.StatusSubscription)

	// Chat history service
	svcs.ChatHistorySvc = service.NewChatHistoryService(repos.ChatHistory)

	// Preset rule service
	svcs.PresetRuleSvc = service.NewPresetRuleService(repos.PresetRule, repos.Rule, repos.DS, zapLogger)

	// User preference service
	svcs.UserPreferenceSvc = service.NewUserPreferenceService(repos.UserPreference, zapLogger)

	// Notification center service
	svcs.UserNotificationSvc = service.NewUserNotificationService(repos.UserNotification, zapLogger)

	// AI rule generation service
	svcs.RuleGenSvc = service.NewRuleGeneratorService(svcs.AISvc, svcs.LabelRegistrySvc, svcs.DSSvc, svcs.RuleSvc, repos.PresetRule, repos.DS, zapLogger)

	// AI Agent service (Phase 3 — 自主执行能力)
	svcs.AgentSvc = service.NewAgentService(svcs.AISvc, repos.AIConversation, nil, zapLogger)

	// AIOps Phase 2 services
	svcs.ChangeEventSvc = service.NewChangeEventService(repos.ChangeEvent, zapLogger)
	svcs.DiagnosticWorkflowSvc = service.NewDiagnosticWorkflowService(repos.DiagnosticWorkflow, svcs.DSSvc, svcs.AISvc, svcs.ChangeEventSvc, zapLogger)

	// Inspection executor (scheduler created after engine block for Leader access)
	svcs.InspectionExecutor = service.NewInspectionExecutor(repos.Inspection, svcs.AgentSvc, zapLogger)

	// Recording rule service
	svcs.RecordingRuleSvc = service.NewRecordingRuleService(repos.RecordingRule, zapLogger)

	// Annotation service
	svcs.AnnotationSvc = service.NewAnnotationService(repos.Annotation, zapLogger)

	// Saved view service
	svcs.SavedViewSvc = service.NewSavedViewService(repos.SavedView, zapLogger)

	// Metric view service
	svcs.MetricViewSvc = service.NewMetricViewService(repos.MetricView, zapLogger)

	// MCP server service
	svcs.MCPServerSvc = service.NewMCPServerService(repos.MCPServer, zapLogger)

	// LLM config service
	svcs.LLMConfigSvc = service.NewLLMConfigService(repos.LLMConfig, db, zapLogger)

	// AI Skill service
	svcs.AISkillSvc = service.NewAISkillService(repos.AISkill, zapLogger)

	// ES index pattern service
	svcs.ESIndexPatternSvc = service.NewESIndexPatternService(repos.ESIndexPattern, db, zapLogger)

	// Builtin dashboard service
	svcs.BuiltinDashboardSvc = service.NewBuiltinDashboardService(repos.BuiltinDashboard, repos.DashboardV2, zapLogger)

	// Task execution services
	svcs.TaskTplSvc = service.NewTaskTplService(repos.TaskTpl, zapLogger)
	svcs.TaskExecutor = service.NewTaskExecutor(repos.TaskTpl, repos.TaskRecord, zapLogger)

	// Builtin metric services
	svcs.BuiltinMetricSvc = service.NewBuiltinMetricService(repos.BuiltinMetric, zapLogger)
	svcs.MetricFilterSvc = service.NewMetricFilterService(repos.MetricFilter, zapLogger)

	// Event pipeline engine
	svcs.PipelineEngine = ppipeline.NewEngine(repos.EventPipeline, repos.EventPipelineExec, zapLogger)

	// Wire AI pipeline into event pipeline processors
	pprocessors.SetAIPipeline(svcs.AlertPipeline)

	// Wire pipeline engine into notify rule service (for PipelineID references)
	svcs.NotifyRuleSvc.SetPipelineEngine(svcs.PipelineEngine, repos.EventPipeline)

	// Alertmanager config import service
	svcs.AlertmanagerImportSvc = service.NewAlertmanagerImportService(svcs.ChannelV2Svc, svcs.InhibitionRuleSvc, zapLogger)

	// AlertEventService — all dependencies resolved via constructor (no setters).
	svcs.EventSvc = service.NewAlertEventService(repos.Event, repos.Timeline, svcs.NotifySvc, svcs.ScheduleSvc, svcs.LarkSvc, svcs.AlertWorkerPool, zapLogger)

	svcs.LarkBotSvc = service.NewLarkBotService(svcs.SettingSvc, svcs.EventSvc, svcs.ScheduleSvc, repos.User, zapLogger)

	// LDAP + OAuth2 services
	svcs.LDAPSvc = service.NewLDAPService(svcs.SettingSvc, repos.User, zapLogger)
	svcs.OAuth2Svc = service.NewOAuth2Service(svcs.SettingSvc, repos.User, zapLogger)

	// User contact service (needs Redis for verification codes)
	if redisClient != nil {
		svcs.UserContactSvc = service.NewUserContactService(repos.UserContact, redisClient.Raw(), svcs.SettingSvc, zapLogger)
	} else {
		svcs.UserContactSvc = service.NewUserContactService(repos.UserContact, nil, svcs.SettingSvc, zapLogger)
	}

	// Dashboard stats service
	svcs.DashboardStatsSvc = service.NewDashboardStatsService(db, zapLogger)

	// Seed default notification media and templates
	seedSvc := service.NewSeedService(repos.NotifyMedia, repos.MessageTemplate, zapLogger)
	if err := seedSvc.SeedDefaults(context.Background()); err != nil {
		zapLogger.Error("failed to seed default notification data", zap.Error(err))
	}

	// Seed built-in dashboards (runs once when table is empty)
	if err := svcs.BuiltinDashboardSvc.SeedDefaults(context.Background()); err != nil {
		zapLogger.Error("failed to seed builtin dashboards", zap.Error(err))
	}

	return svcs
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
	repos := initRepositories(db)
	// Wire team-role querier for RBAC team-role elevation in RequirePerm middleware.
	middleware.TeamRoleQuerier = &teamRoleAdapter{teamRepo: repos.Team}

	// --------------- Services ---------------
	svcs := initServices(repos, db, cfg, zapLogger, nil) // Redis injected below

	// --------------- OIDC service (optional) ---------------
	oidcSvc := d.initOIDCService(cfg, svcs.SettingSvc, repos.User, zapLogger)

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
			// Inject Redis into AuthService for login rate limiting
			svcs.AuthSvc.SetFailStore(rc)
			// Inject Redis token revocation checker for JWT blacklist
			middleware.TokenRevocationChecker = &tokenRevocationAdapter{rc: rc}
			// Inject Redis token blacklister into UserService for user-disable revocation
			svcs.UserSvc.SetTokenBlacklister(rc)
			// Inject Redis StreamBus into AgentService for multi-instance SSE
			streamBus := sredis.NewStreamBus(rc, zapLogger)
			svcs.AgentSvc.SetStreamBus(streamBus)
			// Inject Redis client for task state persistence (cross-instance GetTask)
			svcs.AgentSvc.SetRedisClient(rc)
			// Inject Redis-backed rate limiter for login brute-force protection
			// (shared across instances, unlike in-memory which is per-process)
			middleware.LoginRateLimiter = sredis.NewRedisRateLimiter(rc, 5, 5)
			// Re-init user contact service with Redis for verification codes
			svcs.UserContactSvc = service.NewUserContactService(repos.UserContact, rc.Raw(), svcs.SettingSvc, zapLogger)
			zapLogger.Info("redis connected, engine state persistence enabled, agent SSE stream bus enabled, login rate limiter backed by redis",
				zap.String("addr", cfg.Redis.Addr()),
			)
		}
	} else {
		zapLogger.Info("redis not configured, engine will use in-memory state only")
	}

	// --------------- Engine components ---------------

	// Initialize and start the escalation executor
	escalationExecutor := engine.NewEscalationExecutor(
		repos.EscalationPolicy,
		repos.EscalationStep,
		repos.StepExec,
		repos.Event,
		repos.Timeline,
		repos.Channel,
		repos.User,
		svcs.NotifyMediaSvc,
		repos.UserNotifyConfig,
		repos.Team,
		repos.OnCallShift,
		svcs.LarkSvc,
		svcs.SettingSvc,
		repos.Rule,
		zapLogger,
	)
	escalationExecutor.Start()

	// Inject Redis-backed dedup service into notify rule path
	if redisClient != nil {
		notifyDedupSvc := service.NewNotificationDedupService(redisClient.Raw(), zapLogger)
		svcs.NotifyRuleSvc.SetDedupService(notifyDedupSvc)
	}

	// Initialize and start the heartbeat checker
	heartbeatChecker := engine.NewHeartbeatChecker(repos.Rule, repos.Event, repos.Timeline, zapLogger)
	if cfg.Engine.HeartbeatInterval > 0 {
		heartbeatChecker.SetInterval(time.Duration(cfg.Engine.HeartbeatInterval) * time.Second)
	}
	heartbeatChecker.SetWorkerPool(svcs.AlertWorkerPool)

	// Initialize alert group manager (group_wait / group_interval)
	alertGroupMgr := service.NewAlertGroupManager(
		func(ctx context.Context, event *model.AlertEvent) error {
			return svcs.NotifySvc.RouteAlert(ctx, event)
		},
		repos.Rule,
		zapLogger,
	)

	// Evaluator pointer — declared here so the onAlertFn closure can capture it.
	var evaluator *engine.Evaluator

	// Shared onAlert callback used by both the evaluator and heartbeat checker.
	// Pipeline: bizgroup → inhibition → mute → noise reduction → group → notify.
	onAlertFn := func(ctx context.Context, event *model.AlertEvent) {
		// 1. Annotate event with matching BizGroup scope (must be before
		//    inhibition/mute so match_labels on biz_group can work).
		if groups, err := svcs.BizGroupSvc.FindMatchingGroups(ctx, map[string]string(event.Labels)); err == nil && len(groups) > 0 {
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
			_ = repos.Event.UpdateLabels(ctx, event.ID, event.Labels)
		}

		// 2. Check inhibition rules (suppress target alerts when source is firing).
		var firingEvents []model.AlertEvent
		if evaluator != nil {
			firingEvents = evaluator.GetFiringAlertEvents()
		} else {
			firingEvents, _, _ = svcs.EventSvc.List(ctx, "firing", "", 1, 2000)
		}
		if svcs.InhibitionRuleSvc.IsInhibited(ctx, event, firingEvents) {
			zapLogger.Info("alert inhibited by inhibition rule, skipping notification",
				zap.Uint("event_id", event.ID),
				zap.String("alert_name", event.AlertName),
			)
			return
		}

		// 3. Check mute rules.
		if svcs.MuteRuleSvc.IsAlertMuted(ctx, event) {
			zapLogger.Info("alert muted, skipping notification",
				zap.Uint("event_id", event.ID),
				zap.String("alert_name", event.AlertName),
			)
			return
		}

		// 4. Noise reduction: check exclusion rules + flapping state before notification.
		if svcs.NoiseReducer != nil {
			if suppressed, reason := svcs.NoiseReducer.ShouldSuppressForNotify(ctx, event); suppressed {
				zapLogger.Info("alert excluded by noise reduction, skipping notification",
					zap.Uint("event_id", event.ID),
					zap.String("alert_name", event.AlertName),
					zap.String("reason", reason),
				)
				return
			}
		}

		// 5. Route notification (through group manager for group_wait/group_interval).
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
	alertV2Pipeline := service.NewAlertV2Pipeline(repos.AlertV2, repos.Event, repos.Incident, repos.ChannelV2, zapLogger)
	alertV2Pipeline.InitDefaultChannel(context.Background())
	alertV2Pipeline.SetNoiseReducer(svcs.NoiseReducer)
	alertV2Pipeline.SetDispatchService(svcs.DispatchSvc)
	alertV2Pipeline.SetScheduledDispatchService(svcs.ScheduledDispatchSvc)

	// Wire default channel ID into noise reducer so ShouldSuppress works for engine alerts
	// (which don't carry _channel_id label).
	svcs.NoiseReducer.SetDefaultChannelID(alertV2Pipeline.GetDefaultChannelID())

	// Incident aggregator: bridges AlertEvent fingerprint to Incident lifecycle.
	incidentAggregator := service.NewIncidentAggregator(svcs.IncidentSvc, repos.Event, repos.Incident, alertV2Pipeline.GetDefaultChannelID(), zapLogger)
	alertV2Pipeline.SetIncidentAggregator(incidentAggregator)

	onAlertFn = alertV2Pipeline.WrapOnAlert(onAlertFn)

	// Integration service needs the pipeline (must be after pipeline setup)
	integrationSvc := service.NewIntegrationService(repos.Integration, repos.RoutingRule, alertV2Pipeline, zapLogger)

	// Start the incident auto-close background worker.
	svcs.IncidentSvc.StartAutoCloseWorker(appCtx)

	// Start the scheduled dispatch background worker.
	svcs.ScheduledDispatchSvc.StartWorker(appCtx)

	// Wire the heartbeat checker into the notification pipeline.
	heartbeatChecker.SetOnAlert(onAlertFn)

	// Initialize alert evaluator
	var engineHandler *handler.EngineHandler

	if cfg.Engine.Enabled {
		evaluator = engine.NewEvaluator(
			db, repos.DS, repos.Rule, repos.Event, repos.Timeline, svcs.QueryClient, zapLogger,
		)

		if stateStore != nil {
			evaluator.SetStateStore(stateStore)
		}
		evaluator.SetWorkerPool(svcs.AlertWorkerPool)
		if cfg.Engine.SyncInterval > 0 {
			evaluator.SetSyncInterval(time.Duration(cfg.Engine.SyncInterval) * time.Second)
		}
		evaluator.SetPerDatasourceEval(cfg.Engine.PerDatasourceEval)
		evaluator.SetOnAlert(onAlertFn)
		evaluator.SetMuteRuleRepository(repos.MuteRule)
		evaluator.SetLabelRegistryRepository(repos.LabelRegistry)

		if cfg.Engine.HashRingEnabled && redisClient != nil {
			// Hash ring mode: distribute rules across instances
			instanceID := cfg.Engine.InstanceID
			if instanceID == "" {
				hostname, _ := os.Hostname()
				instanceID = fmt.Sprintf("%s:%d", hostname, os.Getpid())
			}
			replicas := cfg.Engine.HashRingReplicas
			if replicas <= 0 {
				replicas = hashring.DefaultReplicas
			}

			ring := hashring.New(replicas)
			ring.Add(instanceID)
			evaluator.SetHashRing(ring, instanceID)

			// Register this instance in Redis and start ring membership refresh.
			// Uses a key prefix + instance ID with a 30s TTL; each instance
			// refreshes every 10s so stale entries are cleaned up automatically.
			rdb := redisClient.Raw()
			const ringPrefix = "sreagent:engine:ring:"
			const ringTTL = 30 * time.Second
			const ringRefresh = 10 * time.Second

			ctx := context.Background()
			rdb.Set(ctx, ringPrefix+instanceID, "1", ringTTL)

			go func() {
				ticker := time.NewTicker(ringRefresh)
				defer ticker.Stop()
				for {
					select {
					case <-ticker.C:
						// Refresh own registration
						rdb.Set(ctx, ringPrefix+instanceID, "1", ringTTL)
						// Discover all active instances
						keys, err := rdb.Keys(ctx, ringPrefix+"*").Result()
						if err != nil {
							zapLogger.Warn("hash ring: failed to discover instances", zap.Error(err))
							continue
						}
						newRing := hashring.New(replicas)
						for _, k := range keys {
							newRing.Add(k[len(ringPrefix):])
						}
						evaluator.UpdateHashRing(newRing)
					case <-ctx.Done():
						return
					}
				}
			}()

			zapLogger.Info("hash ring mode enabled",
				zap.String("instance_id", instanceID),
				zap.Int("replicas", replicas),
			)
		} else if redisClient != nil {
			// Leader election: only one instance evaluates rules at a time
			leader := engine.NewRedisLeaderElection(redisClient.Raw(), zapLogger)
			evaluator.SetLeaderElection(leader)
			heartbeatChecker.SetLeaderElection(leader)
			d.Leader = leader
		}

		// Wire datasource change callback so endpoint updates trigger evaluator re-sync.
		svcs.DSSvc.SetChangeCallback(evaluator)

		evaluator.Start()

		engineHandler = handler.NewEngineHandler(evaluator)
	}

	heartbeatChecker.Start()

	// Initialize and start the recording rule engine
	recordingRuleEngine := engine.NewRecordingRuleEngine(
		repos.RecordingRule, repos.DS, db, svcs.QueryClient, zapLogger,
	)
	if d.Leader != nil {
		recordingRuleEngine.SetLeaderElection(d.Leader)
	}
	recordingRuleEngine.Start(appCtx)

	// Inspection scheduler (created after engine block so d.Leader is set)
	inspectionSched := service.NewInspectionScheduler(repos.Inspection, svcs.InspectionExecutor, d.Leader, svcs.LarkBotSvc, zapLogger)

	// --------------- AI 工具注册表 ---------------
	toolRegistry := service.NewAIToolRegistry(zapLogger)
	toolRegistry.RegisterBuiltinTools(svcs.DSSvc, svcs.RuleSvc, svcs.IncidentSvc, svcs.AuditLogSvc, svcs.EventSvc, svcs.KnowledgeSvc,
		func() (interface{}, bool) {
			if evaluator == nil {
				return nil, false
			}
			return evaluator.GetStatus(), true
		},
	)
	// Register MCP tools from enabled MCP servers (non-blocking, logs warnings on failure)
	toolRegistry.RegisterMCPTools(svcs.MCPServerSvc)
	svcs.AISvc.SetToolRegistry(toolRegistry)
	svcs.AgentSvc.SetToolRegistry(toolRegistry)

	// Start inspection scheduler (loads enabled tasks from DB)
	if err := inspectionSched.Start(context.Background()); err != nil {
		zapLogger.Error("巡检调度器启动失败", zap.Error(err))
	}

	// --------------- Handlers ---------------

	// OIDC handler — uses getter function for hot-reload support
	var oidcHandler *handler.OIDCHandler
	if oidcSvc != nil {
		oidcHandler = handler.NewOIDCHandler(oidcSvc)
		d.oidcSvc = oidcSvc
		d.oidcHdlr = oidcHandler
	}

	teamNotifyChannelHandler := handler.NewTeamNotifyChannelHandler(svcs.TeamNotifyChannelSvc, zapLogger)
	userTeamNotifyPrefHandler := handler.NewUserTeamNotifyPrefHandler(svcs.UserTeamNotifyPrefSvc, zapLogger)
	statusSubHandler := handler.NewStatusSubscriptionHandler(svcs.StatusSubSvc, zapLogger)

	handlers := &router.Handlers{
		Auth:             func() *handler.AuthHandler { h := handler.NewAuthHandler(svcs.AuthSvc); h.SetUserService(svcs.UserSvc); h.SetRedis(redisClient); h.SetLDAPService(svcs.LDAPSvc); return h }(),
		OIDC:             oidcHandler,
		OIDCSettings:     handler.NewOIDCSettingsHandler(svcs.SettingSvc, d.ReloadOIDC),
		OAuth2:           handler.NewOAuth2Handler(svcs.OAuth2Svc, cfg.JWT.Secret, cfg.JWT.Expire),
		SSOSettings:      handler.NewSSOSettingsHandler(svcs.LDAPSvc, svcs.OAuth2Svc),
		DataSource:       handler.NewDataSourceHandler(svcs.DSSvc, zapLogger),
		AlertRule:        handler.NewAlertRuleHandler(svcs.RuleSvc, zapLogger),
		AlertEvent:       handler.NewAlertEventHandler(svcs.EventSvc, zapLogger),
		User:             handler.NewUserHandler(svcs.UserSvc, zapLogger),
		Team:             handler.NewTeamHandler(svcs.TeamSvc, zapLogger),
		Schedule:         handler.NewScheduleHandler(svcs.ScheduleSvc, zapLogger),
		Dashboard:        handler.NewDashboardHandler(svcs.DashboardStatsSvc),
		AI:               handler.NewAIHandler(svcs.AISvc, svcs.SettingSvc, svcs.EventSvc, svcs.ChatHistorySvc),
		LarkBot:          handler.NewLarkBotHandler(svcs.LarkBotSvc),
		Engine:           engineHandler,
		AlertAction:      handler.NewAlertActionHandler(svcs.EventSvc, repos.User, cfg.JWT.Secret, zapLogger),
		MuteRule:         handler.NewMuteRuleHandler(svcs.MuteRuleSvc, svcs.EventSvc, zapLogger),
		NotifyRule:       handler.NewNotifyRuleHandler(svcs.NotifyRuleSvc, zapLogger),
		NotifyMedia:      handler.NewNotifyMediaHandler(svcs.NotifyMediaSvc, zapLogger),
		MessageTemplate:  handler.NewMessageTemplateHandler(svcs.MessageTemplateSvc, zapLogger),
		SubscribeRule:    handler.NewSubscribeRuleHandler(svcs.SubscribeRuleSvc, zapLogger),
		BizGroup:         handler.NewBizGroupHandler(svcs.BizGroupSvc, zapLogger),
		AlertChannel:     handler.NewAlertChannelHandler(svcs.AlertChannelSvc, zapLogger),
		UserNotifyConfig: handler.NewUserNotifyConfigHandler(svcs.UserNotifyConfigSvc),
		AuditLog:         handler.NewAuditLogHandler(svcs.AuditLogSvc),
		SMTPSettings:     handler.NewSMTPSettingsHandler(svcs.SettingSvc),
		SecuritySettings: handler.NewSecuritySettingsHandler(svcs.SettingSvc, &cfg.JWT),
		InhibitionRule:   handler.NewInhibitionRuleHandler(svcs.InhibitionRuleSvc, svcs.EventSvc, zapLogger),
		Heartbeat:        handler.NewHeartbeatHandler(svcs.RuleSvc),
		LabelRegistry:    handler.NewLabelRegistryHandler(svcs.LabelRegistrySvc),
		DashboardV2:      handler.NewDashboardV2Handler(svcs.DashboardV2Svc),
		AlertRuleTemplate:   handler.NewAlertRuleTemplateHandler(svcs.TemplateSvc),
		ChannelV2:           handler.NewChannelHandler(svcs.ChannelV2Svc),
		IncidentV2:          handler.NewIncidentHandler(svcs.IncidentSvc),
		AlertV2:             handler.NewAlertV2Handler(svcs.AlertV2Svc),
		ExclusionRule:       handler.NewExclusionRuleHandler(svcs.ExclusionRuleSvc),
		DispatchPolicy:      handler.NewDispatchHandler(svcs.DispatchSvc),
		Integration:         handler.NewIntegrationHandler(integrationSvc, zapLogger),
		RoutingRule:         handler.NewRoutingRuleHandler(service.NewRoutingRuleService(repos.RoutingRule)),
		PostMortem:          handler.NewPostMortemHandler(svcs.PostMortemSvc, svcs.AISvc),
		StatusService:       handler.NewStatusServiceHandler(svcs.StatusServiceSvc),
		PresetRule:          handler.NewPresetRuleHandler(svcs.PresetRuleSvc),
		AIRule:              handler.NewAIRuleHandler(svcs.RuleGenSvc),
		AlertmanagerImport:  handler.NewAlertmanagerImportHandler(svcs.AlertmanagerImportSvc),
		UserPreference:      handler.NewUserPreferenceHandler(svcs.UserPreferenceSvc),
		UserNotification:    handler.NewUserNotificationHandler(svcs.UserNotificationSvc),
		Permissions:         handler.NewPermissionsHandler(svcs.TeamSvc),
		Agent:               handler.NewAgentHandler(svcs.AgentSvc),
		Knowledge:           handler.NewKnowledgeHandler(svcs.KnowledgeSvc),
		DiagnosticWorkflow:  handler.NewDiagnosticWorkflowHandler(svcs.DiagnosticWorkflowSvc),
		ChangeEvent:         handler.NewChangeEventHandler(svcs.ChangeEventSvc),
		Inspection:          handler.NewInspectionHandler(service.NewInspectionService(repos.Inspection), inspectionSched, svcs.InspectionExecutor),
		RecordingRule:       handler.NewRecordingRuleHandler(svcs.RecordingRuleSvc, zapLogger),
		BuiltinMetric:       handler.NewBuiltinMetricHandler(svcs.BuiltinMetricSvc, svcs.MetricFilterSvc, zapLogger),
		EventPipeline:       handler.NewEventPipelineHandler(service.NewEventPipelineService(repos.EventPipeline), service.NewEventPipelineExecutionService(repos.EventPipelineExec), svcs.PipelineEngine, svcs.EventSvc, zapLogger),
		Annotation:          handler.NewAnnotationHandler(svcs.AnnotationSvc, zapLogger),
		SavedView:           handler.NewSavedViewHandler(svcs.SavedViewSvc, zapLogger),
		MetricView:          handler.NewMetricViewHandler(svcs.MetricViewSvc, zapLogger),
		MCPServer:           handler.NewMCPServerHandler(svcs.MCPServerSvc, zapLogger),
		LLMConfig:           handler.NewLLMConfigHandler(svcs.LLMConfigSvc, zapLogger),
		AISkill:             handler.NewAISkillHandler(svcs.AISkillSvc, zapLogger),
		ESIndexPattern:      handler.NewESIndexPatternHandler(svcs.ESIndexPatternSvc, zapLogger),
		SiteInfo:            handler.NewSiteInfoHandler(svcs.SettingSvc),
		TaskTpl:             handler.NewTaskTplHandler(svcs.TaskTplSvc, zapLogger),
		Task:                handler.NewTaskHandler(svcs.TaskExecutor, service.NewTaskRecordService(repos.TaskRecord), zapLogger),
		UserContact:         handler.NewUserContactHandler(svcs.UserContactSvc, zapLogger),
		BuiltinDashboard:    handler.NewBuiltinDashboardHandler(svcs.BuiltinDashboardSvc),
		StatusSubscription:  statusSubHandler,
		TeamNotifyChannel:   teamNotifyChannelHandler,
		UserTeamNotifyPref:  userTeamNotifyPrefHandler,
	}

	// Inject audit service into handlers that support it
	handlers.AlertRule.SetAuditService(svcs.AuditLogSvc)
	handlers.AlertEvent.SetAuditService(svcs.AuditLogSvc)
	handlers.User.SetAuditService(svcs.AuditLogSvc)
	handlers.DataSource.SetAuditService(svcs.AuditLogSvc)
	handlers.InhibitionRule.SetAuditService(svcs.AuditLogSvc)
	handlers.NotifyRule.SetAuditService(svcs.AuditLogSvc)
	handlers.NotifyMedia.SetAuditService(svcs.AuditLogSvc)
	handlers.Schedule.SetAuditService(svcs.AuditLogSvc)
	handlers.MuteRule.SetAuditService(svcs.AuditLogSvc)
	handlers.BizGroup.SetAuditService(svcs.AuditLogSvc)
	handlers.ChannelV2.SetAuditService(svcs.AuditLogSvc)
	handlers.RoutingRule.SetAuditService(svcs.AuditLogSvc)
	handlers.Annotation.SetAuditService(svcs.AuditLogSvc)
	handlers.SavedView.SetAuditService(svcs.AuditLogSvc)
	handlers.MetricView.SetAuditService(svcs.AuditLogSvc)
	handlers.RecordingRule.SetAuditService(svcs.AuditLogSvc)
	handlers.MCPServer.SetAuditService(svcs.AuditLogSvc)
	handlers.LLMConfig.SetAuditService(svcs.AuditLogSvc)
	handlers.ESIndexPattern.SetAuditService(svcs.AuditLogSvc)

	// Wire permission-denied audit callback into the RBAC middleware.
	middleware.SetPermLogger(zapLogger)
	middleware.OnPermissionDenied = func(userID uint, perm string, path string) {
		uid := userID
		svcs.AuditLogSvc.Record(&model.AuditLog{
			UserID:       &uid,
			Action:       model.AuditActionPermissionDenied,
			ResourceType: "permission",
			ResourceName: path,
			Detail:       fmt.Sprintf("permission: %s", perm),
			Status:       model.AuditResultDenied,
		})
	}


	// Store references needed for shutdown and hot reload
	d.DSRepo = repos.DS
	d.RuleRepo = repos.Rule
	d.EventRepo = repos.Event
	d.TimelineRepo = repos.Timeline
	d.UserRepo = repos.User
	d.ChannelRepo = repos.Channel
	d.TeamRepo = repos.Team
	d.EscPolicyRepo = repos.EscalationPolicy
	d.EscStepRepo = repos.EscalationStep
	d.OnCallShiftRepo = repos.OnCallShift
	d.AlertRuleHistoryRepo = repos.AlertRuleHistory
	d.SettingSvc = svcs.SettingSvc
	d.EventSvc = svcs.EventSvc
	d.NotifySvc = svcs.NotifySvc
	d.ScheduleSvc = svcs.ScheduleSvc
	d.MuteRuleSvc = svcs.MuteRuleSvc
	d.InhibRuleSvc = svcs.InhibitionRuleSvc
	d.BizGroupSvc = svcs.BizGroupSvc
	d.AlertV2Pipeline = alertV2Pipeline
	d.LabelRegistrySvc = svcs.LabelRegistrySvc
	d.AlertWorkerPool = svcs.AlertWorkerPool
	d.HeartbeatChecker = heartbeatChecker
	d.AlertGroupMgr = alertGroupMgr
	d.EscalationExecutor = escalationExecutor
	d.Evaluator = evaluator
	d.RecordingRuleEngine = recordingRuleEngine
	d.RedisClient = redisClient
	d.StateStore = stateStore
	d.Handlers = handlers
	d.InspectionSched = inspectionSched

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

// ReloadOIDC re-initializes the OIDC service from the DB and hot-swaps it
// on the handler. Called by OIDCSettingsHandler after config save.
func (d *Dependencies) ReloadOIDC() {
	d.oidcMu.Lock()
	defer d.oidcMu.Unlock()

	newSvc := d.initOIDCService(d.cfg, d.SettingSvc, d.UserRepo, d.logger)

	d.oidcSvc = newSvc
	if d.oidcHdlr != nil {
		d.oidcHdlr.SetService(newSvc)
	}

	if newSvc != nil {
		d.logger.Info("OIDC service hot-reloaded successfully")
	} else {
		d.logger.Info("OIDC service disabled after config update")
	}
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

	// 4.3 Stop recording rule engine
	if d.RecordingRuleEngine != nil {
		d.RecordingRuleEngine.Stop()
	}

	// 4.5 Stop inspection scheduler
	if d.InspectionSched != nil {
		d.InspectionSched.Stop()
	}

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
