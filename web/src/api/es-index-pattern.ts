import request from './request'
import type { ApiResponse } from '@/types'

export interface ESIndexPattern {
  id: number
  datasource_id: number
  name: string
  time_field: string
  allow_hide_system_indices: boolean
  fields_format: string
  cross_cluster_enabled: boolean
  note: string
  created_by: string
  updated_by: string
  created_at: string
  updated_at: string
}

export interface CreateESIndexPatternRequest {
  datasource_id: number
  name: string
  time_field?: string
  allow_hide_system_indices?: boolean
  fields_format?: string
  cross_cluster_enabled?: boolean
  note?: string
}

export type UpdateESIndexPatternRequest = CreateESIndexPatternRequest

export const esIndexPatternApi = {
  list: (params?: { datasource_id?: number }) =>
    request.get<ApiResponse<ESIndexPattern[]>>('/es-index-patterns', { params }),

  get: (id: number) =>
    request.get<ApiResponse<ESIndexPattern>>(`/es-index-patterns/${id}`),

  create: (data: CreateESIndexPatternRequest) =>
    request.post<ApiResponse<ESIndexPattern>>('/es-index-patterns', data),

  update: (id: number, data: UpdateESIndexPatternRequest) =>
    request.put<ApiResponse<null>>(`/es-index-patterns/${id}`, data),

  delete: (id: number) =>
    request.delete<ApiResponse<null>>(`/es-index-patterns/${id}`),
}
