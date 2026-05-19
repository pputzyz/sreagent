import request from './request'
import type {
  ApiResponse,
  PageData,
  User,
  Team,
  Schedule,
  ScheduleParticipant,
  ScheduleOverride,
  OnCallShift,
  EscalationPolicy,
  EscalationStep,
} from '@/types'

// ===== Team API =====
export const teamApi = {
  list: (params?: { page?: number; page_size?: number }) =>
    request.get<ApiResponse<PageData<Team>>>('/teams', { params }),

  get: (id: number) =>
    request.get<ApiResponse<Team>>(`/teams/${id}`),

  create: (data: Partial<Team>) =>
    request.post<ApiResponse<Team>>('/teams', data),

  update: (id: number, data: Partial<Team>) =>
    request.put<ApiResponse<Team>>(`/teams/${id}`, data),

  delete: (id: number) =>
    request.delete<ApiResponse<null>>(`/teams/${id}`),

  addMember: (teamId: number, userId: number) =>
    request.post<ApiResponse<null>>(`/teams/${teamId}/members`, { user_id: userId }),

  removeMember: (teamId: number, userId: number) =>
    request.delete<ApiResponse<null>>(`/teams/${teamId}/members/${userId}`),

  listMembers: (teamId: number) =>
    request.get<ApiResponse<User[]>>(`/teams/${teamId}/members`),
}

// ===== Schedule API =====
export const scheduleApi = {
  list: (params?: { page?: number; page_size?: number; team_id?: number }) =>
    request.get<ApiResponse<PageData<Schedule>>>('/schedules', { params }),

  get: (id: number) =>
    request.get<ApiResponse<Schedule>>(`/schedules/${id}`),

  create: (data: Partial<Schedule>) =>
    request.post<ApiResponse<Schedule>>('/schedules', data),

  update: (id: number, data: Partial<Schedule>) =>
    request.put<ApiResponse<Schedule>>(`/schedules/${id}`, data),

  delete: (id: number) =>
    request.delete<ApiResponse<null>>(`/schedules/${id}`),

  getCurrentOnCall: (id: number) =>
    request.get<ApiResponse<User | null>>(`/schedules/${id}/oncall`),

  setParticipants: (id: number, participants: { user_id: number; position: number }[]) =>
    request.put<ApiResponse<ScheduleParticipant[]>>(`/schedules/${id}/participants`, { participants }),

  getParticipants: (id: number) =>
    request.get<ApiResponse<ScheduleParticipant[]>>(`/schedules/${id}/participants`),

  createOverride: (id: number, data: { user_id: number; start_time: string; end_time: string; reason: string }) =>
    request.post<ApiResponse<ScheduleOverride>>(`/schedules/${id}/overrides`, data),

  listOverrides: (id: number) =>
    request.get<ApiResponse<ScheduleOverride[]>>(`/schedules/${id}/overrides`),

  deleteOverride: (id: number, overrideId: number) =>
    request.delete<ApiResponse<null>>(`/schedules/${id}/overrides/${overrideId}`),

  listShifts: (id: number, params: { start?: string; end?: string }) =>
    request.get<ApiResponse<OnCallShift[]>>(`/schedules/${id}/shifts`, { params }),

  createShift: (id: number, data: Partial<OnCallShift>) =>
    request.post<ApiResponse<OnCallShift>>(`/schedules/${id}/shifts`, data),

  updateShift: (id: number, shiftId: number, data: Partial<OnCallShift>) =>
    request.put<ApiResponse<OnCallShift>>(`/schedules/${id}/shifts/${shiftId}`, data),

  deleteShift: (id: number, shiftId: number) =>
    request.delete<ApiResponse<null>>(`/schedules/${id}/shifts/${shiftId}`),

  generateShifts: (id: number, data: { weeks: number }) =>
    request.post<ApiResponse<null>>(`/schedules/${id}/generate-shifts`, data),
}

// ===== Escalation Policy API =====
export const escalationApi = {
  list: (params?: { page?: number; page_size?: number }) =>
    request.get<ApiResponse<PageData<EscalationPolicy>>>('/escalation-policies', { params }),

  get: (id: number) =>
    request.get<ApiResponse<EscalationPolicy>>(`/escalation-policies/${id}`),

  create: (data: Partial<EscalationPolicy>) =>
    request.post<ApiResponse<EscalationPolicy>>('/escalation-policies', data),

  update: (id: number, data: Partial<EscalationPolicy>) =>
    request.put<ApiResponse<EscalationPolicy>>(`/escalation-policies/${id}`, data),

  delete: (id: number) =>
    request.delete<ApiResponse<null>>(`/escalation-policies/${id}`),

}

// ===== iCal Schedule Export =====
export const scheduleICalApi = {
  exportURL: (scheduleId: number) =>
    `/api/v1/schedules/${scheduleId}/ical`,
}
