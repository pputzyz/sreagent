<script setup lang="ts">
import { ref, shallowRef, computed, onMounted, markRaw, type Component } from 'vue'
import { useI18n } from 'vue-i18n'
import { NIcon } from 'naive-ui'
import {
  HardwareChipOutline,
  ChatbubblesOutline,
  MailOutline,
  KeyOutline,
  PeopleOutline,
  BusinessOutline,
  GridOutline,
  PersonAddOutline,
  DocumentTextOutline,
  ShieldCheckmarkOutline,
} from '@vicons/ionicons5'
import UserManagement from './UserManagement.vue'
import TeamManagement from './TeamManagement.vue'
import VirtualUsers from './VirtualUsers.vue'
import BizGroupManagement from './BizGroupManagement.vue'
import AIConfig from './AIConfig.vue'
import LarkBotConfig from './LarkBotConfig.vue'
import OIDCConfig from './OIDCConfig.vue'
import SMTPConfig from './SMTPConfig.vue'
import AuditLog from './AuditLog.vue'
import SecurityConfig from './SecurityConfig.vue'

const { t } = useI18n()

interface SettingsNavItem {
  key: string
  label: string
  icon: Component
  component: Component
  desc: string
  title: string
}

interface SettingsNavGroup {
  label: string
  items: SettingsNavItem[]
}

const userMgmtRef = ref<InstanceType<typeof UserManagement> | null>(null)

const navGroups = shallowRef<SettingsNavGroup[]>([
  {
    label: 'PLATFORM',
    items: [
      {
        key: 'ai',
        label: t('settings.aiConfig'),
        title: t('settings.aiConfig'),
        icon: markRaw(HardwareChipOutline),
        component: markRaw(AIConfig),
        desc: 'Configure language model providers and credentials',
      },
      {
        key: 'larkbot',
        label: t('settings.larkBot'),
        title: t('settings.larkBot'),
        icon: markRaw(ChatbubblesOutline),
        component: markRaw(LarkBotConfig),
        desc: 'Lark bot integration and webhook callbacks',
      },
      {
        key: 'smtp',
        label: t('settings.smtpConfig'),
        title: t('settings.smtpConfig'),
        icon: markRaw(MailOutline),
        component: markRaw(SMTPConfig),
        desc: 'Outbound email server settings',
      },
      {
        key: 'oidc',
        label: t('settings.oidcConfig'),
        title: t('settings.oidcConfig'),
        icon: markRaw(KeyOutline),
        component: markRaw(OIDCConfig),
        desc: 'Single sign-on and identity provider',
      },
      {
        key: 'security',
        label: t('settings.securityConfig'),
        title: t('settings.securityConfig'),
        icon: markRaw(ShieldCheckmarkOutline),
        component: markRaw(SecurityConfig),
        desc: 'JWT expiry, password policy and session controls',
      },
    ],
  },
  {
    label: 'ORGANIZATION',
    items: [
      {
        key: 'users',
        label: t('settings.userManagement'),
        title: t('settings.userManagement'),
        icon: markRaw(PeopleOutline),
        component: markRaw(UserManagement),
        desc: 'Manage platform users, roles and credentials',
      },
      {
        key: 'teams',
        label: t('settings.teamManagement'),
        title: t('settings.teamManagement'),
        icon: markRaw(BusinessOutline),
        component: markRaw(TeamManagement),
        desc: 'Group users into teams for on-call rotation',
      },
      {
        key: 'bizgroups',
        label: t('bizGroup.title'),
        title: t('bizGroup.title'),
        icon: markRaw(GridOutline),
        component: markRaw(BizGroupManagement),
        desc: 'Business group scopes for alert rules',
      },
      {
        key: 'virtual',
        label: t('settings.virtualUsers'),
        title: t('settings.virtualUsers'),
        icon: markRaw(PersonAddOutline),
        component: markRaw(VirtualUsers),
        desc: 'External contacts that receive notifications',
      },
    ],
  },
  {
    label: 'AUDIT',
    items: [
      {
        key: 'audit',
        label: t('settings.auditLog'),
        title: t('settings.auditLog'),
        icon: markRaw(DocumentTextOutline),
        component: markRaw(AuditLog),
        desc: 'Operational audit trail of administrative actions',
      },
    ],
  },
])

const allItems = computed<SettingsNavItem[]>(() =>
  navGroups.value.flatMap((g) => g.items),
)

function readHashKey(): string {
  const raw = (window.location.hash || '').replace(/^#/, '').trim()
  if (raw && allItems.value.some((i) => i.key === raw)) return raw
  return 'ai'
}

const activeKey = ref<string>('ai')
const transitionKey = ref(0)

const activeItem = computed<SettingsNavItem>(
  () =>
    allItems.value.find((i) => i.key === activeKey.value) ??
    allItems.value[0],
)

function selectItem(key: string) {
  if (key === activeKey.value) return
  activeKey.value = key
  transitionKey.value++
  history.replaceState(null, '', `#${key}`)
}

onMounted(() => {
  activeKey.value = readHashKey()
  window.addEventListener('hashchange', () => {
    const next = readHashKey()
    if (next !== activeKey.value) {
      activeKey.value = next
      transitionKey.value++
    }
  })
})

const teamsUsersList = computed(() => userMgmtRef.value?.usersList ?? [])
</script>

<template>
  <div class="settings-page">
    <aside class="settings-nav">
      <div class="sre-stagger">
        <template v-for="group in navGroups" :key="group.label">
          <div class="sre-label-eyebrow nav-group-label">{{ group.label }}</div>
          <a
            v-for="item in group.items"
            :key="item.key"
            class="nav-item"
            :class="{ active: activeKey === item.key }"
            :href="`#${item.key}`"
            @click.prevent="selectItem(item.key)"
          >
            <NIcon :component="item.icon" />
            <span>{{ item.label }}</span>
          </a>
        </template>
      </div>
    </aside>

    <section class="settings-content">
      <header class="content-header">
        <h1 class="content-title">{{ activeItem.title }}</h1>
        <p class="content-desc">{{ activeItem.desc }}</p>
      </header>

      <div :key="transitionKey" class="content-body">
        <UserManagement v-show="activeKey === 'users'" ref="userMgmtRef" />
        <TeamManagement
          v-if="activeKey === 'teams'"
          :all-users="teamsUsersList"
        />
        <BizGroupManagement
          v-else-if="activeKey === 'bizgroups'"
          :all-users="teamsUsersList"
        />
        <component
          v-else-if="activeKey !== 'users'"
          :is="activeItem.component"
        />
      </div>
    </section>
  </div>
</template>

<style scoped>
.settings-page {
  display: grid;
  grid-template-columns: 240px 1fr;
  min-height: calc(100vh - var(--header-h, 56px));
  background: var(--sre-bg-page);
}

.settings-nav {
  background: var(--sre-bg-card);
  border-right: var(--sre-hairline);
  padding: 16px 0 24px;
  position: sticky;
  top: var(--header-h, 56px);
  align-self: start;
  height: calc(100vh - var(--header-h, 56px));
  overflow-y: auto;
}

.nav-group-label {
  padding: 24px 20px 8px;
}
.nav-group-label:first-child {
  padding-top: 8px;
}

.nav-item {
  display: flex;
  align-items: center;
  gap: 10px;
  height: 36px;
  padding: 0 20px;
  color: var(--sre-text-secondary);
  font-size: 13px;
  font-weight: 500;
  cursor: pointer;
  text-decoration: none;
  border-left: 2px solid transparent;
  transition: background var(--sre-duration-fast) var(--sre-ease-out),
    color var(--sre-duration-fast) var(--sre-ease-out),
    border-color var(--sre-duration-fast) var(--sre-ease-out);
}
.nav-item:hover {
  background: var(--sre-bg-hover);
  color: var(--sre-text-primary);
}
.nav-item.active {
  background: var(--sre-primary-soft);
  color: var(--sre-primary);
  border-left-color: var(--sre-primary);
}
.nav-item :deep(.n-icon) {
  font-size: 16px;
  opacity: 0.85;
}
.nav-item.active :deep(.n-icon) {
  opacity: 1;
}

.settings-content {
  padding: 32px 36px;
  overflow-y: auto;
  min-width: 0;
}

.content-header {
  margin-bottom: 24px;
  padding-bottom: 20px;
  border-bottom: var(--sre-hairline);
}
.content-title {
  font-size: 22px;
  font-weight: 600;
  letter-spacing: -0.3px;
  margin: 0 0 4px;
  color: var(--sre-text-primary);
}
.content-desc {
  font-size: 13px;
  color: var(--sre-text-secondary);
  margin: 0;
}

.content-body {
  animation: fade-in 200ms var(--sre-ease-out, ease-out);
}

@keyframes fade-in {
  from {
    opacity: 0;
    transform: translateY(2px);
  }
  to {
    opacity: 1;
    transform: none;
  }
}

@media (max-width: 768px) {
  .settings-page {
    grid-template-columns: 1fr;
  }
  .settings-nav {
    position: static;
    height: auto;
    border-right: none;
    border-bottom: var(--sre-hairline);
  }
  .settings-content {
    padding: 24px 20px;
  }
}
</style>
