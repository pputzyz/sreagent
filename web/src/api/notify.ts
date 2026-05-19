import request from './request'
import type {
  ApiResponse,
  PageData,
  NotifyRule,
  NotifyMedia,
  MessageTemplate,
  SubscribeRule,
  InhibitionRule,
  MuteRule,
  AlertEvent,
} from '@/types'

// ===== Notify Rule API (v2) =====
export const notifyRuleApi = {
  list: (params?: { page?: number; page_size?: number }) =>
    request.get<ApiResponse<PageData<NotifyRule>>>('/notify-rules', { params }),

  get: (id: number) =>
    request.get<ApiResponse<NotifyRule>>(`/notify-rules/${id}`),

  create: (data: Partial<NotifyRule>) =>
    request.post<ApiResponse<NotifyRule>>('/notify-rules', data),

  update: (id: number, data: Partial<NotifyRule>) =>
    request.put<ApiResponse<NotifyRule>>(`/notify-rules/${id}`, data),

  delete: (id: number) =>
    request.delete<ApiResponse<null>>(`/notify-rules/${id}`),
}

// ===== Notify Media API =====
export const notifyMediaApi = {
  list: (params?: { page?: number; page_size?: number; type?: string }) =>
    request.get<ApiResponse<PageData<NotifyMedia>>>('/notify-media', { params }),

  get: (id: number) =>
    request.get<ApiResponse<NotifyMedia>>(`/notify-media/${id}`),

  create: (data: Partial<NotifyMedia>) =>
    request.post<ApiResponse<NotifyMedia>>('/notify-media', data),

  update: (id: number, data: Partial<NotifyMedia>) =>
    request.put<ApiResponse<NotifyMedia>>(`/notify-media/${id}`, data),

  delete: (id: number) =>
    request.delete<ApiResponse<null>>(`/notify-media/${id}`),

  test: (id: number) =>
    request.post<ApiResponse<{ success: boolean; message: string }>>(`/notify-media/${id}/test`),
}

// ===== Message Template API =====
export const messageTemplateApi = {
  list: (params?: { page?: number; page_size?: number }) =>
    request.get<ApiResponse<PageData<MessageTemplate>>>('/message-templates', { params }),

  get: (id: number) =>
    request.get<ApiResponse<MessageTemplate>>(`/message-templates/${id}`),

  create: (data: Partial<MessageTemplate>) =>
    request.post<ApiResponse<MessageTemplate>>('/message-templates', data),

  update: (id: number, data: Partial<MessageTemplate>) =>
    request.put<ApiResponse<MessageTemplate>>(`/message-templates/${id}`, data),

  delete: (id: number) =>
    request.delete<ApiResponse<null>>(`/message-templates/${id}`),

  preview: (data: { content: string; type: string }) =>
    request.post<ApiResponse<{ rendered: string }>>('/message-templates/preview', data),
}

// ===== Subscribe Rule API =====
export const subscribeRuleApi = {
  list: (params?: { page?: number; page_size?: number }) =>
    request.get<ApiResponse<PageData<SubscribeRule>>>('/subscribe-rules', { params }),

  get: (id: number) =>
    request.get<ApiResponse<SubscribeRule>>(`/subscribe-rules/${id}`),

  create: (data: Partial<SubscribeRule>) =>
    request.post<ApiResponse<SubscribeRule>>('/subscribe-rules', data),

  update: (id: number, data: Partial<SubscribeRule>) =>
    request.put<ApiResponse<SubscribeRule>>(`/subscribe-rules/${id}`, data),

  delete: (id: number) =>
    request.delete<ApiResponse<null>>(`/subscribe-rules/${id}`),
}

// ===== Mute Rule API =====
export const muteRuleApi = {
  list: (params?: { page?: number; page_size?: number }) =>
    request.get<ApiResponse<PageData<MuteRule>>>('/mute-rules', { params }),

  get: (id: number) =>
    request.get<ApiResponse<MuteRule>>(`/mute-rules/${id}`),

  create: (data: Partial<MuteRule>) =>
    request.post<ApiResponse<MuteRule>>('/mute-rules', data),

  update: (id: number, data: Partial<MuteRule>) =>
    request.put<ApiResponse<MuteRule>>(`/mute-rules/${id}`, data),

  delete: (id: number) =>
    request.delete<ApiResponse<null>>(`/mute-rules/${id}`),

  preview: () =>
    request.get<ApiResponse<Array<{
      rule_id: number; rule_name: string
      matched_count: number; matched_alerts: AlertEvent[]
    }>>>('/mute-rules/preview'),
}

// ===== Inhibition Rules API =====
export const inhibitionRuleApi = {
  list: (params?: { page?: number; page_size?: number }) =>
    request.get<ApiResponse<PageData<InhibitionRule>>>('/inhibition-rules', { params }),

  get: (id: number) =>
    request.get<ApiResponse<InhibitionRule>>(`/inhibition-rules/${id}`),

  create: (data: Partial<InhibitionRule>) =>
    request.post<ApiResponse<InhibitionRule>>('/inhibition-rules', data),

  update: (id: number, data: Partial<InhibitionRule>) =>
    request.put<ApiResponse<InhibitionRule>>(`/inhibition-rules/${id}`, data),

  delete: (id: number) =>
    request.delete<ApiResponse<null>>(`/inhibition-rules/${id}`),
}
