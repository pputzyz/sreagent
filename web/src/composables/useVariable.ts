import { ref, watch, computed, type Ref } from 'vue'
import { datasourceApi } from '@/api'
import { computeTimeStep } from '@/utils/timeStep'
import type { VariableConfig } from '@/types/dashboard'
import type { TimeRange } from '@/types/query'

// FE3-8: TODO — SSE-based real-time variable value updates
// Currently variables are resolved on-demand (dashboard load, time range change).
// For long-lived dashboards, variable options may become stale.
// Plan:
//  1. Backend: Add SSE endpoint GET /api/v1/variables/stream that pushes updates
//     when label values change (e.g., new series appear in Prometheus).
//  2. Frontend: Connect to SSE in useVariable, update options map on events.
//  3. Only subscribe for query-type variables with a refresh_interval config.
//  4. Add a configurable interval (default 60s) per variable to balance freshness vs load.
//  5. Graceful fallback: if SSE disconnects, fall back to current poll-on-demand behavior.

const ALL_SENTINEL = '__all'

export interface VariableState {
  config: VariableConfig
  value: string | string[]
  options: string[]
  loading: boolean
}

export interface AdhocFilter {
  key: string
  op: string
  value: string
}

// TODO(FE3-8): Real-time variable updates via SSE.
// Currently, variables are resolved once on mount and when timeRange changes (polling via watch).
// For dashboards with high-cardinality query variables that change frequently, consider:
//  1. SSE endpoint on the backend that streams variable option changes (e.g., new label values)
//  2. Frontend EventSource listener that updates options reactively without full re-query
//  3. Debounced re-resolution on SSE events to avoid excessive re-renders
//  4. Per-variable SSE subscription (only for query-type variables with a refresh_interval config)
// This would reduce API polling overhead and provide near-instant variable option updates.
export function useVariable(
  variables: Ref<VariableConfig[]>,
  timeRange: Ref<TimeRange>,
) {
  const states = ref<Map<string, VariableState>>(new Map())
  const adhocFilters = ref<Map<string, AdhocFilter[]>>(new Map())

  // Initialize states from config
  function initStates() {
    const newStates = new Map<string, VariableState>()
    for (const v of variables.value) {
      const existing = states.value.get(v.name)
      let defaultVal: string | string[] = existing?.value ?? v.defaultValue ?? ''
      // For multi-select, ensure value is an array
      if (v.multi && !Array.isArray(defaultVal)) {
        defaultVal = defaultVal ? [defaultVal] : []
      }
      newStates.set(v.name, {
        config: v,
        value: defaultVal,
        options: existing?.options ?? v.options ?? [],
        loading: false,
      })
      // Init adhoc filters map
      if (v.type === 'adhoc' && !adhocFilters.value.has(v.name)) {
        adhocFilters.value.set(v.name, [])
      }
    }
    states.value = newStates
  }

  // Build the effective options list (with includeAll prepended)
  function buildOptions(rawOptions: string[], config: VariableConfig): string[] {
    if (config.includeAll) {
      return [ALL_SENTINEL, ...rawOptions]
    }
    return rawOptions
  }

  // Resolve a query-type variable
  async function resolveQueryVariable(state: VariableState, preResolve?: (s: string) => string) {
    if (!state.config.query || !state.config.datasourceId) return
    state.loading = true
    try {
      // Pre-process the query through variable replacement (for chained deps)
      const query = preResolve ? preResolve(state.config.query) : state.config.query
      const res = await datasourceApi.query(state.config.datasourceId, {
        expression: query,
      })
      const data = res.data.data
      let opts: string[] = []
      if (data.series) {
        for (const s of data.series) {
          // Extract the first label value that's not __name__
          const values = Object.entries(s.labels)
            .filter(([k]) => k !== '__name__')
            .map(([, v]) => v)
          if (values.length > 0) opts.push(values[0])
        }
      }
      // Apply regex filter
      if (state.config.regex) {
        try {
          const re = new RegExp(state.config.regex)
          opts = opts.filter(o => re.test(o))
        } catch (e) {
          console.warn('[useVariable] invalid regex, skipping filter:', state.config.regex, e)
        }
      }
      // Apply sort
      applySort(opts, state.config.sort)

      state.options = buildOptions(opts, state.config)
      // Auto-select if current value not in options
      if (state.config.multi) {
        const current = Array.isArray(state.value) ? state.value : [state.value]
        const valid = current.filter(v => state.options.includes(v))
        if (valid.length === 0 && state.options.length > 0) {
          state.value = state.options[0] === ALL_SENTINEL ? [ALL_SENTINEL] : [state.options[0]]
        }
      } else {
        if (opts.length > 0 && !state.options.includes(state.value as string)) {
          state.value = state.options[0]
        }
      }
    } catch (err) {
      console.warn('[useVariable] Failed to resolve query variable:', state.config.name, err)
    } finally {
      state.loading = false
    }
  }

  // Resolve datasource-type variable
  async function resolveDatasourceVariable(state: VariableState) {
    state.loading = true
    try {
      const res = await datasourceApi.list({ page: 1, page_size: 200 })
      let names = (res.data.data.list || [])
        .filter((ds: { is_enabled: boolean }) => ds.is_enabled)
        .map((ds: { name: string }) => ds.name)

      // Apply regex filter
      if (state.config.regex) {
        try {
          const re = new RegExp(state.config.regex)
          names = names.filter(n => re.test(n))
        } catch (e) {
          console.warn('[useVariable] invalid regex for datasource var:', state.config.regex, e)
        }
      }

      state.options = buildOptions(names, state.config)
      if (names.length > 0 && !state.options.includes(state.value as string)) {
        state.value = state.options[0]
      }
    } catch (err) {
      console.warn('[useVariable] Failed to resolve datasource variable:', state.config.name, err)
    } finally {
      state.loading = false
    }
  }

  // Resolve interval-type variable
  function resolveIntervalVariable(state: VariableState) {
    const defaults = ['1m', '5m', '10m', '30m', '1h']
    const opts = state.config.options?.length ? state.config.options : defaults
    state.options = buildOptions(opts, state.config)
    if (!state.options.includes(state.value as string)) {
      state.value = opts[0]
    }
  }

  // Resolve constant-type variable
  function resolveConstantVariable(state: VariableState) {
    const val = state.config.defaultValue || ''
    state.options = [val]
    state.value = val
  }

  // Apply sort to an options array
  function applySort(opts: string[], sort?: string) {
    if (sort === 'asc') opts.sort()
    else if (sort === 'desc') opts.sort().reverse()
    else if (sort === 'numerical-asc') opts.sort((a, b) => {
      const na = Number(a), nb = Number(b)
      if (isNaN(na) && isNaN(nb)) return 0
      if (isNaN(na)) return 1
      if (isNaN(nb)) return -1
      return na - nb
    })
    else if (sort === 'numerical-desc') opts.sort((a, b) => {
      const na = Number(a), nb = Number(b)
      if (isNaN(na) && isNaN(nb)) return 0
      if (isNaN(na)) return 1
      if (isNaN(nb)) return -1
      return nb - na
    })
  }

  // Resolve all variables sequentially (for chained dependency support).
  // Cycle detection: tracks visited variable names during dependency traversal.
  // If a variable is encountered again, it's a cycle — skip that dependency.
  const MAX_RESOLVE_DEPTH = 10

  let resolveSeq = 0
  async function resolveAll() {
    const seq = ++resolveSeq
    const visited = new Set<string>()

    // Build a preResolve function that uses current states with cycle detection
    function preResolve(s: string): string {
      return replaceInString(s, states.value)
    }

    for (const [name, state] of states.value) {
      if (seq !== resolveSeq) return // superseded by a newer call

      // Cycle detection: skip if this variable was already resolved in this pass
      if (visited.has(name)) {
        console.warn(`[useVariable] Cycle detected for variable "${name}", skipping resolution`)
        continue
      }
      visited.add(name)

      switch (state.config.type) {
        case 'query':
          await resolveQueryVariable(state, preResolve)
          break
        case 'datasource':
          await resolveDatasourceVariable(state)
          break
        case 'interval':
          resolveIntervalVariable(state)
          break
        case 'constant':
          resolveConstantVariable(state)
          break
        case 'adhoc':
          // adhoc doesn't resolve options
          break
        // custom / textbox: keep existing options
      }
    }
  }

  // Escape special regex characters in a string
  function escapeRegex(s: string): string {
    return s.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')
  }

  // Core replacement logic on a raw string with a given states map.
  // Multi-pass resolution (FE4-8): resolves nested $var references iteratively.
  // e.g., $A → "${B}-suffix" → "$B" gets expanded in the next pass.
  // Stops when no more $var patterns are found or MAX_RESOLVE_DEPTH is reached.
  function replaceInString(input: string, stateMap: Map<string, VariableState>): string {
    let result = input
    const MAX_PASSES = 10

    for (let pass = 0; pass < MAX_PASSES; pass++) {
      let changed = false
      for (const [name, state] of stateMap) {
        const escaped = escapeRegex(name)
        const replacement = resolveValueForReplacement(state)
        const patterns = [
          new RegExp(`\\$${escaped}\\b`, 'g'),
          new RegExp(`\\$\\{${escaped}\\}`, 'g'),
          new RegExp(`\\[\\[${escaped}\\]\\]`, 'g'),
        ]
        for (const pattern of patterns) {
          const next = result.replace(pattern, replacement)
          if (next !== result) {
            changed = true
            result = next
          }
        }
      }
      if (!changed) break // no more substitutions — converged
    }
    return result
  }

  // Resolve the replacement value for a variable state
  function resolveValueForReplacement(state: VariableState): string {
    // Adhoc variables are injected via a separate mechanism
    if (state.config.type === 'adhoc') return ''

    const val = state.value

    // Multi-select
    if (state.config.multi && Array.isArray(val)) {
      // Check if __all is selected
      if (val.includes(ALL_SENTINEL)) {
        return state.config.allValue || '.*'
      }
      // PromQL regex syntax: val1|val2|val3
      if (val.length === 0) return ''
      if (val.length === 1) return escapeRegex(val[0])
      return val.map(v => escapeRegex(v)).join('|')
    }

    // Single value with __all
    if (val === ALL_SENTINEL) {
      return state.config.allValue || '.*'
    }

    return Array.isArray(val) ? val.join('|') : val
  }

  // Replace $var and [[var]] in a string
  function replaceVariables(input: string): string {
    let result = replaceInString(input, states.value)
    // Built-in time variables
    result = result.replace(/\$__from/g, String(Math.floor(timeRange.value.start / 1000)))
    result = result.replace(/\$__to/g, String(Math.floor(timeRange.value.end / 1000)))
    result = result.replace(/\$__interval/g, autoInterval(timeRange.value))
    return result
  }

  function setValue(name: string, value: string | string[]) {
    const state = states.value.get(name)
    if (state) state.value = value
  }

  function setMultiValue(name: string, values: string[]) {
    const state = states.value.get(name)
    if (state && state.config.multi) {
      state.value = values
    }
  }

  function addAdhocFilter(name: string, filter: AdhocFilter) {
    const filters = adhocFilters.value.get(name) || []
    filters.push(filter)
    adhocFilters.value.set(name, [...filters])
  }

  function removeAdhocFilter(name: string, index: number) {
    const filters = adhocFilters.value.get(name)
    if (filters) {
      filters.splice(index, 1)
      adhocFilters.value.set(name, [...filters])
    }
  }

  watch(variables, initStates, { immediate: true, deep: true })

  const variableList = computed(() =>
    Array.from(states.value.values())
  )

  return {
    states,
    variableList,
    adhocFilters,
    resolveAll,
    replaceVariables,
    setValue,
    setMultiValue,
    addAdhocFilter,
    removeAdhocFilter,
  }
}

function autoInterval(tr: TimeRange): string {
  return computeTimeStep((tr.end - tr.start) / 1000)
}
