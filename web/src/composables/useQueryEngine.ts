import { ref, type Ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { datasourceApi } from '@/api'
import { getErrorMessage } from '@/utils/format'
import { computeTimeStep } from '@/utils/timeStep'
import type { TimeRange, QueryTarget, QuerySeriesItem } from '@/types/query'

function autoStep(timeRange: TimeRange): string {
  return computeTimeStep((timeRange.end - timeRange.start) / 1000)
}

function generateId(): string {
  if (typeof crypto !== 'undefined' && crypto.randomUUID) {
    return crypto.randomUUID()
  }
  // Fallback for non-secure contexts (HTTP)
  return 'xxxxxxxx-xxxx-4xxx-yxxx-xxxxxxxxxxxx'.replace(/[xy]/g, c => {
    const r = Math.random() * 16 | 0
    return (c === 'x' ? r : (r & 0x3 | 0x8)).toString(16)
  })
}

export function createDefaultTarget(): QueryTarget {
  return {
    id: generateId(),
    datasourceId: null,
    expression: '',
    legendFormat: '',
    enabled: true,
    state: 'idle',
    resultType: null,
    series: [],
    error: null,
  }
}

export function useQueryEngine(timeRange: Ref<TimeRange>) {
  const { t } = useI18n()
  const targets = ref<QueryTarget[]>([createDefaultTarget()])
  const globalLoading = ref(false)
  let runningCount = 0

  function addTarget() {
    const last = targets.value[targets.value.length - 1]
    targets.value.push({
      ...createDefaultTarget(),
      datasourceId: last?.datasourceId ?? null,
    })
  }

  function removeTarget(id: string) {
    if (targets.value.length <= 1) return
    targets.value = targets.value.filter(t => t.id !== id)
  }

  function toggleTarget(id: string) {
    const target = targets.value.find(t => t.id === id)
    if (target) target.enabled = !target.enabled
  }

  function updateTarget(id: string, patch: Partial<QueryTarget>) {
    const target = targets.value.find(t => t.id === id)
    if (target) Object.assign(target, patch)
  }

  async function executeQuery(target: QueryTarget) {
    if (!target.datasourceId || !target.expression.trim()) return

    target.state = 'loading'
    target.error = null
    target.series = []
    target.resultType = null

    try {
      const tr = timeRange.value
      const durationMs = tr.end - tr.start
      const isRange = durationMs > 60000 // > 1min → range query

      if (isRange) {
        const step = autoStep(tr)
        const res = await datasourceApi.rangeQuery(target.datasourceId, {
          expression: target.expression,
          start: Math.floor(tr.start / 1000),
          end: Math.floor(tr.end / 1000),
          step,
        })
        const data = res.data.data
        const rt = data.result_type
        target.resultType = (rt === 'vector' || rt === 'matrix') ? rt : null
        target.series = data.series || []
      } else {
        const res = await datasourceApi.query(target.datasourceId, {
          expression: target.expression,
          time: tr.end / 1000,
        })
        const data = res.data.data
        const rt = data.result_type
        target.resultType = (rt === 'vector' || rt === 'matrix') ? rt : null
        target.series = data.series || []
      }

      target.state = 'idle'
    } catch (err: unknown) {
      target.state = 'error'
      target.error = getErrorMessage(err) || t('tooltip.queryFailed')
    }
  }

  async function executeAll() {
    runningCount++
    globalLoading.value = true
    try {
      const enabledTargets = targets.value.filter(t => t.enabled && t.datasourceId && t.expression.trim())
      await Promise.allSettled(enabledTargets.map(executeQuery))
    } finally {
      runningCount--
      if (runningCount <= 0) {
        runningCount = 0
        globalLoading.value = false
      }
    }
  }

  return {
    targets,
    globalLoading,
    addTarget,
    removeTarget,
    toggleTarget,
    updateTarget,
    executeAll,
    executeQuery,
  }
}
