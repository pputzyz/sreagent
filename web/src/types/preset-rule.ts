export interface PresetRule {
  id: number
  name: string
  display_name: string
  category: string
  sub_category: string
  component: string
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

export interface AIModule {
  enabled: boolean
  description: string
}

export interface AIModuleConfig {
  platform: AIModule
  chat: AIModule
  rule_gen: AIModule
  analysis: AIModule
  agent: AIModule
}

// ===== AI Rule Generation =====
export interface RuleGenerateRequest {
  description: string
  datasource_id?: number
  rule_type?: string
  context?: {
    existing_rules?: boolean
    include_labels?: boolean
    include_routing?: boolean
    target_channel_id?: number
  }
}

export interface RuleGenerateResult {
  type: string
  name: string
  expression?: string
  for_duration?: string
  severity?: string
  labels?: Record<string, string>
  annotations?: Record<string, string>
  source_labels?: string[]
  source_value?: string
  target_labels?: string[]
  equal_labels?: string[]
  description: string
  confidence: number
  warnings: string[]
  suggested_channel?: {
    id: number
    name: string
    reason: string
  }
}

export interface ValidationResult {
  valid: boolean
  result_type?: string
  sample_count?: number
  sample_labels?: string[]
  error?: string
  warnings?: string[]
}
