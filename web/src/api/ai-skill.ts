import request from './request'
import type { ApiResponse } from '@/types'

export interface AISkillFile {
  id: number
  skill_id: number
  name: string
  content: string
  size: number
  created_at: string
  updated_at: string
}

export interface AISkill {
  id: number
  name: string
  description: string
  instructions: string
  license: string
  compatibility: string
  allowed_tools: string
  metadata: string
  enabled: boolean
  created_by: string
  updated_by: string
  created_at: string
  updated_at: string
  files?: AISkillFile[]
  builtin?: boolean
}

export interface CreateAISkillRequest {
  name: string
  description?: string
  instructions?: string
  license?: string
  compatibility?: string
  allowed_tools?: string
  metadata?: Record<string, string>
  enabled?: boolean
}

export type UpdateAISkillRequest = CreateAISkillRequest

export interface CreateAISkillFileRequest {
  name: string
  content?: string
}

export const aiSkillApi = {
  list: (params?: { search?: string }) =>
    request.get<ApiResponse<AISkill[]>>('/ai-skills', { params }),

  get: (id: number) =>
    request.get<ApiResponse<AISkill>>(`/ai-skills/${id}`),

  create: (data: CreateAISkillRequest) =>
    request.post<ApiResponse<AISkill>>('/ai-skills', data),

  update: (id: number, data: UpdateAISkillRequest) =>
    request.put<ApiResponse<AISkill>>(`/ai-skills/${id}`, data),

  delete: (id: number) =>
    request.delete<ApiResponse<null>>(`/ai-skills/${id}`),

  import: (file: File) => {
    const formData = new FormData()
    formData.append('file', file)
    return request.post<ApiResponse<AISkill>>('/ai-skills/import', formData, {
      headers: { 'Content-Type': 'multipart/form-data' },
    })
  },

  getFiles: (skillId: number) =>
    request.get<ApiResponse<AISkillFile[]>>(`/ai-skills/${skillId}/files`),

  getFile: (fileId: number) =>
    request.get<ApiResponse<AISkillFile>>(`/ai-skills/files/${fileId}`),

  addFile: (skillId: number, data: CreateAISkillFileRequest) =>
    request.post<ApiResponse<AISkillFile>>(`/ai-skills/${skillId}/files`, data),

  deleteFile: (fileId: number) =>
    request.delete<ApiResponse<null>>(`/ai-skills/files/${fileId}`),
}
