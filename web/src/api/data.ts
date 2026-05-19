import request from './request'
import type {
  ApiResponse,
  PageData,
  DataSource,
  DashboardStats,
  MTTRStats,
  MTTRTrendPoint,
  AlertTrendPoint,
  TopRuleItem,
  SeverityHistoryPoint,
  QueryResponse,
  LogEntry,
} from '@/types'

// ===== DataSource API =====
export const datasourceApi = {
  list: (params?: { page?: number; page_size?: number; type?: string }) =>
    request.get<ApiResponse<PageData<DataSource>>>('/datasources', { params }),

  get: (id: number) =>
    request.get<ApiResponse<DataSource>>(`/datasources/${id}`),

  create: (data: Partial<DataSource>) =>
    request.post<ApiResponse<DataSource>>('/datasources', data),

  update: (id: number, data: Partial<DataSource>) =>
    request.put<ApiResponse<DataSource>>(`/datasources/${id}`, data),

  delete: (id: number) =>
    request.delete<ApiResponse<null>>(`/datasources/${id}`),

  healthCheck: (id: number) =>
    request.post<ApiResponse<{ status: string; message: string; latency_ms: number; version: string }>>(`/datasources/${id}/health-check`),

  query: (id: number, data: { expression: string; time?: number }) =>
    request.post<ApiResponse<QueryResponse>>(`/datasources/${id}/query`, data),

  rangeQuery: (id: number, data: { expression: string; start: number; end: number; step: string }) =>
    request.post<ApiResponse<QueryResponse>>(`/datasources/${id}/query-range`, data),

  labelKeys: (id: number) =>
    request.get<ApiResponse<string[]>>(`/datasources/${id}/labels/keys`),

  labelValues: (id: number, key: string) =>
    request.get<ApiResponse<string[]>>(`/datasources/${id}/labels/values`, { params: { key } }),

  metricNames: (id: number, search?: string, limit = 100) =>
    request.get<ApiResponse<string[]>>(`/datasources/${id}/metrics`, { params: { search, limit } }),

  logQuery: (id: number, data: { expression: string; start: number; end: number; limit?: number }) =>
    request.post<ApiResponse<{ entries: LogEntry[]; total: number; truncated: boolean }>>(`/datasources/${id}/log-query`, data),
}

// ===== Dashboard API =====
export const dashboardApi = {
  getStats: () =>
    request.get<ApiResponse<DashboardStats>>('/dashboard/stats'),
  getMTTRStats: (hours = 24) =>
    request.get<ApiResponse<MTTRStats>>('/dashboard/mtta-mttr', { params: { hours } }),
  getMTTRTrend: (days = 30) =>
    request.get<ApiResponse<MTTRTrendPoint[]>>('/dashboard/mttr-trend', { params: { days } }),
  getAlertTrend: (days = 30) =>
    request.get<ApiResponse<AlertTrendPoint[]>>('/dashboard/alert-trend', { params: { days } }),
  getTopRules: (days = 30, limit = 10) =>
    request.get<ApiResponse<TopRuleItem[]>>('/dashboard/top-rules', { params: { days, limit } }),
  getSeverityHistory: (days = 30) =>
    request.get<ApiResponse<SeverityHistoryPoint[]>>('/dashboard/severity-history', { params: { days } }),
  exportReportURL: (startDate: string, endDate: string) =>
    `/api/v1/dashboard/export?start_date=${startDate}&end_date=${endDate}`,
}

// ===== Dashboard V2 API =====
export const dashboardV2Api = {
  list: (params?: { page?: number; page_size?: number; search?: string }) =>
    request.get<ApiResponse<PageData<import('@/types/dashboard').DashboardV2>>>('/dashboards', { params }),

  get: (id: number) =>
    request.get<ApiResponse<import('@/types/dashboard').DashboardV2>>(`/dashboards/${id}`),

  create: (data: Partial<import('@/types/dashboard').DashboardV2>) =>
    request.post<ApiResponse<import('@/types/dashboard').DashboardV2>>('/dashboards', data),

  update: (id: number, data: Partial<import('@/types/dashboard').DashboardV2>) =>
    request.put<ApiResponse<import('@/types/dashboard').DashboardV2>>(`/dashboards/${id}`, data),

  delete: (id: number) =>
    request.delete<ApiResponse<null>>(`/dashboards/${id}`),
}

// ===== Label Registry API =====
export const labelRegistryApi = {
  getKeys: (datasourceId?: number) =>
    request.get<ApiResponse<string[]>>('/label-registry/keys', {
      params: datasourceId ? { datasource_id: datasourceId } : {}
    }),

  getValues: (key: string, datasourceId?: number) =>
    request.get<ApiResponse<string[]>>('/label-registry/values', {
      params: { key, ...(datasourceId ? { datasource_id: datasourceId } : {}) }
    }),

  sync: () =>
    request.post<ApiResponse<{ message: string }>>('/label-registry/sync'),
}
