// ===== API Response Types =====
export interface ApiResponse<T = unknown> {
  code: number
  message: string
  data: T
}

export interface PageData<T> {
  list: T[]
  total: number
  page: number
  page_size: number
}

// ===== Auth =====
export interface LoginRequest {
  username: string
  password: string
}

export interface LoginResponse {
  token: string
  expires_in: number
}

export type UserType = 'human' | 'bot' | 'channel'

export interface User {
  id: number
  username: string
  display_name: string
  email: string
  phone: string
  lark_user_id: string
  avatar: string
  role: 'admin' | 'team_lead' | 'member'
  is_active: boolean
  created_at: string
  user_type?: UserType
  notify_target?: string
}

// ===== DataSource =====
export type DataSourceType = 'prometheus' | 'victoriametrics' | 'zabbix' | 'victorialogs' | 'elasticsearch'
export type DataSourceStatus = 'healthy' | 'unhealthy' | 'unknown'

export interface DataSource {
  id: number
  name: string
  type: DataSourceType
  endpoint: string
  description: string
  labels: Record<string, string>
  status: DataSourceStatus
  auth_type: string
  health_check_interval: number
  is_enabled: boolean
  supports_query: boolean
  version?: string
  created_at: string
  updated_at: string
}

// ===== Alert Rule =====
export type AlertSeverity = 'p0' | 'p1' | 'p2' | 'p3' | 'p4' | 'critical' | 'warning' | 'info'
export type AlertRuleStatus = 'active' | 'disabled' | 'draft'
export type AlertRuleType = 'threshold' | 'heartbeat'

export interface AlertRule {
  id: number
  rule_type: AlertRuleType
  name: string
  display_name: string
  description: string
  datasource_id: number | null
  datasource_type: DataSourceType | ''
  datasource?: DataSource
  expression: string
  for_duration: string
  severity: AlertSeverity
  labels: Record<string, string>
  annotations: Record<string, string>
  status: AlertRuleStatus
  enabled?: boolean
  group_name: string
  category: string
  version: number
  created_by: number
  updated_by: number
  created_at: string
  updated_at: string
  // Heartbeat monitoring fields (only relevant when rule_type='heartbeat')
  heartbeat_token: string
  heartbeat_interval: number
  heartbeat_last_at: string | null
  // SLA auto-escalation (0 = disabled)
  ack_sla_minutes: number
  // Group notification timing (0 = disabled)
  group_wait_seconds: number
  group_interval_seconds: number
  // Advanced fields
  eval_interval: number
  recovery_hold: string
  nodata_enabled: boolean
  nodata_duration: string
  suppress_enabled: boolean
}

// ===== Alert Event =====
export type AlertEventStatus = 'firing' | 'acknowledged' | 'assigned' | 'resolved' | 'closed' | 'silenced'

export interface AlertEvent {
  id: number
  fingerprint: string
  rule_id: number | null
  rule?: AlertRule
  alert_name: string
  severity: AlertSeverity
  status: AlertEventStatus
  labels: Record<string, string>
  annotations: Record<string, string>
  source: string
  generator_url: string
  fired_at: string
  acked_at: string | null
  resolved_at: string | null
  closed_at: string | null
  acked_by: number | null
  acked_by_user?: User
  assigned_to: number | null
  assigned_to_user?: User
  resolution: string
  fire_count: number
  silenced_until?: string
  silence_reason?: string
  oncall_user_id?: number | null
  oncall_user?: User
  is_dispatched?: boolean
  sla_escalated_at?: string | null
  created_at: string
}

export interface AlertEventFilter {
  status?: string[]
  severity?: string[]
  start_time?: string
  end_time?: string
  source?: string
  alert_name?: string
  business_line?: string
  view_mode?: AlertViewMode
  user_id?: number
  page: number
  page_size: number
}

export type AlertViewMode = 'mine' | 'unassigned' | 'all'

export type TimelineAction =
  | 'created'
  | 'acknowledged'
  | 'assigned'
  | 'commented'
  | 'escalated'
  | 'resolved'
  | 'closed'
  | 'reopened'
  | 'notified'

export interface AlertTimeline {
  id: number
  event_id: number
  action: TimelineAction
  operator_id: number | null
  operator?: User
  note: string
  extra: string
  created_at: string
}

// ===== Team =====
export interface Team {
  id: number
  name: string
  description: string
  labels: Record<string, string>
  members?: User[]
}

// ===== Schedule =====
export interface Schedule {
  id: number
  name: string
  team_id: number
  team?: Team
  description: string
  rotation_type: 'daily' | 'weekly' | 'custom'
  timezone: string
  handoff_time: string
  handoff_day: number
  is_enabled: boolean
  severity_filter?: string
  created_at: string
}

export interface OnCallShift {
  id: number
  schedule_id: number
  user_id: number
  user?: User
  start_time: string   // ISO date string
  end_time: string
  severity_filter: string  // "" | "critical" | "critical,warning" etc
  source: 'manual' | 'rotation'
  note: string
  created_at: string
}

export interface ScheduleParticipant {
  id: number
  schedule_id: number
  user_id: number
  user?: User
  position: number
}

export interface ScheduleOverride {
  id: number
  schedule_id: number
  user_id: number
  user?: User
  start_time: string
  end_time: string
  reason: string
}

// ===== Escalation Policy =====
export interface EscalationPolicy {
  id: number
  name: string
  description?: string
  team_id: number
  team?: Team
  is_enabled: boolean
  steps?: EscalationStep[]
  created_at: string
}

export interface EscalationStep {
  id?: number
  policy_id?: number
  step_order: number
  delay_minutes: number
  target_type: string
  target_id: number
  notify_channel_id?: number | null
}

// ===== Mute Rule =====
export interface MuteRule {
  id: number
  name: string
  description: string
  match_labels: Record<string, string>
  severities: string
  start_time: string | null
  end_time: string | null
  periodic_start: string
  periodic_end: string
  days_of_week: string
  timezone: string
  created_by: number
  is_enabled: boolean
  rule_ids: string
  created_at: string
}

// ===== Notify Rule (v2, replaces NotifyPolicy) =====
export interface NotifyRule {
  id: number
  name: string
  description: string
  is_enabled: boolean
  severities: string
  match_labels: Record<string, string>
  pipeline: string // JSON array of processor configs
  notify_configs: string // JSON array of notification configs
  repeat_interval: number
  max_notifications: number
  callback_url: string
  created_by: number
  created_at: string
}

/** Time range restriction for a notification config (Nightingale-compatible). */
export interface TimeRange {
  start: string // "HH:MM"
  end: string   // "HH:MM"
  week: number[] // ISO weekday: 1=Mon..7=Sun; empty = all days
}

/** Deserialized notification config element within a NotifyRule.notify_configs JSON. */
export interface NotifyConfigItem {
  severity: string
  media_id: number
  template_id: number
  user_ids?: number[]
  team_ids?: number[]
  time_ranges?: TimeRange[]
}

// ===== Notify Media (replaces NotifyChannel) =====
export interface NotifyMedia {
  id: number
  name: string
  type:
    | 'lark_webhook' | 'email' | 'http' | 'script'
    | 'dingtalk_webhook' | 'wecom_webhook' | 'slack_webhook' | 'discord_webhook'
    | 'telegram_bot' | 'feishu_webhook' | 'feishu_card' | 'feishu_app'
    | 'wecom_app' | 'flashduty' | 'pagerduty' | 'tencent_sms' | 'aliyun_sms'
  description: string
  is_enabled: boolean
  config: string
  variables: string
  is_builtin: boolean
  created_at: string
}

// ===== Message Template =====
export interface MessageTemplate {
  id: number
  name: string
  description: string
  content: string
  type: 'text' | 'html' | 'markdown' | 'lark_card'
  is_builtin: boolean
  created_at: string
}

// ===== Subscribe Rule =====
export interface SubscribeRule {
  id: number
  name: string
  description: string
  is_enabled: boolean
  match_labels: Record<string, string>
  severities: string
  // null ⇒ no override; server falls back to the default NotifyRule pipeline.
  notify_rule_id: number | null
  user_id: number | null
  team_id: number | null
  created_by: number
  created_at: string
}

// ===== Business Group =====
export interface BizGroup {
  id: number
  name: string
  description: string
  parent_id: number | null
  labels: Record<string, string>
  match_labels?: Record<string, string>
  children?: BizGroup[]
  created_at: string
}

// ===== Alert Channel =====
export interface AlertChannel {
  id: number
  name: string
  description: string
  match_labels: Record<string, string>
  severities: string
  media_id: number
  media?: NotifyMedia
  template_id: number | null
  template?: MessageTemplate
  throttle_min: number
  is_enabled: boolean
  created_by: number
  created_at: string
}

// ===== User Notify Config =====
export interface UserNotifyConfig {
  id: number
  user_id: number
  media_type: 'lark_personal' | 'email' | 'webhook'
  config: string
  is_enabled: boolean
}

// ===== Engine Status =====
export interface EngineStatus {
  running: boolean
  total_rules: number
  active_alerts: number
  uptime: string
}

// ===== Dashboard =====
export interface DashboardStats {
  total_datasources: number
  total_rules: number
  active_alerts: number
  resolved_today: number
  total_users: number
  total_teams: number
  severity_breakdown: { critical: number; warning: number; info: number }
}

/** Single latency metric: mean + percentiles. -1 means "no data". */
export interface MTTRMetric {
  mean: number
  p50: number
  p95: number
  count: number
}

export interface SeverityMTTR {
  severity: 'critical' | 'warning' | 'info' | string
  mtta: MTTRMetric
  mttr: MTTRMetric
}

export interface MTTRStats {
  window_hours: number
  /** Overall MTTA across all severities */
  mtta: MTTRMetric
  /** Overall MTTR across all severities */
  mttr: MTTRMetric
  /** Per-severity breakdown, ordered critical → warning → info */
  by_severity: SeverityMTTR[]

  // --- Legacy mirrors kept for backward compat with older UI builds ---
  /** @deprecated use mtta.mean */
  mtta_seconds: number
  /** @deprecated use mttr.mean */
  mttr_seconds: number
  /** @deprecated use mtta.count */
  acked_count: number
  /** @deprecated use mttr.count */
  resolved_count: number
}

/** One day of MTTA/MTTR means for the trend chart. */
export interface MTTRTrendPoint {
  date: string
  mtta_seconds: number
  mttr_seconds: number
  acked_count: number
  resolved_count: number
}

// ===== Audit Log =====
export interface AuditLog {
  id: number
  user_id: number | null
  username: string
  action: string
  resource_type: string
  resource_id: number | null
  resource_name: string
  detail: string
  ip: string
  status: 'success' | 'failed'
  created_at: string
}

// ===== Dashboard Analytics =====
export interface AlertTrendPoint {
  date: string
  fired_count: number
  resolved_count: number
}

export interface TopRuleItem {
  rule_id: number | null
  alert_name: string
  count: number
}

export interface SeverityHistoryPoint {
  date: string
  counts: Record<string, number>
}

export interface AlertGroupItem {
  alert_name: string
  source: string
  total_count: number
  severity_breakdown: Record<string, number>
  status_breakdown: Record<string, number>
  latest_fired_at: string
  oldest_fired_at: string
  max_fire_count: number
}

// ===== Expression Test =====
export interface QueryResponse {
  result_type: 'vector' | 'matrix' | 'logs'
  series: Array<{
    labels: Record<string, string>
    values: Array<{ ts: number; value: number }>
  }>
  raw_count: number
}

// ===== Log Query =====
export type { LogEntry, LogQueryResponse } from './query'

// ===== Inhibition Rules =====
export interface InhibitionRule {
  id: number
  name: string
  description: string
  source_match: Record<string, string>
  target_match: Record<string, string>
  equal_labels: string
  is_enabled: boolean
  created_by: number
  created_at: string
  updated_at: string
}

// ===== v2: Collaboration Channels (协作空间) =====
export type ChannelStatus = 'active' | 'disabled'
export type ChannelAccessLevel = 'public' | 'private'

export interface Channel {
  id: number
  name: string
  description: string
  team_id?: number
  team?: Team
  status: ChannelStatus
  access_level: ChannelAccessLevel
  aggregation_config?: string
  flapping_config?: string
  auto_close_enabled: boolean
  auto_close_origin: string
  auto_close_minutes: number
  follow_alert_close: boolean
  active_incident_count: number
  sort_order: number
  is_starred?: boolean
  mtta_label?: string
  mttr_label?: string
  created_at: string
  updated_at: string
}

export interface ChannelForm {
  name: string
  description?: string
  team_id?: number
  status?: ChannelStatus
  access_level?: ChannelAccessLevel
  auto_close_enabled?: boolean
  auto_close_minutes?: number
  follow_alert_close?: boolean
}

// ===== v2: Incidents (故障) =====
export type IncidentStatus = 'triggered' | 'processing' | 'closed'
export type IncidentSeverity = 'critical' | 'warning' | 'info'

export interface Incident {
  id: number
  title: string
  description: string
  severity: IncidentSeverity
  status: IncidentStatus
  channel_id: number
  channel?: Channel
  labels?: Record<string, string>
  assigned_to?: number
  assigned_user?: User
  triggered_at: string
  acknowledged_at?: string
  resolved_at?: string
  closed_at?: string
  snoozed_until?: string
  alert_count: number
  event_count: number
  is_recovered: boolean
  escalation_policy_id?: number
  current_escalation_step: number
  merged_into_id?: number
  created_at: string
  updated_at: string
}

export type IncidentTimelineAction =
  | 'triggered' | 'acknowledged' | 'unacknowledged'
  | 'snoozed' | 'snooze_expired' | 'escalated' | 'reassigned'
  | 'added_assignee' | 'resolved' | 'closed' | 'reopened'
  | 'merged' | 'commented' | 'notified' | 'alert_merged'
  | 'storm_warning' | 'severity_changed' | 'title_changed'

export interface IncidentTimeline {
  id: number
  incident_id: number
  action: IncidentTimelineAction
  actor_id?: number
  actor?: User
  content: string
  extra?: string
  created_at: string
}

// ===== v2: Alerts (告警 v2) =====
export type AlertV2Status = 'firing' | 'resolved'

export interface AlertV2 {
  id: number
  alert_key: string
  title: string
  severity: AlertSeverity
  status: AlertV2Status
  rule_id?: number
  labels?: Record<string, string>
  annotations?: Record<string, string>
  channel_id?: number
  channel?: Channel
  incident_id?: number
  incident?: Incident
  source: string
  generator_url: string
  first_fired_at: string
  last_fired_at: string
  resolved_at?: string
  event_count: number
  fire_count: number
  created_at: string
  updated_at: string
}

export interface AlertEventV2 {
  id: number
  alert_id: number
  event_status: 'firing' | 'resolved'
  event_severity: AlertSeverity
  labels?: Record<string, string>
  annotations?: Record<string, string>
  value: number
  timestamp: string
  fingerprint: string
  created_at: string
}

// ===== AI Chat =====
export interface ChatMessage {
  id?: number
  user_id?: number
  mode: 'alert' | 'general'
  role: 'user' | 'assistant'
  content: string
  context?: string
  created_at?: string
  _failed?: boolean
}

// ===== Routing Rules =====
export interface RoutingRule {
  id: number
  integration_id: number
  target_channel_id: number
  target_channel?: { id: number; name: string }
  conditions: string  // JSON []FilterCondition
  priority: number
  is_enabled: boolean
  created_at: string
  updated_at: string
}

// ===== Alert Rule Template =====
export interface AlertRuleTemplate {
  id: number
  name: string
  description: string
  category: string
  group_name: string
  severity: AlertSeverity
  rule_type: AlertRuleType
  expression: string
  for_duration: string
  labels: Record<string, string>
  annotations: Record<string, string>
  datasource_type: DataSourceType
  is_builtin: boolean
  created_at: string
  updated_at: string
}

// ===== Exclusion Rule =====
export interface ExclusionRule {
  id: number
  channel_id: number
  name: string
  description: string
  conditions: string
  priority: number
  is_enabled: boolean
  created_at: string
  updated_at: string
}

// ===== Integration =====
export interface Integration {
  id: number
  name: string
  description: string
  type: string
  mode: string
  channel_id: number | null
  channel?: Channel
  webhook_token: string
  pipeline_config: string
  label_enhancement_config: string
  is_enabled: boolean
  total_alerts: number
  created_at: string
  updated_at: string
}

// ===== Dispatch Policy =====
export interface DispatchPolicy {
  id: number
  channel_id: number
  name: string
  description: string
  is_enabled: boolean
  priority: number
  match_conditions: string
  active_time_config: string
  delay_seconds: number
  escalation_policy_id: number | null
  repeat_interval_seconds: number
  max_repeats: number
  notify_mode: string
  unified_media_id: number | null
  label_enhancement_rules: string
  created_at: string
  updated_at: string
}

// ===== Dispatch Log =====
export interface DispatchLog {
  id: number
  incident_id: number
  dispatch_policy_id: number
  status: 'pending' | 'sent' | 'skipped' | 'failed'
  attempt: number
  next_attempt_at: number | null
  note: string
  created_at: string
  updated_at: string
}

// ===== Post-Mortem =====
export interface PostMortem {
  id: number
  incident_id: number
  incident?: Incident
  title: string
  content: string
  status: 'draft' | 'published'
  author?: User
  published_at: string | null
  created_at: string
  updated_at: string
}

// ===== Dashboard V2 Stats =====
export interface IncidentStats {
  total: number
  triggered: number
  processing: number
  closed: number
  mtta_seconds: number
  mttr_seconds: number
}

export interface ChannelStatItem {
  channel_id: number
  channel_name: string
  total: number
  triggered: number
  critical: number
  closed: number
}

export interface TeamStatItem {
  team_id: number
  team_name: string
  total: number
  critical: number
  closed: number
  avg_mttr_seconds?: number
}

export interface IncidentTrendPoint {
  date: string
  triggered: number
  closed: number
}

// ===== User Preferences =====
export interface UserPreferences {
  id?: number
  user_id: number
  theme: 'auto' | 'light' | 'dark'
  language: string
  timezone: string
  default_time_range: string
  notification_severities: string
  ai_chat_mode: 'sidebar' | 'modal' | 'inline'
  created_at?: string
  updated_at?: string
}

// ===== Team Notify Channel =====
export interface TeamNotifyChannel {
  id: number
  team_id: number
  media_id: number
  media_name?: string
  media_type?: string
  is_default: boolean
  created_at?: string
  updated_at?: string
}

export interface UserTeamNotifyPref {
  id: number
  team_id: number
  media_id: number
  media_name?: string
  team_name?: string
  is_muted: boolean
  created_at?: string
  updated_at?: string
}

// ===== Preset Rules & AI Modules =====
export * from './preset-rule'
export * from './ai-module'

// ===== Naive UI Dropdown Option =====
import type { VNodeChild } from 'vue'

export interface DropdownOption {
  type?: 'default' | 'divider'
  label: string
  key: string
  icon?: () => VNodeChild
  props?: Record<string, unknown>
  disabled?: boolean
}
