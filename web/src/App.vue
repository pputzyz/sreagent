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
const isDark = ref(savedTheme ? savedTheme === 'dark' : true)
const theme = computed(() => isDark.value ? darkTheme : null)

// --- v3.0 "Modern Dark" brand tokens ---
const common = {
  primaryColor:        '#22c55e',
  primaryColorHover:   '#4ade80',
  primaryColorPressed: '#16a34a',
  primaryColorSuppl:   '#4ade80',
  errorColor:          '#ef4444',
  errorColorHover:     '#f87171',
  errorColorPressed:   '#dc2626',
  warningColor:        '#f59e0b',
  warningColorHover:   '#fbbf24',
  warningColorPressed: '#d97706',
  infoColor:           '#6366f1',
  infoColorHover:      '#818cf8',
  infoColorPressed:    '#4f46e5',
  successColor:        '#22c55e',
  successColorHover:   '#4ade80',
  successColorPressed: '#16a34a',
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
    bodyColor:     '#0a0a0b',
    cardColor:     '#141416',
    modalColor:    '#141416',
    popoverColor:  '#141416',
    tableColor:    '#141416',
    tableColorHover: 'rgba(255,255,255,0.04)',
    borderColor:   'rgba(255,255,255,0.06)',
    dividerColor:  'rgba(255,255,255,0.06)',
    hoverColor:    'rgba(255,255,255,0.04)',
    textColorBase:      '#ededef',
    textColor1:         '#ededef',
    textColor2:         '#a0a0ab',
    textColor3:         '#63636e',
    textColorDisabled:  'rgba(255,255,255,0.22)',
    placeholderColor:   'rgba(255,255,255,0.25)',
  },
  Card: {
    color:         '#141416',
    colorEmbedded: '#0a0a0b',
    borderColor:   'rgba(255,255,255,0.06)',
    borderRadius:  '10px',
  },
  Button: {
    borderRadiusMedium: '8px',
    borderRadiusSmall:  '6px',
    borderRadiusTiny:   '4px',
    fontWeight:         '500',
  },
  DataTable: {
    thColor:           '#141416',
    tdColor:           '#0a0a0b',
    tdColorHover:      'rgba(255,255,255,0.04)',
    borderColor:       'rgba(255,255,255,0.06)',
    borderRadius:      '10px',
    thFontWeight:      '600',
  },
  Layout: {
    color:       '#0a0a0b',
    siderColor:  '#09090b',
    headerColor: '#09090b',
  },
  Modal:  { color: '#141416', borderRadius: '10px' },
  Drawer: { color: '#141416' },
  Tag:    { borderRadius: '6px' },
  Menu: {
    itemHeight:             '36px',
    borderRadius:           '8px',
    itemColorHover:         'rgba(255,255,255,0.04)',
    itemColorActive:        'rgba(34,197,94,0.12)',
    itemColorActiveHover:   'rgba(34,197,94,0.16)',
    itemTextColor:          '#a0a0ab',
    itemTextColorHover:     '#ededef',
    itemTextColorActive:    '#4ade80',
    itemIconColorActive:    '#4ade80',
    itemIconColorActiveHover:'#4ade80',
  },
  Tabs: {
    tabBorderRadius: '8px',
    tabPaddingSmall: '6px 12px',
  },
  Input: {
    borderRadius: '8px',
  },
  Switch: {
    railColorActive: '#22c55e',
  },
  Slider: {
    fillColor: '#22c55e',
  },
  Progress: {
    fillColor: '#22c55e',
  },
  Popover: {
    color: '#141416',
    borderRadius: '10px',
  },
  Tooltip: {
    color: '#1c1c1f',
    borderRadius: '6px',
  },
}

const lightOverrides: GlobalThemeOverrides = {
  common: {
    ...common,
    bodyColor:     '#fafafa',
    cardColor:     '#ffffff',
    modalColor:    '#ffffff',
    popoverColor:  '#ffffff',
    tableColor:    '#ffffff',
    tableColorHover: 'rgba(0,0,0,0.03)',
    borderColor:   'rgba(0,0,0,0.06)',
    dividerColor:  'rgba(0,0,0,0.06)',
    hoverColor:    'rgba(0,0,0,0.03)',
    textColorBase: '#18181b',
    textColor1:    '#18181b',
    textColor2:    '#52525b',
    textColor3:    '#a1a1aa',
  },
  Card: {
    color:         '#ffffff',
    colorEmbedded: '#f4f4f5',
    borderColor:   'rgba(0,0,0,0.06)',
    borderRadius:  '10px',
  },
  Button: {
    borderRadiusMedium: '8px',
    borderRadiusSmall:  '6px',
    borderRadiusTiny:   '4px',
    fontWeight:         '500',
  },
  DataTable: {
    tdColor:      '#ffffff',
    thColor:      '#f4f4f5',
    tdColorHover: 'rgba(0,0,0,0.03)',
    borderColor:  'rgba(0,0,0,0.06)',
    borderRadius: '10px',
    thFontWeight: '600',
  },
  Layout: {
    color:       '#fafafa',
    siderColor:  '#ffffff',
    headerColor: '#ffffff',
  },
  Modal:  { color: '#ffffff', borderRadius: '10px' },
  Drawer: { color: '#ffffff' },
  Tag:    { borderRadius: '6px' },
  Menu: {
    itemHeight:             '36px',
    borderRadius:           '8px',
    itemColorHover:         'rgba(0,0,0,0.03)',
    itemColorActive:        'rgba(34,197,94,0.10)',
    itemColorActiveHover:   'rgba(34,197,94,0.14)',
    itemTextColor:          '#52525b',
    itemTextColorHover:     '#18181b',
    itemTextColorActive:    '#16a34a',
    itemIconColorActive:    '#16a34a',
    itemIconColorActiveHover:'#16a34a',
  },
  Tabs: {
    tabBorderRadius: '8px',
    tabPaddingSmall: '6px 12px',
  },
  Input: {
    borderRadius: '8px',
  },
  Switch: {
    railColorActive: '#22c55e',
  },
  Slider: {
    fillColor: '#22c55e',
  },
  Progress: {
    fillColor: '#22c55e',
  },
  Select: {
    peers: {
      InternalSelectMenu: { color: '#ffffff' },
    },
  },
  Popover: {
    color: '#ffffff',
    borderRadius: '10px',
  },
  Tooltip: {
    color: '#18181b',
    borderRadius: '6px',
  },
}

const themeOverrides = computed<GlobalThemeOverrides>(() =>
  isDark.value ? darkOverrides : lightOverrides
)

function applyBodyClass(dark: boolean) {
  if (dark) {
    document.body.classList.remove('light-theme')
  } else {
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
