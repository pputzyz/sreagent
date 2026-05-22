import type { LabelMatcher } from '@/components/common/LabelMatcherEditor.vue'

/**
 * Convert a Record<string, string> of label matchers (e.g. { env: '=~prod', region: 'us' })
 * into an array of LabelMatcher objects for the LabelMatcherEditor component.
 */
export function recordToMatchers(record: Record<string, string> | undefined): LabelMatcher[] {
  return Object.entries(record || {}).map(([key, raw]) => {
    for (const op of ['!=', '=~', '!~'] as const) {
      if (raw.startsWith(op)) return { key, op, value: raw.slice(op.length) }
    }
    return { key, op: '=' as const, value: raw }
  })
}

/**
 * Convert an array of LabelMatcher objects back into a Record<string, string>
 * for API payloads.
 */
export function matchersToRecord(matchers: LabelMatcher[]): Record<string, string> {
  return Object.fromEntries((matchers || []).map(m => {
    const v = m.op === '=' ? m.value : `${m.op}${m.value}`
    return [m.key, v]
  }))
}
