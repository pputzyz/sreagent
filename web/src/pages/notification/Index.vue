<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref, shallowRef, watch, type Component } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { NIcon } from 'naive-ui'
import {
  ChatbubblesOutline,
  SendOutline,
  GitNetworkOutline,
  NotificationsOutline,
  DocumentTextOutline,
} from '@vicons/ionicons5'
import AlertChannels from './AlertChannels.vue'
import Media from './Media.vue'
import Rules from './Rules.vue'
import Subscribe from './Subscribe.vue'
import Templates from './Templates.vue'
import PageHeader from '@/components/common/PageHeader.vue'

const { t } = useI18n()
const route = useRoute()
const router = useRouter()

type NavKey = 'channels' | 'media' | 'rules' | 'subscribe' | 'templates'
const VALID: NavKey[] = ['channels', 'media', 'rules', 'subscribe', 'templates']

const COMPONENTS: Record<NavKey, Component> = {
  channels: AlertChannels,
  media: Media,
  rules: Rules,
  subscribe: Subscribe,
  templates: Templates,
}

interface NavItem {
  key: NavKey
  icon: Component
  label: string
  desc: string
}

const groups = computed<Array<{ eyebrow: string; items: NavItem[] }>>(() => [
  {
    eyebrow: t('notification.groupRouting') || 'ROUTING',
    items: [
      { key: 'channels', icon: ChatbubblesOutline, label: t('menu.alertChannels'), desc: t('alertChannel.subtitle') },
      { key: 'media', icon: SendOutline, label: t('menu.notifyMedia'), desc: t('notification.mediaDesc') || t('menu.notifyMedia') },
      { key: 'rules', icon: GitNetworkOutline, label: t('menu.notifyRules'), desc: t('notification.rulesDesc') || t('menu.notifyRules') },
      { key: 'subscribe', icon: NotificationsOutline, label: t('menu.subscriptions'), desc: t('notification.subscribeDesc') || t('menu.subscriptions') },
    ],
  },
  {
    eyebrow: t('notification.groupDesign') || 'DESIGN',
    items: [
      { key: 'templates', icon: DocumentTextOutline, label: t('menu.templates'), desc: t('notification.templatesDesc') || t('menu.templates') },
    ],
  },
])

function readKey(): NavKey {
  const hash = (route.hash || '').replace(/^#/, '')
  if (VALID.includes(hash as NavKey)) return hash as NavKey
  const q = (route.query.tab as string) || ''
  if (VALID.includes(q as NavKey)) return q as NavKey
  return 'channels'
}

const active = ref<NavKey>(readKey())
const currentComponent = shallowRef(COMPONENTS[active.value])

function go(key: NavKey) {
  if (active.value === key) return
  active.value = key
  router.replace({ path: '/alert/notify/policies', hash: '#' + key })
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
  <div class="notif-shell">
    <PageHeader :title="t('menu.notification')" :subtitle="t('notification.subtitle')" />

    <div class="notif-body">
      <aside class="notif-aside">
        <nav v-for="g in groups" :key="g.eyebrow" class="notif-group">
          <div class="sre-label-eyebrow notif-eyebrow">{{ g.eyebrow }}</div>
          <button
            v-for="it in g.items"
            :key="it.key"
            type="button"
            class="notif-nav-item"
            :class="{ active: active === it.key }"
            @click="go(it.key)"
          >
            <span class="notif-nav-marker" />
            <NIcon :component="it.icon" :size="16" class="notif-nav-icon" />
            <span class="notif-nav-label">{{ it.label }}</span>
          </button>
        </nav>
      </aside>

      <section class="notif-content">
        <div v-if="currentItem" class="notif-sub-header">
          <h2 class="notif-sub-title">{{ currentItem.label }}</h2>
          <p class="notif-sub-desc">{{ currentItem.desc }}</p>
        </div>
        <div class="notif-pane">
          <component :is="currentComponent" />
        </div>
      </section>
    </div>
  </div>
</template>

<style scoped>
.notif-shell {
  font-family: var(--sre-font-sans, var(--sre-font-sans), system-ui, sans-serif);
  max-width: 1400px;
}
.notif-header {
  padding: 4px 4px 18px;
}
.notif-title {
  font-size: 22px;
  font-weight: 600;
  letter-spacing: -0.01em;
  color: var(--sre-text-primary);
  margin: 0 0 4px;
}
.notif-subtitle {
  font-size: 13px;
  color: var(--sre-text-secondary);
  margin: 0;
}
.notif-body {
  display: flex;
  align-items: stretch;
  gap: 0;
  background: var(--sre-bg-card);
  border: var(--sre-hairline);
  border-radius: var(--sre-radius-md, 12px);
  overflow: hidden;
  min-height: calc(100vh - 220px);
}
.notif-aside {
  flex: 0 0 200px;
  width: 200px;
  border-right: var(--sre-hairline);
  padding: 18px 10px;
  display: flex;
  flex-direction: column;
  gap: 18px;
  background: var(--sre-bg-card);
}
.notif-group {
  display: flex;
  flex-direction: column;
  gap: 2px;
}
.notif-eyebrow {
  padding: 4px 10px 8px;
  color: var(--sre-text-tertiary);
}
.notif-nav-item {
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
.notif-nav-item:hover {
  background: var(--sre-bg-hover);
  color: var(--sre-text-primary);
}
.notif-nav-marker {
  position: absolute;
  left: 4px;
  top: 8px;
  bottom: 8px;
  width: 2px;
  border-radius: 2px;
  background: transparent;
  transition: background var(--sre-duration-fast, 120ms) var(--sre-ease-out, ease);
}
.notif-nav-item.active {
  background: var(--sre-primary-soft);
  color: var(--sre-primary);
}
.notif-nav-item.active .notif-nav-marker {
  background: var(--sre-primary);
}
.notif-nav-item.active .notif-nav-icon {
  color: var(--sre-primary);
}
.notif-nav-icon {
  flex: 0 0 auto;
  color: var(--sre-text-tertiary);
}
.notif-nav-label {
  flex: 1;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.notif-content {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
}
.notif-sub-header {
  padding: 18px 32px 14px;
  border-bottom: var(--sre-hairline);
}
.notif-sub-title {
  font-size: 16px;
  font-weight: 600;
  margin: 0 0 2px;
  color: var(--sre-text-primary);
  letter-spacing: -0.005em;
}
.notif-sub-desc {
  font-size: 12px;
  color: var(--sre-text-tertiary);
  margin: 0;
}
.notif-pane {
  flex: 1;
  min-width: 0;
  padding: 24px 32px;
}
</style>
