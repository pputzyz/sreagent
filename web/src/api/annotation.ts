import request from './request'
import type { ApiResponse, PageData } from '@/types'

export interface Annotation {
  id: number
  dashboard_id: number
  dashboard_name?: string
  time: string // ISO datetime
  text: string
  tags?: Record<string, string>
  source?: string
  created_by: number
  created_at: string
  updated_at: string
}

export interface CreateAnnotationRequest {
  dashboard_id: number
  time: string
  text: string
}

export type UpdateAnnotationRequest = CreateAnnotationRequest

export const annotationApi = {
  list: (params?: { page?: number; page_size?: number; dashboard_id?: number }) =>
    request.get<ApiResponse<PageData<Annotation>>>('/annotations', { params }),

  get: (id: number) =>
    request.get<ApiResponse<Annotation>>(`/annotations/${id}`),

  create: (data: CreateAnnotationRequest) =>
    request.post<ApiResponse<Annotation>>('/annotations', data),

  update: (id: number, data: UpdateAnnotationRequest) =>
    request.put<ApiResponse<null>>(`/annotations/${id}`, data),

  delete: (id: number) =>
    request.delete<ApiResponse<null>>(`/annotations/${id}`),
}
