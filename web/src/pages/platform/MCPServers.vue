<script setup lang="ts">
import { ref, onMounted, h } from 'vue'
import { useI18n } from 'vue-i18n'
import {
  NButton, NIcon, NSwitch, NDataTable, NCard, NSpace, NTag,
  NDrawer, NDrawerContent, NForm, NFormItem, NInput, NInputNumber,
  NSpin, NEmpty, NPopconfirm, useMessage, useDialog,
} from 'naive-ui'
import {
  AddOutline, TrashOutline, CreateOutline,
  RefreshOutline, CheckmarkCircleOutline, SearchOutline,
  ServerOutline,
} from '@vicons/ionicons5'
import { mcpServerApi } from '@/api/mcp-server'
import type { MCPServer, MCPTool, CreateMCPServerRequest } from '@/api/mcp-server'
import { getErrorMessage } from '@/utils/format'
import PageHeader from '@/components/common/PageHeader.vue'
import EmptyState from '@/components/common/EmptyState.vue'

const { t } = useI18n()
const message = useMessage()
const dialog = useDialog()

const loading = ref(false)
const servers = ref<MCPServer[]>([])
const total = ref(0)
const page = ref(1)
const pageSize = ref(100)

// Drawer state
const showDrawer = ref(false)
const editingId = ref<number | null>(null)
const saving = ref(false)

// Tools drawer state
const showToolsDrawer = ref(false)
const toolsLoading = ref(false)
const toolsList = ref<MCPTool[]>([])
const toolsServerName = ref('')

// Test state
const testingId = ref<number | null>(null)

const form = ref<CreateMCPServerRequest>({
  name: '',
  url: '',
  headers: {},
  description: '',
  enabled: true,
})

// Headers KV editor state
const headerPairs = ref<Array<{ key: string; value: string }>>([])

// --- Table columns ---
const columns = [
  { title: 'ID', key: 'id', width: 60 },
  {
    title: t('mcpServers.name'),
    key: 'name',
    width: 180,
    render: (row: MCPServer) => h('div', { style: 'display:flex;align-items:center;gap:6px' }, [
      h(NIcon, { size: 16, color: '#18a058' }, { default: () => h(ServerOutline) }),
      h('span', { style: 'font-weight:500' }, row.name),
    ]),
  },
  {
    title: t('mcpServers.url'),
    key: 'url',
    ellipsis: { tooltip: true },
    render: (row: MCPServer) => h('code', {
      style: 'font-size:12px;padding:2px 6px;background:var(--sre-bg-elevated);border-radius:4px',
    }, row.url),
  },
  {
    title: t('mcpServers.enabled'),
    key: 'enabled',
    width: 80,
    render: (row: MCPServer) => h(NSwitch, {
      value: row.enabled,
      size: 'small',
      onUpdateValue: async (val: boolean) => {
        try {
          const headersObj = row.headers ? JSON.parse(row.headers) : {}
          await mcpServerApi.update(row.id, {
            name: row.name,
            url: row.url,
            headers: headersObj,
            description: row.description,
            enabled: val,
          })
          row.enabled = val
          message.success(val ? t('common.enabled') : t('common.disabled'))
        } catch (e) {
          message.error(getErrorMessage(e))
        }
      },
    }),
  },
  {
    title: t('mcpServers.description'),
    key: 'description',
    ellipsis: { tooltip: true },
    width: 200,
  },
  {
    title: t('common.actions'),
    key: 'actions',
    width: 300,
    render: (row: MCPServer) => h(NSpace, { size: 'small' }, {
      default: () => [
        h(NButton, {
          size: 'small',
          type: 'primary',
          secondary: true,
          loading: testingId.value === row.id,
          onClick: () => testConnection(row),
        }, {
          icon: () => h(NIcon, null, { default: () => h(CheckmarkCircleOutline) }),
          default: () => t('mcpServers.testConnection'),
        }),
        h(NButton, {
          size: 'small',
          secondary: true,
          onClick: () => openTools(row),
        }, {
          icon: () => h(NIcon, null, { default: () => h(SearchOutline) }),
          default: () => t('mcpServers.viewTools'),
        }),
        h(NButton, {
          size: 'small',
          secondary: true,
          onClick: () => openEdit(row),
        }, {
          icon: () => h(NIcon, null, { default: () => h(CreateOutline) }),
        }),
        h(NPopconfirm, { onPositiveClick: () => deleteServer(row) }, {
          trigger: () => h(NButton, { size: 'small', type: 'error', secondary: true }, {
            icon: () => h(NIcon, null, { default: () => h(TrashOutline) }),
          }),
          default: () => t('mcpServers.confirmDelete'),
        }),
      ],
    }),
  },
]

// --- Fetch list ---
async function fetchList() {
  loading.value = true
  try {
    const { data } = await mcpServerApi.list({ page: page.value, page_size: pageSize.value })
    servers.value = data.data?.list || []
    total.value = data.data?.total || 0
  } catch (e) {
    message.error(getErrorMessage(e))
  } finally {
    loading.value = false
  }
}

// --- Create/Edit ---
function openCreate() {
  editingId.value = null
  form.value = { name: '', url: '', headers: {}, description: '', enabled: true }
  headerPairs.value = []
  showDrawer.value = true
}

function openEdit(row: MCPServer) {
  editingId.value = row.id
  const headersObj = row.headers ? JSON.parse(row.headers) : {}
  form.value = {
    name: row.name,
    url: row.url,
    headers: headersObj,
    description: row.description,
    enabled: row.enabled,
  }
  headerPairs.value = Object.entries(headersObj).map(([key, value]) => ({ key, value: value as string }))
  showDrawer.value = true
}

function addHeader() {
  headerPairs.value.push({ key: '', value: '' })
}

function removeHeader(idx: number) {
  headerPairs.value.splice(idx, 1)
}

function buildHeaders(): Record<string, string> {
  const result: Record<string, string> = {}
  for (const pair of headerPairs.value) {
    if (pair.key.trim()) {
      result[pair.key.trim()] = pair.value
    }
  }
  return result
}

async function handleSave() {
  if (!form.value.name.trim()) {
    message.warning(t('mcpServers.nameRequired'))
    return
  }
  if (!form.value.url.trim()) {
    message.warning(t('mcpServers.urlRequired'))
    return
  }

  saving.value = true
  try {
    const payload: CreateMCPServerRequest = {
      name: form.value.name,
      url: form.value.url,
      headers: buildHeaders(),
      description: form.value.description,
      enabled: form.value.enabled,
    }
    if (editingId.value) {
      await mcpServerApi.update(editingId.value, payload)
      message.success(t('mcpServers.updateSuccess'))
    } else {
      await mcpServerApi.create(payload)
      message.success(t('mcpServers.createSuccess'))
    }
    showDrawer.value = false
    await fetchList()
  } catch (e) {
    message.error(getErrorMessage(e))
  } finally {
    saving.value = false
  }
}

// --- Delete ---
async function deleteServer(row: MCPServer) {
  try {
    await mcpServerApi.delete(row.id)
    message.success(t('mcpServers.deleteSuccess'))
    await fetchList()
  } catch (e) {
    message.error(getErrorMessage(e))
  }
}

// --- Test connection ---
async function testConnection(row: MCPServer) {
  testingId.value = row.id
  try {
    await mcpServerApi.testConnection(row.id)
    message.success(t('mcpServers.testSuccess'))
  } catch (e) {
    message.error(getErrorMessage(e) || t('mcpServers.testFailed'))
  } finally {
    testingId.value = null
  }
}

// --- View tools ---
async function openTools(row: MCPServer) {
  toolsServerName.value = row.name
  toolsList.value = []
  toolsLoading.value = true
  showToolsDrawer.value = true
  try {
    const { data } = await mcpServerApi.listTools(row.id)
    toolsList.value = data.data || []
  } catch (e) {
    message.error(getErrorMessage(e))
  } finally {
    toolsLoading.value = false
  }
}

onMounted(fetchList)
</script>

<template>
  <div class="page-container">
    <PageHeader
      :title="t('mcpServers.title')"
      :subtitle="t('mcpServers.subtitle')"
    >
      <template #actions>
        <NSpace>
          <NButton @click="fetchList">
            <template #icon><NIcon :component="RefreshOutline" /></template>
            {{ t('common.refresh') }}
          </NButton>
          <NButton type="primary" @click="openCreate">
            <template #icon><NIcon :component="AddOutline" /></template>
            {{ t('mcpServers.createServer') }}
          </NButton>
        </NSpace>
      </template>
    </PageHeader>

    <NCard>
      <NDataTable
        v-if="servers.length > 0"
        :columns="columns"
        :data="servers"
        :loading="loading"
        :bordered="false"
        :single-line="false"
        size="small"
      />
      <EmptyState
        v-else-if="!loading"
        :title="t('mcpServers.noData')"
        :description="t('mcpServers.noDataHint')"
      />
    </NCard>

    <!-- Create/Edit Drawer -->
    <NDrawer
      v-model:show="showDrawer"
      :width="480"
      :title="editingId ? t('mcpServers.editServer') : t('mcpServers.createServer')"
    >
      <NDrawerContent :title="editingId ? t('mcpServers.editServer') : t('mcpServers.createServer')">
        <NForm label-placement="top" :model="form">
          <NFormItem :label="t('mcpServers.name')" required>
            <NInput
              v-model:value="form.name"
              :placeholder="t('mcpServers.namePlaceholder')"
            />
          </NFormItem>
          <NFormItem :label="t('mcpServers.url')" required>
            <NInput
              v-model:value="form.url"
              placeholder="http://localhost:3000/sse"
            />
          </NFormItem>
          <NFormItem :label="t('mcpServers.headers')">
            <div style="width:100%">
              <div
                v-for="(pair, idx) in headerPairs"
                :key="idx"
                style="display:flex;gap:8px;margin-bottom:8px;align-items:center"
              >
                <NInput
                  v-model:value="pair.key"
                  :placeholder="t('mcpServers.headerKey')"
                  size="small"
                  style="flex:1"
                />
                <NInput
                  v-model:value="pair.value"
                  :placeholder="t('mcpServers.headerValue')"
                  size="small"
                  style="flex:1"
                />
                <NButton
                  size="small"
                  quaternary
                  type="error"
                  @click="removeHeader(idx)"
                >
                  <template #icon><NIcon :component="TrashOutline" /></template>
                </NButton>
              </div>
              <NButton dashed size="small" @click="addHeader">
                <template #icon><NIcon :component="AddOutline" /></template>
                {{ t('mcpServers.addHeader') }}
              </NButton>
            </div>
          </NFormItem>
          <NFormItem :label="t('mcpServers.description')">
            <NInput
              v-model:value="form.description"
              type="textarea"
              :placeholder="t('mcpServers.descriptionPlaceholder')"
              :rows="3"
            />
          </NFormItem>
          <NFormItem :label="t('mcpServers.enabled')">
            <NSwitch v-model:value="form.enabled" />
          </NFormItem>
        </NForm>
        <template #footer>
          <NSpace>
            <NButton @click="showDrawer = false">{{ t('common.cancel') }}</NButton>
            <NButton type="primary" :loading="saving" @click="handleSave">
              {{ t('common.save') }}
            </NButton>
          </NSpace>
        </template>
      </NDrawerContent>
    </NDrawer>

    <!-- Tools Drawer -->
    <NDrawer
      v-model:show="showToolsDrawer"
      :width="560"
      :title="t('mcpServers.tools') + ' - ' + toolsServerName"
    >
      <NDrawerContent :title="t('mcpServers.tools') + ' - ' + toolsServerName">
        <NSpin :show="toolsLoading">
          <div v-if="toolsList.length === 0 && !toolsLoading" style="padding:40px 0">
            <NEmpty :description="t('mcpServers.noTools')" />
          </div>
          <div v-else>
            <div
              v-for="tool in toolsList"
              :key="tool.name"
              style="
                padding: 12px;
                margin-bottom: 8px;
                border: 1px solid var(--sre-border);
                border-radius: var(--sre-radius-md);
                background: var(--sre-bg-elevated);
              "
            >
              <div style="font-weight:600;font-size:14px;margin-bottom:4px">
                <NTag size="small" type="info" style="margin-right:8px">{{ tool.name }}</NTag>
              </div>
              <div v-if="tool.description" style="font-size:12px;color:var(--sre-text-secondary);margin-bottom:8px">
                {{ tool.description }}
              </div>
              <details v-if="tool.input_schema && Object.keys(tool.input_schema).length > 0">
                <summary style="font-size:12px;cursor:pointer;color:var(--sre-text-tertiary)">
                  {{ t('mcpServers.inputSchema') }}
                </summary>
                <pre style="
                  margin-top:4px;
                  padding:8px;
                  font-size:11px;
                  background:var(--sre-bg);
                  border-radius:4px;
                  overflow-x:auto;
                  max-height:200px;
                ">{{ JSON.stringify(tool.input_schema, null, 2) }}</pre>
              </details>
            </div>
          </div>
        </NSpin>
      </NDrawerContent>
    </NDrawer>
  </div>
</template>

<style scoped>
.page-container {
  padding: 0;
}
</style>
