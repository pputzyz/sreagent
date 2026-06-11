<script setup lang="ts">
import { ref, onMounted, onBeforeUnmount, computed, h } from 'vue'
import { useRouter } from 'vue-router'
import { useMessage, NIcon } from 'naive-ui'
import { useI18n } from 'vue-i18n'
import { channelV2Api, teamApi } from '@/api'
import type { Channel, ChannelStatus, ChannelAccessLevel, Team } from '@/types'
import { getErrorMessage } from '@/utils/format'
import { usePaginatedList } from '@/composables'
import PageHeader from '@/components/common/PageHeader.vue'
import EmptyState from '@/components/common/EmptyState.vue'
import LoadingSkeleton from '@/components/common/LoadingSkeleton.vue'
import {
  AddOutline, RefreshOutline, StarOutline, Star,
  PeopleOutline, SearchOutline, EllipsisHorizontal,
  GridOutline, ListOutline, FolderOpenOutline,
  CreateOutline, TrashOutline, LayersOutline,
} from '@vicons/ionicons5'

const { t } = useI18n()
const message = useMessage()
const router = useRouter()

const searchQuery = ref('')
let searchTimer: ReturnType<typeof setTimeout> | null = null
function onSearchInput() {
  if (searchTimer) clearTimeout(searchTimer)
  searchTimer = setTimeout(() => { page.value = 1; fetchList() }, 300)
}

onBeforeUnmount(() => {
  if (searchTimer) clearTimeout(searchTimer)
})
const statusFilter = ref<'' | 'active' | 'disabled'>('')
const teamFilter = ref<number | null>(null)
const teams = ref<Team[]>([])
const sortBy = ref<'recent' | 'created' | 'name' | 'incidents'>('recent')
const viewMode = ref<'card' | 'list'>('card')

async function loadTeams() {
  try {
    const res = await teamApi.list({ page: 1, page_size: 200 })
    teams.value = res.data.data?.list || []
  } catch { teams.value = [] }
}

const {
  loading,
  items: channels,
  total,
  page,
  pageSize,
  fetchList,
  refresh,
} = usePaginatedList<Channel>({
  apiFn: channelV2Api.list,
  extraParams: () => {
    const params: Record<string, unknown> = {}
    if (searchQuery.value) params.query = searchQuery.value
    if (statusFilter.value) params.status = statusFilter.value
    if (teamFilter.value) params.team_id = teamFilter.value
    return params
  },
  onError: (err: unknown) => {
    message.error((err as Error)?.message ?? t('common.loadFailed'))
  },
})

// Create modal
const showCreateModal = ref(false)
const saving = ref(false)

const form = ref<{
  name: string
  description: string
  team_id: number | undefined
  status: ChannelStatus
  access_level: ChannelAccessLevel
  auto_close_enabled: boolean
  auto_close_minutes: number
  follow_alert_close: boolean
}>({
  name: '',
  description: '',
  team_id: undefined,
  status: 'active',
  access_level: 'public',
  auto_close_enabled: false,
  auto_close_minutes: 60,
  follow_alert_close: true,
})

const sortedChannels = computed(() => {
  const list = [...channels.value]
  switch (sortBy.value) {
    case 'name':
      return list.sort((a, b) => a.name.localeCompare(b.name))
    case 'incidents':
      return list.sort((a, b) => (b.active_incident_count ?? 0) - (a.active_incident_count ?? 0))
    case 'created':
      return list.sort((a, b) => String(b.created_at ?? '').localeCompare(String(a.created_at ?? '')))
    case 'recent':
    default:
      return list.sort((a, b) => {
        if (!!b.is_starred !== !!a.is_starred) return b.is_starred ? 1 : -1
        return String(b.updated_at ?? b.created_at ?? '').localeCompare(String(a.updated_at ?? a.created_at ?? ''))
      })
  }
})

async function toggleStar(ch: Channel) {
  const original = ch.is_starred
  ch.is_starred = !ch.is_starred
  try {
    if (original) {
      await channelV2Api.unstar(ch.id)
    } else {
      await channelV2Api.star(ch.id)
    }
  } catch (e: unknown) {
    ch.is_starred = original
    message.error(getErrorMessage(e) || t('common.failed'))
  }
}

async function deleteChannel(id: number) {
  try {
    await channelV2Api.delete(id)
    message.success(t('common.deleteSuccess'))
    await fetchList()
  } catch (e: unknown) {
    message.error(getErrorMessage(e) || t('common.deleteFailed'))
  }
}

async function createChannel() {
  if (!form.value.name.trim()) {
    message.warning(t('common.required'))
    return
  }
  saving.value = true
  try {
    await channelV2Api.create(form.value)
    message.success(t('common.createSuccess'))
    showCreateModal.value = false
    form.value = {
      name: '',
      description: '',
      team_id: undefined,
      status: 'active',
      access_level: 'public',
      auto_close_enabled: false,
      auto_close_minutes: 60,
      follow_alert_close: true,
    }
    await fetchList()
  } catch (e: unknown) {
    message.error(getErrorMessage(e) || t('common.failed'))
  } finally {
    saving.value = false
  }
}

function openChannel(id: number) {
  router.push(`/oncall/spaces/${id}`)
}

const statusOptions = computed(() => [
  { label: t('channel.all'), value: '' },
  { label: t('common.active'), value: 'active' },
  { label: t('common.disabled'), value: 'disabled' },
])

const teamOptions = computed(() =>
  teams.value.map(team => ({ label: team.name, value: team.id }))
)

const sortOptions = computed(() => [
  { label: t('channel.recentActivity'), value: 'recent' },
  { label: t('channel.createdTime'), value: 'created' },
  { label: t('channel.name'), value: 'name' },
  { label: t('channel.incidentCount'), value: 'incidents' },
])

const pendingDeleteId = ref<number | null>(null)
const showDeleteConfirm = ref(false)

function buildMenuOptions(ch: Channel) {
  return [
    {
      key: 'edit',
      label: t('channel.edit'),
      icon: () => h(NIcon, { component: CreateOutline }),
    },
    {
      key: 'delete',
      label: t('channel.delete'),
      icon: () => h(NIcon, { component: TrashOutline }),
    },
  ]
}

function handleMenuSelect(key: string, ch: Channel) {
  if (key === 'edit') {
    router.push(`/oncall/spaces/${ch.id}?tab=settings`)
  } else if (key === 'delete') {
    pendingDeleteId.value = ch.id
    showDeleteConfirm.value = true
  }
}

function confirmDelete() {
  if (pendingDeleteId.value != null) {
    deleteChannel(pendingDeleteId.value)
  }
  showDeleteConfirm.value = false
  pendingDeleteId.value = null
}

function fmtMetric(val: number | string | undefined | null): string {
  if (val == null || val === '') return '—'
  return String(val)
}

onMounted(() => {
  fetchList()
  loadTeams()
})
</script>

<template>
  <div class="channels-page">
    <PageHeader :title="t('channel.title')" :subtitle="t('channel.subtitle')">
      <template #actions>
        <n-button quaternary circle @click="fetchList" :loading="loading">
          <template #icon><n-icon :component="RefreshOutline" /></template>
        </n-button>
        <n-button type="primary" @click="showCreateModal = true">
          <template #icon><n-icon :component="AddOutline" /></template>
          {{ t('channel.create') }}
        </n-button>
      </template>
    </PageHeader>

    <!-- Filter bar -->
    <div class="filter-bar">
      <n-input
        v-model:value="searchQuery"
        :placeholder="t('common.search')"
        clearable
        class="filter-search"
        @update:value="onSearchInput"
      >
        <template #prefix><n-icon :component="SearchOutline" /></template>
      </n-input>

      <n-radio-group v-model:value="statusFilter" size="medium" @update:value="fetchList">
        <n-radio-button v-for="opt in statusOptions" :key="String(opt.value)" :value="opt.value">
          {{ opt.label }}
        </n-radio-button>
      </n-radio-group>

      <n-select
        v-model:value="teamFilter"
        :options="teamOptions"
        size="medium"
        clearable
        :placeholder="t('channel.teamFilter')"
        class="filter-team"
        @update:value="fetchList"
      />

      <n-select
        v-model:value="sortBy"
        :options="sortOptions"
        size="medium"
        class="filter-sort"
      />

      <div class="view-toggle">
        <n-button
          :quaternary="viewMode !== 'card'"
          :secondary="viewMode === 'card'"
          size="medium"
          @click="viewMode = 'card'"
        >
          <template #icon><n-icon :component="GridOutline" /></template>
        </n-button>
        <n-button
          :quaternary="viewMode !== 'list'"
          :secondary="viewMode === 'list'"
          size="medium"
          @click="viewMode = 'list'"
        >
          <template #icon><n-icon :component="ListOutline" /></template>
        </n-button>
      </div>
    </div>

    <!-- Loading / Empty / Cards -->
    <LoadingSkeleton v-if="loading" :rows="6" variant="card-grid" />

    <EmptyState
      v-else-if="sortedChannels.length === 0"
      :icon="LayersOutline"
      :title="t('channel.noChannels')"
      :description="t('channel.noChannelsDesc')"
      :primary-text="t('channel.create')"
      @primary="showCreateModal = true"
    />

    <div v-else class="channel-grid stagger-grid">
      <div
        v-for="ch in sortedChannels"
        :key="ch.id"
        class="channel-card"
        :class="{ 'is-disabled': ch.status === 'disabled' }"
        @click="openChannel(ch.id)"
      >
        <div class="card-stripe" />

        <button
          class="card-star"
          :class="{ starred: ch.is_starred }"
          @click.stop="toggleStar(ch)"
        >
          <n-icon :component="ch.is_starred ? Star : StarOutline" :size="18" />
        </button>

        <div class="card-body">
          <h3 class="card-name">{{ ch.name }}</h3>
          <p class="card-desc">{{ ch.description || '—' }}</p>

          <div class="card-metrics">
            <div class="metric">
              <div class="metric-value" :class="{ 'is-active': (ch.active_incident_count ?? 0) > 0 }">
                {{ fmtMetric(ch.active_incident_count) }}
              </div>
              <div class="metric-label">{{ t('channel.activeIncidents') }}</div>
            </div>
            <div class="metric">
              <div class="metric-value">{{ fmtMetric(ch.mtta_label) }}</div>
              <div class="metric-label">{{ t('dashboard.mtta') }}</div>
            </div>
            <div class="metric">
              <div class="metric-value">{{ fmtMetric(ch.mttr_label) }}</div>
              <div class="metric-label">{{ t('dashboard.mttr') }}</div>
            </div>
          </div>
        </div>

        <div class="card-footer" @click.stop>
          <div class="footer-left">
            <template v-if="ch.team">
              <n-icon :component="PeopleOutline" :size="14" class="team-icon" />
              <span class="team-name">{{ ch.team.name }}</span>
            </template>
            <template v-else>
              <span class="team-name muted">{{ t('channel.unassignedTeam') }}</span>
            </template>
          </div>
          <div class="footer-right">
            <span class="status-dot" :class="ch.status">
              <span class="dot" />
              {{ ch.status === 'active' ? t('common.active') : t('common.disabled') }}
            </span>
            <n-dropdown
              trigger="click"
              :options="buildMenuOptions(ch)"
              @select="(key: string) => handleMenuSelect(key, ch)"
            >
              <n-button quaternary circle size="tiny" @click.stop>
                <template #icon><n-icon :component="EllipsisHorizontal" /></template>
              </n-button>
            </n-dropdown>
          </div>
        </div>
      </div>
    </div>

    <!-- Pagination -->
    <div v-if="total > pageSize" class="pagination-wrap">
      <n-pagination
        v-model:page="page"
        :page-size="pageSize"
        :item-count="total"
        :page-slot="7"
        @update:page="fetchList"
      />
    </div>

    <!-- Delete confirm -->
    <n-modal
      v-model:show="showDeleteConfirm"
      preset="dialog"
      :title="t('common.delete')"
      :content="t('channel.deleteConfirm')"
      :positive-text="t('common.confirmDelete')"
      :negative-text="t('common.cancel')"
      type="warning"
      @positive-click="confirmDelete"
    />

    <!-- Create Modal -->
    <n-modal
      v-model:show="showCreateModal"
      :title="t('channel.create')"
      preset="card"
      class="ch-modal-create"
      :bordered="false"
    >
      <n-form label-placement="top" size="small">
        <n-form-item :label="t('channel.name')" required>
          <n-input v-model:value="form.name" :placeholder="t('placeholder.spaceName')" />
        </n-form-item>
        <n-form-item :label="t('channel.description')">
          <n-input v-model:value="form.description" type="textarea" :rows="2" />
        </n-form-item>
        <n-form-item :label="t('channel.team')">
          <n-select
            v-model:value="form.team_id"
            :options="teamOptions"
            :placeholder="t('channel.selectTeam')"
            clearable
          />
        </n-form-item>
        <n-form-item :label="t('channel.accessLevel')">
          <n-radio-group v-model:value="form.access_level">
            <n-radio value="public">{{ t('channel.accessPublic') }}</n-radio>
            <n-radio value="private">{{ t('channel.accessPrivate') }}</n-radio>
          </n-radio-group>
        </n-form-item>
        <n-form-item :label="t('channel.autoClose')">
          <n-switch v-model:value="form.auto_close_enabled" />
        </n-form-item>
        <n-form-item v-if="form.auto_close_enabled" :label="t('channel.autoCloseMinutes')">
          <n-input-number v-model:value="form.auto_close_minutes" :min="1" :max="10080" />
        </n-form-item>
        <n-form-item>
          <n-checkbox v-model:checked="form.follow_alert_close">
            {{ t('channel.followAlertClose') }}
          </n-checkbox>
        </n-form-item>
      </n-form>
      <template #footer>
        <n-space justify="end">
          <n-button @click="showCreateModal = false">{{ t('common.cancel') }}</n-button>
          <n-button type="primary" :loading="saving" @click="createChannel">{{ t('common.create') }}</n-button>
        </n-space>
      </template>
    </n-modal>
  </div>
</template>

<style scoped>
.channels-page { max-width: 1400px; }

.filter-bar {
  display: flex;
  align-items: center;
  gap: 12px;
  flex-wrap: wrap;
  margin-bottom: 20px;
  padding: 12px 14px;
  background: var(--sre-bg-card);
  border: 1px solid var(--sre-border);
  border-radius: 8px;
}
.filter-search { width: 280px; flex: 0 0 auto; }
.filter-team { width: 160px; }
.filter-sort { width: 160px; }
.view-toggle { display: flex; gap: 4px; margin-left: auto; }

.channel-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(320px, 1fr));
  gap: 16px;
  margin-bottom: 16px;
}

.channel-card {
  position: relative;
  background: var(--sre-bg-card);
  border: 1px solid var(--sre-border);
  border-radius: 8px;
  padding: 16px 18px 14px calc(var(--sre-stripe-w) + 16px);
  cursor: pointer;
  overflow: hidden;
  transition: transform 0.18s ease, box-shadow 0.18s ease, border-color 0.18s ease;
  display: flex;
  flex-direction: column;
  min-height: 188px;
}
.channel-card:hover {
  box-shadow: var(--sre-shadow-md);
  border-color: var(--sre-primary);
}
.channel-card:hover .card-star { opacity: 1; }
.channel-card.is-disabled { opacity: 0.78; }

.card-stripe {
  position: absolute;
  left: 0; top: 4px; bottom: 4px;
  width: var(--sre-stripe-w);
  background: var(--sre-primary);
  border-radius: 0 4px 4px 0;
}

.card-star {
  position: absolute;
  top: 10px;
  right: 10px;
  width: 30px;
  height: 30px;
  display: flex;
  align-items: center;
  justify-content: center;
  border: none;
  border-radius: 6px;
  background: transparent;
  color: var(--sre-text-tertiary);
  cursor: pointer;
  opacity: 0;
  transition: opacity 0.18s ease, background 0.18s ease, color 0.18s ease;
}
.card-star:hover { background: var(--sre-bg-hover); color: var(--sre-warning); }
.card-star.starred { opacity: 1; color: var(--sre-warning); }

.card-body { flex: 1 1 auto; display: flex; flex-direction: column; }
.card-name {
  font-size: 16px;
  font-weight: 600;
  color: var(--sre-text-primary);
  margin: 4px 36px 4px 0;
  line-height: 1.4;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.card-desc {
  font-size: 12px;
  color: var(--sre-text-secondary);
  margin: 0 0 14px;
  line-height: 1.5;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.card-metrics {
  display: flex;
  align-items: stretch;
  gap: 8px;
  padding: 10px 0;
  margin-top: auto;
  border-top: 1px dashed var(--sre-border);
}
.metric {
  flex: 1 1 0;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 2px;
  text-align: center;
}
.metric + .metric { border-left: 1px solid var(--sre-border); }
.metric-value {
  font-size: 20px;
  font-weight: 600;
  color: var(--sre-text-primary);
  line-height: 1.2;
  font-variant-numeric: tabular-nums;
}
.metric-value.is-active { color: var(--sre-danger); }
.metric-label {
  font-size: 11px;
  color: var(--sre-text-secondary);
}

.card-footer {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding-top: 10px;
  border-top: 1px solid var(--sre-border);
  font-size: 12px;
}
.footer-left {
  display: flex;
  align-items: center;
  gap: 6px;
  color: var(--sre-text-secondary);
  min-width: 0;
}
.team-icon { color: var(--sre-text-tertiary); }
.team-name { overflow: hidden; text-overflow: ellipsis; white-space: nowrap; max-width: 140px; }
.team-name.muted { color: var(--sre-text-tertiary); font-style: italic; }

.footer-right { display: flex; align-items: center; gap: 8px; }
.status-dot {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  font-size: 12px;
  color: var(--sre-text-secondary);
}
.status-dot .dot {
  width: 6px;
  height: 6px;
  border-radius: 50%;
  background: var(--sre-text-tertiary);
}
.status-dot.active .dot { background: var(--sre-primary); box-shadow: 0 0 0 3px var(--sre-primary-soft); }
.status-dot.disabled .dot { background: var(--sre-text-tertiary); }

.pagination-wrap {
  display: flex;
  justify-content: center;
  padding: 16px 0;
}

</style>

<style>
@import '@/styles/channels.css';
</style>
