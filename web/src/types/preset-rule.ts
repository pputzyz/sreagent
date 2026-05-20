export interface PresetRule {
  id: number
  name: string
  display_name: string
  category: string
  sub_category: string
  component: string
  cluster: string
  expression: string
  for_duration: string
  severity: string
  alert_type: string
  labels: Record<string, string>
  annotations: Record<string, string>
  source: string
  is_builtin: boolean
  usage_count: number
  description: string
  created_at: string
  updated_at: string
}

export interface PresetRuleOverride {
  datasource_id?: number
  channel_id?: number
  labels?: Record<string, string>
  severity?: string
}

export interface BatchApplyResult {
  preset_id: number
  alert_rule_id?: number
  matched_datasource_id?: number
  error?: string
}

export interface BatchApplyRequest {
  preset_ids: number[]
  auto_match_datasource?: boolean
  fallback_datasource_id?: number
  channel_id?: number
}
