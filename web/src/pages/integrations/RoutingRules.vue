<script setup lang="ts">
/**
 * RoutingRules.vue — 共享集成的路由规则配置面板
 * 按优先级从上到下匹配，命中即停，未命中则丢弃（或可配置默认空间）
 */
import { ref, onMounted, h, computed, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useMessage, NButton, NSpace, NTag, NPopconfirm, NSwitch } from 'naive-ui'
import { channelV2Api, routingRuleApi } from '@/api'
import type { RoutingRule, Channel } from '@/types'
import { getErrorMessage } from '@/utils/format'
import { AddOutline, TrashOutline, CreateOutline, ArrowUpOutline, ArrowDownOutline, GitNetworkOutline } from '@vicons/ionicons5'
import EmptyState from '@/components/common/EmptyState.vue'

const props = defineProps<{ integrationId: number }>()
const message = useMessage()
const { t } = useI18n()

const rules = ref<RoutingRule[]>([])
const channels = ref<Channel[]>([])
const loading = ref(false)
const showModal = ref(false)
const saving = ref(false)
const editingId = ref<number | null>(null)

const emptyForm = () => ({
  target_channel_id: null as number | null,
  conditions: '[]',
  priority: 0,
  is_enabled: true,
})
const form = ref(emptyForm())

const channelOptions = computed(() =>
  channels.value.map(c => ({ label: c.name, value: c.id }))
)

// --- Structured condition editor ---
interface ConditionItem {
  field: string
  operator: string
  value: string
}

const conditionItems = ref<ConditionItem[]>([])

const fieldOptions = [
  { label: 'severity', value: 'severity' },
  { label: 'status', value: 'status' },
  { label: 'rule_name', value: 'rule_name' },
  { label: 'datasource_id', value: 'datasource_id' },
  { label: 'tags', value: 'tags' },
]

const operatorOptions = [
  { label: '= (等于)', value: 'eq' },
  { label: '!= (不等于)', value: 'ne' },
  { label: '=~ (正则匹配)', value: 'regex' },
  { label: '!~ (正则不匹配)', value: 'not_regex' },
  { label: 'in (包含)', value: 'in' },
]

function parseConditions(json: string): ConditionItem[] {
  try {
    const arr = JSON.parse(json || '[]')
    if (!Array.isArray(arr)) return []
    return arr.map((c: Record<string, string>) => ({
      field: c.field || '',
      operator: c.operator || 'eq',
      value: c.value || '',
    }))
  } catch { return [] }
}

function serializeConditions(items: ConditionItem[]): string {
  return JSON.stringify(items.filter(c => c.field && c.value).map(c => ({
    field: c.field,
    operator: c.operator,
    value: c.value,
  })))
}

function addCondition() {
  conditionItems.value.push({ field: '', operator: 'eq', value: '' })
}

function removeCondition(idx: number) {
  conditionItems.value.splice(idx, 1)
}

watch(conditionItems, (items) => {
  form.value.conditions = serializeConditions(items)
}, { deep: true })

async function load() {
  loading.value = true
  try {
    const [rRes, cRes] = await Promise.all([
      routingRuleApi.listByIntegration(props.integrationId),
      channelV2Api.list({ status: 'active', page: 1, page_size: 100 }),
    ])
    rules.value = (rRes.data.data ?? []).sort((a: RoutingRule, b: RoutingRule) => a.priority - b.priority)
    channels.value = cRes.data.data?.list ?? []
  } catch (e: unknown) {
    message.error(getErrorMessage(e) || t('common.loadFailed'))
  } finally {
    loading.value = false
  }
}

function openCreate() {
  editingId.value = null
  form.value = emptyForm()
  conditionItems.value = []
  showModal.value = true
}

function openEdit(rule: RoutingRule) {
  editingId.value = rule.id
  form.value = {
    target_channel_id: rule.target_channel_id,
    conditions: rule.conditions || '[]',
    priority: rule.priority,
    is_enabled: rule.is_enabled,
  }
  conditionItems.value = parseConditions(form.value.conditions)
  showModal.value = true
}

async function save() {
  if (!form.value.target_channel_id) {
    message.warning(t('routingRule.selectChannelRequired'))
    return
  }
  saving.value = true
  try {
    if (editingId.value) {
      await routingRuleApi.update(editingId.value, {
        target_channel_id: form.value.target_channel_id!,
        conditions: form.value.conditions,
        priority: form.value.priority,
        is_enabled: form.value.is_enabled,
      })
    } else {
      await routingRuleApi.create(props.integrationId, {
        target_channel_id: form.value.target_channel_id!,
        conditions: form.value.conditions,
        priority: form.value.priority,
        is_enabled: form.value.is_enabled,
      })
    }
    message.success(t('common.savedSuccess'))
    showModal.value = false
    await load()
  } catch (e: unknown) {
    message.error(getErrorMessage(e) || t('common.saveFailed'))
  } finally {
    saving.value = false
  }
}

async function deleteRule(id: number) {
  try {
    await routingRuleApi.delete(id)
    message.success(t('common.deleteSuccess'))
    await load()
  } catch (e: unknown) {
    message.error(getErrorMessage(e) || t('common.deleteFailed'))
  }
}

async function toggleEnabled(rule: RoutingRule) {
  try {
    await routingRuleApi.update(rule.id, { is_enabled: !rule.is_enabled })
    rule.is_enabled = !rule.is_enabled
  } catch (e: unknown) {
    message.error(getErrorMessage(e) || t('common.failed'))
  }
}

async function movePriority(index: number, direction: 'up' | 'down') {
  const swapIdx = direction === 'up' ? index - 1 : index + 1
  if (swapIdx < 0 || swapIdx >= rules.value.length) return

  const a = rules.value[index]
  const b = rules.value[swapIdx]
  const tmpPri = a.priority
  a.priority = b.priority
  b.priority = tmpPri
  ;[rules.value[index], rules.value[swapIdx]] = [rules.value[swapIdx], rules.value[index]]

  try {
    await Promise.all([
      routingRuleApi.update(a.id, { priority: a.priority }),
      routingRuleApi.update(b.id, { priority: b.priority }),
    ])
  } catch (e: unknown) {
    message.error(t('routingRule.adjustPriorityFailed'))
    await load()
  }
}

const columns = computed(() => [
  {
    title: t('routingRule.priority'),
    key: 'priority',
    width: 80,
    render: (_: RoutingRule, index: number) =>
      h('span', { style: 'font-size:12px;color:var(--sre-text-secondary)' }, String(index + 1)),
  },
  {
    title: t('routingRule.targetChannel'),
    key: 'target_channel',
    render: (row: RoutingRule) =>
      h('span', { style: 'font-weight:500' }, row.target_channel?.name ?? `#${row.target_channel_id}`),
  },
  {
    title: t('routingRule.matchConditions'),
    key: 'conditions',
    render: (row: RoutingRule) => {
      try {
        const conds = JSON.parse(row.conditions || '[]')
        if (!conds.length) return h('span', { style: 'color:var(--sre-text-secondary)' }, t('routingRule.catchAllRule'))
        return h('span', { style: 'font-size:12px' }, t('routingRule.conditionsCount', { n: conds.length }))
      } catch {
        return h('span', { style: 'color:var(--sre-text-secondary)' }, '—')
      }
    },
  },
  {
    title: t('routingRule.enabled'),
    key: 'is_enabled',
    width: 70,
    render: (row: RoutingRule) =>
      h(NSwitch, {
        value: row.is_enabled,
        size: 'small',
        onUpdateValue: () => toggleEnabled(row),
      }),
  },
  {
    title: t('common.actions'),
    key: 'actions',
    width: 160,
    render: (row: RoutingRule, index: number) =>
      h(NSpace, { size: 'small' }, {
        default: () => [
          h(NButton, { size: 'tiny', circle: true, quaternary: true, disabled: index === 0, onClick: () => movePriority(index, 'up') },
            { icon: () => h('n-icon', { component: ArrowUpOutline }) }),
          h(NButton, { size: 'tiny', circle: true, quaternary: true, disabled: index === rules.value.length - 1, onClick: () => movePriority(index, 'down') },
            { icon: () => h('n-icon', { component: ArrowDownOutline }) }),
          h(NButton, { size: 'tiny', quaternary: true, onClick: () => openEdit(row) },
            { icon: () => h('n-icon', { component: CreateOutline }) }),
          h(NPopconfirm, { onPositiveClick: () => deleteRule(row.id) }, {
            trigger: () => h(NButton, { size: 'tiny', quaternary: true, type: 'error' },
              { icon: () => h('n-icon', { component: TrashOutline }) }),
            default: () => t('routingRule.confirmDelete'),
          }),
        ],
      }),
  },
])

onMounted(load)
</script>

<template>
  <div class="routing-rules">
    <div class="rr-header">
      <div>
        <p class="rr-desc">
          {{ t('routingRule.description') }}
        </p>
      </div>
      <n-button type="primary" size="small" @click="openCreate">
        <template #icon><n-icon :component="AddOutline" /></template>
        {{ t('routingRule.addRule') }}
      </n-button>
    </div>

    <n-spin :show="loading">
      <EmptyState
        v-if="!loading && rules.length === 0"
        :icon="GitNetworkOutline"
        :title="t('routingRule.noRules')"
        :description="t('routingRule.description')"
        :primary-text="t('routingRule.addRule')"
        @primary="openCreate"
      />
      <n-data-table
        v-else-if="rules.length > 0"
        :columns="columns"
        :data="rules"
        :row-key="(row: RoutingRule) => row.id"
        size="small"
      />
    </n-spin>

    <!-- Create / Edit Modal -->
    <n-modal
      v-model:show="showModal"
      :title="editingId ? t('routingRule.editRule') : t('routingRule.createRule')"
      preset="card"
      :bordered="false"
      class="rr-modal"
    >
      <n-form label-placement="top" size="small">
        <n-form-item :label="t('routingRule.targetChannelLabel')" required>
          <n-select
            v-model:value="form.target_channel_id"
            :options="channelOptions"
            :placeholder="t('routingRule.selectChannelPlaceholder')"
            filterable
          />
        </n-form-item>
        <n-form-item :label="t('routingRule.priorityLabel')">
          <n-input-number v-model:value="form.priority" :min="0" :max="9999" class="rr-input-full" />
        </n-form-item>
        <n-form-item :label="t('routingRule.conditionsLabel')">
          <div style="width: 100%; display: flex; flex-direction: column; gap: 8px;">
            <div v-for="(cond, idx) in conditionItems" :key="idx" style="display: flex; align-items: center; gap: 8px;">
              <n-select
                v-model:value="cond.field"
                :options="fieldOptions"
                :placeholder="t('routingRule.fieldPlaceholder')"
                filterable
                style="flex: 1"
              />
              <n-select
                v-model:value="cond.operator"
                :options="operatorOptions"
                style="width: 160px"
              />
              <n-input
                v-model:value="cond.value"
                :placeholder="t('routingRule.valuePlaceholder')"
                style="flex: 1"
              />
              <button class="sre-icon-btn" style="color: var(--sre-danger)" @click="removeCondition(idx)" :aria-label="t('common.delete')">
                <span class="sre-dots">&times;</span>
              </button>
            </div>
            <n-button dashed size="small" @click="addCondition" style="align-self: flex-start">
              {{ t('routingRule.addCondition') }}
            </n-button>
          </div>
        </n-form-item>
        <n-form-item :label="t('routingRule.enabled')">
          <n-switch v-model:value="form.is_enabled" />
        </n-form-item>
      </n-form>
      <template #footer>
        <n-space justify="end">
          <n-button @click="showModal = false">{{ t('common.cancel') }}</n-button>
          <n-button type="primary" :loading="saving" @click="save">{{ t('common.save') }}</n-button>
        </n-space>
      </template>
    </n-modal>
  </div>
</template>

<style scoped>
.routing-rules { padding: 4px 0; }

.rr-header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 12px;
  margin-bottom: 16px;
}

.rr-desc {
  font-size: 12px;
  color: var(--sre-text-secondary);
  margin: 0;
  line-height: 1.6;
  max-width: 600px;
}

.rr-modal { width: 520px; }
.rr-input-full { width: 100%; }
.rr-conditions-input { font-family: var(--sre-font-mono, monospace); font-size: 12px; }
.rr-conditions-hint { font-size: 11px; color: var(--sre-text-secondary); }
</style>
