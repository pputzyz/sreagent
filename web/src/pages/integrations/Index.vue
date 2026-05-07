<script setup lang="ts">
import { ref, onMounted, computed, h } from 'vue'
import { useMessage, NTag, NButton, NSpace, NPopconfirm, NTooltip } from 'naive-ui'
import { useI18n } from 'vue-i18n'
import { integrationV2Api, channelV2Api } from '@/api'
import PageHeader from '@/components/common/PageHeader.vue'
import { AddOutline, RefreshOutline, CopyOutline, LinkOutline, GitNetworkOutline } from '@vicons/ionicons5'
import RoutingRules from './RoutingRules.vue'

const { t } = useI18n()
const message = useMessage()

const loading = ref(false)
const integrations = ref<any[]>([])
const total = ref(0)
const page = ref(1)
const pageSize = ref(20)
const channels = ref<any[]>([])

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

const typeOptions = [
  { label: t('integration.typeStandard'), value: 'standard' },
  { label: t('integration.typeAlertManager'), value: 'alertmanager' },
  { label: t('integration.typeGrafana'), value: 'grafana' },
]
const modeOptions = [
  { label: t('integration.modeExclusive'), value: 'exclusive' },
  { label: t('integration.modeShared'), value: 'shared' },
]
const typeTagType: Record<string, 'default' | 'info' | 'success' | 'warning'> = {
  standard: 'default',
  alertmanager: 'info',
  grafana: 'warning',
}

const channelOptions = computed(() => channels.value.map(c => ({ label: c.name, value: c.id })))

async function load() {
  loading.value = true
  try {
    const [intRes, chRes] = await Promise.all([
      integrationV2Api.list({ page: page.value, page_size: pageSize.value }),
      channelV2Api.list({ status: 'active', page: 1, page_size: 100 }),
    ])
    integrations.value = intRes.data.data?.list ?? []
    total.value = intRes.data.data?.total ?? 0
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

function copyToClipboard(text: string) {
  navigator.clipboard.writeText(text).then(() => message.success('已复制'))
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
  if (!form.value.name.trim()) return
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

const columns = computed(() => [
  {
    title: t('integration.name'),
    key: 'name',
    render: (row: any) => h('span', { style: 'font-weight:500' }, row.name),
  },
  {
    title: t('integration.type'),
    key: 'type',
    width: 120,
    render: (row: any) =>
      h(NTag, { type: typeTagType[row.type] ?? 'default', size: 'small' },
        { default: () => row.type.toUpperCase() }),
  },
  {
    title: t('integration.mode'),
    key: 'mode',
    width: 100,
    render: (row: any) => h('span', { style: 'font-size:12px' }, row.mode === 'exclusive' ? t('integration.modeExclusive') : t('integration.modeShared')),
  },
  {
    title: t('integration.channel'),
    key: 'channel',
    render: (row: any) => h('span', {}, row.channel?.name ?? '—'),
  },
  {
    title: t('integration.totalAlerts'),
    key: 'total_alerts',
    width: 100,
    render: (row: any) => h('span', {}, String(row.total_alerts ?? 0)),
  },
  {
    title: t('common.status'),
    key: 'is_enabled',
    width: 80,
    render: (row: any) =>
      h(NTag, { type: row.is_enabled ? 'success' : 'default', size: 'small' },
        { default: () => row.is_enabled ? t('common.enabled') : t('common.disabled') }),
  },
  {
    title: t('integration.webhookUrl'),
    key: 'webhook_url',
    render: (row: any) =>
      h(NSpace, { size: 'small', align: 'center' }, {
        default: () => [
          h('span', { style: 'font-family:monospace;font-size:11px;color:var(--sre-text-secondary)' },
            row.webhook_token ? row.webhook_token.substring(0, 12) + '…' : '—'),
          h(NButton, {
            size: 'tiny', quaternary,
            onClick: () => copyToClipboard(webhookUrl(row.webhook_token)),
          }, { default: () => h(CopyOutline) }),
        ],
      }),
  },
  {
    title: t('common.actions'),
    key: 'actions',
    width: 120,
    render: (row: any) =>
      h(NSpace, { size: 'small' }, {
        default: () => [
          row.mode === 'shared'
            ? h(NButton, {
                size: 'tiny',
                type: 'info',
                ghost: true,
                onClick: () => openRoutingRules(row),
              }, {
                default: () => '路由规则',
                icon: () => h('n-icon', { component: GitNetworkOutline }),
              })
            : null,
          h(NButton, { size: 'tiny', onClick: () => openEdit(row) }, { default: () => t('common.edit') }),
          h(NPopconfirm, { onPositiveClick: () => deleteInteg(row.id) }, {
            trigger: () => h(NButton, { size: 'tiny', quaternary: true, type: 'error' }, { default: () => t('common.delete') }),
            default: () => t('common.confirmDeleteMsg'),
          }),
        ].filter(Boolean),
      }),
  },
])

// Fix: quaternary is a boolean prop for NButton, not a variable
const quaternary = true

// Routing rules drawer (for shared integrations)
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
  <div class="integrations-page">
    <PageHeader :title="t('integration.title')" :subtitle="t('integration.subtitle')">
      <template #actions>
        <n-button circle quaternary @click="load">
          <template #icon><n-icon :component="RefreshOutline" /></template>
        </n-button>
        <n-button type="primary" @click="openCreate">
          <template #icon><n-icon :component="AddOutline" /></template>
          {{ t('integration.create') }}
        </n-button>
      </template>
    </PageHeader>

    <n-card :bordered="false" class="table-card">
      <n-data-table
        :loading="loading"
        :columns="columns"
        :data="integrations"
        :row-key="(row: any) => row.id"
        size="small"
      />
      <div v-if="total > pageSize" style="display:flex;justify-content:flex-end;margin-top:12px">
        <n-pagination
          v-model:page="page"
          :page-count="Math.ceil(total / pageSize)"
          @update:page="load"
        />
      </div>
    </n-card>

    <!-- Webhook URL tip -->
    <n-alert type="info" style="margin-top:16px;border-radius:10px" :show-icon="false">
      <span style="font-size:12px">
        <strong>Webhook URL 格式：</strong>
        <code>POST {{ $router.options.history.base || '' }}/api/v1/integrations/&lt;token&gt;/alerts</code>
        &nbsp;·&nbsp;{{ t('integration.rateLimit') }}
      </span>
    </n-alert>

    <!-- Create/Edit modal -->
    <n-modal
      v-model:show="showModal"
      :title="editingId ? t('common.edit') : t('integration.create')"
      preset="card"
      style="width:540px"
      :bordered="false"
    >
      <n-scrollbar style="max-height:70vh">
        <n-form label-placement="top" size="small" style="padding-right:12px">
          <n-form-item :label="t('integration.name')" required>
            <n-input v-model:value="form.name" />
          </n-form-item>
          <n-form-item :label="t('common.description')">
            <n-input v-model:value="form.description" type="textarea" :rows="2" />
          </n-form-item>
          <n-grid :cols="2" :x-gap="12">
            <n-form-item-gi :label="t('integration.type')">
              <n-select v-model:value="form.type" :options="typeOptions" :disabled="!!editingId" />
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
            <n-checkbox v-model:checked="form.is_enabled">{{ t('common.enabled') }}</n-checkbox>
          </n-form-item>
        </n-form>
      </n-scrollbar>
      <template #footer>
        <n-space justify="end">
          <n-button @click="showModal = false">{{ t('common.cancel') }}</n-button>
          <n-button type="primary" :loading="saving" @click="save">{{ t('common.save') }}</n-button>
        </n-space>
      </template>
    </n-modal>

    <!-- Routing Rules Drawer (shared integrations only) -->
    <n-drawer
      v-model:show="showRoutingDrawer"
      :width="680"
      placement="right"
    >
      <n-drawer-content :title="`路由规则 — ${routingIntegName}`" closable>
        <RoutingRules v-if="showRoutingDrawer" :integration-id="routingIntegId" />
      </n-drawer-content>
    </n-drawer>
  </div>
</template>

<style scoped>
.integrations-page { max-width: 1400px; }
.table-card { border-radius: 12px; }
</style>
