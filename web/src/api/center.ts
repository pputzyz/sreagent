import request from './request'
import type { ApiResponse, PageData } from '@/types'

// ===== Notification Center =====
export interface UserNotification {
  id: number
  user_id: number
  title: string
  content: string
  type: 'alert' | 'incident' | 'system' | 'todo'
  is_read: boolean
  link: string
  metadata?: Record<string, string>
  created_at: string
  updated_at: string
}

export const notificationCenterApi = {
  list: (params?: { page?: number; page_size?: number; is_read?: boolean }) =>
    request.get<ApiResponse<PageData<UserNotification>>>('/notifications', { params }),

  unreadCount: () =>
    request.get<ApiResponse<{ count: number }>>('/notifications/unread-count'),

  markRead: (id: number) =>
    request.patch<ApiResponse<null>>(`/notifications/${id}/read`),

  markAllRead: () =>
    request.post<ApiResponse<null>>('/notifications/read-all'),

  delete: (id: number) =>
    request.delete<ApiResponse<null>>(`/notifications/${id}`),
}

// ===== Todo / Task Center =====
export interface TodoItem {
  id: number
  user_id: number
  title: string
  description: string
  type: string
  status: 'pending' | 'completed' | 'dismissed'
  priority: 'high' | 'medium' | 'low'
  link: string
  due_at?: string
  completed_at?: string
  created_at: string
  updated_at: string
}

export interface CreateTodoRequest {
  title: string
  description?: string
  type?: string
  priority?: 'high' | 'medium' | 'low'
  link?: string
  due_at?: string
}

export const todoApi = {
  list: (params?: { page?: number; page_size?: number; status?: string }) =>
    request.get<ApiResponse<PageData<TodoItem>>>('/todos', { params }),

  pendingCount: () =>
    request.get<ApiResponse<{ count: number }>>('/todos/pending-count'),

  create: (data: CreateTodoRequest) =>
    request.post<ApiResponse<TodoItem>>('/todos', data),

  update: (id: number, data: CreateTodoRequest) =>
    request.put<ApiResponse<TodoItem>>(`/todos/${id}`, data),

  complete: (id: number) =>
    request.patch<ApiResponse<null>>(`/todos/${id}/complete`),

  delete: (id: number) =>
    request.delete<ApiResponse<null>>(`/todos/${id}`),
}

// ===== RBAC Permissions =====
export interface TeamRole {
  team_id: number
  role: string
}

export interface MyPermissions {
  role: string
  perms: string[]
  teams: TeamRole[]
}

export const permissionsApi = {
  getMy: () =>
    request.get<ApiResponse<MyPermissions>>('/me/permissions'),
}
