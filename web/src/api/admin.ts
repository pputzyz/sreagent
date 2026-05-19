import request from './request'
import type {
  ApiResponse,
  PageData,
  User,
  AuditLog,
  BizGroup,
  ChatMessage,
  Pet,
  PetInteraction,
  AIProvidersConfig,
} from '@/types'

// ===== User API =====
export const userApi = {
  list: (params?: { page?: number; page_size?: number; role?: string; is_active?: boolean }) =>
    request.get<ApiResponse<PageData<User>>>('/users', { params }),

  get: (id: number) =>
    request.get<ApiResponse<User>>(`/users/${id}`),

  create: (data: Partial<User> & { password?: string }) =>
    request.post<ApiResponse<User>>('/users', data),

  update: (id: number, data: Partial<User>) =>
    request.put<ApiResponse<User>>(`/users/${id}`, data),

  toggleActive: (id: number, is_active: boolean) =>
    request.patch<ApiResponse<null>>(`/users/${id}/active`, { is_active }),

  changePassword: (id: number, data: { password: string }) =>
    request.patch<ApiResponse<null>>(`/users/${id}/password`, data),

  createVirtual: (data: { username: string; display_name: string; user_type: 'bot' | 'channel'; notify_target?: string }) =>
    request.post<ApiResponse<User>>('/users/virtual', data),

  delete: (id: number) =>
    request.delete<ApiResponse<null>>(`/users/${id}`),
}

// ===== Business Group API =====
export const bizGroupApi = {
  list: (params?: { page?: number; page_size?: number }) =>
    request.get<ApiResponse<PageData<BizGroup>>>('/biz-groups', { params }),

  tree: () =>
    request.get<ApiResponse<BizGroup[]>>('/biz-groups/tree'),

  get: (id: number) =>
    request.get<ApiResponse<BizGroup>>(`/biz-groups/${id}`),

  create: (data: Partial<BizGroup>) =>
    request.post<ApiResponse<BizGroup>>('/biz-groups', data),

  update: (id: number, data: Partial<BizGroup>) =>
    request.put<ApiResponse<BizGroup>>(`/biz-groups/${id}`, data),

  delete: (id: number) =>
    request.delete<ApiResponse<null>>(`/biz-groups/${id}`),

  addMember: (id: number, data: { user_id: number; role?: string }) =>
    request.post<ApiResponse<null>>(`/biz-groups/${id}/members`, data),

  removeMember: (id: number, uid: number) =>
    request.delete<ApiResponse<null>>(`/biz-groups/${id}/members/${uid}`),

  listMembers: (id: number) =>
    request.get<ApiResponse<User[]>>(`/biz-groups/${id}/members`),
}

// ===== AI API =====
export const aiApi = {
  getConfig: () =>
    request.get<ApiResponse<{ provider: string; api_key: string; base_url: string; model: string; enabled: boolean }>>('/ai/config'),

  updateConfig: (data: { provider?: string; api_key?: string; base_url?: string; model?: string; enabled?: boolean }) =>
    request.put<ApiResponse<null>>('/ai/config', data),

  testConnection: () =>
    request.post<ApiResponse<{ success: boolean; message: string }>>('/ai/test'),

  generateReport: (eventId: number) =>
    request.post<ApiResponse<{ summary: string; probable_causes: string[]; impact: string; recommended_steps: string[] }>>('/ai/alert-report', { event_id: eventId }),

  suggestSOP: (eventId: number) =>
    request.post<ApiResponse<{ title: string; steps: string[]; references: string[] }>>('/ai/suggest-sop', { event_id: eventId }),

  getProviders: () =>
    request.get<ApiResponse<AIProvidersConfig>>('/ai/providers'),

  saveProviders: (config: AIProvidersConfig) =>
    request.put<ApiResponse<null>>('/ai/providers', config),

  testProvider: (key: string) =>
    request.post<ApiResponse<{ message: string }>>('/ai/test-provider', { key }),
}

// ===== AI Chat API =====
export const aiChatApi = {
  send: (data: { mode: 'alert' | 'general' | 'pet'; message: string; context?: string }) =>
    request.post<ApiResponse<{ reply: string }>>('/ai/chat', data),

  getHistory: (mode: 'alert' | 'general' | 'pet') =>
    request.get<ApiResponse<ChatMessage[]>>('/ai/history', { params: { mode } }),

  clearHistory: (mode: 'alert' | 'general' | 'pet') =>
    request.delete<ApiResponse<null>>('/ai/history', { params: { mode } }),
}

// ===== Pet API =====
export const petApi = {
  get: () =>
    request.get<ApiResponse<Pet>>('/pet'),

  update: (data: { name?: string }) =>
    request.put<ApiResponse<Pet>>('/pet', data),

  feed: () =>
    request.post<ApiResponse<Pet>>('/pet/feed'),

  play: () =>
    request.post<ApiResponse<Pet>>('/pet/play'),

  getInteractions: () =>
    request.get<ApiResponse<PetInteraction[]>>('/pet/interactions'),
}

// ===== Auth API =====
export const authApi = {
  login: (data: { username: string; password: string }) =>
    request.post<ApiResponse<{ token: string; expires_in: number }>>('/auth/login', data),

  getProfile: () =>
    request.get<ApiResponse<User>>('/auth/profile'),

  updateMe: (data: { display_name?: string; email?: string; phone?: string; avatar?: string }) =>
    request.put<ApiResponse<null>>('/me/profile', data),

  changeMyPassword: (data: { old_password: string; new_password: string }) =>
    request.post<ApiResponse<null>>('/me/password', data),

  refreshToken: (token: string) =>
    request.post<ApiResponse<{ token: string; expires_in: number }>>('/auth/refresh', { token }),

  bindLark: (larkOpenId: string) =>
    request.put<ApiResponse<null>>('/me/lark-bind', { lark_open_id: larkOpenId }),

  getOIDCConfig: () =>
    request.get<ApiResponse<{ enabled: boolean; login_url?: string }>>('/auth/oidc/config'),
}

// ===== Audit Log API =====
export const auditLogApi = {
  list: (params?: {
    page?: number; page_size?: number;
    action?: string; resource_type?: string;
    start_time?: string; end_time?: string;
  }) => request.get<ApiResponse<PageData<AuditLog>>>('/audit-logs', { params }),
}

// ===== User Notify Config API =====
export const userNotifyConfigApi = {
  list: () => request.get<ApiResponse<{ id: number; user_id: number; media_type: string; config: string; is_enabled: boolean }[]>>('/me/notify-configs'),
  upsert: (data: { media_type?: string; config?: string; is_enabled?: boolean }) => request.put<ApiResponse<{ id: number; user_id: number; media_type: string; config: string; is_enabled: boolean }>>('/me/notify-configs', data),
  deleteByType: (mediaType: string) => request.delete<ApiResponse<null>>(`/me/notify-configs/${mediaType}`),
}

// ===== Lark Bot API =====
export const larkBotApi = {
  getConfig: () =>
    request.get<ApiResponse<{ app_id: string; app_secret: string; default_webhook: string; verification_token: string; encrypt_key: string; bot_enabled: boolean }>>('/lark/bot/config'),

  updateConfig: (data: { app_id?: string; app_secret?: string; default_webhook?: string; verification_token?: string; encrypt_key?: string; bot_enabled?: boolean }) =>
    request.put<ApiResponse<null>>('/lark/bot/config', data),
}

// ===== OIDC Settings API =====
export const oidcSettingsApi = {
  getConfig: () =>
    request.get<ApiResponse<{
      enabled: boolean
      issuer_url: string
      client_id: string
      client_secret: string
      redirect_url: string
      scopes: string
      role_claim: string
      role_mapping: string
      default_role: string
      auto_provision: boolean
    }>>('/settings/oidc'),

  updateConfig: (data: {
    enabled?: boolean
    issuer_url?: string
    client_id?: string
    client_secret?: string
    redirect_url?: string
    scopes?: string
    role_claim?: string
    role_mapping?: string
    default_role?: string
    auto_provision?: boolean
  }) =>
    request.put<ApiResponse<{ message: string }>>('/settings/oidc', data),
}

// ===== SMTP Settings API =====
export const smtpSettingsApi = {
  getConfig: () =>
    request.get<ApiResponse<{
      smtp_host: string; smtp_port: number; smtp_tls: boolean
      username: string; password: string; from: string; enabled: boolean
    }>>('/settings/smtp'),

  updateConfig: (data: {
    smtp_host?: string; smtp_port?: number; smtp_tls?: boolean
    username?: string; password?: string; from?: string; enabled?: boolean
  }) => request.put<ApiResponse<null>>('/settings/smtp', data),

  testConnection: (to: string) =>
    request.post<ApiResponse<{ message: string }>>('/settings/smtp/test', { to }),
}

// ===== Security Settings API =====
export const securitySettingsApi = {
  getConfig: () =>
    request.get<ApiResponse<{ jwt_expire_seconds: number }>>('/settings/security'),

  updateConfig: (data: { jwt_expire_seconds: number }) =>
    request.put<ApiResponse<null>>('/settings/security', data),
}

// ===== Status Service API =====
export interface StatusServiceItem {
  id: number
  name: string
  status: 'operational' | 'degraded' | 'outage' | 'maintenance'
  description: string
  url: string
  icon: string
  sort_order: number
  created_at: string
  updated_at: string
}

export const statusServiceApi = {
  list: () =>
    request.get<ApiResponse<StatusServiceItem[]>>('/status-services'),

  get: (id: number) =>
    request.get<ApiResponse<StatusServiceItem>>(`/status-services/${id}`),

  create: (data: { name: string; status: string; description?: string; url?: string; icon?: string; sort_order?: number }) =>
    request.post<ApiResponse<StatusServiceItem>>('/status-services', data),

  update: (id: number, data: { name?: string; status?: string; description?: string; url?: string; icon?: string; sort_order?: number }) =>
    request.put<ApiResponse<StatusServiceItem>>(`/status-services/${id}`, data),

  delete: (id: number) =>
    request.delete<ApiResponse<null>>(`/status-services/${id}`),
}
