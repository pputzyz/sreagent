<script setup lang="ts">
import {
  NConfigProvider,
  NMessageProvider,
  NDialogProvider,
  NNotificationProvider,
  darkTheme,
} from 'naive-ui'
import type { GlobalThemeOverrides } from 'naive-ui'
import { ref, provide, watch, onMounted, computed } from 'vue'
import { usePreferencesStore } from '@/stores/preferences'
import { siteInfoApi } from '@/api' // P1-27: fetch site branding on app load

const preferencesStore = usePreferencesStore()

// Preferences store is the single source of truth for theme.
// Resolve 'auto' via prefers-color-scheme media query.
const isDark = ref(false)

function resolveTheme(): boolean {
  const theme = preferencesStore.prefs.theme
  if (theme === 'auto') {
    return window.matchMedia('(prefers-color-scheme: dark)').matches
  }
  return theme === 'dark'
}

// Initialize from preferences (or localStorage fallback for pre-login)
onMounted(() => {
  if (preferencesStore.loaded) {
    isDark.value = resolveTheme()
  } else {
    // Fallback for before preferences are loaded
    const savedTheme = localStorage.getItem('sre-theme')
    isDark.value = savedTheme === 'dark'
  }
})

// Watch preferences store theme changes (single source of truth)
watch(() => preferencesStore.prefs.theme, () => {
  isDark.value = resolveTheme()
})

// Watch system color scheme changes for 'auto' mode
const mql = window.matchMedia('(prefers-color-scheme: dark)')
mql.addEventListener('change', () => {
  if (preferencesStore.prefs.theme === 'auto') {
    isDark.value = mql.matches
  }
})

const theme = computed(() => isDark.value ? darkTheme : null)

// --- v6.0 brand tokens (teal) ---
// WCAG contrast notes for #0D9488:
//   On #ffffff (light bg): ratio ~3.7:1 — passes AA for large text (>=18px/14px bold), icons, and UI components
//   On #0a1018 (dark bg):  ratio ~3.5:1 — passes AA for large text and UI components
//   Used for active menu items (13px semibold), icons, and interactive accents — all qualifying as UI components
//   For body text on white, the secondary/tertiary colors provide >=4.5:1 contrast
const common = {
  primaryColor:        '#0D9488',
  primaryColorHover:   '#14B8A6',
  primaryColorPressed: '#0F766E',
  primaryColorSuppl:   '#14B8A6',
  errorColor:          '#ef4444',
  errorColorHover:     '#f87171',
  errorColorPressed:   '#dc2626',
  warningColor:        '#f59e0b',
  warningColorHover:   '#fbbf24',
  warningColorPressed: '#d97706',
  infoColor:           '#3b82f6',
  infoColorHover:      '#60a5fa',
  infoColorPressed:    '#2563eb',
  successColor:        '#10B981',
  successColorHover:   '#34D399',
  successColorPressed: '#059669',
  borderRadius:        '10px',   // matches --sre-radius-md in global.css
  borderRadiusSmall:   '8px',    // matches --sre-radius-sm in global.css
  fontFamily:
    '"Inter", "Segoe UI", -apple-system, BlinkMacSystemFont, "PingFang SC", "Hiragino Sans GB", "Microsoft YaHei", Roboto, "Helvetica Neue", Arial, sans-serif',
  fontFamilyMono:
    '"JetBrains Mono", "Cascadia Code", "SF Mono", "Consolas", "Menlo", ui-monospace, monospace',
}

const darkOverrides: GlobalThemeOverrides = {
  common: {
    ...common,
    bodyColor:     '#0a1018',
    cardColor:     'rgba(15,23,42,0.65)',
    modalColor:    '#0f172a',
    popoverColor:  '#0f172a',
    tableColor:    '#0a1018',
    tableColorHover: 'rgba(148,163,184,0.06)',
    borderColor:   'rgba(148,163,184,0.08)',
    dividerColor:  'rgba(148,163,184,0.08)',
    hoverColor:    'rgba(148,163,184,0.06)',
    textColorBase:      '#f1f5f9',
    textColor1:         '#f1f5f9',
    textColor2:         '#cbd5e1',
    textColor3:         '#94a3b8',
    textColorDisabled:  'rgba(148,163,184,0.3)',
    placeholderColor:   'rgba(148,163,184,0.35)',
  },
  Card: {
    color:         'rgba(15,23,42,0.65)',
    colorEmbedded: '#0a1018',
    borderColor:   'rgba(148,163,184,0.08)',
    borderRadius:  '12px',
  },
  Button: {
    borderRadiusMedium: '8px',
    borderRadiusSmall:  '6px',
    borderRadiusTiny:   '4px',
    fontWeight:         '500',
  },
  DataTable: {
    thColor:           '#0f172a',
    tdColor:           '#0a1018',
    tdColorHover:      'rgba(148,163,184,0.06)',
    borderColor:       'rgba(148,163,184,0.08)',
    borderRadius:      '12px',
    thFontWeight:      '600',
  },
  Layout: {
    color:       '#0a1018',
    siderColor:  '#0a1018',
    headerColor: '#0a1018',
  },
  Modal:  { color: '#0f172a', borderRadius: '12px' },
  Drawer: { color: '#0f172a' },
  Tag:    { borderRadius: '6px' },
  Menu: {
    itemHeight:             '36px',
    borderRadius:           '8px',
    itemColorHover:         'rgba(148,163,184,0.06)',
    itemColorActive:        'rgba(13,148,136,0.12)',
    itemColorActiveHover:   'rgba(13,148,136,0.16)',
    itemTextColor:          '#cbd5e1',
    itemTextColorHover:     '#f1f5f9',
    itemTextColorActive:    '#0D9488',
    itemIconColorActive:    '#0D9488',
    itemIconColorActiveHover:'#0D9488',
  },
  Tabs: {
    tabBorderRadius: '8px',
    tabPaddingSmall: '6px 12px',
  },
  Input: {
    borderRadius: '8px',
  },
  Switch: {
    railColorActive: '#0D9488',
  },
  Slider: {
    fillColor: '#0D9488',
  },
  Progress: {
    fillColor: '#0D9488',
  },
  Popover: {
    color: '#0f172a',
    borderRadius: '12px',
  },
  Tooltip: {
    color: '#1e293b',
    borderRadius: '6px',
  },
}

const lightOverrides: GlobalThemeOverrides = {
  common: {
    ...common,
    bodyColor:     '#FAFAF9',
    cardColor:     '#ffffff',
    modalColor:    '#ffffff',
    popoverColor:  '#ffffff',
    tableColor:    '#ffffff',
    tableColorHover: 'rgba(0,0,0,0.03)',
    borderColor:   'rgba(0,0,0,0.06)',
    dividerColor:  'rgba(0,0,0,0.06)',
    hoverColor:    'rgba(0,0,0,0.03)',
    textColorBase: '#1C1917',
    textColor1:    '#1C1917',
    textColor2:    '#57534E',
    textColor3:    '#78716C',
  },
  Card: {
    color:         '#ffffff',
    colorEmbedded: '#f5f5f4',
    borderColor:   'rgba(0,0,0,0.06)',
    borderRadius:  '12px',
  },
  Button: {
    borderRadiusMedium: '8px',
    borderRadiusSmall:  '6px',
    borderRadiusTiny:   '4px',
    fontWeight:         '500',
  },
  DataTable: {
    tdColor:      '#ffffff',
    thColor:      '#f5f5f4',
    tdColorHover: 'rgba(0,0,0,0.03)',
    borderColor:  'rgba(0,0,0,0.06)',
    borderRadius: '12px',
    thFontWeight: '600',
  },
  Layout: {
    color:       '#f8fafc',
    siderColor:  '#ffffff',
    headerColor: '#ffffff',
  },
  Modal:  { color: '#ffffff', borderRadius: '12px' },
  Drawer: { color: '#ffffff' },
  Tag:    { borderRadius: '6px' },
  Menu: {
    itemHeight:             '36px',
    borderRadius:           '8px',
    itemColorHover:         'rgba(0,0,0,0.03)',
    itemColorActive:        'rgba(13,148,136,0.10)',
    itemColorActiveHover:   'rgba(13,148,136,0.14)',
    itemTextColor:          '#6b7280',
    itemTextColorHover:     '#111827',
    itemTextColorActive:    '#0D9488',
    itemIconColorActive:    '#0D9488',
    itemIconColorActiveHover:'#0D9488',
  },
  Tabs: {
    tabBorderRadius: '8px',
    tabPaddingSmall: '6px 12px',
  },
  Input: {
    borderRadius: '8px',
  },
  Switch: {
    railColorActive: '#0D9488',
  },
  Slider: {
    fillColor: '#0D9488',
  },
  Progress: {
    fillColor: '#0D9488',
  },
  Select: {
    peers: {
      InternalSelectMenu: { color: '#ffffff' },
    },
  },
  Popover: {
    color: '#ffffff',
    borderRadius: '12px',
  },
  Tooltip: {
    color: '#111827',
    borderRadius: '6px',
  },
}

const themeOverrides = computed<GlobalThemeOverrides>(() =>
  isDark.value ? darkOverrides : lightOverrides
)

function applyBodyClass(dark: boolean) {
  if (dark) {
    document.body.classList.add('dark-theme')
    document.body.classList.remove('light-theme')
  } else {
    document.body.classList.remove('dark-theme')
    document.body.classList.add('light-theme')
  }
}

onMounted(() => {
  applyBodyClass(isDark.value)
  // P1-27: Fetch site branding on app load and apply globally
  fetchSiteBranding()
})

// P1-27: Apply site info (title, favicon, custom CSS) globally
async function fetchSiteBranding() {
  try {
    const { data } = await siteInfoApi.get()
    const info = data.data
    if (!info) return
    // Apply document title
    if (info.site_name) {
      document.title = info.site_name
    }
    // Apply favicon
    if (info.favicon_url) {
      let link = document.querySelector("link[rel~='icon']") as HTMLLinkElement
      if (!link) {
        link = document.createElement('link')
        link.rel = 'icon'
        document.head.appendChild(link)
      }
      link.href = info.favicon_url
    }
    // Apply custom CSS
    if (info.custom_css) {
      let style = document.getElementById('site-custom-css')
      if (!style) {
        style = document.createElement('style')
        style.id = 'site-custom-css'
        document.head.appendChild(style)
      }
      style.textContent = info.custom_css
    }
    // Store for login page access
    if (info.site_name) {
      localStorage.setItem('sre-site-name', info.site_name)
    }
    if (info.login_title) {
      localStorage.setItem('sre-login-title', info.login_title)
    }
    if (info.logo_url) {
      localStorage.setItem('sre-logo-url', info.logo_url)
    }
  } catch {
    // Site info endpoint may not be available — ignore
  }
}

watch(isDark, (val) => {
  localStorage.setItem('sre-theme', val ? 'dark' : 'light')
  applyBodyClass(val)
})

// Toggle writes back to preferences store (single source of truth)
provide('toggleTheme', () => {
  const newTheme = isDark.value ? 'light' : 'dark'
  preferencesStore.update({ theme: newTheme })
})
provide('isDark', isDark)
</script>

<template>
  <NConfigProvider :theme="theme" :theme-overrides="themeOverrides">
    <NMessageProvider placement="top-right" :duration="2800" :max="4">
      <NDialogProvider>
        <NNotificationProvider placement="top-right" :max="4">
          <router-view />
        </NNotificationProvider>
      </NDialogProvider>
    </NMessageProvider>
  </NConfigProvider>
</template>

<style>
body {
  margin: 0;
  padding: 0;
}
</style>
