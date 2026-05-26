import request from './request'
import type { ApiResponse } from '@/types'

export interface BuiltinDashboard {
  id: number
  name: string
  ident: string
  category: string
  component: string
  tags: string
  config: string
  version: number
  built_in: boolean
}

export const builtinDashboardApi = {
  list: (params?: { category?: string; component?: string; search?: string }) =>
    request.get<ApiResponse<BuiltinDashboard[]>>('/builtin-dashboards', { params }),

  get: (id: number) =>
    request.get<ApiResponse<BuiltinDashboard>>(`/builtin-dashboards/${id}`),

  getByIdent: (ident: string) =>
    request.get<ApiResponse<BuiltinDashboard>>(`/builtin-dashboards/ident/${ident}`),

  importDash: (ident: string) =>
    request.post<ApiResponse<{ id: number; name: string }>>(`/builtin-dashboards/${ident}/import`),

  categories: () =>
    request.get<ApiResponse<string[]>>('/builtin-dashboards/categories'),

  components: () =>
    request.get<ApiResponse<string[]>>('/builtin-dashboards/components'),
}
