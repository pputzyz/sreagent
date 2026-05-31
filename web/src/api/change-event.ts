import request from './request'
import type { ApiResponse, PageData } from '@/types'

export interface ChangeEvent {
  id: number
  source: string
  change_type: string
  service: string
  environment: string
  commit_sha: string
  author: string
  description: string
  risk_level: string
  metadata: Record<string, string>
  timestamp: string
  created_at: string
  updated_at: string
}

export interface IngestChangeEventRequest {
  source: string
  change_type?: string
  service: string
  environment: string
  commit_sha?: string
  author?: string
  description?: string
  risk_level?: string
  metadata?: Record<string, string>
  timestamp?: string
}

export const changeEventApi = {
  list: (params?: { page?: number; page_size?: number; service?: string; environment?: string; source?: string; incident_id?: number }) =>
    request.get<ApiResponse<PageData<ChangeEvent>>>('/change-events', { params }),
  get: (id: number) =>
    request.get<ApiResponse<ChangeEvent>>(`/change-events/${id}`),
  ingest: (data: IngestChangeEventRequest) =>
    request.post<ApiResponse<ChangeEvent>>('/change-events', data),
  delete: (id: number) =>
    request.delete<ApiResponse<null>>(`/change-events/${id}`),
}
