import request from './request'
import type { ApiResponse, PageData } from '@/types'
import type {
  PresetRule, PresetRuleOverride, AIModuleConfig,
  RuleGenerateRequest, RuleGenerateResult, ValidationResult,
} from '@/types/preset-rule'

// ===== Preset Rule API =====
export const presetRuleApi = {
  list: (params?: { page?: number; page_size?: number; category?: string; search?: string }) =>
    request.get<ApiResponse<PageData<PresetRule>>>('/preset-rules', { params }),

  get: (id: number) =>
    request.get<ApiResponse<PresetRule>>(`/preset-rules/${id}`),

  categories: () =>
    request.get<ApiResponse<string[]>>('/preset-rules/categories'),

  apply: (id: number, override?: PresetRuleOverride) =>
    request.post<ApiResponse<PresetRule>>(`/preset-rules/${id}/apply`, override),

  importYAML: (yaml: string) =>
    request.post<ApiResponse<{ imported: number; skipped: number; errors: string[] }>>('/preset-rules/import', { yaml }),

  delete: (id: number) =>
    request.delete<ApiResponse<null>>(`/preset-rules/${id}`),
}

// ===== AI Module API =====
export const aiModuleApi = {
  getModules: () =>
    request.get<ApiResponse<AIModuleConfig>>('/ai/modules'),

  updateModules: (config: AIModuleConfig) =>
    request.put<ApiResponse<null>>('/ai/modules', config),
}

// ===== AI Rule Generation API =====
export const aiRuleApi = {
  generate: (data: RuleGenerateRequest) =>
    request.post<ApiResponse<RuleGenerateResult>>('/ai/rules/generate', data),

  validate: (data: { datasource_id: number; expression: string }) =>
    request.post<ApiResponse<ValidationResult>>('/ai/rules/validate', data),

  suggestLabels: (data: { datasource_id: number; expression: string }) =>
    request.post<ApiResponse<{ detected_metrics: Record<string, string>; suggested_labels: Record<string, { value: string; confidence: number; source: string }> }>>('/ai/rules/suggest-labels', data),

  generateInhibition: (data: { description: string; datasource_id?: number }) =>
    request.post<ApiResponse<RuleGenerateResult>>('/ai/rules/generate-inhibition', data),
}
