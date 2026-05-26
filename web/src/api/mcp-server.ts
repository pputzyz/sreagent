import request from './request'
import type { ApiResponse, PageData } from '@/types'

export interface MCPServer {
  id: number
  name: string
  url: string
  headers: string
  description: string
  enabled: boolean
  created_at: string
  updated_at: string
}

export interface MCPTool {
  name: string
  description: string
  input_schema: Record<string, any>
}

export interface CreateMCPServerRequest {
  name: string
  url: string
  headers?: Record<string, string>
  description?: string
  enabled?: boolean
}

export type UpdateMCPServerRequest = CreateMCPServerRequest

export const mcpServerApi = {
  list: (params?: { page?: number; page_size?: number }) =>
    request.get<ApiResponse<PageData<MCPServer>>>('/mcp-servers', { params }),

  get: (id: number) =>
    request.get<ApiResponse<MCPServer>>(`/mcp-servers/${id}`),

  create: (data: CreateMCPServerRequest) =>
    request.post<ApiResponse<MCPServer>>('/mcp-servers', data),

  update: (id: number, data: UpdateMCPServerRequest) =>
    request.put<ApiResponse<null>>(`/mcp-servers/${id}`, data),

  delete: (id: number) =>
    request.delete<ApiResponse<null>>(`/mcp-servers/${id}`),

  testConnection: (id: number) =>
    request.post<ApiResponse<{ message: string }>>(`/mcp-servers/${id}/test`),

  listTools: (id: number) =>
    request.get<ApiResponse<MCPTool[]>>(`/mcp-servers/${id}/tools`),
}
