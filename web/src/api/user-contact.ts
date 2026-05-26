import request from './request'
import type { ApiResponse } from '@/types'

export interface UserContact {
  id: number
  user_id: number
  type: string // email, phone, feishu, wecom, dingtalk, webhook
  value: string
  name: string
  is_default: boolean
}

export const userContactApi = {
  list: () =>
    request.get<ApiResponse<UserContact[]>>('/user/contacts'),

  create: (data: { type: string; value: string; name: string; is_default?: boolean }) =>
    request.post<ApiResponse<UserContact>>('/user/contacts', data),

  update: (id: number, data: { type?: string; value?: string; name?: string; is_default?: boolean }) =>
    request.put<ApiResponse<UserContact>>(`/user/contacts/${id}`, data),

  delete: (id: number) =>
    request.delete<ApiResponse<null>>(`/user/contacts/${id}`),

  setDefault: (id: number) =>
    request.post<ApiResponse<null>>(`/user/contacts/${id}/default`),
}
