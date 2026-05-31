import request from './request'
import type { ApiResponse, PageData } from '@/types'

export interface BuiltinMetric {
  id: number
  collector: string
  typ: string
  name: string
  unit: string
  note: string
  lang: string
  expression: string
  expression_type: string
  metric_type: string
  extra_fields: Record<string, string>
  translation: TranslationEntry[]
  created_at: string
  updated_at: string
  created_by: string
  updated_by: string
}

export interface TranslationEntry {
  lang: string
  name: string
  note: string
}

export interface MetricFilter {
  id: number
  name: string
  configs: FilterConfig[]
  groups_perm: GroupPerm[]
  created_at: string
  created_by: string
}

export interface FilterConfig {
  label: string
  operator: string
  value: string
}

export interface GroupPerm {
  gid: number
  write: boolean
}

export const builtinMetricApi = {
  list: (params?: {
    page?: number
    page_size?: number
    collector?: string
    typ?: string
    query?: string
    unit?: string
  }) => request.get<ApiResponse<PageData<BuiltinMetric>>>('/builtin-metrics', { params }),

  get: (id: number) =>
    request.get<ApiResponse<BuiltinMetric>>(`/builtin-metrics/${id}`),

  create: (data: Partial<BuiltinMetric>) =>
    request.post<ApiResponse<BuiltinMetric>>('/builtin-metrics', data),

  update: (id: number, data: Partial<BuiltinMetric>) =>
    request.put<ApiResponse<null>>(`/builtin-metrics/${id}`, data),

  delete: (ids: number[]) =>
    request.post<ApiResponse<null>>('/builtin-metrics/delete', { ids }),

  batchCreate: (metrics: Partial<BuiltinMetric>[]) =>
    request.post<ApiResponse<Record<string, string>>>('/builtin-metrics/batch', { metrics }),

  getTypes: (params?: { collector?: string; query?: string }) =>
    request.get<ApiResponse<string[]>>('/builtin-metrics/types', { params }),

  getCollectors: (params?: { typ?: string; query?: string }) =>
    request.get<ApiResponse<string[]>>('/builtin-metrics/collectors', { params }),
}

export const metricFilterApi = {
  list: () =>
    request.get<ApiResponse<MetricFilter[]>>('/builtin-metric-filters'),

  create: (data: Partial<MetricFilter>) =>
    request.post<ApiResponse<MetricFilter>>('/builtin-metric-filters', data),

  update: (data: Partial<MetricFilter>) =>
    request.put<ApiResponse<null>>('/builtin-metric-filters', data),

  delete: (ids: number[]) =>
    request.post<ApiResponse<null>>('/builtin-metric-filters/delete', { ids }),
}
