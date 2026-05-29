import { defineStore } from 'pinia'
import { ref, watch } from 'vue'
import type { UserPreferences } from '@/types'
import { authApi } from '@/api'

const defaultPrefs: UserPreferences = {
  user_id: 0,
  theme: 'auto',
  language: 'zh-CN',
  timezone: Intl.DateTimeFormat().resolvedOptions().timeZone || 'Asia/Shanghai',
  default_time_range: '1h',
  notification_severities: 'critical,warning',
  ai_chat_mode: 'sidebar',
}

export const usePreferencesStore = defineStore('preferences', () => {
  const prefs = ref<UserPreferences>({ ...defaultPrefs })
  const loaded = ref(false)

  async function load() {
    try {
      const { data } = await authApi.getPreferences()
      if (data.data) {
        prefs.value = { ...defaultPrefs, ...data.data }
      }
    } catch (e) {
      console.warn('Failed to load preferences, using defaults', e)
    } finally {
      loaded.value = true
    }
  }

  async function update(patch: Partial<UserPreferences>) {
    try {
      const { data } = await authApi.updatePreferences(patch)
      if (data.data) {
        prefs.value = { ...prefs.value, ...data.data }
      }
    } catch (e) {
      console.warn('Failed to update preferences', e)
    }
  }

  function reset() {
    prefs.value = { ...defaultPrefs }
    loaded.value = false
  }

  // Apply theme to document — syncs with App.vue's isDark toggle.
  // App.vue is the single source of truth for theme; this writes the data-theme
  // attribute for CSS selectors that use [data-theme].
  function applyTheme() {
    const theme = prefs.value.theme
    if (theme === 'auto') {
      const prefersDark = window.matchMedia('(prefers-color-scheme: dark)').matches
      document.documentElement.setAttribute('data-theme', prefersDark ? 'dark' : 'light')
    } else {
      document.documentElement.setAttribute('data-theme', theme)
    }
    // Sync localStorage so App.vue's isDark picks it up on next load
    if (theme !== 'auto') {
      localStorage.setItem('sre-theme', theme)
    }
  }

  // Apply language to i18n
  function applyLanguage(locale: { value: string }) {
    if (prefs.value.language) {
      locale.value = prefs.value.language
    }
  }

  // Watch theme changes
  watch(() => prefs.value.theme, applyTheme)

  return {
    prefs,
    loaded,
    load,
    update,
    reset,
    applyTheme,
    applyLanguage,
  }
})
