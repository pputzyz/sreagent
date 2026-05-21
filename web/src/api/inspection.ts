import request from './request'
import type { ApiResponse, PageData } from '@/types'

export interface InspectionTask {
  id: number
  name: string
  description: string
  cron_expr: string
  target_type: string
  target_ids: string
  allowed_tools: string
  output_channels: string
  enabled: boolean
  created_by: number
  created_at: string
  updated_at: string
}

export interface InspectionRun {
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

export interface InspectionFinding {
  severity: string
  category: string
  object: string
  detail: string
}

export const inspectionApi = {
  // Tasks
  listTasks: (params?: { enabled?: boolean }) =>
    request.get<ApiResponse<{ list: InspectionTask[]; total: number }>>('/inspection/tasks', { params }),

  getTask: (id: number) =>
    request.get<ApiResponse<InspectionTask>>(`/inspection/tasks/${id}`),

  createTask: (data: Partial<InspectionTask>) =>
    request.post<ApiResponse<InspectionTask>>('/inspection/tasks', data),

  updateTask: (id: number, data: Partial<InspectionTask>) =>
    request.put<ApiResponse<InspectionTask>>(`/inspection/tasks/${id}`, data),

  deleteTask: (id: number) =>
    request.delete<ApiResponse<null>>(`/inspection/tasks/${id}`),

  runNow: (id: number) =>
    request.post<ApiResponse<{ message: string; task_id: number }>>(`/inspection/tasks/${id}/run`),

  // Runs
  listRuns: (params?: { task_id?: number; page?: number; page_size?: number }) =>
    request.get<ApiResponse<PageData<InspectionRun>>>('/inspection/runs', { params }),

  getRun: (id: number) =>
    request.get<ApiResponse<InspectionRun>>(`/inspection/runs/${id}`),

  // Utils
  validateCron: (cronExpr: string) =>
    request.post<ApiResponse<{ valid: boolean; next_runs: string[] }>>('/inspection/validate-cron', { cron_expr: cronExpr }),
}
