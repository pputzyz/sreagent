import request from './request'
import type { ApiResponse, PageData } from '@/types'

export interface ReportTask {
  id: number
  name: string
  description: string
  cron_expr: string
  report_type: string
  scope: string
  prompt_template: string
  allowed_tools: string
  output_channels: string
  enabled: boolean
  created_by: number
  created_at: string
  updated_at: string
}

export interface ReportRun {
  id: number
  task_id: number
  status: string
  started_at: string
  finished_at?: string
  report_markdown?: string
  report_summary?: string
  findings_json?: string
  error_msg?: string
  ai_conversation_id?: number
}

export const reportApi = {
  listTasks: (params?: { enabled?: boolean }) =>
    request.get<ApiResponse<{ list: ReportTask[]; total: number }>>('/report-tasks', { params }),

  getTask: (id: number) =>
    request.get<ApiResponse<ReportTask>>(`/report-tasks/${id}`),

  createTask: (data: Partial<ReportTask>) =>
    request.post<ApiResponse<ReportTask>>('/report-tasks', data),

  updateTask: (id: number, data: Partial<ReportTask>) =>
    request.put<ApiResponse<ReportTask>>(`/report-tasks/${id}`, data),

  deleteTask: (id: number) =>
    request.delete<ApiResponse<null>>(`/report-tasks/${id}`),

  runNow: (id: number) =>
    request.post<ApiResponse<{ message: string; task_id: number }>>(`/report-tasks/${id}/run`),

  listRuns: (params?: { task_id?: number; page?: number; page_size?: number }) =>
    request.get<ApiResponse<PageData<ReportRun>>>('/report-runs', { params }),

  getRun: (id: number) =>
    request.get<ApiResponse<ReportRun>>(`/report-runs/${id}`),
}
