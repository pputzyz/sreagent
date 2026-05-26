import request from './request'
import type {
  ApiResponse,
  PageData,
  User,
  AuditLog,
  BizGroup,
  ChatMessage,
  AIProvidersConfig,
  UserPreferences,
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
    request.get<ApiResponse<{ provider: string; api_key: string; base_url: string; model: string; enabled: boolean; temperature: number; max_tokens: number; system_prompt: string; retry_max: number; context_max_chars: number }>>('/ai/config'),

  updateConfig: (data: { provider?: string; api_key?: string; base_url?: string; model?: string; enabled?: boolean; temperature?: number; max_tokens?: number; system_prompt?: string; retry_max?: number; context_max_chars?: number }) =>
    request.put<ApiResponse<null>>('/ai/config', data),

  testConnection: () =>
    request.post<ApiResponse<{ success: boolean; message: string }>>('/ai/test'),

  generateReport: (eventId: number) =>
    request.post<ApiResponse<{ report: string; event_id: number }>>('/ai/alert-report', { event_id: eventId }),

  suggestSOP: (eventId: number) =>
    request.post<ApiResponse<{ sop: string; event_id: number }>>('/ai/suggest-sop', { event_id: eventId }),

  analyzeAlert: (eventId: number) =>
    request.post<ApiResponse<{ summary: string; severity: string; probable_causes: string[]; impact: string; recommended_steps: string[]; root_cause_hint: string }>>('/ai/analyze-alert', { event_id: eventId }),

  getProviders: () =>
    request.get<ApiResponse<AIProvidersConfig>>('/ai/providers'),

  saveProviders: (config: AIProvidersConfig) =>
    request.put<ApiResponse<null>>('/ai/providers', config),

  testProvider: (key: string) =>
    request.post<ApiResponse<{ message: string }>>('/ai/test-provider', { key }),

  getGlobal: () =>
    request.get<ApiResponse<{ retry_max: number; context_max_chars: number; default_temperature: number; default_max_tokens: number; monthly_token_budget: number; data_masking_enabled: boolean }>>('/ai/global'),

  saveGlobal: (data: { retry_max?: number; context_max_chars?: number; default_temperature?: number; default_max_tokens?: number; monthly_token_budget?: number; data_masking_enabled?: boolean }) =>
    request.put<ApiResponse<null>>('/ai/global', data),
}

// ===== AI Chat API =====
export const aiChatApi = {
  send: (data: { mode: 'alert' | 'general'; message: string; context?: string }) =>
    request.post<ApiResponse<{ reply: string }>>('/ai/chat', data),

  getHistory: (mode: 'alert' | 'general') =>
    request.get<ApiResponse<ChatMessage[]>>('/ai/history', { params: { mode } }),

  clearHistory: (mode: 'alert' | 'general') =>
    request.delete<ApiResponse<null>>('/ai/history', { params: { mode } }),
}

// ===== AI Agent API =====
export interface AgentStep {
  index: number
  description: string
  tool: string
  parameters: Record<string, unknown>
  result: string
  status: 'pending' | 'running' | 'completed' | 'failed'
  duration_ms: number
}

export interface AgentTask {
  id: string
  query: string
  status: 'planning' | 'executing' | 'completed' | 'failed'
  steps: AgentStep[]
  result: string
  error?: string
  created_at: string
  completed_at?: string
}

export const aiAgentApi = {
  run: (query: string) =>
    request.post<ApiResponse<AgentTask>>('/ai/agent/run', { query }),

  getTask: (id: string) =>
    request.get<ApiResponse<AgentTask>>(`/ai/agent/tasks/${id}`),

  listConversations: (page = 1, pageSize = 1) =>
    request.get<ApiResponse<{ list: unknown[]; total: number }>>('/ai/agent/conversations', { params: { page, page_size: pageSize } }),
}

// ===== Auth API =====
export const authApi = {
  login: (data: { username: string; password: string; captcha_id?: string; captcha?: string }) =>
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

  getOAuth2Config: () =>
    request.get<ApiResponse<{ enabled: boolean; name?: string; login_url?: string }>>('/auth/oauth2/config'),

  getCaptcha: () =>
    request.get<ApiResponse<{ captcha_id: string; image: string }>>('/auth/captcha'),

  getPreferences: () =>
    request.get<ApiResponse<UserPreferences>>('/me/preferences'),

  updatePreferences: (data: Partial<UserPreferences>) =>
    request.put<ApiResponse<UserPreferences>>('/me/preferences', data),
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
    request.get<ApiResponse<{ app_id: string; app_secret: string; default_webhook: string; verification_token: string; encrypt_key: string; bot_enabled: boolean; resolve_strategy: string; update_on_state_change: boolean; delete_only_in_business_hours: boolean; business_hours_start: string; business_hours_end: string; commands_enabled: boolean; natural_language_enabled: boolean; debug_mode: boolean }>>('/lark/bot/config'),

  updateConfig: (data: { app_id?: string; app_secret?: string; default_webhook?: string; verification_token?: string; encrypt_key?: string; bot_enabled?: boolean; resolve_strategy?: string; update_on_state_change?: boolean; delete_only_in_business_hours?: boolean; business_hours_start?: string; business_hours_end?: string; commands_enabled?: boolean; natural_language_enabled?: boolean; debug_mode?: boolean }) =>
    request.put<ApiResponse<null>>('/lark/bot/config', data),

  testBotAPI: () =>
    request.post<ApiResponse<{ message: string }>>('/lark/bot/test'),

  getBotStatus: () =>
    request.get<ApiResponse<{ configured: boolean; app_id: string; webhook_set: boolean; commands_enabled: boolean; natural_language_enabled: boolean; debug_mode: boolean }>>('/lark/bot/status'),
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
      username_claim?: string
      email_claim?: string
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

// ===== LDAP Settings API =====
export const ldapSettingsApi = {
  getConfig: () =>
    request.get<ApiResponse<{
      enabled: boolean; host: string; port: number; base_dn: string
      bind_dn: string; bind_password: string; user_filter: string
      user_attr: string; email_attr: string; display_name_attr: string
      start_tls: boolean; skip_verify: boolean; default_role: string; auto_provision: boolean
    }>>('/settings/ldap'),

  updateConfig: (data: {
    enabled?: boolean; host?: string; port?: number; base_dn?: string
    bind_dn?: string; bind_password?: string; user_filter?: string
    user_attr?: string; email_attr?: string; display_name_attr?: string
    start_tls?: boolean; skip_verify?: boolean; default_role?: string; auto_provision?: boolean
  }) => request.put<ApiResponse<{ message: string }>>('/settings/ldap', data),

  testConnection: () =>
    request.post<ApiResponse<{ success: boolean; message: string }>>('/settings/ldap/test'),
}

// ===== OAuth2 Settings API =====
export const oauth2SettingsApi = {
  getConfig: () =>
    request.get<ApiResponse<{
      enabled: boolean; name: string; client_id: string; client_secret: string
      auth_url: string; token_url: string; user_info_url: string
      redirect_url: string; scopes: string; user_id_field: string
      email_field: string; username_field: string; default_role: string; auto_provision: boolean
    }>>('/settings/oauth2'),

  updateConfig: (data: {
    enabled?: boolean; name?: string; client_id?: string; client_secret?: string
    auth_url?: string; token_url?: string; user_info_url?: string
    redirect_url?: string; scopes?: string; user_id_field?: string
    email_field?: string; username_field?: string; default_role?: string; auto_provision?: boolean
  }) => request.put<ApiResponse<{ message: string }>>('/settings/oauth2', data),
}

// ===== SMTP Settings API =====
export const smtpSettingsApi = {
  getConfig: () =>
    request.get<ApiResponse<{
      smtp_host: string; smtp_port: number; smtp_tls: boolean
      username: string; password: string; from: string; from_name?: string; enabled: boolean
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

// ===== Site Info Settings API =====
export interface SiteInfo {
  site_name: string
  logo_url: string
  favicon_url: string
  login_title: string
  login_subtitle: string
  footer_text: string
  custom_css: string
}

export const siteInfoApi = {
  get: () =>
    request.get<ApiResponse<SiteInfo>>('/settings/site-info'),

  save: (data: SiteInfo) =>
    request.put<ApiResponse<null>>('/settings/site-info', data),
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

export const statusSubscriptionApi = {
  subscribe: (email: string) =>
    request.post<ApiResponse<{ message: string }>>('/status-subscriptions', { email }),

  unsubscribe: (email: string) =>
    request.delete<ApiResponse<{ message: string }>>(`/status-subscriptions?email=${encodeURIComponent(email)}`),

  list: () =>
    request.get<ApiResponse<Array<{ id: number; email: string; is_active: boolean; created_at: string }>>>('/status-subscriptions'),
}
