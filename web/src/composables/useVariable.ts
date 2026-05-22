import { ref, watch, computed, type Ref } from 'vue'
import { datasourceApi } from '@/api'
import { computeTimeStep } from '@/utils/timeStep'
import type { VariableConfig } from '@/types/dashboard'
import type { TimeRange } from '@/types/query'

export interface VariableState {
  config: VariableConfig
  value: string
  options: string[]
  loading: boolean
}

export function useVariable(
  variables: Ref<VariableConfig[]>,
  timeRange: Ref<TimeRange>,
) {
  const states = ref<Map<string, VariableState>>(new Map())

  // Initialize states from config
  function initStates() {
    const newStates = new Map<string, VariableState>()
    for (const v of variables.value) {
      const existing = states.value.get(v.name)
      newStates.set(v.name, {
        config: v,
        value: existing?.value ?? v.defaultValue ?? '',
        options: existing?.options ?? v.options ?? [],
        loading: false,
      })
    }
    states.value = newStates
  }

  // Resolve a query-type variable
  async function resolveQueryVariable(state: VariableState) {
    if (!state.config.query || !state.config.datasourceId) return
    state.loading = true
    try {
      const res = await datasourceApi.query(state.config.datasourceId, {
        expression: state.config.query,
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
        } catch { /* ignore invalid regex */ }
      }
      // Apply sort
      if (state.config.sort === 'asc') opts.sort()
      else if (state.config.sort === 'desc') opts.sort().reverse()
      else if (state.config.sort === 'numerical-asc') opts.sort((a, b) => Number(a) - Number(b))
      else if (state.config.sort === 'numerical-desc') opts.sort((a, b) => Number(b) - Number(a))

      state.options = opts
      if (opts.length > 0 && !opts.includes(state.value)) {
        state.value = opts[0]
      }
    } catch {
      // keep existing options
    } finally {
      state.loading = false
    }
  }

  // Resolve all variables
  async function resolveAll() {
    for (const [, state] of states.value) {
      if (state.config.type === 'query') {
        await resolveQueryVariable(state)
      }
    }
  }

  // Escape special regex characters in a string
  function escapeRegex(s: string): string {
    return s.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')
  }

  // Replace $var and [[var]] in a string
  function replaceVariables(input: string): string {
    let result = input
    for (const [name, state] of states.value) {
      const escaped = escapeRegex(name)
      result = result.replace(new RegExp(`\\$${escaped}\\b`, 'g'), state.value)
      result = result.replace(new RegExp(`\\$\\{${escaped}\\}`, 'g'), state.value)
      result = result.replace(new RegExp(`\\[\\[${escaped}\\]\\]`, 'g'), state.value)
    }
    // Built-in time variables
    result = result.replace(/\$__from/g, String(Math.floor(timeRange.value.start / 1000)))
    result = result.replace(/\$__to/g, String(Math.floor(timeRange.value.end / 1000)))
    result = result.replace(/\$__interval/g, autoInterval(timeRange.value))
    return result
  }

  function setValue(name: string, value: string) {
    const state = states.value.get(name)
    if (state) state.value = value
  }

  watch(variables, initStates, { immediate: true, deep: true })

  const variableList = computed(() =>
    Array.from(states.value.values())
  )

  return { states, variableList, resolveAll, replaceVariables, setValue }
}

function autoInterval(tr: TimeRange): string {
  return computeTimeStep((tr.end - tr.start) / 1000)
}
