import request from './request'
import type {
  ApiResponse,
  PageData,
  Channel,
  ChannelForm,
  Incident,
  IncidentTimeline,
  AlertV2,
  AlertEventV2,
  RoutingRule,
  Integration,
  DispatchPolicy,
  DispatchLog,
  ExclusionRule,
  PostMortem,
  IncidentStats,
  ChannelStatItem,
  TeamStatItem,
  IncidentTrendPoint,
} from '@/types'

// ===== v2: Collaboration Channels API =====
export const channelV2Api = {
  list: (params?: { query?: string; status?: string; page?: number; page_size?: number }) =>
    request.get<ApiResponse<PageData<Channel>>>('/channels', { params }),

  get: (id: number) =>
    request.get<ApiResponse<Channel>>(`/channels/${id}`),

  create: (data: ChannelForm) =>
    request.post<ApiResponse<Channel>>('/channels', data),

  update: (id: number, data: Partial<ChannelForm>) =>
    request.put<ApiResponse<Channel>>(`/channels/${id}`, data),

  delete: (id: number) =>
    request.delete<ApiResponse<null>>(`/channels/${id}`),

  star: (id: number) =>
    request.post<ApiResponse<null>>(`/channels/${id}/star`),

  unstar: (id: number) =>
    request.delete<ApiResponse<null>>(`/channels/${id}/star`),

  updateNoiseConfig: (id: number, data: { aggregation_config?: string; flapping_config?: string }) =>
    request.put<ApiResponse<Channel>>(`/channels/${id}`, data),

  listExclusionRules: (channelId: number) =>
    request.get<ApiResponse<ExclusionRule[]>>(`/channels/${channelId}/exclusion-rules`),

  createExclusionRule: (channelId: number, data: { name: string; conditions: string; is_enabled: boolean; priority?: number; description?: string }) =>
    request.post<ApiResponse<ExclusionRule>>(`/channels/${channelId}/exclusion-rules`, data),

  updateExclusionRule: (ruleId: number, data: { name?: string; conditions?: string; is_enabled?: boolean; priority?: number }) =>
    request.put<ApiResponse<ExclusionRule>>(`/exclusion-rules/${ruleId}`, data),

  deleteExclusionRule: (ruleId: number) =>
    request.delete<ApiResponse<null>>(`/exclusion-rules/${ruleId}`),
}

// ===== v2: Incidents API =====
export const incidentApi = {
  list: (params?: {
    channel_id?: number
    status?: string
    severity?: string
    query?: string
    assigned_to?: number
    page?: number
    page_size?: number
  }) => request.get<ApiResponse<PageData<Incident>>>('/incidents', { params }),

  get: (id: number) =>
    request.get<ApiResponse<Incident>>(`/incidents/${id}`),

  create: (data: { title: string; description?: string; severity?: string; channel_id: number; assigned_to?: number }) =>
    request.post<ApiResponse<Incident>>('/incidents', data),

  acknowledge: (id: number) =>
    request.post<ApiResponse<null>>(`/incidents/${id}/acknowledge`),

  close: (id: number) =>
    request.post<ApiResponse<null>>(`/incidents/${id}/close`),

  // FE1-13: Bulk operations
  bulkAcknowledge: (ids: number[]) =>
    request.post<ApiResponse<null>>('/incidents/bulk-acknowledge', { ids }),

  bulkClose: (ids: number[]) =>
    request.post<ApiResponse<null>>('/incidents/bulk-close', { ids }),

  reopen: (id: number) =>
    request.post<ApiResponse<null>>(`/incidents/${id}/reopen`),

  snooze: (id: number, until: string) =>
    request.post<ApiResponse<null>>(`/incidents/${id}/snooze`, { until }),

  reassign: (id: number, userId: number) =>
    request.post<ApiResponse<null>>(`/incidents/${id}/reassign`, { user_id: userId }),

  merge: (id: number, targetId: number) =>
    request.post<ApiResponse<null>>(`/incidents/${id}/merge`, { target_id: targetId }),

  escalate: (id: number) =>
    request.post<ApiResponse<null>>(`/incidents/${id}/escalate`),

  addComment: (id: number, content: string) =>
    request.post<ApiResponse<null>>(`/incidents/${id}/comment`, { content }),

  getTimeline: (id: number) =>
    request.get<ApiResponse<IncidentTimeline[]>>(`/incidents/${id}/timeline`),

  getDispatchLogs: (id: number) =>
    request.get<ApiResponse<DispatchLog[]>>(`/incidents/${id}/dispatch-logs`),

  // Post-mortem (复盘)
  getPostMortem: (incidentId: number) =>
    request.get<ApiResponse<PostMortem>>(`/incidents/${incidentId}/post-mortem`),

  updatePostMortem: (incidentId: number, data: { title?: string; content?: string; status?: string }) =>
    request.put<ApiResponse<PostMortem>>(`/incidents/${incidentId}/post-mortem`, data),

  publishPostMortem: (incidentId: number) =>
    request.post<ApiResponse<PostMortem>>(`/incidents/${incidentId}/post-mortem/publish`),

  aiGeneratePostMortem: (incidentId: number) =>
    request.post<ApiResponse<PostMortem>>(`/incidents/${incidentId}/post-mortem/ai-generate`),

  aiSummaryPostMortem: (incidentId: number) =>
    request.post<ApiResponse<PostMortem>>(`/incidents/${incidentId}/post-mortem/ai-summary`),
}

// ===== Post-Mortem List API =====
export const postMortemApi = {
  list: (params?: { channel_id?: number; status?: string; page?: number; page_size?: number }) =>
    request.get<ApiResponse<PageData<PostMortem>>>('/post-mortems', { params }),
}

// ===== v2: Integrations API =====
export const integrationV2Api = {
  list: (params?: { channel_id?: number; page?: number; page_size?: number }) =>
    request.get<ApiResponse<PageData<Integration>>>('/integrations', { params }),

  get: (id: number) =>
    request.get<ApiResponse<Integration>>(`/integrations/${id}`),

  create: (data: {
    name: string
    description?: string
    type: string
    mode?: string
    channel_id?: number
    pipeline_config?: string
    label_enhancement_config?: string
    is_enabled?: boolean
  }) => request.post<ApiResponse<Integration>>('/integrations', data),

  update: (id: number, data: Partial<Integration>) =>
    request.put<ApiResponse<Integration>>(`/integrations/${id}`, data),

  delete: (id: number) =>
    request.delete<ApiResponse<null>>(`/integrations/${id}`),
}

// ===== v2: Routing Rules API =====
export const routingRuleApi = {
  listByIntegration: (integrationId: number) =>
    request.get<ApiResponse<RoutingRule[]>>('/routing-rules', { params: { integration_id: integrationId } }),

  create: (integrationId: number, data: {
    target_channel_id: number
    conditions?: string
    priority?: number
    is_enabled?: boolean
  }) => request.post<ApiResponse<RoutingRule>>('/routing-rules', { ...data, integration_id: integrationId }),

  update: (id: number, data: {
    target_channel_id?: number
    conditions?: string
    priority?: number
    is_enabled?: boolean
  }) => request.put<ApiResponse<RoutingRule>>(`/routing-rules/${id}`, data),

  delete: (id: number) =>
    request.delete<ApiResponse<null>>(`/routing-rules/${id}`),
}

// ===== v2: Dispatch Policies API =====
export const dispatchApi = {
  list: (channelId: number) =>
    request.get<ApiResponse<DispatchPolicy[]>>(`/channels/${channelId}/dispatch-policies`),

  get: (id: number) =>
    request.get<ApiResponse<DispatchPolicy>>(`/dispatch-policies/${id}`),

  create: (channelId: number, data: {
    name: string
    description?: string
    is_enabled?: boolean
    priority?: number
    match_conditions?: string
    active_time_config?: string
    delay_seconds?: number
    escalation_policy_id?: number
    repeat_interval_seconds?: number
    max_repeats?: number
    notify_mode?: string
    unified_media_id?: number
    label_enhancement_rules?: string
  }) => request.post<ApiResponse<DispatchPolicy>>(`/channels/${channelId}/dispatch-policies`, data),

  update: (id: number, data: Partial<DispatchPolicy>) =>
    request.put<ApiResponse<DispatchPolicy>>(`/dispatch-policies/${id}`, data),

  delete: (id: number) =>
    request.delete<ApiResponse<null>>(`/dispatch-policies/${id}`),
}

// ===== v2: Alerts API =====
export const alertV2Api = {
  list: (params?: {
    channel_id?: number
    incident_id?: number
    status?: string
    severity?: string
    query?: string
    page?: number
    page_size?: number
  }) => request.get<ApiResponse<PageData<AlertV2>>>('/alerts', { params }),

  get: (id: number) =>
    request.get<ApiResponse<AlertV2>>(`/alerts/${id}`),

  listEvents: (id: number, params?: { page?: number; page_size?: number }) =>
    request.get<ApiResponse<PageData<AlertEventV2>>>(`/alerts/${id}/events`, { params }),
}

// ===== v2 Dashboard Stats API =====
export const dashboardV2StatsApi = {
  incidentStats: () =>
    request.get<ApiResponse<IncidentStats>>('/dashboard/incident-stats'),

  channelStats: (days = 30) =>
    request.get<ApiResponse<ChannelStatItem[]>>('/dashboard/channel-stats', { params: { days } }),

  teamStats: (days = 30) =>
    request.get<ApiResponse<TeamStatItem[]>>('/dashboard/team-stats', { params: { days } }),

  incidentTrend: (days = 30) =>
    request.get<ApiResponse<IncidentTrendPoint[]>>('/dashboard/incident-trend', { params: { days } }),
}
