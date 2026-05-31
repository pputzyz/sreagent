import request from './request'
import type { ApiResponse, PageData } from '@/types'

export interface MetricViewFilter {
  label: string
  oper: '=' | '!=' | '=~' | '!~'
  value: string
}

export interface MetricViewDynamicLabel {
  label: string
  value: string
}

export interface MetricViewConfig {
  filters: MetricViewFilter[]
  dynamicLabels: MetricViewDynamicLabel[]
  dimensionLabels: string[][]
  ignorePrefix: string
}

export interface MetricView {
  id: number
  name: string
  configs: string
  configs_json: MetricViewConfig
  is_favorite: boolean
  created_by: number
  updated_by: number
  created_at: string
  updated_at: string
}

export interface CreateMetricViewRequest {
  name: string
  configs: MetricViewConfig
}

export type UpdateMetricViewRequest = CreateMetricViewRequest

export const metricViewApi = {
  list: (params?: { page?: number; page_size?: number }) =>
    request.get<ApiResponse<PageData<MetricView>>>('/metric-views', { params }),

  get: (id: number) =>
    request.get<ApiResponse<MetricView>>(`/metric-views/${id}`),

  create: (data: CreateMetricViewRequest) =>
    request.post<ApiResponse<MetricView>>('/metric-views', data),

  update: (id: number, data: UpdateMetricViewRequest) =>
    request.put<ApiResponse<MetricView>>(`/metric-views/${id}`, data),

  delete: (id: number) =>
    request.delete<ApiResponse<null>>(`/metric-views/${id}`),

  favorite: (id: number) =>
    request.post<ApiResponse<null>>(`/metric-views/${id}/favorite`),

  unfavorite: (id: number) =>
    request.delete<ApiResponse<null>>(`/metric-views/${id}/favorite`),
}
