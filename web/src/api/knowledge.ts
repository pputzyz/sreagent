import request from './request'
import type { ApiResponse, PageData } from '@/types'

export interface KnowledgeDocument {
  id: number
  title: string
  content: string
  source: string // sop | incident_case | runbook | template | wiki
  tags: string[]
  helpful_count: number
  created_by: number
  updated_by: number
  created_at: string
  updated_at: string
}

export interface CreateKnowledgeRequest {
  title: string
  content: string
  source: string
  tags?: string[]
}

export type UpdateKnowledgeRequest = CreateKnowledgeRequest

export const knowledgeApi = {
  list: (params?: { page?: number; page_size?: number; source?: string; search?: string }) =>
    request.get<ApiResponse<PageData<KnowledgeDocument>>>('/knowledge', { params }),

  get: (id: number) =>
    request.get<ApiResponse<KnowledgeDocument>>(`/knowledge/${id}`),

  create: (data: CreateKnowledgeRequest) =>
    request.post<ApiResponse<KnowledgeDocument>>('/knowledge', data),

  update: (id: number, data: UpdateKnowledgeRequest) =>
    request.put<ApiResponse<KnowledgeDocument>>(`/knowledge/${id}`, data),

  delete: (id: number) =>
    request.delete<ApiResponse<null>>(`/knowledge/${id}`),

  markHelpful: (id: number) =>
    request.post<ApiResponse<null>>(`/knowledge/${id}/helpful`),
}
