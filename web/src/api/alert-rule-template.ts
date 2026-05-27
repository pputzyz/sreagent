import request from './request'
import type { ApiResponse, PageData } from '@/types'

// ===== Backend API types =====
export interface AlertRuleTemplate {
  id: number
  category: string
  name: string
  description: string
  datasource_type: string
  expression: string
  for_duration: string
  severity: string
  labels: Record<string, string>
  annotations: Record<string, string>
  group_name: string
  eval_interval: number
  is_builtin: boolean
  usage_count: number
  created_by: number
  updated_by: number
  no_data_enabled: boolean
  no_data_duration: string
  ack_sla_minutes: number
  created_at: string
  updated_at: string
}

export interface CreateAlertRuleTemplateRequest {
  category?: string
  name: string
  description?: string
  datasource_type: string
  expression: string
  for_duration?: string
  severity: string
  labels?: Record<string, string>
  annotations?: Record<string, string>
  group_name?: string
  eval_interval?: number
  no_data_enabled?: boolean
  no_data_duration?: string
  ack_sla_minutes?: number
}

export type UpdateAlertRuleTemplateRequest = Partial<CreateAlertRuleTemplateRequest>

// ===== API =====
export const alertRuleTemplateApi = {
  list: (params?: { page?: number; page_size?: number; category?: string; search?: string }) =>
    request.get<ApiResponse<PageData<AlertRuleTemplate>>>('/alert-rule-templates', { params }),

  listCategories: () =>
    request.get<ApiResponse<string[]>>('/alert-rule-templates/categories'),

  get: (id: number) =>
    request.get<ApiResponse<AlertRuleTemplate>>(`/alert-rule-templates/${id}`),

  create: (data: CreateAlertRuleTemplateRequest) =>
    request.post<ApiResponse<AlertRuleTemplate>>('/alert-rule-templates', data),

  update: (id: number, data: UpdateAlertRuleTemplateRequest) =>
    request.put<ApiResponse<AlertRuleTemplate>>(`/alert-rule-templates/${id}`, data),

  delete: (id: number) =>
    request.delete<ApiResponse<null>>(`/alert-rule-templates/${id}`),

  apply: (id: number, overrides?: Record<string, unknown>) =>
    request.post<ApiResponse<unknown>>(`/alert-rule-templates/${id}/apply`, overrides),
}
