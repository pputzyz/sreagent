import request from './request'
import type { ApiResponse, PageData } from '@/types'

export interface LLMExtraConfig {
  timeout_seconds?: number
  skip_tls_verify?: boolean
  proxy?: string
  custom_headers?: Record<string, string>
  temperature?: number
  max_tokens?: number
  context_length?: number
}

export interface LLMConfig {
  id: number
  name: string
  provider: string // openai | azure | ollama | anthropic | custom
  api_url: string
  api_key: string
  model: string
  extra_config: string // JSON string of LLMExtraConfig
  enabled: boolean
  is_default: boolean
  description: string
  created_at: string
  updated_at: string
}

export interface CreateLLMConfigRequest {
  name: string
  provider: string
  api_url?: string
  api_key?: string
  model?: string
  extra_config?: string
  enabled?: boolean
  is_default?: boolean
  description?: string
}

export type UpdateLLMConfigRequest = CreateLLMConfigRequest

export interface TestConnectionResponse {
  success: boolean
  message: string
  latency_ms?: number
}

export const llmConfigApi = {
  list: (params?: { page?: number; page_size?: number }) =>
    request.get<ApiResponse<PageData<LLMConfig>>>('/llm-configs', { params }),

  get: (id: number) =>
    request.get<ApiResponse<LLMConfig>>(`/llm-configs/${id}`),

  create: (data: CreateLLMConfigRequest) =>
    request.post<ApiResponse<LLMConfig>>('/llm-configs', data),

  update: (id: number, data: UpdateLLMConfigRequest) =>
    request.put<ApiResponse<LLMConfig>>(`/llm-configs/${id}`, data),

  delete: (id: number) =>
    request.delete<ApiResponse<null>>(`/llm-configs/${id}`),

  testConnection: (id: number) =>
    request.post<ApiResponse<TestConnectionResponse>>(`/llm-configs/${id}/test`),
}
