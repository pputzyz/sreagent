<template>
  <div class="forwarders-page">
    <n-card :bordered="false">
      <template #header>
        <div class="card-header">
          <n-space align="center" justify="space-between">
            <n-space align="center">
              <n-icon size="24" :depth="3">
                <SwapOutlined />
              </n-icon>
              <n-text strong style="font-size: 18px">{{ t('forwarder.title') }}</n-text>
              <n-tag v-if="stats" type="info" size="small">
                {{ t('forwarder.enabledCount') }}: {{ stats.enabled_count }}
              </n-tag>
            </n-space>
            <n-space>
              <n-button @click="handleRefresh">
                <template #icon><n-icon><ReloadOutlined /></n-icon></template>
                {{ t('common.refresh') }}
              </n-button>
              <n-button type="primary" @click="handleCreate">
                <template #icon><n-icon><PlusOutlined /></n-icon></template>
                {{ t('forwarder.create') }}
              </n-button>
            </n-space>
          </n-space>
        </div>
      </template>

      <!-- Filters -->
      <n-space style="margin-bottom: 16px">
        <n-select
          v-model:value="filters.direction"
          :placeholder="t('forwarder.direction')"
          clearable
          :options="directionOptions"
          style="width: 180px"
        />
        <n-select
          v-model:value="filters.enabled"
          :placeholder="t('forwarder.status')"
          clearable
          :options="statusOptions"
          style="width: 120px"
        />
      </n-space>

      <!-- Table -->
      <n-data-table
        :columns="columns"
        :data="forwarders"
        :loading="loading"
        :pagination="pagination"
        :row-key="(row: AlertForwarder) => row.id"
        @update:page="handlePageChange"
        @update:page-size="handlePageSizeChange"
      />
    </n-card>

    <!-- Create/Edit Modal -->
    <n-modal
      v-model:show="showModal"
      :title="editingId ? t('forwarder.edit') : t('forwarder.create')"
      style="width: 800px"
      :mask-closable="false"
    >
      <ForwarderForm
        :id="editingId"
        @success="handleFormSuccess"
        @cancel="showModal = false"
      />
    </n-modal>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, computed, onMounted, h } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRouter } from 'vue-router'
import {
  NCard, NButton, NIcon, NSpace, NText, NTag, NDataTable, NSelect, NModal,
  NSwitch, NPopconfirm, useMessage
} from 'naive-ui'
import {
  SwapOutlined, ReloadOutlined, PlusOutlined, EditOutlined,
  DeleteOutlined, PlayCircleOutlined, PauseCircleOutlined
} from '@vicons/antd'
import type { DataTableColumns, PaginationProps } from 'naive-ui'
import {
  listAlertForwarders, deleteAlertForwarder, enableAlertForwarder,
  disableAlertForwarder, getForwarderStats, testAlertForwarder
} from '@/api/alert-forwarder'
import type { AlertForwarder, ForwarderStats } from '@/api/alert-forwarder'
import ForwarderForm from './ForwarderForm.vue'

const { t } = useI18n()
const message = useMessage()
const router = useRouter()

// State
const loading = ref(false)
const forwarders = ref<AlertForwarder[]>([])
const stats = ref<ForwarderStats | null>(null)
const showModal = ref(false)
const editingId = ref<number | null>(null)

const filters = reactive({
  direction: null as string | null,
  enabled: null as boolean | null
})

const pagination = reactive<PaginationProps>({
  page: 1,
  pageSize: 20,
  itemCount: 0,
  showSizePicker: true,
  pageSizes: [10, 20, 50, 100]
})

// Options
const directionOptions = computed(() => [
  { label: t('forwarder.directionInbound'), value: 'inbound' },
  { label: t('forwarder.directionOutbound'), value: 'outbound' },
  { label: t('forwarder.directionBidirectional'), value: 'bidirectional' }
])

const statusOptions = computed(() => [
  { label: t('common.enabled'), value: true },
  { label: t('common.disabled'), value: false }
])

// Columns
const columns = computed<DataTableColumns<AlertForwarder>>(() => [
  {
    title: t('common.name'),
    key: 'name',
    width: 200,
    ellipsis: { tooltip: true }
  },
  {
    title: t('forwarder.direction'),
    key: 'direction',
    width: 120,
    render(row) {
      const typeMap: Record<string, string> = {
        inbound: 'info',
        outbound: 'success',
        bidirectional: 'warning'
      }
      return h(NTag, { type: typeMap[row.direction] as any, size: 'small' }, {
        default: () => t(`forwarder.direction${row.direction.charAt(0).toUpperCase() + row.direction.slice(1)}`)
      })
    }
  },
  {
    title: t('forwarder.status'),
    key: 'enabled',
    width: 100,
    render(row) {
      return h(NSwitch, {
        value: row.enabled,
        onUpdateValue: () => handleToggleEnabled(row)
      })
    }
  },
  {
    title: t('forwarder.priority'),
    key: 'priority',
    width: 80
  },
  {
    title: t('forwarder.inboundSeverityMapping'),
    key: 'inbound_severity_mapping',
    width: 140,
    render(row) {
      if (row.inbound_severity_mapping?.enabled) {
        return h(NTag, { type: 'warning', size: 'small' }, {
          default: () => t('common.enabled')
        })
      }
      return h(NTag, { type: 'default', size: 'small' }, {
        default: () => t('common.disabled')
      })
    }
  },
  {
    title: t('forwarder.outboundSeverityMapping'),
    key: 'outbound_severity_mapping',
    width: 140,
    render(row) {
      if (row.outbound_severity_mapping?.enabled) {
        return h(NTag, { type: 'warning', size: 'small' }, {
          default: () => t('common.enabled')
        })
      }
      return h(NTag, { type: 'default', size: 'small' }, {
        default: () => t('common.disabled')
      })
    }
  },
  {
    title: t('forwarder.platformCapabilities'),
    key: 'platform_capabilities',
    width: 150,
    render(row) {
      const caps = row.platform_capabilities
      if (!caps) return '-'
      const enabled = []
      if (caps.enable_notification) enabled.push(t('forwarder.capNotification'))
      if (caps.enable_escalation) enabled.push(t('forwarder.capEscalation'))
      if (caps.enable_mute) enabled.push(t('forwarder.capMute'))
      if (caps.enable_inhibition) enabled.push(t('forwarder.capInhibition'))
      if (caps.enable_ai_analysis) enabled.push(t('forwarder.capAI'))
      return h(NText, { depth: 3 }, { default: () => enabled.join(', ') || '-' })
    }
  },
  {
    title: t('common.createdAt'),
    key: 'created_at',
    width: 160,
    render(row) {
      return new Date(row.created_at).toLocaleString()
    }
  },
  {
    title: t('common.actions'),
    key: 'actions',
    width: 200,
    fixed: 'right',
    render(row) {
      return h(NSpace, { size: 'small' }, {
        default: () => [
          h(NButton, {
            size: 'small',
            onClick: () => handleEdit(row)
          }, {
            icon: () => h(NIcon, null, { default: () => h(EditOutlined) }),
            default: () => t('common.edit')
          }),
          h(NButton, {
            size: 'small',
            type: 'info',
            onClick: () => handleTest(row)
          }, {
            default: () => t('forwarder.test')
          }),
          h(NPopconfirm, {
            onPositiveClick: () => handleDelete(row)
          }, {
            trigger: () => h(NButton, {
              size: 'small',
              type: 'error'
            }, {
              icon: () => h(NIcon, null, { default: () => h(DeleteOutlined) }),
              default: () => t('common.delete')
            }),
            default: () => t('forwarder.deleteConfirm')
          })
        ]
      })
    }
  }
])

// Methods
async function fetchForwarders() {
  loading.value = true
  try {
    const params: any = {
      page: pagination.page,
      page_size: pagination.pageSize
    }
    if (filters.direction) params.direction = filters.direction
    if (filters.enabled !== null) params.enabled = filters.enabled

    const res = await listAlertForwarders(params)
    forwarders.value = res.data.data?.list || []
    pagination.itemCount = res.data.data?.total || 0
  } catch (error: any) {
    message.error(error.message || t('common.error'))
  } finally {
    loading.value = false
  }
}

async function fetchStats() {
  try {
    const res = await getForwarderStats()
    stats.value = res.data.data || null
  } catch (error) {
    // Ignore stats error
  }
}

function handleRefresh() {
  fetchForwarders()
  fetchStats()
}

function handleCreate() {
  editingId.value = null
  showModal.value = true
}

function handleEdit(row: AlertForwarder) {
  editingId.value = row.id
  showModal.value = true
}

async function handleTest(row: AlertForwarder) {
  try {
    const res = await testAlertForwarder(row.id)
    message.success(t('forwarder.testSuccess'))
    console.log('Test result:', res.data.data)
  } catch (error: any) {
    message.error(error.message || t('forwarder.testFailed'))
  }
}

async function handleDelete(row: AlertForwarder) {
  try {
    await deleteAlertForwarder(row.id)
    message.success(t('common.deleteSuccess'))
    fetchForwarders()
    fetchStats()
  } catch (error: any) {
    message.error(error.message || t('common.error'))
  }
}

async function handleToggleEnabled(row: AlertForwarder) {
  try {
    if (row.enabled) {
      await disableAlertForwarder(row.id)
    } else {
      await enableAlertForwarder(row.id)
    }
    row.enabled = !row.enabled
    message.success(row.enabled ? t('forwarder.enabled') : t('forwarder.disabled'))
    fetchStats()
  } catch (error: any) {
    message.error(error.message || t('common.error'))
  }
}

function handlePageChange(page: number) {
  pagination.page = page
  fetchForwarders()
}

function handlePageSizeChange(pageSize: number) {
  pagination.pageSize = pageSize
  pagination.page = 1
  fetchForwarders()
}

function handleFormSuccess() {
  showModal.value = false
  fetchForwarders()
  fetchStats()
}

// Watch filters
import { watch } from 'vue'
watch(() => filters.direction, () => {
  pagination.page = 1
  fetchForwarders()
})

watch(() => filters.enabled, () => {
  pagination.page = 1
  fetchForwarders()
})

// Init
onMounted(() => {
  fetchForwarders()
  fetchStats()
})
</script>

<style scoped>
.forwarders-page {
  padding: 16px;
}

.card-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
}
</style>
