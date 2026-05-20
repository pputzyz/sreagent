import request from './request'
import type { ApiResponse, PageData } from '@/types'

// ===== Notification Center =====
export interface UserNotification {
  id: number
  user_id: number
  title: string
  content: string
  type: 'alert' | 'incident' | 'system'
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
