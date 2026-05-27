import request from './request'
import type { ApiResponse, PageData } from '@/types'

export interface DiagnosticWorkflow {
  id: number
  name: string
  description: string
  trigger_labels: Record<string, string>
  trigger_severity: string
  category: string
  enabled: boolean
  max_steps: number
  require_approval: boolean
  created_by: number | null
  created_at: string
  updated_at: string
}

export interface DiagnosticWorkflowStep {
  id?: number
  workflow_id?: number
  step_order: number
  name: string
  step_type: string
  datasource_id: number | null
  expression: string
  condition_expr: string
  auto_advance: boolean
  timeout_seconds: number
  on_failure: string
}

export interface DiagnosticRun {
  id: number
  workflow_id: number
  incident_id: number | null
  user_id: number | null
  status: string
  current_step: number
  result_summary: string
  started_at: string | null
  completed_at: string | null
  created_at: string
}

export interface DiagnosticRunStep {
  id: number
  run_id: number
  step_order: number
  step_name: string
  step_type: string
  expression: string
  result: string
  status: string
  duration_ms: number
  error: string
  started_at: string | null
  completed_at: string | null
}

export const diagnosticApi = {
  listWorkflows: (params?: { page?: number; page_size?: number; category?: string; enabled?: boolean }) =>
    request.get<ApiResponse<PageData<DiagnosticWorkflow>>>('/diagnostic-workflows', { params }),
  getWorkflow: (id: number) =>
    request.get<ApiResponse<{ workflow: DiagnosticWorkflow; steps: DiagnosticWorkflowStep[] }>>(`/diagnostic-workflows/${id}`),
  createWorkflow: (data: { workflow: Partial<DiagnosticWorkflow>; steps: DiagnosticWorkflowStep[] }) =>
    request.post<ApiResponse<DiagnosticWorkflow>>('/diagnostic-workflows', data),
  updateWorkflow: (id: number, data: Partial<DiagnosticWorkflow>) =>
    request.put<ApiResponse<DiagnosticWorkflow>>(`/diagnostic-workflows/${id}`, data),
  deleteWorkflow: (id: number) =>
    request.delete<ApiResponse<null>>(`/diagnostic-workflows/${id}`),
  replaceSteps: (id: number, steps: DiagnosticWorkflowStep[]) =>
    request.put<ApiResponse<null>>(`/diagnostic-workflows/${id}/steps`, steps),
  startRun: (id: number, incidentId?: number) =>
    request.post<ApiResponse<DiagnosticRun>>(`/diagnostic-workflows/${id}/run`, { incident_id: incidentId }),
  listRuns: (params?: { page?: number; page_size?: number; workflow_id?: number; status?: string }) =>
    request.get<ApiResponse<PageData<DiagnosticRun>>>('/diagnostic-runs', { params }),
  getRun: (id: number) =>
    request.get<ApiResponse<{ run: DiagnosticRun; steps: DiagnosticRunStep[] }>>(`/diagnostic-runs/${id}`),
}
