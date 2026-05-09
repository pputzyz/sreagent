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
import AuroraBackground from '@/components/common/AuroraBackground.vue'

const savedTheme = localStorage.getItem('sre-theme')
const isDark = ref(savedTheme ? savedTheme === 'dark' : true)
const theme = computed(() => isDark.value ? darkTheme : null)

// --- v2.7 "Vibrant Glass + Clay" shared brand tokens ---
const common = {
  primaryColor:        '#10b981',
  primaryColorHover:   '#34d399',
  primaryColorPressed: '#059669',
  primaryColorSuppl:   '#34d399',
  errorColor:          '#ef4444',
  errorColorHover:     '#f87171',
  errorColorPressed:   '#dc2626',
  warningColor:        '#f59e0b',
  warningColorHover:   '#fbbf24',
  warningColorPressed: '#d97706',
  infoColor:           '#6366f1',
  infoColorHover:      '#818cf8',
  infoColorPressed:    '#4f46e5',
  successColor:        '#10b981',
  successColorHover:   '#34d399',
  successColorPressed: '#059669',
  borderRadius:        '12px',
  borderRadiusSmall:   '8px',
  fontFamily:
    '-apple-system, BlinkMacSystemFont, "Segoe UI", "PingFang SC", "Microsoft YaHei", Roboto, "Helvetica Neue", Arial, sans-serif',
  fontFamilyMono:
    '"JetBrains Mono", "SF Mono", "Fira Code", ui-monospace, Consolas, Menlo, monospace',
}

const darkOverrides: GlobalThemeOverrides = {
  common: {
    ...common,
    bodyColor:     '#090D1A',
    cardColor:     'rgba(255,255,255,0.035)',
    modalColor:    'rgba(18,22,36,0.95)',
    popoverColor:  'rgba(18,22,36,0.95)',
    tableColor:    'rgba(255,255,255,0.025)',
    tableColorHover: 'rgba(255,255,255,0.05)',
    borderColor:   'rgba(255,255,255,0.07)',
    dividerColor:  'rgba(255,255,255,0.07)',
    hoverColor:    'rgba(255,255,255,0.05)',
    textColorBase:      'rgba(255,255,255,0.93)',
    textColor1:         'rgba(255,255,255,0.93)',
    textColor2:         'rgba(255,255,255,0.60)',
    textColor3:         'rgba(255,255,255,0.38)',
    textColorDisabled:  'rgba(255,255,255,0.22)',
    placeholderColor:   'rgba(255,255,255,0.30)',
  },
  Card: {
    color:         'rgba(255,255,255,0.035)',
    colorEmbedded: 'rgba(255,255,255,0.025)',
    borderColor:   'rgba(255,255,255,0.07)',
    borderRadius:  '16px',
  },
  Button: {
    borderRadiusMedium: '12px',
    borderRadiusSmall:  '8px',
    borderRadiusTiny:   '6px',
    fontWeight:         '500',
  },
  DataTable: {
    thColor:           'rgba(255,255,255,0.03)',
    tdColor:           'rgba(255,255,255,0.02)',
    tdColorHover:      'rgba(255,255,255,0.05)',
    borderColor:       'rgba(255,255,255,0.06)',
    borderRadius:      '14px',
  },
  Layout: {
    color:       '#090D1A',
    siderColor:  'rgba(9,13,26,0.85)',
    headerColor: 'rgba(9,13,26,0.75)',
  },
  Modal:  { color: 'rgba(18,22,36,0.95)' },
  Drawer: { color: 'rgba(18,22,36,0.95)' },
  Tag:    { borderRadius: '8px' },
  Menu: {
    itemHeight:             '38px',
    borderRadius:           '10px',
    itemColorHover:         'rgba(255,255,255,0.05)',
    itemColorActive:        'rgba(16,185,129,0.14)',
    itemColorActiveHover:   'rgba(16,185,129,0.20)',
    itemTextColor:          'rgba(255,255,255,0.60)',
    itemTextColorHover:     'rgba(255,255,255,0.93)',
    itemTextColorActive:    '#34d399',
    itemIconColorActive:    '#34d399',
    itemIconColorActiveHover:'#34d399',
  },
  Tabs: {
    tabBorderRadius: '10px',
    tabPaddingSmall: '6px 12px',
  },
  Input: {
    borderRadius: '10px',
  },
  Switch: {
    railColorActive: '#10b981',
  },
  Slider: {
    fillColor: '#10b981',
  },
  Progress: {
    fillColor: '#10b981',
  },
  Popover: {
    color: 'rgba(18,22,36,0.95)',
  },
}

const lightOverrides: GlobalThemeOverrides = {
  common: {
    ...common,
    bodyColor:     '#f1f4f9',
    cardColor:     'rgba(255,255,255,0.82)',
    modalColor:    '#ffffff',
    popoverColor:  '#ffffff',
    tableColor:    '#ffffff',
    tableColorHover: 'rgba(15,23,42,0.03)',
    borderColor:   'rgba(15,23,42,0.08)',
    dividerColor:  'rgba(15,23,42,0.08)',
    hoverColor:    'rgba(15,23,42,0.04)',
    textColorBase: 'rgba(15,23,42,0.93)',
    textColor1:    'rgba(15,23,42,0.93)',
    textColor2:    'rgba(15,23,42,0.55)',
    textColor3:    'rgba(15,23,42,0.36)',
  },
  Card: {
    color:         'rgba(255,255,255,0.82)',
    colorEmbedded: '#f7f8fa',
    borderColor:   'rgba(15,23,42,0.08)',
    borderRadius:  '16px',
  },
  Button: {
    borderRadiusMedium: '12px',
    borderRadiusSmall:  '8px',
    borderRadiusTiny:   '6px',
    fontWeight:         '500',
  },
  DataTable: {
    tdColor:      '#ffffff',
    thColor:      '#f7f8fa',
    tdColorHover: 'rgba(15,23,42,0.03)',
    borderColor:  'rgba(15,23,42,0.06)',
    borderRadius: '14px',
  },
  Layout: {
    color:       '#f1f4f9',
    siderColor:  'rgba(255,255,255,0.75)',
    headerColor: 'rgba(255,255,255,0.65)',
  },
  Modal:  { color: '#ffffff' },
  Drawer: { color: '#ffffff' },
  Tag:    { borderRadius: '8px' },
  Menu: {
    itemHeight:             '38px',
    borderRadius:           '10px',
    itemColorHover:         'rgba(15,23,42,0.04)',
    itemColorActive:        'rgba(16,185,129,0.10)',
    itemColorActiveHover:   'rgba(16,185,129,0.15)',
    itemTextColor:          'rgba(15,23,42,0.55)',
    itemTextColorHover:     'rgba(15,23,42,0.93)',
    itemTextColorActive:    '#059669',
    itemIconColorActive:    '#059669',
    itemIconColorActiveHover:'#059669',
  },
  Tabs: {
    tabBorderRadius: '10px',
    tabPaddingSmall: '6px 12px',
  },
  Input: {
    borderRadius: '10px',
  },
  Switch: {
    railColorActive: '#10b981',
  },
  Slider: {
    fillColor: '#10b981',
  },
  Progress: {
    fillColor: '#10b981',
  },
  Select: {
    peers: {
      InternalSelectMenu: { color: '#ffffff' },
    },
  },
  Popover: {
    color: '#ffffff',
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
    <AuroraBackground />
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
