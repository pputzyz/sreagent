<script setup lang="ts">
import { ref, shallowRef, onMounted, computed, h } from 'vue'
import { useMessage, NButton, NIcon, NDropdown } from 'naive-ui'
import { useI18n } from 'vue-i18n'
import { integrationV2Api, channelV2Api } from '@/api'
import RoutingRules from './RoutingRules.vue'
import {
  AddOutline,
  CopyOutline,
  GitNetworkOutline,
  EllipsisHorizontalOutline,
  CreateOutline,
  TrashOutline,
  RefreshOutline,
} from '@vicons/ionicons5'

const { t } = useI18n()
const message = useMessage()

const loading = ref(false)
const integrations = shallowRef<any[]>([])
const channels = shallowRef<any[]>([])

// Filters
const filterType = ref<'all' | 'standard' | 'alertmanager' | 'grafana'>('all')
const filterMode = ref<'all' | 'exclusive' | 'shared'>('all')

const typeFilterOptions = computed(() => [
  { label: t('common.all'), value: 'all' },
  { label: 'Standard', value: 'standard' },
  { label: 'AlertManager', value: 'alertmanager' },
  { label: 'Grafana', value: 'grafana' },
])
const modeFilterOptions = computed(() => [
  { label: t('common.all'), value: 'all' },
  { label: t('integration.modeExclusive'), value: 'exclusive' },
  { label: t('integration.modeShared'), value: 'shared' },
])

const filteredIntegrations = computed(() => {
  return integrations.value.filter((it) => {
    if (filterType.value !== 'all' && it.type !== filterType.value) return false
    if (filterMode.value !== 'all' && it.mode !== filterMode.value) return false
    return true
  })
})

// Modal
const showModal = ref(false)
const saving = ref(false)
const editingId = ref<number | null>(null)

const defaultForm = () => ({
  name: '',
  description: '',
  type: 'standard',
  mode: 'exclusive',
  channel_id: null as number | null,
  pipeline_config: '',
  label_enhancement_config: '',
  is_enabled: true,
})
const form = ref(defaultForm())

const typeOptions = computed(() => [
  { label: 'Standard', value: 'standard' },
  { label: 'AlertManager', value: 'alertmanager' },
  { label: 'Grafana', value: 'grafana' },
])
const modeOptions = computed(() => [
  { label: t('integration.modeExclusive'), value: 'exclusive' },
  { label: t('integration.modeShared'), value: 'shared' },
])
const channelOptions = computed(() => channels.value.map((c) => ({ label: c.name, value: c.id })))

async function load() {
  loading.value = true
  try {
    const [intRes, chRes] = await Promise.all([
      integrationV2Api.list({ page: 1, page_size: 200 }),
      channelV2Api.list({ status: 'active', page: 1, page_size: 100 }),
    ])
    integrations.value = intRes.data.data?.list ?? []
    channels.value = chRes.data.data?.list ?? []
  } catch (e: any) {
    message.error(e?.message ?? t('common.loadFailed'))
  } finally {
    loading.value = false
  }
}

function webhookUrl(token: string) {
  return `${window.location.origin}/api/v1/integrations/${token}/alerts`
}

function webhookShort(token: string) {
  if (!token) return '—'
  const prefix = `/api/v1/integrations/`
  const tail = token.length > 14 ? token.slice(0, 8) + '…' + token.slice(-4) : token
  return prefix + tail + '/alerts'
}

function copyWebhook(integ: any) {
  if (!integ.webhook_token) return
  navigator.clipboard.writeText(webhookUrl(integ.webhook_token)).then(() =>
    message.success(t('common.copied') || '已复制'),
  )
}

function typeLabel(type: string) {
  if (type === 'alertmanager') return 'AlertManager'
  if (type === 'grafana') return 'Grafana'
  return 'Standard'
}

function formatNumber(n: number | null | undefined): string {
  const v = n ?? 0
  if (v >= 1000) return (v / 1000).toFixed(v >= 10000 ? 0 : 1) + 'k'
  return String(v)
}

function openCreate() {
  editingId.value = null
  Object.assign(form.value, defaultForm())
  showModal.value = true
}

function openEdit(integ: any) {
  editingId.value = integ.id
  Object.assign(form.value, {
    name: integ.name,
    description: integ.description ?? '',
    type: integ.type,
    mode: integ.mode,
    channel_id: integ.channel_id ?? null,
    pipeline_config: integ.pipeline_config ?? '',
    label_enhancement_config: integ.label_enhancement_config ?? '',
    is_enabled: integ.is_enabled,
  })
  showModal.value = true
}

async function save() {
  if (!form.value.name.trim()) {
    message.warning(t('integration.name') + ' ' + (t('common.required') || 'required'))
    return
  }
  saving.value = true
  try {
    const payload = {
      ...form.value,
      channel_id: form.value.channel_id ?? undefined,
    }
    if (editingId.value) {
      await integrationV2Api.update(editingId.value, payload)
    } else {
      await integrationV2Api.create(payload)
    }
    message.success(t('common.savedSuccess'))
    showModal.value = false
    await load()
  } catch (e: any) {
    message.error(e?.message ?? t('common.saveFailed'))
  } finally {
    saving.value = false
  }
}

async function deleteInteg(id: number) {
  try {
    await integrationV2Api.delete(id)
    message.success(t('common.deleteSuccess'))
    await load()
  } catch (e: any) {
    message.error(e?.message ?? t('common.deleteFailed'))
  }
}

function rowActions(_integ: any) {
  return [
    {
      key: 'edit',
      label: t('common.edit'),
      icon: () => h(NIcon, { component: CreateOutline }),
    },
    {
      key: 'delete',
      label: t('common.delete'),
      icon: () => h(NIcon, { component: TrashOutline }),
      props: { style: 'color: var(--sre-error, #ef4444)' },
    },
  ]
}

function handleAction(key: string, integ: any) {
  if (key === 'edit') openEdit(integ)
  else if (key === 'delete') {
    if (window.confirm(t('common.confirmDeleteMsg') || 'Confirm delete?')) {
      deleteInteg(integ.id)
    }
  }
}

// Routing rules drawer
const showRoutingDrawer = ref(false)
const routingIntegId = ref<number>(0)
const routingIntegName = ref('')

function openRoutingRules(integ: any) {
  routingIntegId.value = integ.id
  routingIntegName.value = integ.name
  showRoutingDrawer.value = true
}

onMounted(load)
</script>

<template>
  <div class="integ-page">
    <!-- Header -->
    <header class="integ-header">
      <div class="integ-header-text">
        <h1 class="integ-title">Integrations</h1>
        <p class="integ-subtitle">Connect external alert sources via webhook endpoints</p>
      </div>
      <div class="integ-header-actions">
        <n-button quaternary circle :loading="loading" @click="load">
          <template #icon><n-icon :component="RefreshOutline" /></template>
        </n-button>
        <n-button type="primary" @click="openCreate">
          <template #icon><n-icon :component="AddOutline" /></template>
          New Integration
        </n-button>
      </div>
    </header>

    <!-- Filters -->
    <section class="integ-filters">
      <div class="filter-group">
        <span class="sre-label-eyebrow">Type</span>
        <n-radio-group v-model:value="filterType" size="small">
          <n-radio-button v-for="o in typeFilterOptions" :key="o.value" :value="o.value">
            {{ o.label }}
          </n-radio-button>
        </n-radio-group>
      </div>
      <div class="filter-group">
        <span class="sre-label-eyebrow">Mode</span>
        <n-radio-group v-model:value="filterMode" size="small">
          <n-radio-button v-for="o in modeFilterOptions" :key="o.value" :value="o.value">
            {{ o.label }}
          </n-radio-button>
        </n-radio-group>
      </div>
    </section>

    <!-- Empty state -->
    <div v-if="!loading && filteredIntegrations.length === 0" class="integ-empty">
      <n-icon :component="GitNetworkOutline" size="48" class="integ-empty-icon" />
      <div class="integ-empty-title">No integrations yet</div>
      <div class="integ-empty-sub">
        Create your first integration to start receiving alerts from external systems.
      </div>
      <n-button type="primary" @click="openCreate">
        <template #icon><n-icon :component="AddOutline" /></template>
        Create Integration
      </n-button>
    </div>

    <!-- Card grid -->
    <section v-else class="integ-grid sre-stagger">
      <div
        v-for="integ in filteredIntegrations"
        :key="integ.id"
        class="integ-card sre-lift"
        @click="openEdit(integ)"
      >
        <div class="card-stripe" :data-type="integ.type"></div>

        <div class="card-status">
          <span
            class="sre-dot"
            :data-severity="integ.is_enabled ? 'success' : null"
          ></span>
          <span class="card-status-text">{{ integ.is_enabled ? 'Active' : 'Disabled' }}</span>
        </div>

        <div class="card-title">{{ integ.name }}</div>

        <div class="card-badges">
          <span class="card-badge" :data-type="integ.type">{{ typeLabel(integ.type) }}</span>
          <span class="card-badge-mode">
            {{ integ.mode === 'shared' ? 'Shared' : 'Exclusive' }}
          </span>
        </div>

        <div class="card-desc">{{ integ.description || '—' }}</div>

        <div class="card-webhook" @click.stop>
          <code class="webhook-url">{{ webhookShort(integ.webhook_token) }}</code>
          <n-button
            quaternary
            size="tiny"
            :title="t('integration.webhookUrl') + ' — copy'"
            @click.stop="copyWebhook(integ)"
          >
            <template #icon><n-icon :component="CopyOutline" /></template>
          </n-button>
        </div>

        <div class="card-footer">
          <span class="tnum">{{ formatNumber(integ.total_alerts) }} alerts</span>
          <template v-if="integ.channel">
            <span class="sre-meta-divider"></span>
            <span class="card-footer-channel">→ {{ integ.channel.name }}</span>
          </template>
        </div>

        <div class="card-actions" @click.stop>
          <n-button
            v-if="integ.mode === 'shared'"
            size="tiny"
            type="info"
            ghost
            @click="openRoutingRules(integ)"
          >
            <template #icon><n-icon :component="GitNetworkOutline" /></template>
            路由规则
          </n-button>
          <span v-else></span>
          <n-dropdown
            :options="rowActions(integ)"
            trigger="click"
            placement="bottom-end"
            @select="(k) => handleAction(k, integ)"
          >
            <n-button quaternary circle size="small" @click.stop>
              <template #icon><n-icon :component="EllipsisHorizontalOutline" /></template>
            </n-button>
          </n-dropdown>
        </div>
      </div>
    </section>

    <!-- Create/Edit modal -->
    <n-modal
      v-model:show="showModal"
      :title="editingId ? t('common.edit') : t('integration.create')"
      preset="card"
      style="width: 560px"
      :bordered="false"
      class="integ-modal"
    >
      <n-scrollbar style="max-height: 70vh">
        <n-form label-placement="top" size="small" style="padding-right: 12px">
          <n-form-item :label="t('integration.name')" required>
            <n-input v-model:value="form.name" />
          </n-form-item>
          <n-form-item :label="t('common.description')">
            <n-input v-model:value="form.description" type="textarea" :rows="2" />
          </n-form-item>
          <n-grid :cols="2" :x-gap="12">
            <n-form-item-gi :label="t('integration.type')">
              <n-select
                v-model:value="form.type"
                :options="typeOptions"
                :disabled="!!editingId"
              />
            </n-form-item-gi>
            <n-form-item-gi :label="t('integration.mode')">
              <n-select v-model:value="form.mode" :options="modeOptions" />
            </n-form-item-gi>
          </n-grid>
          <n-form-item v-if="form.mode === 'exclusive'" :label="t('integration.channel')">
            <n-select v-model:value="form.channel_id" :options="channelOptions" clearable />
          </n-form-item>
          <n-form-item :label="t('integration.pipelineConfig')">
            <n-input
              v-model:value="form.pipeline_config"
              type="textarea"
              :rows="4"
              :placeholder="t('integration.pipelineConfigHint')"
            />
          </n-form-item>
          <n-form-item :label="t('integration.labelEnhancement')">
            <n-input v-model:value="form.label_enhancement_config" type="textarea" :rows="3" />
          </n-form-item>
          <n-form-item>
            <n-checkbox v-model:checked="form.is_enabled">
              {{ t('common.enabled') }}
            </n-checkbox>
          </n-form-item>
        </n-form>
      </n-scrollbar>
      <template #footer>
        <n-space justify="end">
          <n-button @click="showModal = false">{{ t('common.cancel') }}</n-button>
          <n-button type="primary" :loading="saving" @click="save">
            {{ t('common.save') }}
          </n-button>
        </n-space>
      </template>
    </n-modal>

    <!-- Routing rules drawer -->
    <n-drawer v-model:show="showRoutingDrawer" :width="680" placement="right">
      <n-drawer-content :title="`路由规则 — ${routingIntegName}`" closable>
        <RoutingRules v-if="showRoutingDrawer" :integration-id="routingIntegId" />
      </n-drawer-content>
    </n-drawer>
  </div>
</template>

<style scoped>
.integ-page {
  max-width: 1400px;
  font-family: 'Geist', system-ui, -apple-system, sans-serif;
}

/* Header */
.integ-header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 16px;
  padding: 4px 0 18px;
}
.integ-header-text { display: flex; flex-direction: column; gap: 4px; }
.integ-title {
  font-family: 'Geist', system-ui, sans-serif;
  font-size: 22px;
  font-weight: 600;
  margin: 0;
  letter-spacing: -0.01em;
  color: var(--sre-text-primary);
}
.integ-subtitle {
  font-size: 13px;
  color: var(--sre-text-secondary);
  margin: 0;
}
.integ-header-actions { display: flex; align-items: center; gap: 8px; }

/* Filters */
.integ-filters {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 20px;
  padding: 12px 0 18px;
  border-bottom: var(--sre-hairline);
  margin-bottom: 18px;
}
.filter-group { display: flex; align-items: center; gap: 10px; }

/* Grid */
.integ-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(320px, 1fr));
  gap: 16px;
}

/* Card */
.integ-card {
  position: relative;
  background: var(--sre-bg-card, #fff);
  border: var(--sre-hairline);
  border-radius: var(--sre-radius-md, 10px);
  padding: 20px;
  display: flex;
  flex-direction: column;
  gap: 10px;
  cursor: pointer;
  overflow: hidden;
  transition: transform var(--sre-duration-fast, 160ms) var(--sre-ease-out, ease-out),
    border-color var(--sre-duration-fast, 160ms) var(--sre-ease-out, ease-out),
    box-shadow var(--sre-duration-fast, 160ms) var(--sre-ease-out, ease-out);
}
.integ-card:hover {
  border-color: var(--sre-primary, #2563eb);
}

.card-stripe {
  position: absolute;
  top: 0; left: 0; right: 0;
  height: 3px;
  background: var(--sre-text-tertiary, #9ca3af);
}
.card-stripe[data-type='alertmanager'] { background: #e6522c; }
.card-stripe[data-type='grafana']      { background: #f59e0b; }
.card-stripe[data-type='standard']     { background: var(--sre-primary, #2563eb); }

.card-status {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 11px;
  color: var(--sre-text-tertiary);
  text-transform: uppercase;
  letter-spacing: 0.6px;
  font-weight: 500;
}

.card-title {
  font-size: 16px;
  font-weight: 600;
  color: var(--sre-text-primary);
  margin: 4px 0 0;
  letter-spacing: -0.005em;
}

.card-badges { display: flex; gap: 6px; flex-wrap: wrap; }
.card-badge {
  font-size: 11px;
  padding: 2px 8px;
  border-radius: 4px;
  font-weight: 500;
  letter-spacing: 0.3px;
  background: var(--sre-bg-elevated);
  color: var(--sre-text-secondary);
}
.card-badge[data-type='alertmanager'] { background: rgba(230, 82, 44, 0.14); color: #e6522c; }
.card-badge[data-type='grafana']      { background: rgba(245, 158, 11, 0.14); color: #f59e0b; }
.card-badge[data-type='standard']     {
  background: var(--sre-primary-soft, rgba(37, 99, 235, 0.1));
  color: var(--sre-primary, #2563eb);
}
.card-badge-mode {
  font-size: 11px;
  padding: 2px 8px;
  border-radius: 4px;
  background: var(--sre-bg-elevated);
  color: var(--sre-text-secondary);
  font-weight: 500;
}

.card-desc {
  font-size: 12px;
  color: var(--sre-text-secondary);
  line-height: 1.5;
  overflow: hidden;
  text-overflow: ellipsis;
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
  min-height: 36px;
}

.card-webhook {
  display: flex;
  align-items: center;
  gap: 4px;
  background: var(--sre-bg-elevated);
  border-radius: 6px;
  padding: 6px 8px;
}
.webhook-url {
  font-family: var(--sre-font-mono, ui-monospace, SFMono-Regular, monospace);
  font-size: 11px;
  color: var(--sre-text-tertiary);
  flex: 1;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.card-footer {
  display: flex;
  align-items: center;
  gap: 10px;
  font-size: 12px;
  color: var(--sre-text-tertiary);
  border-top: var(--sre-hairline);
  padding-top: 10px;
  margin-top: auto;
}
.card-footer-channel {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.card-actions {
  display: flex;
  align-items: center;
  justify-content: space-between;
  border-top: var(--sre-hairline);
  margin: 0 -20px -20px;
  padding: 8px 16px;
  gap: 8px;
}

/* Empty state */
.integ-empty {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  text-align: center;
  padding: 80px 24px;
  border: var(--sre-hairline);
  border-radius: var(--sre-radius-md, 10px);
  background: var(--sre-bg-card, #fff);
  gap: 8px;
}
.integ-empty-icon { color: var(--sre-text-tertiary); margin-bottom: 8px; }
.integ-empty-title {
  font-size: 16px;
  font-weight: 600;
  color: var(--sre-text-primary);
}
.integ-empty-sub {
  font-size: 13px;
  color: var(--sre-text-secondary);
  margin-bottom: 12px;
  max-width: 360px;
}
</style>
