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

const savedTheme = localStorage.getItem('sre-theme')
const isDark = ref(savedTheme ? savedTheme === 'dark' : false)
const theme = computed(() => isDark.value ? darkTheme : null)

// --- v3.1 "Soft Warm SaaS" brand tokens (teal/amber) ---
const common = {
  primaryColor:        '#0d9488',
  primaryColorHover:   '#14b8a6',
  primaryColorPressed: '#0f766e',
  primaryColorSuppl:   '#14b8a6',
  errorColor:          '#ef4444',
  errorColorHover:     '#f87171',
  errorColorPressed:   '#dc2626',
  warningColor:        '#f59e0b',
  warningColorHover:   '#fbbf24',
  warningColorPressed: '#d97706',
  infoColor:           '#3b82f6',
  infoColorHover:      '#60a5fa',
  infoColorPressed:    '#2563eb',
  successColor:        '#0d9488',
  successColorHover:   '#14b8a6',
  successColorPressed: '#0f766e',
  borderRadius:        '8px',
  borderRadiusSmall:   '6px',
  fontFamily:
    '"Inter", -apple-system, BlinkMacSystemFont, "PingFang SC", "Hiragino Sans GB", "Microsoft YaHei", "Segoe UI", Roboto, "Helvetica Neue", Arial, sans-serif',
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
    itemTextColorActive:    '#14b8a6',
    itemIconColorActive:    '#14b8a6',
    itemIconColorActiveHover:'#14b8a6',
  },
  Tabs: {
    tabBorderRadius: '8px',
    tabPaddingSmall: '6px 12px',
  },
  Input: {
    borderRadius: '8px',
  },
  Switch: {
    railColorActive: '#0d9488',
  },
  Slider: {
    fillColor: '#0d9488',
  },
  Progress: {
    fillColor: '#0d9488',
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
    bodyColor:     '#f8fafc',
    cardColor:     '#ffffff',
    modalColor:    '#ffffff',
    popoverColor:  '#ffffff',
    tableColor:    '#ffffff',
    tableColorHover: 'rgba(0,0,0,0.03)',
    borderColor:   'rgba(0,0,0,0.06)',
    dividerColor:  'rgba(0,0,0,0.06)',
    hoverColor:    'rgba(0,0,0,0.03)',
    textColorBase: '#111827',
    textColor1:    '#111827',
    textColor2:    '#6b7280',
    textColor3:    '#9ca3af',
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
    itemTextColorActive:    '#0f766e',
    itemIconColorActive:    '#0f766e',
    itemIconColorActiveHover:'#0f766e',
  },
  Tabs: {
    tabBorderRadius: '8px',
    tabPaddingSmall: '6px 12px',
  },
  Input: {
    borderRadius: '8px',
  },
  Switch: {
    railColorActive: '#0d9488',
  },
  Slider: {
    fillColor: '#0d9488',
  },
  Progress: {
    fillColor: '#0d9488',
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

onMounted(() => { applyBodyClass(isDark.value) })

watch(isDark, (val) => {
  localStorage.setItem('sre-theme', val ? 'dark' : 'light')
  applyBodyClass(val)
})

provide('toggleTheme', () => { isDark.value = !isDark.value })
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
