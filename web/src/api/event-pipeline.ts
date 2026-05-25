import request from './request'
import type { ApiResponse, PageData } from '@/types'

export interface TagFilter {
  key: string
  func: string // ==, =~, in, !=, !~, not in
  value: any
}

export interface ProcessorConfig {
  typ: string
  config: Record<string, any>
}

export interface EventPipeline {
  id: number
  name: string
  description: string
  disabled: boolean
  filter_enable: boolean
  label_filters: TagFilter[]
  processors: ProcessorConfig[]
  created_by: number
  updated_by: number
  created_at: string
  updated_at: string
}

export interface EventPipelineExecution {
  id: string
  pipeline_id: number
  pipeline_name: string
  event_id: number
  mode: string
  status: string
  node_results: string // JSON string of NodeResult[]
  error_message: string
  duration_ms: number
  trigger_by: string
  created_at: string
  finished_at: string
}

export interface NodeResult {
  processor_type: string
  status: string
  message: string
  duration_ms: number
}

export const eventPipelineApi = {
  list: (params?: { page?: number; page_size?: number; query?: string; disabled?: string }) =>
    request.get<ApiResponse<PageData<EventPipeline>>>('/event-pipelines', { params }),

  get: (id: number) =>
    request.get<ApiResponse<EventPipeline>>(`/event-pipelines/${id}`),

  create: (data: Partial<EventPipeline>) =>
    request.post<ApiResponse<EventPipeline>>('/event-pipelines', data),

  update: (id: number, data: Partial<EventPipeline>) =>
    request.put<ApiResponse<null>>(`/event-pipelines/${id}`, data),

  delete: (id: number) =>
    request.delete<ApiResponse<null>>(`/event-pipelines/${id}`),

  listExecutions: (id: number, params?: { page?: number; page_size?: number }) =>
    request.get<ApiResponse<PageData<EventPipelineExecution>>>(`/event-pipelines/${id}/executions`, { params }),

  getExecution: (id: string) =>
    request.get<ApiResponse<EventPipelineExecution>>(`/event-pipeline-executions/${id}`),

  tryRun: (id: number) =>
    request.post<ApiResponse<any>>(`/event-pipelines/${id}/tryrun`),

  cleanExecutions: (days?: number) =>
    request.post<ApiResponse<null>>('/event-pipeline-executions/clean', null, { params: { days } }),

  listProcessorTypes: () =>
    request.get<ApiResponse<string[]>>('/event-pipelines/processor-types'),
}
