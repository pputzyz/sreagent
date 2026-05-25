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
  annotations?: AnnotationConfig[]
}

export type PanelType = 'timeseries' | 'stat' | 'gauge' | 'table' | 'pie' | 'bar' | 'text' | 'row'

export interface PanelConfig {
  id: string
  title: string
  description?: string
  type: PanelType
  gridPos: { x: number; y: number; w: number; h: number }
  targets: PanelTarget[]
  options: PanelOptions
  repeatByVariable?: string
  transparent?: boolean
}

export interface PanelTarget {
  datasourceId: number
  expression: string
  legendFormat: string
  refId?: string
  instant?: boolean
  hide?: boolean
  yAxisPosition?: 'left' | 'right'
}

export interface PanelOptions {
  // General
  thresholds?: ThresholdItem[]
  valueMappings?: ValueMapping[]
  unit?: string
  decimals?: number

  // Timeseries
  drawStyle?: 'line' | 'bars' | 'points'
  fillOpacity?: number      // 0-100
  stacking?: 'normal' | 'none'
  showLegend?: boolean
  legendPosition?: 'bottom' | 'right' | 'hidden'
  lineWidth?: number
  gradientMode?: 'none' | 'opacity'

  // Stat
  colorMode?: 'value' | 'background'
  graphMode?: 'none' | 'area'
  textMode?: 'auto' | 'value' | 'name' | 'value_and_name'

  // Gauge
  min?: number
  max?: number

  // Table
  sortBy?: string
  sortOrder?: 'asc' | 'desc'
  enablePagination?: boolean
  pageSize?: number

  // Text
  content?: string
  mode?: 'markdown' | 'html'

  // Row
  collapsed?: boolean

  // Legacy
  color?: string
}

export interface ThresholdItem {
  value: number
  color: string
  mode?: 'absolute' | 'percentage'
}

export interface ValueMapping {
  type: 'value' | 'range' | 'special'
  match?: { value?: string; from?: number; to?: number }
  result: { text?: string; color?: string; icon?: string }
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
  allValue?: string
  refresh: 'onLoad' | 'onTimeRangeChange' | 'never'
  sort: 'disabled' | 'asc' | 'desc' | 'numerical-asc' | 'numerical-desc'
  datasourceId?: number
}

export interface AnnotationConfig {
  enable: boolean
  datasourceId?: number
  query?: string
  tags?: string[]
  color?: string
  icon?: string
}

export interface Annotation {
  id: number
  dashboard_id: number
  time: string
  end_time?: string
  text: string
  tags: Record<string, string>
  source: string
  created_by: number
  created_at: string
}
