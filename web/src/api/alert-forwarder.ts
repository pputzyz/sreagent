import request from './request'
import type { ApiResponse, PageData } from '@/types'

export type ForwarderDirection = 'inbound' | 'outbound' | 'bidirectional'
export type InboundMode = 'integrate' | 'proxy'
export type SourceFormat = 'alertmanager' | 'grafana' | 'prometheus' | 'generic'
export type AuthType = 'none' | 'bearer' | 'basic' | 'hmac'

export interface AlertForwarder {
  id: number
  name: string
  description: string
  enabled: boolean
  direction: ForwarderDirection
  priority: number
  inbound_config?: InboundConfig
  outbound_config?: OutboundConfig
  inbound_severity_mapping?: SeverityMappingConfig
  outbound_severity_mapping?: SeverityMappingConfig
  platform_capabilities?: PlatformCapabilitiesConfig
  match_labels?: Record<string, string>
  created_at: string
  updated_at: string
}

export interface InboundConfig {
  source_format: SourceFormat
  mode: InboundMode
  auth_type: AuthType
  auth_config?: AuthConfig
  proxy_target?: OutboundConfig
}

export interface AuthConfig {
  token?: string
  username?: string
  password?: string
  hmac_secret?: string
  hmac_header?: string
  hmac_algorithm?: string
}

export interface OutboundConfig {
  target_media_id?: number
  target_url?: string
  method?: string
  headers?: Record<string, string>
  body_template?: string
  timeout?: number
  retry_times?: number
  retry_interval?: number
}

export interface SeverityMappingConfig {
  enabled: boolean
  mapping: Record<string, string>
  default_severity?: string
}

export interface PlatformCapabilitiesConfig {
  enable_escalation: boolean
  enable_mute: boolean
  enable_inhibition: boolean
  enable_notification: boolean
  enable_ai_analysis: boolean
  pipeline_id?: number
}

export interface ForwarderListParams {
  page?: number
  page_size?: number
  direction?: string
  enabled?: boolean
}

export interface ForwarderStats {
  by_direction: Record<string, number>
  enabled_count: number
}

// Create a new alert forwarder
export function createAlertForwarder(data: Partial<AlertForwarder>) {
  return request.post<ApiResponse<AlertForwarder>>('/alert-forwarders', data)
}

// Get alert forwarder by ID
export function getAlertForwarder(id: number) {
  return request.get<ApiResponse<AlertForwarder>>(`/alert-forwarders/${id}`)
}

// List alert forwarders
export function listAlertForwarders(params?: ForwarderListParams) {
  return request.get<ApiResponse<PageData<AlertForwarder>>>('/alert-forwarders', { params })
}

// Update alert forwarder
export function updateAlertForwarder(id: number, data: Partial<AlertForwarder>) {
  return request.put<ApiResponse<AlertForwarder>>(`/alert-forwarders/${id}`, data)
}

// Delete alert forwarder
export function deleteAlertForwarder(id: number) {
  return request.delete<ApiResponse<void>>(`/alert-forwarders/${id}`)
}

// Enable alert forwarder
export function enableAlertForwarder(id: number) {
  return request.post<ApiResponse<void>>(`/alert-forwarders/${id}/enable`)
}

// Disable alert forwarder
export function disableAlertForwarder(id: number) {
  return request.post<ApiResponse<void>>(`/alert-forwarders/${id}/disable`)
}

// Batch enable alert forwarders
export function batchEnableAlertForwarders(ids: number[]) {
  return request.post<ApiResponse<void>>('/alert-forwarders/batch/enable', { ids })
}

// Batch disable alert forwarders
export function batchDisableAlertForwarders(ids: number[]) {
  return request.post<ApiResponse<void>>('/alert-forwarders/batch/disable', { ids })
}

// Batch delete alert forwarders
export function batchDeleteAlertForwarders(ids: number[]) {
  return request.post<ApiResponse<void>>('/alert-forwarders/batch/delete', { ids })
}

// Test alert forwarder
export function testAlertForwarder(id: number) {
  return request.post<ApiResponse<Record<string, any>>>(`/alert-forwarders/${id}/test`)
}

// Get forwarder stats
export function getForwarderStats() {
  return request.get<ApiResponse<ForwarderStats>>('/alert-forwarders/stats')
}
