export interface DashboardV2 {
  id: number
  name: string
  description: string
  tags: Record<string, string>
  config: string  // JSON string of DashboardConfig
  created_by: number
  updated_by: number
  is_public: boolean
  created_at: string
  updated_at: string
}

export interface DashboardConfig {
  panels: PanelConfig[]
  layout: { cols: number; rowHeight: number }
  variables: VariableConfig[]
}

export interface PanelConfig {
  id: string
  title: string
  type: 'timeseries' | 'stat' | 'gauge' | 'table' | 'pie' | 'bar'
  gridPos: { x: number; y: number; w: number; h: number }
  targets: PanelTarget[]
  options: Record<string, unknown>
}

export interface PanelTarget {
  datasourceId: number
  expression: string
  legendFormat: string
}

export interface VariableConfig {
  name: string
  label: string
  type: 'query' | 'custom' | 'interval' | 'datasource' | 'textbox' | 'constant' | 'adhoc'
  query?: string
  regex?: string
  options?: string[]
  defaultValue?: string
  multi: boolean
  includeAll: boolean
  refresh: 'onLoad' | 'onTimeRangeChange' | 'never'
  sort: 'disabled' | 'asc' | 'desc' | 'numerical-asc' | 'numerical-desc'
  datasourceId?: number
}
