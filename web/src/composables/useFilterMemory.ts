/**
 * Composable for persisting filter state to localStorage.
 *
 * Usage:
 *   const { restore, save } = useFilterMemory('alert-events')
 *   const search = ref(restore('search', ''))
 *   const severity = ref<string | null>(restore('severity', null))
 *   watch([search, severity], () => save({ search: search.value, severity: severity.value }))
 */
import { watch, onScopeDispose, type Ref, type UnwrapRef } from 'vue'

const STORAGE_PREFIX = 'sre.filters.'

/**
 * Creates a filter memory instance scoped to a page key.
 * @param pageKey Unique key for the page (e.g., 'alert-events', 'alert-rules')
 * @param debounceMs Debounce delay for saving (default 300ms)
 */
export function useFilterMemory(pageKey: string, debounceMs = 300) {
  const storageKey = STORAGE_PREFIX + pageKey

  /** Restore a single filter value from localStorage. Falls back to defaultValue. */
  function restore<T>(field: string, defaultValue: T): T {
    try {
      const raw = localStorage.getItem(storageKey)
      if (!raw) return defaultValue
      const data = JSON.parse(raw) as Record<string, unknown>
      if (field in data && data[field] !== undefined) {
        return data[field] as T
      }
    } catch {
      // corrupted storage, ignore
    }
    return defaultValue
  }

  /** Restore all stored fields as a partial object. */
  function restoreAll<T extends Record<string, unknown>>(): Partial<T> {
    try {
      const raw = localStorage.getItem(storageKey)
      if (!raw) return {}
      return JSON.parse(raw) as Partial<T>
    } catch {
      return {}
    }
  }

  let saveTimer: ReturnType<typeof setTimeout> | null = null

  /** Save the full filter state object to localStorage (debounced). */
  function save(state: Record<string, unknown>) {
    if (saveTimer) clearTimeout(saveTimer)
    saveTimer = setTimeout(() => {
      try {
        localStorage.setItem(storageKey, JSON.stringify(state))
      } catch {
        // storage full, ignore
      }
    }, debounceMs)
  }

  /**
   * Bind reactive refs to auto-persist.
   * Returns a stop function to cancel the watcher.
   */
  function bindRefs<T extends Record<string, Ref<unknown>>>(
    refs: T,
  ): () => void {
    // Build initial state and restore
    const keys = Object.keys(refs) as Array<keyof T>
    const stored = restoreAll()

    // Apply stored values to refs
    for (const key of keys) {
      const k = key as string
      if (k in stored) {
        const refObj = refs[key]
        ;(refObj as Ref).value = stored[k] as UnwrapRef<Ref>
      }
    }

    // Watch all refs and save on change
    const stop = watch(
      keys.map((k) => refs[k]),
      () => {
        const state: Record<string, unknown> = {}
        for (const key of keys) {
          state[key as string] = refs[key].value
        }
        save(state)
      },
      { deep: true },
    )

    // Auto-stop the watcher when the calling scope is disposed.
    onScopeDispose(() => {
      stop()
      if (saveTimer) {
        clearTimeout(saveTimer)
        saveTimer = null
      }
    })

    return stop
  }

  /** Clear stored filters for this page. */
  function clear() {
    localStorage.removeItem(storageKey)
  }

  return { restore, restoreAll, save, bindRefs, clear }
}

/**
 * Clear ALL stored filter memories across all pages.
 * Call this on logout to prevent stale filter state leaking between users.
 */
export function clearAllFilterMemories() {
  const keysToRemove: string[] = []
  for (let i = 0; i < localStorage.length; i++) {
    const key = localStorage.key(i)
    if (key?.startsWith(STORAGE_PREFIX)) {
      keysToRemove.push(key)
    }
  }
  keysToRemove.forEach(k => localStorage.removeItem(k))
}
