<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { Shield, ShieldCheck, ShieldAlert } from 'lucide-vue-next'

const { t } = useI18n()

interface RoleDef {
  key: string
  nameKey: string
  descKey: string
  icon: typeof Shield
  color: string
  softBg: string
  perms: string[]
}

const roles: RoleDef[] = [
  {
    key: 'admin',
    nameKey: 'rolesModule.adminName',
    descKey: 'rolesModule.adminDesc',
    icon: ShieldAlert,
    color: '#ef4444',
    softBg: 'rgba(239, 68, 68, 0.08)',
    perms: [
      'rolesModule.permManageUsers',
      'rolesModule.permManageTeams',
      'rolesModule.permManageRoles',
      'rolesModule.permSystemConfig',
      'rolesModule.permManageRules',
      'rolesModule.permManageSchedules',
      'rolesModule.permManageChannels',
    ],
  },
  {
    key: 'team_lead',
    nameKey: 'rolesModule.teamLeadName',
    descKey: 'rolesModule.teamLeadDesc',
    icon: ShieldCheck,
    color: '#f59e0b',
    softBg: 'rgba(245, 158, 11, 0.08)',
    perms: [
      'rolesModule.permManageTeams',
      'rolesModule.permManageRules',
      'rolesModule.permManageSchedules',
      'rolesModule.permManageChannels',
      'rolesModule.permAcknowledgeAlerts',
      'rolesModule.permCreateResources',
    ],
  },
  {
    key: 'member',
    nameKey: 'rolesModule.memberName',
    descKey: 'rolesModule.memberDesc',
    icon: Shield,
    color: '#3b82f6',
    softBg: 'rgba(59, 130, 246, 0.08)',
    perms: [
      'rolesModule.permCreateResources',
      'rolesModule.permEditOwn',
      'rolesModule.permAcknowledgeAlerts',
    ],
  },
]
</script>

<template>
  <div class="page-container">
    <!-- Header -->
    <div class="roles-header">
      <h1 class="page-title">{{ t('rolesModule.title') }}</h1>
      <p class="page-subtitle">{{ t('rolesModule.subtitle') }}</p>
    </div>

    <!-- Role Cards Grid -->
    <div class="roles-grid stagger-card">
      <div
        v-for="role in roles"
        :key="role.key"
        class="role-card surface-card"
      >
        <!-- Top stripe -->
        <div class="role-stripe" :style="{ background: role.color }" />

        <div class="role-card-inner">
          <!-- Icon + Title -->
          <div class="role-header">
            <div class="role-icon" :style="{ background: role.softBg, color: role.color }">
              <component :is="role.icon" :size="22" />
            </div>
            <div class="role-title-group">
              <span class="role-name">{{ t(role.nameKey) }}</span>
              <span class="role-key text-mono">{{ role.key }}</span>
            </div>
          </div>

          <!-- Description -->
          <p class="role-desc">{{ t(role.descKey) }}</p>

          <!-- Permissions -->
          <div class="role-perms">
            <span class="role-perms-label eyebrow">{{ t('rolesModule.permissionCount', { n: role.perms.length }) }}</span>
            <div class="role-perm-list">
              <span
                v-for="perm in role.perms"
                :key="perm"
                class="role-perm-tag"
                :style="{ borderColor: role.color + '30', background: role.softBg, color: role.color }"
              >
                {{ t(perm) }}
              </span>
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.roles-header {
  padding: 24px 0 20px;
  animation: sre-fade-in 400ms var(--sre-ease-out) both;
}

.roles-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(320px, 1fr));
  gap: 16px;
}

.role-card {
  position: relative;
  border-radius: var(--sre-radius-xl);
  border: 1px solid var(--sre-border);
  background: var(--sre-bg-card);
  overflow: hidden;
  transition: border-color var(--sre-duration-fast) var(--sre-ease-out),
              box-shadow var(--sre-duration-fast) var(--sre-ease-out),
              transform var(--sre-duration-fast) var(--sre-ease-out);
}

.role-card:hover {
  border-color: var(--sre-border-strong);
  box-shadow: var(--sre-shadow-lift);
  transform: translateY(-2px);
}

.role-stripe {
  height: 3px;
  width: 100%;
}

.role-card-inner {
  padding: 20px 22px 22px;
  display: flex;
  flex-direction: column;
  gap: 14px;
}

.role-header {
  display: flex;
  align-items: center;
  gap: 14px;
}

.role-icon {
  width: 44px;
  height: 44px;
  border-radius: 12px;
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
}

.role-title-group {
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.role-name {
  font-size: var(--sre-fs-lg);
  font-weight: var(--sre-fw-semibold);
  color: var(--sre-text-primary);
  line-height: var(--sre-lh-tight);
}

.role-key {
  font-size: var(--sre-fs-xs);
  color: var(--sre-text-tertiary);
  letter-spacing: 0.02em;
}

.role-desc {
  font-size: var(--sre-fs-sm);
  color: var(--sre-text-secondary);
  line-height: var(--sre-lh-normal);
  margin: 0;
}

.role-perms {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.role-perms-label {
  font-size: var(--sre-fs-2xs);
  font-weight: var(--sre-fw-semibold);
  letter-spacing: 0.06em;
  text-transform: uppercase;
  color: var(--sre-text-tertiary);
}

.role-perm-list {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
}

.role-perm-tag {
  display: inline-flex;
  align-items: center;
  padding: 3px 10px;
  border-radius: var(--sre-radius-pill);
  border: 1px solid;
  font-size: var(--sre-fs-xs);
  font-weight: var(--sre-fw-medium);
  line-height: 1.5;
  white-space: nowrap;
}

/* Responsive */
@media (max-width: 768px) {
  .roles-grid {
    grid-template-columns: 1fr;
  }
}
</style>
