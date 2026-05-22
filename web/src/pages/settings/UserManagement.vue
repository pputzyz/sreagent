<script setup lang="ts">
import { computed, ref, onMounted, watch, h } from 'vue'
import {
  useMessage,
  NButton,
  NIcon,
  NInput,
  NRadioGroup,
  NRadioButton,
  NSwitch,
  NDropdown,
  NModal,
  NForm,
  NFormItem,
  NGrid,
  NGi,
  NSelect,
  NSpace,
} from 'naive-ui'
import { useI18n } from 'vue-i18n'
import { userApi } from '@/api'
import type { User } from '@/types'
import { useCrudPage } from '@/composables/useCrudPage'
import type { CrudApiModule } from '@/composables/useCrudPage'
import { AddOutline, EllipsisHorizontal, SearchOutline } from '@vicons/ionicons5'
import LoadingSkeleton from '@/components/common/LoadingSkeleton.vue'

const message = useMessage()
const { t } = useI18n()

const crud = useCrudPage<User>({
  api: userApi as unknown as CrudApiModule<User>,
  defaultForm: () => ({
    username: '',
    display_name: '',
    email: '',
    phone: '',
    role: 'member' as User['role'],
    password: '',
    is_active: true,
  }),
  i18nKeys: {
    created: 'settings.userCreated',
    updated: 'settings.userUpdated',
    deleted: 'settings.userDeleted',
    createTitle: 'settings.createUser',
    editTitle: 'settings.editUser',
  },
  rowToForm: (row) => ({
    username: row.username,
    display_name: row.display_name,
    email: row.email,
    phone: row.phone,
    role: row.role,
    password: '',
    is_active: row.is_active,
  }),
  validate: (form) => {
    if (!form.username?.trim()) return t('settings.usernameRequired')
    return null
  },
  formToPayload: (form) => ({
    username: form.username,
    display_name: form.display_name,
    email: form.email,
    phone: form.phone,
    role: form.role,
    is_active: form.is_active,
  }),
  pageSize: 200,
})

const {
  loading,
  items: usersList,
  search,
  showModal,
  modalTitle,
  editingId,
  saving,
  form,
  fetchList,
  openCreate,
  openEdit,
  handleSave,
} = crud

const filterRole = ref<'all' | 'admin' | 'team_lead' | 'member' | 'viewer'>('all')
const filterStatus = ref<'all' | 'active' | 'inactive'>('all')
const firstLoad = ref(true)
const passwordField = ref('')

watch(loading, (isLoading) => {
  if (!isLoading) firstLoad.value = false
})

watch(showModal, (v) => {
  if (v) passwordField.value = ''
})

const roleOptions = computed(() => [
  { label: t('settings.admin'), value: 'admin' as const },
  { label: t('settings.teamLead'), value: 'team_lead' as const },
  { label: t('settings.member'), value: 'member' as const },
  { label: t('settings.viewer'), value: 'viewer' as const },
])

function roleLabel(role: string): string {
  switch (role) {
    case 'admin': return t('settings.admin')
    case 'team_lead': return t('settings.teamLead')
    case 'member': return t('settings.member')
    case 'viewer': return t('settings.viewer')
    default: return role
  }
}

function initials(u: User): string {
  const s = (u.display_name || u.username || '?').trim()
  return s.charAt(0).toUpperCase()
}

const filtered = computed(() => {
  const q = search.value.trim().toLowerCase()
  return (usersList.value || []).filter(u => {
    if (u.user_type && u.user_type !== 'human') return false
    if (filterRole.value !== 'all' && u.role !== filterRole.value) return false
    if (filterStatus.value === 'active' && !u.is_active) return false
    if (filterStatus.value === 'inactive' && u.is_active) return false
    if (q) {
      const hay = `${u.username} ${u.display_name} ${u.email}`.toLowerCase()
      if (!hay.includes(q)) return false
    }
    return true
  })
})

// openCreate and openEdit are now provided by useCrudPage

// Custom handleSave to support password and changePassword
async function handleSaveUser() {
  if (!form.value.username?.trim()) {
    message.warning(t('settings.usernameRequired'))
    return
  }
  if (!editingId.value && !passwordField.value.trim()) {
    message.warning(t('settings.passwordRequired'))
    return
  }
  saving.value = true
  try {
    if (editingId.value) {
      await userApi.update(editingId.value, {
        username: form.value.username,
        display_name: form.value.display_name,
        email: form.value.email,
        phone: form.value.phone,
        role: form.value.role,
      } as Partial<User>)
      if (passwordField.value.trim()) {
        await userApi.changePassword(editingId.value, { password: passwordField.value })
      }
      message.success(t('settings.userUpdated'))
    } else {
      await userApi.create({
        username: form.value.username,
        display_name: form.value.display_name,
        email: form.value.email,
        phone: form.value.phone,
        role: form.value.role,
        password: passwordField.value,
        is_active: form.value.is_active,
      } as Partial<User> & { password?: string })
      message.success(t('settings.userCreated'))
    }
    showModal.value = false
    passwordField.value = ''
    await fetchList()
  } catch (err: unknown) {
    message.error((err as Error).message)
  } finally {
    saving.value = false
  }
}

async function toggleActive(u: User) {
  try {
    await userApi.toggleActive(u.id, !u.is_active)
    message.success(u.is_active ? t('settings.userDeactivated') : t('settings.userActivated'))
    fetchList()
  } catch (err: unknown) {
    message.error((err as Error).message)
  }
}

function rowMenuOptions(u: User) {
  return [
    { key: 'edit', label: t('common.edit') },
    { key: 'reset', label: t('settings.newPasswordKeep') },
    { key: 'toggle', label: u.is_active ? t('settings.deactivate') : t('settings.activate') },
  ]
}

function handleMenu(key: string, u: User) {
  if (key === 'edit') openEdit(u)
  else if (key === 'reset') openEdit(u)
  else if (key === 'toggle') toggleActive(u)
}

defineExpose({ usersList, fetchUsers: fetchList })

onMounted(fetchList)

const ellipsisIcon = () => h(NIcon, { component: EllipsisHorizontal })
</script>

<template>
  <div class="user-mgmt">
    <header class="page-header">
      <div>
        <h2 class="page-title">{{ t('settings.userManagement') }}</h2>
        <p class="page-subtitle">{{ t('settings.userManagementDesc') }}</p>
      </div>
      <NButton type="primary" size="small" @click="openCreate">
        <template #icon><NIcon :component="AddOutline" /></template>
        {{ t('settings.createUser') }}
      </NButton>
    </header>

    <div class="filter-bar">
      <div class="filter-group">
        <span class="sre-label-eyebrow">{{ t('settings.role') }}</span>
        <NRadioGroup v-model:value="filterRole" size="small">
          <NRadioButton value="all">{{ t('common.all') }}</NRadioButton>
          <NRadioButton value="admin">{{ t('settings.admin') }}</NRadioButton>
          <NRadioButton value="team_lead">{{ t('settings.teamLead') }}</NRadioButton>
          <NRadioButton value="member">{{ t('settings.member') }}</NRadioButton>
          <NRadioButton value="viewer">{{ t('settings.viewer') }}</NRadioButton>
        </NRadioGroup>
      </div>
      <div class="filter-group">
        <span class="sre-label-eyebrow">{{ t('common.status') }}</span>
        <NRadioGroup v-model:value="filterStatus" size="small">
          <NRadioButton value="all">{{ t('common.all') }}</NRadioButton>
          <NRadioButton value="active">{{ t('settings.active') }}</NRadioButton>
          <NRadioButton value="inactive">{{ t('settings.inactive') }}</NRadioButton>
        </NRadioGroup>
      </div>
      <NInput
        v-model:value="search"
        size="small"
        :placeholder="t('common.search')"
        clearable
        class="search-input"
      >
        <template #prefix><NIcon :component="SearchOutline" /></template>
      </NInput>
    </div>

    <LoadingSkeleton v-if="firstLoad && loading" :rows="6" variant="row" />
    <div v-else class="user-list sre-stagger">
      <div
        v-for="u in filtered"
        :key="u.id"
        class="sre-row-card user-row"
        :data-dim="!u.is_active || undefined"
      >
        <div class="user-avatar">{{ initials(u) }}</div>
        <div class="user-main">
          <div class="user-headline">
            <span class="user-name">{{ u.display_name || u.username }}</span>
            <span v-if="u.display_name" class="user-username">({{ u.username }})</span>
          </div>
          <div class="user-meta">
            <span class="user-role-chip" :data-role="u.role">{{ roleLabel(u.role) }}</span>
            <span class="sre-meta-divider"></span>
            <span>{{ u.email || '—' }}</span>
            <template v-if="u.phone">
              <span class="sre-meta-divider"></span>
              <span class="tnum">{{ u.phone }}</span>
            </template>
          </div>
          <div class="user-footer">
            <span class="sre-dot" :data-severity="u.is_active ? 'success' : null"></span>
            <span class="user-status">{{ u.is_active ? t('settings.active') : t('settings.inactive') }}</span>
          </div>
        </div>
        <div class="user-actions">
          <NSwitch
            :value="u.is_active"
            size="small"
            @update:value="() => toggleActive(u)"
          />
          <NDropdown
            trigger="click"
            :options="rowMenuOptions(u)"
            @select="(k: string) => handleMenu(k, u)"
          >
            <NButton size="tiny" quaternary :render-icon="ellipsisIcon" />
          </NDropdown>
        </div>
      </div>
      <div v-if="!loading && filtered.length === 0" class="empty-state">
        {{ t('settings.noUsers') }}
      </div>
    </div>

    <NModal v-model:show="showModal" preset="card" :title="modalTitle" style="width: 560px" :bordered="false">
      <NForm label-placement="top">
        <NGrid :x-gap="12" :cols="2">
          <NGi>
            <NFormItem :label="t('auth.username')" required>
              <NInput v-model:value="form.username" :placeholder="t('userMgmt.usernamePlaceholder')" :disabled="!!editingId" />
            </NFormItem>
          </NGi>
          <NGi>
            <NFormItem :label="t('settings.displayName')">
              <NInput v-model:value="form.display_name" :placeholder="t('userMgmt.displayNamePlaceholder')" />
            </NFormItem>
          </NGi>
        </NGrid>
        <NGrid :x-gap="12" :cols="2">
          <NGi>
            <NFormItem :label="t('settings.email')">
              <NInput v-model:value="form.email" :placeholder="t('userMgmt.emailPlaceholder')" />
            </NFormItem>
          </NGi>
          <NGi>
            <NFormItem :label="t('settings.phone')">
              <NInput v-model:value="form.phone" :placeholder="t('userMgmt.phonePlaceholder')" />
            </NFormItem>
          </NGi>
        </NGrid>
        <NGrid :x-gap="12" :cols="2">
          <NGi>
            <NFormItem :label="t('settings.role')">
              <NSelect v-model:value="form.role" :options="roleOptions" />
            </NFormItem>
          </NGi>
          <NGi>
            <NFormItem :label="editingId ? t('settings.newPasswordKeep') : t('auth.password')" :required="!editingId">
              <NInput
                v-model:value="passwordField"
                type="password"
                :placeholder="t('auth.enterPassword')"
                show-password-on="click"
              />
            </NFormItem>
          </NGi>
        </NGrid>
      </NForm>
      <template #action>
        <NSpace justify="end">
          <NButton @click="showModal = false">{{ t('common.cancel') }}</NButton>
          <NButton type="primary" :loading="saving" @click="handleSaveUser">
            {{ editingId ? t('common.update') : t('common.create') }}
          </NButton>
        </NSpace>
      </template>
    </NModal>
  </div>
</template>

<style scoped>
.user-mgmt { font-family: var(--sre-font-sans); }

.page-header {
  display: flex; align-items: flex-start; justify-content: space-between;
  padding-bottom: 16px;
  border-bottom: var(--sre-hairline);
  margin-bottom: 20px;
}
.page-title { font-size: 18px; font-weight: 600; margin: 0 0 4px; color: var(--sre-text-primary); }
.page-subtitle { font-size: 12px; color: var(--sre-text-secondary); margin: 0; }

.filter-bar {
  display: flex; align-items: center; gap: 20px;
  flex-wrap: wrap;
  margin-bottom: 16px;
  padding: 12px 14px;
  background: var(--sre-bg-elevated);
  border: var(--sre-hairline);
  border-radius: var(--sre-radius-md);
}
.filter-group { display: flex; align-items: center; gap: 8px; }
.search-input { max-width: 220px; margin-left: auto; }

.user-list { display: flex; flex-direction: column; gap: 6px; }

.user-row {
  display: flex; align-items: center; gap: 12px;
  padding: 12px 14px;
  transition: all var(--sre-duration-fast) var(--sre-ease-out);
  /* Virtual scrolling: skip rendering off-screen rows */
  content-visibility: auto;
  contain-intrinsic-size: auto 56px;
}
.user-row[data-dim] { opacity: 0.55; }

.user-avatar {
  width: 32px; height: 32px; border-radius: 50%;
  display: flex; align-items: center; justify-content: center;
  background: var(--sre-primary-soft); color: var(--sre-primary);
  font-size: 13px; font-weight: 600; flex-shrink: 0;
}
.user-main { flex: 1; display: flex; flex-direction: column; gap: 4px; min-width: 0; }
.user-headline {
  display: flex; align-items: baseline; gap: 6px;
  font-size: 14px; font-weight: 600;
  color: var(--sre-text-primary);
}
.user-name { color: var(--sre-text-primary); }
.user-username {
  font-size: 12px;
  color: var(--sre-text-secondary);
  font-family: var(--sre-font-mono);
  font-weight: 400;
}
.user-meta, .user-footer {
  display: flex; align-items: center;
  font-size: 12px;
  color: var(--sre-text-secondary);
  gap: 6px;
}
.user-role-chip {
  font-size: 11px;
  padding: 2px 8px;
  border-radius: 4px;
  font-weight: 500;
  letter-spacing: 0.3px;
}
.user-role-chip[data-role="admin"]     { background: var(--sre-critical-soft); color: var(--sre-critical); }
.user-role-chip[data-role="team_lead"] { background: var(--sre-warning-soft); color: var(--sre-warning); }
.user-role-chip[data-role="member"]    { background: var(--sre-primary-soft); color: var(--sre-primary); }
.user-role-chip[data-role="viewer"]    { background: var(--sre-bg-elevated); color: var(--sre-text-secondary); }
.user-status {
  font-size: 11px; font-weight: 600;
  color: var(--sre-text-secondary);
  text-transform: uppercase;
  letter-spacing: 0.6px;
}
.user-actions { display: flex; align-items: center; gap: 8px; flex-shrink: 0; }

.empty-state {
  padding: 40px 0; text-align: center;
  font-size: 13px;
  color: var(--sre-text-tertiary);
}
</style>
