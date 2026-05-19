<script setup lang="ts">
import { reactive, ref, shallowRef, computed, onMounted, h, type Component } from 'vue'
import { useMessage, NDropdown } from 'naive-ui'
import { useI18n } from 'vue-i18n'
import { notifyMediaApi } from '@/api'
import type { NotifyMedia } from '@/types'
import { getErrorMessage } from '@/utils/format'
import {
  AddOutline,
  SearchOutline,
  EllipsisHorizontal,
  ChatbubblesOutline,
  MailOutline,
  GlobeOutline,
  TerminalOutline,
  FlashOutline,
} from '@vicons/ionicons5'
import KVEditor from '@/components/common/KVEditor.vue'
import PageHeader from '@/components/common/PageHeader.vue'
import LoadingSkeleton from '@/components/common/LoadingSkeleton.vue'

const message = useMessage()
const { t } = useI18n()

const loading = ref(false)
const mediaList = shallowRef<NotifyMedia[]>([])
const showModal = ref(false)
const modalTitle = ref('')
const editingId = ref<number | null>(null)
const saving = ref(false)
const testingId = ref<number | null>(null)

const search = ref('')
const typeFilter = ref<string>('')

type MediaType = 'lark_webhook' | 'email' | 'http' | 'script'

const form = reactive({
  name: '',
  description: '',
  type: 'lark_webhook' as MediaType,
  is_enabled: true,
  variables: '{}',
  webhook_url: '',
  smtp_host: '',
  smtp_port: 25,
  username: '',
  password: '',
  from: '',
  method: 'POST',
  url: '',
  headers: [] as { key: string; value: string }[],
  body: '',
  path: '',
  args: '',
})

const typeOptions = computed(() => [
  { label: t('media.larkWebhook'), value: 'lark_webhook' },
  { label: t('media.email'), value: 'email' },
  { label: t('media.http'), value: 'http' },
  { label: t('media.script'), value: 'script' },
])

const filterTypeOptions = computed(() => [
  { label: t('common.all'), value: '' },
  ...typeOptions.value,
])

const methodOptions = [
  { label: 'GET', value: 'GET' },
  { label: 'POST', value: 'POST' },
  { label: 'PUT', value: 'PUT' },
  { label: 'PATCH', value: 'PATCH' },
]

function getTypeLabel(type: string) {
  const map: Record<string, string> = {
    lark_webhook: t('media.typeLark'),
    email: t('media.typeEmail'),
    http: t('media.typeHttp'),
    script: t('media.typeScript'),
  }
  return map[type] || type
}

function getTypeIcon(type: string) {
  const map: Record<string, Component> = {
    lark_webhook: ChatbubblesOutline,
    email: MailOutline,
    http: GlobeOutline,
    script: TerminalOutline,
  }
  return map[type] || FlashOutline
}

function getTargetSummary(row: NotifyMedia): string {
  try {
    const cfg = JSON.parse(row.config || '{}')
    switch (row.type) {
      case 'lark_webhook':
        return cfg.webhook_url ? cfg.webhook_url.replace(/^https?:\/\//, '') : '—'
      case 'email':
        return cfg.from ? `${cfg.from} via ${cfg.smtp_host}:${cfg.smtp_port}` : (cfg.smtp_host || '—')
      case 'http':
        return `${cfg.method || 'POST'} ${cfg.url || ''}`.trim()
      case 'script':
        return cfg.path || '—'
      default:
        return '—'
    }
  } catch {
    return '—'
  }
}

const filtered = computed(() => {
  const q = search.value.trim().toLowerCase()
  return mediaList.value.filter(m => {
    if (typeFilter.value && m.type !== typeFilter.value) return false
    if (!q) return true
    return (
      m.name.toLowerCase().includes(q) ||
      (m.description || '').toLowerCase().includes(q) ||
      getTargetSummary(m).toLowerCase().includes(q)
    )
  })
})

async function fetchData() {
  loading.value = true
  try {
    const { data } = await notifyMediaApi.list({ page: 1, page_size: 100 })
    mediaList.value = data.data.list || []
  } catch (err: unknown) {
    message.error(getErrorMessage(err))
  } finally {
    loading.value = false
  }
}

function parseConfig(configStr: string): Record<string, unknown> {
  try { return JSON.parse(configStr || '{}') } catch { return {} }
}

function buildConfigString(): string {
  switch (form.type) {
    case 'lark_webhook':
      return JSON.stringify({ webhook_url: form.webhook_url }, null, 2)
    case 'email':
      return JSON.stringify({
        smtp_host: form.smtp_host, smtp_port: form.smtp_port,
        username: form.username, password: form.password, from: form.from,
      }, null, 2)
    case 'http': {
      const hdrs: Record<string, string> = {}
      for (const h of form.headers) { if (h.key.trim()) hdrs[h.key.trim()] = h.value }
      return JSON.stringify({ method: form.method, url: form.url, headers: hdrs, body: form.body }, null, 2)
    }
    case 'script':
      return JSON.stringify({ path: form.path, args: form.args }, null, 2)
    default:
      return '{}'
  }
}

function resetForm() {
  Object.assign(form, {
    name: '', description: '', type: 'lark_webhook', is_enabled: true, variables: '{}',
    webhook_url: '', smtp_host: '', smtp_port: 25, username: '', password: '', from: '',
    method: 'POST', url: '', headers: [], body: '', path: '', args: '',
  })
}

function openCreate() {
  editingId.value = null
  modalTitle.value = t('media.create')
  resetForm()
  showModal.value = true
}

function openEdit(row: NotifyMedia) {
  editingId.value = row.id
  modalTitle.value = t('media.edit')
  const cfg = parseConfig(row.config)
  Object.assign(form, {
    name: row.name, description: row.description, type: row.type,
    is_enabled: row.is_enabled, variables: row.variables || '{}',
    webhook_url: cfg.webhook_url || '',
    smtp_host: cfg.smtp_host || '', smtp_port: cfg.smtp_port || 25,
    username: cfg.username || '', password: cfg.password || '', from: cfg.from || '',
    method: cfg.method || 'POST', url: cfg.url || '',
    headers: Object.entries(cfg.headers || {}).map(([key, value]) => ({ key, value: String(value) })),
    body: cfg.body || '', path: cfg.path || '', args: cfg.args || '',
  })
  showModal.value = true
}

async function handleSave() {
  if (!form.name.trim()) { message.warning(t('media.nameRequired')); return }
  try { JSON.parse(form.variables) } catch { message.warning(t('media.variables') + ': ' + t('media.invalidJson')); return }
  saving.value = true
  try {
    const payload = {
      name: form.name, description: form.description, type: form.type,
      is_enabled: form.is_enabled, config: buildConfigString(), variables: form.variables,
    }
    if (editingId.value) {
      await notifyMediaApi.update(editingId.value, payload)
      message.success(t('media.updated'))
    } else {
      await notifyMediaApi.create(payload)
      message.success(t('media.created'))
    }
    showModal.value = false
    fetchData()
  } catch (err: unknown) { message.error(getErrorMessage(err)) } finally { saving.value = false }
}

async function handleDelete(id: number) {
  try {
    await notifyMediaApi.delete(id)
    message.success(t('media.deleted'))
    fetchData()
  } catch (err: unknown) { message.error(getErrorMessage(err)) }
}

async function handleTest(id: number) {
  testingId.value = id
  try {
    const { data } = await notifyMediaApi.test(id)
    if (data.data.success) message.success(t('media.testSuccess'))
    else message.warning(`${t('media.testFailed')}: ${data.data.message}`)
  } catch (err: unknown) { message.error(getErrorMessage(err)) } finally { testingId.value = null }
}

function rowMenuOptions(row: NotifyMedia) {
  return [
    { label: t('common.edit'), key: 'edit' },
    { label: t('common.test'), key: 'test' },
    { type: 'divider', key: 'd1' },
    {
      label: t('common.delete'), key: 'delete',
      disabled: row.is_builtin,
      props: { style: row.is_builtin ? '' : 'color: var(--sre-danger)' },
    },
  ]
}

function onRowMenu(key: string, row: NotifyMedia) {
  if (key === 'edit') openEdit(row)
  else if (key === 'test') handleTest(row.id)
  else if (key === 'delete' && !row.is_builtin) {
    if (confirm(t('media.deleteConfirm'))) handleDelete(row.id)
  }
}

// Render dropdown trigger via h to keep template light
const RowMenu = (row: NotifyMedia) => h(NDropdown, {
  trigger: 'click',
  options: rowMenuOptions(row),
  onSelect: (key: string) => onRowMenu(key, row),
}, {
  default: () => h('button', { class: 'sre-icon-btn', 'aria-label': t('common.actions') },
    h('span', { class: 'sre-dots' })),
})

onMounted(fetchData)
</script>

<template>
  <div class="media-page">
    <PageHeader :title="t('media.title')" :subtitle="t('media.subtitle')">
      <template #actions>
        <n-button type="primary" size="small" @click="openCreate">
          <template #icon><n-icon :component="AddOutline" /></template>
          {{ t('media.create') }}
        </n-button>
      </template>
    </PageHeader>

    <div class="toolbar">
      <n-input v-model:value="search" size="small" :placeholder="t('common.search')" clearable style="width: 240px">
        <template #prefix><n-icon :component="SearchOutline" /></template>
      </n-input>
      <n-select v-model:value="typeFilter" size="small" :options="filterTypeOptions" style="width: 160px" />
      <span class="count tnum">{{ filtered.length }} / {{ mediaList.length }}</span>
    </div>

    <LoadingSkeleton v-if="loading && filtered.length === 0" :rows="4" variant="row" />

    <div v-else-if="filtered.length === 0" class="empty">
      <n-icon :component="FlashOutline" size="36" />
      <div class="empty-text">{{ t('media.noData') }}</div>
      <n-button type="primary" size="small" @click="openCreate">{{ t('media.create') }}</n-button>
    </div>

    <ul v-else class="row-list sre-stagger">
      <li v-for="m in filtered" :key="m.id" class="sre-notify-card sre-lift" :data-type="m.type">
        <div class="row-l1">
          <span class="type-icon" :data-type="m.type"><n-icon :component="getTypeIcon(m.type)" size="16" /></span>
          <span class="row-name">{{ m.name }}</span>
          <span class="type-chip" :data-type="m.type">{{ getTypeLabel(m.type) }}</span>
          <span v-if="m.is_builtin" class="builtin-chip">{{ t('media.builtin') }}</span>
          <span class="status-text" :class="{ off: !m.is_enabled }">
            {{ m.is_enabled ? t('common.on') : t('common.off') }}
          </span>
          <div class="row-actions">
            <n-button quaternary size="tiny" :loading="testingId === m.id" @click="handleTest(m.id)">
              {{ t('common.test') }}
            </n-button>
            <component :is="RowMenu(m)" />
          </div>
        </div>
        <div class="row-l2">
          <code class="target tnum">{{ getTargetSummary(m) }}</code>
        </div>
        <div class="row-l3" v-if="m.description">
          <span class="meta">{{ m.description }}</span>
        </div>
      </li>
    </ul>

    <n-modal v-model:show="showModal" preset="card" :title="modalTitle" :bordered="false" class="media-modal">
      <n-form label-placement="top">
        <n-grid :x-gap="12" :cols="2">
          <n-gi>
            <n-form-item :label="t('media.name')" required>
              <n-input v-model:value="form.name" :placeholder="t('media.namePlaceholder')" />
            </n-form-item>
          </n-gi>
          <n-gi>
            <n-form-item :label="t('media.type')">
              <n-select v-model:value="form.type" :options="typeOptions" />
            </n-form-item>
          </n-gi>
        </n-grid>

        <n-form-item :label="t('media.description')">
          <n-input v-model:value="form.description" :placeholder="t('media.description')" />
        </n-form-item>

        <n-divider style="margin: 12px 0">{{ t('media.config') }}</n-divider>

        <template v-if="form.type === 'lark_webhook'">
          <n-form-item :label="t('media.webhookUrl')" required>
            <n-input v-model:value="form.webhook_url" :placeholder="t('mediaMgmt.webhookUrlPlaceholder')" />
          </n-form-item>
        </template>

        <template v-if="form.type === 'email'">
          <n-grid :x-gap="12" :cols="2">
            <n-gi>
              <n-form-item :label="t('media.smtpHost')">
                <n-input v-model:value="form.smtp_host" :placeholder="t('mediaMgmt.smtpHostPlaceholder')" />
              </n-form-item>
            </n-gi>
            <n-gi>
              <n-form-item :label="t('media.smtpPort')">
                <n-input-number v-model:value="form.smtp_port" :min="1" :max="65535" style="width: 100%" />
              </n-form-item>
            </n-gi>
          </n-grid>
          <n-grid :x-gap="12" :cols="2">
            <n-gi>
              <n-form-item :label="t('media.username')">
                <n-input v-model:value="form.username" :placeholder="t('mediaMgmt.usernamePlaceholder')" />
              </n-form-item>
            </n-gi>
            <n-gi>
              <n-form-item :label="t('media.password')">
                <n-input v-model:value="form.password" type="password" show-password-on="click" :placeholder="t('media.passwordPlaceholder')" />
              </n-form-item>
            </n-gi>
          </n-grid>
          <n-form-item :label="t('media.from')">
            <n-input v-model:value="form.from" :placeholder="t('mediaMgmt.fromPlaceholder')" />
          </n-form-item>
        </template>

        <template v-if="form.type === 'http'">
          <n-grid :x-gap="12" :cols="4">
            <n-gi>
              <n-form-item :label="t('media.method')">
                <n-select v-model:value="form.method" :options="methodOptions" />
              </n-form-item>
            </n-gi>
            <n-gi :span="3">
              <n-form-item :label="t('media.url')">
                <n-input v-model:value="form.url" :placeholder="t('mediaMgmt.httpUrlPlaceholder')" />
              </n-form-item>
            </n-gi>
          </n-grid>
          <n-form-item :label="t('media.headers')">
            <KVEditor v-model:modelValue="form.headers" :key-placeholder="t('media.headerName')" :value-placeholder="t('media.headerValue')" :add-label="t('media.addHeader')" />
          </n-form-item>
          <n-form-item :label="t('media.body')">
            <n-input v-model:value="form.body" type="textarea" :rows="4"
              :placeholder="t('mediaMgmt.httpBodyPlaceholder')"
              style="font-family: var(--sre-font-mono); font-size: 12px" />
          </n-form-item>
        </template>

        <template v-if="form.type === 'script'">
          <n-form-item :label="t('media.path')">
            <n-input v-model:value="form.path" :placeholder="t('mediaMgmt.scriptPathPlaceholder')" />
          </n-form-item>
          <n-form-item :label="t('media.args')">
            <n-input v-model:value="form.args" :placeholder="t('mediaMgmt.scriptArgsPlaceholder')" />
          </n-form-item>
        </template>

        <n-divider style="margin: 12px 0" />

        <n-form-item :label="t('media.variables')">
          <n-input v-model:value="form.variables" type="textarea" :rows="3"
            :placeholder="t('media.variablesHint')"
            style="font-family: var(--sre-font-mono); font-size: 12px" />
        </n-form-item>

        <n-form-item :label="t('common.enabled')">
          <n-switch v-model:value="form.is_enabled" />
        </n-form-item>
      </n-form>

      <template #action>
        <n-space justify="end">
          <n-button @click="showModal = false">{{ t('common.cancel') }}</n-button>
          <n-button type="primary" :loading="saving" @click="handleSave">
            {{ editingId ? t('common.update') : t('common.create') }}
          </n-button>
        </n-space>
      </template>
    </n-modal>
  </div>
</template>

<style scoped>
.media-page { font-family: var(--sre-font-sans); max-width: 1400px; }

.sub-header {
  display: flex; align-items: flex-end; justify-content: space-between;
  padding-bottom: 14px; border-bottom: 1px solid var(--sre-hairline, rgba(255,255,255,0.06));
  margin-bottom: 14px;
}
.sub-title { font: 600 18px/1.2 var(--sre-font-sans), sans-serif; margin: 0; letter-spacing: -0.01em; }
.sub-sub { font-size: 12px; color: var(--sre-text-secondary); margin: 4px 0 0; }

.toolbar { display: flex; gap: 8px; align-items: center; margin-bottom: 12px; }
.count { font-size: 12px; color: var(--sre-text-secondary); margin-left: auto; font-variant-numeric: tabular-nums; }

.loading, .empty { padding: 60px 20px; text-align: center; color: var(--sre-text-secondary); }
.empty { display: flex; flex-direction: column; gap: 12px; align-items: center; }
.empty-text { font-size: 13px; }

.row-list { list-style: none; padding: 0; margin: 0; display: flex; flex-direction: column; gap: 6px; }

.row-l1 { display: flex; align-items: center; gap: 10px; }
.type-icon {
  width: 28px; height: 28px; display: inline-flex; align-items: center; justify-content: center;
  border-radius: 6px; background: var(--sre-bg-elevated);
}
.type-icon[data-type="lark_webhook"] { color: var(--sre-info); background: var(--sre-info-soft); }
.type-icon[data-type="email"]        { color: var(--sre-text-secondary); background: var(--sre-bg-elevated); }
.type-icon[data-type="http"]         { color: var(--sre-success); background: var(--sre-success-soft); }
.type-icon[data-type="script"]       { color: var(--sre-warning); background: var(--sre-warning-soft); }

.row-name { font: 600 14px/1.3 var(--sre-font-sans), sans-serif; letter-spacing: -0.005em; }

.type-chip {
  font: 500 10px/1 var(--sre-font-mono), monospace; text-transform: uppercase;
  padding: 3px 6px; border-radius: 4px; letter-spacing: .04em;
  background: var(--sre-bg-elevated); color: var(--sre-text-secondary);
}
.type-chip[data-type="lark_webhook"] { background: var(--sre-info-soft); color: var(--sre-info); }
.type-chip[data-type="email"]        { background: var(--sre-bg-elevated); color: var(--sre-text-secondary); }
.type-chip[data-type="http"]         { background: var(--sre-success-soft); color: var(--sre-success); }
.type-chip[data-type="script"]       { background: var(--sre-warning-soft); color: var(--sre-warning); }

.builtin-chip {
  font: 500 10px/1 var(--sre-font-mono), monospace; padding: 3px 6px; border-radius: 4px;
  background: var(--sre-info-soft); color: var(--sre-info); letter-spacing: .04em;
}
.status-text { font-size: 11px; color: var(--sre-success); }
.status-text.off { color: var(--sre-text-secondary); }

.row-actions { margin-left: auto; display: flex; align-items: center; gap: 4px; }

.row-l2 { padding-left: 38px; }
.target {
  font: 12px/1.4 var(--sre-font-mono), monospace;
  color: var(--sre-text-secondary);
  font-variant-numeric: tabular-nums;
  word-break: break-all;
}
.row-l3 { padding-left: 38px; }
.meta { font-size: 12px; color: var(--sre-text-secondary); }

.media-modal { width: 600px; }
</style>
