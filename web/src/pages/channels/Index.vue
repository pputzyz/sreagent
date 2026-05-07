<script setup lang="ts">
import { ref, onMounted, computed, h } from 'vue'
import { useRouter } from 'vue-router'
import { useMessage, NButton, NTag, NSpace, NPopconfirm, NIcon } from 'naive-ui'
import { useI18n } from 'vue-i18n'
import { channelV2Api } from '@/api'
import type { Channel } from '@/types'
import PageHeader from '@/components/common/PageHeader.vue'
import {
  AddOutline, RefreshOutline, StarOutline, Star,
  PeopleOutline,
} from '@vicons/ionicons5'

const { t } = useI18n()
const message = useMessage()
const router = useRouter()

const loading = ref(false)
const channels = ref<Channel[]>([])
const total = ref(0)
const page = ref(1)
const pageSize = ref(20)
const searchQuery = ref('')
const statusFilter = ref('')

// Create modal
const showCreateModal = ref(false)
const saving = ref(false)
import type { ChannelStatus, ChannelAccessLevel } from '@/types'

const form = ref<{
  name: string
  description: string
  status: ChannelStatus
  access_level: ChannelAccessLevel
  auto_close_enabled: boolean
  auto_close_minutes: number
  follow_alert_close: boolean
}>({
  name: '',
  description: '',
  status: 'active',
  access_level: 'public',
  auto_close_enabled: false,
  auto_close_minutes: 60,
  follow_alert_close: true,
})

async function loadChannels() {
  loading.value = true
  try {
    const res = await channelV2Api.list({
      query: searchQuery.value,
      status: statusFilter.value,
      page: page.value,
      page_size: pageSize.value,
    })
    channels.value = res.data.data?.list ?? []
    total.value = res.data.data?.total ?? 0
  } catch (e: any) {
    message.error(e?.message ?? t('common.loadFailed'))
  } finally {
    loading.value = false
  }
}

async function toggleStar(ch: Channel) {
  try {
    if (ch.is_starred) {
      await channelV2Api.unstar(ch.id)
    } else {
      await channelV2Api.star(ch.id)
    }
    ch.is_starred = !ch.is_starred
  } catch (e: any) {
    message.error(e?.message ?? t('common.failed'))
  }
}

async function deleteChannel(id: number) {
  try {
    await channelV2Api.delete(id)
    message.success(t('common.deleteSuccess'))
    await loadChannels()
  } catch (e: any) {
    message.error(e?.message ?? t('common.deleteFailed'))
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
      status: 'active' as ChannelStatus,
      access_level: 'public' as ChannelAccessLevel,
      auto_close_enabled: false,
      auto_close_minutes: 60,
      follow_alert_close: true,
    }
    await loadChannels()
  } catch (e: any) {
    message.error(e?.message ?? t('common.failed'))
  } finally {
    saving.value = false
  }
}

function openChannel(id: number) {
  router.push(`/channels/${id}`)
}

const statusTypeMap: Record<string, 'success' | 'warning' | 'default'> = {
  active: 'success',
  disabled: 'warning',
}

onMounted(loadChannels)
</script>

<template>
  <div class="channels-page">
    <PageHeader :title="t('channel.title')" :subtitle="t('channel.subtitle')">
      <template #actions>
        <n-button circle quaternary @click="loadChannels">
          <template #icon><n-icon :component="RefreshOutline" /></template>
        </n-button>
        <n-button type="primary" @click="showCreateModal = true">
          <template #icon><n-icon :component="AddOutline" /></template>
          {{ t('channel.create') }}
        </n-button>
      </template>
    </PageHeader>

    <!-- Filters -->
    <n-card :bordered="false" class="filter-card">
      <n-space>
        <n-input
          v-model:value="searchQuery"
          :placeholder="t('common.search')"
          clearable
          style="width: 240px"
          @update:value="loadChannels"
        />
        <n-select
          v-model:value="statusFilter"
          :options="[
            { label: t('common.active'), value: 'active' },
            { label: t('common.disabled'), value: 'disabled' },
          ]"
          :placeholder="t('common.status')"
          clearable
          style="width: 140px"
          @update:value="loadChannels"
        />
      </n-space>
    </n-card>

    <!-- Channel cards -->
    <div v-if="loading" class="loading-wrap">
      <n-spin size="large" />
    </div>

    <div v-else-if="channels.length === 0" class="empty-wrap">
      <n-empty :description="t('channel.noChannels')" />
    </div>

    <div v-else class="channel-grid">
      <div
        v-for="ch in channels"
        :key="ch.id"
        class="channel-card"
        @click="openChannel(ch.id)"
      >
        <div class="card-header">
          <div class="card-title">
            <span class="name">{{ ch.name }}</span>
            <n-tag :type="statusTypeMap[ch.status] ?? 'default'" size="small">
              {{ ch.status === 'active' ? t('common.active') : t('common.disabled') }}
            </n-tag>
          </div>
          <div class="card-actions" @click.stop>
            <n-button
              quaternary circle size="small"
              :type="ch.is_starred ? 'warning' : 'default'"
              @click="toggleStar(ch)"
            >
              <template #icon>
                <n-icon :component="ch.is_starred ? Star : StarOutline" />
              </template>
            </n-button>
          </div>
        </div>

        <p class="card-desc">{{ ch.description || '—' }}</p>

        <div class="card-stats">
          <div class="stat">
            <span class="stat-value">{{ ch.active_incident_count }}</span>
            <span class="stat-label">{{ t('channel.activeIncidents') }}</span>
          </div>
          <div v-if="ch.team" class="stat">
            <n-icon :component="PeopleOutline" size="14" />
            <span class="stat-label">{{ ch.team.name }}</span>
          </div>
        </div>

        <div class="card-footer" @click.stop>
          <n-popconfirm @positive-click="deleteChannel(ch.id)">
            <template #trigger>
              <n-button size="tiny" quaternary type="error">{{ t('common.delete') }}</n-button>
            </template>
            {{ t('channel.deleteConfirm') }}
          </n-popconfirm>
        </div>
      </div>
    </div>

    <!-- Pagination -->
    <div v-if="total > pageSize" class="pagination-wrap">
      <n-pagination
        v-model:page="page"
        :page-count="Math.ceil(total / pageSize)"
        @update:page="loadChannels"
      />
    </div>

    <!-- Create Modal -->
    <n-modal
      v-model:show="showCreateModal"
      :title="t('channel.create')"
      preset="card"
      style="width: 480px"
      :bordered="false"
    >
      <n-form label-placement="top" size="small">
        <n-form-item :label="t('channel.name')" required>
          <n-input v-model:value="form.name" :placeholder="t('channel.name')" />
        </n-form-item>
        <n-form-item :label="t('channel.description')">
          <n-input v-model:value="form.description" type="textarea" :rows="2" />
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

.filter-card {
  border-radius: 12px;
  margin-bottom: 16px;
}

.loading-wrap, .empty-wrap {
  display: flex;
  justify-content: center;
  align-items: center;
  min-height: 300px;
}

.channel-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(280px, 1fr));
  gap: 16px;
  margin-bottom: 16px;
}

.channel-card {
  background: var(--sre-bg-card);
  border: 1px solid var(--sre-border);
  border-radius: 12px;
  padding: 16px;
  cursor: pointer;
  transition: box-shadow 0.2s, border-color 0.2s;
}

.channel-card:hover {
  box-shadow: 0 4px 16px rgba(0,0,0,0.12);
  border-color: var(--sre-primary);
}

.card-header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  margin-bottom: 8px;
}

.card-title {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
}

.name {
  font-size: 15px;
  font-weight: 600;
  color: var(--sre-text-primary);
}

.card-desc {
  font-size: 12px;
  color: var(--sre-text-secondary);
  margin: 0 0 12px;
  line-height: 1.5;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.card-stats {
  display: flex;
  align-items: center;
  gap: 16px;
  margin-bottom: 12px;
}

.stat {
  display: flex;
  align-items: center;
  gap: 4px;
}

.stat-value {
  font-size: 20px;
  font-weight: 700;
  color: var(--sre-primary);
}

.stat-label {
  font-size: 11px;
  color: var(--sre-text-secondary);
}

.card-footer {
  border-top: 1px solid var(--sre-border);
  padding-top: 10px;
}

.pagination-wrap {
  display: flex;
  justify-content: center;
  padding: 16px 0;
}
</style>
