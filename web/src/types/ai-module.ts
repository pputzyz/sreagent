export interface AIModule {
  enabled: boolean
  description: string
  provider_key?: string
}

export interface AIModuleConfig {
  platform: AIModule
  chat: AIModule
  rule_gen: AIModule
  analysis: AIModule
  agent: AIModule
}

export interface AIProvider {
  key: string
  provider: string // 'openai' | 'azure' | 'ollama' | 'custom'
  api_key: string
  base_url: string
  model: string
  enabled: boolean
}

export interface AIProvidersConfig {
  default_provider: string
  providers: AIProvider[]
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

export interface MuteRuleGenerateResult {
  type: string
  name: string
  description: string
  match_labels: Record<string, string>
  severities: string[]
  start_time?: string
  end_time?: string
  periodic_start?: string
  periodic_end?: string
  days_of_week: string[]
  timezone: string
  rule_ids?: number[]
  confidence: number
  warnings: string[]
}

export interface ValidationResult {
  valid: boolean
  result_type?: string
  sample_count?: number
  sample_labels?: string[]
  error?: string
  warnings?: string[]
}

export interface DryRunResult {
  rule: RuleGenerateResult
  validation?: ValidationResult
}
