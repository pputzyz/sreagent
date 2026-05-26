import request from './request'
import type { ApiResponse, PageData } from '@/types'

// ===== Backend API types =====
export interface SavedViewApiItem {
  id: number
  name: string
  description: string
  tab: 'metrics' | 'logs'
  datasource_id: number
  expression: string
  query_config: string
  is_public: boolean
  created_by: number
  updated_by: number
  created_at: string
  updated_at: string
}

export interface SavedViewCreate {
  name: string
  description?: string
  tab: 'metrics' | 'logs'
  datasource_id: number
  expression: string
  query_config?: string
  is_public?: boolean
}

export type SavedViewUpdate = SavedViewCreate

// ===== API =====
export const savedViewApi = {
  list: (params?: { tab?: string; page?: number; page_size?: number }) =>
    request.get<ApiResponse<PageData<SavedViewApiItem>>>('/saved-views', { params }),

  get: (id: number) =>
    request.get<ApiResponse<SavedViewApiItem>>(`/saved-views/${id}`),

  create: (data: SavedViewCreate) =>
    request.post<ApiResponse<SavedViewApiItem>>('/saved-views', data),

  update: (id: number, data: SavedViewUpdate) =>
    request.put<ApiResponse<SavedViewApiItem>>(`/saved-views/${id}`, data),

  delete: (id: number) =>
    request.delete<ApiResponse<null>>(`/saved-views/${id}`),

  copy: (id: number) =>
    request.post<ApiResponse<SavedViewApiItem>>(`/saved-views/${id}/copy`),
}
