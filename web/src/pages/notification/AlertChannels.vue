<script setup lang="ts">
import { computed, onMounted, reactive, ref, shallowRef } from 'vue'
import { useMessage, useDialog } from 'naive-ui'
import { useI18n } from 'vue-i18n'
import {
  AddOutline,
  ChatbubblesOutline,
  CopyOutline,
  EllipsisHorizontal,
  SearchOutline,
} from '@vicons/ionicons5'
import { alertChannelApi, notifyMediaApi, messageTemplateApi } from '@/api'
import type { AlertChannel, NotifyMedia, MessageTemplate } from '@/types'
import { getErrorMessage } from '@/utils/format'
import { usePaginatedList } from '@/composables'
import KVEditor from '@/components/common/KVEditor.vue'
import EmptyState from '@/components/common/EmptyState.vue'

const message = useMessage()
const dialog = useDialog()
const { t } = useI18n()

const {
  loading,
  items: channels,
  total,
  page,
  pageSize,
  fetchList,
  refresh,
} = usePaginatedList<AlertChannel>({
  apiFn: alertChannelApi.list,
  pageSize: 50,
  onError: (err: unknown) => {
    message.error((err as Error)?.message)
  },
})

const search = ref('')
const statusFilter = ref<'all' | 'enabled' | 'disabled'>('all')

const showModal = ref(false)
const modalTitle = ref('')
const editingId = ref<number | null>(null)
const saving = ref(false)
const testingId = ref<number | null>(null)

const mediaList = shallowRef<NotifyMedia[]>([])
const templateList = shallowRef<MessageTemplate[]>([])

const severityOptions = [
  { label: () => t('alert.critical'), value: 'critical' },
  { label: () => t('alert.warning'), value: 'warning' },
  { label: () => t('alert.info'), value: 'info' },
]

const statusOptions = computed(() => [
  { label: t('common.all') || 'All', value: 'all' },
  { label: t('common.enabled'), value: 'enabled' },
  { label: t('common.disabled'), value: 'disabled' },
])

const form = reactive({
  name: '',
  description: '',
  match_labels: [] as { key: string; value: string }[],
  severities: [] as string[],
  media_id: null as number | null,
  template_id: null as number | null,
  throttle_min: 5,
  is_enabled: true,
})

const mediaOptions = computed(() =>
  mediaList.value.map((m) => ({ label: m.name, value: m.id })),
)
const templateOptions = computed(() => [
  { label: t('alertChannel.defaultTemplate'), value: undefined as number | undefined },
  ...templateList.value.map((tp) => ({ label: tp.name, value: tp.id })),
])

const filteredChannels = computed(() => {
  const q = search.value.trim().toLowerCase()
  return channels.value.filter((c) => {
    if (statusFilter.value === 'enabled' && !c.is_enabled) return false
    if (statusFilter.value === 'disabled' && c.is_enabled) return false
    if (q) {
      const hay = [
        c.name,
        c.description,
        ...Object.entries(c.match_labels || {}).map(([k, v]) => `${k}=${v}`),
      ]
        .join(' ')
        .toLowerCase()
      if (!hay.includes(q)) return false
    }
    return true
  })
})

function severityBadges(severitiesStr: string) {
  if (!severitiesStr) return []
  return severitiesStr
    .split(',')
    .map((s) => s.trim())
    .filter(Boolean)
}

function labelEntries(matchLabels: Record<string, string>) {
  return Object.entries(matchLabels || {})
}

function mediaName(id: number) {
  return mediaList.value.find((m) => m.id === id)?.name || `#${id}`
}

function mediaWebhookHint(id: number) {
  const media = mediaList.value.find((m) => m.id === id)
  if (!media) return ''
  const cfg = media.config
  try {
    const parsed = JSON.parse(cfg) as Record<string, unknown>
    return String(parsed.webhook || parsed.url || '')
  } catch {
    return cfg
  }
}

function shortUrl(url: string) {
  if (!url) return ''
  if (url.length <= 56) return url
  return url.slice(0, 36) + '…' + url.slice(-16)
}

async function copyText(text: string) {
  if (!text) return
  try {
    await navigator.clipboard.writeText(text)
    message.success(t('common.copied') || 'Copied')
  } catch {
    message.error(t('common.copyFailed'))
  }
}

async function fetchMedia() {
  try {
    const { data } = await notifyMediaApi.list({ page: 1, page_size: 200 })
    mediaList.value = data.data.list || []
  } catch {
    /* ignore */
  }
}

async function fetchTemplates() {
  try {
    const { data } = await messageTemplateApi.list({ page: 1, page_size: 200 })
    templateList.value = data.data.list || []
  } catch {
    /* ignore */
  }
}

function resetForm() {
  form.name = ''
  form.description = ''
  form.match_labels = []
  form.severities = []
  form.media_id = null
  form.template_id = null
  form.throttle_min = 5
  form.is_enabled = true
}

function openCreate() {
  editingId.value = null
  resetForm()
  modalTitle.value = t('alertChannel.create')
  showModal.value = true
}

function openEdit(row: AlertChannel) {
  editingId.value = row.id
  form.name = row.name
  form.description = row.description
  form.match_labels = Object.entries(row.match_labels || {}).map(([key, value]) => ({ key, value }))
  form.severities = row.severities ? row.severities.split(',').map((s) => s.trim()).filter(Boolean) : []
  form.media_id = row.media_id
  form.template_id = row.template_id
  form.throttle_min = row.throttle_min
  form.is_enabled = row.is_enabled
  modalTitle.value = t('common.edit')
  showModal.value = true
}

function buildPayload() {
  const matchLabels: Record<string, string> = {}
  form.match_labels.forEach(({ key, value }) => {
    if (key.trim()) matchLabels[key.trim()] = value
  })
  return {
    name: form.name,
    description: form.description,
    match_labels: matchLabels,
    severities: form.severities.join(','),
    media_id: form.media_id as number,
    template_id: form.template_id,
    throttle_min: form.throttle_min,
    is_enabled: form.is_enabled,
  }
}

async function handleSave() {
  if (!form.name.trim()) {
    message.warning(t('alertChannel.nameRequired'))
    return
  }
  if (!form.media_id) {
    message.warning(t('alertChannel.mediaRequired'))
    return
  }
  saving.value = true
  try {
    const payload = buildPayload()
    if (editingId.value) {
      await alertChannelApi.update(editingId.value, payload)
      message.success(t('alertChannel.updated'))
    } else {
      await alertChannelApi.create(payload)
      message.success(t('alertChannel.created'))
    }
    showModal.value = false
    fetchList()
  } catch (err: unknown) {
    message.error(getErrorMessage(err))
  } finally {
    saving.value = false
  }
}

async function handleDelete(id: number) {
  try {
    await alertChannelApi.delete(id)
    message.success(t('alertChannel.deleted'))
    fetchList()
  } catch (err: unknown) {
    message.error(getErrorMessage(err))
  }
}

async function handleTest(id: number) {
  testingId.value = id
  try {
    await alertChannelApi.test(id)
    message.success(t('alertChannel.testSuccess'))
  } catch (err: unknown) {
    message.error(getErrorMessage(err) || t('alertChannel.testFailed'))
  } finally {
    testingId.value = null
  }
}

function rowMenuOptions(row: AlertChannel) {
  return [
    { label: t('alertChannel.testSend'), key: 'test' },
    { label: t('common.edit'), key: 'edit' },
    { type: 'divider' as const, key: 'd1' },
    { label: t('common.delete'), key: 'delete', props: { style: 'color: var(--sre-danger)' } },
  ]
}

function onMenuSelect(key: string, row: AlertChannel) {
  if (key === 'test') handleTest(row.id)
  else if (key === 'edit') openEdit(row)
  else if (key === 'delete') {
    dialog.warning({
      title: t('common.confirmDelete'),
      content: t('alertChannel.deleteConfirm'),
      positiveText: t('common.confirm'),
      negativeText: t('common.cancel'),
      onPositiveClick: () => handleDelete(row.id),
    })
  }
}

onMounted(() => {
  fetchList()
  fetchMedia()
  fetchTemplates()
})
</script>

<template>
  <div class="ac-page">
    <!-- Toolbar -->
    <div class="ac-toolbar">
      <div class="ac-toolbar-left">
        <n-input
          v-model:value="search"
          size="small"
          :placeholder="t('common.search') || 'Search'"
          clearable
          class="ac-search-input"
        >
          <template #prefix>
            <n-icon :component="SearchOutline" />
          </template>
        </n-input>
        <n-select
          v-model:value="statusFilter"
          size="small"
          :options="statusOptions"
          class="ac-status-select"
        />
      </div>
      <div class="ac-toolbar-right">
        <n-button type="primary" size="small" @click="openCreate">
          <template #icon><n-icon :component="AddOutline" /></template>
          {{ t('alertChannel.create') }}
        </n-button>
      </div>
    </div>

    <!-- List -->
    <n-spin :show="loading">
      <EmptyState
        v-if="filteredChannels.length === 0 && !loading"
        :icon="ChatbubblesOutline"
        :title="t('alertChannel.noData')"
        :primary-text="t('alertChannel.create')"
        @primary="openCreate"
      />

      <ul v-else class="ac-list sre-stagger">
        <li
          v-for="row in filteredChannels"
          :key="row.id"
          class="ac-row sre-lift"
          @click="openEdit(row)"
        >
          <div class="ac-headline">
            <span class="sre-dot" :data-severity="row.is_enabled ? 'success' : 'tertiary'" />
            <span class="ac-name">{{ row.name }}</span>
            <span class="ac-status">{{ row.is_enabled ? t('common.enabled') : t('common.disabled') }}</span>
            <span class="ac-row-spacer" />
            <n-dropdown
              trigger="click"
              :options="rowMenuOptions(row)"
              @select="(k: string) => onMenuSelect(k, row)"
            >
              <button type="button" class="ac-menu-btn" @click.stop>
                <n-icon :component="EllipsisHorizontal" :size="16" />
              </button>
            </n-dropdown>
          </div>

          <div class="ac-line">
            <span class="ac-line-label">{{ t('alertChannel.match') }}</span>
            <div v-if="labelEntries(row.match_labels).length" class="ac-match">
              <span
                v-for="[k, v] in labelEntries(row.match_labels)"
                :key="k"
                class="ac-chip"
              >{{ k }}={{ v }}</span>
            </div>
            <span v-else class="ac-muted">—</span>

            <template v-if="severityBadges(row.severities).length">
              <span class="sre-meta-divider">·</span>
              <span class="ac-line-label">{{ t('alertChannel.severityLabel') }}</span>
              <div class="ac-match">
                <span
                  v-for="s in severityBadges(row.severities)"
                  :key="s"
                  class="ac-chip ac-chip-sev"
                  :data-severity="s"
                >{{ s.toUpperCase() }}</span>
              </div>
            </template>
          </div>

          <div class="ac-line">
            <span class="ac-line-label">{{ t('alertChannel.webhookLabel') }}</span>
            <span class="ac-webhook">{{ mediaName(row.media_id) }}</span>
            <template v-if="mediaWebhookHint(row.media_id)">
              <span class="sre-meta-divider">·</span>
              <span class="ac-webhook ac-webhook-url" :title="mediaWebhookHint(row.media_id)">
                {{ shortUrl(mediaWebhookHint(row.media_id)) }}
              </span>
              <button
                type="button"
                class="ac-copy"
                :title="t('common.copy') || 'Copy'"
                @click.stop="copyText(mediaWebhookHint(row.media_id))"
              >
                <n-icon :component="CopyOutline" :size="12" />
              </button>
            </template>
          </div>

          <div class="ac-line ac-line-meta">
            <span class="ac-line-label">{{ t('alertChannel.throttleLabel') }}</span>
            <span class="tnum">{{ row.throttle_min }} {{ t('alertChannel.throttleUnit') }}</span>
            <template v-if="row.template_id">
              <span class="sre-meta-divider">·</span>
              <span>{{ t('alertChannel.template') }}: {{ templateList.find(tp => tp.id === row.template_id)?.name || '#' + row.template_id }}</span>
            </template>
          </div>
        </li>
      </ul>
    </n-spin>

    <!-- Create / Edit Modal -->
    <n-modal
      v-model:show="showModal"
      :title="modalTitle"
      preset="card"
      :bordered="false"
      class="ac-modal"
    >
      <n-form label-placement="left" label-width="100" size="medium">
        <n-form-item :label="t('common.name')" required>
          <n-input v-model:value="form.name" :placeholder="t('alertChannel.nameRequired')" clearable />
        </n-form-item>
        <n-form-item :label="t('common.description')">
          <n-input v-model:value="form.description" type="textarea" :rows="2" clearable />
        </n-form-item>
        <n-form-item :label="t('alertChannel.matchLabels')">
          <KVEditor v-model:modelValue="form.match_labels" :add-label="t('alertChannel.addLabel')" />
        </n-form-item>
        <n-form-item :label="t('alertChannel.severities')">
          <n-select
            v-model:value="form.severities"
            :options="severityOptions"
            multiple
            :placeholder="t('common.selectSeverities')"
            clearable
            class="ac-form-select"
          />
        </n-form-item>
        <n-form-item :label="t('alertChannel.mediaLabel')" required>
          <n-select
            v-model:value="form.media_id"
            :options="mediaOptions"
            :placeholder="t('alertChannel.mediaRequired')"
            clearable
            class="ac-form-select"
          />
        </n-form-item>
        <n-form-item :label="t('alertChannel.template')">
          <n-select
            v-model:value="form.template_id"
            :options="templateOptions"
            clearable
            class="ac-form-select"
          />
        </n-form-item>
        <n-form-item :label="t('alertChannel.throttle')">
          <n-input-number v-model:value="form.throttle_min" :min="0" :max="10080" class="ac-form-throttle" />
        </n-form-item>
        <n-form-item :label="t('common.enabled')">
          <n-switch v-model:value="form.is_enabled" />
        </n-form-item>
      </n-form>
      <template #footer>
        <div class="ac-modal-footer">
          <n-button size="small" @click="showModal = false">{{ t('common.cancel') }}</n-button>
          <n-button size="small" type="primary" :loading="saving" @click="handleSave">
            {{ t('common.save') }}
          </n-button>
        </div>
      </template>
    </n-modal>
  </div>
</template>

<style scoped>
.ac-page {
  font-family: var(--sre-font-sans, var(--sre-font-sans), system-ui, sans-serif);
  display: flex;
  flex-direction: column;
  gap: 14px;
}
.ac-toolbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
}
.ac-toolbar-left {
  display: flex;
  align-items: center;
  gap: 8px;
}
.ac-search-input { width: 240px; }
.ac-status-select { width: 140px; }
.ac-list {
  list-style: none;
  margin: 0;
  padding: 0;
  display: flex;
  flex-direction: column;
  gap: 8px;
}
.ac-row {
  display: flex;
  flex-direction: column;
  gap: 6px;
  padding: 14px 18px;
  background: var(--sre-bg-card);
  border: var(--sre-hairline);
  border-radius: var(--sre-radius-md, 10px);
  cursor: pointer;
  transition: all var(--sre-duration-fast, 120ms) var(--sre-ease-out, ease);
}
.ac-row:hover {
  border-color: var(--sre-border-strong);
  background: var(--sre-bg-hover);
}
.ac-headline {
  display: flex;
  align-items: center;
  gap: 8px;
}
.ac-name {
  font-size: 14px;
  font-weight: 600;
  color: var(--sre-text-primary);
  letter-spacing: -0.005em;
}
.ac-status {
  font-size: 11px;
  font-weight: 500;
  color: var(--sre-text-secondary);
  text-transform: uppercase;
  letter-spacing: 0.6px;
}
.ac-row-spacer { flex: 1; }
.ac-menu-btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 24px;
  height: 24px;
  border-radius: 6px;
  border: none;
  background: transparent;
  color: var(--sre-text-tertiary);
  cursor: pointer;
  transition: all var(--sre-duration-fast, 120ms) var(--sre-ease-out, ease);
}
.ac-menu-btn:hover {
  background: var(--sre-bg-elevated);
  color: var(--sre-text-primary);
}

.ac-line {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 6px;
  font-size: 12px;
  color: var(--sre-text-tertiary);
}
.ac-line-label {
  font-size: 10px;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.7px;
  color: var(--sre-text-tertiary);
  opacity: 0.7;
  margin-right: 2px;
}
.ac-line-meta { color: var(--sre-text-tertiary); }
.ac-muted { color: var(--sre-text-tertiary); opacity: 0.7; }

.ac-match {
  display: flex;
  flex-wrap: wrap;
  gap: 4px;
}
.ac-chip {
  font-family: var(--sre-font-mono, 'Geist Mono', monospace);
  font-size: 11px;
  background: var(--sre-bg-elevated);
  border-radius: 4px;
  padding: 2px 6px;
  color: var(--sre-text-secondary);
  border: 1px solid var(--sre-border);
  line-height: 1.4;
}
.ac-chip-sev { font-weight: 600; letter-spacing: 0.4px; }
.ac-chip-sev[data-severity='critical'] { color: var(--sre-critical); border-color: var(--sre-critical-soft); }
.ac-chip-sev[data-severity='warning'] { color: var(--sre-warning); border-color: var(--sre-warning-soft); }
.ac-chip-sev[data-severity='info'] { color: var(--sre-info); border-color: var(--sre-info-soft); }

.ac-webhook {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  font-family: var(--sre-font-mono, 'Geist Mono', monospace);
  font-size: 11px;
  color: var(--sre-text-secondary);
}
.ac-webhook-url { color: var(--sre-text-tertiary); }
.ac-copy {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 18px;
  height: 18px;
  border: none;
  background: transparent;
  color: var(--sre-text-tertiary);
  cursor: pointer;
  border-radius: 4px;
  transition: all var(--sre-duration-fast, 120ms) var(--sre-ease-out, ease);
}
.ac-copy:hover {
  background: var(--sre-bg-elevated);
  color: var(--sre-text-primary);
}

.ac-empty {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 64px 24px;
  gap: 12px;
  text-align: center;
}
.ac-empty-icon { color: var(--sre-text-tertiary); opacity: 0.5; }
.ac-empty-title {
  font-size: 14px;
  color: var(--sre-text-secondary);
  margin-bottom: 4px;
}

/* Modal */
.ac-modal { width: 560px; }
.ac-modal-footer { display: flex; justify-content: flex-end; gap: 8px; }
.ac-form-select { width: 100%; }
.ac-form-throttle { width: 160px; }
</style>
