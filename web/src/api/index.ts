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

// ===== Notification Center & Permissions =====
export { notificationCenterApi, permissionsApi } from './center'
export type { UserNotification, MyPermissions, TeamRole } from './center'
