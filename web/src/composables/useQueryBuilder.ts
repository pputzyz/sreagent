/**
 * useQueryBuilder — Reactive state management for the Visual PromQL Builder.
 *
 * Converts a structured builder state into a PromQL expression string,
 * and provides basic parsing of simple PromQL back into builder state.
 *
 * Supports the 80% use case:
 *   metric{labels} with optional function wrapper and aggregation.
 */
import { ref, computed } from 'vue'

// ---- Types ----

export type LabelOperator = '=' | '!=' | '=~' | '!~'

export interface LabelFilter {
  id: string
  key: string
  operator: LabelOperator
  value: string
}

/** Supported range-vector functions (take a range selector [5m]) */
export type RangeFunction =
  | 'rate'
  | 'irate'
  | 'increase'
  | 'delta'
  | 'deriv'
  | 'avg_over_time'
  | 'min_over_time'
  | 'max_over_time'
  | 'sum_over_time'
  | 'count_over_time'
  | 'last_over_time'
  | 'stddev_over_time'
  | 'quantile_over_time'
  | 'absent_over_time'

/** Supported aggregation operators */
export type AggregationOp =
  | 'sum'
  | 'avg'
  | 'min'
  | 'max'
  | 'count'
  | 'group'
  | 'stddev'
  | 'stdvar'
  | 'topk'
  | 'bottomk'
  | 'count_values'
  | 'quantile'
  | 'absent'

export type BinaryOp = '+' | '-' | '*' | '/' | '%' | '^'
export type GroupModifier = 'by' | 'without' | ''

export interface RangeFunctionConfig {
  enabled: boolean
  fn: RangeFunction
  range: string  // e.g. "5m", "1h"
}

export interface AggregationConfig {
  enabled: boolean
  op: AggregationOp
  groupModifier: GroupModifier
  groupLabels: string[]
}

export interface BinaryOperand {
  id: string
  op: BinaryOp
  type: 'scalar' | 'metric'
  scalarValue: string
  metricExpression: string
}

export interface QueryBuilderState {
  metricName: string
  labelFilters: LabelFilter[]
  rangeFunction: RangeFunctionConfig
  aggregation: AggregationConfig
  binaryOperands: BinaryOperand[]
}

// ---- Helpers ----

let nextId = 0
function uid(): string {
  return `qb_${++nextId}_${Date.now().toString(36)}`
}

export function createLabelFilter(): LabelFilter {
  return { id: uid(), key: '', operator: '=', value: '' }
}

export function createBinaryOperand(): BinaryOperand {
  return { id: uid(), op: '+', type: 'scalar', scalarValue: '', metricExpression: '' }
}

export function createDefaultState(): QueryBuilderState {
  return {
    metricName: '',
    labelFilters: [],
    rangeFunction: { enabled: false, fn: 'rate', range: '5m' },
    aggregation: { enabled: false, op: 'sum', groupModifier: '', groupLabels: [] },
    binaryOperands: [],
  }
}

// ---- Range function definitions ----

export const RANGE_FUNCTIONS: { label: string; value: RangeFunction; needsRange: boolean }[] = [
  { label: 'rate()', value: 'rate', needsRange: true },
  { label: 'irate()', value: 'irate', needsRange: true },
  { label: 'increase()', value: 'increase', needsRange: true },
  { label: 'delta()', value: 'delta', needsRange: true },
  { label: 'deriv()', value: 'deriv', needsRange: true },
  { label: 'avg_over_time()', value: 'avg_over_time', needsRange: true },
  { label: 'min_over_time()', value: 'min_over_time', needsRange: true },
  { label: 'max_over_time()', value: 'max_over_time', needsRange: true },
  { label: 'sum_over_time()', value: 'sum_over_time', needsRange: true },
  { label: 'count_over_time()', value: 'count_over_time', needsRange: true },
  { label: 'last_over_time()', value: 'last_over_time', needsRange: true },
  { label: 'stddev_over_time()', value: 'stddev_over_time', needsRange: true },
  { label: 'quantile_over_time()', value: 'quantile_over_time', needsRange: true },
  { label: 'absent_over_time()', value: 'absent_over_time', needsRange: true },
]

export const AGGREGATION_OPS: { label: string; value: AggregationOp }[] = [
  { label: 'sum', value: 'sum' },
  { label: 'avg', value: 'avg' },
  { label: 'min', value: 'min' },
  { label: 'max', value: 'max' },
  { label: 'count', value: 'count' },
  { label: 'group', value: 'group' },
  { label: 'stddev', value: 'stddev' },
  { label: 'stdvar', value: 'stdvar' },
  { label: 'topk', value: 'topk' },
  { label: 'bottomk', value: 'bottomk' },
  { label: 'count_values', value: 'count_values' },
  { label: 'quantile', value: 'quantile' },
  { label: 'absent', value: 'absent' },
]

export const BINARY_OPS: { label: string; value: BinaryOp }[] = [
  { label: '+', value: '+' },
  { label: '-', value: '-' },
  { label: '*', value: '*' },
  { label: '/', value: '/' },
  { label: '%', value: '%' },
  { label: '^', value: '^' },
]

export const DURATION_PRESETS = ['1m', '5m', '10m', '15m', '30m', '1h', '2h', '6h', '12h', '1d', '1w']

// ---- toPromQL ----

/** Escape a label value for PromQL */
function escapeLabelValue(v: string): string {
  return v.replace(/\\/g, '\\\\').replace(/"/g, '\\"')
}

/** Build the inner selector: metric_name{key="op value"} */
function buildSelector(metricName: string, filters: LabelFilter[]): string {
  const name = metricName.trim()
  const active = filters.filter(f => f.key.trim() && f.value.trim())
  if (active.length === 0) return name
  const labelParts = active.map(f => {
    const k = f.key.trim()
    const v = escapeLabelValue(f.value.trim())
    return `${k}${f.operator}"${v}"`
  })
  return name ? `${name}{${labelParts.join(',')}}` : `{${labelParts.join(',')}}`
}

/** Convert builder state to a PromQL expression string */
export function toPromQL(state: QueryBuilderState): string {
  let expr = buildSelector(state.metricName, state.labelFilters)

  // Wrap in range function
  if (state.rangeFunction.enabled && expr) {
    expr = `${state.rangeFunction.fn}(${expr}[${state.rangeFunction.range}])`
  }

  // Apply aggregation
  if (state.aggregation.enabled && expr) {
    const { op, groupModifier, groupLabels } = state.aggregation
    if (groupModifier && groupLabels.length > 0) {
      expr = `${op} ${groupModifier}(${groupLabels.join(', ')}) (${expr})`
    } else {
      expr = `${op}(${expr})`
    }
  }

  // Apply binary operands
  for (const operand of state.binaryOperands) {
    const rhs = operand.type === 'scalar'
      ? operand.scalarValue.trim()
      : operand.metricExpression.trim()
    if (!rhs) continue
    expr = `(${expr}) ${operand.op} ${rhs}`
  }

  return expr
}

// ---- fromPromQL (basic parser) ----

/**
 * Attempt to parse a simple PromQL expression into QueryBuilderState.
 * Handles: metric{labels}, wrapped in range functions, wrapped in aggregations.
 * Does NOT handle nested binary operations or sub-queries.
 */
export function fromPromQL(promql: string): QueryBuilderState | null {
  const state = createDefaultState()
  let expr = promql.trim()
  if (!expr) return state

  // Try to strip aggregation: op(...) or op by(...) (...)
  const aggMatch = expr.match(/^(sum|avg|min|max|count|group|stddev|stdvar|topk|bottomk|count_values|quantile|absent)\s*(?:(by|without)\s*\(([^)]*)\)\s*)?\((.+)\)$/s)
  if (aggMatch) {
    state.aggregation.enabled = true
    state.aggregation.op = aggMatch[1] as AggregationOp
    if (aggMatch[2]) {
      state.aggregation.groupModifier = aggMatch[2] as GroupModifier
      state.aggregation.groupLabels = aggMatch[3].split(',').map(s => s.trim()).filter(Boolean)
    }
    expr = aggMatch[4].trim()
  }

  // Try to strip range function: fn(expr[duration])
  const fnMatch = expr.match(/^(rate|irate|increase|delta|deriv|avg_over_time|min_over_time|max_over_time|sum_over_time|count_over_time|last_over_time|stddev_over_time|quantile_over_time|absent_over_time)\((.+)\[([^\]]+)\]\)$/s)
  if (fnMatch) {
    state.rangeFunction.enabled = true
    state.rangeFunction.fn = fnMatch[1] as RangeFunction
    state.rangeFunction.range = fnMatch[3]
    expr = fnMatch[2].trim()
  }

  // Try to strip binary operations: (expr) op rhs
  const binMatch = expr.match(/^\((.+)\)\s*([+\-*\/\%^])\s*(.+)$/s)
  if (binMatch) {
    expr = binMatch[1].trim()
    const rhs = binMatch[3].trim()
    const op = createBinaryOperand()
    op.op = binMatch[2] as BinaryOp
    // Check if rhs is a number
    if (/^-?\d+(\.\d+)?$/.test(rhs)) {
      op.type = 'scalar'
      op.scalarValue = rhs
    } else {
      op.type = 'metric'
      op.metricExpression = rhs
    }
    state.binaryOperands.push(op)
  }

  // Parse selector: metric_name{key="value", ...} or {key="value", ...}
  const selectorMatch = expr.match(/^([a-zA-Z_:][a-zA-Z0-9_:]*)?\s*(?:\{([^}]*)\})?$/s)
  if (!selectorMatch) return null

  state.metricName = selectorMatch[1] || ''
  if (selectorMatch[2]) {
    // Parse label filters
    // Split by comma, but respect quoted strings
    const filterStr = selectorMatch[2]
    const filterParts = splitLabelFilters(filterStr)
    for (const part of filterParts) {
      const m = part.trim().match(/^([a-zA-Z_][a-zA-Z0-9_]*)\s*(!=~?|=~?)\s*"((?:[^"\\]|\\.)*)"$/)
      if (m) {
        const filter = createLabelFilter()
        filter.key = m[1]
        filter.operator = m[2] as LabelOperator
        filter.value = m[3].replace(/\\"/g, '"').replace(/\\\\/g, '\\')
        state.labelFilters.push(filter)
      }
    }
  }

  return state
}

/** Split label filter string by comma, respecting quoted values */
function splitLabelFilters(s: string): string[] {
  const parts: string[] = []
  let current = ''
  let inQuote = false
  let escaped = false
  for (const ch of s) {
    if (escaped) {
      current += ch
      escaped = false
      continue
    }
    if (ch === '\\') {
      escaped = true
      current += ch
      continue
    }
    if (ch === '"') {
      inQuote = !inQuote
      current += ch
      continue
    }
    if (ch === ',' && !inQuote) {
      parts.push(current)
      current = ''
      continue
    }
    current += ch
  }
  if (current.trim()) parts.push(current)
  return parts
}

// ---- Composable ----

export function useQueryBuilder() {
  const state = ref<QueryBuilderState>(createDefaultState())

  // Generated PromQL
  const generatedPromQL = computed(() => toPromQL(state.value))

  /** Parse a PromQL expression into builder state */
  function parseExpression(expr: string) {
    if (!expr || !expr.trim()) {
      state.value = createDefaultState()
      return
    }
    const parsed = fromPromQL(expr)
    if (parsed) {
      state.value = parsed
    }
  }

  // ---- Mutation helpers ----

  function addLabelFilter() {
    state.value.labelFilters.push(createLabelFilter())
  }

  function removeLabelFilter(id: string) {
    state.value.labelFilters = state.value.labelFilters.filter(f => f.id !== id)
  }

  function addBinaryOperand() {
    state.value.binaryOperands.push(createBinaryOperand())
  }

  function removeBinaryOperand(id: string) {
    state.value.binaryOperands = state.value.binaryOperands.filter(o => o.id !== id)
  }

  function reset() {
    state.value = createDefaultState()
  }

  return {
    state,
    generatedPromQL,
    parseExpression,
    addLabelFilter,
    removeLabelFilter,
    addBinaryOperand,
    removeBinaryOperand,
    reset,
  }
}
