import { ref, reactive, watch, computed, onBeforeUnmount } from 'vue'
import { onBeforeRouteLeave, type LocationQuery } from 'vue-router'
import { useMessage } from 'naive-ui'
import { useI18n } from 'vue-i18n'

interface UseConfigFormOptions<T extends Record<string, unknown>> {
  /** Load config from API — should return the config object */
  load: () => Promise<T>
  /** Save config to API — receives the full form object */
  save: (form: T) => Promise<void>
  /** Optional: test connection — runs AFTER save if form is dirty */
  test?: () => Promise<void>
  /** Keys that trigger auto-save on change (typically switch/toggle fields) */
  autoSaveKeys?: (keyof T)[]
  /** Debounce ms for auto-save. Default 400 */
  debounceMs?: number
}

export function useConfigForm<T extends Record<string, unknown>>(options: UseConfigFormOptions<T>) {
  const message = useMessage()
  const { t } = useI18n()

  const loading = ref(false)
  const saving = ref(false)
  const testing = ref(false)

  // The reactive form that the template binds to
  const form = reactive({} as T)

  // Snapshot of the last saved/loaded state for dirty detection
  let lastSaved: T | null = null

  // ─── Dirty detection ───
  function snapshot(): string {
    return JSON.stringify(form)
  }

  const isDirty = computed(() => {
    if (!lastSaved) return false
    return JSON.stringify(form) !== JSON.stringify(lastSaved)
  })

  // ─── Load ───
  async function load() {
    loading.value = true
    try {
      const data = await options.load()
      Object.assign(form, data)
      lastSaved = JSON.parse(JSON.stringify(data))
    } finally {
      loading.value = false
    }
  }

  // ─── Save ───
  async function save(): Promise<boolean> {
    saving.value = true
    try {
      await options.save({ ...form } as T)
      lastSaved = JSON.parse(JSON.stringify(form))
      message.success(t('common.savedSuccess'))
      return true
    } catch (err: unknown) {
      const msg = err instanceof Error ? err.message : String(err)
      message.error(msg)
      return false
    } finally {
      saving.value = false
    }
  }

  // ─── Save and Test ───
  async function saveAndTest() {
    if (!options.test) return
    // Save first if dirty
    if (isDirty.value) {
      const ok = await save()
      if (!ok) return
    }
    testing.value = true
    try {
      await options.test()
    } finally {
      testing.value = false
    }
  }

  // ─── Reset to last saved ───
  function reset() {
    if (lastSaved) {
      Object.assign(form, JSON.parse(JSON.stringify(lastSaved)))
    }
  }

  // ─── Debounced auto-save for toggle fields ───
  let debounceTimer: ReturnType<typeof setTimeout> | null = null

  function triggerAutoSave() {
    if (debounceTimer) clearTimeout(debounceTimer)
    debounceTimer = setTimeout(async () => {
      await save()
    }, options.debounceMs ?? 400)
  }

  // Watch autoSaveKeys for changes
  if (options.autoSaveKeys?.length) {
    for (const key of options.autoSaveKeys) {
      watch(
        () => form[key],
        () => { triggerAutoSave() },
      )
    }
  }

  onBeforeUnmount(() => {
    if (debounceTimer) clearTimeout(debounceTimer)
  })

  // ─── Route leave guard ───
  let confirmedLeave = false

  onBeforeRouteLeave((_to, _from) => {
    if (isDirty.value && !confirmedLeave) {
      // Naive UI's useMessage can't show dialogs here, so we use native confirm
      const ok = window.confirm(t('common.unsavedChanges') || 'You have unsaved changes. Leave anyway?')
      if (!ok) return false
    }
    return true
  })

  /** Call after a successful save to allow navigation without prompt */
  function confirmLeave() {
    confirmedLeave = true
  }

  return {
    form,
    loading,
    saving,
    testing,
    isDirty,
    load,
    save,
    saveAndTest,
    reset,
    confirmLeave,
  }
}
