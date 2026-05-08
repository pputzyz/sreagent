<script setup lang="ts">
import { computed, reactive, ref, shallowRef, onMounted, h } from 'vue'
import {
  useMessage,
  NButton,
  NIcon,
  NInput,
  NDropdown,
  NModal,
  NForm,
  NFormItem,
  NSpace,
  NSpin,
  NDivider,
  NPopconfirm,
  NAvatar,
  NSelect,
} from 'naive-ui'
import { useI18n } from 'vue-i18n'
import { teamApi } from '@/api'
import type { User, Team } from '@/types'
import { kvArrayToRecord } from '@/utils/format'
import { AddOutline, EllipsisHorizontal, SearchOutline } from '@vicons/ionicons5'
import KVEditor from '@/components/common/KVEditor.vue'

const props = defineProps<{ allUsers: User[] }>()

const message = useMessage()
const { t } = useI18n()

const loading = ref(false)
const teamsList = shallowRef<Team[]>([])
const search = ref('')

const showModal = ref(false)
const modalTitle = ref('')
const editingId = ref<number | null>(null)
const saving = ref(false)

const showMembersModal = ref(false)
const membersTeamId = ref<number | null>(null)
const membersTeamName = ref('')
const teamMembers = shallowRef<User[]>([])
const selectedMemberUserId = ref<number | null>(null)
const membersLoading = ref(false)

const form = reactive({
  name: '',
  description: '',
  labels: [] as { key: string; value: string }[],
})

const allUserOptions = computed(() =>
  props.allUsers.map(u => ({ label: u.display_name || u.username, value: u.id }))
)

const filtered = computed(() => {
  const q = search.value.trim().toLowerCase()
  if (!q) return teamsList.value
  return teamsList.value.filter(tm =>
    `${tm.name} ${tm.description}`.toLowerCase().includes(q)
  )
})

function initials(u: User): string {
  const s = (u.display_name || u.username || '?').trim()
  return s.charAt(0).toUpperCase()
}

async function fetchTeams() {
  loading.value = true
  try {
    const { data } = await teamApi.list({ page: 1, page_size: 100 })
    teamsList.value = data.data.list || []
  } catch (err: any) {
    message.error(err.message)
  } finally {
    loading.value = false
  }
}

function openCreate() {
  editingId.value = null
  modalTitle.value = t('settings.createTeam')
  Object.assign(form, { name: '', description: '', labels: [] })
  showModal.value = true
}

function openEdit(tm: Team) {
  editingId.value = tm.id
  modalTitle.value = t('settings.editTeam')
  Object.assign(form, {
    name: tm.name,
    description: tm.description,
    labels: Object.entries(tm.labels || {}).map(([key, value]) => ({ key, value })),
  })
  showModal.value = true
}

async function handleSave() {
  if (!form.name.trim()) {
    message.warning(t('settings.nameRequired'))
    return
  }
  saving.value = true
  try {
    const payload = {
      name: form.name,
      description: form.description,
      labels: kvArrayToRecord(form.labels),
    }
    if (editingId.value) {
      await teamApi.update(editingId.value, payload)
      message.success(t('settings.teamUpdated'))
    } else {
      await teamApi.create(payload)
      message.success(t('settings.teamCreated'))
    }
    showModal.value = false
    fetchTeams()
  } catch (err: any) {
    message.error(err.message)
  } finally {
    saving.value = false
  }
}

async function handleDelete(id: number) {
  try {
    await teamApi.delete(id)
    message.success(t('settings.teamDeleted'))
    fetchTeams()
  } catch (err: any) {
    message.error(err.message)
  }
}

async function openMembers(tm: Team) {
  membersTeamId.value = tm.id
  membersTeamName.value = tm.name
  selectedMemberUserId.value = null
  showMembersModal.value = true
  await fetchTeamMembers(tm.id)
}

async function fetchTeamMembers(teamId: number) {
  membersLoading.value = true
  try {
    const { data } = await teamApi.listMembers(teamId)
    teamMembers.value = data.data || []
  } catch (err: any) {
    message.error(err.message)
    teamMembers.value = []
  } finally {
    membersLoading.value = false
  }
}

async function handleAddMember() {
  if (!membersTeamId.value || !selectedMemberUserId.value) return
  if (teamMembers.value.find(m => m.id === selectedMemberUserId.value)) {
    message.warning(t('settings.memberExists'))
    return
  }
  try {
    await teamApi.addMember(membersTeamId.value, selectedMemberUserId.value)
    message.success(t('settings.memberAdded'))
    selectedMemberUserId.value = null
    await fetchTeamMembers(membersTeamId.value)
    fetchTeams()
  } catch (err: any) {
    message.error(err.message)
  }
}

async function handleRemoveMember(userId: number) {
  if (!membersTeamId.value) return
  try {
    await teamApi.removeMember(membersTeamId.value, userId)
    message.success(t('settings.memberRemoved'))
    await fetchTeamMembers(membersTeamId.value)
    fetchTeams()
  } catch (err: any) {
    message.error(err.message)
  }
}

function cardMenuOptions() {
  return [
    { key: 'edit', label: t('common.edit') },
    { key: 'members', label: t('settings.members') },
    { type: 'divider', key: 'd' },
    { key: 'delete', label: t('common.delete') },
  ]
}

function handleCardMenu(key: string, tm: Team, evt?: MouseEvent) {
  if (key === 'edit') openEdit(tm)
  else if (key === 'members') openMembers(tm)
  else if (key === 'delete') {
    if (confirm(t('settings.deleteTeamConfirm'))) handleDelete(tm.id)
  }
  evt?.stopPropagation()
}

const ellipsisIcon = () => h(NIcon, { component: EllipsisHorizontal })

onMounted(fetchTeams)
</script>

<template>
  <div class="team-mgmt">
    <header class="page-header">
      <div>
        <h2 class="page-title">{{ t('settings.teamManagement') }}</h2>
        <p class="page-subtitle">{{ t('settings.teamManagementDesc') }}</p>
      </div>
      <NButton type="primary" size="small" @click="openCreate">
        <template #icon><NIcon :component="AddOutline" /></template>
        {{ t('settings.createTeam') }}
      </NButton>
    </header>

    <div class="filter-bar">
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

    <div class="team-grid sre-stagger">
      <div
        v-for="tm in filtered"
        :key="tm.id"
        class="team-card sre-lift"
        @click="openEdit(tm)"
      >
        <div class="team-card-head">
          <div class="team-name">{{ tm.name }}</div>
          <NDropdown
            trigger="click"
            :options="cardMenuOptions()"
            @select="(k: string) => handleCardMenu(k, tm)"
          >
            <NButton
              size="tiny"
              quaternary
              :render-icon="ellipsisIcon"
              @click.stop
            />
          </NDropdown>
        </div>
        <p class="team-desc">{{ tm.description || '—' }}</p>

        <div class="team-members">
          <div class="member-avatars">
            <div
              v-for="(m, i) in (tm.members || []).slice(0, 4)"
              :key="m.id"
              class="member-avatar"
              :style="{ zIndex: 4 - i }"
              :title="m.display_name || m.username"
            >
              {{ initials(m) }}
            </div>
            <span v-if="(tm.members?.length || 0) > 4" class="member-more tnum">
              +{{ (tm.members?.length || 0) - 4 }}
            </span>
            <span v-if="!tm.members?.length" class="member-empty">{{ t('settings.noMembers') }}</span>
          </div>
          <span class="team-meta tnum">{{ tm.members?.length || 0 }} {{ t('settings.members').toLowerCase() }}</span>
        </div>

        <div class="team-footer">
          <div class="team-labels">
            <span
              v-for="[k, v] in Object.entries(tm.labels || {}).slice(0, 2)"
              :key="k"
              class="label-chip mono"
            >{{ k }}={{ v }}</span>
            <span v-if="Object.keys(tm.labels || {}).length > 2" class="tnum label-more">
              +{{ Object.keys(tm.labels || {}).length - 2 }}
            </span>
          </div>
          <NButton size="tiny" quaternary @click.stop="openMembers(tm)">
            {{ t('settings.members') }}
          </NButton>
        </div>
      </div>

      <div v-if="!loading && filtered.length === 0" class="empty-state">
        {{ t('settings.noTeams') }}
      </div>
    </div>

    <NModal v-model:show="showModal" preset="card" :title="modalTitle" style="width: 560px" :bordered="false">
      <NForm label-placement="top">
        <NFormItem :label="t('common.name')" required>
          <NInput v-model:value="form.name" placeholder="e.g. Platform Engineering" />
        </NFormItem>
        <NFormItem :label="t('common.description')">
          <NInput v-model:value="form.description" type="textarea" :placeholder="t('common.description')" :rows="2" />
        </NFormItem>
        <NFormItem :label="t('settings.labels')">
          <KVEditor v-model="form.labels" :add-label="t('settings.addTeamLabel')" />
        </NFormItem>
      </NForm>
      <template #action>
        <NSpace justify="end">
          <NButton @click="showModal = false">{{ t('common.cancel') }}</NButton>
          <NButton type="primary" :loading="saving" @click="handleSave">
            {{ editingId ? t('common.update') : t('common.create') }}
          </NButton>
        </NSpace>
      </template>
    </NModal>

    <NModal
      v-model:show="showMembersModal"
      preset="card"
      :title="t('settings.members') + ' — ' + membersTeamName"
      style="width: 560px"
      :bordered="false"
    >
      <NSpin :show="membersLoading">
        <div class="members-list">
          <div v-for="m in teamMembers" :key="m.id" class="member-row sre-row-card">
            <NAvatar :size="28" round>{{ initials(m) }}</NAvatar>
            <div class="member-info">
              <div class="member-name">{{ m.display_name || m.username }}</div>
              <div class="member-meta">
                <span>{{ m.email || m.username }}</span>
                <span class="sre-meta-divider"></span>
                <span class="user-role-chip" :data-role="m.role">{{ m.role }}</span>
              </div>
            </div>
            <NPopconfirm @positive-click="handleRemoveMember(m.id)">
              <template #trigger>
                <NButton size="tiny" quaternary type="error">{{ t('common.remove') }}</NButton>
              </template>
              {{ t('settings.removeMemberConfirm') }}
            </NPopconfirm>
          </div>
          <div v-if="teamMembers.length === 0" class="empty-state-sm">
            {{ t('settings.noMembers') }}
          </div>
        </div>
      </NSpin>

      <NDivider style="margin: 16px 0" />

      <div class="add-member-row">
        <NSelect
          v-model:value="selectedMemberUserId"
          :options="allUserOptions"
          :placeholder="t('settings.selectUserToAdd')"
          filterable
          style="flex: 1"
        />
        <NButton type="primary" @click="handleAddMember" :disabled="!selectedMemberUserId">
          {{ t('common.add') }}
        </NButton>
      </div>
    </NModal>
  </div>
</template>

<style scoped>
.team-mgmt { font-family: var(--sre-font-sans); }

.page-header {
  display: flex; align-items: flex-start; justify-content: space-between;
  padding-bottom: 16px;
  border-bottom: var(--sre-hairline);
  margin-bottom: 20px;
}
.page-title { font-size: 18px; font-weight: 600; margin: 0 0 4px; color: var(--sre-text-primary); }
.page-subtitle { font-size: 12px; color: var(--sre-text-secondary); margin: 0; }

.filter-bar { display: flex; margin-bottom: 16px; }
.search-input { max-width: 280px; }

.team-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(280px, 1fr));
  gap: 16px;
}

.team-card {
  background: var(--sre-bg-card);
  border: var(--sre-hairline);
  border-radius: var(--sre-radius-md);
  padding: 16px 18px;
  cursor: pointer;
  display: flex; flex-direction: column; gap: 10px;
  transition: all var(--sre-duration-fast) var(--sre-ease-out);
  min-height: 168px;
}
.team-card:hover {
  transform: translateY(-2px);
  border-color: var(--sre-primary);
}
.team-card-head {
  display: flex; align-items: flex-start; justify-content: space-between;
  gap: 8px;
}
.team-name {
  font-size: 15px; font-weight: 600;
  color: var(--sre-text-primary);
  flex: 1; min-width: 0;
  overflow: hidden; text-overflow: ellipsis; white-space: nowrap;
}
.team-desc {
  font-size: 12px;
  color: var(--sre-text-secondary);
  margin: 0;
  line-height: 1.5;
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
  min-height: 36px;
}

.team-members { display: flex; align-items: center; gap: 10px; }
.member-avatars { display: flex; align-items: center; }
.member-avatar {
  width: 24px; height: 24px; border-radius: 50%;
  background: var(--sre-primary-soft); color: var(--sre-primary);
  font-size: 10px; font-weight: 600;
  display: flex; align-items: center; justify-content: center;
  border: 2px solid var(--sre-bg-card);
  margin-left: -6px;
}
.member-avatar:first-child { margin-left: 0; }
.member-more {
  font-size: 11px;
  color: var(--sre-text-tertiary);
  margin-left: 6px;
}
.member-empty { font-size: 11px; color: var(--sre-text-tertiary); font-style: italic; }
.team-meta {
  font-size: 11px;
  color: var(--sre-text-tertiary);
  margin-left: auto;
}

.team-footer {
  display: flex; align-items: center; justify-content: space-between;
  padding-top: 10px;
  border-top: var(--sre-hairline);
  margin-top: auto;
}
.team-labels { display: flex; align-items: center; gap: 6px; flex-wrap: wrap; min-width: 0; }
.label-chip {
  font-size: 10.5px;
  padding: 2px 6px;
  border-radius: 4px;
  background: var(--sre-bg-elevated);
  color: var(--sre-text-secondary);
  font-family: var(--sre-font-mono);
}
.label-more {
  font-size: 10.5px;
  color: var(--sre-text-tertiary);
}

.empty-state {
  grid-column: 1 / -1;
  padding: 40px 0; text-align: center;
  font-size: 13px;
  color: var(--sre-text-tertiary);
}

.members-list { display: flex; flex-direction: column; gap: 6px; max-height: 360px; overflow-y: auto; }
.member-row {
  display: flex; align-items: center; gap: 10px;
  padding: 10px 12px;
}
.member-info { flex: 1; min-width: 0; }
.member-name {
  font-size: 13px; font-weight: 600;
  color: var(--sre-text-primary);
}
.member-meta {
  display: flex; align-items: center; gap: 6px;
  font-size: 11px;
  color: var(--sre-text-tertiary);
}
.user-role-chip {
  font-size: 10.5px;
  padding: 1px 6px; border-radius: 3px;
  font-weight: 500;
  background: var(--sre-bg-elevated);
  color: var(--sre-text-secondary);
  text-transform: uppercase;
  letter-spacing: 0.4px;
}
.user-role-chip[data-role="admin"]     { background: var(--sre-critical-soft); color: var(--sre-critical); }
.user-role-chip[data-role="team_lead"] { background: var(--sre-warning-soft); color: var(--sre-warning); }
.user-role-chip[data-role="member"]    { background: var(--sre-primary-soft); color: var(--sre-primary); }

.empty-state-sm {
  padding: 24px 0; text-align: center;
  font-size: 12px;
  color: var(--sre-text-tertiary);
}

.add-member-row { display: flex; gap: 8px; align-items: center; }
</style>
