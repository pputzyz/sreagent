// ===== Alert domain =====
export {
  alertRuleApi,
  alertEventApi,
  alertChannelApi,
  alertGroupsApi,
  alertExportApi,
  engineApi,
  expressionApi,
  templateApi,
} from './alert'

// ===== Notification domain =====
export {
  notifyRuleApi,
  notifyMediaApi,
  messageTemplateApi,
  subscribeRuleApi,
  muteRuleApi,
  inhibitionRuleApi,
} from './notify'

// ===== Oncall domain =====
export {
  teamApi,
  scheduleApi,
  escalationApi,
  scheduleICalApi,
} from './oncall'

// ===== Admin domain =====
export {
  userApi,
  bizGroupApi,
  aiApi,
  aiChatApi,
  aiAgentApi,
  authApi,
  auditLogApi,
  userNotifyConfigApi,
  larkBotApi,
  oidcSettingsApi,
  ldapSettingsApi,
  oauth2SettingsApi,
  smtpSettingsApi,
  securitySettingsApi,
  statusServiceApi,
} from './admin'
export type { StatusServiceItem, AgentTask, AgentStep } from './admin'

// ===== Data domain =====
export {
  datasourceApi,
  dashboardApi,
  dashboardV2Api,
  dashboardBizGroupApi,
  labelRegistryApi,
} from './data'

// ===== Incident domain =====
export {
  channelV2Api,
  incidentApi,
  postMortemApi,
  integrationV2Api,
  routingRuleApi,
  dispatchApi,
  alertV2Api,
  dashboardV2StatsApi,
} from './incident'

// ===== Preset rules & AI modules =====
export { presetRuleApi, aiModuleApi, aiRuleApi } from './preset-rules'

// ===== Recording Rules =====
export { recordingRuleApi } from './recording'
export type { RecordingRule, CreateRecordingRuleRequest, UpdateRecordingRuleRequest } from './recording'

// ===== Builtin Metrics =====
export { builtinMetricApi, metricFilterApi } from './builtin-metric'
export type { BuiltinMetric, MetricFilter, TranslationEntry } from './builtin-metric'

// ===== Notification Center & Permissions =====
export { notificationCenterApi, permissionsApi } from './center'
export type { UserNotification, MyPermissions, TeamRole } from './center'

// ===== Event Pipeline =====
export { eventPipelineApi } from './event-pipeline'
export type { EventPipeline, EventPipelineExecution, ProcessorConfig, TagFilter, NodeResult } from './event-pipeline'

// ===== Metric Views =====
export { metricViewApi } from './metric-view'
export type { MetricView, MetricViewConfig, MetricViewFilter, MetricViewDynamicLabel, CreateMetricViewRequest, UpdateMetricViewRequest } from './metric-view'

// ===== LLM Configs =====
export { llmConfigApi } from './llm-config'
export type { LLMConfig, LLMExtraConfig, CreateLLMConfigRequest, UpdateLLMConfigRequest, TestConnectionResponse } from './llm-config'

// ===== MCP Servers =====
export { mcpServerApi } from './mcp-server'
export type { MCPServer, MCPTool, CreateMCPServerRequest, UpdateMCPServerRequest } from './mcp-server'

// ===== AI Skills =====
export { aiSkillApi } from './ai-skill'
export type { AISkill, AISkillFile, CreateAISkillRequest, UpdateAISkillRequest } from './ai-skill'

// ===== ES Index Patterns =====
export { esIndexPatternApi } from './es-index-pattern'
export type { ESIndexPattern, CreateESIndexPatternRequest, UpdateESIndexPatternRequest } from './es-index-pattern'

// ===== Builtin Dashboards =====
export { builtinDashboardApi } from './builtin-dashboard'
export type { BuiltinDashboard } from './builtin-dashboard'

// ===== User Contacts =====
export { userContactApi } from './user-contact'
export type { UserContact } from './user-contact'

// ===== Site Info =====
export { siteInfoApi } from './admin'
export type { SiteInfo } from './admin'
