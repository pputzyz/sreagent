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
import { usePreferencesStore, ACCENT_COLORS } from '@/stores/preferences'
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

// --- v6.0 brand tokens (reactive to accent color) ---
// Derive Naive UI primary colors from the user's chosen accent color.
// Hover = lighten 10%, Pressed = darken 10% (approximation via hex manipulation).
const accentColor = computed(() => {
  const key = preferencesStore.prefs.accent_color || 'teal'
  return ACCENT_COLORS[key] || ACCENT_COLORS.teal
})

function lighten(hex: string, amount: number): string {
  const r = parseInt(hex.slice(1, 3), 16)
  const g = parseInt(hex.slice(3, 5), 16)
  const b = parseInt(hex.slice(5, 7), 16)
  const lr = Math.min(255, Math.round(r + (255 - r) * amount))
  const lg = Math.min(255, Math.round(g + (255 - g) * amount))
  const lb = Math.min(255, Math.round(b + (255 - b) * amount))
  return `#${lr.toString(16).padStart(2, '0')}${lg.toString(16).padStart(2, '0')}${lb.toString(16).padStart(2, '0')}`
}

function darken(hex: string, amount: number): string {
  const r = parseInt(hex.slice(1, 3), 16)
  const g = parseInt(hex.slice(3, 5), 16)
  const b = parseInt(hex.slice(5, 7), 16)
  const dr = Math.round(r * (1 - amount))
  const dg = Math.round(g * (1 - amount))
  const db = Math.round(b * (1 - amount))
  return `#${dr.toString(16).padStart(2, '0')}${dg.toString(16).padStart(2, '0')}${db.toString(16).padStart(2, '0')}`
}

const common = computed(() => ({
  primaryColor:        accentColor.value.primary,
  primaryColorHover:   lighten(accentColor.value.primary, 0.15),
  primaryColorPressed: darken(accentColor.value.primary, 0.15),
  primaryColorSuppl:   lighten(accentColor.value.primary, 0.15),
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
  borderRadius:        '10px',
  borderRadiusSmall:   '8px',
  fontFamily:
    '"Inter", "Segoe UI", -apple-system, BlinkMacSystemFont, "PingFang SC", "Hiragino Sans GB", "Microsoft YaHei", Roboto, "Helvetica Neue", Arial, sans-serif',
  fontFamilyMono:
    '"JetBrains Mono", "Cascadia Code", "SF Mono", "Consolas", "Menlo", ui-monospace, monospace',
}))

const darkOverrides = computed<GlobalThemeOverrides>(() => ({
  common: {
    ...common.value,
    bodyColor:     '#1a1a2e',
    cardColor:     '#22223a',
    modalColor:    '#2a2a45',
    popoverColor:  '#2a2a45',
    tableColor:    '#1a1a2e',
    tableColorHover: 'rgba(148,163,184,0.06)',
    borderColor:   'rgba(148,163,184,0.08)',
    dividerColor:  'rgba(148,163,184,0.08)',
    hoverColor:    'rgba(148,163,184,0.06)',
    textColorBase:      '#f1f5f9',
    textColor1:         '#f1f5f9',
    textColor2:         '#94a3b8',
    textColor3:         '#64748b',
    textColorDisabled:  'rgba(148,163,184,0.3)',
    placeholderColor:   'rgba(148,163,184,0.35)',
  },
  Card: {
    color:         '#22223a',
    colorEmbedded: '#1a1a2e',
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
    thColor:           '#16162a',
    tdColor:           '#1a1a2e',
    tdColorHover:      'rgba(148,163,184,0.06)',
    borderColor:       'rgba(148,163,184,0.08)',
    borderRadius:      '12px',
    thFontWeight:      '600',
  },
  Layout: {
    color:       '#1a1a2e',
    siderColor:  '#1a1a2e',
    headerColor: '#1a1a2e',
  },
  Modal:  { color: '#2a2a45', borderRadius: '12px' },
  Drawer: { color: '#2a2a45' },
  Tag:    { borderRadius: '6px' },
  Menu: {
    itemHeight:             '36px',
    borderRadius:           '8px',
    itemColorHover:         'rgba(148,163,184,0.06)',
    itemColorActive:        accentColor.value.soft,
    itemColorActiveHover:   accentColor.value.soft,
    itemTextColor:          '#94a3b8',
    itemTextColorHover:     '#f1f5f9',
    itemTextColorActive:    accentColor.value.primary,
    itemIconColorActive:    accentColor.value.primary,
    itemIconColorActiveHover: accentColor.value.primary,
  },
  Tabs: {
    tabBorderRadius: '8px',
    tabPaddingSmall: '6px 12px',
  },
  Input: {
    borderRadius: '8px',
  },
  Switch: {
    railColorActive: accentColor.value.primary,
  },
  Slider: {
    fillColor: accentColor.value.primary,
  },
  Progress: {
    fillColor: accentColor.value.primary,
  },
  Popover: {
    color: '#2a2a45',
    borderRadius: '12px',
  },
  Tooltip: {
    color: '#2a2a45',
    borderRadius: '6px',
  },
}))

const lightOverrides = computed<GlobalThemeOverrides>(() => ({
  common: {
    ...common.value,
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
    itemColorActive:        accentColor.value.soft,
    itemColorActiveHover:   accentColor.value.soft,
    itemTextColor:          '#6b7280',
    itemTextColorHover:     '#111827',
    itemTextColorActive:    accentColor.value.primary,
    itemIconColorActive:    accentColor.value.primary,
    itemIconColorActiveHover: accentColor.value.primary,
  },
  Tabs: {
    tabBorderRadius: '8px',
    tabPaddingSmall: '6px 12px',
  },
  Input: {
    borderRadius: '8px',
  },
  Switch: {
    railColorActive: accentColor.value.primary,
  },
  Slider: {
    fillColor: accentColor.value.primary,
  },
  Progress: {
    fillColor: accentColor.value.primary,
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
}))

const themeOverrides = computed<GlobalThemeOverrides>(() =>
  isDark.value ? darkOverrides.value : lightOverrides.value
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
