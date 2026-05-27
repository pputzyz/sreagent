<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref, shallowRef, watch, type Component } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { NIcon } from 'naive-ui'
import {
  SettingsOutline,
  HardwareChipOutline,
  SparklesOutline,
  CodeOutline,
} from '@vicons/ionicons5'
import AISettings from '@/pages/settings/AISettings.vue'
import LLMConfigs from '@/pages/platform/LLMConfigs.vue'
import MCPServers from '@/pages/platform/MCPServers.vue'
import SkillManager from '@/pages/ai/SkillManager.vue'
import PageHeader from '@/components/common/PageHeader.vue'

const { t } = useI18n()
const route = useRoute()
const router = useRouter()

type NavKey = 'settings' | 'llm' | 'mcp' | 'skills'
const VALID: NavKey[] = ['settings', 'llm', 'mcp', 'skills']

const COMPONENTS: Record<NavKey, Component> = {
  settings: AISettings,
  llm: LLMConfigs,
  mcp: MCPServers,
  skills: SkillManager,
}

interface NavItem {
  key: NavKey
  icon: Component
  label: string
  desc: string
}

const groups = computed<Array<{ eyebrow: string; items: NavItem[] }>>(() => [
  {
    eyebrow: 'SETTINGS',
    items: [
      { key: 'settings', icon: SettingsOutline, label: t('menu.aiConfig'), desc: t('aiSettings.subtitle') },
      { key: 'llm', icon: HardwareChipOutline, label: t('menu.llmConfigs'), desc: t('llmConfigs.subtitle') || t('menu.llmConfigs') },
    ],
  },
  {
    eyebrow: 'EXTENSIONS',
    items: [
      { key: 'mcp', icon: SparklesOutline, label: t('menu.mcpServers'), desc: t('mcpServers.subtitle') || t('menu.mcpServers') },
      { key: 'skills', icon: CodeOutline, label: t('menu.aiSkills'), desc: t('aiSkills.subtitle') || t('menu.aiSkills') },
    ],
  },
])

function readKey(): NavKey {
  const hash = (route.hash || '').replace(/^#/, '')
  if (VALID.includes(hash as NavKey)) return hash as NavKey
  const q = (route.query.tab as string) || ''
  if (VALID.includes(q as NavKey)) return q as NavKey
  return 'settings'
}

const active = ref<NavKey>(readKey())
const currentComponent = shallowRef(COMPONENTS[active.value])

function go(key: NavKey) {
  if (active.value === key) return
  active.value = key
  router.replace({ path: '/platform/ai-config', hash: '#' + key })
}

watch(active, (k) => {
  currentComponent.value = COMPONENTS[k]
})

watch(() => route.fullPath, () => {
  const k = readKey()
  if (k !== active.value) active.value = k
})

const currentItem = computed<NavItem | undefined>(() => {
  for (const g of groups.value) {
    const f = g.items.find((i) => i.key === active.value)
    if (f) return f
  }
  return undefined
})

function onHash() {
  const k = readKey()
  if (k !== active.value) active.value = k
}
onMounted(() => window.addEventListener('hashchange', onHash))
onUnmounted(() => window.removeEventListener('hashchange', onHash))
</script>

<template>
  <div class="aic-shell">
    <PageHeader :title="t('menu.aiConfig')" :subtitle="t('aiConfig.subtitle')" />

    <div class="aic-body">
      <aside class="aic-aside">
        <nav v-for="g in groups" :key="g.eyebrow" class="aic-group">
          <div class="sre-label-eyebrow aic-eyebrow">{{ g.eyebrow }}</div>
          <button
            v-for="it in g.items"
            :key="it.key"
            type="button"
            class="aic-nav-item"
            :class="{ active: active === it.key }"
            @click="go(it.key)"
          >
            <span class="aic-nav-marker" />
            <NIcon :component="it.icon" :size="16" class="aic-nav-icon" />
            <span class="aic-nav-label">{{ it.label }}</span>
          </button>
        </nav>
      </aside>

      <section class="aic-content">
        <div v-if="currentItem" class="aic-sub-header">
          <h2 class="aic-sub-title">{{ currentItem.label }}</h2>
          <p class="aic-sub-desc">{{ currentItem.desc }}</p>
        </div>
        <div class="aic-pane">
          <component :is="currentComponent" />
        </div>
      </section>
    </div>
  </div>
</template>

<style scoped>
.aic-shell {
  font-family: var(--sre-font-sans, var(--sre-font-sans), system-ui, sans-serif);
  max-width: 1400px;
}
.aic-header {
  padding: 4px 4px 18px;
}
.aic-title {
  font-size: 22px;
  font-weight: 600;
  letter-spacing: -0.01em;
  color: var(--sre-text-primary);
  margin: 0 0 4px;
}
.aic-subtitle {
  font-size: 13px;
  color: var(--sre-text-secondary);
  margin: 0;
}
.aic-body {
  display: flex;
  align-items: stretch;
  gap: 0;
  background: var(--sre-bg-card);
  border: var(--sre-hairline);
  border-radius: var(--sre-radius-md, 12px);
  overflow: hidden;
  min-height: calc(100vh - 220px);
}
.aic-aside {
  flex: 0 0 200px;
  width: 200px;
  border-right: var(--sre-hairline);
  padding: 18px 10px;
  display: flex;
  flex-direction: column;
  gap: 18px;
  background: var(--sre-bg-card);
}
.aic-group {
  display: flex;
  flex-direction: column;
  gap: 2px;
}
.aic-eyebrow {
  padding: 4px 10px 8px;
  color: var(--sre-text-tertiary);
}
.aic-nav-item {
  position: relative;
  display: flex;
  align-items: center;
  gap: 10px;
  width: 100%;
  background: transparent;
  border: none;
  padding: 8px 10px 8px 14px;
  border-radius: 6px;
  color: var(--sre-text-secondary);
  font-size: 13px;
  font-weight: 500;
  cursor: pointer;
  text-align: left;
  transition: background var(--sre-duration-fast, 120ms) var(--sre-ease-out, ease),
    color var(--sre-duration-fast, 120ms) var(--sre-ease-out, ease);
}
.aic-nav-item:hover {
  background: var(--sre-bg-hover);
  color: var(--sre-text-primary);
}
.aic-nav-marker {
  position: absolute;
  left: 4px;
  top: 8px;
  bottom: 8px;
  width: 2px;
  border-radius: 2px;
  background: transparent;
  transition: background var(--sre-duration-fast, 120ms) var(--sre-ease-out, ease);
}
.aic-nav-item.active {
  background: var(--sre-primary-soft);
  color: var(--sre-primary);
}
.aic-nav-item.active .aic-nav-marker {
  background: var(--sre-primary);
}
.aic-nav-item.active .aic-nav-icon {
  color: var(--sre-primary);
}
.aic-nav-icon {
  flex: 0 0 auto;
  color: var(--sre-text-tertiary);
}
.aic-nav-label {
  flex: 1;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.aic-content {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
}
.aic-sub-header {
  padding: 18px 32px 14px;
  border-bottom: var(--sre-hairline);
}
.aic-sub-title {
  font-size: 16px;
  font-weight: 600;
  margin: 0 0 2px;
  color: var(--sre-text-primary);
  letter-spacing: -0.005em;
}
.aic-sub-desc {
  font-size: 12px;
  color: var(--sre-text-tertiary);
  margin: 0;
}
.aic-pane {
  flex: 1;
  min-width: 0;
  padding: 24px 32px;
}
</style>
