<script setup lang="ts">
import { ref, shallowRef, computed, onMounted, h, type Ref } from 'vue'
import { NButton, NIcon, NInput, NRadioGroup, NRadioButton, NDropdown, NModal, NForm, NFormItem, NSelect, NGrid, NGi, NSwitch, NInputNumber, NSpace, NDrawer, NDrawerContent, NDataTable, NPagination, useMessage } from 'naive-ui'
import { useI18n } from 'vue-i18n'
import { datasourceApi } from '@/api'
import type { DataSource, DataSourceType, DataSourceStatus } from '@/types'
import { kvArrayToRecord } from '@/utils/format'
import { useCrudPage } from '@/composables/useCrudPage'
import type { CrudApiModule } from '@/composables/useCrudPage'
import {
  AddOutline,
  RefreshOutline,
  PulseOutline,
  EllipsisHorizontalOutline,
  ServerOutline,
  SearchOutline,
  CreateOutline,
  TrashOutline,
} from '@vicons/ionicons5'
import KVEditor from '@/components/common/KVEditor.vue'
import PageHeader from '@/components/common/PageHeader.vue'
import EmptyState from '@/components/common/EmptyState.vue'
import LoadingSkeleton from '@/components/common/LoadingSkeleton.vue'

interface DSCard extends DataSource {
  _testing?: boolean
  _latencyMs?: number
  _lastCheckAt?: string
}

const message = useMessage()
const { t } = useI18n()

interface DataSourceForm {
  name: string
  type: DataSourceType
  endpoint: string
  description: string
  auth_type: string
  auth_username: string
  auth_password: string
  auth_token: string
  auth_key_header: string
  auth_key_value: string
  labels: { key: string; value: string }[]
  health_check_interval: number
  is_enabled: boolean
}

const crud = useCrudPage<DataSource>({
  api: datasourceApi as unknown as CrudApiModule<DataSource>,
  defaultForm: () => ({
    name: '', type: 'prometheus' as DataSourceType,
    endpoint: '', description: '',
    auth_type: 'none', auth_username: '', auth_password: '',
    auth_token: '', auth_key_header: '', auth_key_value: '',
    labels: [] as { key: string; value: string }[],
    health_check_interval: 60, is_enabled: true,
  } as unknown as Partial<DataSource>),
  i18nKeys: {
    created: 'datasource.created',
    updated: 'datasource.updated',
    deleted: 'datasource.deleted',
    deleteConfirm: 'datasource.deleteConfirm',
    createTitle: 'datasource.add',
    editTitle: 'common.edit',
  },
  rowToForm: (row) => ({
    name: row.name, type: row.type, endpoint: row.endpoint,
    description: row.description,
    auth_type: row.auth_type || 'none',
    auth_username: '', auth_password: '',
    auth_token: '', auth_key_header: '', auth_key_value: '',
    labels: Object.entries(row.labels || {}).map(([key, value]) => ({ key, value })),
    health_check_interval: row.health_check_interval || 60,
    is_enabled: row.is_enabled,
  } as unknown as Partial<DataSource>),
  formToPayload: (form) => {
    const f = form as unknown as DataSourceForm
    let auth_config = ''
    if (f.auth_type === 'basic' && (f.auth_username || f.auth_password)) {
      auth_config = JSON.stringify({ username: f.auth_username, password: f.auth_password })
    } else if (f.auth_type === 'bearer' && f.auth_token) {
      auth_config = JSON.stringify({ token: f.auth_token })
    } else if (f.auth_type === 'api_key' && f.auth_key_value) {
      auth_config = JSON.stringify({ header: f.auth_key_header || 'X-API-Key', value: f.auth_key_value })
    }
    return {
      name: form.name, type: form.type, endpoint: form.endpoint,
      description: form.description,
      auth_type: f.auth_type, auth_config,
      labels: kvArrayToRecord(f.labels || []),
      health_check_interval: f.health_check_interval,
      is_enabled: form.is_enabled,
    }
  },
  validate: (form) => {
    if (!form.name?.trim()) return t('datasource.nameRequired')
    if (!form.endpoint?.trim()) return t('datasource.endpointRequired')
    return null
  },
  pageSize: 100,
})

const {
  loading,
  items: rawDatasources,
  total,
  page,
  pageSize,
  search,
  showModal,
  modalTitle,
  editingId,
  saving,
  fetchList,
  openCreate,
  openEdit,
  handleSave,
  confirmDelete,
} = crud
const form = crud.form as unknown as Ref<DataSourceForm>
// Cast items to DSCard[] so template can access _testing, _latencyMs, _lastCheckAt
const datasources = rawDatasources as unknown as Ref<DSCard[]>

const typeFilter = ref<'all' | DataSourceType>('all')

// Test connection
const testing = ref(false)
async function testConnection() {
  testing.value = true
  try {
    // Build auth_config from form fields
    let authConfig = ''
    const f = form.value
    if (f.auth_type === 'basic') {
      authConfig = JSON.stringify({ username: f.auth_username, password: f.auth_password })
    } else if (f.auth_type === 'bearer') {
      authConfig = JSON.stringify({ token: f.auth_token })
    } else if (f.auth_type === 'api_key') {
      authConfig = JSON.stringify({ header: f.auth_key_header, key: f.auth_key_value })
    }
    const { data } = await datasourceApi.testConnection({
      type: f.type,
      endpoint: f.endpoint,
      auth_type: f.auth_type || 'none',
      auth_config: authConfig,
    })
    const r = data.data
    message.success(`${r.status} · ${r.latency_ms}ms${r.version ? ' · ' + r.version : ''}`, { duration: 5000 })
  } catch (err: unknown) {
    const axiosErr = err as { response?: { data?: { message?: string } }; message?: string }
    message.error(axiosErr?.response?.data?.message || axiosErr?.message || t('datasource.testConnectionFailed') || '连接失败')
  } finally {
    testing.value = false
  }
}

// Health check history drawer
interface HealthLogEntry {
  id: number
  dsId: number
  time: string
  status: 'healthy' | 'unhealthy'
  latency: string
  error: string
}
const healthDrawerVisible = ref(false)
const healthDrawerTitle = ref('')
const healthDrawerDsId = ref(0)
const allHealthLogs = ref<HealthLogEntry[]>([])
const healthLogs = computed(() => allHealthLogs.value.filter(l => l.dsId === healthDrawerDsId.value))
let healthLogIdSeq = 0

const healthLogColumns = [
  { title: t('datasource.healthLogTime'), key: 'time', width: 180 },
  {
    title: t('common.status'), key: 'status', width: 100,
    render: (row: HealthLogEntry) => row.status === 'healthy' ? t('datasource.healthy') : t('datasource.unhealthy'),
  },
  { title: t('datasource.latency'), key: 'latency', width: 100 },
  { title: t('common.error'), key: 'error', ellipsis: { tooltip: true } },
]

function openHealthDrawer(ds: DSCard) {
  healthDrawerDsId.value = ds.id
  healthDrawerTitle.value = `${ds.name} — ${t('datasource.healthLog')}`
  healthDrawerVisible.value = true
}

const typeOptions = [
  { label: () => t('datasource.typePrometheus'), value: 'prometheus' },
  { label: () => t('datasource.typeVictoriaMetrics'), value: 'victoriametrics' },
  { label: () => t('datasource.typeVictoriaLogs'), value: 'victorialogs' },
  { label: () => t('datasource.typeZabbix'), value: 'zabbix' },
  { label: () => t('datasource.typeElasticsearch'), value: 'elasticsearch' },
]

const authTypeOptions = [
  { label: () => t('datasource.authNone'), value: 'none' },
  { label: () => t('datasource.authBasic'), value: 'basic' },
  { label: () => t('datasource.authBearer'), value: 'bearer' },
  { label: () => t('datasource.authApiKey'), value: 'api_key' },
]

const filteredList = computed(() => {
  const q = search.value.trim().toLowerCase()
  return datasources.value.filter((d) => {
    if (typeFilter.value !== 'all' && d.type !== typeFilter.value) return false
    if (q && !`${d.name} ${d.endpoint} ${d.description}`.toLowerCase().includes(q)) return false
    return true
  })
})

async function testHealth(ds: DSCard) {
  ds._testing = true
  datasources.value = [...datasources.value]
  try {
    const { data } = await datasourceApi.healthCheck(ds.id)
    const r = data.data
    ds._latencyMs = r.latency_ms >= 0 ? r.latency_ms : undefined
    ds._lastCheckAt = new Date().toISOString()
    ds.status = r.status as DataSourceStatus
    if (r.version) ds.version = r.version
    // Push to health log (max 50 entries total)
    healthLogIdSeq++
    allHealthLogs.value = [
      { id: healthLogIdSeq, dsId: ds.id, time: new Date().toLocaleString(), status: r.status as 'healthy' | 'unhealthy', latency: r.latency_ms >= 0 ? `${r.latency_ms}ms` : '—', error: r.message || '' },
      ...allHealthLogs.value,
    ].slice(0, 50)
    if (r.status === 'healthy') {
      message.success(`${ds.name} · ${r.latency_ms}ms${r.version ? ' · ' + r.version : ''}`, { duration: 3500 })
    } else {
      message.error(`${ds.name} · ${r.message}`, { duration: 5000 })
    }
  } catch (err: unknown) {
    message.error((err as Error)?.message || t('common.loadFailed'))
  } finally {
    ds._testing = false
    datasources.value = [...datasources.value]
  }
}

function rowActions(_ds: DSCard) {
  return [
    { label: t('common.edit'), key: 'edit', icon: () => h(NIcon, { component: CreateOutline }) },
    { label: t('common.delete'), key: 'delete', icon: () => h(NIcon, { component: TrashOutline }) },
  ]
}

function handleAction(key: string, ds: DSCard) {
  if (key === 'edit') openEdit(ds)
  else if (key === 'delete') confirmDelete(ds.id)
}

function healthSev(ds: DSCard): 'success' | 'warning' | 'critical' | null {
  if (ds.status === 'healthy') return 'success'
  if (ds.status === 'unhealthy') return 'critical'
  return null
}
function healthLabel(ds: DSCard) {
  if (ds.status === 'healthy') return t('datasource.healthy')
  if (ds.status === 'unhealthy') return t('datasource.unhealthy')
  return t('datasource.unknown')
}
function typeLabel(type: string) {
  const m: Record<string, string> = {
    prometheus: t('datasource.typePrometheus'),
    victoriametrics: t('datasource.typeVictoriaMetrics'),
    victorialogs: t('datasource.typeVictoriaLogs'),
    zabbix: t('datasource.typeZabbix'),
    elasticsearch: t('datasource.typeElasticsearch'),
  }
  return m[type] || type
}
function relTime(iso?: string) {
  if (!iso) return '—'
  const ms = Date.now() - new Date(iso).getTime()
  if (ms < 0 || isNaN(ms)) return '—'
  const s = Math.floor(ms / 1000)
  if (s < 60) return t('common.secsAgo', { n: s })
  const m = Math.floor(s / 60)
  if (m < 60) return t('common.minsAgo', { n: m })
  const hr = Math.floor(m / 60)
  if (hr < 24) return t('common.hoursAgo', { n: hr })
  const d = Math.floor(hr / 24)
  return t('common.daysAgo', { n: d })
}

onMounted(fetchList)
</script>

<template>
  <div class="datasources-page">
    <PageHeader :title="t('datasource.title')" :subtitle="t('datasource.subtitle')">
      <template #actions>
        <NButton quaternary @click="fetchList" :loading="loading">
          <template #icon><NIcon :component="RefreshOutline" /></template>
          {{ t('common.refresh') }}
        </NButton>
        <NButton type="primary" @click="openCreate">
          <template #icon><NIcon :component="AddOutline" /></template>
          {{ t('datasource.add') }}
        </NButton>
      </template>
    </PageHeader>

    <div class="ds-toolbar">
      <NRadioGroup v-model:value="typeFilter" size="small">
        <NRadioButton value="all">{{ t('common.all') }}</NRadioButton>
        <NRadioButton value="prometheus">{{ t('datasource.typePrometheus') }}</NRadioButton>
        <NRadioButton value="victoriametrics">{{ t('datasource.typeVictoriaMetrics') }}</NRadioButton>
        <NRadioButton value="victorialogs">{{ t('datasource.typeVictoriaLogs') }}</NRadioButton>
        <NRadioButton value="zabbix">{{ t('datasource.typeZabbix') }}</NRadioButton>
        <NRadioButton value="elasticsearch">{{ t('datasource.typeElasticsearch') }}</NRadioButton>
      </NRadioGroup>
      <NInput v-model:value="search" :placeholder="t('common.search')" clearable size="small" class="ds-search">
        <template #prefix><NIcon :component="SearchOutline" /></template>
      </NInput>
    </div>

    <LoadingSkeleton v-if="loading" variant="card-grid" :rows="6" />
    <template v-else>
      <div v-if="filteredList.length > 0" class="ds-grid sre-stagger">
        <div
          v-for="(ds, idx) in filteredList"
          :key="ds.id"
          class="ds-card sre-lift"
          :style="{ '--sre-stagger-i': idx } as Record<string, string | number>"
          @click="openEdit(ds)"
        >
          <div class="ds-stripe" :data-type="ds.type"></div>

          <div class="ds-status" @click.stop="openHealthDrawer(ds)" style="cursor: pointer">
            <span class="sre-dot" :data-severity="healthSev(ds) || ''"></span>
            <span class="ds-status-text">{{ healthLabel(ds) }}</span>
            <span v-if="!ds.is_enabled" class="ds-disabled">· {{ t('common.disabled') }}</span>
          </div>

          <div class="ds-name">{{ ds.name }}</div>
          <div class="ds-type">{{ typeLabel(ds.type) }}</div>

          <code class="ds-endpoint" :title="ds.endpoint">{{ ds.endpoint }}</code>

          <div class="ds-stats">
            <div class="ds-stat-row">
              <span class="sre-label-eyebrow">{{ t('datasource.latency') }}</span>
              <span class="ds-stat-val tnum">{{ ds._latencyMs != null ? ds._latencyMs + 'ms' : '—' }}</span>
            </div>
            <div class="ds-stat-row">
              <span class="sre-label-eyebrow">{{ t('datasource.version') }}</span>
              <span class="ds-stat-val mono">{{ ds.version || '—' }}</span>
            </div>
            <div class="ds-stat-row">
              <span class="sre-label-eyebrow">{{ t('datasource.lastCheck') }}</span>
              <span class="ds-stat-val">{{ relTime(ds._lastCheckAt) }}</span>
            </div>
          </div>

          <div class="ds-actions" @click.stop>
            <NButton size="tiny" :loading="ds._testing" @click="testHealth(ds)">
              <template #icon><NIcon :component="PulseOutline" /></template>
              {{ t('common.test') }}
            </NButton>
            <NDropdown :options="rowActions(ds)" trigger="click" @select="handleAction($event, ds)">
              <NButton quaternary circle size="small">
                <template #icon><NIcon :component="EllipsisHorizontalOutline" /></template>
              </NButton>
            </NDropdown>
          </div>
        </div>

        <div v-if="total > pageSize" class="ds-pagination">
          <NPagination
            v-model:page="page"
            :page-size="pageSize"
            :item-count="total"
            :page-slot="7"
            @update:page="fetchList"
          />
        </div>
      </div>

      <div v-else class="ds-empty">
        <EmptyState
          :icon="ServerOutline"
          :title="t('datasource.noData')"
          :primary-text="t('datasource.addFirst')"
          @primary="openCreate"
        />
      </div>
    </template>

    <!-- Health Check History Drawer -->
    <NDrawer v-model:show="healthDrawerVisible" :width="520" placement="right">
      <NDrawerContent :title="healthDrawerTitle">
        <NDataTable
          :columns="healthLogColumns"
          :data="healthLogs.slice(0, 10)"
          :row-key="(r: HealthLogEntry) => r.id"
          size="small"
          :single-line="false"
          striped
        />
      </NDrawerContent>
    </NDrawer>

    <NModal v-model:show="showModal" preset="card" :title="modalTitle" style="width: 560px" :bordered="false">
      <NForm label-placement="top">
        <NFormItem :label="t('common.name')" required>
          <NInput v-model:value="form.name" :placeholder="t('datasourceMgmt.namePlaceholder')" />
        </NFormItem>

        <NGrid :x-gap="12" :cols="2">
          <NGi>
            <NFormItem :label="t('common.type')">
              <NSelect v-model:value="form.type" :options="typeOptions" />
            </NFormItem>
          </NGi>
          <NGi>
            <NFormItem :label="t('datasource.authType')">
              <NSelect v-model:value="form.auth_type" :options="authTypeOptions" />
            </NFormItem>
          </NGi>
        </NGrid>

        <NFormItem :label="t('datasource.endpointUrl')" required>
          <NInput v-model:value="form.endpoint" :placeholder="t('datasourceMgmt.endpointPlaceholder')" />
        </NFormItem>

        <template v-if="form.auth_type === 'basic'">
          <NGrid :x-gap="12" :cols="2">
            <NGi>
              <NFormItem :label="t('datasource.authUsername')">
                <NInput v-model:value="form.auth_username" :placeholder="editingId ? t('datasource.authCredentialsNote') : t('datasource.authUsername')" />
              </NFormItem>
            </NGi>
            <NGi>
              <NFormItem :label="t('datasource.authPassword')">
                <NInput v-model:value="form.auth_password" type="password" show-password-on="click" :placeholder="editingId ? t('datasource.authCredentialsNote') : t('datasource.authPassword')" />
              </NFormItem>
            </NGi>
          </NGrid>
        </template>

        <template v-if="form.auth_type === 'bearer'">
          <NFormItem :label="t('datasource.authToken')">
            <NInput v-model:value="form.auth_token" type="password" show-password-on="click" :placeholder="editingId ? t('datasource.authCredentialsNote') : t('datasource.authToken')" />
          </NFormItem>
        </template>

        <template v-if="form.auth_type === 'api_key'">
          <NGrid :x-gap="12" :cols="2">
            <NGi>
              <NFormItem :label="t('datasource.authApiKeyHeader')">
                <NInput v-model:value="form.auth_key_header" :placeholder="t('datasource.authApiKeyHeaderPlaceholder')" />
              </NFormItem>
            </NGi>
            <NGi>
              <NFormItem :label="t('datasource.authApiKeyValue')">
                <NInput v-model:value="form.auth_key_value" type="password" show-password-on="click" :placeholder="editingId ? t('datasource.authCredentialsNote') : t('datasource.authApiKeyValue')" />
              </NFormItem>
            </NGi>
          </NGrid>
        </template>

        <NFormItem :label="t('common.description')">
          <NInput v-model:value="form.description" type="textarea" :placeholder="t('common.description')" :rows="2" />
        </NFormItem>

        <NGrid :x-gap="12" :cols="2">
          <NGi>
            <NFormItem :label="t('datasource.healthCheckInterval')">
              <NInputNumber v-model:value="form.health_check_interval" :min="10" :max="3600" style="width: 100%" />
            </NFormItem>
          </NGi>
          <NGi>
            <NFormItem :label="t('common.enabled')">
              <NSwitch v-model:value="form.is_enabled" />
            </NFormItem>
          </NGi>
        </NGrid>

        <NFormItem :label="t('datasource.labels')">
          <KVEditor v-model:modelValue="form.labels" :add-label="t('datasource.addLabel')" />
        </NFormItem>
      </NForm>

      <template #action>
        <NSpace justify="end">
          <NButton @click="showModal = false">{{ t('common.cancel') }}</NButton>
          <NButton :loading="testing" @click="testConnection">
            {{ t('datasource.testConnection') || '测试连接' }}
          </NButton>
          <NButton type="primary" :loading="saving" @click="handleSave">
            {{ editingId ? t('common.update') : t('common.create') }}
          </NButton>
        </NSpace>
      </template>
    </NModal>
  </div>
</template>

<style scoped>
.datasources-page {
  max-width: 1400px;
}

.ds-toolbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  margin: 12px 0 20px;
  flex-wrap: wrap;
}
.ds-search {
  max-width: 280px;
}

.ds-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(300px, 1fr));
  gap: 16px;
}

.ds-card {
  position: relative;
  background: var(--sre-bg-card);
  border: var(--sre-hairline);
  border-radius: var(--sre-radius-md, 8px);
  padding: 20px;
  display: flex;
  flex-direction: column;
  gap: 8px;
  cursor: pointer;
  overflow: hidden;
  transition: border-color 180ms ease, box-shadow 180ms ease;
}
.ds-card:hover {
  border-color: var(--sre-primary);
}

.ds-stripe {
  position: absolute;
  top: 0;
  left: 0;
  right: 0;
  height: 3px;
  background: var(--sre-text-tertiary);
}
.ds-stripe[data-type="prometheus"]      { background: var(--sre-ds-prometheus); }
.ds-stripe[data-type="victoriametrics"] { background: var(--sre-ds-victoriametrics); }
.ds-stripe[data-type="victorialogs"]    { background: var(--sre-ds-victorialogs); }
.ds-stripe[data-type="zabbix"]          { background: var(--sre-ds-zabbix); }
.ds-stripe[data-type="elasticsearch"]   { background: var(--sre-ds-elasticsearch); }

.ds-status {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 11px;
  color: var(--sre-text-tertiary);
  text-transform: uppercase;
  letter-spacing: 0.6px;
  margin-top: 4px;
}
.ds-status-text { font-weight: 600; }
.ds-disabled { color: var(--sre-text-tertiary); }

.ds-name {
  font-size: 16px;
  font-weight: 600;
  color: var(--sre-text-primary);
  margin-top: 4px;
  letter-spacing: -0.01em;
}
.ds-type {
  font-size: 12px;
  color: var(--sre-text-secondary);
}

.ds-endpoint {
  font-family: var(--sre-font-mono, ui-monospace, SFMono-Regular, Menlo, monospace);
  font-size: 11px;
  background: var(--sre-bg-elevated);
  border-radius: 4px;
  padding: 4px 8px;
  color: var(--sre-text-tertiary);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  display: block;
  margin-top: 4px;
}

.ds-stats {
  display: flex;
  flex-direction: column;
  gap: 4px;
  padding-top: 10px;
  margin-top: 4px;
  border-top: var(--sre-hairline);
}
.ds-stat-row {
  display: flex;
  justify-content: space-between;
  align-items: center;
  font-size: 12px;
}
.ds-stat-val {
  color: var(--sre-text-primary);
  font-weight: 500;
}
.mono {
  font-family: var(--sre-font-mono, ui-monospace, SFMono-Regular, Menlo, monospace);
}

.ds-actions {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding-top: 10px;
  margin-top: 4px;
}

.ds-empty {
  padding: 80px 0;
  display: flex;
  justify-content: center;
}

.ds-pagination {
  margin-top: 24px;
  display: flex;
  justify-content: flex-end;
}
</style>
