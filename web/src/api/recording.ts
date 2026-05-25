import request from './request'
import type { ApiResponse, PageData } from '@/types'

export interface RecordingRule {
  id: number
  group_id: number
  name: string
  prom_ql: string
  datasource_ids: number[]
  cron_pattern: string
  disabled: number
  append_tags: string[]
  note: string
  query_configs: QueryConfig[]
  created_at: string
  updated_at: string
  created_by: string
  updated_by: string
}

export interface QueryConfig {
  queries: Query[]
  new_metric: string
  exp: string
  write_datasource_id: number
  delay: number
  writeback_enabled: boolean
}

export interface Query {
  datasource_ids: number[]
  cate: string
  config: any
}

export interface CreateRecordingRuleRequest {
  group_id: number
  name: string
  prom_ql: string
  datasource_ids?: number[]
  cron_pattern?: string
  disabled?: number
  append_tags?: string[]
  note?: string
  query_configs?: QueryConfig[]
}

export interface UpdateRecordingRuleRequest {
  name: string
  prom_ql: string
  datasource_ids?: number[]
  cron_pattern?: string
  disabled?: number
  append_tags?: string[]
  note?: string
  query_configs?: QueryConfig[]
}

export const recordingRuleApi = {
  list: (params?: {
    page?: number
    page_size?: number
    group_id?: number
    query?: string
    disabled?: number
  }) => request.get<ApiResponse<PageData<RecordingRule>>>('/recording-rules', { params }),

  get: (id: number) =>
    request.get<ApiResponse<RecordingRule>>(`/recording-rules/${id}`),

  create: (data: CreateRecordingRuleRequest) =>
    request.post<ApiResponse<RecordingRule>>('/recording-rules', data),

  update: (id: number, data: UpdateRecordingRuleRequest) =>
    request.put<ApiResponse<null>>(`/recording-rules/${id}`, data),

  delete: (id: number) =>
    request.delete<ApiResponse<null>>(`/recording-rules/${id}`),

  batchCreate: (group_id: number, rules: CreateRecordingRuleRequest[]) =>
    request.post<ApiResponse<Record<string, string>>>('/recording-rules/batch', { group_id, rules }),

  batchDelete: (group_id: number, ids: number[]) =>
    request.post<ApiResponse<null>>('/recording-rules/batch-delete', { group_id, ids }),

  updateFields: (ids: number[], fields: Record<string, any>) =>
    request.put<ApiResponse<null>>('/recording-rules/fields', { ids, fields }),
}
