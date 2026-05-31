import request from './request'
import type {
  ApiResponse,
  PageData,
  AlertRule,
  AlertEvent,
  AlertEventFilter,
  AlertTimeline,
  AlertChannel,
  AlertGroupItem,
  QueryResponse,
  LogEntry,
} from '@/types'

// ===== Alert Rule API =====
export const alertRuleApi = {
  list: (params?: { page?: number; page_size?: number; severity?: string; status?: string; group_name?: string; category?: string }) =>
    request.get<ApiResponse<PageData<AlertRule>>>('/alert-rules', { params }),

  get: (id: number) =>
    request.get<ApiResponse<AlertRule>>(`/alert-rules/${id}`),

  create: (data: Partial<AlertRule>) =>
    request.post<ApiResponse<AlertRule>>('/alert-rules', data),

  update: (id: number, data: Partial<AlertRule>) =>
    request.put<ApiResponse<AlertRule>>(`/alert-rules/${id}`, data),

  delete: (id: number) =>
    request.delete<ApiResponse<null>>(`/alert-rules/${id}`),

  toggleStatus: (id: number, status: string) =>
    request.patch<ApiResponse<null>>(`/alert-rules/${id}/status`, { status }),

  listCategories: () =>
    request.get<ApiResponse<string[]>>('/alert-rules/categories'),

  exportRules: (params?: { format?: string; category?: string; group_name?: string }) =>
    request.get('/alert-rules/export', { params, responseType: 'blob' }),

  importRules: (file: File, datasourceId?: number) => {
    const formData = new FormData()
    formData.append('file', file)
    if (datasourceId) formData.append('datasource_id', String(datasourceId))
    return request.post<ApiResponse<{ total: number; success: number; failed: number; errors: string[] }>>('/alert-rules/import', formData, {
      headers: { 'Content-Type': 'multipart/form-data' },
    })
  },

  batchEnable: (ids: number[]) =>
    request.post<ApiResponse<null>>('/alert-rules/batch/enable', { ids }),

  batchDisable: (ids: number[]) =>
    request.post<ApiResponse<null>>('/alert-rules/batch/disable', { ids }),

  batchDelete: (ids: number[]) =>
    request.post<ApiResponse<null>>('/alert-rules/batch/delete', { ids }),

  labelValidationPreview: (limit = 10) =>
    request.get<ApiResponse<{ total: number; passing: number; failing: number; samples: Array<{ rule_id: number; rule_name: string; pass: boolean; issues?: string[] }> }>>('/alert-rules/label-validation-preview', { params: { limit } }),
}

// ===== Alert Event API =====
export const alertEventApi = {
  list: (params?: AlertEventFilter) =>
    request.get<ApiResponse<PageData<AlertEvent>>>('/alert-events', { params }),

  get: (id: number) =>
    request.get<ApiResponse<AlertEvent>>(`/alert-events/${id}`),

  acknowledge: (id: number) =>
    request.post<ApiResponse<null>>(`/alert-events/${id}/acknowledge`),

  assign: (id: number, data: { assign_to: number; note?: string }) =>
    request.post<ApiResponse<null>>(`/alert-events/${id}/assign`, data),

  resolve: (id: number, data?: { resolution?: string }) =>
    request.post<ApiResponse<null>>(`/alert-events/${id}/resolve`, data),

  close: (id: number, data?: { note?: string }) =>
    request.post<ApiResponse<null>>(`/alert-events/${id}/close`, data),

  silence: (id: number, data: { duration_minutes: number; reason: string }) =>
    request.post<ApiResponse<null>>(`/alert-events/${id}/silence`, data),

  comment: (id: number, data: { note: string }) =>
    request.post<ApiResponse<null>>(`/alert-events/${id}/comment`, data),

  getTimeline: (id: number) =>
    request.get<ApiResponse<AlertTimeline[]>>(`/alert-events/${id}/timeline`),

  batchAcknowledge: (ids: number[]) =>
    request.post<ApiResponse<null>>('/alert-events/batch/acknowledge', { ids }),

  batchClose: (ids: number[]) =>
    request.post<ApiResponse<null>>('/alert-events/batch/close', { ids }),
}

// ===== Alert Channel API =====
export const alertChannelApi = {
  list: (params?: { page?: number; page_size?: number }) =>
    request.get<ApiResponse<PageData<AlertChannel>>>('/alert-channels', { params }),

  get: (id: number) =>
    request.get<ApiResponse<AlertChannel>>(`/alert-channels/${id}`),

  create: (data: Partial<AlertChannel>) =>
    request.post<ApiResponse<AlertChannel>>('/alert-channels', data),

  update: (id: number, data: Partial<AlertChannel>) =>
    request.put<ApiResponse<AlertChannel>>(`/alert-channels/${id}`, data),

  delete: (id: number) =>
    request.delete<ApiResponse<null>>(`/alert-channels/${id}`),

  test: (id: number) =>
    request.post<ApiResponse<{ success: boolean; message: string }>>(`/alert-channels/${id}/test`),
}

// ===== Alert Groups API =====
export const alertGroupsApi = {
  list: (params?: { status?: string; severity?: string }) =>
    request.get<ApiResponse<AlertGroupItem[]>>('/alert-events/groups', { params }),
}

// ===== Alert Export API =====
export const alertExportApi = {
  exportCSV: (params?: {
    status?: string; severity?: string; view_mode?: string
    start?: string; end?: string
  }) => {
    const query = new URLSearchParams()
    if (params?.status) query.set('status', params.status)
    if (params?.severity) query.set('severity', params.severity)
    if (params?.view_mode) query.set('view_mode', params.view_mode)
    if (params?.start) query.set('start', params.start)
    if (params?.end) query.set('end', params.end)
    return `/api/v1/alert-events/export?${query.toString()}`
  },
}

// ===== Engine API =====
export const engineApi = {
  getStatus: () =>
    request.get<ApiResponse<{ running: boolean; total_rules: number; active_alerts: number; uptime: string; is_leader: boolean }>>('/engine/status'),
}

// ===== Expression Test / Query =====
export const expressionApi = {
  query: (datasourceId: number, data: { expression: string; time?: number }) =>
    request.post<ApiResponse<QueryResponse>>(`/datasources/${datasourceId}/query`, data),

  rangeQuery: (datasourceId: number, data: { expression: string; start: number; end: number; step: string }) =>
    request.post<ApiResponse<QueryResponse>>(`/datasources/${datasourceId}/query-range`, data),

  logQuery: (datasourceId: number, data: { expression: string; start: number; end: number; limit?: number }) =>
    request.post<ApiResponse<{ entries: LogEntry[]; total: number; truncated: boolean }>>(`/datasources/${datasourceId}/log-query`, data),
}
