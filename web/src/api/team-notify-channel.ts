import request from './request'
import type { ApiResponse } from '@/types'

// ===== Team Notify Channel API =====

export interface TeamNotifyChannel {
  id: number
  team_id: number
  media_id: number
  media_name?: string
  media_type?: string
  is_default: boolean
  created_at?: string
  updated_at?: string
}

export interface CreateTeamNotifyChannelRequest {
  team_id: number
  media_id: number
  is_default?: boolean
}

export interface UpdateTeamNotifyChannelRequest {
  team_id: number
  media_id: number
  is_default?: boolean
}

export const teamNotifyChannelApi = {
  list: (teamId: number) =>
    request.get<ApiResponse<TeamNotifyChannel[]>>(`/team-notify-channels/${teamId}`),

  create: (data: CreateTeamNotifyChannelRequest) =>
    request.post<ApiResponse<TeamNotifyChannel>>('/team-notify-channels', data),

  update: (id: number, data: UpdateTeamNotifyChannelRequest) =>
    request.put<ApiResponse<TeamNotifyChannel>>(`/team-notify-channels/${id}`, data),

  setDefault: (id: number) =>
    request.post<ApiResponse<null>>(`/team-notify-channels/${id}/default`),

  delete: (id: number) =>
    request.delete<ApiResponse<null>>(`/team-notify-channels/${id}`),
}

// ===== User Team Notify Pref API =====

export interface UserTeamNotifyPref {
  id: number
  team_id: number
  media_id: number
  media_name?: string
  team_name?: string
  is_muted: boolean
  created_at?: string
  updated_at?: string
}

export interface UpsertUserTeamNotifyPrefRequest {
  team_id: number
  media_id: number
  is_muted?: boolean
}

export const userTeamNotifyPrefApi = {
  list: () =>
    request.get<ApiResponse<UserTeamNotifyPref[]>>('/user/team-notify-prefs'),

  upsert: (data: UpsertUserTeamNotifyPrefRequest) =>
    request.post<ApiResponse<UserTeamNotifyPref>>('/user/team-notify-prefs', data),

  delete: (id: number) =>
    request.delete<ApiResponse<null>>(`/user/team-notify-prefs/${id}`),
}
