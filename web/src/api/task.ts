import request from './request'
import type { ApiResponse, PageData } from '@/types'

// ===== Task Template =====

export interface TaskTpl {
  id: number
  name: string
  script: string
  args: string
  batch: number
  tolerance: number
  timeout: number
  account: string
  pause: string
  hosts: string // JSON array
  tags: string // JSON array
  note: string
  create_by: string
  update_by: string
  created_at: string
  updated_at: string
}

export interface CreateTaskTplRequest {
  name: string
  script: string
  args?: string
  batch?: number
  tolerance?: number
  timeout?: number
  account?: string
  pause?: string
  hosts?: string
  tags?: string
  note?: string
}

export type UpdateTaskTplRequest = Partial<CreateTaskTplRequest>

// ===== Task Record =====

export interface TaskRecord {
  id: number
  tpl_id: number
  event_id: number
  title: string
  account: string
  batch: number
  tolerance: number
  timeout: number
  script: string
  args: string
  hosts: string // JSON array
  status: number // 0=pending 1=running 2=success 3=fail
  create_by: string
  created_at: string
  updated_at: string
}

export interface TaskHostRecord {
  id: number
  task_id: number
  host: string
  status: number // 0=pending 1=running 2=success 3=fail
  stdout: string
  stderr: string
  exit_code: number
  duration_ms: number
  created_at: string
  updated_at: string
}

export interface ExecuteTaskRequest {
  tpl_id: number
  hosts?: string[]
  event_id?: number
  title?: string
}

// ===== API =====

export const taskTplApi = {
  list: (params?: { keyword?: string; page?: number; page_size?: number }) =>
    request.get<ApiResponse<PageData<TaskTpl>>>('/task-tpls', { params }),

  get: (id: number) =>
    request.get<ApiResponse<TaskTpl>>(`/task-tpls/${id}`),

  create: (data: CreateTaskTplRequest) =>
    request.post<ApiResponse<TaskTpl>>('/task-tpls', data),

  update: (id: number, data: UpdateTaskTplRequest) =>
    request.put<ApiResponse<TaskTpl>>(`/task-tpls/${id}`, data),

  delete: (id: number) =>
    request.delete<ApiResponse<null>>(`/task-tpls/${id}`),
}

export const taskApi = {
  execute: (data: ExecuteTaskRequest) =>
    request.post<ApiResponse<TaskRecord>>('/tasks', data),

  list: (params?: { tpl_id?: number; event_id?: number; status?: number; page?: number; page_size?: number }) =>
    request.get<ApiResponse<PageData<TaskRecord>>>('/tasks', { params }),

  get: (id: number) =>
    request.get<ApiResponse<TaskRecord>>(`/tasks/${id}`),

  getHosts: (id: number) =>
    request.get<ApiResponse<TaskHostRecord[]>>(`/tasks/${id}/hosts`),
}

// ===== Status helpers =====

export const TaskStatus = {
  Pending: 0,
  Running: 1,
  Success: 2,
  Fail: 3,
} as const

export function getTaskStatusLabel(status: number): string {
  switch (status) {
    case TaskStatus.Pending: return 'pending'
    case TaskStatus.Running: return 'running'
    case TaskStatus.Success: return 'success'
    case TaskStatus.Fail: return 'failed'
    default: return 'unknown'
  }
}

export function getTaskStatusType(status: number): 'default' | 'info' | 'success' | 'error' {
  switch (status) {
    case TaskStatus.Pending: return 'default'
    case TaskStatus.Running: return 'info'
    case TaskStatus.Success: return 'success'
    case TaskStatus.Fail: return 'error'
    default: return 'default'
  }
}
